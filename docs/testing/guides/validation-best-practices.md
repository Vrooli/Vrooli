# Validation Best Practices

**Status**: Active
**Last Updated**: 2025-11-22
**Audience**: Scenario developers, AI agents (ecosystem-manager)

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
│   ├── phases/          ❌ NEVER reference these (orchestration)
│   │   ├── test-unit.sh
│   │   └── test-integration.sh
│   ├── cli/             ❌ NEVER reference these (CLI wrapper tests)
│   │   └── *.bats
│   ├── unit/            ❌ NEVER reference these (test runners)
│   ├── integration/     ❌ NEVER reference these (test runners)
│   └── playbooks/       ✅ ALWAYS reference these (e2e automation)
│       └── **/*.json
│
├── api/
│   ├── **/*_test.go     ✅ ALWAYS reference these (API unit tests)
│   └── **/tests/**      ✅ ALWAYS reference these (API test directories)
│
└── ui/src/
    └── **/*.test.tsx    ✅ ALWAYS reference these (UI unit tests)
```

### Why test/phases/ is Rejected

**test/phases/** scripts are **orchestrators**, not test sources:

```bash
# test/phases/test-unit.sh
#!/bin/bash
# This script RUNS tests, it doesn't CONTAIN tests

# Runs Go tests
go test ./api/...

# Runs Vitest tests
npm run test

# ❌ Referencing this provides ZERO traceability to actual test code
# ✅ Instead, reference the actual test files being executed
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

**Problem**: The CLI should be a thin wrapper over the API. If your CLI test passes but the API logic is broken, the requirement isn't truly validated.

**Solution**: Reference API tests AND e2e automation:
```json
{
  "id": "PROFILE-CREATE",
  "validation": [
    {"type": "test", "ref": "api/profile_service_test.go"},  // ← Tests business logic
    {"type": "automation", "ref": "test/playbooks/profiles/create.json"}  // ← Tests e2e flow
  ]
}
```

---

## Multi-Layer Validation Strategy

### Component-Aware Requirements

Validation requirements adapt based on your scenario's components:

```bash
# Detect components
ls api/*.go &> /dev/null && echo "Has API component"
ls ui/package.json &> /dev/null && echo "Has UI component"
```

### Validation Requirements by Criticality

| Criticality | Layers Required | Example Combinations |
|-------------|-----------------|----------------------|
| **P0/P1** (Critical) | ≥2 AUTOMATED layers | API + UI, API + E2E, UI + E2E |
| **P2** (Optional) | ≥1 AUTOMATED layer | API alone, UI alone, E2E alone |

**Important**: Manual validations don't count toward diversity.

### Full-Stack Scenario Example

**Scenario has**: API (Go) + UI (React)
**Applicable layers**: API, UI, E2E

```json
{
  "id": "USER-AUTH-LOGIN",
  "prd_ref": "OT-P0-003",  // ← P0 = needs ≥2 automated layers
  "status": "complete",
  "validation": [
    // Layer 1: API unit tests
    {
      "type": "test",
      "ref": "api/auth/login_test.go",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests JWT generation, password hashing, rate limiting"
    },
    // Layer 2: UI unit tests
    {
      "type": "test",
      "ref": "ui/src/components/LoginForm.test.tsx",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests form validation, error handling, success flow"
    },
    // Layer 3: E2E automation (bonus)
    {
      "type": "automation",
      "ref": "test/playbooks/auth/login-flow.json",
      "phase": "integration",
      "status": "implemented",
      "notes": "Full user journey from login page to dashboard"
    }
  ]
}

// ✅ Passes: 3 automated layers (API + UI + E2E)
// ✅ Comprehensive coverage across all components
```

### API-Only Scenario Example

**Scenario has**: API (Go) only
**Applicable layers**: API, E2E

```json
{
  "id": "DEPENDENCY-ANALYSIS",
  "prd_ref": "OT-P0-001",  // ← P0 = needs ≥2 automated layers
  "status": "complete",
  "validation": [
    // Layer 1: API unit tests
    {
      "type": "test",
      "ref": "api/analyzer_test.go",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests graph resolution, cycle detection, conflict handling"
    },
    // Layer 2: E2E automation (BAS can test API endpoints)
    {
      "type": "automation",
      "ref": "test/playbooks/analyze-dependencies.json",
      "phase": "integration",
      "status": "implemented",
      "notes": "Tests API endpoints via HTTP requests, validates responses"
    }
  ]
}

// ✅ Passes: 2 automated layers (API + E2E) for API-only scenario
// Note: UI tests would be ignored (no UI component exists)
```

