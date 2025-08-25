# Vrooli Scenario Testing Framework

**A declarative testing framework that replaces 1000+ lines of boilerplate with clean, maintainable YAML configurations.**

## 🎯 Overview

The Scenario Testing Framework provides a robust, extensible testing system for Vrooli's business scenarios. It follows a **declarative approach** where tests are defined in YAML configuration files, eliminating repetitive boilerplate code while maintaining full testing capabilities.

### Key Benefits

- **🚀 Rapid Development**: Define tests in YAML instead of writing repetitive shell scripts
- **🔄 Reusable Components**: Modular handlers and clients that work across all scenarios
- **🛡️ Graceful Degradation**: Tests continue with warnings when optional services are unavailable
- **📊 Rich Reporting**: Detailed test results with timing and status information
- **🎯 Service Discovery**: Intelligent URL resolution from multiple sources
- **⚡ Extensible Architecture**: Easy to add new test types and resource handlers

## 🏗️ Architecture

```
framework/
├── scenario-test-runner.sh    # Main orchestrator (like pytest)
├── run-all-scenarios.sh       # Batch runner for all scenarios
├── clients/                   # Resource client libraries
│   ├── common.sh             # Service discovery & HTTP utilities
│   └── ollama.sh             # Ollama-specific operations
├── handlers/                 # Test type implementations
│   ├── http.sh               # HTTP endpoint testing
│   ├── custom.sh             # Custom business logic tests
│   ├── database.sh           # Database operations
│   └── chain.sh              # Multi-step workflows
└── validators/               # Validation components
    ├── resources.sh          # Service health checks
    └── structure.sh          # File/directory validation
```

### Component Responsibilities

#### **🎛️ scenario-test-runner.sh** (Main Orchestrator)
- Parses YAML test configurations
- Orchestrates test execution across different handlers
- Provides consistent reporting and error handling
- Manages test lifecycle (setup, execution, teardown)

#### **📚 clients/** (Resource Libraries)
**Purpose**: Provide standardized interfaces for interacting with different resources

- **`common.sh`**: Core utilities used by all components
  - Service discovery (service.json, environment variables, defaults)
  - HTTP request handling with retries and error handling
  - Health checking for multiple service types
  - URL resolution and validation
  - JSON/YAML parsing utilities

- **`ollama.sh`**: Ollama-specific operations
  - Model management (list, pull, ensure availability)
  - Text generation with validation
  - Performance testing
  - System information retrieval

#### **⚙️ handlers/** (Test Type Implementations)
**Purpose**: Execute specific types of tests defined in YAML configurations

- **`http.sh`**: HTTP endpoint testing
  - Multi-endpoint health checking
  - Request/response validation
  - Authentication testing
  - File upload testing
  - Mock response generation for unavailable services

- **`custom.sh`**: Custom business logic tests
  - Execute scenario-specific test scripts
  - Provide framework utilities to custom tests
  - Template generation for new custom tests
  - Integration with framework reporting

- **`database.sh`**: Database operations
  - SQL execution and validation
  - Schema verification
  - Data integrity checks
  - Connection testing

- **`chain.sh`**: Multi-step workflows
  - Sequential test execution
  - State passing between steps
  - Conditional execution
  - Rollback on failure

#### **✅ validators/** (Validation Components)
**Purpose**: Validate different aspects of scenarios before and during testing

- **`resources.sh`**: Service health checks
  - Resource availability validation
  - Configuration parsing from service.json
  - Health endpoint checking
  - Graceful degradation handling

- **`structure.sh`**: File/directory validation
  - Required file/directory existence
  - File format validation (YAML, JSON, SQL, shell)
  - Content validation (metadata, manifest)
  - Configuration-driven structure checks

## 📋 Usage Guide

### Basic Usage

```bash
# Test a single scenario
./framework/scenario-test-runner.sh --scenario /path/to/scenario

# Test with verbose output
./framework/scenario-test-runner.sh --scenario /path/to/scenario --verbose

# Dry run (show what would be executed)
./framework/scenario-test-runner.sh --scenario /path/to/scenario --dry-run
```

