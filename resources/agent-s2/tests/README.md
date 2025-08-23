# Agent S2 Test Suite

Comprehensive test suite for Agent S2 dual-mode functionality, covering unit tests, integration tests, security validation, and end-to-end workflows.

## Overview

The test suite validates:
- **Dual-mode functionality** (sandbox and host modes)
- **Environment discovery** and capability detection
- **Security monitoring** and threat detection
- **Mode switching** and configuration management
- **API endpoints** and responses
- **Error handling** and recovery

## Test Structure

```
tests/
├── unit/                    # Unit tests
│   └── test_security.py     # Security system tests
├── integration/             # Integration tests  
│   └── test_mode_workflow.py # End-to-end workflow tests
├── modes/                   # Mode-specific tests
│   └── test_dual_mode.py    # Dual-mode functionality tests
├── results/                 # Test results and reports
├── run_tests.sh            # Test runner script
└── README.md               # This file
```

## Quick Start

### Prerequisites

- Python 3.7+
- Docker (for integration tests)
- Agent S2 codebase

### Running Tests

```bash
# Run all tests
./run_tests.sh

# Run specific test types
./run_tests.sh unit          # Unit tests only
./run_tests.sh integration   # Integration tests only
./run_tests.sh modes        # Mode-specific tests
./run_tests.sh security     # Security tests only
./run_tests.sh quick        # Quick tests (exclude slow tests)

# With additional options
./run_tests.sh --verbose all              # Verbose output
./run_tests.sh --coverage unit            # With coverage report
./run_tests.sh --html-report integration  # Generate HTML report
./run_tests.sh --bail security           # Stop on first failure
```

### Setting Up Test Environment

```bash
# Setup Python virtual environment and dependencies
./run_tests.sh --setup-venv

# Clean previous test results
./run_tests.sh --clean
```

## Test Categories

### Unit Tests

**Location**: `unit/`
**Purpose**: Test individual components in isolation
**Coverage**: 
- Configuration system
- Environment discovery
- Security validation
- Mode context management

**Example**:
```bash
./run_tests.sh unit --coverage
```

### Integration Tests

**Location**: `integration/`
**Purpose**: Test complete workflows and system interactions
**Coverage**:
- Installation workflows
- Mode switching
- API functionality
- Error handling and recovery
- Performance characteristics

**Requirements**: Running Agent S2 instance
**Example**:
```bash
# Start Agent S2 first
../manage.sh --action start --mode sandbox

# Run integration tests
./run_tests.sh integration --verbose
```

### Mode-Specific Tests

**Location**: `modes/`
**Purpose**: Test dual-mode functionality comprehensively
**Coverage**:
- Sandbox mode capabilities
- Host mode capabilities  
- Mode switching
- Security constraints
- Application discovery

**Example**:
```bash
./run_tests.sh modes --html-report
```

### Security Tests

**Purpose**: Validate security monitoring and validation
**Coverage**:
- Threat detection
- Audit logging
- Input validation
- Security constraint enforcement

**Example**:
```bash
./run_tests.sh security --verbose
```

## Test Configuration

### Environment Variables

Key environment variables for testing:

```bash
# Mode configuration
export AGENT_S2_MODE=sandbox                    # Current mode
export AGENT_S2_HOST_MODE_ENABLED=false         # Enable host mode
export AGENT_S2_HOST_AUDIT_LOGGING=true         # Enable audit logging

# API configuration
export AGENT_S2_PORT=4113                       # API port
export AGENT_S2_HOST=localhost                  # API host

# Security configuration
export AGENT_S2_HOST_SECURITY_PROFILE=agent-s2-host
export AGENT_S2_HOST_DISPLAY_ACCESS=false
```

### Pytest Configuration

The test suite uses pytest with these key configurations:
- **Markers**: `@pytest.mark.slow` for long-running tests
- **Fixtures**: Shared setup/teardown for Agent S2 testing
- **Timeouts**: Appropriate timeouts for API calls and mode switches
- **Parallel execution**: Supported for unit tests (use `--parallel`)

## Test Reports

### Coverage Reports

Generate code coverage reports:
```bash
./run_tests.sh --coverage all
```

