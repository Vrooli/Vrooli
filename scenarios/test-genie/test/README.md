# Test Genie Test Suite

Comprehensive testing infrastructure for the Test Genie scenario, covering core functionality, database integration, AI services, and performance validation.

## 🚀 Quick Start

```bash
# Run all tests (recommended)
./run-all-tests.sh

# Run all tests in parallel (faster)
PARALLEL=true ./run-all-tests.sh

# Run individual test suites
./test-basic-functionality.sh
./test-database-integration.sh
./test-ai-integration.sh
```

## 📋 Test Suites

### 1. Basic Functionality (`test-basic-functionality.sh`)
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

### 2. Database Integration (`test-database-integration.sh`)
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

### 3. AI Integration (`test-ai-integration.sh`) 
**Purpose**: Tests OpenCode integration, AI generation, and fallback mechanisms

**Test Cases**:
- ✅ OpenCode service health and model availability
- ✅ AI-powered test suite generation
- ✅ Fallback generation mechanism testing
- ✅ Performance test type generation
- ✅ Security test type generation  
- ✅ AI response quality validation
- ✅ Concurrent AI generation (3 parallel requests)

**Prerequisites**:
- OpenCode resource running with models available
- AI service integration configured

### 4. Master Test Runner (`run-all-tests.sh`)
**Purpose**: Orchestrates all test suites with comprehensive reporting

**Features**:
- 🔄 Sequential or parallel test execution
- 📊 Comprehensive test reporting and statistics
- 📁 Individual test logging with log retention
- ⚡ Performance timing for each test suite
- 🎯 Success rate calculation and recommendations
- 🔧 Debugging information and useful commands

## 📊 Understanding Test Results

### Success Indicators
```bash
✅ PASSED - Test completed successfully
⚠️  WARNING - Test passed with warnings (non-critical issues)
❌ FAILED - Test failed (requires investigation)
```

### Log Analysis
Test logs are stored in `/tmp/test-genie-logs-<pid>/`:
- `basic-test.log` - Core functionality test output
- `database-test.log` - Database integration test output  
- `ai-test.log` - AI integration test output

### Performance Benchmarks
- **API Response Time**: < 1000ms for health checks
- **Database Operations**: < 5000ms for 10 concurrent operations
- **AI Generation**: Variable based on model size and complexity
- **Concurrent Load**: Should handle 5 simultaneous requests

## 🔧 Configuration and Customization

### Environment Variables
```bash
# Parallel execution mode
PARALLEL=true ./run-all-tests.sh

# Custom API port (auto-detected by default)
API_PORT=8200 ./test-basic-functionality.sh

# Custom OpenCode configuration
./test-ai-integration.sh
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
# Check OpenCode resource
resource-opencode status

# Verify available models
resource-opencode models | head -5
```

### Performance Issues

**Slow database operations**:
- Check PostgreSQL resource utilization
- Verify connection pool configuration
- Review database logs for bottlenecks

**AI generation timeouts**:
- Verify OpenCode models are available (`resource-opencode models`)
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
- ✅ OpenCode AI service integration
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
- Check resource status: `resource-postgres status`, `resource-opencode status`
- Validate configuration: Review `.vrooli/service.json`
