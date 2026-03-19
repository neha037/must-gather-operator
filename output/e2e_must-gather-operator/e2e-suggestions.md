# E2E Test Suggestions: must-gather-operator

## Repository Summary

- **Repository**: github.com/openshift/must-gather-operator
- **Framework**: controller-runtime (Kubebuilder)
- **Managed CRDs**: MustGather (operator.openshift.io/v1alpha1)
- **E2E Pattern**: Ginkgo v2 (existing tests in test/e2e/)
- **Operator Namespace**: must-gather-operator
- **Install Mechanism**: OLM (support-log-gather-operator package)

## Changes Detected

### 1. API Types (api/v1alpha1/mustgather_types.go)
- **Change**: Added `Metrics bool` field to `GatherSpec` struct
- **Lines**: +6 insertions
- **Impact**: New API field for enabling Prometheus metrics collection

### 2. Integration Tests (api/v1alpha1/tests/)
- **Change**: Added comprehensive test suite (`mustgather.testsuite.yaml`)
- **Lines**: +492 insertions
- **Impact**: API validation tests for the new metrics field and existing functionality

### 3. Controller (controllers/mustgather/template.go)
- **Change**: Updated job template generation to handle metrics flag
- **Lines**: +18 insertions, -3 deletions
- **Impact**: Sets `GATHER_METRICS=true` environment variable when metrics flag is enabled

### 4. CRD Manifest (deploy/crds/operator.openshift.io_mustgathers.yaml)
- **Change**: Updated CRD schema with metrics field
- **Lines**: +6 insertions
- **Impact**: OpenAPI v3 schema includes the new boolean field

## Test Scenario Recommendations

### Highly Recommended (Must Have)

These tests directly verify the changes introduced in this diff:

#### 1. **Metrics Field API Validation** ✅ INCLUDED
- **Why**: Verifies the new `metrics` field is accepted by the API
- **Tests**:
  - Create MustGather with `metrics: true`
  - Create MustGather with `metrics: false`
  - Create MustGather without metrics field (backward compatibility)
  - Create MustGather with both `audit` and `metrics` flags
- **Location**: `e2e_test.go` - "MustGather Metrics Flag" Describe block
- **Priority**: P0 - Critical

#### 2. **GATHER_METRICS Environment Variable** ✅ INCLUDED
- **Why**: Verifies controller implementation sets the env var correctly
- **Tests**:
  - Verify GATHER_METRICS=true when metrics=true
  - Verify GATHER_METRICS is NOT set when metrics is false/omitted
  - Verify env var is set even with custom images (if ImageStream tests exist)
- **Location**: `e2e_test.go` - "When the controller processes MustGather" Context
- **Priority**: P0 - Critical

#### 3. **Flag Independence** ✅ INCLUDED
- **Why**: Verifies audit and metrics flags work independently
- **Tests**:
  - audit=true, metrics=false → audit command used, no GATHER_METRICS
  - audit=false, metrics=true → GATHER_METRICS set, no audit command
  - audit=true, metrics=true → both behaviors active
- **Location**: `e2e_test.go` - "Should set audit command and metrics env var independently"
- **Priority**: P0 - Critical

#### 4. **End-to-End Workflow with Metrics** ✅ INCLUDED
- **Why**: Verifies complete workflow with new field
- **Tests**:
  - Create MustGather with metrics=true
  - Wait for job completion
  - Verify status shows "Completed"
- **Location**: `e2e_test.go` - "When MustGather job completes" Context
- **Priority**: P1 - High

### Optional (Nice to Have)

These tests provide additional coverage but are not strictly required for this diff:

#### 5. **Custom Image with Metrics**
- **Why**: Verifies metrics flag works with custom must-gather images
- **Prerequisites**: Requires ImageStream setup
- **Test**:
  - Create ImageStream with custom image
  - Create MustGather with imageStreamRef and metrics=true
  - Verify GATHER_METRICS env var is set
- **Location**: Could be added to `e2e_test.go` or existing custom image tests
- **Priority**: P2 - Medium
- **Note**: Not included in generated tests due to ImageStream setup complexity