### UI-Only Scenario Example

**Scenario has**: UI (React) only
**Applicable layers**: UI, E2E

```json
{
  "id": "THEME-SWITCHER",
  "prd_ref": "OT-P1-005",  // ← P1 = needs ≥2 automated layers
  "status": "complete",
  "validation": [
    // Layer 1: UI unit tests
    {
      "type": "test",
      "ref": "ui/src/components/ThemeSwitcher.test.tsx",
      "phase": "unit",
      "status": "implemented",
      "notes": "Tests toggle, persistence to localStorage, CSS variable updates"
    },
    // Layer 2: E2E automation
    {
      "type": "automation",
      "ref": "test/playbooks/ui/theme-toggle.json",
      "phase": "integration",
      "status": "implemented",
      "notes": "Verifies visual theme changes, persistence across page reloads"
    }
  ]
}

// ✅ Passes: 2 automated layers (UI + E2E) for UI-only scenario
// Note: API tests would be ignored (no API component exists)
```

---

## Quality Over Quantity

### The Problem: Superficial Tests

**Gaming pattern**: Create empty or trivial test files to satisfy requirements.

❌ **Bad Example - Placeholder Test**:
```go
// api/placeholder_test.go (5 lines, no logic)
package api

import "testing"

// Empty file, counts as "API layer" but validates nothing
```

```json
// Requirement validation
{
  "id": "WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/placeholder_test.go"},  // ← Superficial
    {"type": "test", "ref": "ui/src/empty.test.tsx"}     // ← Superficial
  ]
}
// ❌ Appears to have 2 layers, but tests validate nothing
```

### Quality Indicators

Test files are automatically analyzed for quality. **Files must score ≥4/5** to count:

| Indicator | Threshold | Score |
|-----------|-----------|-------|
| **Lines of Code** | ≥20 LOC (non-comment) | 1 point |
| **Assertions** | ≥1 assertion | 1 point |
| **Test Functions** | ≥1 test case | 1 point |
| **Multiple Test Cases** | ≥3 test functions | 1 point |
| **Assertion Density** | ≥0.1 (1 per 10 LOC) | 1 point |

**Quality Score**: Need ≥4/5 to be meaningful.

### Good Example: Meaningful Test

✅ **Proper Test Coverage**:
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
    workflow, err := service.Create(context.Background(), &CreateWorkflowRequest{
        Name: "Test Workflow",
        Nodes: []Node{{ID: "start", Type: "navigate"}},
        Edges: []Edge{{From: "start", To: "end"}},
    })
    require.NoError(t, err)
    assert.NotEmpty(t, workflow.ID)
    assert.Equal(t, "Test Workflow", workflow.Name)

    // Test Case 2: Duplicate name handling
    _, err = service.Create(context.Background(), &CreateWorkflowRequest{
        Name: "Test Workflow",  // Same name
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "already exists")

    // Test Case 3: Invalid nodes (cycle detection)
    _, err = service.Create(context.Background(), &CreateWorkflowRequest{
        Name: "Cyclic",
        Nodes: []Node{{ID: "a"}},
        Edges: []Edge{{From: "a", To: "a"}},  // Self-loop
    })
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cycle detected")
}

func TestWorkflowService_Update(t *testing.T) {
    // ... 4 more test cases covering update scenarios ...
}

func TestWorkflowService_Delete(t *testing.T) {
    // ... 3 test cases for deletion ...
}

