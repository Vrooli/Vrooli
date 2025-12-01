# Test Genie - Known Issues & Limitations

## 🚨 Critical Issues (P0)

None currently identified. All P0 requirements are marked complete and tests pass.

## ⚠️ Major Issues (P1)

### 1. Visual Test Coverage Analysis - Not Implemented
**Status**: Missing
**Impact**: Users cannot visually identify coverage gaps
**PRD Reference**: P1 requirement line 40

The PRD specifies "Visual test coverage analysis with gap identification" but this capability is not implemented:
- No visualization dashboard for coverage metrics
- Gap identification exists in API but no visual representation
- Coverage data stored in database but not rendered in UI

**Recommended Fix**:
- Add coverage visualization charts to UI dashboard
- Implement gap identification heat maps
- Create trend analysis graphs over time

### 2. Performance Regression Detection - Not Implemented
**Status**: Missing
**Impact**: Cannot track performance degradation over time
**PRD Reference**: P1 requirement line 41

No historical trend analysis for performance metrics:
- Performance data collected but not tracked historically
- No baseline comparison for regression detection
- Missing alerting for performance degradation

**Recommended Fix**:
- Store performance baselines in database
- Add trending queries for performance metrics
- Implement threshold-based alerting

### 3. Cross-Scenario Integration Tests - Not Implemented
**Status**: Missing
**Impact**: Cannot validate scenario interdependencies
**PRD Reference**: P1 requirement line 42

Test generation focuses on individual scenarios, not cross-scenario interactions:
- No integration test generation for scenario dependencies
- Missing validation of API contracts between scenarios
- No end-to-end workflow testing across multiple scenarios

**Recommended Fix**:
- Add dependency analysis to test generation
- Generate integration tests for scenario API calls
- Create multi-scenario workflow test suites

### 4. Test Maintenance Automation - Partially Implemented
**Status**: Partial
**Impact**: Test suites require manual maintenance
**PRD Reference**: P1 requirement line 43

Currently has basic test update capability but missing:
- Automatic detection of code changes requiring test updates
- Smart test regeneration when source code changes
- Intelligent test pruning for outdated tests

**Recommended Fix**:
- Implement file watching for source code changes
- Add diff analysis to identify affected tests
- Auto-regenerate tests when API contracts change

### 5. Custom Test Template Management - Not Implemented
**Status**: Missing
**Impact**: Users cannot create reusable test patterns
**PRD Reference**: P1 requirement line 44

No template system for custom test patterns:
- Users cannot save common test patterns
- No template library or marketplace
- Hard-coded test generation patterns only

**Recommended Fix**:
- Create template storage in database
- Add template CRUD endpoints to API
- Build template management UI in dashboard

## 🔧 Minor Issues (P2)

### 1. Legacy Test Format Warning
**Status**: Resolved (phased runner adopted)
**Impact**: Phased testing architecture is now the canonical entrypoint
**Source**: Scenario status diagnostics (updated)

The scenario now uses the shared phased testing runner (`test/run-tests.sh`) and writes phase summaries to `coverage/phase-results/`. `scenario-test.yaml` is no longer required for automated testing workflows.

### 2. Missing CLI Tests
**Status**: Gap
**Impact**: CLI functionality not validated
**Source**: Test infrastructure analysis

No automated tests for CLI commands:
- CLI binary exists and works manually
- No test coverage for CLI argument parsing
- No validation of CLI output formatting

**Recommended Fix**:
- Add BATS tests in cli/test-genie.bats
- Test all CLI commands and flags
- Validate JSON output parsing

### 3. Missing UI Automation Tests
**Status**: Gap
**Impact**: UI functionality not validated
**Source**: Test infrastructure analysis

UI component exists but lacks automation tests:
- No browser-based UI tests
- Missing workflow validation
- Manual testing only

**Recommended Fix**:
- Add browser automation tests using browser-automation-studio
- Test critical UI workflows
- Validate dashboard rendering

### 4. Quality Gates Unchecked
**Status**: Documentation Gap
**Impact**: No formal validation of completion criteria
**PRD Reference**: Lines 62-66

All quality gates in PRD are unchecked:
- [ ] All P0 requirements implemented and tested
- [ ] Generated tests achieve >95% code coverage
- [ ] Test execution completes successfully in CI/CD environment
- [ ] Test results provide actionable failure information
- [ ] Performance benchmarks establish reliable baselines
- [ ] Documentation covers all test generation patterns

**Recommended Fix**:
- Run comprehensive validation suite
- Document test coverage metrics
- Validate against each quality gate criterion
- Update PRD with evidence

### 5. Performance Validation Unchecked
**Status**: Documentation Gap
**Impact**: No proof of meeting performance targets
**PRD Reference**: Lines 722-727

Performance validation items unchecked:
- [ ] Test suite generation completes within 60 seconds
- [ ] Test execution completes within 5 minutes for full vault
- [ ] Coverage analysis completes within 30 seconds
- [ ] Generated tests achieve >95% code coverage
- [ ] False positive rate < 5% for test failures

**Recommended Fix**:
- Run performance benchmarks
- Collect actual metrics
- Compare against targets
- Document results with timestamps

## 📝 Known Limitations

### 1. AI Generation Accuracy
**Type**: Inherent limitation
**Workaround**: Human review workflows recommended
**Future Fix**: Improved prompt engineering and model fine-tuning

Test generation quality depends on AI model capabilities:
- Delegated to App Issue Tracker agents
- Quality varies based on scenario complexity
- Some edge cases may be missed

### 2. Resource Requirements
**Type**: Scalability concern
**Workaround**: Configurable test execution levels
**Future Fix**: Intelligent test optimization

Comprehensive testing requires significant resources:
- Multiple concurrent test executions consume memory
- Large test suites require extended execution time
- Database storage grows with test results

### 3. Delegation Dependency
**Type**: External dependency
**Workaround**: Fallback to local templates if unavailable
**Risk**: Test generation fails if App Issue Tracker is down

Test generation delegated to App Issue Tracker:
- Requires App Issue Tracker scenario to be running
- Network communication adds latency
- Fallback templates are less sophisticated

## 🔄 Migration Tasks

### Legacy Test Format Migration
**Priority**: Medium
**Effort**: 2-4 hours
**Status**: In progress (runner live; reporting enhancements pending)

Current: `test/run-tests.sh` (phased runner)
Target: Maintain phased test suite instrumentation

Completed steps:
1. Introduced shared runner in `test/run-tests.sh`
2. Aligned structure phase and documentation with phased tooling
3. Updated `.vrooli/service.json` to execute the phased suite
4. Preserved existing phase scripts for unit/cli/integration/business checks

Next reinforcement:
- Publish runner outputs to central dashboards
- Continue expanding phase coverage (performance, automation-report links)

## 📈 Progress Tracking

### Overall Completion
- **P0 Requirements**: 6/6 complete (100%)
- **P1 Requirements**: 0/5 complete (0%)
- **P2 Requirements**: 0/4 complete (0%)
- **Quality Gates**: 0/6 validated (0%)
- **Performance Validation**: 0/5 validated (0%)

### Last Updated
**Date**: 2025-10-03
**Baseline Established**: Initial PROBLEMS.md creation
**Next Review**: After P1 improvements

---

**Note**: This document tracks known issues and limitations. Update after each improvement iteration to maintain accuracy.
