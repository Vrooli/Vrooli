# Resource Interface Compliance System

This document provides a comprehensive overview of the Vrooli resource interface compliance system, which ensures all resources follow consistent interface standards.

## 🎯 Overview

The Resource Interface Compliance System validates that all Vrooli resource `manage.sh` scripts implement a standardized interface. This system transforms the previous "honor-based" approach into a **verified contract system** while maintaining backward compatibility.

### Benefits

- **Consistent User Experience**: All resources behave the same way
- **Early Issue Detection**: Interface problems caught before functional testing
- **Prevented Drift**: Automatic detection of non-compliant resources
- **Quality Assurance**: Higher confidence in resource reliability
- **Documentation**: Tests serve as executable interface documentation

## 📋 Interface Standards

### Required Actions

All resource `manage.sh` scripts **must** implement these actions:

| Action | Purpose | Example |
|--------|---------|---------|
| `install` | Install the resource | `./manage.sh --action install --yes yes` |
| `start` | Start the service | `./manage.sh --action start` |
| `stop` | Stop the service | `./manage.sh --action stop` |
| `status` | Check service status | `./manage.sh --action status` |
| `logs` | View service logs | `./manage.sh --action logs` |

### Optional Actions

These actions are **recommended** but not required:

| Action | Purpose | Example |
|--------|---------|---------|
| `uninstall` | Remove the resource | `./manage.sh --action uninstall --yes yes` |
| `restart` | Restart the service | `./manage.sh --action restart` |

### Required Argument Patterns

| Argument | Purpose | Behavior |
|----------|---------|----------|
| `--help`, `-h` | Show usage information | Must exit with code 0 and display usage |
| `--version` | Show version information | Should display resource name or version |
| `--action <action>` | Specify action to perform | Primary way to invoke actions |
| `--yes` | Non-interactive mode | Recommended for automation |

### Required Error Handling

| Scenario | Expected Behavior | Exit Code |
|----------|------------------|-----------|
| Invalid action | Show error + usage information | Non-zero |
| No arguments | Show usage or require action | Non-zero |
| Missing dependencies | Clear error message | Non-zero |
| Syntax errors | Proper bash error handling | Non-zero |

## 🛠️ Tools and Usage

### 1. Integration Test Runner

The main test runner now includes **mandatory interface compliance validation**:

```bash
# Run all tests (includes interface compliance validation)
./run.sh

# Test specific resource
./run.sh --resource ollama

# Run only interface compliance tests
./run.sh --interface-only

# Skip interface compliance validation
./run.sh --skip-interface

# Verbose output for detailed compliance issues
./run.sh --interface-only --verbose
```

### 2. Standalone Validation Script

For direct validation of all resources:

```bash
# Validate all resources
../validate-all-interfaces.sh

# Validate specific resource
../validate-all-interfaces.sh --resource ollama

# Detailed output
../validate-all-interfaces.sh --verbose

# JSON output
../validate-all-interfaces.sh --format json

# Generate report
../validate-all-interfaces.sh --report --format csv
```

### 3. Individual Resource Tests

Each resource test now includes interface compliance as Phase 1:

```bash
# Run individual resource test (includes interface compliance)
./single/ai/ollama.test.sh
```

## 📊 Test Execution Flow

### Standard Test Run

1. **Phase 1**: Resource Discovery
2. **Phase 2**: Test File Validation (optional with `--validate-tests`)
3. **Phase 2b**: Interface Compliance Validation (mandatory, skip with `--skip-interface`)
4. **Phase 3**: Single Resource Tests
5. **Phase 4**: Business Scenario Tests
6. **Phase 5**: Test Results

### Interface-Only Mode

When using `--interface-only`:

1. **Phase 1**: Resource Discovery
2. **Phase 3**: Interface Compliance Tests (detailed)

## 🔧 Integration Guide

### For New Resources

1. **Follow the Template**: Use the BATS template in `/tests/bats-fixtures/setup/resource_templates/`
2. **Implement Required Actions**: Ensure all required actions are implemented
3. **Test Early**: Run interface compliance tests during development
4. **Follow Patterns**: Look at existing compliant resources for examples

### For Existing Resources

1. **Run Validation**: Use `../validate-all-interfaces.sh --resource <name>` to check compliance
2. **Fix Issues**: Address any interface compliance failures
3. **Update Tests**: Integrate interface compliance into resource tests
4. **Verify**: Run tests to ensure both interface and functional compliance