// ... 5 more test functions covering edge cases ...
```

**Quality score**: 5/5
- ✅ 187 LOC (non-comment)
- ✅ 42 assertions
- ✅ 12 test functions
- ✅ Assertion density: 0.22 (great coverage)

---

## Manual Validations: When and How

### When Manual Validations Are Acceptable

✅ **Good uses**:
1. **Temporary bridge** for pending/in_progress requirements before automation
2. **Supplementary evidence** alongside automated tests
3. **Quarterly audits** that genuinely can't be automated (security pen tests)

❌ **Bad uses**:
1. **Complete requirements** with ONLY manual validation
2. **>10% of all validations** being manual (automation deficit)
3. **Lazy validation** ("I manually tested it once")

### Manual Validation Example

```json
{
  "id": "SECURITY-PENTEST",
  "prd_ref": "OT-P1-010",
  "status": "in_progress",  // ← Temporary status
  "validation": [
    {
      "type": "manual",
      "ref": "docs/security/penetration-test-checklist.md",
      "phase": "integration",
      "status": "planned",
      "notes": "Quarterly security audit by external team (expires 30 days after last run)"
    },
    // TODO: Add automated security scans when tooling is available
  ]
}
```

**Log manual validation execution**:
```bash
vrooli scenario requirements manual-log my-scenario SECURITY-PENTEST \
  --status passed \
  --notes "Q4 2025 penetration test completed by SecureTeam, no critical findings"
