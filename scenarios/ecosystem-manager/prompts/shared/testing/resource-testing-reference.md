# 🧪 Resource Testing Reference

## Purpose
Single source of truth for all resource testing commands and validation procedures.

## Standard Test Commands

### CLI Testing Commands
```bash
# Full test suite (delegates to test/run-tests.sh)
vrooli resource [name] test all        # All phases (<600s)

# Individual test phases
vrooli resource [name] test smoke      # Quick health check (<30s)
vrooli resource [name] test integration # Full functionality (<120s)
vrooli resource [name] test unit       # Library functions (<60s)
```

### Lifecycle Validation
```bash
# Complete lifecycle test
vrooli resource [name] manage install  # Install dependencies
vrooli resource [name] manage start --wait  # Start with health wait
vrooli resource [name] status          # Show running status
vrooli resource [name] test smoke      # Quick validation
vrooli resource [name] manage stop     # Clean shutdown (<30s)
vrooli resource [name] manage restart  # Restart test
vrooli resource [name] manage uninstall # Clean removal
```

### Health Check Standards
```bash
# Required health endpoint (always with timeout)
timeout 5 curl -sf http://localhost:${PORT}/health

# System health checks
vrooli status                    # Overall system
vrooli status --verbose          # Detailed
vrooli status --json             # JSON format

# Resource-specific health
vrooli resource status           # All resources
vrooli resource status [name]    # Specific resource
vrooli resource status --json    # JSON format
```

### Quick Validation Sequence
```bash
# Full validation in one command
vrooli resource [name] manage start --wait && \
vrooli resource [name] test all && \
vrooli resource [name] manage stop

# Quick health only
vrooli resource [name] test smoke
```

## v2.0 Contract Validation

### Structure Validation
```bash
# Required directories
ls -la resources/[name]/lib/           # core.sh, test.sh required
ls -la resources/[name]/config/        # defaults.sh, schema.json, runtime.json
ls -la resources/[name]/test/          # run-tests.sh, phases/ directory
ls -la resources/[name]/test/phases/   # test-smoke.sh, test-integration.sh

# Contract compliance check
/scripts/resources/tools/validate-universal-contract.sh [name]

# CLI interface check
vrooli resource [name] help | grep -E "manage|test|content|status"
```

### Content Management Testing
```bash
# If resource supports content
vrooli resource [name] content list    # Lists available
vrooli resource [name] content add     # Can add
vrooli resource [name] content get     # Can retrieve
vrooli resource [name] content remove  # Can delete
```

## Test Requirements

### Performance Requirements
- **Smoke tests**: Must complete in <30s
- **Integration tests**: Must complete in <120s  
- **Unit tests**: Must complete in <60s
- **Full suite**: Must complete in <600s (10 minutes)
- **Health checks**: Must respond in <5s (timeout enforced)
- **Shutdown**: Must complete in <30s

### Exit Code Standards
- `0` = Success
- `1` = Error/Failure
- `2` = Not applicable/Skipped

## Generator Testing Requirements

### Uniqueness Validation
```bash
# Verify no duplicate exists
grep -r "[resource-name]" /home/matthalloran8/Vrooli/resources/
# Must show <20% functional overlap
```

### Minimal Functionality Test
```bash
# One P0 requirement must work
vrooli resource [name] manage start --wait
curl -sf http://localhost:${PORT}/health  # Must respond
vrooli resource [name] test smoke          # Must pass
```

## Improver Testing Requirements  

### No-Regression Testing
```bash
# All previous tests must still pass
vrooli resource [name] test all

# Performance must not degrade
time vrooli resource [name] test smoke  # Compare with baseline
```

### Progress Validation
```bash
# Test specific improved features
# Document test command for each PRD requirement advanced
# Example:
vrooli resource [name] test integration --filter [feature]
```

## Common Issues & Solutions

### Port Conflicts
```bash
netstat -tlnp | grep [PORT]            # Check port usage
# Update config/defaults.sh with available port
```

### Missing Dependencies
```bash
vrooli resource [name] info --json | jq .dependencies
vrooli resource [name] manage install  # Install missing
```

### Health Check Timeouts
```bash
# For slow-starting resources
vrooli resource [name] manage start --wait --timeout 120
```

### Test Failures
```bash
# Run with verbose output
vrooli resource [name] test smoke --verbose
vrooli resource [name] test all --debug

# Check logs
vrooli resource [name] logs --tail 50
```

## Testing Best Practices

**DO:**
✅ Always use `timeout` for network calls  
✅ Test edge cases and error conditions  
✅ Verify cleanup after tests  
✅ Use CLI commands, not direct script execution  
✅ Document test commands for each feature  

**DON'T:**
❌ Skip tests to save time  
❌ Test only happy path  
❌ Ignore intermittent failures  
❌ Use hardcoded ports or credentials  
❌ Leave test artifacts behind  

## Required Test Coverage

### Generators Must Test
1. Health endpoint responds
2. Basic lifecycle works (start/stop)
3. One P0 requirement functions
4. No port conflicts
5. CLI commands available

### Improvers Must Test
1. All previous functionality preserved
2. Specific improvements work
3. Performance maintained/improved
4. Documentation accurate
5. Integration points stable

## Test Execution Order
1. **Functional** → Verify lifecycle
2. **Integration** → Check dependencies
3. **Documentation** → Validate accuracy
4. **Testing** → Run full suite
5. **Memory** → Update Qdrant

**Remember**: FAIL = STOP. Fix issues before proceeding.