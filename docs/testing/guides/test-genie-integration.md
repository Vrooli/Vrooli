# AI-Powered Test Generation with test-genie

> **Generate contextual tests automatically using AI, then integrate with requirement tracking**

## Table of Contents
- [Overview](#overview)
- [When to Use test-genie](#when-to-use-test-genie)
- [Quick Start](#quick-start)
- [Coverage-Driven Development](#coverage-driven-development)
- [Generated Test Quality](#generated-test-quality)
- [Integration with Requirement Tracking](#integration-with-requirement-tracking)
- [Best Practices](#best-practices)
- [Limitations](#limitations)
- [See Also](#see-also)

## Overview

**test-genie** is a Vrooli scenario that analyzes your codebase and generates contextual unit and integration tests using AI. It integrates with the requirement tracking system to help fill coverage gaps efficiently.

**Key capabilities**:
- ✅ Analyzes codebase structure and patterns
- ✅ Generates Go, Vitest (TypeScript/React), and Python tests
- ✅ Suggests test cases based on code complexity
- ✅ Identifies untested code paths
- ✅ Can be integrated into requirement-driven workflows

**Current status** (as of 2025-10-03):
- P0 requirements complete (4.8% internal test coverage, targeting 95%)
- P1 features pending (visual coverage analysis, cross-scenario integration tests)
- Generation quality depends on AI model capabilities

## When to Use test-genie

### ✅ **Good Use Cases**

**1. Filling Coverage Gaps**
```bash
# After writing core functionality, identify what needs testing
vrooli scenario requirements report my-scenario --show-gaps
test-genie generate my-scenario --types unit --target-gaps
```

**2. Bootstrap Testing for New Scenarios**
```bash
# Generate initial test suite for new scenario
test-genie generate my-new-scenario --types unit,integration
```

**3. Adding Tests to Legacy Code**
```bash
# Generate tests for existing untested code
test-genie generate old-scenario --types unit --comprehensive
```

**4. Exploring Test Patterns**
```bash
# Generate examples to learn testing patterns in the codebase
test-genie generate example-scenario --types unit --output ./examples/
```

### ❌ **Poor Use Cases**

**1. Complex Business Logic**
- AI may not understand domain-specific requirements
- Manual tests better capture business rules

**2. Integration Tests Requiring Setup**
- AI can't provision external services
- Manual setup scripts required

**3. UI Component Testing**
- Use BAS workflows for UI automation instead
- More maintainable than generated Playwright tests

**Decision Tree**:
```
Need tests for...
│
├─ Business logic / domain rules?
│  └─ ❌ Write manually (AI won't understand domain)
│
├─ UI interactions?
│  └─ ✅ Use BAS workflows (see ui-automation-with-bas.md)
│
├─ Pure functions / utilities?
│  └─ ✅ Use test-genie (generates good unit tests)
│
├─ API handlers with mocked DB?
│  └─ ✅ Use test-genie as starting point (review carefully)
│
└─ Integration tests with real services?
   └─ ⚠️ Use test-genie for structure, write assertions manually
```

## Quick Start

### 1. Ensure test-genie is Running

```bash
# Check status
vrooli scenario status test-genie

# Start if needed
vrooli scenario start test-genie
```

### 2. Generate Tests for Scenario

```bash
# Generate unit tests only
test-genie generate my-scenario --types unit

# Generate unit + integration tests
test-genie generate my-scenario --types unit,integration

# Comprehensive generation (all test types)
test-genie generate my-scenario --comprehensive
```

### 3. Review Generated Tests

Generated tests are saved to:
- Go: `api/*_test.go` (alongside source files)
- Vitest: `ui/src/**/__tests__/*.test.ts`
- Python: `scripts/test_*.py`

**Example output**:
```
✅ Generated 12 unit tests for my-scenario:
   - api/handlers_test.go (5 tests)
   - api/services/project_service_test.go (3 tests)
   - ui/src/stores/__tests__/projectStore.test.ts (4 tests)

⚠️  Review before committing:
   - Verify test assertions match expected behavior
   - Add [REQ:ID] tags for requirement tracking
   - Adjust mocks if needed
```

### 4. Add Requirement Tags

```typescript
// BEFORE (generated)
describe('projectStore', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});

// AFTER (tagged for tracking)
describe('projectStore [REQ:MY-PROJECT-CRUD]', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});
```

### 5. Run and Verify

```bash
cd scenarios/my-scenario
./test/phases/test-unit.sh

# Check that tests pass and requirements tracked
vrooli scenario requirements report my-scenario
```

## Coverage-Driven Development

### Workflow: Identify → Generate → Review → Track

```mermaid
graph TB
    A[Check Coverage<br/>vrooli scenario requirements report] --> B{Coverage Gaps?}
    B -->|Yes| C[Generate Tests<br/>test-genie generate]
    B -->|No| Z[Done!]
    C --> D[Review Generated Tests<br/>Verify assertions]
    D --> E[Add [REQ:ID] Tags]
    E --> F[Run Tests<br/>./test/run-tests.sh]
    F --> G{Tests Pass?}
    G -->|No| H[Fix Tests<br/>Adjust generated code]
    H --> F
    G -->|Yes| I[Commit Tests]
    I --> A

    style A fill:#e1f5ff
    style C fill:#fff3e0
    style D fill:#fff9c4
    style F fill:#e8f5e9
    style I fill:#c8e6c9
```

### Example: Filling Gaps Systematically

**Step 1: Identify gaps**
```bash
$ vrooli scenario requirements report my-scenario --format json > coverage.json
$ cat coverage.json | jq '.requirements[] | select(.status != "complete") | .id'
"MY-PROJECT-CREATE"
"MY-WORKFLOW-EXECUTE"
"MY-USER-AUTH"
```

**Step 2: Generate tests for specific requirements**
```bash
# Generate tests targeting specific files
test-genie generate my-scenario \
  --types unit \
  --files "api/handlers/projects.go,api/services/auth.go"
```

**Step 3: Review and tag**
```go
// Generated test
func TestCreateProject(t *testing.T) {
    // ... generated code ...
}

// After review and tagging
func TestCreateProject(t *testing.T) {
    t.Run("creates project with valid data [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // ... generated code (reviewed and adjusted) ...
    })
}
```

**Step 4: Verify coverage improved**
```bash
./test/run-tests.sh
vrooli scenario requirements report my-scenario

# Output shows:
# MY-PROJECT-CREATE: pending → in_progress (1 validation implemented)
```

## Generated Test Quality

### What test-genie Does Well

✅ **Structural patterns**
```typescript
// Generated Vitest test structure
describe('Service', () => {
  beforeEach(() => {
    // Proper setup
  });

  it('handles success case', async () => {
    // Mock setup
    // Function call
    // Assertions
  });

  it('handles error case', async () => {
    // Error scenario
  });
});
```

✅ **Mock setup**
```go
// Generated Go mock setup
mockDB := &MockDatabase{
    GetByIDFunc: func(id string) (*Project, error) {
        return &Project{ID: id, Name: "Test"}, nil
    },
}
```

✅ **Basic assertions**
```typescript
expect(result).toBeDefined();
expect(result.id).toBe('expected-id');
expect(error).toBeNull();
```

### What Requires Human Review

⚠️ **Business logic validation**
```typescript
// Generated (may be wrong!)
expect(discountedPrice).toBe(100);

// Review needed: Is discount calculation correct?
// Should verify: originalPrice * (1 - discountRate) === discountedPrice
```

⚠️ **Edge cases**
```go
// Generated covers happy path
TestCreate() // success case

// You should add:
TestCreateWithEmptyName() // validation
TestCreateWithDuplicateName() // conflict
TestCreateWhenDBDown() // error handling
```

⚠️ **Complex state transitions**
```typescript
// Generated may miss state machine logic
it('transitions from draft to published', () => {
  // May not verify intermediate states or side effects
});
```

### Review Checklist

Before committing generated tests:

- [ ] **Assertions match expected behavior** (not just "truthy")
- [ ] **Edge cases covered** (empty, null, boundary values)
- [ ] **Error scenarios tested** (network failures, validation errors)
- [ ] **Mocks realistic** (don't mock away important logic)
- [ ] **Test names descriptive** (explain what's being tested)
- [ ] **[REQ:ID] tags added** (for requirement tracking)

## Integration with Requirement Tracking

### Adding Generated Tests to Requirements

**1. Generate tests**
```bash
test-genie generate my-scenario --types unit
```

**2. Review and identify which requirement each test validates**
```typescript
// This test validates MY-PROJECT-CREATE
describe('projectStore [REQ:MY-PROJECT-CREATE]', () => {
  it('creates project', () => { ... });
});

// This test validates MY-USER-AUTH
describe('authService [REQ:MY-USER-AUTH]', () => {
  it('authenticates user', () => { ... });
});
```

**3. Run tests to populate phase results**
```bash
./test/phases/test-unit.sh
```

**4. Auto-sync updates requirements**
```bash
# Automatically happens after test run, or manually:
node scripts/requirements/report.js --scenario my-scenario --mode sync
```

**5. Verify tracking**
```bash
vrooli scenario requirements report my-scenario
# Shows: MY-PROJECT-CREATE (1 validation: implemented ✅)
```

### Bulk Tagging Strategy

When generating many tests, use this workflow:

```bash
# 1. Generate all tests
test-genie generate my-scenario --comprehensive

# 2. Create mapping file
cat > test-req-mapping.txt <<EOF
api/handlers/projects_test.go:TestCreateProject:MY-PROJECT-CREATE
api/handlers/workflows_test.go:TestExecuteWorkflow:MY-WORKFLOW-EXECUTE
ui/src/stores/__tests__/authStore.test.ts:authStore:MY-USER-AUTH
EOF

# 3. Use script to bulk-add tags (example script)
while IFS=: read -r file test req; do
  # Add [REQ:$req] to test name
  sed -i "s/\(${test}\)/\1 [REQ:${req}]/" "$file"
done < test-req-mapping.txt

# 4. Verify and commit
git diff  # Review changes
git add .
git commit -m "Add generated tests with requirement tracking"
```

## Best Practices

### 1. Generate, Don't Copy-Paste

```bash
# ✅ GOOD: Generate fresh for each scenario
test-genie generate scenario-a --types unit
test-genie generate scenario-b --types unit

# ❌ BAD: Copy tests between scenarios
# Tests should match specific codebase structure
```

### 2. Review Before Committing

```bash
# ✅ GOOD: Review workflow
test-genie generate my-scenario --types unit
git diff  # Review generated tests
# Adjust assertions, add tags
git commit

# ❌ BAD: Blind commit
test-genie generate my-scenario --types unit
git add . && git commit -m "add tests"  # No review!
```

### 3. Iterate on Generation

```bash
# ✅ GOOD: Incremental approach
test-genie generate my-scenario --types unit --files "api/handlers/projects.go"
# Review, adjust
test-genie generate my-scenario --types unit --files "api/services/auth.go"
# Review, adjust

# ❌ BAD: Generate everything at once
test-genie generate my-scenario --comprehensive
# Overwhelming to review 100+ tests
```

### 4. Use as Starting Point

```go
// Generated (starting point)
func TestCreateProject(t *testing.T) {
    result, err := CreateProject("Test")
    assert.NoError(t, err)
    assert.NotNil(t, result)
}

// Improved (after review)
func TestCreateProject(t *testing.T) {
    t.Run("creates project with valid name [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        mockDB := &MockDB{}
        service := NewProjectService(mockDB)

        result, err := service.CreateProject(CreateProjectInput{
            Name: "Test Project",
            OwnerID: "user-123",
        })

        assert.NoError(t, err)
        assert.NotNil(t, result)
        assert.Equal(t, "Test Project", result.Name)
        assert.Equal(t, "user-123", result.OwnerID)
        assert.NotEmpty(t, result.ID)

        // Verify DB interaction
        assert.Equal(t, 1, len(mockDB.Projects))
    })
}
```

### 5. Combine with Manual Tests

```bash
# Use test-genie for coverage baseline
test-genie generate my-scenario --types unit

# Then manually add:
# - Business logic tests (domain-specific)
# - BAS workflows (UI automation)
# - Integration tests (real service interactions)
```

## Limitations

### Current Limitations (as of 2025-10-03)

1. **Internal test coverage: 4.8%** (targeting 95%)
   - test-genie itself needs more tests
   - Generated test quality may vary

2. **P1 features pending**:
   - Visual coverage analysis (HTML reports with gap highlights)
   - Performance regression detection
   - Cross-scenario integration test generation

3. **AI model dependent**:
   - Quality varies based on underlying AI model
   - May hallucinate non-existent functions
   - May misunderstand complex business logic

4. **No UI test generation**:
   - Cannot generate BAS workflows (use manual authoring)
   - Cannot generate Playwright tests (discouraged anyway)

### Known Issues

**Issue 1: Hallucinated imports**
```typescript
// Generated code may import non-existent modules
import { nonExistentHelper } from '../utils/fake';  // ❌ Doesn't exist

// Solution: Review imports carefully
```

**Issue 2: Incorrect mock behavior**
```go
// Generated mock may not match real behavior
mockDB.GetByID = func(id string) (*Project, error) {
    return &Project{ID: "hardcoded"}, nil  // ❌ Ignores input ID
}

// Solution: Verify mocks match real implementation
```

**Issue 3: Missing edge cases**
```typescript
// Generated only tests happy path
it('creates project', () => { ... });  // ✅ Success case

// Missing:
it('rejects empty project name', () => { ... });  // ❌ Not generated
it('handles database errors', () => { ... });      // ❌ Not generated
```

## See Also

### Related Guides
- **[Quick Start](quick-start.md)** - Basic testing workflow
- **[Requirement Tracking](requirement-tracking.md)** - Tag tests with `[REQ:ID]`
- **[Scenario Unit Testing](scenario-unit-testing.md)** - Writing manual unit tests
- **[Writing Testable UIs](writing-testable-uis.md)** - Design UIs for automation (not generated)

### Reference Implementation
- **[test-genie scenario](../../../scenarios/test-genie/)** - Source code and internals
- **[test-genie PRD](../../../scenarios/test-genie/PRD.md)** - Feature documentation

### Alternative Approaches
- **[BAS Workflows](ui-automation-with-bas.md)** - For UI testing (preferred over generated Playwright)
- **Manual test authoring** - For complex business logic

---

**Remember**: test-genie is a productivity tool, not a replacement for human judgment. Always review generated tests before committing.
