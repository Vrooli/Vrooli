# Progress Verification Framework

## Measuring Real Progress vs Activity

### 1. Meaningful Progress Metrics

Track what matters:
```yaml
meaningful_progress:
  - P0_requirements_completed: count
  - P1_requirements_completed: count  
  - Tests_passing: percentage
  - User_facing_features: count
  - Revenue_impacting_features: count
  - Integration_points_working: count

activity_metrics: # Less important
  - Lines_of_code_changed: count
  - Files_modified: count
  - Comments_added: count
```

### 2. Progress Validation Tests

Before marking progress:
```bash
# Functional test
./test.sh feature_name

# Integration test
curl -X POST http://localhost:$PORT/api/feature

# End-to-end test
vrooli scenario test --e2e

# Performance test
ab -n 100 -c 10 http://localhost:$PORT/api/endpoint
```

### 3. Visual UI Validation (CRITICAL for Scenarios with UI)

**For ANY scenario with a user interface, you MUST validate visual appearance:**

```bash
# 1. Start the scenario
vrooli scenario [name] develop

# 2. Take screenshots of major views
resource-browserless screenshot http://localhost:[PORT]/ --output /tmp/homepage.png
resource-browserless screenshot http://localhost:[PORT]/dashboard --output /tmp/dashboard.png
resource-browserless screenshot http://localhost:[PORT]/form --output /tmp/form.png

# 3. READ AND INSPECT each screenshot (THIS IS CRITICAL!)
Read /tmp/homepage.png     # Actually view the UI - check for visual issues
Read /tmp/dashboard.png    # Look for layout breaks, missing elements
Read /tmp/form.png         # Verify forms render correctly

# 4. Document what you see
## Visual Validation Checklist
- [ ] Layout is intact (no overlapping elements)
- [ ] All text is readable and properly formatted
- [ ] Buttons are visible and properly styled
- [ ] Forms render with all fields visible
- [ ] No missing images, icons, or assets
- [ ] Colors match expected theme
- [ ] Responsive design works (if applicable)
- [ ] Error messages display correctly
- [ ] Loading states appear properly
- [ ] Navigation elements are accessible

# 5. Before/After comparison for improvements
# BEFORE making changes:
resource-browserless screenshot http://localhost:[PORT] --output /tmp/before.png
Read /tmp/before.png  # Document current state

# AFTER making changes:
resource-browserless screenshot http://localhost:[PORT] --output /tmp/after.png
Read /tmp/after.png   # Compare to before

# Document improvements or regressions:
## Visual Changes
- Improvements: [list visual improvements]
- Regressions: [list any visual degradation]
- Unchanged: [list what remained the same]
```

**Visual Validation Failure Protocol:**
- If UI is broken, DO NOT mark UI features as complete
- Take additional screenshots of problem areas
- Document specific visual issues with screenshots
- Include visual fixes in improvement plan

### 4. Incremental Progress Tracking

Document each iteration:
```markdown
## Iteration #N Progress
- **Started**: [timestamp]
- **Completed**: [timestamp]
- **Items Attempted**: [list]
- **Items Completed**: [list]
- **Items Failed**: [list with reasons]
- **Rollbacks Required**: [any reverted changes]
- **Net Progress**: [actual advancement]
```

### 5. Anti-Progress Detection

Watch for backwards movement:
- Previously working features now broken
- Test coverage decreased
- Performance degradation
- New bugs introduced
- Documentation outdated by changes
- **UI regressions (visual breaks, layout issues)**

### 6. Progress Velocity Calculation

```
velocity = (P0_complete * 3 + P1_complete * 2 + P2_complete * 1) / time_spent
health = tests_passing_percentage * 0.4 + docs_current * 0.2 + no_regressions * 0.4
real_progress = velocity * health
```

## Progress Gates

Must pass before claiming progress:
1. **Functionality Gate**: Feature actually works
2. **Stability Gate**: Doesn't break existing features
3. **Test Gate**: Has meaningful tests
4. **Documentation Gate**: Docs updated
5. **Integration Gate**: Works with rest of system

## Progress Report Template

```markdown
## Progress Report - [Date]

### Claimed Progress
- [What was supposed to be done]

### Actual Progress  
- [What really got done]
- [What still needs work]

### Evidence of Progress
- Test results: [link/output]
- Demo: [screenshot/recording]
- Metrics: [before/after]

### Blockers Encountered
- [List any obstacles]

### Next Steps
- [Specific next actions]
```

## Regression Prevention

Before marking complete:
1. Run full test suite
2. Check dependent features
3. Verify backwards compatibility
4. Test edge cases
5. Load test if applicable

## Net Progress Formula

```
net_progress = new_features_working - features_broken - technical_debt_added
```

Only positive net progress counts!