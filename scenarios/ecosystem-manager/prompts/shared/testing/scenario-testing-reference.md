# 🧪 Scenario Testing Reference

## Purpose
Single source of truth for all scenario testing commands and validation procedures.

## Standard Test Commands

### Makefile Testing (PREFERRED)
```bash
# From scenario directory - comprehensive management
make run          # Start via Vrooli lifecycle
make status       # Show running status
make test         # Run all tests
make logs         # View logs
make stop         # Stop cleanly
make clean        # Clean artifacts
make help         # Show available commands
```

### CLI Testing (ALTERNATIVE)
```bash
# Direct Vrooli CLI commands
vrooli scenario run [name]       # Start scenario
vrooli scenario status [name]    # Check status
vrooli scenario test [name]      # Run tests
vrooli scenario logs [name]      # View logs
vrooli scenario stop [name]      # Stop scenario
```

### API Testing
```bash
# Health check (required)
curl -sf http://localhost:[PORT]/api/health

# Status endpoint
curl -sf http://localhost:[PORT]/api/status

# Test specific endpoints
curl -X GET http://localhost:[PORT]/api/[endpoint]
curl -X POST http://localhost:[PORT]/api/[endpoint] \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'

# Response time validation
time curl -sf http://localhost:[PORT]/api/health  # Should be <500ms
```

### UI Testing (if applicable)
```bash
# Take screenshots for validation
vrooli resource browserless screenshot http://localhost:[PORT]
vrooli resource browserless screenshot http://localhost:[PORT]/[page]

# In your response, validate screenshots:
# Read screenshot.png
# Verify: No errors, correct rendering, features visible

# Basic UI availability
curl -sf http://localhost:[PORT]  # Should return HTML
```

### CLI Testing (if scenario has CLI)
```bash
./cli/[scenario-name] --help
./cli/[scenario-name] [command] --help
./cli/[scenario-name] [command] [args]
```

## Test Suite Structure

### Required Test Files
```bash
# Check test structure
ls -la scenarios/[name]/test/          # Test directory
ls -la scenarios/[name]/test/run-tests.sh  # Main runner
ls -la scenarios/[name]/test/api/      # API tests
ls -la scenarios/[name]/test/ui/       # UI tests (if applicable)
ls -la scenarios/[name]/test/integration/  # Integration tests
```

### Test Phases via Makefile
```bash
make test              # All tests
make test-api          # API tests only
make test-ui           # UI tests only (if applicable)
make test-integration  # Integration tests
make test-smoke        # Quick validation
```

## Quick Validation Sequences

### Full Validation
```bash
# Complete validation sequence
make run && \
sleep 10 && \
curl -sf http://localhost:[PORT]/api/health && \
make test && \
make stop
```

### Quick Health Check
```bash
make status
curl -sf http://localhost:[PORT]/api/health
```

### UI Validation with Screenshot
```bash
make run
sleep 10
vrooli resource browserless screenshot http://localhost:[PORT]
# Examine screenshot.png in response
make stop
```

## Structure Validation

### Required Directories
```bash
ls -la scenarios/[name]/          # Root directory
ls -la scenarios/[name]/api/      # API implementation
ls -la scenarios/[name]/ui/       # UI implementation (if applicable)
ls -la scenarios/[name]/cli/      # CLI implementation (if applicable)
ls -la scenarios/[name]/test/     # Test files
ls -la scenarios/[name]/initialization/  # Setup files (if needed)
```

### Required Files
```bash
ls scenarios/[name]/Makefile      # Standard commands
ls scenarios/[name]/PRD.md        # Product requirements
ls scenarios/[name]/README.md     # Documentation
ls scenarios/[name]/service.json  # Service configuration
```

## Generator Testing Requirements

### Uniqueness Validation
```bash
# Verify no duplicate scenario
grep -r "[scenario-name]" /home/matthalloran8/Vrooli/scenarios/
# Must provide >80% unique business value
```

### Minimal Functionality Test
```bash
# One P0 requirement must work
make run
curl -sf http://localhost:[PORT]/api/health  # Must respond
# Test one specific P0 feature
make test-smoke  # Must pass
make stop
```

### User Journey Validation
```bash
# At least one complete user workflow must function
# Document the test sequence for the journey
# Example:
make run
curl -X POST http://localhost:[PORT]/api/register -d '...'
curl -X POST http://localhost:[PORT]/api/login -d '...'
curl -X GET http://localhost:[PORT]/api/dashboard
make stop
```

## Improver Testing Requirements

### PRD Accuracy Validation
```bash
# Test each ✅ requirement
# If broken, uncheck in PRD
# Update completion percentages
# Add dates: [x] Feature ✅ YYYY-MM-DD
```

### No-Regression Testing
```bash
# All previous tests must pass
make test

# UI must not be broken (if applicable)
vrooli resource browserless screenshot http://localhost:[PORT]
# Compare with baseline screenshots

# API performance maintained
time curl -sf http://localhost:[PORT]/api/health
```

### Progress Validation
```bash
# Test specific improvements
# Document commands that prove advancement
# Example for new feature:
curl -X POST http://localhost:[PORT]/api/new-feature -d '...'
```

## Common Issues & Solutions

### Port Conflicts
```bash
lsof -i :[PORT]                    # Check port usage
# Update .env or config with available port
```

### Missing Resources
```bash
make status | grep -i "resource"   # Check requirements
vrooli resource [name] manage start  # Start required resources
```

### Build Failures
```bash
make clean                         # Clean artifacts
make build                         # Rebuild
make run                          # Try again
```

### UI Not Loading
```bash
# Check API first
curl -sf http://localhost:[PORT]/api/health

# Rebuild UI
cd ui && npm install && npm run build
# Or: make build-ui
```

### Test Timeouts
```bash
# Increase timeout for slow scenarios
make test TIMEOUT=300  # 5 minutes

# Or in test files
timeout 120 [test-command]  # 2 minutes
```

## Testing Best Practices

**DO:**
✅ Use Makefile commands for consistency  
✅ Test complete user journeys  
✅ Validate business value delivery  
✅ Screenshot UI for visual validation  
✅ Test with realistic data  

**DON'T:**
❌ Test only code, ignore user experience  
❌ Skip integration tests  
❌ Ignore UI/UX issues  
❌ Test with minimal data  
❌ Leave test data in production config  

## Required Test Coverage

### Generators Must Test
1. API health endpoint responds
2. Makefile commands work
3. One P0 requirement functions
4. One user journey completes
5. No conflicts with other scenarios

### Improvers Must Test
1. All previous features preserved
2. Specific improvements work
3. User experience improved/maintained
4. Business value increased
5. PRD accuracy validated

## Performance Requirements

### Response Times
- **API Health**: <500ms
- **UI Load**: <3 seconds
- **API Endpoints**: <2 seconds (typical)
- **Data Operations**: <5 seconds

### Test Execution Times
- **Smoke Tests**: <1 minute
- **API Tests**: <5 minutes
- **UI Tests**: <10 minutes (if applicable)
- **Full Suite**: <15 minutes

## Test Execution Order
1. **Functional** → Verify scenario runs
2. **Integration** → Check connections and UI
3. **User Journeys** → Validate workflows
4. **Business Value** → Confirm value delivery
5. **Memory** → Update Qdrant

**Remember**: Test user value, not just code. Business impact matters most.