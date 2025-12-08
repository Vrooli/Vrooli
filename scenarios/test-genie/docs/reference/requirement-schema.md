# Requirement Registry Schema Reference

**Status**: Active
**Last Updated**: 2025-12-02

---

Human-readable reference for the Vrooli requirement registry JSON schema.

## Overview

The requirement registry is a JSON file (or collection of modular JSON files) that defines:
- **Requirements**: What needs to be built/validated
- **Validations**: How requirements are proven (tests, automation, manual steps)
- **Hierarchy**: Parent-child relationships and dependencies
- **Status**: Implementation and test progress

**Schema Version**: 1.0.0

---

## File Structure

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `_metadata` | object | No | File-level metadata (description, sync settings) |
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

---

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
| `status` | enum | See Status Values | Implementation progress |
| `prd_ref` | string | - | Reference to PRD |

#### ID Pattern Examples

**Valid:**
- `BAS-WORKFLOW-PERSIST-CRUD`
- `APP-FUNC-001`
- `MY-SCENARIO-FEATURE-NAME`

**Invalid:**
- `bas-workflow-crud` (lowercase)
- `WORKFLOW_CRUD` (no prefix)
- `1BAS-WORKFLOW` (starts with digit)

#### Status Values

| Value | Meaning | When to Use |
|-------|---------|-------------|
| `pending` | Not yet planned | Requirement identified but not scheduled |
| `planned` | Scheduled, approach defined | Validation approach documented, no code yet |
| `in_progress` | Work started | At least one validation exists |
| `complete` | Fully implemented | All validations passing |
| `not_implemented` | Deprioritized | Requirement deferred or cancelled |

#### PRD Reference Format

**Operational Target Format** (Recommended):
```
"prd_ref": "OT-P0-001"
```
- Pattern: `OT-P[012]-NNN` where P0/P1/P2 indicates priority
- Criticality is automatically derived

**Freeform Format** (Legacy):
```
"prd_ref": "Functional Requirements > Must Have > Visual workflow builder"
```

### Optional Fields

