# Progress Verification & PRD Accuracy

## Core Philosophy: Trust But Verify
**PRD checkboxes often lie.** Never fake progress. The PRD is sacred - it must reflect reality.

## PRD Checkbox Standards

### ✅ Complete = ALL of:
1. Fully implemented as specified
2. Tests pass consistently  
3. Documentation exists
4. Can be demoed successfully
5. No known issues

### ☐ Incomplete = ANY of:
- Missing functionality
- Tests fail
- No documentation
- Errors during demo
- Known bugs

### Partial Work:
```markdown
- [ ] Search with filters
  - Basic search works
  - Filter UI incomplete
  - Date filtering broken
```

## Assessment Protocol

### 1. Verify Current State
```bash
# Test each ✅ in PRD
vrooli [resource/scenario] [name] develop
timeout 5 curl -sf http://localhost:${PORT}/health
./test.sh

# For EACH checked item:
grep -r "feature_name" . --include="*.go" --include="*.js"
curl -X POST http://localhost:PORT/api/feature
# If broken, uncheck it immediately
```

### 2. Common False Positives
- ✅ "Health checks implemented" → Missing timeout handling
- ✅ "CLI commands complete" → Missing help text
- ✅ "Error handling robust" → Just try/catch, no recovery
- ✅ "UI responsive" → Layout breaks on mobile
- ✅ "Tests comprehensive" → Only happy path covered

### 3. Visual UI Validation (Scenarios with UI)
```bash
# Take screenshots
resource-browserless screenshot http://localhost:PORT --output /tmp/ui.png
Read /tmp/ui.png  # MUST inspect visually

# Check for:
- [ ] Layout intact
- [ ] Text readable
- [ ] Forms render
- [ ] No missing assets
- [ ] Error states visible
```

## Progress Metrics

### What Counts:
```yaml
real_progress:
  - P0_requirements_completed
  - Tests_passing_percentage
  - Integration_points_working
  - UI_features_visual_verified
  
not_progress:
  - Lines_changed
  - Files_touched
  - Comments_added
```

### Net Progress Formula:
```
net = features_working - features_broken - debt_added
# Only positive net progress counts!
```

## Regression Detection

### Before ANY Change:
1. Capture current state (tests, screenshots)
2. Document what works
3. Set rollback point

### After Change:
1. Run full test suite
2. Check dependent features
3. Compare screenshots
4. Verify no breaks

### If Regression Found:
- STOP immediately
- Document the break
- Rollback or fix
- Never hide regressions

## Progress Report Template
```markdown
## [Date] Progress

### Verified Complete
- [Feature]: [test command that proves it works]

### Partial Progress
- [Feature]: [what works] / [what doesn't]

### Regressions
- [What broke]: [error details]

### Net Progress
- Added: X features
- Broken: Y features
- Net: X-Y
```

## Quick Validation Commands
```bash
# PRD accuracy check
for item in $(grep "✅" PRD.md); do
  echo "Testing: $item"
  # Run specific test
done

# Regression check
./test.sh --all
git diff HEAD~1 test-results.txt

# Visual check (UI only)
resource-browserless screenshot http://localhost:PORT
Read screenshot.png
```

## Remember
- **Never** mark incomplete work as done
- **Always** test before checking
- **Document** partial progress
- **Uncheck** broken features immediately
- **Net positive** progress only