Coverage reports are saved to:
- `results/coverage_html/index.html` (HTML report)
- `results/coverage.xml` (XML format)

### HTML Reports

Generate detailed HTML test reports:
```bash
./run_tests.sh --html-report all
```

HTML reports include:
- Test results and status
- Execution times
- Error details and tracebacks
- Environment information

### JUnit XML

Generate JUnit XML for CI/CD integration:
```bash
./run_tests.sh --junit-xml all
```

## Troubleshooting

### Common Issues

**1. Agent S2 Not Running**
```
Error: Agent S2 is not running - integration tests may fail
```
**Solution**: Start Agent S2 before running integration tests:
```bash
../manage.sh --action start --mode sandbox
```

**2. Import Errors**
```
ModuleNotFoundError: No module named 'agent_s2'
```
**Solution**: Ensure you're running from the correct directory and Python path is set:
```bash
cd /path/to/agent-s2/tests
./run_tests.sh --setup-venv
```

**3. Permission Errors (Host Mode)**
```
Host mode validation failed
```
**Solution**: Install security components:
```bash
sudo ../security/install-security.sh install
```

**4. Timeout Errors**
```
requests.exceptions.Timeout
```
**Solution**: Increase timeout values or check Agent S2 performance:
```bash
# Check Agent S2 logs
../manage.sh --action logs

# Check system resources
docker stats agent-s2
```

### Test Isolation

Tests are designed to be isolated and can be run independently:
- **Unit tests**: No external dependencies
- **Integration tests**: Require running Agent S2 but restore state
- **Mode tests**: Handle mode switching gracefully

### Debugging Tests

Run tests with maximum verbosity:
```bash
./run_tests.sh --verbose modes
```

Run a specific test:
```bash
cd modes
python -m pytest test_dual_mode.py::TestConfiguration::test_config_mode_detection -v
```

Use Python debugger:
```python
import pytest
pytest.main(["-v", "-s", "--pdb", "test_file.py::test_function"])
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Agent S2 Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up Python
        uses: actions/setup-python@v2
        with:
          python-version: '3.8'
      
      - name: Install dependencies
        run: |
          cd scripts/resources/agents/agent-s2/tests
          ./run_tests.sh --setup-venv
      
      - name: Run unit tests
        run: |
          cd scripts/resources/agents/agent-s2/tests  
          ./run_tests.sh unit --coverage --junit-xml
      
      - name: Start Agent S2
        run: |
          cd scripts/resources/agents/agent-s2
          ./manage.sh --action install --mode sandbox --yes yes
      
      - name: Run integration tests
        run: |
          cd scripts/resources/agents/agent-s2/tests
          ./run_tests.sh integration --junit-xml
      
      - name: Upload test results
        uses: actions/upload-artifact@v2
        if: always()
        with:
          name: test-results
          path: scripts/resources/agents/agent-s2/tests/results/
```

## Contributing

When adding new tests:

1. **Follow naming conventions**: `test_*.py` files, `test_*` functions
2. **Use appropriate markers**: `@pytest.mark.slow` for long tests
3. **Include docstrings**: Describe what the test validates
4. **Handle cleanup**: Restore state in fixtures
5. **Test both modes**: Validate sandbox and host mode behavior when applicable

### Test Template

```python
def test_new_functionality(agent_fixture):
    """Test description of what this validates"""
    # Arrange
    initial_state = get_current_state()
    
    # Act
    result = perform_action()
    
    # Assert
    assert result == expected_result
    
    # Cleanup (if needed)
    restore_state(initial_state)
```

## Performance Benchmarks

The test suite includes performance tests to ensure:
- API responses < 5 seconds
- Mode switching < 30 seconds  
- Concurrent request handling
- Memory usage within limits

Run performance tests:
```bash
./run_tests.sh integration --verbose
```

## Security Testing

Security tests validate:
- **Threat detection**: Suspicious patterns are flagged
- **Input validation**: Malicious inputs are blocked
- **Access control**: Mode restrictions are enforced
- **Audit logging**: Security events are logged

Security tests run in both modes to ensure constraints are appropriate for each security level.

---

For questions or issues with the test suite, please check the main Agent S2 documentation or create an issue in the project repository.