# Validation Best Practices

**Status**: Active
**Last Updated**: 2025-12-02
**Audience**: Scenario developers, AI agents

---

## Overview

This guide provides best practices for creating high-quality requirement validations that ensure true implementation quality rather than superficial checkmarks. Proper validation prevents "gaming the system" while building robust, maintainable test coverage.

**Key Principle**: Quality over quantity. One meaningful test is worth more than ten superficial placeholders.

---

## Valid vs Invalid Validation Sources

### The Golden Rule

**Validation refs must point to ACTUAL TEST SOURCES, not test orchestration infrastructure.**

### Directory Structure Reference

```
scenario/
├── test/
│   ├── phases/          ❌ NEVER reference (orchestration)
│   │   ├── test-unit.sh
│   │   └── test-integration.sh
│   ├── cli/             ❌ NEVER reference (CLI wrapper tests)
│   │   └── *.bats
│   └── playbooks/       ✅ ALWAYS reference (e2e automation)
│       └── **/*.json
│
├── api/
│   ├── **/*_test.go     ✅ ALWAYS reference (API unit tests)
│   └── **/tests/**      ✅ ALWAYS reference (API test directories)
│
└── ui/src/
    └── **/*.test.tsx    ✅ ALWAYS reference (UI unit tests)
```

### Why test/phases/ is Rejected