```

### Why Manual Validations Don't Count Toward Diversity

Manual validations are **excluded** from diversity requirements because:
1. They don't provide continuous validation (run once, forget about it)
2. They can be gamed ("mark as manually verified" without real testing)
3. The goal is **automated** multi-layer coverage for CI/CD

**Manual validations are tracked separately** and surface warnings if:
- >10% of all validations are manual
- ≥5 complete requirements have ONLY manual validation (no automated tests)

---

## Anti-Patterns to Avoid

### 1. Monolithic Test Files

❌ **Bad**: One test file validates 6+ requirements

```json
// test/cli/everything.bats validates REQ-001 through REQ-037
{
  "id": "REQ-001",
  "validation": [{"ref": "test/cli/everything.bats"}]
},
{
  "id": "REQ-002",
  "validation": [{"ref": "test/cli/everything.bats"}]
},
// ... 35 more requirements, same file
```

✅ **Good**: Focused test files per requirement

```json
{
  "id": "PROFILE-CREATE",
  "validation": [
    {"ref": "api/profile_service_test.go"},  // TestProfileCreate()
    {"ref": "test/playbooks/profiles/create.json"}
  ]
},
{
  "id": "PROFILE-UPDATE",
  "validation": [
    {"ref": "api/profile_service_test.go"},  // TestProfileUpdate()
    {"ref": "test/playbooks/profiles/update.json"}
  ]
}
```

### 2. Perfect 1:1 Test-to-Requirement Ratio

❌ **Bad**: Exactly 37 tests for 37 requirements (suspicious)

**Why it's bad**: Real test coverage requires multiple test types per requirement (API, UI, e2e). A perfect 1:1 ratio suggests single-layer validation.

✅ **Good**: 1.5-2.0x test-to-requirement ratio with diverse sources
- 37 requirements → 55-74 tests
- Mix of API unit tests, UI unit tests, e2e automation

### 3. Excessive 1:1 Operational Target Mapping

❌ **Bad**: 37 operational targets for 37 requirements

```json
{"id": "REQ-001", "prd_ref": "OT-P0-001"},
{"id": "REQ-002", "prd_ref": "OT-P0-002"},
// ... 37 unique OT-XXX targets
```

**Why it's bad**: Operational targets represent PRD goals, not individual requirements. This inflates target completion metrics.

✅ **Good**: 5-10 operational targets grouping related requirements

```json
{"id": "PROFILE-CREATE", "prd_ref": "OT-P0-001"},  // Profile Management
{"id": "PROFILE-UPDATE", "prd_ref": "OT-P0-001"},  // Same OT
{"id": "PROFILE-DELETE", "prd_ref": "OT-P0-001"},  // Same OT
{"id": "DEPLOY-VALIDATE", "prd_ref": "OT-P0-002"}, // Deployment Flow
{"id": "DEPLOY-EXECUTE", "prd_ref": "OT-P0-002"},  // Same OT
// 5 OT targets group 37 requirements by PRD goals
```

---

## Checking Your Setup

### Completeness Score

```bash
vrooli scenario completeness <scenario-name>
```

**Interpret warnings**:

| Warning | Severity | What to Fix |
|---------|----------|-------------|
| "X requirements lack multi-layer AUTOMATED validation" | 🔴 High | Add API/UI/E2E tests for P0/P1 requirements |
| "X requirements reference unsupported test/ directories" | 🔴 High | Move refs from test/cli/ to actual test sources |
| "X test files validate ≥4 requirements each" | 🟡 Medium | Break monolithic test files into focused tests |
| "X test files appear superficial" | 🟡 Medium | Add assertions, edge cases, meaningful logic |
| "1:1 test-to-requirement ratio" | 🟡 Medium | Add more tests across different layers |
| "100% 1:1 operational target mapping" | 🔴 High | Group requirements under shared OT-XXX targets |

### Fixing Common Issues

#### Issue: "Requirements lack multi-layer validation"

**Before**:
```json
{
  "id": "WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",  // P0 needs ≥2 layers
  "validation": [
    {"type": "test", "ref": "api/workflow_test.go"}  // ← Only 1 layer
  ]
}
```

**After**:
```json
{
  "id": "WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/workflow_test.go"},  // ← API layer
    {"type": "test", "ref": "ui/src/stores/workflowStore.test.ts"},  // ← UI layer
    {"type": "automation", "ref": "test/playbooks/workflows/crud.json"}  // ← E2E layer (bonus)
  ]
}
```

#### Issue: "Unsupported test/ directories"

**Before**:
```json
{
  "validation": [
    {"type": "test", "ref": "test/cli/profile-ops.bats"},  // ← Invalid
    {"type": "test", "ref": "test/phases/test-unit.sh"}    // ← Invalid
  ]
}
```

**After**:
```json
{
  "validation": [
    {"type": "test", "ref": "api/profile_service_test.go"},  // ← Valid API test
    {"type": "automation", "ref": "test/playbooks/profiles/create.json"}  // ← Valid E2E
  ]
}
```

---

## Reference Implementation: browser-automation-studio

The `browser-automation-studio` scenario demonstrates proper validation setup:

**Strengths**:
1. ✅ Diverse validation sources (API Go tests, UI Vitest tests, BAS playbooks)
2. ✅ Multiple validations per requirement (up to 6 for critical requirements)
3. ✅ Proper operational target grouping (7 targets for 63 requirements)
4. ✅ Test-to-requirement ratio of 1.43:1 (90 tests for 63 requirements)
5. ✅ No test/phases/ or test/cli/ refs (all proper test sources)

**Example requirement**:
```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/automation/compiler/compiler_test.go"},
    {"type": "test", "ref": "api/browserless/runtime/session_test.go"},
    {"type": "test", "ref": "api/database/repository_test.go"},
    {"type": "test", "ref": "ui/src/stores/projectStore.test.ts"},
    {"type": "test", "ref": "ui/src/stores/workflowStore.test.ts"},
    {"type": "automation", "ref": "test/playbooks/capabilities/02-builder/demo-sanity.json"}
  ]
}
// ✅ 6 validations across 3 layers (API, UI, E2E)
```

**Browse the reference**:
- [BAS Requirements Registry](/scenarios/browser-automation-studio/requirements/)
- [BAS Test Structure](/scenarios/browser-automation-studio/test/)

---

## Summary: The Validation Checklist

When creating requirement validations, ensure:

- [ ] **Refs point to actual test sources** (api/, ui/src/, test/playbooks/), not orchestration (test/phases/, test/cli/)
- [ ] **P0/P1 requirements have ≥2 AUTOMATED layers** from applicable components
- [ ] **Test files are meaningful** (≥20 LOC, assertions, multiple test cases)
- [ ] **No monolithic test files** (avoid 1 file validating 6+ requirements)
- [ ] **Manual validations are temporary** (use for pending work, replace with automation)
- [ ] **Operational targets group requirements** (5-10 OT targets, not 1:1 mapping)
- [ ] **Test-to-requirement ratio is 1.5-2.0x** (diverse test sources)

**Run validation check**:
```bash
vrooli scenario completeness <scenario-name>
```

Fix warnings before marking requirements as complete!

---

## See Also

- [Requirement Schema Reference](../reference/requirement-schema.md) - Schema details and field definitions
- [Gaming Prevention Guide](../reference/gaming-prevention.md) - How the system detects gaming patterns
- [Phased Testing Architecture](../architecture/PHASED_TESTING.md) - Test phase orchestration
- [UI Automation with BAS](./ui-automation-with-bas.md) - E2E automation guide
