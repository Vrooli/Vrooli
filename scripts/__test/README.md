# Vrooli Testing Infrastructure

This directory contains Vrooli's comprehensive testing infrastructure, providing three complementary approaches to ensure code quality and reliability.

## 🎯 Overview

Vrooli uses a three-pronged testing strategy:

1. **Unit Testing with Mocks** (`fixtures/`) - Fast, isolated tests using BATS framework
2. **Integration Testing** (`resources/`) - Real-world validation with actual services
3. **Test Execution & Analysis** (`shell/`) - Runners, caching, and performance profiling

## 🚀 Quick Start Guide

### Which Testing Framework Should I Use?

| Scenario | Framework | Directory | Example |
|----------|-----------|-----------|---------|
| Testing a function in isolation | Unit w/ Mocks | `fixtures/` | Mock external dependencies |
| Testing Docker commands | Unit w/ Mocks | `fixtures/` | Mock docker responses |
| Testing real API integration | Integration | `resources/` | Test against live services |
| Testing multi-service workflows | Integration | `resources/` | Validate end-to-end flows |
| Running all tests efficiently | Test Runner | `shell/` | Cached, parallel execution |

### Common Testing Patterns

#### 1. Unit Test with Mocked Dependencies
```bash
#!/usr/bin/env bats
source "$(dirname "${BATS_TEST_FILENAME}")/../../fixtures/setup.bash"

setup() { 
    vrooli_auto_setup  # Automatically detects test type and sets up mocks
}
teardown() { 
    vrooli_cleanup_test  # Cleans up test environment
}

@test "docker container starts" {
    run docker run -d nginx
    assert_success
    assert_output_contains "container_id"
}
```

#### 2. Integration Test with Real Services
```bash
#!/usr/bin/env bats
source "$PROJECT_ROOT/scripts/__test/resources/common_test_helper.bash"

setup() { setup_resource_test "ollama"; }
teardown() { cleanup_test_environment; }

@test "ollama generates response" {
    run curl -X POST "$OLLAMA_URL/api/generate" \
        -d '{"model":"llama2","prompt":"Hello"}'
    assert_success
    assert_json_contains ".response"
}
```

#### 3. Running Tests with Caching
```bash
# Run all tests with intelligent caching
cd scripts/__test/shell/core
./run-tests.sh

# Profile test performance
./test-profiler.sh --analyze

# Clear cache if needed
./cache-manager.sh --clear
```

## 📁 Directory Structure

```
__test/
├── fixtures/           # BATS unit testing with mocks
│   ├── bats/          # Core BATS infrastructure
│   │   ├── core/      # Common setup and assertions
│   │   ├── mocks/     # Mock implementations
│   │   ├── docs/      # BATS documentation
│   │   └── templates/ # Test templates
│   └── README.md      # Detailed BATS guide
│
├── resources/         # Integration testing framework
│   ├── framework/     # Core testing infrastructure
│   ├── single/        # Individual resource tests
│   ├── scenarios/     # Multi-resource workflows
│   ├── fixtures/      # Test data (images, docs, audio)
│   └── README.md      # Integration testing guide
│
├── shell/             # Test execution and analysis
│   ├── core/          # Active test infrastructure
│   │   ├── run-tests.sh    # Main test runner
│   │   ├── cache-manager.sh # Test result caching
│   │   └── test-profiler.sh # Performance analysis
│   └── README.md      # Runner documentation
│
└── README.md          # This file
```

## 🔧 Key Features

### Unit Testing (fixtures/)
- **35+ Custom Assertions**: Specialized checks for Docker, HTTP, files, JSON
- **Lazy-Loaded Mocks**: Only loads what your test needs
- **Resource-Aware**: Automatically configures mocks for specific services
- **Fast Execution**: Sub-second test runs with full isolation

