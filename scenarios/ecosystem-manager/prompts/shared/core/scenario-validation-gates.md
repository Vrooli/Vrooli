# ✅ Scenario Validation Gates

## Purpose
Ensure scenarios meet requirements, provide business value, and function correctly before marking tasks complete.

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
Validate scenario lifecycle.
**See: 'scenario-testing-reference' section for all Makefile and CLI commands**

### 2. Integration 🔗
Test scenario integrations, APIs, and UI.
**See: 'scenario-testing-reference' section for API, UI, and integration testing**

### 3. Documentation 📚
Validate documentation completeness:
- [ ] **PRD.md**: Business value clear, requirements tracked
- [ ] **README.md**: Installation, usage, API endpoints documented
- [ ] **Makefile**: Help text explains all commands
- [ ] **UI workflows**: User journeys documented (if applicable)

### 4. Testing 🧪
Run comprehensive test suite.
**See: 'scenario-testing-reference' section for test commands**

### 5. Memory 🧠
Update Qdrant knowledge base with scenario details.

## Scenario Structure Validation
**See: 'scenario-testing-reference' section for structure requirements and Makefile commands**

## Generator-Specific Gates

### PRD Gate (50% effort for generators)
- [ ] **Executive Summary**: Clear business value proposition
- [ ] **Requirements**: P0 (must-have), P1 (should-have), P2 (nice-to-have)
- [ ] **Revenue Model**: How scenario generates value
- [ ] **Success Metrics**: Measurable outcomes defined
- [ ] **Technical Architecture**: System design documented
- [ ] **User Journeys**: Primary workflows described

### Uniqueness Gate
```bash
# Verify no duplicate scenario exists
grep -r "[scenario-name]" /home/matthalloran8/Vrooli/scenarios/
# Must provide unique business value (>80% different)
```

### Scaffold Gate
- [ ] Directory structure follows template
- [ ] API health endpoint responds (if API exists)
- [ ] UI loads without errors (if UI exists)
- [ ] One P0 requirement demonstrably works
- [ ] Basic user journey functional

## Improver-Specific Gates

### PRD Accuracy (20% effort for improvers)
- [ ] Test each ✅ requirement - uncheck if broken
- [ ] Update completion percentages
- [ ] Add completion dates: `[x] Feature ✅ YYYY-MM-DD`
- [ ] Verify revenue model still valid

### No-Regression Gate
```bash
# Previous functionality still works
make test         # All existing tests pass

# UI not broken (if applicable)
vrooli resource browserless screenshot http://localhost:[PORT]
# Compare with previous screenshots

# Performance maintained
time curl -sf http://localhost:[PORT]/api/health
```

### Progress Gate
- [ ] At least 1 PRD requirement advanced (☐ → ✅)
- [ ] User-visible improvement delivered
- [ ] Business value increased
- [ ] Test proves the improvement

## Scenario-Specific Validations
**See: 'scenario-testing-reference' section for API, UI, and CLI validation procedures**

## Execution Order
1. **Functional** → Verify scenario runs
2. **Integration** → Check connections and UI
3. **Documentation** → Ensure completeness
4. **Testing** → Validate thoroughly
5. **Memory** → Update knowledge

**FAIL = STOP** - Fix issues before proceeding

## Quick Reference
- **All test commands**: See 'scenario-testing-reference' section
- **Common issues & solutions**: See 'scenario-testing-reference' section
- **Performance requirements**: See 'scenario-testing-reference' section
- **User journey testing**: See 'scenario-testing-reference' section

## Remember
- Use Makefile commands for consistency
- Test user journeys, not just code
- Verify business value delivery
- Screenshot UI for visual validation
- Document what actually works