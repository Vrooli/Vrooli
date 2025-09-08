# 📐 v2.0 Resource Contract

## Authoritative Reference

**All v2.0 resource requirements are defined in the official universal contract:**

**File:** `/scripts/resources/contracts/v2.0/universal.yaml`

**Read this file for complete specifications including:**
- Required CLI commands and subcommands
- File structure requirements  
- Runtime configuration schema
- Testing requirements
- Performance and security standards
- Migration guidance

## Key Contract Elements

### Required Commands
- `help` - Show comprehensive help
- `info` - Show runtime configuration  
- `manage` - Lifecycle management (install/start/stop/restart/uninstall)
- `test` - Validation testing (smoke/integration/unit/all)
- `content` - Content management (add/list/get/remove/execute)
- `status` - Show detailed status
- `logs` - View resource logs
- `credentials` - Display integration credentials (optional)

### Required Files
- `cli.sh` - Primary CLI entrypoint
- `lib/core.sh` - Core functionality
- `lib/test.sh` - Test implementations  
- `config/defaults.sh` - Default configuration
- `config/schema.json` - Configuration schema
- `config/runtime.json` - Runtime behavior and dependencies
- `test/run-tests.sh` - Main test runner
- `test/phases/test-smoke.sh` - Quick health validation
- `test/phases/test-integration.sh` - End-to-end functionality
- `test/phases/test-unit.sh` - Library function validation

### Critical Requirements
1. **Runtime Configuration** - Must define startup_order, dependencies, timeouts
2. **Health Validation** - Smoke tests must complete in <30s
3. **Standard Exit Codes** - 0=success, 1=error, 2=not-applicable
4. **Timeout Handling** - All operations must have timeout limits
5. **Graceful Shutdown** - Stop commands must handle cleanup

## Validation Commands

```bash
# Validate contract compliance
/scripts/resources/tools/validate-universal-contract.sh <resource-name>

# Test smoke tests (must be <30s)
resource-<name> test smoke

# Test all lifecycle hooks
resource-<name> manage install
resource-<name> manage start
resource-<name> status
resource-<name> manage stop
resource-<name> manage uninstall
```

## Common Compliance Issues

### ❌ Missing Runtime Config
```bash
# Must exist: config/runtime.json
{
  "startup_order": 500,
  "dependencies": ["postgres"],
  "startup_timeout": 60,
  "startup_time_estimate": "10-30s", 
  "recovery_attempts": 3,
  "priority": "medium"
}
```

### ❌ No Timeout in Health Checks
```bash
# BAD
curl http://localhost:$PORT/health

# GOOD (per universal.yaml spec)
timeout 5 curl -sf http://localhost:$PORT/health
```

### ❌ Incomplete Command Structure
```bash
# BAD - Missing required subcommands
resource-name help
resource-name status

# GOOD - Full command structure per universal.yaml
resource-name manage install
resource-name manage start  
resource-name test smoke
resource-name content list
```

## Remember

**The universal.yaml file is the single source of truth** for all v2.0 requirements. When implementing or improving resources:

1. **Read universal.yaml first** - Complete specification
2. **Validate compliance** - Use provided validation tools
3. **Follow patterns** - Consistency across all resources  
4. **Test thoroughly** - All test phases must work

Every resource following the universal contract integrates seamlessly and works reliably.