# Test Genie Test Suite

Comprehensive testing infrastructure for the Test Genie scenario, covering core functionality, database integration, AI services, and performance validation.

## 🚀 Quick Start

```bash
# Run the full phased suite (default preset executes all phases)
./run-tests.sh

# Focused runs
./run-tests.sh quick        # structure + unit
./run-tests.sh smoke        # structure + dependencies + cli
./run-tests.sh integration  # single phase

# Run individual phase scripts directly
./phases/test-structure.sh
./phases/test-integration.sh
./phases/test-dependencies.sh
./phases/test-business.sh
```

> ℹ️ `run-tests.sh` wraps the shared Vrooli test runner (`scripts/scenarios/testing/shell/runner.sh`), which manages lifecycle hooks, artifacts, caching, and presets. The legacy `run-all-tests.sh` now delegates to `run-tests.sh` for backward compatibility.

## 📋 Test Suites

### 1. Structure Validation (`test/phases/test-structure.sh`)
**Purpose**: Confirms the scenario layout, lifecycle assets, and key entrypoints exist

**Checks**:
- ✅ Required directories (`api`, `cli`, `ui`, `prompts`, `initialization`, `test/phases`)
- ✅ Lifecycle contract files (`Makefile`, `.vrooli/service.json`, `test/run-tests.sh`)
- ✅ Primary entrypoints (`api/main.go`, `cli/test-genie`, `ui/server.js`)

**Prerequisites**:
- None — runs entirely offline

### 2. Integration (`test/phases/test-integration.sh`)
**Purpose**: Tests core API functionality and CLI integration

**Test Cases**:
- ✅ API health check
- ✅ Test suite generation via API
- ✅ Test suite retrieval and validation
- ✅ Coverage analysis functionality
- ✅ Test vault creation and management
- ✅ System status monitoring
- ✅ CLI integration (if installed)
- ✅ Basic load testing (5 concurrent requests)

**Prerequisites**: 
- Test Genie scenario running (`vrooli scenario run test-genie`)
- Basic system dependencies (curl, jq)

### 3. Dependencies (`test/phases/test-dependencies.sh`)
**Purpose**: Tests database connectivity, persistence, and performance

**Test Cases**:
- ✅ Database connection health verification
- ✅ Data persistence (create/retrieve operations)
- ✅ List operations and query functionality
- ✅ Complex database queries (coverage analysis)
- ✅ Transaction integrity testing
- ✅ Concurrent database operations (5 parallel writes)
- ✅ Database performance benchmarking
- ✅ Data cleanup verification

**Prerequisites**:
- PostgreSQL resource available and configured
- Test Genie API running with database connectivity

### 4. Business Logic & AI (`test/phases/test-business.sh`)
**Purpose**: Tests App Issue Tracker delegation, AI generation, and fallback mechanisms

**Test Cases**:
- ✅ App Issue Tracker health and API availability
- ✅ AI-powered test suite generation
- ✅ Fallback generation mechanism testing
- ✅ Performance test type generation
- ✅ Security test type generation  
- ✅ AI response quality validation
- ✅ Concurrent AI generation (3 parallel requests)

**Prerequisites**:
- App Issue Tracker scenario running and reachable (`vrooli scenario status app-issue-tracker`)
- AI service integration configured

### Master Test Runner (`run-tests.sh`)
**Purpose**: Provides a unified entrypoint for phased testing, presets, and managed runtime orchestration.

**Features**:
- 📁 Per-phase logs and artifacts under `test/artifacts/`
- 📊 Structured JSON results (`coverage/phase-results/*.json`)
- ⚡ Timeout controls per phase with configurable presets
- 🎯 Quick/smoke/comprehensive presets for iterative workflows
- 🔧 Automatic lifecycle management for runtime-dependent phases

## 📊 Understanding Test Results

### Success Indicators
```bash
✅ PASSED - Test completed successfully
⚠️  WARNING - Test passed with warnings (non-critical issues)
❌ FAILED - Test failed (requires investigation)
```

### Log Analysis
Test logs are stored in `test/artifacts/` by default:
- `phase-structure.log` - Structural validation output
- `phase-dependencies.log` - Dependency and resource checks
- `phase-cli.log` - CLI validation output
- `phase-integration.log` - Core functionality + API/CLI coverage
- `phase-business.log` - AI and delegation workflow validation

