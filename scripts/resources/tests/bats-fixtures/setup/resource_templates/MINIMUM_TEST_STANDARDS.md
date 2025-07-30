# Minimum Test Standards for Vrooli Resources

This document defines the mandatory testing requirements that every resource in the Vrooli ecosystem must meet to ensure quality, reliability, and maintainability.

## 📋 Overview

Every resource must implement a comprehensive test suite that covers all critical functionality. These standards ensure:
- **Consistent quality** across all resources
- **Reliable integration** with the Vrooli platform
- **Maintainable codebase** with clear testing patterns
- **Predictable behavior** in production environments

## 🎯 Mandatory Test Categories

### 1. **Entry Point Tests** (`manage.bats`)
**Purpose**: Validate command-line interface and argument parsing
**Required Coverage**:
- [ ] Help/usage display (`--help`, `-h`)
- [ ] Version information (`--version`)
- [ ] Argument validation (required parameters, invalid actions)
- [ ] All supported actions (install, status, start, stop, logs, etc.)
- [ ] Flag combinations (--force, --yes, --dry-run)
- [ ] Error handling for missing dependencies
- [ ] Configuration loading validation

**Minimum Test Count**: 10+ tests

### 2. **Configuration Tests** (`config/defaults.bats`)
**Purpose**: Ensure robust configuration management
**Required Coverage**:
- [ ] Configuration initialization (`RESOURCE::export_config`)
- [ ] Default value validation
- [ ] Environment variable override handling
- [ ] Configuration validation functions
- [ ] Invalid configuration rejection (ports, URLs, paths)
- [ ] Configuration display (with sensitive data masking)
- [ ] Configuration file loading and saving
- [ ] Dependency checks
- [ ] Edge cases (empty values, special characters)

**Minimum Test Count**: 15+ tests

### 3. **Message Localization Tests** (`config/messages.bats`)
**Purpose**: Validate message system and internationalization
**Required Coverage**:
- [ ] Message system initialization
- [ ] Message retrieval for valid keys
- [ ] Missing message handling (graceful fallback)
- [ ] Parameter substitution in messages
- [ ] Language switching functionality
- [ ] Message file loading
- [ ] Standard message key availability
- [ ] Error message validation

**Minimum Test Count**: 8+ tests

### 4. **Common Utilities Tests** (`lib/common.bats`)
**Purpose**: Test shared utility functions
**Required Coverage**:
- [ ] Service state detection (`is_running`, `get_container_id`)
- [ ] Health check functions with retry logic
- [ ] Configuration getters/setters
- [ ] Directory management (creation, cleanup, permissions)
- [ ] Validation helpers (ports, URLs, formats)
- [ ] Logging functions (info, error, debug)
- [ ] Error handling functions
- [ ] Performance/timing validation

**Minimum Test Count**: 20+ tests

### 5. **Status Monitoring Tests** (`lib/status.bats`)
**Purpose**: Ensure comprehensive health monitoring
**Required Coverage**:
- [ ] Basic status reporting (healthy/unhealthy states)
- [ ] Detailed status information
- [ ] Health check validation
- [ ] Readiness checking with timeouts
- [ ] Performance monitoring
- [ ] Connectivity validation
- [ ] Port accessibility checks
- [ ] Multiple output formats (JSON, brief, verbose)
- [ ] Error scenarios (Docker down, network issues)
- [ ] Status caching (if implemented)

**Minimum Test Count**: 15+ tests

### 6. **Installation Tests** (`lib/install.bats`)
**Purpose**: Validate installation and lifecycle management
**Required Coverage**:
- [ ] Prerequisites checking (Docker, disk space, port availability)
- [ ] Full installation process
- [ ] User confirmation handling
- [ ] Dry-run mode validation
- [ ] Force reinstallation
- [ ] Docker image management (pull, verify)
- [ ] Container creation with correct parameters
- [ ] Configuration setup
- [ ] Data directory initialization
- [ ] Post-installation validation
- [ ] Uninstallation process
- [ ] Upgrade procedures

**Minimum Test Count**: 18+ tests

## 🔧 Optional But Recommended Tests

### 7. **API Integration Tests** (`lib/api.bats`)
**For resources with HTTP APIs**
**Recommended Coverage**:
- [ ] Basic API connectivity
- [ ] HTTP method support (GET, POST, PUT, DELETE)
- [ ] Authentication handling
- [ ] Error response handling (4xx, 5xx)
- [ ] JSON parsing and validation
- [ ] Rate limiting handling
- [ ] Request retry logic
- [ ] File upload/download (if applicable)
- [ ] Streaming responses (if applicable)

**Minimum Test Count**: 12+ tests

### 8. **Docker Management Tests** (`lib/docker.bats`)
**For containerized resources**
**Recommended Coverage**:
- [ ] Container lifecycle (create, start, stop, remove)
- [ ] Image management
- [ ] Volume mounting
- [ ] Network configuration
- [ ] Environment variable passing
- [ ] Resource constraints
- [ ] Log access