### Adding Interface Compliance to Resource Tests

See [INTERFACE_COMPLIANCE_INTEGRATION.md](INTERFACE_COMPLIANCE_INTEGRATION.md) for detailed instructions on integrating interface compliance tests into existing resource tests.

## 📝 Compliance Test Details

The interface compliance framework tests these categories:

### 1. Help and Usage Compliance

- `--help` displays proper usage information
- `-h` displays usage information  
- `--version` displays version/resource information

### 2. Required Actions Compliance

- All required actions are implemented and accessible
- Actions handle dry-run mode appropriately
- Optional actions provide warnings when missing

### 3. Error Handling Compliance

- Invalid actions show helpful error messages
- No arguments case shows usage information
- Proper exit codes for different scenarios

### 4. Configuration Compliance

- Script loads and executes without syntax errors
- Configuration is handled gracefully
- Dependencies are checked appropriately

### 5. Argument Patterns Compliance

- `--yes` flag is supported for non-interactive mode
- Standard argument parsing patterns are followed

## 🚨 Common Issues and Solutions

### Missing 'logs' Action

**Issue**: `Action 'logs' is not implemented (required)`

**Solution**: Add logs action to your manage.sh script:

```bash
case "$ACTION" in
    # ... other actions ...
    "logs")
        if [[ "$CONTAINER_RUNTIME" == "docker" ]]; then
            docker logs "$CONTAINER_NAME" "${EXTRA_ARGS[@]}"
        else
            echo "Logs not available for non-containerized installation"
        fi
        ;;
esac
```

### Incorrect No-Arguments Handling

**Issue**: `No arguments handling is not compliant`

**Solution**: Show usage when called with no arguments:

```bash
# At the beginning of your script
if [[ $# -eq 0 ]]; then
    show_usage
    exit 1
fi
```

### Missing Help Patterns

**Issue**: `--help displays proper usage information` fails

**Solution**: Implement proper help handling:

```bash
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OPTIONS:
    --action <action>    Action to perform (install|start|stop|status|logs)
    --help, -h          Show this help message
    --version           Show version information
    --yes               Non-interactive mode

EXAMPLES:
    $0 --action install --yes yes
    $0 --action status
EOF
}

case "$1" in
    --help|-h)
        show_usage
        exit 0
        ;;
    --version)
        echo "$(basename "$(dirname "$0")") resource manager"
        exit 0
        ;;
esac
```

## 📈 Compliance Levels

### Level 1: Basic Compliance (Required)

- All required actions implemented
- Proper help/usage patterns
- Basic error handling

### Level 2: Enhanced Compliance (Recommended)

- Optional actions implemented
- Advanced error handling
- Non-interactive mode support
- Proper version information

### Level 3: Exemplary Compliance (Ideal)

- Comprehensive error messages
- Detailed usage information
- Advanced argument patterns
- Excellent user experience

## 🔄 Development Workflow

### For Resource Developers

1. **Develop**: Create resource following interface standards
2. **Test**: Run `../validate-all-interfaces.sh --resource <name>`
3. **Fix**: Address any compliance issues
4. **Integrate**: Add interface compliance to resource tests
5. **Verify**: Run full test suite

### For Integration

1. **Mandatory Validation**: Interface compliance runs automatically
2. **Early Detection**: Issues caught before functional tests
3. **Clear Guidance**: Specific instructions for fixing issues
4. **Non-Blocking**: Tests continue even with compliance issues

## 📚 Additional Resources

- [Integration Guide](INTERFACE_COMPLIANCE_INTEGRATION.md) - How to add compliance to resource tests
- [Framework Source](framework/interface-compliance.sh) - Compliance testing framework
- [Validation Script](../validate-all-interfaces.sh) - Standalone validation tool
- [BATS Templates](bats-fixtures/setup/resource_templates/) - Test templates for new resources

## 🎯 Future Enhancements

### Planned Features

- **Automatic Fixes**: Script to automatically fix common compliance issues
- **Advanced Validation**: More sophisticated interface pattern detection
- **Performance Testing**: Interface response time validation
- **Documentation Generation**: Auto-generate documentation from interface compliance

### Integration Opportunities

- **CI/CD Integration**: Automated compliance checking in pipelines
- **Pre-commit Hooks**: Validate compliance before commits
- **Resource Generator**: Template-based resource creation with built-in compliance

This system ensures that all Vrooli resources maintain a consistent, reliable interface while preserving their unique functionality and flexibility.