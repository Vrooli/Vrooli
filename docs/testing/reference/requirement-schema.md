# 📐 Requirement Registry Schema Reference

Human-readable reference for the Vrooli requirement registry JSON schema. For the canonical schema definition, see [scripts/requirements/schema.json](/scripts/requirements/schema.json).

## Overview

The requirement registry is a JSON file (or collection of modular JSON files) that defines:
- **Requirements**: What needs to be built/validated
- **Validations**: How requirements are proven (tests, automation, manual steps)
- **Hierarchy**: Parent-child relationships and dependencies
- **Status**: Implementation and test progress

**Schema Version**: 1.0.0

## File Structure

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `_metadata` | object | No | File-level metadata (description, sync settings) |
| `meta` | object | No | Legacy metadata from YAML format |
| `imports` | array | No | Child requirement files (index.json only) |
| `reporting` | object | No | Reporting configuration (index.json only) |
| `requirements` | array | **Yes** | Array of requirement definitions |

### _metadata Object

Controls auto-sync behavior and file documentation.

```json
{
  "_metadata": {
    "description": "Human-readable description of this requirements module",
    "module": "projects.ui",
    "last_validated_at": "2025-11-05T00:00:00.000Z",
    "auto_sync_enabled": true,
    "schema_version": "1.0.0"
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `description` | string | - | What requirements this file contains |
| `module` | string | - | Module name (e.g., `projects.ui`, `execution.telemetry`) |
| `last_validated_at` | ISO 8601 | - | When requirements were last manually reviewed |
| `auto_sync_enabled` | boolean | `true` | Whether auto-sync should update this file |
| `schema_version` | string | `"1.0.0"` | Schema version this file conforms to |

### imports Array (Index Files Only)

Declares child requirement files to import.

```json
{
  "imports": [
    "projects/dialog.json",
    "workflow-builder/core.json",
    "execution/telemetry.json"
  ]
}
```

**Pattern**: `[a-zA-Z0-9_/-]+\.json`

**Resolution**: Relative to `requirements/` directory

## Requirement Definition

### Required Fields

```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "title": "Workflows persist nodes, edges, and metadata",
  "status": "complete",
  "prd_ref": "OT-P0-001"
}
```

| Field | Type | Pattern/Enum | Description |
|-------|------|--------------|-------------|
| `id` | string | `[A-Z][A-Z0-9]+-[A-Z0-9-]+` | Unique requirement identifier |
| `title` | string | min 1 char | Short description (1-2 sentences) |
| `status` | enum | See [Status Values](#status-values) | Implementation progress |
| `prd_ref` | string | - | Reference to PRD (see [PRD Reference Format](#prd-reference-format)) |

#### ID Pattern Examples

✅ **Valid:**
- `BAS-WORKFLOW-PERSIST-CRUD`
- `APP-FUNC-001`
- `MY-SCENARIO-FEATURE-NAME`

❌ **Invalid:**
- `bas-workflow-crud` (lowercase)
- `WORKFLOW_CRUD` (no prefix)
- `BAS_WORKFLOW` (underscore instead of hyphen)
- `1BAS-WORKFLOW` (starts with digit)

#### Status Values

| Value | Meaning | When to Use |
|-------|---------|-------------|
| `pending` | Not yet planned | Requirement identified but not scheduled |
| `planned` | Scheduled, approach defined | Validation approach documented, no code yet |
| `in_progress` | Work started | At least one validation exists (test written) |
| `complete` | Fully implemented | All validations passing (auto-updated by sync) |
| `not_implemented` | Deprioritized | Requirement deferred or cancelled |

#### PRD Reference Format

The `prd_ref` field can use two formats:

**1. Operational Target Format** (Recommended for new scenarios):
```
"prd_ref": "OT-P0-001"
```
- Pattern: `OT-P[012]-NNN` where P0/P1/P2 indicates priority
- Criticality is automatically derived: `OT-P0-*` → P0, `OT-P1-*` → P1, `OT-P2-*` → P2
- Links directly to operational targets in PRD

**2. Freeform PRD Section** (Legacy format):
```
"prd_ref": "Functional Requirements > Must Have > Visual workflow builder"
```
- Any descriptive text pointing to PRD section
- Criticality must be manually managed (not auto-derived)
- More flexible but less structured

**Note**: The `criticality` field is no longer stored in requirement files. For operational target format, it is computed from `prd_ref` when requirements are loaded.

### Optional Fields

```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "title": "Workflows persist nodes, edges, and metadata",
  "status": "complete",
  "prd_ref": "OT-P0-001",

  "category": "workflow.builder",
  "description": "Validates compiler, database, and API layers so workflows round-trip",
  "tags": ["storage", "persistence", "crud"],
  "children": ["BAS-PROJECT-CREATE", "BAS-WORKFLOW-SAVE"],
  "depends_on": ["BAS-DATABASE-SETUP"],
  "blocks": ["BAS-WORKFLOW-SHARE"],
  "validation": [...]
}
```

**Note**: Criticality (P0) is derived from `prd_ref` (OT-P0-001).

| Field | Type | Description |
|-------|------|-------------|
| `category` | string | Hierarchical grouping (e.g., `workflow.builder`, `projects.ui`) |
| `description` | string | Detailed explanation of what this requirement validates |
| `tags` | array of strings | Custom tags for filtering and categorization |
| `children` | array of strings | Child requirement IDs for hierarchical aggregation |
| `depends_on` | array of strings | Requirement IDs this depends on (not yet enforced) |
| `blocks` | array of strings | Requirement IDs blocked by this one (not yet enforced) |
| `validation` | array of objects | Validation methods (see [Validation Definition](#validation-definition)) |

## Validation Definition

Each requirement can have multiple validation entries proving it's implemented correctly.

### Required Fields

```json
{
  "type": "test",
  "phase": "unit",
  "status": "implemented"
}
```

| Field | Type | Enum | Description |
|-------|------|------|-------------|
| `type` | string | `test`, `automation`, `manual` | Validation method |
| `phase` | string | `unit`, `integration`, `business`, `performance`, `structure`, `dependencies` | Test phase |
| `status` | string | `not_implemented`, `planned`, `implemented`, `failing` | Current state |

### Optional Fields

```json
{
  "type": "test",
  "ref": "ui/src/stores/__tests__/projectStore.test.ts",
  "phase": "unit",
  "status": "implemented",
  "notes": "Auto-added from vitest evidence",
  "scenario": "browser-automation-studio",
  "folder": "/Testing Harness",
  "workflow_id": "WORKFLOW-123456",
  "metadata": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ref` | string | File path reference (test file, automation workflow, etc.) |
| `workflow_id` | string | Workflow ID for automation validations (alternative to `ref`) |
| `notes` | string | Human-readable notes about this validation |
| `scenario` | string | External scenario name (for cross-scenario validations) |
| `folder` | string | Folder path (for BAS automation workflows) |
| `metadata` | object | Additional validation-specific metadata |

### Validation Types

#### Type: test

**Purpose**: Unit, integration, business, or performance tests

**Common patterns:**
```json
// Go test
{
  "type": "test",
  "ref": "api/handlers/projects_test.go",
  "phase": "unit",
  "status": "implemented",
  "notes": "Covers CRUD operations with mocked database"
}

// Vitest test
{
  "type": "test",
  "ref": "ui/src/components/__tests__/ProjectModal.test.tsx",
  "phase": "unit",
  "status": "implemented",
  "notes": "Auto-added from vitest evidence"
}

// CLI BATS integration test
{
  "type": "test",
  "ref": "test/cli/profile-operations.bats",
  "phase": "integration",
  "status": "implemented",
  "notes": "BATS tests with [REQ:ID] tags"
}
```

**Validation Specificity Requirements:**

**CRITICAL**: The `ref` field MUST point to **specific test files**, not generic phase orchestration scripts.

✅ **ALLOWED:**
```json
// Go unit/integration tests
{"ref": "api/handlers/projects_test.go"}

// Vitest unit tests
{"ref": "ui/src/**/*.test.{ts,tsx}"}

// CLI BATS integration tests
{"ref": "test/cli/*.bats"}
```

❌ **FORBIDDEN:**
```json
// Generic phase scripts - TOO VAGUE
{"ref": "test/phases/test-integration.sh"}
{"ref": "test/phases/test-unit.sh"}
```

**Why**: Phase scripts orchestrate test execution but don't contain tests themselves. They provide zero traceability to actual test code and create gaming opportunities where many requirements point to the same vague reference.

**Auto-detection:**
- Vitest tests: Phase parser reads `ui/coverage/vitest-requirements.json`
- Go tests: Phase parser extracts `[REQ:ID]` from test output
- Auto-sync creates/updates these entries automatically

#### Type: automation

**Purpose**: Browser Automation Studio workflow tests

**Common patterns:**
```json
// JSON workflow file
{
  "type": "automation",
  "ref": "test/playbooks/ui/projects/create-project.json",
  "phase": "integration",
  "status": "implemented",
  "scenario": "browser-automation-studio",
  "folder": "/Testing Harness",
  "notes": "BAS workflow testing full project creation flow"
}

// Existing workflow ID
{
  "type": "automation",
  "workflow_id": "WORKFLOW-123456",
  "phase": "integration",
  "status": "planned",
  "scenario": "browser-automation-studio",
  "notes": "Shared workflow, not committed to repo"
}
```

**Execution:**
- Phase scripts call `testing::phase::run_bas_automation_validations`
- Helper imports workflow JSON or references existing workflow ID
- Returns pass/fail/skip status

**See Also:** [UI Automation with BAS](../guides/ui-automation-with-bas.md)

#### Type: manual

**Purpose**: Explicitly documented exceptions (e.g., quarterly penetration tests that cannot be automated yet). Manual entries should be rare, short-lived, and backed by artifacts **logged via** `vrooli scenario requirements manual-log` so auto-sync can track `validated_at`, `validated_by`, and `expires_at` metadata.

**Important**: Manual validations are **excluded from validation diversity requirements**. They serve as temporary measures before automated tests are implemented and do not count toward the multi-layer validation requirement for critical (P0/P1) requirements. See [Validation Diversity Requirements](#validation-diversity-requirements) for details.

```json
{
  "type": "manual",
  "ref": "docs/testing/runbooks/security-audit.md",
  "phase": "integration",
  "status": "planned",
  "notes": "Quarterly security penetration testing checklist (expires 30 days after last run)"
}
```

**Usage:**
- Reference runbooks, checklists, or documentation that describe the manual procedure.
- Record every execution with `vrooli scenario requirements manual-log <scenario> <REQ-ID> --status passed --notes "..."` (the CLI stores entries in `coverage/manual-validations/log.jsonl`).
- Log entries expire automatically (30-day default) so rerun the checklist before the deadline or convert it to an automated BAS workflow.
- `report.js --mode sync` consumes the manifest and updates `_sync_metadata.manual` fields. `vrooli scenario status` surfaces any expired or missing manual records.
- Treat manual entries as temporary bridges. Prefer replacing them with Browser Automation Studio workflows or other automated validations—see [UI Automation with BAS](../guides/ui-automation-with-bas.md) for UI/component coverage patterns.

## Validation Reference Patterns & Quality Requirements

### Allowed Test Source Directories

The `ref` field for validations must point to **actual test sources**, not test orchestration infrastructure.

#### Directory Structure: Valid vs Invalid Sources

```
scenario/
├── test/
│   ├── phases/          ❌ Orchestration scripts (NOT test sources)
│   │   ├── test-unit.sh        # Runs tests, doesn't contain them
│   │   └── test-integration.sh # Runs tests, doesn't contain them
│   ├── cli/             ❌ CLI wrapper tests (NOT for requirement validation)
│   │   └── *.bats              # Tests CLI thin wrapper, not business logic
│   ├── unit/            ❌ Test runner infrastructure
│   ├── integration/     ❌ Test runner infrastructure
│   └── playbooks/       ✅ E2E automation (actual test sources)
│       └── **/*.json           # BAS playbook workflows
│
├── api/
│   ├── **/*_test.go     ✅ API unit tests (actual test sources)
│   └── **/tests/**      ✅ API test directories (actual test sources)
│
└── ui/src/
    └── **/*.test.tsx    ✅ UI unit tests (actual test sources)
```

#### Rationale for Restrictions

| Directory | Why Rejected | What to Use Instead |
|-----------|--------------|---------------------|
| `test/phases/` | Orchestration scripts that **run** tests, not test sources themselves. Zero traceability to actual test code. | Reference the actual test files being orchestrated (API tests, UI tests, playbooks) |
| `test/cli/` | CLI wrapper tests validate the CLI layer, not the underlying business logic. The CLI should be a thin wrapper over the API. | Reference API tests (`api/**/*_test.go`) and e2e automation (`test/playbooks/`) |
| `test/unit/`, `test/integration/` | Test runner scripts or language-specific harnesses, not test sources. | Reference actual test files (Go tests in `api/`, Vitest tests in `ui/src/`) |

#### Allowed Validation Ref Patterns

```json
// ✅ ALLOWED - API unit tests (Go)
{"type": "test", "ref": "api/handlers/projects_test.go"}
{"type": "test", "ref": "api/services/workflow_service_test.go"}
{"type": "test", "ref": "api/database/repository_test.go"}

// ✅ ALLOWED - UI unit tests (Vitest/Jest)
{"type": "test", "ref": "ui/src/components/ProjectModal.test.tsx"}
{"type": "test", "ref": "ui/src/stores/workflowStore.test.ts"}

// ✅ ALLOWED - E2E automation (BAS playbooks)
{"type": "automation", "ref": "test/playbooks/capabilities/projects/create-project.json"}
{"type": "automation", "ref": "test/playbooks/capabilities/workflows/execute-workflow.json"}

// ❌ REJECTED - Orchestration scripts (no traceability)
{"type": "test", "ref": "test/phases/test-unit.sh"}
{"type": "test", "ref": "test/phases/test-integration.sh"}

// ❌ REJECTED - CLI wrapper tests (wrong layer)
{"type": "test", "ref": "test/cli/profile-operations.bats"}

// ❌ REJECTED - Test infrastructure
{"type": "test", "ref": "test/unit/run-unit-tests.sh"}
{"type": "test", "ref": "test/integration/setup.sh"}
```

### Test File Quality Requirements

Validation refs must point to **meaningful** test files, not superficial placeholders created to satisfy requirements.

#### Quality Indicators (Automated Detection)

Test files are analyzed for quality using these heuristics:

| Indicator | Threshold | Why It Matters |
|-----------|-----------|----------------|
| **Lines of Code** | ≥20 LOC (non-comment) | Shows actual test logic, not just imports |
| **Assertions** | ≥1 assertion | Tests must verify behavior, not just call functions |
| **Test Functions** | ≥1 test case | File must contain at least one actual test |
| **Multiple Test Functions** | ≥3 test cases | Comprehensive coverage, not just happy path |
| **Assertion Density** | ≥0.1 (1 per 10 LOC) | Good balance of setup and verification |

**Quality Score**: Files must score ≥4/5 indicators to count toward validation diversity.

#### Examples: Good vs Bad Test Quality

❌ **Bad - Superficial Test File** (quality score: 1/5):
```go
// api/placeholder_test.go (5 lines, no assertions, no logic)
package api

import "testing"

// Empty file, just exists to satisfy requirement
```

✅ **Good - Meaningful Test File** (quality score: 5/5):
```go
// api/dependency_analyzer_test.go (150+ lines, multiple test cases, edge cases)
package api

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestDependencyAnalyzer_RecursiveResolution(t *testing.T) {
    // Setup: Create test dependency graph
    analyzer := NewAnalyzer()

    // Test Case 1: Linear dependencies (A → B → C)
    result := analyzer.Resolve("A")
    assert.Equal(t, 3, len(result), "Should resolve 3 dependencies")

    // Test Case 2: Circular dependency detection
    err := analyzer.Resolve("CircularA")
    assert.Error(t, err, "Should detect circular dependencies")

    // Test Case 3: Missing dependency handling
    result = analyzer.Resolve("NonExistent")
    assert.Empty(t, result, "Should handle missing dependencies gracefully")

    // Test Case 4: Performance (≤5s for depth 10)
    start := time.Now()
    analyzer.Resolve("DeepTree")
    assert.Less(t, time.Since(start), 5*time.Second, "Should resolve quickly")
}

func TestDependencyAnalyzer_ConflictDetection(t *testing.T) {
    // ... 4 more test cases covering conflict scenarios ...
}

// ... 8 more test functions ...
```

**For Playbooks**: Similar quality checks apply:
- Must have ≥1 node/step with actions
- File size ≥100 bytes (not empty placeholder)
- Valid JSON structure with executable steps

### Validation Diversity Requirements

**Critical requirements (P0/P1) must be validated across ≥2 AUTOMATED test layers** to ensure comprehensive coverage. The specific layers required depend on scenario components.

#### Component-Aware Diversity

Diversity requirements adapt based on which components your scenario has:

```bash
# Detect scenario components
vrooli scenario info <name>  # Shows: API, UI, or both

# Component detection
- API component: api/ directory with .go files exists
- UI component: ui/ directory with package.json exists
```

#### Diversity Requirements by Component Type

##### Full-Stack Scenarios (API + UI components)

**Applicable layers**: API, UI, E2E

**P0/P1 requirements** must have ≥2 layers from:
- ✅ API unit tests + UI unit tests
- ✅ API unit tests + E2E automation
- ✅ UI unit tests + E2E automation
- ❌ API unit tests only (insufficient)
- ❌ UI unit tests + manual validation (manual doesn't count)

**Example**:
```json
{
  "id": "APP-WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",  // ← P0 criticality
  "status": "complete",
  "validation": [
    {"type": "test", "ref": "api/workflow_service_test.go", "phase": "unit"},     // ← API layer
    {"type": "test", "ref": "ui/src/stores/workflowStore.test.ts", "phase": "unit"}, // ← UI layer
    {"type": "automation", "ref": "test/playbooks/workflows/crud.json"}           // ← E2E layer (bonus)
  ]
  // ✅ Passes: 3 automated layers (API + UI + E2E)
}
```

##### API-Only Scenarios (No UI component)

**Applicable layers**: API, E2E

**P0/P1 requirements** must have ≥2 layers from:
- ✅ API unit tests + E2E automation
- ❌ API unit tests only (insufficient)
- ❌ API unit tests + manual validation (manual doesn't count)
- ⚠️ UI unit tests ignored (no UI component exists)

**Example**:
```json
{
  "id": "API-DEPENDENCY-ANALYSIS",
  "prd_ref": "OT-P0-002",  // ← P0 criticality
  "status": "complete",
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go", "phase": "unit"},         // ← API layer
    {"type": "automation", "ref": "test/playbooks/analyze-deps.json"}         // ← E2E layer
  ]
  // ✅ Passes: 2 automated layers (API + E2E) for API-only scenario
}
```

##### UI-Only Scenarios (No API component)

**Applicable layers**: UI, E2E

**P0/P1 requirements** must have ≥2 layers from:
- ✅ UI unit tests + E2E automation
- ❌ UI unit tests only (insufficient)
- ❌ UI unit tests + manual validation (manual doesn't count)
- ⚠️ API unit tests ignored (no API component exists)

#### Why Manual Validations Don't Count

Manual validations are **excluded from diversity requirements** because:
1. They are temporary measures before automation
2. They don't provide continuous validation
3. They can be gamed (mark as "manually verified" without real testing)
4. The goal is **automated** multi-layer coverage

**Manual validations are acceptable for**:
- Pending/in_progress requirements (temporary bridge)
- Requirements with existing automated tests (supplementary evidence)

**Manual validations are problematic when**:
- Complete requirements have ONLY manual validation (no automated tests)
- >10% of all validations are manual (suggests automation deficit)

#### P2 Requirements

**P2 requirements** (lower priority) only require ≥1 automated layer:
- ✅ API unit test alone is acceptable
- ✅ UI unit test alone is acceptable
- ✅ E2E automation alone is acceptable

**Example**:
```json
{
  "id": "APP-OPTIONAL-FEATURE",
  "prd_ref": "OT-P2-005",  // ← P2 criticality
  "status": "complete",
  "validation": [
    {"type": "test", "ref": "api/optional_test.go", "phase": "unit"}  // ← 1 layer sufficient for P2
  ]
  // ✅ Passes: P2 only needs 1 automated layer
}
```

### Checking Your Validation Setup

```bash
# View completeness score with gaming pattern warnings
vrooli scenario completeness <scenario-name>