### Integration Testing (resources/)
- **Real Service Validation**: Tests against actual running services
- **Health Check Discovery**: Only tests healthy, enabled resources
- **Scenario Testing**: Complex multi-service workflows
- **Comprehensive Fixtures**: Real test data for images, documents, audio, workflows

### Test Execution (shell/)
- **Intelligent Caching**: Skips unchanged tests for 10x faster runs
- **Parallel Execution**: Runs tests concurrently where possible
- **Performance Profiling**: Identifies slow tests and bottlenecks
- **Detailed Reporting**: Clear output with actionable recommendations

## 📚 Documentation

Each subdirectory contains detailed documentation:

- **[BATS Testing Guide](fixtures/bats/README.md)** - Mock-based unit testing
- **[Integration Testing Guide](resources/README.md)** - Real service validation
- **[Test Runner Guide](shell/README.md)** - Execution and optimization

### Additional Resources

- **[Getting Started](resources/docs/GETTING_STARTED.md)** - 5-minute testing tour
- **[Common Patterns](resources/docs/COMMON_PATTERNS.md)** - Copy-paste examples
- **[Troubleshooting](resources/docs/TROUBLESHOOTING.md)** - Solutions to common issues
- **[Architecture Overview](resources/docs/ARCHITECTURE_OVERVIEW.md)** - System design

## 🎨 Best Practices

### When to Use Each Framework

**Use Unit Tests (fixtures/) when:**
- Testing logic in isolation
- External dependencies should be controlled
- Fast feedback is critical
- Testing error conditions difficult to reproduce

**Use Integration Tests (resources/) when:**
- Validating real service behavior
- Testing complex workflows
- Ensuring API compatibility
- Verifying end-to-end functionality

**Use Test Runners (shell/) when:**
- Running full test suites
- Analyzing test performance
- Setting up CI/CD pipelines
- Debugging test failures

### Writing Effective Tests

1. **Name tests descriptively**: `@test "ollama generates code from natural language prompt"`
2. **One assertion per test**: Keep tests focused and debuggable
3. **Use appropriate assertions**: Choose specific assertions over generic ones
4. **Clean up resources**: Always include teardown functions
5. **Document complex tests**: Add comments explaining non-obvious behavior

## 🚦 Running Tests

### Quick Commands

```bash
# Run all shell script tests (fastest)
pnpm test:shell

# Run specific BATS test file
bats scripts/__test/fixtures/bats/test_new_infrastructure.bats

# Run integration tests for a specific resource
cd scripts/__test/resources
./run.sh --resource ollama

# Run scenario tests
./run.sh --scenario research-assistant

# Profile test performance
cd scripts/__test/shell/core
./test-profiler.sh --analyze --threshold 1.0
```

### CI/CD Integration

Tests are automatically run in CI/CD pipelines. See [.github/workflows/](/.github/workflows/) for configuration.

## 🤝 Contributing

When adding new tests:

1. **Choose the right framework** based on the guidelines above
2. **Follow existing patterns** in the respective directory
3. **Add documentation** for complex test scenarios
4. **Run tests locally** before submitting PRs
5. **Update this README** if adding new testing capabilities

## 📈 Metrics & Performance

Current testing infrastructure performance:

- **Unit Tests**: ~0.1s per test (with mocks)
- **Integration Tests**: ~2-5s per test (real services)
- **Full Suite**: ~3-5 minutes (with caching)
- **Cache Hit Rate**: ~85% on average
- **Parallel Speedup**: 3-4x on multi-core systems

## 🔍 Troubleshooting

Common issues and solutions:

1. **"command not found: bats"** - Run `pnpm install` to install dependencies
2. **"Resource not healthy"** - Ensure services are running with `docker ps`
3. **"Cache corrupted"** - Clear with `./cache-manager.sh --clear`
4. **"Test timeout"** - Increase timeout in test or check service health

For more help, see the troubleshooting guides in each subdirectory.

---

**Need help?** Check the detailed documentation in each subdirectory or ask in the project chat.