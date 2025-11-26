# Resource Testing Strategy

This document outlines Vrooli's comprehensive three-layer validation system for resource management scripts and interfaces.

## 🏗️ Three-Layer Validation Architecture

```
┌─────────────────────────────────────────┐
│         Layer 1: Syntax Validation      │
│     Fast, static analysis (~1 second)   │
├─────────────────────────────────────────┤
│        Layer 2: Behavioral Testing      │
│   Function execution, I/O verification  │
│             (~30 seconds)                │
├─────────────────────────────────────────┤
│      Layer 3: Integration Testing       │
│  Cross-resource, real scenarios (~5min) │
└─────────────────────────────────────────┘
```

### Layer 1: Syntax Validation 

**Purpose**: Ensure manage.sh scripts follow structural standards without execution

**Speed**: < 1 second per resource  
**Coverage**: 100% of interface requirements  
**Parallelizable**: Yes

**Validates**:
- ✅ Required actions exist (install, start, stop, status, logs)
- ✅ Argument parsing patterns are consistent  
- ✅ Error handling patterns present (set -euo pipefail, trap, exit codes)
- ✅ Help/usage patterns (--help, -h, --version)
- ✅ Configuration file structure (config/defaults.sh, config/messages.sh)
- ✅ Library structure (lib/common.sh, lib/install.sh, lib/status.sh)

**Implementation**:
```bash
# Syntax validation example
validate_syntax() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Check required actions exist
    local required_actions=("install" "start" "stop" "status" "logs")
    for action in "${required_actions[@]}"; do
        if ! grep -q "\"$action\")" "$manage_script"; then
            echo "FAIL: Missing required action: $action"
            return 1
        fi
    done
    
    # Check error handling patterns
    if ! grep -q "set -euo pipefail" "$manage_script"; then
        echo "WARN: Missing strict error handling"
    fi
    
    return 0
}
```

### Layer 2: Behavioral Testing

**Purpose**: Execute functions in controlled environment to verify behavior

**Speed**: < 30 seconds per resource  
**Coverage**: Core functionality paths  
**Environment**: Mock/sandbox environment

**Validates**:
- ✅ Actions execute without syntax errors
- ✅ Exit codes are correct (0=success, 1=error, 2=skip)
- ✅ Help text is well-formatted
- ✅ Invalid actions show usage and exit with error
- ✅ Required parameters are validated
- ✅ Configuration loading works correctly
- ✅ JSON output format (if supported)

**Implementation**:
```bash
# Behavioral testing example
validate_behavior() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Test help functionality
    local help_output
    if help_output=$("$manage_script" --help 2>&1); then
        if [[ "$help_output" =~ Usage:|Actions:|OPTIONS: ]]; then
            echo "PASS: Help output well-formatted"
        else
            echo "FAIL: Help output exists but poorly formatted"
            return 1
        fi
    else
        echo "FAIL: --help flag not supported"
        return 1
    fi
    
    # Test invalid action handling
    local invalid_output
    local invalid_exit_code
    invalid_output=$("$manage_script" --action invalid_action 2>&1) || invalid_exit_code=$?
    if [[ $invalid_exit_code -eq 1 ]]; then
        echo "PASS: Invalid action returns exit code 1"
    else
        echo "FAIL: Invalid action should return exit code 1, got $invalid_exit_code"
        return 1
    fi
    
    return 0
}
```

### Layer 3: Integration Testing

**Purpose**: Real-world functionality in actual environment

**Speed**: < 5 minutes per resource  
**Coverage**: End-to-end scenarios  
**Environment**: Real services and dependencies

**Validates**:
- ✅ Service installation and startup
- ✅ Health endpoints respond correctly
- ✅ API functionality works as expected
- ✅ Resource-specific operations
- ✅ Cross-resource integrations
- ✅ Performance under load
- ✅ Error recovery scenarios

**Implementation**:
```bash
# Integration testing example
validate_integration() {
    local resource_name="$1"
    local resource_path="$2"
    local manage_script="$resource_path/manage.sh"
    
    echo "Starting integration test for $resource_name..."
    
    # Install the resource
    if ! "$manage_script" --action install --yes; then
        echo "FAIL: Installation failed"
        return 1
    fi
    
    # Start the service
    if ! "$manage_script" --action start; then
        echo "FAIL: Service startup failed"
        return 1
    fi
    
    # Wait for health check
    local max_attempts=30
    for ((i=1; i<=max_attempts; i++)); do
        if "$manage_script" --action status >/dev/null 2>&1; then
            echo "PASS: Service healthy after ${i}s"
            break
        fi
        if [[ $i -eq $max_attempts ]]; then
            echo "FAIL: Service not healthy after ${max_attempts}s"
            return 1
        fi
        sleep 1
    done
    
    # Test resource-specific functionality
    test_resource_functionality "$resource_name" "$manage_script"
    
    return 0
}
```