### Performance Benchmarks
- **API Response Time**: < 1000ms for health checks
- **Database Operations**: < 5000ms for 10 concurrent operations
- **AI Generation**: Variable based on model size and complexity
- **Concurrent Load**: Should handle 5 simultaneous requests

## 🔧 Configuration and Customization

### Environment Variables
```bash
# Custom API port (auto-detected by default)
API_PORT=8200 ./phases/test-integration.sh

# Custom delegation target configuration
./phases/test-business.sh
```

### Test Customization
Edit individual test scripts to modify:
- Test data and scenarios
- Timeout values and performance thresholds
- Concurrent operation counts
- Coverage targets and analysis depth

## 🔍 Troubleshooting

### Common Issues

**"test-genie scenario is not running"**
```bash
# Start the scenario
vrooli scenario run test-genie
# Or using Makefile
make run
```

**"curl: command not found" or "jq: command not found"**
```bash
# Ubuntu/Debian
sudo apt install curl jq

# macOS
brew install curl jq
```

**Database connection failures**
```bash
# Check PostgreSQL resource status
resource-postgres status

# Verify environment variables
env | grep POSTGRES
```

**AI integration failures**
```bash
# Check App Issue Tracker scenario
vrooli scenario status app-issue-tracker

# Inspect health endpoint
curl -s http://localhost:8090/health | jq
```

### Performance Issues

**Slow database operations**:
- Check PostgreSQL resource utilization
- Verify connection pool configuration
- Review database logs for bottlenecks

**AI generation timeouts**:
- Verify App Issue Tracker is healthy (`curl -s http://localhost:8090/health`)
- Check available system memory
- Consider smaller model alternatives

## 📈 Test Coverage

### API Endpoint Coverage
- ✅ `/health` - Health monitoring
- ✅ `/api/v1/test-suite/generate` - Test generation
- ✅ `/api/v1/test-suite/:id` - Suite retrieval
- ✅ `/api/v1/test-suites` - Suite listing
- ✅ `/api/v1/test-analysis/coverage` - Coverage analysis
- ✅ `/api/v1/test-vault/create` - Vault creation
- ✅ `/api/v1/system/status` - System monitoring

### Integration Coverage
- ✅ PostgreSQL database operations
- ✅ App Issue Tracker delegation integration
- ✅ CLI tool functionality
- ✅ Concurrent operation handling
- ✅ Error handling and fallback mechanisms

## 🎯 Best Practices

### Running Tests
1. **Always run from scenario root**: Tests use relative paths
2. **Check prerequisites**: Ensure all resources are running
3. **Review logs**: Check detailed logs for any warnings
4. **Run after changes**: Execute tests after code modifications
5. **Use parallel mode**: For faster feedback during development

### Test Development
1. **Follow naming conventions**: `test-<category>-<purpose>.sh`
2. **Include cleanup**: Always clean up test artifacts
3. **Use dynamic ports**: Leverage `vrooli scenario port` command
4. **Comprehensive logging**: Include detailed test output
5. **Graceful failure**: Provide clear error messages and next steps

## 🚀 Advanced Usage

### Custom Test Scenarios
```bash
# Create custom test data
mkdir -p /tmp/my-test-scenario/{src,api,ui}
echo "export function myFunc() {}" > /tmp/my-test-scenario/src/main.js

# Run coverage analysis on custom scenario
curl -X POST http://localhost:$API_PORT/api/v1/test-analysis/coverage \
  -H "Content-Type: application/json" \
  -d '{"scenario_name":"my-scenario","source_code_paths":["/tmp/my-test-scenario/src"]}'
```

### Performance Testing
```bash
# Load test with custom parameters  
for i in {1..50}; do
  curl -s http://localhost:$API_PORT/health &
done
wait
```

### Integration with CI/CD
```bash
# In CI pipeline
./run-all-tests.sh
if [ $? -eq 0 ]; then
  echo "All tests passed - ready for deployment"
else
  echo "Tests failed - blocking deployment"
  exit 1
fi
```

---

## 📞 Support

For issues with tests:
1. Check test logs in `/tmp/test-genie-logs-*/`
2. Verify scenario is running: `vrooli scenario status test-genie`
3. Review test output for specific error details
4. Ensure all prerequisites are met

For Test Genie functionality issues:
- Review scenario logs: `make logs` or `vrooli scenario logs test-genie`
- Check resource status: `resource-postgres status`, `vrooli scenario status app-issue-tracker`
- Validate configuration: Review `.vrooli/service.json`
