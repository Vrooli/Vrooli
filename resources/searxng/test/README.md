# SearXNG Resource Tests

This directory contains the new v2.0 CLI contract testing framework for the SearXNG resource.

## Test Structure

The tests are organized into phases with specific time limits:

### Test Runner
- **`run-tests.sh`** - Main test orchestrator with phase selection and verbose output

### Test Phases
- **`phases/test-smoke.sh`** - Quick health validation (30s max)
- **`phases/test-unit.sh`** - Library function tests (60s max)  
- **`phases/test-integration.sh`** - Full end-to-end testing (120s max)

## Usage

```bash
# Run all test phases
./run-tests.sh

# Run specific phases
./run-tests.sh smoke
./run-tests.sh smoke unit
./run-tests.sh integration

# Verbose output
./run-tests.sh --verbose all

# Continue on failures
./run-tests.sh --continue all
```

## Test Coverage

### Smoke Tests
- ✅ Container existence and status
- ✅ Port accessibility
- ✅ Web interface availability
- ✅ Stats endpoint functionality
- ✅ Basic search API validation

### Unit Tests
- ✅ Configuration validation functions
- ✅ Status data collection
- ✅ Container state function accessibility
- ✅ URL construction validation
- ✅ Configuration display functions
- ✅ Network management function availability

### Integration Tests
- 🚧 JSON search API with multiple queries
- 🚧 Multiple output formats (JSON, XML, CSV)
- 🚧 Search categories and pagination
- 🚧 Safe search and language parameters
- 🚧 Rate limiting behavior
- 🚧 Privacy and security headers

## Status

- **Smoke Tests**: ✅ Fully functional
- **Unit Tests**: ✅ Fully functional
- **Integration Tests**: 🚧 Framework complete, some tests need debugging

## Migration from Old Tests

The old integration test (`integration-test.sh.old`) has been replaced with this new phase-based structure that:

1. **Eliminates namespace issues** with test framework functions
2. **Follows v2.0 CLI contract standards** seen in vault/k6 resources
3. **Provides better isolation** between test phases
4. **Includes proper timeout handling** and error recovery
5. **Supports verbose output** for debugging

## Notes

- Tests automatically detect and avoid readonly variable conflicts
- Container state functions are tested with appropriate timeouts
- Integration tests use direct API calls to avoid library function hangs
- All tests follow the standard v2.0 emoji indicators (✓, ✗, ⚠)