# Common warnings:
# 🔴 "X requirements lack multi-layer AUTOMATED validation"
#    → Add tests across API, UI, and e2e layers for P0/P1 requirements
#
# 🔴 "X requirements reference unsupported test/ directories"
#    → Move refs from test/cli/ or test/phases/ to actual test sources
#
# 🟡 "X test files validate ≥4 requirements each"
#    → Break monolithic test files into focused tests per requirement
#
# 🟡 "X test files appear superficial"
#    → Add assertions, edge cases, and meaningful test logic
```

## Complete Examples

### Minimal Requirement

```json
{
  "id": "APP-BASIC-001",
  "title": "Application starts successfully",
  "status": "complete",
  "prd_ref": "OT-P0-001",
  "validation": [
    {
      "type": "test",
      "ref": "api/main_test.go",
      "phase": "unit",
      "status": "implemented"
    }
  ]
}
```

### Complex Multi-Phase Requirement

```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "category": "workflow.builder",
  "prd_ref": "OT-P0-001",
  "title": "Workflows persist nodes, edges, and metadata",
  "description": "Validates compiler, database, and API layers so workflows round-trip with folder hierarchy and normalized metadata required for execution planning.",
  "status": "complete",
  "tags": ["crud", "persistence", "workflows"],
  "validation": [
    {
      "type": "test",
      "ref": "api/automation/compiler/compiler_test.go",
      "phase": "unit",
      "status": "implemented",
      "notes": "Ensures ordering, cycle detection, and unsupported node handling"
    },
    {
      "type": "test",
      "ref": "api/database/repository_test.go",
      "phase": "unit",
      "status": "implemented",
      "notes": "Confirms workflow persistence and retrieval with folder hierarchy metadata"
    },
    {
      "type": "test",
      "ref": "ui/src/stores/__tests__/projectStore.test.ts",
      "phase": "unit",
      "status": "implemented",
      "notes": "Auto-added from vitest evidence"
    },
    {
      "type": "automation",
      "ref": "test/playbooks/projects/workflow-crud.json",
      "phase": "integration",
      "status": "implemented",
      "scenario": "browser-automation-studio",
      "notes": "End-to-end workflow creation, editing, and deletion"
    }
  ]
}
```

### Parent Requirement with Children

```json
{
  "id": "BAS-FUNC-001",
  "category": "foundation",
  "prd_ref": "OT-P0-001",
  "title": "Persist visual workflows with nodes/edges and folder hierarchy",
  "description": "Tracks the complete experience for creating and managing projects and workflows. Child implementation requirements cover dialog affordances, validation, and persistence.",
  "status": "in_progress",
  "children": [
    "BAS-PROJECT-DIALOG-OPEN",
    "BAS-PROJECT-DIALOG-CLOSE",
    "BAS-PROJECT-CREATE-SUCCESS",
    "BAS-PROJECT-CREATE-VALIDATION",
    "BAS-WORKFLOW-PERSIST-CRUD",
    "BAS-WORKFLOW-DEMO-SEED"
  ],
  "validation": []
}
```

**Behavior:**
- Parent status is rolled up from children
- Parent becomes `complete` only when all children are `complete`
- Parent validations are usually empty (children contain the actual tests)

## Index File Example

```json
{
  "_metadata": {
    "description": "Parent-level requirements with imports to child modules",
    "auto_sync_enabled": true,
    "schema_version": "1.0.0"
  },
  "meta": {
    "scenario": "browser-automation-studio",
    "version": "0.2.0",
    "owners": [
      {
        "platform": "browser-automation-studio",
        "contact": "execution-team@vrooli.dev"
      }
    ]
  },
  "imports": [
    "projects/dialog.json",
    "workflow-builder/core.json",
    "execution/telemetry.json"
  ],
  "requirements": [
    {
      "id": "BAS-FUNC-001",
      "title": "Visual workflow builder",
      "status": "in_progress",
      "prd_ref": "OT-P0-001",
      "children": [
        "BAS-PROJECT-DIALOG-OPEN",
        "BAS-WORKFLOW-PERSIST-CRUD"
      ]
    }
  ],
  "reporting": {
    "script": "scripts/requirements/report.js",
    "outputs": [
      {
        "format": "json",
        "description": "Machine-readable coverage report"
      }
    ]
  }
}
```

## Sync Metadata Storage

As of November 2025, sync metadata (test run timestamps, test names, durations, etc.) is stored separately from requirement files in `coverage/sync/*.json` files.

### Rationale

Requirement files (`requirements/**/*.json`) are git-tracked and should only contain semantic state (status, descriptions, etc.). Ephemeral test metadata belongs in the gitignored `coverage/` directory.

**Benefit**: Running tests no longer dirties requirement files with timestamp-only changes. Git status stays clean between test runs.

### Storage Location

- **Requirement files**: `requirements/01-module-name/module.json`
- **Sync metadata**: `coverage/sync/01-module-name.json`

### What Lives Where

**In requirement files** (git-tracked):
- Requirement IDs, titles, descriptions
- Status fields (updated only when tests pass/fail)
- PRD references, criticality, dependencies
- Validation definitions (type, ref, phase)

**In sync metadata files** (gitignored):
- Test run timestamps (`last_updated`, `last_test_run`)
- Test execution durations (`test_duration_ms`)
- Test names that cover each requirement (`test_names`)
- Phase result file references (`phase_results`, `phase_result`)
- Tests run commands (`tests_run`)

### Sync Metadata File Format

**File**: `coverage/sync/01-template-management.json`

```json
{
  "module_id": "template-management",
  "module_file": "requirements/01-template-management/module.json",
  "last_synced_at": "2025-11-21T22:45:00.000Z",
  "requirements": {
    "TMPL-AVAILABILITY": {
      "last_updated": "2025-11-21T22:45:00.000Z",
      "updated_by": "auto-sync",
      "test_coverage_count": 3,
      "all_tests_passing": true,
      "phase_results": [
        "coverage/phase-results/integration.json"
      ],
      "tests_run": [
        "test/run-tests.sh"
      ],
      "validations": {
        "0": {
          "last_test_run": "2025-11-21T22:45:00.000Z",
          "test_duration_ms": 1000,
          "auto_updated": true,
          "test_names": [
            "[REQ:TMPL-AVAILABILITY] CLI command 'template list' executes successfully"
          ],
          "phase_result": "coverage/phase-results/integration.json",
          "source_phase": "integration",
          "tests_run": ["test/run-tests.sh"]
        }
      }
    }
  }
}
```

### How It Works

1. **Test Execution**: Tests run and produce `coverage/phase-results/*.json`
2. **Sync Process**: `report.js --mode sync` reads phase results
3. **File Updates**:
   - **ALWAYS writes**: `coverage/sync/*.json` (all metadata, every run)
   - **ONLY writes when status changes**: `requirements/**/*.json` (semantic changes only)
4. **Snapshot Generation**: Combines data from both sources for reporting

### Accessing Sync Metadata

**In snapshots** (`coverage/requirements-sync/latest.json`):
```json
{
  "requirements": {
    "TMPL-AVAILABILITY": {
      "id": "TMPL-AVAILABILITY",
      "status": "complete",
      "sync_metadata": {
        "last_updated": "2025-11-21T22:45:00.000Z",
        "test_coverage_count": 3,
        "all_tests_passing": true
      },
      "validations": [
        {
          "type": "test",
          "status": "implemented",
          "sync_metadata": {
            "last_test_run": "2025-11-21T22:45:00.000Z",
            "test_names": ["..."]
          }
        }
      ]
    }
  }
}
```

The snapshot loader (`snapshot.js`) automatically merges requirement file data with sync metadata from `coverage/sync/` files.

## Validation Rules

The JSON schema enforces these constraints:

1. **ID Pattern**: Must match `[A-Z][A-Z0-9]+-[A-Z0-9-]+`
2. **Status Enum**: Must be one of the defined status values
3. **PRD Ref**: Required (any string format allowed)
4. **Validation Type**: Must be `test`, `automation`, `manual`, or `lighthouse`
5. **Phase Name**: Must be valid phase name
6. **Imports Pattern**: Must be `*.json` files
7. **Required Fields**: `id`, `title`, `status`, `prd_ref` on requirements; `type`, `phase`, `status` on validations

**Validate your registry:**
```bash
node scripts/requirements/validate.js --scenario <name>
```

## Schema Evolution

**Current Version**: 1.0.0

**Future Additions** (planned):
- Dependency enforcement (test ordering based on `depends_on`)
- Blocker visualization in reports
- Test data management metadata
- Performance benchmark thresholds

**Breaking Changes**: Will increment schema version to 2.0.0

## See Also

### Documentation
- [Requirement Tracking Guide](../guides/requirement-tracking.md) - Complete system usage
- [Phased Testing Architecture](../architecture/PHASED_TESTING.md) - How requirements integrate with phases
- [UI Automation with BAS](../guides/ui-automation-with-bas.md) - Using automation validations

### Schema Files
- [scripts/requirements/schema.json](/scripts/requirements/schema.json) - Canonical JSON schema
- [scripts/requirements/validate.js](/scripts/requirements/validate.js) - Validation tool
- [scripts/requirements/report.js](/scripts/requirements/report.js) - Reporting and sync tool

### Examples
- [browser-automation-studio/requirements/](/scenarios/browser-automation-studio/requirements/) - Reference modular registry
- [browser-automation-studio/requirements/index.json](/scenarios/browser-automation-studio/requirements/index.json) - Index file example
- [browser-automation-studio/requirements/projects/dialog.json](/scenarios/browser-automation-studio/requirements/projects/dialog.json) - Child module example