### Batch Testing

```bash
# Test all scenarios
./framework/run-all-scenarios.sh

# Quick mode (shorter timeouts)
./framework/run-all-scenarios.sh --quick

# Filter scenarios
./framework/run-all-scenarios.sh --filter "multi-modal*"

# Parallel execution (experimental)
./framework/run-all-scenarios.sh --parallel
```

### Integration with Main Test Suite

```bash
# From project root
pnpm test:scenarios              # Test all scenarios
pnpm test:scenarios:quick        # Quick mode
pnpm test:scenarios:verbose      # Verbose output
```

## 📄 Configuration Format

### scenario-test.yaml Structure

```yaml
version: 1.0
scenario: my-scenario

# File/directory structure validation
structure:
  required_files:
    - metadata.yaml
    - manifest.yaml
    - README.md
    - initialization/database/schema.sql
  required_dirs:
    - initialization
    - deployment
    - ui

# Resource requirements
resources:
  required: [ollama, windmill, postgres]    # Must be available
  optional: [whisper, comfyui]              # Can fail gracefully
  health_timeout: 30

# Test definitions
tests:
  - name: "Service Health Check"
    type: http
    service: ollama
    endpoint: /api/tags
    method: GET
    expect:
      status: 200
      contains: "models"

  - name: "Multi-Step Workflow"
    type: chain
    steps:
      - id: step1
        service: ollama
        action: health_check
      - id: step2
        service: windmill
        action: health_check

  - name: "Custom Business Logic"
    type: custom
    script: custom-tests.sh
    function: test_business_workflow

# Success criteria
validation:
  success_rate: 80                # Allow 20% failure rate
  response_time: 15000            # Max 15s for responses
  revenue_potential: 25000        # Business value indicator
```

### Test Types

#### HTTP Tests
```yaml
- name: "API Endpoint Test"
  type: http
  service: my-service
  endpoint: /api/v1/health
  method: GET
  required: true                  # Fail if service unavailable
  fallback: mock                 # Use mock response if unavailable
  expect:
    status: 200
    contains: "healthy"
    not_contains: "error"
```

#### Chain Tests
```yaml
- name: "End-to-End Workflow"
  type: chain
  description: "Complete business workflow"
  steps:
    - id: authenticate
      service: auth-service
      action: login
      expect:
        status: success
    - id: process_data
      service: data-service
      action: process
      depends_on: authenticate
```

#### Custom Tests
```yaml
- name: "Business Logic Validation"
  type: custom
  script: custom-tests.sh          # Relative to scenario directory
  function: test_specific_logic    # Function to call
```

#### Database Tests
```yaml
- name: "Schema Validation"
  type: database
  service: postgres
  action: validate_schema
  schema_file: initialization/database/schema.sql
```

## 🔧 Extension Guide

### Adding New Test Types

1. **Create Handler Script**:
   ```bash
   # framework/handlers/my-test-type.sh
   execute_my_test_type_test() {
       local test_name="$1"
       local test_data="$2"
       
       # Implementation here
       return 0  # Success
       return 1  # Failure
       return 2  # Degraded (warning)
   }
   
   export -f execute_my_test_type_test
   ```

2. **Use in YAML**:
   ```yaml
   tests:
     - name: "My Custom Test"
       type: my-test-type
       # Additional configuration
   ```

### Adding New Resource Clients

1. **Create Client Script**:
   ```bash
   # framework/clients/my-resource.sh
   source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
   
   get_my_resource_url() {
       get_resource_url "my-resource"
   }
   
   test_my_resource_basic() {
       local url=$(get_my_resource_url)
       check_url_health "$url"
   }
   
   export -f get_my_resource_url
   export -f test_my_resource_basic
   ```

2. **Update common.sh** (if needed):
   ```bash
   # Add to get_resource_url() defaults
   my-resource)
       resource_url="http://localhost:8080"
       ;;
   ```

### Writing Custom Tests