**Minimum Test Count**: 10+ tests

## 📊 Test Quality Requirements

### **Mock Usage Standards**
- **Mandatory**: Use shared mock framework from `common_setup.bash`
- **Isolation**: Each test must be completely isolated
- **Realistic**: Mocks must simulate real system behavior
- **State Management**: Proper setup/teardown with cleanup

### **Test Organization Standards**
- **Naming**: Clear, descriptive test names following pattern: `resource::function_name should describe expected behavior`
- **AAA Pattern**: Arrange, Act, Assert structure
- **Coverage**: Every exported function must have tests
- **Error Scenarios**: Both positive and negative test cases required

### **Performance Standards**
- **Execution Time**: Test suite must complete within 2 minutes for basic resources
- **Resource Usage**: Tests must clean up all created resources
- **Parallel Safety**: Tests must be safe for parallel execution

## 🚨 Critical Requirements

### **Error Handling**
Every resource must test:
- [ ] Missing dependencies (Docker, curl, jq)
- [ ] Network failures
- [ ] Permission errors
- [ ] Invalid configurations
- [ ] Resource conflicts
- [ ] Timeout scenarios

### **Security Validation**
Every resource must test:
- [ ] Sensitive data masking in output
- [ ] Configuration file permissions
- [ ] API authentication (if applicable)
- [ ] Input validation and sanitization

### **Integration Points**
Every resource must test:
- [ ] Interaction with Docker daemon
- [ ] Network connectivity requirements
- [ ] File system access patterns
- [ ] External service dependencies

## 📈 Coverage Metrics

### **Minimum Coverage Targets**
- **Function Coverage**: 100% of exported functions
- **Branch Coverage**: 80% of conditional branches
- **Error Path Coverage**: 100% of error handling paths
- **Configuration Coverage**: 100% of configuration parameters

### **Quality Gates**
- [ ] All tests pass consistently
- [ ] No intermittent test failures
- [ ] Test execution time under limits
- [ ] Proper cleanup verification
- [ ] Mock framework compliance

## 🔍 Validation Checklist

Before considering a resource's test suite complete, verify:

### **Test Structure**
- [ ] All mandatory test files exist
- [ ] Test files follow naming conventions
- [ ] Proper BATS header and version requirements
- [ ] Shared infrastructure usage (common_setup.bash)

### **Test Implementation**
- [ ] Minimum test counts met for each category
- [ ] Comprehensive error scenario coverage
- [ ] Realistic mock usage throughout
- [ ] Proper assertion patterns

### **Test Execution**
- [ ] All tests pass independently
- [ ] Tests pass in parallel execution
- [ ] No resource leaks after test completion
- [ ] Consistent results across runs

### **Documentation**
- [ ] Test files have clear descriptions
- [ ] Complex test logic is commented
- [ ] Resource-specific test requirements documented

## 🎯 Implementation Priority

### **Phase 1: Core Requirements (Immediate)**
1. `manage.bats` - Entry point validation
2. `config/defaults.bats` - Configuration management
3. `lib/common.bats` - Utility functions
4. `lib/status.bats` - Health monitoring

### **Phase 2: Lifecycle Management (Week 2)**
1. `lib/install.bats` - Installation procedures
2. `config/messages.bats` - Message localization

### **Phase 3: Specialized Testing (Week 3)**
1. `lib/api.bats` - API integration (if applicable)
2. `lib/docker.bats` - Container management (if applicable)
3. Resource-specific specialized tests

## 📚 Resources and Support

### **Templates Available**
- All templates located in: `/scripts/resources/tests/bats-fixtures/setup/resource_templates/`
- Copy and customize templates for each new resource
- Follow template comments for implementation guidance

### **Testing Framework**
- **BATS**: Primary testing framework
- **Mock System**: Comprehensive mocking for isolation
- **Shared Infrastructure**: Common utilities and helpers
- **CI Integration**: JSON output support for automation

### **Example Resources**
Study these exemplary implementations:
- **browserless**: Complete coverage with excellent patterns
- **unstructured-io**: Comprehensive configuration testing
- **ollama**: Complex workflow and data structure testing
- **qdrant**: Recent standardization example

## ⚠️ Common Pitfalls to Avoid

- **Incomplete Mock Setup**: Always use `setup_standard_mocks()`
- **Shared State**: Tests must not depend on each other
- **Hardcoded Values**: Use configuration variables
- **Missing Cleanup**: Always implement proper teardown
- **Weak Assertions**: Use specific, meaningful assertions
- **Skip Error Cases**: Test failure scenarios thoroughly

## 🎉 Success Criteria

A resource test suite meets standards when:
1. **All mandatory tests implemented** and passing
2. **Quality metrics achieved** (coverage, performance)
3. **Integration verified** with Vrooli platform
4. **Documentation complete** and clear
5. **Maintenance ready** for long-term updates

This ensures every resource contributes to a robust, reliable, and maintainable Vrooli ecosystem.