```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "title": "Workflows persist nodes, edges, and metadata",
  "status": "complete",
  "prd_ref": "OT-P0-001",

  "category": "workflow.builder",
  "description": "Validates compiler, database, and API layers",
  "tags": ["storage", "persistence", "crud"],
  "children": ["BAS-PROJECT-CREATE", "BAS-WORKFLOW-SAVE"],
  "depends_on": ["BAS-DATABASE-SETUP"],
  "validation": [...]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `category` | string | Hierarchical grouping |
| `description` | string | Detailed explanation |
| `tags` | array of strings | Custom tags for filtering |
| `children` | array of strings | Child requirement IDs |
| `depends_on` | array of strings | Requirement IDs this depends on |
| `validation` | array of objects | Validation methods |

---

## Validation Definition

Each requirement can have multiple validation entries.

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
| `type` | string | `test`, `automation`, `manual`, `lighthouse` | Validation method |
| `phase` | string | `unit`, `integration`, `business`, `performance` | Test phase |
| `status` | string | `not_implemented`, `planned`, `implemented`, `failing` | Current state |

### Optional Fields

```json
{
  "type": "test",
  "ref": "ui/src/stores/__tests__/projectStore.test.ts",
  "phase": "unit",
  "status": "implemented",
  "notes": "Auto-added from vitest evidence",
  "scenario": "browser-automation-studio"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ref` | string | File path reference |
| `workflow_id` | string | Workflow ID for automation validations |
| `notes` | string | Human-readable notes |
| `scenario` | string | External scenario name |

### Validation Types

#### Type: test

```json
{
  "type": "test",
  "ref": "api/handlers/projects_test.go",
  "phase": "unit",
  "status": "implemented",
  "notes": "Covers CRUD operations"
}
```

**Validation Specificity Requirements:**

**ALLOWED:**
```json
{"ref": "api/handlers/projects_test.go"}
{"ref": "ui/src/**/*.test.{ts,tsx}"}
{"ref": "test/cli/*.bats"}
```

**FORBIDDEN:**
```json
{"ref": "test/phases/test-integration.sh"}
{"ref": "test/phases/test-unit.sh"}
```

#### Type: automation

```json
{
  "type": "automation",
  "ref": "test/playbooks/ui/projects/create-project.json",
  "phase": "integration",
  "status": "implemented",
  "scenario": "browser-automation-studio"
}
```

#### Type: manual

```json
{
  "type": "manual",
  "ref": "docs/testing/runbooks/security-audit.md",
  "phase": "integration",
  "status": "planned",
  "notes": "Quarterly security testing (expires 30 days)"
}
```

**Important**: Manual validations are **excluded from validation diversity requirements**.

#### Type: lighthouse

```json
{
  "type": "lighthouse",
  "ref": ".vrooli/lighthouse.json",
  "page_id": "home",
  "category": "performance",
  "threshold": 0.85,
  "phase": "performance",
  "status": "implemented"
}
```

---

## Validation Diversity Requirements

**P0/P1 requirements must have >= 2 AUTOMATED test layers.**

### By Component Type

**Full-Stack (API + UI):**
- Applicable layers: API, UI, E2E
- Need 2+ from: API + UI, API + E2E, UI + E2E

**API-Only:**
- Applicable layers: API, E2E
- Need: API + E2E

**UI-Only:**
- Applicable layers: UI, E2E
- Need: UI + E2E

### Example

```json
{
  "id": "APP-WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/workflow_service_test.go", "phase": "unit"},
    {"type": "test", "ref": "ui/src/stores/workflowStore.test.ts", "phase": "unit"},
    {"type": "automation", "ref": "test/playbooks/workflows/crud.json"}
  ]
}
// 3 automated layers (API + UI + E2E)
```

---

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
  "description": "Validates compiler, database, and API layers",
  "status": "complete",
  "tags": ["crud", "persistence"],
  "validation": [
    {
      "type": "test",
      "ref": "api/automation/compiler/compiler_test.go",
      "phase": "unit",
      "status": "implemented"
    },
    {
      "type": "test",
      "ref": "ui/src/stores/__tests__/projectStore.test.ts",
      "phase": "unit",
      "status": "implemented"
    },
    {
      "type": "automation",
      "ref": "test/playbooks/projects/workflow-crud.json",
      "phase": "integration",
      "status": "implemented"
    }
  ]
}
```

### Index File Example

```json
{
  "_metadata": {
    "description": "Parent-level requirements with imports",
    "auto_sync_enabled": true,
    "schema_version": "1.0.0"
  },
  "imports": [
    "projects/dialog.json",
    "workflow-builder/core.json"
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
  ]
}
```

---

## Sync Metadata Storage

Sync metadata is stored separately in `coverage/sync/*.json` files.

### What Lives Where

**In requirement files** (git-tracked):
- Requirement IDs, titles, descriptions
- Status fields
- PRD references, criticality, dependencies
- Validation definitions

**In sync metadata files** (gitignored):
- Test run timestamps
- Test execution durations
- Test names that cover each requirement

---

## Validation Rules

1. **ID Pattern**: Must match `[A-Z][A-Z0-9]+-[A-Z0-9-]+`
2. **Status Enum**: Must be one of the defined values
3. **PRD Ref**: Required
4. **Validation Type**: Must be `test`, `automation`, `manual`, or `lighthouse`

**Validate your registry:**
```bash
vrooli scenario requirements validate <name>
```

---

## See Also

- [Requirement Flow Architecture](../concepts/requirement-flow.md) - End-to-end flow
- [Requirements Sync Guide](../phases/business/requirements-sync.md) - How sync works
- [Validation Best Practices](../guides/validation-best-practices.md) - Quality guidelines
- [Gaming Prevention](gaming-prevention.md) - Detection system
