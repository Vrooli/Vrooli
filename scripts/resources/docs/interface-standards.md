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

### Quick Validation
```bash
# Test specific resource
cd scripts/resources/tests
./run.sh --interface-only --resource <resource-name>

# Test all resources  
../validate-all-interfaces.sh
```

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

- **Complete Guide**: [tests/INTERFACE_COMPLIANCE_README.md](tests/INTERFACE_COMPLIANCE_README.md)
- **Integration**: [tests/INTERFACE_COMPLIANCE_INTEGRATION.md](tests/INTERFACE_COMPLIANCE_INTEGRATION.md)
- **Framework**: [tests/framework/interface-compliance.sh](tests/framework/interface-compliance.sh)
- **Validation**: [validate-all-interfaces.sh](validate-all-interfaces.sh)

## 🎯 Benefits

- **Consistent UX**: All resources behave the same way
- **Early Detection**: Issues caught during development  
- **Quality Assurance**: Automated compliance validation
- **Documentation**: Tests serve as executable specs

The interface compliance system ensures reliability and consistency across all Vrooli resources while maintaining their unique functionality.