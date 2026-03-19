# E2E Test Cases: must-gather-operator

## Operator Information
- **Repository**: github.com/openshift/must-gather-operator
- **Framework**: controller-runtime
- **API Group**: operator.openshift.io
- **Managed CRDs**: MustGather
- **Operator Namespace**: must-gather-operator
- **Changes Analyzed**: git diff master...HEAD

## Prerequisites
- OpenShift cluster with admin access
- `oc` CLI installed and authenticated
- Operator deployed in the cluster (via OLM or manual deployment)

## Installation

### Via OLM (Production)
```bash
# Create operator namespace
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

# Wait for CSV to be successful
oc wait --for=jsonpath='{.status.phase}'=Succeeded \
  csv -l operators.coreos.com/support-log-gather-operator.must-gather-operator \
  -n must-gather-operator --timeout=300s

# Wait for operator deployment
oc wait --for=condition=Available deployment/must-gather-operator \
  -n must-gather-operator --timeout=300s
```

### Manual Deployment (Development)
```bash
# Deploy from repository examples
oc apply -f examples/other_resources/

# Wait for operator deployment
oc wait --for=condition=Available deployment/must-gather-operator \
  -n must-gather-operator --timeout=300s
```

## CR Deployment

### Basic MustGather CR
```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-mustgather-basic
  namespace: default
spec: {}
EOF
```

### MustGather with Metrics Flag (NEW)
```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-mustgather-metrics
  namespace: default
spec:
  gatherSpec:
    metrics: true
EOF
```

### MustGather with Audit and Metrics (NEW)
```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: MustGather
metadata:
  name: test-mustgather-audit-metrics
  namespace: default
spec:
  gatherSpec:
    audit: true
    metrics: true
EOF
```

## Test Cases

### Category 1: API Type Changes (New Metrics Field)

#### Test 1.1: Create MustGather with metrics=true
- **Objective**: Verify the new `metrics` field is accepted by the API
- **Steps**:
  1. Create a MustGather CR with `spec.gatherSpec.metrics: true`
  2. Wait for the CR to be created
  3. Verify the field is persisted
- **Expected**: CR is created successfully with metrics field set to true
- **Verification**:
  ```bash
  oc get mustgather test-mustgather-metrics -n default -o jsonpath='{.spec.gatherSpec.metrics}'
  # Expected output: true
  ```

#### Test 1.2: Create MustGather with metrics=false
- **Objective**: Verify metrics field can be explicitly set to false
- **Steps**:
  1. Create a MustGather CR with `spec.gatherSpec.metrics: false`
  2. Verify the field is persisted
- **Expected**: CR is created successfully with metrics field set to false
- **Verification**:
  ```bash
  oc get mustgather test-mustgather-metrics-false -n default -o jsonpath='{.spec.gatherSpec.metrics}'
  # Expected output: false or empty (false is the default)
  ```

#### Test 1.3: Create MustGather with both audit and metrics flags
- **Objective**: Verify both audit and metrics flags can be used together
- **Steps**:
  1. Create a MustGather CR with both `audit: true` and `metrics: true`
  2. Verify both fields are persisted
- **Expected**: CR is created successfully with both flags
- **Verification**:
  ```bash
  oc get mustgather test-mustgather-audit-metrics -n default -o jsonpath='{.spec.gatherSpec}'
  # Expected output should show both audit: true and metrics: true
  ```

#### Test 1.4: Verify metrics field is optional
- **Objective**: Verify backward compatibility - metrics field is optional
- **Steps**:
  1. Create a MustGather CR without the metrics field
  2. Verify the CR is created successfully
- **Expected**: CR is created successfully without metrics field (defaults to false)
- **Verification**:
  ```bash
  oc get mustgather test-mustgather-basic -n default -o yaml
  # Verify gatherSpec.metrics is either absent or false
  ```

### Category 2: Controller Changes (Metrics Environment Variable)

#### Test 2.1: Verify GATHER_METRICS env var is set when metrics=true
- **Objective**: Verify the controller sets GATHER_METRICS=true environment variable
- **Steps**:
  1. Create a MustGather CR with `metrics: true`
  2. Wait for the job to be created
  3. Inspect the job's gather container environment variables
- **Expected**: The gather container has GATHER_METRICS=true environment variable
- **Verification**:
  ```bash
  # Wait for job creation
  oc wait --for=create job/test-mustgather-metrics -n default --timeout=60s

  # Get the gather container env vars
  oc get job test-mustgather-metrics -n default \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}'
  # Expected output: true
  ```