## 🎯 Validation Levels

### Quick Validation (`--level quick`)
- **Duration**: < 10 seconds total
- **Coverage**: Layer 1 only (syntax validation)
- **Use Case**: Development feedback, pre-commit hooks
- **Command**: `./validate-interfaces.sh --level quick`

### Standard Validation (`--level standard`)
- **Duration**: < 2 minutes total  
- **Coverage**: Layers 1-2 (syntax + behavior)
- **Use Case**: CI/CD pipelines, pull request validation
- **Command**: `./validate-interfaces.sh --level standard`

### Full Validation (`--level full`)
- **Duration**: < 30 minutes total
- **Coverage**: All 3 layers (syntax + behavior + integration)
- **Use Case**: Release testing, comprehensive validation
- **Command**: `./validate-interfaces.sh --level full`

## 🔧 Interface Contracts

Resources must conform to versioned interface contracts that define expected behavior.

### Core Contract (v1.0)

```yaml
# resources/contracts/v1.0/core.yaml
version: "1.0"
required_actions:
  install:
    parameters:
      - name: force
        type: flag  
        default: false
        description: "Skip confirmation prompts"
    exit_codes:
      0: "Successfully installed"
      1: "Installation failed"
      2: "Already installed"
    
  start:
    parameters:
      - name: wait
        type: flag
        default: false
        description: "Wait for service to be ready"
    exit_codes:
      0: "Service started"
      1: "Start failed"
      2: "Already running"
    
  stop:
    parameters:
      - name: force
        type: flag
        default: false
        description: "Force stop without graceful shutdown"
    exit_codes:
      0: "Service stopped"
      1: "Stop failed"
      2: "Not running"
    
  status:
    parameters: []
    exit_codes:
      0: "Service running and healthy"
      1: "Service error or unhealthy"
      2: "Service not running"
    output_format:
      - text
      - json (optional)
    
  logs:
    parameters:
      - name: tail
        type: integer
        default: 50
        description: "Number of log lines to show"
    exit_codes:
      0: "Logs displayed"
      1: "Error retrieving logs"

help_patterns:
  - "--help"
  - "-h"
  - "--version"

error_handling:
  - "set -euo pipefail"
  - "trap cleanup EXIT"
  - "Meaningful error messages"
  - "Consistent exit codes"
```

### Category-Specific Contracts

Each category extends the core contract with specialized requirements:

```yaml
# resources/contracts/v1.0/ai.yaml
extends: core.yaml
additional_actions:
  models:
    description: "List available AI models"
    parameters:
      - name: format
        type: string
        values: ["text", "json"]
        default: "text"
  
  generate:
    description: "Generate content using AI model"
    parameters:
      - name: text
        type: string
        required: true
      - name: model
        type: string
        required: false
```

## 🚀 Running Validations

### Command-Line Interface

```bash
# Validate all resources (standard level)
./validate-interfaces.sh

# Validate specific resource
./validate-interfaces.sh --resource ollama

# Different validation levels
./validate-interfaces.sh --level quick     # Syntax only
./validate-interfaces.sh --level standard  # Syntax + behavior  
./validate-interfaces.sh --level full      # All layers

# Output formats
./validate-interfaces.sh --format json     # Machine-readable
./validate-interfaces.sh --format junit    # CI/CD integration
./validate-interfaces.sh --format html     # Human-readable report

# Generate detailed report
./validate-interfaces.sh --report --verbose
```

### Integration with Testing Framework

```bash
# Run through the main test runner
cd resources/tests
./run.sh --interface-validation --level standard

# Include in full resource testing
pnpm test:resources  # Includes interface validation

# Skip interface validation if needed
./run.sh --skip-interface
```

### CI/CD Integration

