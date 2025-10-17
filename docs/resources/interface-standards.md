# Resource Interface Standards v2.0

This document defines the v2.0 Universal CLI Contract that all Vrooli resources must implement.

## 🔧 Quick Reference

### Core Command Structure
```bash
resource-<name> help                    # Show comprehensive help
resource-<name> status                  # Show resource status
resource-<name> logs                    # View resource logs
```

### Management Commands
```bash
resource-<name> manage install          # Install the resource
resource-<name> manage start            # Start the service
resource-<name> manage stop             # Stop the service
resource-<name> manage restart          # Restart the service
resource-<name> manage uninstall        # Remove the resource
```

### Testing Commands
```bash
resource-<name> test smoke              # Quick health validation (30s max)
resource-<name> test integration        # End-to-end functionality (120s max)
resource-<name> test unit               # Library function validation (60s max)
resource-<name> test all                # Run all test suites
```

### Content Management
```bash
resource-<name> content add             # Add content to resource
resource-<name> content list            # List available content
resource-<name> content get             # Retrieve specific content
resource-<name> content remove          # Remove content
resource-<name> content execute         # Execute/process content
```

### Common Flags
```bash
--force                                 # Skip confirmation prompts
--format json|yaml|text                 # Output format
--verbose                               # Detailed output
--timeout <seconds>                     # Operation timeout
```

## 🎯 v2.0 Contract Benefits

### **Semantic Clarity**
- `test` commands validate the RESOURCE itself (health, connectivity, functions)
- `content` commands use the resource's BUSINESS FUNCTIONALITY
- Clear distinction prevents confusion between "testing the tool" vs "using the tool"

### **Consistent Interface**
- All resources follow the same command patterns
- Standardized flags and exit codes
- Predictable behavior across all resources

### **Better Error Handling**
- Standardized exit codes (0=success, 1=error, 2=not-found/skipped)
- Proper timeout handling
- Consistent error messages

## 📋 Migration from v1.0

### Deprecated Patterns (DO NOT USE)
```bash
# OLD v1.0 patterns - DEPRECATED
./manage.sh --action install           # Use: resource-X manage install
./manage.sh --action start             # Use: resource-X manage start
resource-X inject file.json            # Use: resource-X content add --file file.json
```

### Updated Patterns
```bash
# NEW v2.0 patterns
resource-postgres manage install       # Management
resource-postgres content add --file schema.sql  # Content
resource-postgres test smoke          # Validation
resource-postgres status --format json # Monitoring
```

## 🧪 Validation

### Contract Compliance Testing
```bash
# Validate resource against v2.0 contract
./scripts/resources/tools/validate-universal-contract.sh <resource-name>

```

### Test Structure Requirements
```
resources/<name>/
├── cli.sh                    # Primary CLI entry point (REQUIRED)
├── lib/core.sh              # Core functionality (REQUIRED)
├── test/
│   ├── run-tests.sh         # Main test runner (REQUIRED)
│   └── phases/
│       ├── test-smoke.sh    # Quick health check (REQUIRED)
│       ├── test-integration.sh # Full functionality (REQUIRED)
│       └── test-unit.sh     # Library validation (REQUIRED)
└── config/
    ├── defaults.sh          # Default configuration (REQUIRED)
    └── exports.sh           # Environment variable exports (OPTIONAL)
```

## 📊 Performance Requirements

| Command | Max Response Time |
|---------|------------------|
| help | 2 seconds |
| status | 10 seconds |
| logs | 15 seconds |
| test smoke | 30 seconds |
| test integration | 120 seconds |
| manage start | 120 seconds |
| manage stop | 60 seconds |

## 🔒 Security Requirements

- Input validation required
- Never log secrets
- Mask credentials in output
- Use secure storage for keys
- Proper file permissions (cli.sh: 755, lib/*.sh: 644)

## 🚀 Implementation Guide

1. **Create CLI Entry Point**: Implement `cli.sh` with v2.0 command structure
2. **Add Core Library**: Implement required functions in `lib/core.sh`
3. **Implement Tests**: Create required test phases in `test/phases/`
4. **Validate Compliance**: Run contract validation tools
5. **Remove Deprecated**: Delete old `manage.sh` and deprecated patterns

## 📖 Related Documentation

- **Universal Contract**: `/scripts/resources/contracts/v2.0/universal.yaml`
- **Migration Guide**: `/docs/cli-v2-migration/01-migration-guide.md`
- **CLI Framework**: `/scripts/resources/lib/cli-command-framework-v2.sh`
- **Integration Tests**: `/scripts/resources/tests/lib/integration-test-lib.sh`

## 🎯 Contract Enforcement

The v2.0 contract is actively enforced through:
- **Automated validation** during resource startup
- **Integration tests** that verify contract compliance
- **Migration tools** that detect and report deprecated patterns
- **Dashboard monitoring** that tracks migration progress

Resources that don't comply with v2.0 patterns will show warnings and may experience degraded functionality as the platform evolves.