#### 6. **Metrics Flag with Storage Options**
- **Why**: Verifies metrics works with PersistentVolume storage
- **Test**:
  - Create PVC
  - Create MustGather with metrics=true and PV storage
  - Verify job uses PV and has GATHER_METRICS set
- **Location**: Could be added to existing storage tests
- **Priority**: P2 - Medium
- **Note**: Orthogonal to metrics feature - storage should not affect env var

#### 7. **Metrics Flag with Upload Target**
- **Why**: Verifies metrics works with SFTP upload
- **Test**:
  - Create secret with SFTP credentials
  - Create MustGather with metrics=true and uploadTarget
  - Verify both upload container and GATHER_METRICS are present
- **Location**: Could be added to existing upload tests
- **Priority**: P2 - Medium
- **Note**: Orthogonal to metrics feature - upload should not affect env var

### Not Recommended

These scenarios are NOT relevant to this diff:

#### 8. **Validation Rules for Metrics**
- **Why Not**: The metrics field is a simple boolean with no validation rules beyond type checking
- **Already Covered By**: Integration test suite (`mustgather.testsuite.yaml`)

#### 9. **Metrics Field Immutability**
- **Why Not**: The entire spec is immutable (enforced by CEL rule on MustGather)
- **Already Tested**: Existing immutability tests cover all spec fields

#### 10. **Metrics with Non-Admin Users**
- **Why Not**: No RBAC changes in this diff - metrics doesn't change permissions
- **Already Covered By**: Existing non-admin user tests

## Implementation Recommendations

### 1. Copy Tests to Repository

```bash
# Copy the generated e2e test file
cp output/e2e_must-gather-operator/e2e_test.go \
   test/e2e/mustgather_metrics_test.go

# Review and adjust as needed
# - Verify client variables match (adminClient, testCtx, etc.)
# - Adjust timeouts if needed
# - Add any custom helper functions
```

### 2. Run Tests Against Live Cluster

```bash
# Ensure operator is deployed
oc get deployment must-gather-operator -n must-gather-operator

# Run the new tests
go test -v -tags=e2e ./test/e2e/... -run "MustGather Metrics Flag"

# Run all e2e tests
make test-e2e
```

### 3. Integration with CI/CD

- Add the generated test file to the repository's e2e test suite
- Ensure CI pipeline runs e2e tests on pull requests
- Consider running tests in both OLM and manual deployment modes

## Gaps and Limitations

### 1. Actual Metrics Collection
- **Gap**: Generated tests verify the GATHER_METRICS env var is set, but don't verify that the gather image actually collects metrics when the var is present
- **Reason**: Requires knowledge of gather image internals and would significantly increase test duration
- **Mitigation**: Manual testing or image-specific tests

### 2. Custom Image Scenarios
- **Gap**: Tests don't cover custom images with metrics flag
- **Reason**: Requires ImageStream setup which varies by environment
- **Mitigation**: Manual testing or add tests if ImageStream test infrastructure exists

### 3. Performance Impact
- **Gap**: Tests don't measure performance impact of metrics collection
- **Reason**: Performance testing is typically separate from functional e2e tests
- **Mitigation**: Add performance tests if needed

## Next Steps

1. **Review Generated Tests**: Review `e2e_test.go` and adjust any test-specific details
2. **Copy to Repository**: Add the test file to `test/e2e/mustgather_metrics_test.go`
3. **Run Locally**: Test against a development cluster
4. **Verify CI Integration**: Ensure tests run in CI pipeline
5. **Document**: Update repository documentation with metrics field usage examples

## Test Execution Checklist

- [ ] Operator deployed and running
- [ ] E2E test file added to `test/e2e/`
- [ ] Tests compile without errors
- [ ] Tests pass against live cluster
- [ ] Tests added to CI pipeline
- [ ] Documentation updated with metrics examples
- [ ] Integration tests in `api/v1alpha1/tests/` are also passing

## References

- **Enhancement Proposal**: https://github.com/openshift/enhancements/pull/1906
- **Tracking Issue**: MG-155
- **Generated from diff**: master...HEAD (2 commits, 4 files changed)
