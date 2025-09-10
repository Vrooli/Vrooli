# Iteration Phase

## Core Principle
**One change at a time.** Test. Commit. Never break working features.

## Iteration Process

### 1. Pre-Iteration Checkpoint
```bash
timeout 5 curl -sf http://localhost:${PORT}/health > /tmp/before.txt
./test.sh >> /tmp/before.txt
# Document what currently works
```

### 2. Implementation Pattern
```python
def enhanced_function(params):
    result = original_logic(params)  # Keep existing
    if new_feature_enabled:           # Add new
        result = enhance_result(result)
    return result
```

### 3. Validation After EACH Change
```bash
# Test new feature
curl -X POST localhost:PORT/api/new_feature

# Verify old features still work  
timeout 5 curl -sf http://localhost:${PORT}/health
./test.sh

# Check for regressions
diff /tmp/before.txt /tmp/after.txt
```

### 4. Update PRD Immediately
Update checkboxes following the format in the 'prd-protocol' section.

## Common Patterns

```yaml
Authentication:
  add: middleware → protected_routes
  preserve: public_routes → no_auth
  test: both_work

Health_Checks:
  add: timeout_handling
  preserve: existing_checks
  test: all_respond

CLI_Commands:
  add: new_command
  preserve: existing_commands
  test: help_shows_all
```

## Iteration Checklist
- [ ] Checkpoint before change
- [ ] Single focused change
- [ ] Test new functionality
- [ ] Verify no regressions
- [ ] Update PRD (see 'prd-protocol' section)
- [ ] Commit if successful

## Failure Protocol
1. If tests fail → revert immediately
2. If regression found → stop and fix
3. If unclear → smaller change
4. Never hide failures

## Remember
- Progress = working_features++
- Not progress = broken_features++
- Small wins > big failures