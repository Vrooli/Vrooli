# Resource Test Templates

This directory contains standardized test templates for Vrooli resource testing. These templates ensure consistency, quality, and maintainability across all resource tests.

## 📋 Minimum Test Standards

Every resource must implement the following test files:

### **Required Test Files**
1. **`manage.bats`** - Entry point and argument parsing tests
2. **`config/defaults.bats`** - Configuration validation tests  
3. **`config/messages.bats`** - Message localization tests
4. **`lib/common.bats`** - Shared utility function tests
5. **`lib/install.bats`** - Installation procedure tests
6. **`lib/status.bats`** - Health monitoring tests

### **Recommended Test Files**
- **`lib/api.bats`** - API endpoint tests (for services with APIs)
- **`lib/docker.bats`** - Container management tests (for containerized resources)
- **`lib/[specialized].bats`** - Resource-specific functionality tests

## 🏗️ Test Structure Template

```
resource-name/
├── manage.bats                 # Entry point testing
├── config/
│   ├── defaults.bats          # Configuration validation
│   └── messages.bats          # Message localization
└── lib/
    ├── common.bats            # Shared utilities
    ├── install.bats           # Installation procedures
    ├── status.bats            # Health monitoring
    ├── api.bats               # API functionality (if applicable)
    ├── docker.bats            # Container management (if applicable)
    └── [specialized].bats     # Resource-specific tests
```

## 🧪 Test Quality Standards

### **Test Coverage Requirements**
- **Function Coverage**: Every exported function must have tests
- **Error Scenarios**: Test failure modes and edge cases
- **Configuration**: Validate all configuration parameters
- **Integration**: Test interactions with dependencies
- **Performance**: Basic performance validation for critical paths

### **Test Organization**
- **One test file per module**: Mirror the source code structure
- **Descriptive test names**: `resource::function_name describes what it tests`
- **Proper isolation**: Each test is independent with setup/teardown
- **Mock usage**: Use shared mock framework for external dependencies

### **Best Practices**
- Use shared test infrastructure (`common_setup.bash`)
- Follow AAA pattern: Arrange, Act, Assert
- Test behavior, not implementation details
- Include both positive and negative test cases
- Maintain clear, readable assertions

## 📁 Template Files

### Available Templates
- **`template-manage.bats`** - Entry point test template
- **`template-config-defaults.bats`** - Configuration validation template
- **`template-config-messages.bats`** - Message localization template
- **`template-lib-common.bats`** - Common utilities template
- **`template-lib-install.bats`** - Installation procedures template
- **`template-lib-status.bats`** - Health monitoring template
- **`template-lib-api.bats`** - API testing template
- **`template-lib-docker.bats`** - Container management template

### Template Usage
1. Copy the appropriate template file
2. Rename to match your resource and module
3. Replace `RESOURCE_NAME` placeholders with your resource name
4. Replace `MODULE_NAME` placeholders with your module name
5. Implement the specific test cases for your functionality
6. Verify all tests pass with `bats your-test-file.bats`

## 🔧 Testing Guidelines

### **Mock Framework Usage**
```bash
# Standard setup pattern
setup() {
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    setup_standard_mocks
    
    # Resource-specific mock configuration
    export RESOURCE_PORT="8080"
    export RESOURCE_CONTAINER_NAME="resource-test"
    
    # Load resource components
    source "${RESOURCE_DIR}/config/defaults.sh"
    source "${RESOURCE_DIR}/lib/module.sh"
}
```

### **Test Naming Convention**
```bash
@test "resource::function_name should describe expected behavior" {
    # Test implementation
}

@test "resource::function_name should handle error when invalid input" {
    # Error scenario testing
}
```

### **Assertion Patterns**
```bash
# Success case
run resource::function_name "valid_input"
[ "$status" -eq 0 ]
[[ "$output" =~ "expected pattern" ]]

# Error case  
run resource::function_name "invalid_input"
[ "$status" -ne 0 ]
[[ "$output" =~ "error message pattern" ]]
```

## 🚀 Getting Started

1. **Choose templates** based on your resource requirements
2. **Copy and customize** templates for your resource
3. **Implement test cases** following the patterns
4. **Run tests** to verify implementation
5. **Review coverage** to ensure all functions are tested

## 📖 Reference Examples

Study these exemplary resource tests for best practices:
- **browserless/lib/common.bats** - Excellent mock usage and function coverage
- **unstructured-io/config/defaults.bats** - Comprehensive configuration testing
- **ollama/lib/models.bats** - Complex workflow and data structure testing
- **qdrant/lib/common.bats** - Recent standardization with shared infrastructure

## 🔍 Quality Checklist

- [ ] All required test files implemented
- [ ] Every exported function has tests
- [ ] Error scenarios covered
- [ ] Configuration validation complete
- [ ] Mock framework used appropriately
- [ ] Tests are isolated and independent
- [ ] Clear, descriptive test names
- [ ] Documentation updated