1. **Create custom-tests.sh**:
   ```bash
   #!/bin/bash
   set -euo pipefail
   
   # Source framework utilities
   FRAMEWORK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../framework" && pwd)"
   source "$FRAMEWORK_DIR/handlers/custom.sh"
   source "$FRAMEWORK_DIR/clients/common.sh"
   
   test_my_business_logic() {
       print_custom_info "Testing specific business logic"
       
       # Use framework utilities
       local service_url=$(get_resource_url "my-service")
       if [[ -n "$service_url" ]]; then
           if check_url_health "$service_url"; then
               print_custom_success "Service is healthy"
           else
               print_custom_error "Service not available"
               return 1
           fi
       fi
       
       return 0
   }
   
   run_custom_tests() {
       test_my_business_logic
   }
   
   export -f run_custom_tests
   ```

2. **Reference in YAML**:
   ```yaml
   tests:
     - name: "Business Logic Test"
       type: custom
       script: custom-tests.sh
       function: test_my_business_logic
   ```

## 🎨 Best Practices

### Test Design
- **Keep tests atomic**: Each test should verify one specific aspect
- **Use meaningful names**: Test names should clearly describe what's being tested
- **Handle failures gracefully**: Use `required: false` for optional dependencies
- **Provide fallbacks**: Use `fallback: mock` for development environments

### Performance
- **Use appropriate timeouts**: Set realistic timeouts for different operations
- **Minimize test dependencies**: Avoid unnecessary service requirements
- **Optimize for CI**: Use `--quick` mode for faster CI runs

### Maintenance
- **Document custom tests**: Add clear comments in custom test scripts
- **Version your configurations**: Update version numbers when changing test formats
- **Monitor test health**: Review degraded tests regularly

## 🚀 Integration Examples

### CI/CD Integration

```yaml
# .github/workflows/test.yml
- name: Test Scenarios
  run: |
    pnpm test:scenarios:quick
  timeout-minutes: 15

- name: Test Specific Scenarios
  run: |
    pnpm test:scenarios --filter "multi-modal*"
```

### Development Workflow

```bash
# During development
./framework/scenario-test-runner.sh --scenario ./my-scenario --verbose

# Before committing
pnpm test:scenarios:quick

# Full validation
pnpm test:scenarios
```

## 🔍 Troubleshooting

### Common Issues

1. **Service Not Found**:
   ```
   Error: Required service URL not found: my-service
   ```
   - Check `.vrooli/service.json` configuration
   - Verify environment variables (`MY_SERVICE_URL`)
   - Ensure service is running on expected port

2. **Test Timeout**:
   ```
   Error: Test timed out after 30s
   ```
   - Increase timeout in scenario-test.yaml
   - Check service performance
   - Use `--quick` mode for faster tests

3. **YAML Parsing Error**:
   ```
   Error: Invalid YAML in scenario-test.yaml
   ```
   - Validate YAML syntax with `yq` or online validator
   - Check indentation consistency
   - Ensure proper quoting of strings

### Debug Mode

```bash
# Enable debug output
export FRAMEWORK_DEBUG=true
./framework/scenario-test-runner.sh --scenario ./my-scenario --verbose
```

### Service Discovery Debug

```bash
# Check service URL resolution
source ./framework/clients/common.sh
get_resource_url "my-service"
```

## 📊 Performance Metrics

The framework tracks several performance indicators:

- **Test Execution Time**: Individual and total test duration
- **Success Rate**: Percentage of passing tests
- **Service Response Time**: Average response times for health checks
- **Resource Utilization**: Memory and CPU usage during tests

## 🤝 Contributing

### Adding New Features

1. **Design**: Follow the existing architectural patterns
2. **Implement**: Create modular, reusable components
3. **Test**: Add tests for new functionality
4. **Document**: Update this README with new features

### Code Style

- Use consistent error handling patterns
- Follow shell scripting best practices
- Add comprehensive comments for complex logic
- Export functions that should be available to other components

---

**This framework successfully eliminates 1000+ lines of boilerplate per scenario while providing robust, extensible testing capabilities. It demonstrates excellent architectural patterns that should be maintained and extended.**