# Known Issues and Problems

## Recent Improvements (2025-10-03)

### ✅ Fixed: Initial Discovery Delay
- **Previous Problem**: Scenarios were not discovered until 60 seconds after API startup
- **Solution**: Added initial discovery call before starting server
- **Status**: Fixed - API now discovers 10 scenarios immediately on startup
- **Verification**: Logs show "✅ Initial discovery complete: 10 scenarios found"

### ✅ Migrated to Phased Testing Architecture
- **Previous Problem**: Used legacy scenario-test.yaml format
- **Solution**: Implemented comprehensive phased testing with 6 test phases
- **Status**: Complete - All smoke tests passing (structure + integration)
- **Test Coverage**: Structure, Dependencies, Unit, Integration, Business, Performance phases

### ✅ Added Unit Test Coverage
- **Previous Problem**: No unit tests for Go API code
- **Solution**: Created orchestrator_test.go with comprehensive coverage
- **Status**: Complete - 8 unit tests covering core orchestrator functionality
- **Verification**: All tests pass with `go test -v ./...`

## Current Issues

### 1. Dynamic Port Assignment
- **Problem**: The scenario uses dynamic port allocation that changes on each restart
- **Impact**: Minimal - UI and CLI correctly discover ports dynamically
- **Status**: Working as designed - not an issue
- **Note**: This is by design for the lifecycle system

### 2. Legacy scenario-test.yaml
- **Problem**: Legacy test file still present
- **Impact**: Causes warning in structure tests
- **Next Steps**: Can be removed after confirming phased tests cover all cases
- **Recommendation**: Delete scenario-test.yaml in next improvement cycle

## Recommendations for Next Improvement

1. **Remove Legacy Files**: Delete scenario-test.yaml after final verification
2. **Add Handler Unit Tests**: Create tests for HTTP handlers with httptest
3. **Add CLI BATS Tests**: Create comprehensive CLI test suite in test/cli/
4. **Performance Baselines**: Document and enforce performance thresholds
5. **UI Unit Tests**: Add Jest tests for UI components