```yaml
# .github/workflows/resource-validation.yml
name: Resource Interface Validation

on:
  pull_request:
    paths:
      - 'resources/**/manage.sh'
      - 'resources/contracts/**'

jobs:
  validate-interfaces:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Quick Interface Validation
        run: |
          cd resources
          ./validate-interfaces.sh --level quick --format junit
      
      - name: Standard Interface Validation  
        run: |
          cd resources
          ./validate-interfaces.sh --level standard --format junit
        if: github.event_name == 'pull_request'
      
      - name: Full Integration Testing
        run: |
          cd resources  
          ./validate-interfaces.sh --level full --format junit
        if: github.ref == 'refs/heads/master'
```

## 🔄 Migration Strategy

### Automated Compliance Fixes

```bash
# Auto-fix common issues
./tools/fix-interface-compliance.sh --resource ollama --dry-run

# Apply fixes
./tools/fix-interface-compliance.sh --resource ollama --apply

# Batch fix all resources
./tools/batch-fix-interfaces.sh --auto-only
```

### Compatibility Wrappers

For resources that can't be immediately updated:

```bash
# Generate compatibility wrapper
./tools/generate-compatibility-wrapper.sh --resource <resource> --version <version>

# This creates: <resource>/lib/compat-v<version>.sh
```

### Migration Checklist

1. **Assessment**: Run validation to identify issues
2. **Auto-fix**: Apply automated fixes where possible  
3. **Manual fixes**: Address remaining issues
4. **Testing**: Validate fixes don't break functionality
5. **Documentation**: Update resource-specific docs
6. **Backward compatibility**: Add wrappers if needed

## 📊 Validation Reports

### Text Report
```
Resource Interface Validation Report
Generated: 2024-01-15 14:30:00
========================================

Summary:
  Total resources: 23
  Passed: 20 (87%)
  Failed: 3 (13%)
  
Failed resources:
  • huginn (missing logs action)
  • vault (non-standard exit codes)  
  • questdb (missing help patterns)

Detailed Results:
[+] ollama - All validations passed
[!] huginn - Layer 1 failed: Missing 'logs' action
...
```

### JSON Report
```json
{
  "summary": {
    "total": 23,
    "passed": 20,
    "failed": 3,
    "timestamp": "2024-01-15T14:30:00Z"
  },
  "results": [
    {
      "resource": "ollama",
      "status": "passed",
      "layers": {
        "syntax": "passed",
        "behavior": "passed", 
        "integration": "passed"
      },
      "details": "All validations passed"
    }
  ]
}
```

## 🛠️ Development Workflow

### Adding New Resources

1. **Create structure**: Use resource template
2. **Implement interface**: Follow contract specifications
3. **Validate early**: Run quick validation during development
4. **Test thoroughly**: Run full validation before submission
5. **Document**: Add resource-specific documentation

### Modifying Existing Resources

1. **Check compatibility**: Validate current state
2. **Plan changes**: Consider interface impacts
3. **Implement**: Make changes following contracts
4. **Validate**: Ensure compliance maintained
5. **Test integration**: Verify with dependent resources

### Pre-commit Validation

```bash
#!/bin/bash
# .git/hooks/pre-commit
changed_resources=$(git diff --cached --name-only | grep 'manage\.sh$')

if [[ -n "$changed_resources" ]]; then
    echo "Validating changed resource interfaces..."
    
    for resource_script in $changed_resources; do
        resource_dir=$(dirname "$resource_script")
        resource_name=$(basename "$resource_dir")
        
        if ! resources/validate-interfaces.sh --resource "$resource_name" --level quick; then
            echo "❌ Interface validation failed for $resource_name"
            echo "Run: resources/validate-interfaces.sh --resource $resource_name --verbose"
            exit 1
        fi
    done
    
    echo "✅ All resource interfaces valid"
fi
```

## 📋 Best Practices

### Interface Design
- **Consistency**: Follow established patterns
- **Clarity**: Clear action names and parameters
- **Flexibility**: Support common use cases
- **Backward compatibility**: Version interface changes

### Testing Strategy
- **Layer appropriately**: Use quick validation during development
- **Automate validation**: Include in CI/CD pipelines  
- **Test realistic scenarios**: Use integration layer for real-world testing
- **Document expected behavior**: Keep contracts up-to-date

### Performance Optimization
- **Parallel execution**: Run validations concurrently when possible
- **Smart caching**: Cache validation results for unchanged resources
- **Timeout management**: Set appropriate limits for each layer
- **Resource isolation**: Prevent test interference

This three-layer validation system ensures consistent, reliable resource interfaces while supporting rapid development and maintaining high quality standards across all Vrooli resources.
