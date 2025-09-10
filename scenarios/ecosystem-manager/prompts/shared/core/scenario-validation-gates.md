# ✅ Scenario Validation Gates

## Purpose
Ensure scenarios meet requirements, provide business value, and function correctly before marking tasks complete.

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
Validate scenario lifecycle using Makefile commands:
```bash
# From scenario directory
make run          # Starts successfully via Vrooli lifecycle
make status       # Shows running status
make test         # All tests pass
make stop         # Stops cleanly
```

**Alternative via Vrooli CLI:**
```bash
vrooli scenario run [name]       # Starts successfully
vrooli scenario status [name]    # Shows running
vrooli scenario stop [name]      # Stops cleanly
```

### 2. Integration 🔗
Test scenario integrations and UI:
```bash
# API health check
curl -sf http://localhost:[PORT]/api/health   # Returns 200 OK

# UI screenshot validation (if UI exists)
vrooli resource browserless screenshot http://localhost:[PORT]
# Then in your response: Read screenshot.png
# Verify UI loads correctly, no errors

# Test resource dependencies
make status       # Shows all required resources
```

### 3. Documentation 📚
Validate documentation completeness:
- [ ] **PRD.md**: Business value clear, requirements tracked
- [ ] **README.md**: Installation, usage, API endpoints documented
- [ ] **Makefile**: Help text explains all commands
- [ ] **UI workflows**: User journeys documented (if applicable)

### 4. Testing 🧪
Run comprehensive test suite:
```bash
# Full test suite via Makefile
make test         # All tests pass
make test-api     # API tests pass (if applicable)
make test-ui      # UI tests pass (if applicable)
make test-integration  # Integration tests pass

# Or individual test files
./test/run-tests.sh       # If exists
```

## Scenario Structure Validation

### Required Structure
```bash
# Check core directories exist
ls -la scenarios/[name]/          # Root directory
ls -la scenarios/[name]/api/      # API implementation (if applicable)
ls -la scenarios/[name]/ui/       # UI implementation (if applicable)  
ls -la scenarios/[name]/cli/      # CLI implementation (if applicable)
ls -la scenarios/[name]/test/     # Test files

# Required files
ls scenarios/[name]/Makefile      # Standard commands
ls scenarios/[name]/PRD.md        # Product requirements
ls scenarios/[name]/README.md     # Documentation
```

### Makefile Commands
Verify standard Makefile targets work:
```bash
make help         # Shows available commands
make run          # Starts scenario
make stop         # Stops scenario
make test         # Runs tests
make logs         # Shows logs
make status       # Shows status
make clean        # Cleans artifacts
```

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

### API Validation (if applicable)
```bash
# Test key endpoints
curl -X GET http://localhost:[PORT]/api/health
curl -X GET http://localhost:[PORT]/api/status
curl -X GET http://localhost:[PORT]/api/[endpoint]

# Check response times
time curl -sf http://localhost:[PORT]/api/health  # <500ms
```

### UI Validation (if applicable)
```bash
# Take screenshot for visual validation
vrooli resource browserless screenshot http://localhost:[PORT]
vrooli resource browserless screenshot http://localhost:[PORT]/[page]

# In your response, examine screenshots:
# Read screenshot.png
# Verify: No errors, UI renders correctly, features visible
```

### CLI Validation (if applicable)
```bash
# Test CLI commands
./cli/[scenario-name] --help
./cli/[scenario-name] [command] --help
./cli/[scenario-name] [command] [args]
```

## Execution Order
1. **Functional** → Verify scenario runs
2. **Integration** → Check connections and UI
3. **Documentation** → Ensure completeness
4. **Testing** → Validate thoroughly
5. **Memory** → Update knowledge

**FAIL = STOP** - Fix issues before proceeding

## Quick Validation Commands
```bash
# Full scenario validation sequence
make run && \
sleep 10 && \
curl -sf http://localhost:[PORT]/api/health && \
make test && \
make stop

# Quick health check only
make status
curl -sf http://localhost:[PORT]/api/health

# UI validation with screenshot
vrooli resource browserless screenshot http://localhost:[PORT]
```

## Common Validation Failures

### Port Conflicts
```bash
# Check if port is in use
lsof -i :[PORT]
# Update .env or config file with available port
```

### Missing Resources
```bash
# Check which resources are required
make status | grep -i "resource"
# Start required resources
vrooli resource [name] manage start
```

### Build Failures
```bash
# Clean and rebuild
make clean
make build
make run
```

### UI Not Loading
```bash
# Check if API is running first
curl -sf http://localhost:[PORT]/api/health

# Check UI build
cd ui && npm install && npm run build
# Or: make build-ui
```

## Remember
- Use Makefile commands for consistency
- Test user journeys, not just code
- Verify business value delivery
- Screenshot UI for visual validation
- Document what actually works