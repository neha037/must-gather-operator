# E2E Execution Steps: must-gather-operator

## Prerequisites

```bash
# Verify required tools
which oc
oc version
oc whoami
oc get nodes
oc get clusterversion

# Verify cluster has OLM
oc get packagemanifests | grep support-log-gather-operator
```

## Step 1: Install Operator

### Option A: Via OLM (Recommended for Testing)

```bash
# Create namespace
oc create namespace must-gather-operator

# Create OperatorGroup
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: must-gather-operator-group
  namespace: must-gather-operator
spec:
  targetNamespaces:
  - must-gather-operator
EOF

# Create Subscription
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: support-log-gather-operator
  namespace: must-gather-operator
spec:
  channel: stable
  name: support-log-gather-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF

# Wait for installation to complete (5 minutes timeout)
echo "Waiting for CSV to be ready..."
oc wait --for=jsonpath='{.status.phase}'=Succeeded \
  csv -l operators.coreos.com/support-log-gather-operator.must-gather-operator \
  -n must-gather-operator --timeout=300s

# Get CSV name
CSV_NAME=$(oc get csv -n must-gather-operator \
  -l operators.coreos.com/support-log-gather-operator.must-gather-operator \
  -o jsonpath='{.items[0].metadata.name}')
echo "Installed CSV: $CSV_NAME"

# Wait for operator deployment
echo "Waiting for operator deployment..."
oc wait --for=condition=Available deployment/must-gather-operator \
  -n must-gather-operator --timeout=300s

# Verify operator is running
oc get deployment must-gather-operator -n must-gather-operator
oc get pods -n must-gather-operator
```

### Option B: Manual Deployment (Development)

```bash
# If testing from source
cd must-gather-operator

# Deploy prerequisites
oc apply -f examples/other_resources/

# Wait for operator deployment
oc wait --for=condition=Available deployment/must-gather-operator \
  -n must-gather-operator --timeout=300s

# Verify deployment
oc get pods -n must-gather-operator
```

## Step 2: Deploy Test CRs

### Test 2.1: Basic MustGather (Baseline)

```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-baseline
  namespace: default
spec: {}
EOF

# Verify CR created
oc get mustgather test-baseline -n default

# Wait for job creation
echo "Waiting for job to be created..."
oc wait --for=create job/test-baseline -n default --timeout=60s

# Check job
oc get job test-baseline -n default -o yaml
```

### Test 2.2: MustGather with Metrics Flag (NEW)

```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-metrics
  namespace: default
spec:
  gatherSpec:
    metrics: true
EOF

# Verify CR created with metrics field
oc get mustgather test-metrics -n default -o jsonpath='{.spec.gatherSpec.metrics}'
echo ""  # newline

# Expected output: true

# Wait for job creation
echo "Waiting for job to be created..."
oc wait --for=create job/test-metrics -n default --timeout=60s
```

### Test 2.3: MustGather with Both Audit and Metrics (NEW)

```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-audit-metrics
  namespace: default
spec:
  gatherSpec:
    audit: true
    metrics: true
EOF

# Verify both fields
oc get mustgather test-audit-metrics -n default -o jsonpath='{.spec.gatherSpec}'
echo ""  # newline

# Expected output should show both audit and metrics
```

## Step 3: Verify Diff-Specific Changes

### Test 3.1: Verify GATHER_METRICS Environment Variable

```bash
# For test-metrics job (metrics: true)
echo "Checking GATHER_METRICS env var for test-metrics job..."
METRICS_ENV=$(oc get job test-metrics -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}')

if [ "$METRICS_ENV" = "true" ]; then
  echo "✓ PASS: GATHER_METRICS environment variable is set to 'true'"
else
  echo "✗ FAIL: GATHER_METRICS environment variable not found or incorrect value: '$METRICS_ENV'"
  exit 1
fi

# For test-baseline job (no metrics field)
echo "Checking that GATHER_METRICS is NOT set for baseline job..."
BASELINE_ENV=$(oc get job test-baseline -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[*].name}')

if echo "$BASELINE_ENV" | grep -q "GATHER_METRICS"; then
  echo "✗ FAIL: GATHER_METRICS should not be set for baseline job"
  exit 1
else
  echo "✓ PASS: GATHER_METRICS is not set (as expected)"
fi
```

### Test 3.2: Verify Audit and Metrics Independence

```bash
# Create audit-only MustGather
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-audit-only
  namespace: default
spec:
  gatherSpec:
    audit: true
EOF

# Wait for job
oc wait --for=create job/test-audit-only -n default --timeout=60s

# Verify audit uses special command binary
AUDIT_COMMAND=$(oc get job test-audit-only -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].command}')

if echo "$AUDIT_COMMAND" | grep -q "gather_audit_logs"; then
  echo "✓ PASS: Audit command binary is used"
else
  echo "✗ FAIL: Audit command binary not detected"
fi

# Verify GATHER_METRICS is NOT set for audit-only
AUDIT_METRICS_ENV=$(oc get job test-audit-only -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}')

if [ -z "$AUDIT_METRICS_ENV" ]; then
  echo "✓ PASS: GATHER_METRICS not set for audit-only (independent flags)"
else
  echo "✗ FAIL: GATHER_METRICS should not be set for audit-only"
fi
```

