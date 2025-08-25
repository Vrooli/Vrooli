# Resource Contract System

## Overview

The Resource Contract System defines standardized interfaces that all Vrooli resources must implement. This ensures consistency, reliability, and maintainability across 30+ resources.

## Contract Versions

### v2.0 (Current) - Universal Contract
- **Location**: `/v2.0/universal.yaml`
- **Status**: Active (migrating from v1.0)
- **Key Features**:
  - Single universal contract for ALL resources
  - Grouped commands (manage/test/content)
  - Modern CLI structure (cli.sh)
  - Three-layer validation system
  - Deprecation tracking support

### v1.0 (Legacy) - Category-Based Contracts
- **Location**: `/v1.0/*.yaml`
- **Status**: Deprecated (removal planned for v3.0)
- **Files**:
  - `core.yaml` - Base requirements
  - `ai.yaml`, `storage.yaml`, etc. - Category-specific
- **Issues**:
  - References outdated manage.sh structure
  - Category-based approach doesn't match reality
  - Missing modern CLI features

## Migration Timeline

```
2025-01: v2.0 Released (Current)
2025-06: v1.0 Deprecated (stop accepting v1.0)
2025-12: v1.0 Removed (v2.0 only)
```

## Quick Start

### Check Resource Compliance

```bash
# Validate against v2.0 (recommended)
./tools/validate-universal-contract.sh --resource ollama

# Check all resources
./tools/validate-universal-contract.sh

# Full behavioral validation (Layer 2)
./tools/validate-universal-contract.sh --layer 2 --verbose
```

### Detect Deprecated Patterns

```bash
# Scan for deprecated patterns
./tools/detect-deprecated.sh

# Get migration suggestions
./tools/detect-deprecated.sh --fix-suggestions

# Generate JSON report
./tools/detect-deprecated.sh --format json --output report.json
```

## Contract Structure (v2.0)

### Required Commands

All resources MUST implement these command groups:

#### Core Commands
- `help` - Show comprehensive help with examples
- `status` - Display resource status
- `logs` - View resource logs
- `credentials` - Show integration credentials (if applicable)

#### Management Commands (grouped under `manage`)
- `manage install` - Install resource
- `manage uninstall` - Remove resource
- `manage start` - Start service
- `manage stop` - Stop service
- `manage restart` - Restart service

#### Testing Commands (grouped under `test`)
- `test all` - Run all tests
- `test integration` - Integration tests
- `test unit` - Unit tests (optional)
- `test smoke` - Quick health check

#### Content Commands (grouped under `content`)
- `content add` - Add content/data
- `content list` - List available content
- `content get` - Retrieve content
- `content remove` - Delete content
- `content execute` - Process content (optional)

### File Structure

```
resource-name/
├── cli.sh              # Primary entrypoint (required)
├── lib/
│   ├── core.sh        # Core functionality (required)
│   └── test.sh        # Test implementations (required)
├── config/
│   └── defaults.sh    # Default configuration (required)
└── test/
    └── integration-test.sh  # Integration tests
```

### Deprecated Files
- `manage.sh` → migrate to `cli.sh`
- `manage.bats` → migrate to `lib/test.sh`
- `inject.sh` → use `content add` command

## Validation Layers

### Layer 1: Syntax (✅ Implemented)
- File structure validation
- Command registration checks
- Shell syntax verification
- File permission validation

### Layer 2: Behavioral (🚧 In Progress)
- Command execution testing
- Exit code verification
- Output format validation
- Flag handling checks

### Layer 3: Integration (📋 Planned)
- Full lifecycle testing
- Content management flow
- Performance validation
- Cross-resource integration

## Tools

### Core Tools

| Tool | Purpose | Status |
|------|---------|--------|
| `validate-universal-contract.sh` | Validate v2.0 compliance | ✅ Active |
| `detect-deprecated.sh` | Find deprecated patterns | ✅ Active |
| `validate-interfaces.sh` | Legacy v1.0 validation | ⚠️ Deprecated |

### Migration Tools (Coming Soon)

| Tool | Purpose | Status |
|------|---------|--------|
| `migrate-resource.sh` | Auto-migrate to v2.0 | 📋 Planned |
| `generate-cli.sh` | Generate cli.sh from manage.sh | 📋 Planned |
| `cleanup-deprecated.sh` | Remove deprecated files | 📋 Planned |

## Migration Guide

### For Resource Maintainers

1. **Check Current Status**
   ```bash
   ./tools/detect-deprecated.sh --resource your-resource
   ```

2. **Validate Against v2.0**
   ```bash
   ./tools/validate-universal-contract.sh --resource your-resource
   ```

3. **Migrate Structure** (when tool available)
   ```bash
   ./tools/migrate-resource.sh --resource your-resource
   ```

4. **Update Commands**
   - Group lifecycle commands under `manage`
   - Group test commands under `test`
   - Replace `inject` with `content add`

5. **Test Thoroughly**
   ```bash
   ./tools/validate-universal-contract.sh --layer 2 --resource your-resource
   ```

### For New Resources

1. Use the v2.0 contract from the start
2. Copy structure from a modern resource (e.g., k6)
3. Implement all required commands
4. Validate with Layer 2 testing

## Best Practices

### DO ✅
- Use `cli.sh` as primary entrypoint
- Leverage the CLI command framework
- Group related commands (manage/test)
- Implement all required commands
- Follow the standard file structure
- Include comprehensive help with examples

### DON'T ❌
- Create new `manage.sh` files
- Use `--action` flag pattern
- Skip required commands
- Use deprecated patterns
- Ignore validation warnings

## Backward Compatibility

During the migration period (until v1.0 removal):

- Both `manage.sh` and `cli.sh` can coexist
- Validation tools support both v1.0 and v2.0
- Resources can migrate incrementally
- Deprecation warnings don't block functionality

## FAQ

**Q: Why move from category-based to universal contracts?**
A: Resources don't fit neatly into categories. A universal contract ensures consistency while allowing resource-specific extensions.

**Q: What about resource-specific commands?**
A: Resources can add custom commands beyond the required set. The contract defines the minimum interface.

**Q: How do grouped commands work?**
A: Commands like `manage install` are implemented as subcommands. The CLI framework handles routing.

**Q: When should I migrate my resource?**
A: As soon as possible. v1.0 support ends in 2025-12.

## Support

For questions or issues:
- Check existing resources for examples
- Review the universal contract: `/v2.0/universal.yaml`
- Run validation tools with `--verbose` for details
- Open an issue if you need help migrating