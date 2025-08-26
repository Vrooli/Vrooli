# Population System v2.0

**Clean, simple scenario-based content population for Vrooli resources**

## Overview

The Population System enables scenarios to populate resources with content (workflows, configurations, data). It follows the v2.0 universal contract where all resources use consistent `content add` patterns.

## Quick Start

```bash
# List available scenarios
./populate.sh list

# Populate resources from a scenario
./populate.sh add my-scenario

# Preview what would be populated (dry-run)
./populate.sh add my-scenario --dry-run

# Validate scenario configuration
./populate.sh validate my-scenario
```

## Commands

### `add <scenario>`
Deploy content from a scenario to resources.

Options:
- `--dry-run` - Preview changes without executing
- `--parallel` - Enable parallel processing (default)
- `--verbose` - Show detailed output

### `validate <scenario>`
Check if a scenario configuration is valid.

### `list`
Show all available scenarios.

### `status`
Display current resource availability.

## Scenario Format

Scenarios define what content to populate into which resources:

```json
{
  "name": "my-application",
  "description": "Complete application setup",
  "resources": {
    "n8n": {
      "content": [
        {
          "type": "workflow",
          "file": "workflows/user-onboarding.json",
          "name": "User Onboarding"
        }
      ]
    },
    "postgres": {
      "content": [
        {
          "type": "schema",
          "file": "sql/database-schema.sql"
        }
      ]
    }
  }
}
```

## Scenario Locations

The system searches for scenarios in these locations:
1. `$VROOLI_SCENARIOS_DIR` (if set)
2. `$APP_ROOT/scenarios/`
3. `~/.vrooli/scenarios/`
4. `./scenarios/` (current directory)

## Content Types by Resource

### n8n
- `workflow` - Automation workflows
- `credential` - API credentials

### postgres
- `schema` - Database schemas
- `migration` - Schema migrations
- `seed` - Seed data

### qdrant
- `collection` - Vector collections
- `vectors` - Vector data

### minio
- `bucket` - Storage buckets
- `file` - Files to upload

### windmill
- `script` - Python/TypeScript scripts
- `app` - Windmill applications

## v2.0 Contract Compliance

All resources must implement the v2.0 content interface:
```bash
resource-<name> content add --file <path>
resource-<name> content list
resource-<name> content get <id>
resource-<name> content remove <id>
```

## Architecture

```
populate/
├── populate.sh          # Main entry point (80 lines)
├── lib/
│   ├── core.sh        # Core logic (150 lines)
│   ├── content.sh     # Content management (130 lines)
│   ├── validate.sh    # Validation (140 lines)
│   └── scenario.sh    # Scenario loading (150 lines)
└── docs/
    └── README.md      # This file
```

Total: ~650 lines (vs 1000+ in old system)

## Key Improvements from v1.0

1. **Simplicity** - Reduced from 1000+ lines to ~650
2. **Clarity** - Clean command structure: `populate.sh add <scenario>`
3. **Consistency** - All resources use `content add` pattern
4. **Reliability** - Proper validation before execution
5. **Performance** - Parallel processing by default
6. **Maintainability** - Small, focused functions

## Troubleshooting

### Resource Not Found
```
Resource CLI not found: n8n
```
**Solution:** Install the resource CLI or ensure it's in PATH.

### Resource Not Running
```
Resource not running: postgres
```
**Solution:** Start the resource first: `resource-postgres manage start`

### File Not Found
```
File not found: workflows/my-workflow.json
```
**Solution:** Check file paths are relative to scenario directory.

### Invalid JSON
```
Invalid JSON format
```
**Solution:** Validate JSON syntax with `jq` or online validator.

## Examples

### Simple Scenario
```json
{
  "name": "hello-world",
  "resources": {
    "n8n": {
      "content": [
        {
          "type": "workflow",
          "file": "hello.json"
        }
      ]
    }
  }
}
```

### Complex Scenario
```json
{
  "name": "saas-platform",
  "description": "Complete SaaS application",
  "resources": {
    "postgres": {
      "content": [
        {"type": "schema", "file": "schema.sql"},
        {"type": "seed", "file": "initial-data.sql"}
      ]
    },
    "n8n": {
      "content": [
        {"type": "workflow", "file": "onboarding.json"},
        {"type": "workflow", "file": "billing.json"},
        {"type": "credential", "file": "stripe-api.json"}
      ]
    },
    "qdrant": {
      "content": [
        {"type": "collection", "name": "products"},
        {"type": "vectors", "file": "product-embeddings.json"}
      ]
    }
  }
}
```

## Contributing

When adding support for new resources:

1. Ensure resource implements v2.0 `content` commands
2. Add content type validation in `lib/validate.sh`
3. Update content handling in `lib/content.sh`
4. Document content types in this README

## Migration from v1.0

Old v1.0 commands map to v2.0 as follows:

```bash
# Old
engine.sh --action inject --scenario NAME --config-file PATH

# New
populate.sh add NAME

# Old
engine.sh --action validate --scenario NAME

# New
populate.sh validate NAME

# Old
engine.sh --action list-scenarios

# New
populate.sh list
```

## Support

For issues or questions, check:
- Resource documentation in `/resources/<name>/README.md`
- Universal contract in `/scripts/resources/contracts/v2.0/`
- Test examples in `/scripts/resources/populate/test/`