# Resource Interface Standards

This document defines the standardized interface that all Vrooli resource `manage.sh` scripts must implement.

## 🔧 Quick Reference

### Required Actions
```bash
./manage.sh --action install    # Install the resource
./manage.sh --action start      # Start the service
./manage.sh --action stop       # Stop the service  
./manage.sh --action status     # Check service status
./manage.sh --action logs       # View service logs
```

### Required Help Patterns
```bash
./manage.sh --help             # Show usage information
./manage.sh -h                 # Show usage information
./manage.sh --version          # Show version information
```

### Required Error Handling
- **Invalid actions**: Exit non-zero with error message
- **No arguments**: Exit non-zero and show usage
- **Missing dependencies**: Clear error messages

## 🧪 Validation

Vrooli uses a **three-layer validation system** for comprehensive testing:

### Layer 1: Syntax Validation (< 1 second)
```bash
# Quick syntax checks without execution
./validate-interfaces.sh --level quick --resource <resource-name>
```

### Layer 2: Behavioral Testing (< 30 seconds)
```bash
# Function execution in controlled environment
./validate-interfaces.sh --level standard --resource <resource-name>
```

### Layer 3: Integration Testing (< 5 minutes)
```bash
# Real-world functionality testing
./validate-interfaces.sh --level full --resource <resource-name>
```

### Complete Testing Strategy
For detailed information on the testing architecture, see **[Testing Strategy](TESTING_STRATEGY.md)**

### Integration Test Runner
```bash
# Interface compliance runs automatically as Phase 2b
./run.sh --resource <resource-name>

# Skip interface compliance if needed
./run.sh --skip-interface
```

## 📋 Common Issues

| Issue | Fix |
|-------|-----|
| Missing `logs` action | Add `logs` case to action switch statement |
| No-args runs install | Add argument count check at script start |
| Missing help patterns | Implement `--help`, `-h`, `--version` handling |

## 📖 Full Documentation

- **Testing Strategy**: [TESTING_STRATEGY.md](TESTING_STRATEGY.md) - Complete three-layer validation system
- **Complete Guide**: [tests/INTERFACE_COMPLIANCE_README.md](tests/INTERFACE_COMPLIANCE_README.md)
- **Integration**: [tests/INTERFACE_COMPLIANCE_INTEGRATION.md](tests/INTERFACE_COMPLIANCE_INTEGRATION.md)
- **Framework**: [tests/framework/interface-compliance.sh](tests/framework/interface-compliance.sh)
- **Validation**: [validate-all-interfaces.sh](validate-all-interfaces.sh)

## 🎯 Benefits

- **Consistent UX**: All resources behave the same way
- **Early Detection**: Three-layer validation catches issues at the right level
- **Quality Assurance**: Automated compliance validation from syntax to integration
- **Development Speed**: Quick feedback with layered testing (1s → 30s → 5min)
- **Documentation**: Tests serve as executable specs

The interface compliance system ensures reliability and consistency across all Vrooli resources while maintaining their unique functionality.