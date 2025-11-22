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
  "criticality": "P0"
}
```

| Field | Type | Pattern/Enum | Description |
|-------|------|--------------|-------------|
| `id` | string | `[A-Z][A-Z0-9]+-[A-Z0-9-]+` | Unique requirement identifier |
| `title` | string | min 1 char | Short description (1-2 sentences) |
| `status` | enum | See [Status Values](#status-values) | Implementation progress |
| `criticality` | enum | `P0`, `P1`, `P2` | Priority level |

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

#### Criticality Levels

| Level | Meaning | Example Use Cases |
|-------|---------|-------------------|
| `P0` | Critical (Must Have) | Core functionality, blocking bugs, security |
| `P1` | Important (Should Have) | Secondary features, non-blocking bugs |
| `P2` | Nice-to-Have (Could Have) | Polish, edge cases, minor optimizations |

### Optional Fields

```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "title": "Workflows persist nodes, edges, and metadata",
  "status": "complete",
  "criticality": "P0",

  "category": "workflow.builder",
  "prd_ref": "Operational Targets > P0 > OT-P0-001",
  "description": "Validates compiler, database, and API layers so workflows round-trip",
  "tags": ["storage", "persistence", "crud"],
  "children": ["BAS-PROJECT-CREATE", "BAS-WORKFLOW-SAVE"],
  "depends_on": ["BAS-DATABASE-SETUP"],
  "blocks": ["BAS-WORKFLOW-SHARE"],
  "validation": [...]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `category` | string | Hierarchical grouping (e.g., `workflow.builder`, `projects.ui`) |
| `prd_ref` | string | Reference to PRD section this requirement traces to |
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

## Complete Examples

### Minimal Requirement

```json
{
  "id": "APP-BASIC-001",
  "title": "Application starts successfully",
  "status": "complete",
  "criticality": "P0",
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
  "prd_ref": "Operational Targets > P0 > OT-P0-001",
  "title": "Workflows persist nodes, edges, and metadata",
  "description": "Validates compiler, database, and API layers so workflows round-trip with folder hierarchy and normalized metadata required for execution planning.",
  "status": "complete",
  "criticality": "P0",
  "tags": ["crud", "persistence", "workflows"],
  "validation": [
    {
      "type": "test",
      "ref": "api/browserless/compiler/compiler_test.go",
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
  "prd_ref": "Operational Targets > P0 > OT-P0-001",
  "title": "Persist visual workflows with nodes/edges and folder hierarchy",
  "description": "Tracks the complete experience for creating and managing projects and workflows. Child implementation requirements cover dialog affordances, validation, and persistence.",
  "status": "in_progress",
  "criticality": "P0",
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
      "criticality": "P0",
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
3. **Criticality Enum**: Must be `P0`, `P1`, or `P2`
4. **Validation Type**: Must be `test`, `automation`, `manual`, or `lighthouse`
5. **Phase Name**: Must be valid phase name
6. **Imports Pattern**: Must be `*.json` files
7. **Required Fields**: `id`, `title`, `status`, `criticality` on requirements; `type`, `phase`, `status` on validations

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
