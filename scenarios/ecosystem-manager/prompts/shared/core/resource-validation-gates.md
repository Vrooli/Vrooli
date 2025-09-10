# ✅ Resource Validation Gates

## Purpose
Ensure resources meet v2.0 contract requirements and function correctly before marking tasks complete.

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
Validate resource lifecycle and health using CLI commands:
```bash
# Start and verify health
vrooli resource [name] manage start --wait
vrooli resource [name] status          # Must show running/healthy
vrooli resource [name] test smoke      # Quick health check (<30s)

# Clean shutdown
vrooli resource [name] manage stop     # Must stop cleanly (<30s)
```

### 2. Integration 🔗
Test resource connections and dependencies:
```bash
# Run integration tests
vrooli resource [name] test integration  # Full functionality (<120s)

# Verify dependencies connect
vrooli resource [name] info --json       # Check dependencies list
```

### 3. Documentation 📚
Validate documentation completeness:
- [ ] **PRD.md**: Requirements tracked with `[x] ✅ YYYY-MM-DD` format
- [ ] **README.md**: Overview, usage examples, configuration
- [ ] **lib/*.sh**: Core scripts properly commented
- [ ] **Test commands**: Each P0 requirement has test validation

### 4. Testing 🧪
Run comprehensive test suite:
```bash
# Full test suite (delegates to test/run-tests.sh)
vrooli resource [name] test all        # All phases pass (<600s)

# Individual test phases if needed
vrooli resource [name] test unit       # Library functions (<60s)
vrooli resource [name] test smoke      # Quick validation (<30s)
```

## Resource Contract Compliance

### v2.0 Contract Requirements
Verify mandatory structure exists:
```bash
# Check CLI interface
vrooli resource [name] help            # Shows all commands
vrooli resource [name] info            # Shows runtime.json data

# Verify lifecycle commands work
vrooli resource [name] manage install  # Installs dependencies
vrooli resource [name] manage start    # Starts service
vrooli resource [name] manage stop     # Stops cleanly
vrooli resource [name] manage restart  # Restarts properly

# Content management (if applicable)
vrooli resource [name] content list    # Lists available content
vrooli resource [name] content add     # Can add content
```

### Directory Structure Validation
```bash
# Required directories and files
ls -la resources/[name]/lib/           # core.sh, test.sh required
ls -la resources/[name]/config/        # defaults.sh, schema.json, runtime.json
ls -la resources/[name]/test/          # run-tests.sh, phases/ directory
ls -la resources/[name]/test/phases/   # test-smoke.sh, test-integration.sh
```

## Generator-Specific Gates

### PRD Gate (50% effort for generators)
- [ ] All mandatory sections complete
- [ ] 5+ P0 requirements defined
- [ ] Each P0 has test command
- [ ] Technical specifications documented
- [ ] Integration points identified

### Uniqueness Gate
```bash
# Verify no duplicate exists
grep -r "[resource-name]" /home/matthalloran8/Vrooli/resources/
# Must show <20% functional overlap
```

### Scaffold Gate
- [ ] v2.0 directory structure created
- [ ] Health endpoint responds: `curl -sf http://localhost:${PORT}/health`
- [ ] Basic CLI commands functional
- [ ] One P0 requirement demonstrably works

## Improver-Specific Gates

### PRD Accuracy (20% effort for improvers)
```bash
# Test each claimed ✅ requirement
# If test fails, uncheck the box
# Update progress percentage based on actual completion
```

### No-Regression Gate
```bash
# Run all existing tests
vrooli resource [name] test all        # Previous tests still pass

# Verify performance maintained
time vrooli resource [name] test smoke  # Not slower than before
```

### Progress Gate
- [ ] At least 1 PRD requirement advanced (☐ → ✅)
- [ ] Measurable improvement documented
- [ ] Test commands prove the improvement

## Execution Order
1. **Functional** → Verify lifecycle works
2. **Integration** → Check dependencies
3. **Documentation** → Ensure clarity
4. **Testing** → Validate thoroughly  
5. **Memory** → Update knowledge

**FAIL = STOP** - Fix issues before proceeding

## Quick Validation Commands
```bash
# Full resource validation sequence
vrooli resource [name] manage start --wait && \
vrooli resource [name] test all && \
vrooli resource [name] manage stop

# Quick health check only
vrooli resource [name] test smoke

# Check v2.0 compliance
vrooli resource [name] help | grep -E "manage|test|content|status"
```

## Common Validation Failures

### Port Conflicts
```bash
# Check if port is already in use
netstat -tlnp | grep [PORT]
# Update config/defaults.sh with available port
```

### Missing Dependencies
```bash
# Check runtime.json for dependencies
vrooli resource [name] info --json | jq .dependencies
# Install missing dependencies
vrooli resource [name] manage install
```

### Health Check Timeouts
```bash
# Increase timeout for slow-starting resources
vrooli resource [name] manage start --wait --timeout 120
```

## Remember
- Use CLI commands, not direct script execution
- Follow v2.0 contract strictly
- Test everything you claim works
- Document test commands for validation