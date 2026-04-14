# Resource Interface Standards v2.0

> Historical/transitional document. This contract captures the shell-era universal CLI shape and remains relevant only for explicit compatibility surfaces. It is not the default starting point for new resources after the manifest-native migration. Use the blueprint/template workflow in [README.md](README.md) for new resource work. Canonical resources are now manifest-native and use `resource.json` as the single source of truth.

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
resource-<name> install                 # Install the resource
resource-<name> start                   # Start the service
resource-<name> stop                    # Stop the service
resource-<name> restart                 # Restart the service
resource-<name> uninstall               # Remove the resource
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
./manage.sh --action install           # Use: resource-X install
./manage.sh --action start             # Use: resource-X start
resource-X inject file.json            # Use: resource-X content add --file file.json
```

### Updated Patterns
```bash
# NEW v2.0 patterns
resource-postgres install              # Management
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
├── resource.json            # Canonical manifest (REQUIRED for native resources)
├── cli/                     # Go CLI module (main.go, go.mod, install wrappers)
├── lib/                     # Optional compatibility helpers only
├── test/
│   ├── run-tests.sh         # Main test runner (REQUIRED)
│   └── phases/
│       ├── test-smoke.sh    # Quick health check (REQUIRED)
│       ├── test-integration.sh # Full functionality (REQUIRED)
│       └── test-unit.sh     # Library validation (REQUIRED)
```

For native resources, runtime configuration, dependency authoring schema, and environment exports belong in `resource.json`, not `config/runtime.json`, `config/schema.json`, or `config/exports.sh`.

## 📊 Performance Requirements

| Command | Max Response Time |
|---------|------------------|
| help | 2 seconds |
| status | 10 seconds |
| logs | 15 seconds |
| test smoke | 30 seconds |
| test integration | 120 seconds |
| start | 120 seconds |
| stop | 60 seconds |

## 🔒 Security Requirements

- Input validation required
- Never log secrets
- Mask credentials in output
- Use secure storage for keys
- Proper file permissions for optional compatibility scripts and generated assets

## 🚀 Implementation Guide

1. **Start with a canonical template**: Scaffold a resource from `templates/resources/<archetype>/`
2. **Define the native manifest**: Put runtime, dependency schema, orchestration, and environment exports in `resource.json`
3. **Implement Tests**: Create required test phases in `test/phases/`
4. **Validate Compliance**: Run contract validation tools
5. **Use shell only as explicit compatibility code**: Do not introduce new canonical `config/runtime.json` or `config/schema.json` files

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