**test/phases/** scripts are **orchestrators**, not test sources:

```bash
# test/phases/test-unit.sh
# This script RUNS tests, it doesn't CONTAIN tests

go test ./api/...      # Runs Go tests
npm run test           # Runs Vitest tests

# Referencing this provides ZERO traceability to actual test code
```

**Proper references**:
- `api/handlers/projects_test.go` (specific Go test file)
- `ui/src/components/ProjectModal.test.tsx` (specific Vitest test)
- `test/playbooks/projects/create.json` (specific BAS workflow)

### Why test/cli/ is Rejected

**test/cli/** tests validate the CLI wrapper, not business logic:

```bash
# test/cli/profile-operations.bats - Tests CLI interface
@test "vrooli profile create accepts --name flag" {
  run vrooli profile create --name test
  [ "$status" -eq 0 ]
}
```

The CLI should be a thin wrapper over the API. If CLI tests pass but API logic is broken, the requirement isn't truly validated.

**Solution**: Reference API tests AND e2e automation:
```json
{
  "id": "PROFILE-CREATE",
  "validation": [
    {"type": "test", "ref": "api/profile_service_test.go"},
    {"type": "automation", "ref": "test/playbooks/profiles/create.json"}
  ]
}
```

---

## Multi-Layer Validation Strategy

### Component-Aware Requirements

Validation requirements adapt based on your scenario's components:

| Component | Layer 1 | Layer 2 | Layer 3 |
|-----------|---------|---------|---------|
| API | Go unit tests | - | E2E |
| UI | - | Vitest tests | E2E |
| Full-stack | Go tests | Vitest tests | E2E |

### Validation Requirements by Criticality

| Criticality | Layers Required | Example Combinations |
|-------------|-----------------|----------------------|
| **P0/P1** (Critical) | >= 2 AUTOMATED layers | API + UI, API + E2E, UI + E2E |
| **P2** (Optional) | >= 1 AUTOMATED layer | API alone, UI alone, E2E alone |

**Important**: Manual validations don't count toward diversity.

### Full-Stack Scenario Example

```json
{
  "id": "USER-AUTH-LOGIN",
  "prd_ref": "OT-P0-003",
  "status": "complete",
  "validation": [
    {
      "type": "test",
      "ref": "api/auth/login_test.go",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests JWT generation, password hashing, rate limiting"
    },
    {
      "type": "test",
      "ref": "ui/src/components/LoginForm.test.tsx",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests form validation, error handling, success flow"
    },
    {
      "type": "automation",
      "ref": "test/playbooks/auth/login-flow.json",
      "phase": "integration",
      "status": "implemented",
      "notes": "Full user journey from login page to dashboard"
    }
  ]
}
```

### API-Only Scenario Example

```json
{
  "id": "DEPENDENCY-ANALYSIS",
  "prd_ref": "OT-P0-001",
  "status": "complete",
  "validation": [
    {
      "type": "test",
      "ref": "api/analyzer_test.go",
      "phase": "unit",
      "status": "implemented"
    },
    {
      "type": "automation",
      "ref": "test/playbooks/analyze-dependencies.json",
      "phase": "integration",
      "status": "implemented"
    }
  ]
}
```

---

## Quality Over Quantity

### The Problem: Superficial Tests

**Gaming pattern**: Create empty or trivial test files to satisfy requirements.

**Bad Example**:
```go
// api/placeholder_test.go (5 lines, no logic)
package api
import "testing"
// Empty file, counts as "API layer" but validates nothing
```

### Quality Indicators

Test files are automatically analyzed for quality. **Files must score >= 4/5** to count:

| Indicator | Threshold | Score |
|-----------|-----------|-------|
| Lines of Code | >= 20 LOC (non-comment) | 1 point |
| Assertions | >= 1 assertion | 1 point |
| Test Functions | >= 1 test case | 1 point |
| Multiple Test Cases | >= 3 test functions | 1 point |
| Assertion Density | >= 0.1 (1 per 10 LOC) | 1 point |

### Good Example: Meaningful Test

```go
// api/workflow_service_test.go (187 lines, quality score: 5/5)
package api

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkflowService_Create(t *testing.T) {
    service := NewWorkflowService(testDB)

    // Test Case 1: Valid workflow creation
    workflow, err := service.Create(ctx, &CreateWorkflowRequest{
        Name: "Test Workflow",
        Nodes: []Node{{ID: "start", Type: "navigate"}},
    })
    require.NoError(t, err)
    assert.NotEmpty(t, workflow.ID)

    // Test Case 2: Duplicate name handling
    _, err = service.Create(ctx, &CreateWorkflowRequest{
        Name: "Test Workflow",
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "already exists")

    // Test Case 3: Invalid nodes (cycle detection)
    _, err = service.Create(ctx, &CreateWorkflowRequest{
        Name: "Cyclic",
        Edges: []Edge{{From: "a", To: "a"}},
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cycle detected")
}
```

---

## Manual Validations: When and How

### When Manual Validations Are Acceptable

**Good uses**:
1. **Temporary bridge** for pending/in_progress requirements before automation
2. **Supplementary evidence** alongside automated tests
3. **Quarterly audits** that genuinely can't be automated (security pen tests)

**Bad uses**:
1. **Complete requirements** with ONLY manual validation
2. **> 10% of all validations** being manual
3. **Lazy validation** ("I manually tested it once")

### Manual Validation Example

```json
{
  "id": "SECURITY-PENTEST",
  "prd_ref": "OT-P1-010",
  "status": "in_progress",
  "validation": [
    {
      "type": "manual",
      "ref": "docs/security/penetration-test-checklist.md",
      "phase": "integration",
      "status": "planned",
      "notes": "Quarterly security audit (expires 30 days after last run)"
    }
  ]
}
```

**Log manual validation execution**:
```bash
vrooli scenario requirements manual-log my-scenario SECURITY-PENTEST \
  --status passed \
  --notes "Q4 2025 penetration test completed"
```

### Why Manual Validations Don't Count Toward Diversity

Manual validations are **excluded** because:
1. They don't provide continuous validation
2. They can be gamed ("manually verified" without real testing)
3. The goal is **automated** multi-layer coverage for CI/CD

---

## Anti-Patterns to Avoid

### 1. Monolithic Test Files

**Bad**: One test file validates 6+ requirements

**Good**: Focused test files per requirement

### 2. Perfect 1:1 Test-to-Requirement Ratio

**Bad**: Exactly 37 tests for 37 requirements (suspicious)

**Good**: 1.5-2.0x test-to-requirement ratio with diverse sources

### 3. Excessive 1:1 Operational Target Mapping

**Bad**: 37 operational targets for 37 requirements

**Good**: 5-10 operational targets grouping related requirements

---

## Checking Your Setup

### Completeness Score

```bash
vrooli scenario completeness <scenario-name>
```

**Interpret warnings**:

| Warning | Severity | What to Fix |
|---------|----------|-------------|
| "X requirements lack multi-layer validation" | High | Add API/UI/E2E tests |
| "X requirements reference unsupported directories" | High | Move refs to actual test sources |
| "X test files validate >= 4 requirements each" | Medium | Break into focused tests |
| "X test files appear superficial" | Medium | Add assertions and edge cases |
| "100% 1:1 operational target mapping" | High | Group requirements under shared OTs |

---

## Summary: The Validation Checklist

When creating requirement validations, ensure:

- [ ] **Refs point to actual test sources** (api/, ui/src/, test/playbooks/)
- [ ] **P0/P1 requirements have >= 2 AUTOMATED layers**
- [ ] **Test files are meaningful** (>= 20 LOC, assertions, multiple test cases)
- [ ] **No monolithic test files** (avoid 1 file validating 6+ requirements)
- [ ] **Manual validations are temporary** (replace with automation)
- [ ] **Operational targets group requirements** (5-10 OT targets)
- [ ] **Test-to-requirement ratio is 1.5-2.0x**

**Run validation check**:
```bash
vrooli scenario completeness <scenario-name>
```

Fix warnings before marking requirements as complete!

---

## See Also

- [Requirement Schema Reference](../reference/requirement-schema.md) - Schema details
- [Gaming Prevention Reference](../reference/gaming-prevention.md) - How gaming is detected
- [Phased Testing](phased-testing.md) - Test phase orchestration
- [UI Automation with BAS](ui-automation-with-bas.md) - E2E automation guide
- [Scenario Unit Testing](scenario-unit-testing.md) - Writing quality tests