### Test 3.3: Verify Both Flags Together

```bash
# Check the audit-metrics job
COMBINED_COMMAND=$(oc get job test-audit-metrics -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].command}')

COMBINED_METRICS_ENV=$(oc get job test-audit-metrics -n default \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}')

# Verify both: audit command and metrics env var
if echo "$COMBINED_COMMAND" | grep -q "gather_audit_logs" && [ "$COMBINED_METRICS_ENV" = "true" ]; then
  echo "✓ PASS: Both audit command and metrics env var are set"
else
  echo "✗ FAIL: Both flags should be active"
  echo "  Command contains gather_audit_logs: $(echo "$COMBINED_COMMAND" | grep -q "gather_audit_logs" && echo "yes" || echo "no")"
  echo "  GATHER_METRICS value: $COMBINED_METRICS_ENV"
fi
```

## Step 4: Wait for Job Completion

```bash
# Monitor all test jobs
echo "Monitoring job completion..."

for job in test-baseline test-metrics test-audit-only test-audit-metrics; do
  echo "Checking job: $job"

  # Check if job exists
  if ! oc get job $job -n default &> /dev/null; then
    echo "  Job $job not found, skipping"
    continue
  fi

  # Get job status
  JOB_STATUS=$(oc get job $job -n default -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}')

  if [ "$JOB_STATUS" = "True" ]; then
    echo "  ✓ $job completed successfully"
  else
    echo "  Job $job is still running or failed"
    # Optionally wait
    # oc wait --for=condition=Complete job/$job -n default --timeout=600s
  fi
done

# Check MustGather CR statuses
echo ""
echo "MustGather CR statuses:"
oc get mustgathers -n default -o custom-columns=NAME:.metadata.name,STATUS:.status.status,COMPLETED:.status.completed
```

## Step 5: Verify Job Logs (Optional)

```bash
# Get pod for test-metrics job
METRICS_POD=$(oc get pods -n default -l job-name=test-metrics --no-headers -o custom-columns=:metadata.name | head -1)

if [ -n "$METRICS_POD" ]; then
  echo "Checking logs for pod: $METRICS_POD"

  # Check gather container logs
  echo "--- Gather container logs ---"
  oc logs $METRICS_POD -n default -c gather | head -50

  # Verify environment variables in running pod
  echo "--- Environment variables ---"
  oc exec $METRICS_POD -n default -c gather -- env | grep GATHER
fi
```

## Step 6: Cleanup

```bash
echo "Cleaning up test resources..."

# Delete MustGather CRs (in reverse order of creation)
oc delete mustgather test-audit-metrics -n default --ignore-not-found=true
oc delete mustgather test-audit-only -n default --ignore-not-found=true
oc delete mustgather test-metrics -n default --ignore-not-found=true
oc delete mustgather test-baseline -n default --ignore-not-found=true

# Wait a moment for cleanup
sleep 5

# Verify jobs are deleted
oc get jobs -n default -l app.kubernetes.io/name=mustgather || echo "No jobs found (cleaned up)"

# Uninstall operator (OLM)
echo "Uninstalling operator..."
oc delete subscription support-log-gather-operator -n must-gather-operator --ignore-not-found=true

# Delete CSV
oc delete csv -l operators.coreos.com/support-log-gather-operator.must-gather-operator \
  -n must-gather-operator --ignore-not-found=true

# Delete operator group
oc delete operatorgroup must-gather-operator-group -n must-gather-operator --ignore-not-found=true

# Delete namespace
oc delete namespace must-gather-operator --ignore-not-found=true

echo "Cleanup complete!"
```

## Troubleshooting

### If Jobs Don't Start

```bash
# Check operator logs
oc logs deployment/must-gather-operator -n must-gather-operator --tail=50

# Check MustGather status
oc get mustgather <name> -n default -o yaml

# Check for events
oc get events -n default --sort-by='.lastTimestamp' | grep MustGather
```

### If Environment Variable Not Set

```bash
# Describe the job spec
oc get job <job-name> -n default -o yaml | grep -A 20 "containers:" | grep -A 10 "name: gather"

# Check controller-runtime version (should support env vars)
oc get deployment must-gather-operator -n must-gather-operator \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### If Tests Fail

```bash
# Get comprehensive job information
oc describe job <job-name> -n default

# Get pod logs
oc logs -l job-name=<job-name> -n default --all-containers=true

# Check CR conditions
oc get mustgather <name> -n default -o jsonpath='{.status.conditions}' | jq '.'
```