#### Test 2.2: Verify GATHER_METRICS env var is NOT set when metrics=false
- **Objective**: Verify the controller does NOT set GATHER_METRICS when metrics is false
- **Steps**:
  1. Create a MustGather CR with `metrics: false` or without metrics field
  2. Wait for the job to be created
  3. Inspect the job's gather container environment variables
- **Expected**: The gather container does NOT have GATHER_METRICS environment variable
- **Verification**:
  ```bash
  # Get all env vars for gather container
  oc get job test-mustgather-basic -n default \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[*].name}'
  # Verify GATHER_METRICS is not in the list
  ```

#### Test 2.3: Verify metrics env var with custom images
- **Objective**: Verify GATHER_METRICS is set even with custom images
- **Prerequisites**: ImageStream with custom image exists
- **Steps**:
  1. Create ImageStream with custom must-gather image
  2. Create MustGather CR with `imageStreamRef` and `metrics: true`
  3. Wait for job creation
  4. Inspect the gather container environment variables
- **Expected**: The gather container has GATHER_METRICS=true environment variable
- **Verification**:
  ```bash
  oc get job <job-name> -n default \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}'
  # Expected output: true
  ```

### Category 3: Integration Tests

#### Test 3.1: End-to-end must-gather with metrics
- **Objective**: Verify complete workflow with metrics enabled
- **Steps**:
  1. Create a MustGather CR with metrics enabled
  2. Wait for the job to complete successfully
  3. Verify the CR status shows "Completed"
  4. Verify the job succeeded
- **Expected**: Must-gather job completes successfully
- **Verification**:
  ```bash
  # Wait for completion
  oc wait --for=jsonpath='{.status.completed}'=true \
    mustgather/test-mustgather-metrics -n default --timeout=600s

  # Check status
  oc get mustgather test-mustgather-metrics -n default \
    -o jsonpath='{.status.status}'
  # Expected output: Completed

  # Verify job succeeded
  oc get job test-mustgather-metrics -n default \
    -o jsonpath='{.status.succeeded}'
  # Expected output: 1
  ```

#### Test 3.2: Verify audit and metrics work independently
- **Objective**: Verify audit and metrics flags are independent
- **Steps**:
  1. Create MustGather with only `audit: true`
  2. Verify GATHER_METRICS is not set but audit command binary is used
  3. Create MustGather with only `metrics: true`
  4. Verify GATHER_METRICS is set but audit command binary is not used
- **Expected**: Each flag works independently
- **Verification**:
  ```bash
  # For audit-only: check command uses gather_audit_logs
  oc get job test-mustgather-audit-only -n default \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].command}'
  # Should contain "gather_audit_logs"

  # For metrics-only: check GATHER_METRICS env var
  oc get job test-mustgather-metrics-only -n default \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env[?(@.name=="GATHER_METRICS")].value}'
  # Expected output: true
  ```

## Verification

### Operator Health
```bash
# Check operator is running
oc get deployment must-gather-operator -n must-gather-operator
oc get pods -n must-gather-operator

# Check operator logs
oc logs deployment/must-gather-operator -n must-gather-operator
```

### MustGather Resources
```bash
# List all MustGather CRs
oc get mustgathers -A

# Check status of specific MustGather
oc get mustgather <name> -n <namespace> -o yaml

# Check associated jobs
oc get jobs -A -l app.kubernetes.io/name=mustgather

# Check job pods
oc get pods -A -l job-name=<mustgather-name>
```

### Job Environment Variables
```bash
# Check gather container environment
oc get job <job-name> -n <namespace> \
  -o jsonpath='{.spec.template.spec.containers[?(@.name=="gather")].env}' | jq '.'
```

## Cleanup

### Delete Test MustGather CRs
```bash
oc delete mustgather test-mustgather-basic -n default
oc delete mustgather test-mustgather-metrics -n default
oc delete mustgather test-mustgather-metrics-false -n default
oc delete mustgather test-mustgather-audit-metrics -n default
```

### Uninstall Operator (OLM)
```bash
# Delete subscription
oc delete subscription support-log-gather-operator -n must-gather-operator

# Delete CSV
oc delete csv -l operators.coreos.com/support-log-gather-operator.must-gather-operator \
  -n must-gather-operator

# Delete operator group
oc delete operatorgroup must-gather-operator-group -n must-gather-operator

# Delete namespace
oc delete namespace must-gather-operator
```

### Uninstall Operator (Manual)
```bash
# Delete operator resources
oc delete -f examples/other_resources/

# Delete namespace
oc delete namespace must-gather-operator
```

## Notes

- All tests assume the operator is already deployed and running
- The metrics field is new in this release - test backward compatibility
- The GATHER_METRICS environment variable allows both default and custom images to respect the metrics flag
- Jobs are automatically cleaned up by the operator after completion (unless retainResourcesOnCompletion is set)
