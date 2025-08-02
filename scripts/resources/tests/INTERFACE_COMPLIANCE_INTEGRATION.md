# Interface Compliance Integration Guide

This document shows how to integrate interface compliance tests into existing resource tests.

## Overview

Interface compliance tests should be added as **Phase 1** of every resource test, before functional tests run. This ensures that resources meet the standard interface requirements before testing their specific functionality.

## Integration Steps

### 1. Source the Interface Compliance Framework

Add this import to your test file after the other framework imports:

```bash
# Source framework helpers
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/interface-compliance.sh"  # Add this line
```

### 2. Add Interface Compliance Test Function

Add this function to your test file (customize for your resource):

```bash
# Test [RESOURCE] interface compliance
test_[resource]_interface_compliance() {
    echo "🔧 Testing [RESOURCE] interface compliance..."
    
    # Find the manage.sh script for [resource]
    local manage_script=""
    local potential_paths=(
        "$SCRIPT_DIR/../[category]/[resource]/manage.sh"
        "$SCRIPT_DIR/../../[category]/[resource]/manage.sh"
        "$(cd "$SCRIPT_DIR/../.." && pwd)/[category]/[resource]/manage.sh"
    )
    
    for path in "${potential_paths[@]}"; do
        if [[ -f "$path" ]]; then
            manage_script="$path"
            break
        fi
    done
    
    if [[ -z "$manage_script" ]]; then
        # Try to find it dynamically
        manage_script=$(find "$(cd "$SCRIPT_DIR/../.." && pwd)" -name "manage.sh" -path "*[resource]*" -type f 2>/dev/null | head -1)
    fi
    
    if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
        echo "⚠️  Could not find [resource] manage.sh script - skipping interface compliance test"
        return 0
    fi
    
    echo "📍 Using manage script: $manage_script"
    
    # Run interface compliance test
    if test_resource_interface_compliance "$TEST_RESOURCE" "$manage_script"; then
        echo "✅ [RESOURCE] interface compliance test passed"
        return 0
    else
        echo "❌ [RESOURCE] interface compliance test failed"
        echo "💡 The manage.sh script should be updated to pass interface compliance:"
        echo "   • Ensure all required actions are implemented: install, start, stop, status, logs"
        echo "   • Handle no arguments by showing usage (not running install)"
        echo "   • Provide proper help/usage with --help and -h flags"
        return 1
    fi
}
```

### 3. Update Main Test Function

Modify your main test function to include interface compliance as Phase 1:

```bash
main() {
    echo "🧪 Starting [RESOURCE] Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    echo "📋 Running [RESOURCE] test suite..."
    echo
    
    # Phase 1: Interface Compliance (should be first)
    echo "Phase 1: Interface Compliance"
    test_[resource]_interface_compliance
    echo
    
    # Phase 2: Functional Tests
    echo "Phase 2: Functional Tests"
    test_[resource]_health
    test_[resource]_functionality
    # ... other functional tests
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ [RESOURCE] integration test failed"
        exit 1
    else
        echo "✅ [RESOURCE] integration test passed"
        exit 0
    fi
}
```

## Example: Ollama Integration

See `/scripts/resources/tests/single/ai/ollama.test.sh` for a complete example of interface compliance integration.

## Required Actions

All resources must implement these actions in their `manage.sh` script:

### Required Actions:
- `install` - Install the resource
- `start` - Start the service  
- `stop` - Stop the service
- `status` - Check service status
- `logs` - View service logs

### Optional Actions:
- `uninstall` - Remove the resource
- `restart` - Restart the service

### Required Argument Patterns:
- `--help`, `-h` - Display usage information
- `--version` - Display version information
- `--action <action>` - Specify action to perform
- `--yes` - Non-interactive mode (recommended)

### Required Error Handling:
- Exit with non-zero code for invalid actions
- Show usage when called with no arguments
- Handle missing dependencies gracefully

## Benefits

1. **Early Detection**: Interface issues are caught before functional tests
2. **Consistency**: All resources follow the same interface patterns
3. **Documentation**: Tests serve as executable interface documentation
4. **Quality Assurance**: Prevents interface drift over time

## Running Interface-Only Tests

You can test interface compliance separately using:

```bash
# Test all resources
./run.sh --interface-only

# Test specific resource
./run.sh --interface-only --resource ollama

# Use standalone validation script
../validate-all-interfaces.sh --resource ollama --verbose
```

This integration ensures that all Vrooli resources maintain a consistent, reliable interface while preserving their unique functionality.