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

### 5. Security & Standards 🔒
Run Scenario Auditor scans per `security-requirements` and resolve/record the worst violations.

## Execution Order
1. **Functional** → Verify scenario runs
2. **Integration** → Check connections and UI
3. **Documentation** → Ensure completeness
4. **Testing** → Validate thoroughly
5. **Security & Standards** → Capture Scenario Auditor results

**FAIL = STOP** - Fix issues before proceeding

## Remember
- Use Makefile commands for consistency
- Test user journeys, not just code
- Verify business value delivery
- Screenshot UI for visual validation
- Document what actually works
