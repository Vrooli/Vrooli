# End-to-End Testing Example: From PRD to Coverage

> **Complete walkthrough showing how requirement tracking, unit tests, and BAS workflows integrate**

## Table of Contents
- [Overview](#overview)
- [The Complete Flow](#the-complete-flow)
- [Example Scenario](#example-scenario)
- [Step 1: Define Requirement](#step-1-define-requirement)
- [Step 2: Design Testable UI](#step-2-design-testable-ui)
- [Step 3: Write Unit Tests](#step-3-write-unit-tests)
- [Step 4: Create BAS Workflow](#step-4-create-bas-workflow)
- [Step 5: Configure Phase Script](#step-5-configure-phase-script)
- [Step 6: Run Tests](#step-6-run-tests)
- [Step 7: Verify Coverage](#step-7-verify-coverage)
- [The Auto-Sync Magic](#the-auto-sync-magic)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

This guide demonstrates the complete testing workflow in Vrooli, from PRD requirement to live coverage report. We'll use a real example from `browser-automation-studio`: implementing project creation.

**What you'll learn**:
- How requirements link to PRD specifications
- How to tag tests with `[REQ:ID]` for automatic tracking
- How to write BAS workflows that validate requirements
- How auto-sync keeps coverage up-to-date

**Time to complete**: 30 minutes (first time), 10 minutes (once familiar)

## The Complete Flow

```mermaid
graph TB
    PRD[PRD.md<br/>Functional Requirements] -->|prd_ref| REQ[requirements/projects/core.json<br/>BAS-PROJECT-CREATE]

    REQ -->|Tag with [REQ:ID]| UNIT[Unit Tests<br/>projectStore.test.ts<br/>handlers_test.go]

    REQ -->|automation validation| BAS[BAS Workflow<br/>test/playbooks/ui/projects/create.json]

    UNIT -->|Execute| PHASE_UNIT[Phase: Unit<br/>test/phases/test-unit.sh]
    BAS -->|Execute| PHASE_INT[Phase: Integration<br/>test/phases/test-integration.sh]

    PHASE_UNIT --> RESULTS_UNIT[coverage/phase-results/unit.json]
    PHASE_INT --> RESULTS_INT[coverage/phase-results/integration.json]

    RESULTS_UNIT --> SYNC[Auto-Sync<br/>report.js --mode sync]
    RESULTS_INT --> SYNC

    SYNC -->|Updates status| REQ

    REQ --> REPORT[Coverage Report<br/>P0: 5/5 complete ✅]

    style PRD fill:#fff3e0
    style REQ fill:#e1f5ff
    style UNIT fill:#f3e5f5
    style BAS fill:#e8f5e9
    style PHASE_UNIT fill:#ede7f6
    style PHASE_INT fill:#ede7f6
    style SYNC fill:#fff9c4
    style REPORT fill:#c8e6c9
```

## Example Scenario

**Feature**: Users can create projects from the dashboard

**PRD Requirement** (from `PRD.md`):
> **Must Have**: Visual workflow builder with project management. Users shall be able to create, edit, and delete projects from the dashboard.

**Technical Requirement** (what we'll implement):
- Project creation dialog opens from dashboard
- Form validates project name (required, unique)
- Successful creation navigates to project detail
- Error messages shown for validation failures

## Step 1: Define Requirement

Create or edit `requirements/projects/core.json`:

```json
{
  "_metadata": {
    "description": "Core project management requirements",
    "auto_sync_enabled": true,
    "schema_version": "1.0.0"
  },
  "requirements": [
    {
      "id": "BAS-PROJECT-CREATE",
      "category": "projects.core",
      "prd_ref": "Operational Targets > P0 > OT-P0-001",
      "title": "Users can create projects from dashboard",
      "description": "Validates project creation flow including dialog opening, form validation, API persistence, and navigation to project detail page.",
      "status": "pending",
      "criticality": "P0",
      "validation": [
        {
          "type": "test",
          "ref": "ui/src/stores/__tests__/projectStore.test.ts",
          "phase": "unit",
          "status": "not_implemented",
          "notes": "Store logic with mocked API"
        },
        {
          "type": "test",
          "ref": "api/handlers/projects_test.go",
          "phase": "unit",
          "status": "not_implemented",
          "notes": "API handler with mocked database"
        },
        {
          "type": "automation",
          "ref": "test/playbooks/ui/projects/create-project.json",
          "phase": "integration",
          "status": "not_implemented",
          "notes": "BAS workflow testing full stack"
        }
      ]
    }
  ]
}
```

**Key fields**:
- `id`: Unique identifier used in test tags
- `prd_ref`: Links back to PRD section
- `status`: Will auto-update to `in_progress` when first test runs
- `validation`: Declares what tests should exist (auto-sync will update statuses)

## Step 2: Design Testable UI

**File**: `ui/src/components/Dashboard.tsx`

```tsx
import { useState } from 'react';
import { Plus } from 'lucide-react';
import { ProjectModal } from './ProjectModal';

export const Dashboard = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <div data-testid="dashboard-container">
      <header>
        <h1>Projects</h1>
        <button
          data-testid="dashboard-new-project-button"
          onClick={() => setIsModalOpen(true)}
          aria-label="Create new project"
        >
          <Plus aria-hidden="true" />
          New Project
        </button>
      </header>

      <div data-testid="project-list">
        {/* Project cards... */}
      </div>

      <ProjectModal
        open={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  );
};
```

**File**: `ui/src/components/ProjectModal.tsx`

```tsx
import { useState } from 'react';
import { useProjectStore } from '../stores/projectStore';

export const ProjectModal = ({ open, onClose }) => {
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const createProject = useProjectStore(s => s.createProject);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    try {
      await createProject({ name });
      onClose();
      // Navigate to project detail (not shown)
    } catch (err) {
      setError(err.message);
    }
  };

  if (!open) return null;

  return (
    <div data-testid="project-modal" role="dialog">
      <h2 data-testid="project-modal-title">Create Project</h2>

      <form data-testid="project-modal-form" onSubmit={handleSubmit}>
        <label htmlFor="project-name">Project Name</label>
        <input
          id="project-name"
          data-testid="project-name-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />

        {error && (
          <div data-testid="error-message" role="alert">
            {error}
          </div>
        )}

        <button
          type="button"
          data-testid="project-modal-cancel"
          onClick={onClose}
        >
          Cancel
        </button>
        <button
          type="submit"
          data-testid="project-modal-submit"
        >
          Create
        </button>
      </form>
    </div>
  );
};
```

**Critical observations**:
- ✅ Every interactive element has `data-testid`
- ✅ Semantic HTML (`<form>`, `<label>`, proper button types)
- ✅ Error state visible with `data-testid="error-message"`
- ✅ Modal state detectable (renders or not)

## Step 3: Write Unit Tests

### Vitest Test (Frontend)

**File**: `ui/src/stores/__tests__/projectStore.test.ts`

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { act } from '@testing-library/react';
import { useProjectStore } from '../projectStore';

// Mock fetch
global.fetch = vi.fn();

// TAG WITH REQUIREMENT ID (suite-level, all tests inherit)
describe('projectStore [REQ:BAS-PROJECT-CREATE]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useProjectStore.setState({ projects: [], error: null });
  });

  it('creates project successfully', async () => {
    const mockProject = {
      id: 'proj-1',
      name: 'Test Project',
      created_at: '2025-01-01',
    };

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ project: mockProject }),
    });

    await act(async () => {
      await useProjectStore.getState().createProject({ name: 'Test Project' });
    });

    const state = useProjectStore.getState();
    expect(state.projects).toHaveLength(1);
    expect(state.projects[0].name).toBe('Test Project');
    expect(state.error).toBeNull();
  });

  it('handles duplicate project name error', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 409,
      json: async () => ({ error: 'Project name already exists' }),
    });

    await act(async () => {
      try {
        await useProjectStore.getState().createProject({ name: 'Existing' });
      } catch (err) {
        // Expected
      }
    });

    const state = useProjectStore.getState();
    expect(state.error).toBe('Project name already exists');
  });
});
```

**Key point**: `[REQ:BAS-PROJECT-CREATE]` in describe block means ALL tests inherit this tag.

### Go Test (Backend)

**File**: `api/handlers/projects_test.go`

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

// TAG WITH REQUIREMENT ID (test-level in Go)
func TestCreateProject(t *testing.T) {
    t.Run("creates project successfully [REQ:BAS-PROJECT-CREATE]", func(t *testing.T) {
        payload := CreateProjectRequest{
            Name:        "Test Project",
            Description: "Test description",
        }
        body, _ := json.Marshal(payload)

        req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        handleCreateProject(w, req)

        if w.Code != http.StatusCreated {
            t.Errorf("Expected status 201, got %d", w.Code)
        }

        var resp CreateProjectResponse
        json.Unmarshal(w.Body.Bytes(), &resp)

        if resp.Project.Name != "Test Project" {
            t.Errorf("Expected project name 'Test Project', got %s", resp.Project.Name)
        }
    })

    t.Run("rejects duplicate project name [REQ:BAS-PROJECT-CREATE]", func(t *testing.T) {
        // Create first project
        createProject(t, "Duplicate Name")

        // Attempt to create duplicate
        payload := CreateProjectRequest{Name: "Duplicate Name"}
        body, _ := json.Marshal(payload)

        req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewReader(body))
        w := httptest.NewRecorder()

        handleCreateProject(w, req)

        if w.Code != http.StatusConflict {
            t.Errorf("Expected status 409 for duplicate, got %d", w.Code)
        }
    })
}
```

## Step 4: Create BAS Workflow

**File**: `test/playbooks/ui/projects/create-project.json`

```json
{
  "metadata": {
    "description": "Tests project creation from dashboard through to project detail page",
    "requirement": "BAS-PROJECT-CREATE",
    "version": 1
  },
  "settings": {
    "executionViewport": {
      "width": 1440,
      "height": 900,
      "preset": "desktop"
    }
  },
  "nodes": [
    {
      "id": "navigate-dashboard",
      "type": "navigate",
      "position": { "x": 0, "y": 0 },
      "data": {
        "label": "Navigate to dashboard",
        "destinationType": "scenario",
        "scenario": "browser-automation-studio",
        "scenarioPath": "/",
        "waitUntil": "networkidle0",
        "timeoutMs": 30000
      }
    },
    {
      "id": "click-new-project",
      "type": "click",
      "position": { "x": 220, "y": 0 },
      "data": {
        "label": "Click New Project button",
        "selector": "[data-testid='dashboard-new-project-button']",
        "waitForSelector": "[data-testid='dashboard-new-project-button']",
        "timeoutMs": 10000
      }
    },
    {
      "id": "wait-modal",
      "type": "wait",
      "position": { "x": 440, "y": 0 },
      "data": {
        "label": "Wait for modal to appear",
        "durationMs": 500
      }
    },
    {
      "id": "assert-modal-open",
      "type": "assert",
      "position": { "x": 660, "y": 0 },
      "data": {
        "label": "Verify modal opened",
        "selector": "[data-testid='project-modal']",
        "assertMode": "exists",
        "timeoutMs": 5000,
        "failureMessage": "Project modal should be visible"
      }
    },
    {
      "id": "fill-project-name",
      "type": "type",
      "position": { "x": 880, "y": 0 },
      "data": {
        "label": "Enter project name",
        "selector": "[data-testid='project-name-input']",
        "text": "E2E Test Project",
        "clearExisting": true
      }
    },
    {
      "id": "screenshot-form",
      "type": "screenshot",
      "position": { "x": 1100, "y": -100 },
      "data": {
        "label": "Capture filled form",
        "fullPage": false,
        "captureDomSnapshot": true
      }
    },
    {
      "id": "click-submit",
      "type": "click",
      "position": { "x": 1100, "y": 100 },
      "data": {
        "label": "Click Create button",
        "selector": "[data-testid='project-modal-submit']",
        "waitForMs": 500
      }
    },
    {
      "id": "wait-navigation",
      "type": "wait",
      "position": { "x": 1320, "y": 100 },
      "data": {
        "label": "Wait for navigation",
        "durationMs": 2000
      }
    },
    {
      "id": "assert-project-detail",
      "type": "assert",
      "position": { "x": 1540, "y": 100 },
      "data": {
        "label": "Verify on project detail page",
        "selector": "h1",
        "assertMode": "contains_text",
        "expectedText": "E2E Test Project",
        "timeoutMs": 10000,
        "failureMessage": "Should navigate to project detail page"
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "navigate-dashboard", "target": "click-new-project" },
    { "id": "e2", "source": "click-new-project", "target": "wait-modal" },
    { "id": "e3", "source": "wait-modal", "target": "assert-modal-open" },
    { "id": "e4", "source": "assert-modal-open", "target": "fill-project-name" },
    { "id": "e5", "source": "fill-project-name", "target": "screenshot-form" },
    { "id": "e6", "source": "fill-project-name", "target": "click-submit" },
    { "id": "e7", "source": "click-submit", "target": "wait-navigation" },
    { "id": "e8", "source": "wait-navigation", "target": "assert-project-detail" }
  ]
}
```

**Key elements**:
- `metadata`: Describes the workflow (and its `reset` scope). Requirement linkage now comes from `requirements/*.json` via `validation.ref` entries, so the workflow JSON does **not** embed a requirement ID.
- Node types: `navigate`, `click`, `type`, `assert`, `screenshot`, `wait`
- Selectors use `data-testid` from UI code
- Linear flow with branching for screenshot

## Step 5: Configure Phase Script

**File**: `test/phases/test-integration.sh`

```bash
#!/bin/bash
set -euo pipefail

PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$PHASE_DIR/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "${SCENARIO_DIR}/../.." && pwd)}"

# Source phase helpers
source "$APP_ROOT/scripts/scenarios/testing/shell/phase-helpers.sh"
source "$APP_ROOT/scripts/scenarios/testing/playbooks/browser-automation-studio.sh"

# Initialize phase
testing::phase::init \
  --name "integration" \
  --scenario-dir "$SCENARIO_DIR"

# Run BAS automation validations (finds all type: automation validations)
if ! testing::phase::run_bas_automation_validations \
    --scenario "browser-automation-studio" \
    --manage-runtime auto; then
  testing::phase::add_error "BAS workflow validations failed"
fi

# Complete phase
testing::phase::end_with_summary
```

**What happens**:
1. `testing::phase::init` loads requirement registry
2. `testing::phase::run_bas_automation_validations` finds all `type: automation` validations
3. For each validation, imports workflow JSON, executes via BAS API, waits for completion
4. Success/failure status recorded with `testing::phase::add_requirement`
5. Results saved to `coverage/phase-results/integration.json`

## Step 6: Run Tests

```bash
cd scenarios/browser-automation-studio

# Run all phases
./test/run-tests.sh

# Or run specific phases
./test/phases/test-unit.sh         # Runs Vitest + Go tests
./test/phases/test-integration.sh  # Runs BAS workflows
```

**Expected output**:

```
=== Running Comprehensive Tests for browser-automation-studio ===

[Phase 1/6] Structure Validation...
✅ Phase completed: structure (2.3s)

[Phase 2/6] Dependency Checks...
✅ Phase completed: dependencies (8.1s)

[Phase 3/6] Unit Tests...
Running Go tests...
  ✓ PASS REQ:BAS-PROJECT-CREATE (2 tests, 145ms)
Running Node tests...
  ✓ PASS REQ:BAS-PROJECT-CREATE (2 tests, 523ms)
✅ Phase completed: unit (12.5s)
  Requirements tracked: BAS-PROJECT-CREATE (passed)

[Phase 4/6] Integration Tests...
Importing workflow: create-project.json
Executing workflow: create-project-8f3a2b
  ✓ navigate-dashboard
  ✓ click-new-project
  ✓ wait-modal
  ✓ assert-modal-open
  ✓ fill-project-name
  ✓ screenshot-form
  ✓ click-submit
  ✓ wait-navigation
  ✓ assert-project-detail
Workflow completed: success (8.2s)
✅ Phase completed: integration (15.8s)
  Requirements tracked: BAS-PROJECT-CREATE (passed)

[Phase 5/6] Business Logic Tests...
✅ Phase completed: business (5.2s)

[Phase 6/6] Performance Tests...
✅ Phase completed: performance (3.1s)

=== All Tests Passed! ===
Total time: 47.0s

📋 Syncing requirement registry...
  Updated: BAS-PROJECT-CREATE (pending → in_progress)
  Updated validation statuses: 3 implemented, 0 failing
✅ Requirements registry synced
```

## Step 7: Verify Coverage

```bash
# Generate coverage report
node scripts/requirements/report.js \
  --scenario browser-automation-studio \
  --format markdown
```

**Output**:

```markdown
# Requirement Coverage: browser-automation-studio

Generated: 2025-11-05T18:30:00.000Z

## Summary

| Metric | Value |
|--------|-------|
| Total Requirements | 21 |
| Complete | 19 |
| In Progress | 2 |
| Pending | 0 |
| Criticality Gap (P0/P1 incomplete) | 2 |

## Requirements by Status

### In Progress (2)

#### BAS-PROJECT-CREATE (P0)
**Category**: projects.core
**Title**: Users can create projects from dashboard
**PRD Ref**: Operational Targets > P0 > OT-P0-001
**Live Status**: ✅ All validations passing

**Validations**:
- ✅ `ui/src/stores/__tests__/projectStore.test.ts` (unit) - **implemented**
  - Evidence: Node test ✓ PASS REQ:BAS-PROJECT-CREATE (2 tests, 523ms)
- ✅ `api/handlers/projects_test.go` (unit) - **implemented**
  - Evidence: Go test ✓ PASS REQ:BAS-PROJECT-CREATE (2 tests, 145ms)
- ✅ `test/playbooks/ui/projects/create-project.json` (integration) - **implemented**
  - Evidence: BAS workflow completed successfully (8.2s)

**Next Action**: Mark as complete (all validations passing)

---

### Complete (19)

[... other requirements ...]
```

## The Auto-Sync Magic

**What happened automatically**:

1. **Test Execution** → Tags extracted
   - Vitest: `vitest-requirement-reporter` extracts `[REQ:ID]` from suite/test names
   - Go: Phase script parses `go test -v` output for `[REQ:ID]` patterns
   - BAS: `requirements/*.json` entries reference playbooks via `validation.ref`, so coverage links back to workflows without relying on playbook metadata

2. **Phase Results** → Evidence collected
   - Saved to `coverage/phase-results/unit.json`:
     ```json
     {
       "phase": "unit",
       "requirements": [
         {
           "id": "BAS-PROJECT-CREATE",
           "status": "passed",
           "evidence": "Node test ✓ PASS (2 tests); Go test ✓ PASS (2 tests)"
         }
       ]
     }
     ```

3. **Auto-Sync** → Registry updated
   - `report.js --mode sync` reads phase results
   - Finds requirement `BAS-PROJECT-CREATE` in `requirements/projects/core.json`
   - Updates validation statuses: `not_implemented` → `implemented`
   - Updates requirement status: `pending` → `in_progress` (because tests exist)
   - Would update to `complete` if all validations pass (they do!)

4. **Coverage Report** → Live status
   - Reads updated registry
   - Shows P0/P1/P2 completion
   - Identifies gaps (requirements without tests)

**No manual tracking. No spreadsheets. No drift.**

## Troubleshooting

### Requirement not showing in phase results

**Symptom**: Test runs but requirement not in `coverage/phase-results/*.json`

**Causes**:
1. Tag doesn't match registry ID exactly
2. Vitest reporter not configured
3. Test didn't run (skipped/filtered)

**Solutions**:
```typescript
// ✅ CORRECT: Exact match
describe('projectStore [REQ:BAS-PROJECT-CREATE]', () => { ... })

// Registry has:
{ "id": "BAS-PROJECT-CREATE", ... }

// ❌ WRONG: Typo
describe('projectStore [REQ:BAS-PROJECT-CREAT]', () => { ... })
```

### BAS workflow fails to import

**Symptom**: Integration phase fails with "workflow not found"

**Causes**:
1. JSON file doesn't exist at path
2. JSON syntax error
3. BAS scenario not running

**Solutions**:
```bash
# Check file exists
ls -la test/playbooks/ui/projects/create-project.json

# Validate JSON
jq . test/playbooks/ui/projects/create-project.json

# Ensure BAS is running
vrooli scenario status browser-automation-studio
```

### Selector not found in workflow

**Symptom**: BAS workflow fails at specific step with "selector not found"

**Causes**:
1. `data-testid` doesn't exist in UI
2. Timing issue (element not loaded yet)
3. Modal/dialog not opened

**Solutions**:
1. Check UI code has `data-testid="..."` attribute
2. Add `wait` node before `click`/`assert` nodes
3. Add `waitForMs` to node `data`

## See Also

### Related Guides
- **[Writing Testable UIs](writing-testable-uis.md)** - Use `data-testid`, avoid pitfalls
- **[BAS Workflow Authoring](ui-automation-with-bas.md)** - Complete workflow guide
- **[Requirement Tracking](requirement-tracking.md)** - Full system documentation
- **[Quick Start](quick-start.md)** - 5-minute introduction

### Reference Implementation
- **[browser-automation-studio](../../../scenarios/browser-automation-studio/)** - Real example of self-testing scenario
- **[Requirements registry](../../../scenarios/browser-automation-studio/requirements/)** - Modular structure example

### Tools
- **[@vrooli/vitest-requirement-reporter](../../../packages/vitest-requirement-reporter/)** - Vitest integration package
- **[report.js](../../../scripts/requirements/report.js)** - Coverage reporting tool

---

**Remember**: This workflow becomes second nature after 2-3 iterations. The auto-sync eliminates 80% of tracking overhead.
