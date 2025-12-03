# End-to-End Testing Example

**Status**: Active
**Last Updated**: 2025-12-02

---

## Overview

This document walks through a complete example from PRD requirement to automated test coverage. It demonstrates how all the testing components work together.

## The Scenario

We'll implement and test a "Project Creation" feature for a workflow builder application.

---

## Stage 1: PRD Definition

### PRD Extract

```markdown
## Operational Targets

### OT-P0-001: Project Management
Users can create, view, edit, and delete projects.

**Success Criteria:**
- Users can create a new project with name and description
- Created projects appear in the project list
- Projects persist across sessions
```

---

## Stage 2: Requirements Definition

### Create Requirement File

**File**: `requirements/01-foundation/projects.json`

```json
{
  "_metadata": {
    "description": "Project management requirements",
    "module": "foundation.projects",
    "auto_sync_enabled": true
  },
  "requirements": [
    {
      "id": "APP-PROJECT-CREATE",
      "title": "Users can create a new project",
      "description": "Project creation with name and description that persists to database",
      "status": "pending",
      "prd_ref": "OT-P0-001",
      "validation": []
    },
    {
      "id": "APP-PROJECT-LIST",
      "title": "Created projects appear in project list",
      "description": "Projects visible in dashboard after creation",
      "status": "pending",
      "prd_ref": "OT-P0-001",
      "validation": []
    }
  ]
}
```

---

## Stage 3: Implementation

### API Layer

**File**: `api/project_service.go`

```go
package api

import (
    "context"
    "fmt"
)

type Project struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

type CreateProjectRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type ProjectService struct {
    repo ProjectRepository
}

func (s *ProjectService) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
    if req.Name == "" {
        return nil, fmt.Errorf("name is required")
    }

    project := &Project{
        ID:          generateID(),
        Name:        req.Name,
        Description: req.Description,
    }

    if err := s.repo.Insert(ctx, project); err != nil {
        return nil, fmt.Errorf("failed to save project: %w", err)
    }

    return project, nil
}

func (s *ProjectService) List(ctx context.Context) ([]*Project, error) {
    return s.repo.FindAll(ctx)
}
```

### UI Layer

**File**: `ui/src/features/projects/CreateProjectDialog.tsx`

```typescript
import { selectors } from '@/consts/selectors';

export function CreateProjectDialog({ onClose, onSuccess }) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');

  const handleSubmit = async () => {
    const project = await createProject({ name, description });
    onSuccess(project);
    onClose();
  };

  return (
    <Dialog data-testid={selectors.projects.createDialog}>
      <DialogTitle>Create Project</DialogTitle>
      <TextField
        data-testid={selectors.projects.nameInput}
        label="Name"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <TextField
        data-testid={selectors.projects.descriptionInput}
        label="Description"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
      />
      <Button
        data-testid={selectors.projects.submitButton}
        onClick={handleSubmit}
      >
        Create
      </Button>
    </Dialog>
  );
}
```

---

## Stage 4: Unit Tests

### API Unit Test

**File**: `api/project_service_test.go`

```go
package api

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestProjectService_Create(t *testing.T) {
    t.Run("creates project with valid input [REQ:APP-PROJECT-CREATE]", func(t *testing.T) {
        repo := NewMockRepository()
        service := NewProjectService(repo)

        project, err := service.Create(context.Background(), &CreateProjectRequest{
            Name:        "Test Project",
            Description: "A test project",
        })

        require.NoError(t, err)
        assert.NotEmpty(t, project.ID)
        assert.Equal(t, "Test Project", project.Name)
        assert.Equal(t, "A test project", project.Description)

        // Verify persisted
        saved, _ := repo.FindByID(context.Background(), project.ID)
        assert.Equal(t, project.Name, saved.Name)
    })

    t.Run("rejects empty name [REQ:APP-PROJECT-CREATE]", func(t *testing.T) {
        repo := NewMockRepository()
        service := NewProjectService(repo)

        _, err := service.Create(context.Background(), &CreateProjectRequest{
            Name: "",
        })

        require.Error(t, err)
        assert.Contains(t, err.Error(), "name is required")
    })
}

func TestProjectService_List(t *testing.T) {
    t.Run("returns all projects [REQ:APP-PROJECT-LIST]", func(t *testing.T) {
        repo := NewMockRepository()
        service := NewProjectService(repo)

        // Create test data
        service.Create(context.Background(), &CreateProjectRequest{Name: "Project 1"})
        service.Create(context.Background(), &CreateProjectRequest{Name: "Project 2"})

        projects, err := service.List(context.Background())

        require.NoError(t, err)
        assert.Len(t, projects, 2)
    })
}
```

### UI Unit Test

**File**: `ui/src/features/projects/CreateProjectDialog.test.tsx`

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CreateProjectDialog } from './CreateProjectDialog';

describe('CreateProjectDialog [REQ:APP-PROJECT-CREATE]', () => {
  it('renders form fields', () => {
    render(<CreateProjectDialog onClose={jest.fn()} onSuccess={jest.fn()} />);

    expect(screen.getByTestId('projects-name-input')).toBeInTheDocument();
    expect(screen.getByTestId('projects-description-input')).toBeInTheDocument();
    expect(screen.getByTestId('projects-submit-button')).toBeInTheDocument();
  });

  it('submits form with entered values', async () => {
    const onSuccess = jest.fn();
    const mockCreate = jest.fn().mockResolvedValue({ id: '123', name: 'Test' });
    jest.mock('@/api', () => ({ createProject: mockCreate }));

    render(<CreateProjectDialog onClose={jest.fn()} onSuccess={onSuccess} />);

    fireEvent.change(screen.getByTestId('projects-name-input'), {
      target: { value: 'My Project' },
    });
    fireEvent.change(screen.getByTestId('projects-description-input'), {
      target: { value: 'Description' },
    });
    fireEvent.click(screen.getByTestId('projects-submit-button'));

    await waitFor(() => {
      expect(mockCreate).toHaveBeenCalledWith({
        name: 'My Project',
        description: 'Description',
      });
    });
  });
});
```

---

## Stage 5: E2E Automation

### BAS Workflow

**File**: `test/playbooks/capabilities/01-foundation/projects/create-project.json`

```json
{
  "metadata": {
    "description": "Test complete project creation flow",
    "version": 1,
    "reset": "none"
  },
  "settings": {
    "executionViewport": { "width": 1440, "height": 900, "preset": "desktop" }
  },
  "nodes": [
    {
      "id": "navigate",
      "type": "navigate",
      "data": {
        "label": "Navigate to dashboard",
        "destinationType": "scenario",
        "scenario": "my-app",
        "scenarioPath": "/",
        "waitUntil": "networkidle0",
        "waitForMs": 1500
      }
    },
    {
      "id": "click-create",
      "type": "click",
      "data": {
        "label": "Click create project button",
        "selector": "@selector/dashboard.createProjectButton",
        "waitForSelector": "@selector/dashboard.createProjectButton",
        "waitForMs": 500
      }
    },
    {
      "id": "assert-dialog",
      "type": "assert",
      "data": {
        "label": "Verify dialog opened",
        "selector": "@selector/projects.createDialog",
        "assertMode": "exists",
        "timeoutMs": 5000
      }
    },
    {
      "id": "fill-name",
      "type": "type",
      "data": {
        "label": "Enter project name",
        "selector": "@selector/projects.nameInput",
        "text": "E2E Test Project",
        "clearExisting": true
      }
    },
    {
      "id": "fill-description",
      "type": "type",
      "data": {
        "label": "Enter description",
        "selector": "@selector/projects.descriptionInput",
        "text": "Created by E2E test"
      }
    },
    {
      "id": "submit",
      "type": "click",
      "data": {
        "label": "Submit form",
        "selector": "@selector/projects.submitButton",
        "waitForMs": 2000
      }
    },
    {
      "id": "assert-success",
      "type": "assert",
      "data": {
        "label": "Verify project in list",
        "selector": "@selector/projects.cardByName(name=E2E Test Project)",
        "assertMode": "exists",
        "timeoutMs": 10000
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "navigate", "target": "click-create" },
    { "id": "e2", "source": "click-create", "target": "assert-dialog" },
    { "id": "e3", "source": "assert-dialog", "target": "fill-name" },
    { "id": "e4", "source": "fill-name", "target": "fill-description" },
    { "id": "e5", "source": "fill-description", "target": "submit" },
    { "id": "e6", "source": "submit", "target": "assert-success" }
  ]
}
```

---

## Stage 6: Link Validations

### Update Requirements

**File**: `requirements/01-foundation/projects.json` (updated)

```json
{
  "requirements": [
    {
      "id": "APP-PROJECT-CREATE",
      "title": "Users can create a new project",
      "status": "in_progress",
      "prd_ref": "OT-P0-001",
      "validation": [
        {
          "type": "test",
          "ref": "api/project_service_test.go",
          "phase": "unit",
          "status": "implemented",
          "notes": "Go unit tests for service layer"
        },
        {
          "type": "test",
          "ref": "ui/src/features/projects/CreateProjectDialog.test.tsx",
          "phase": "unit",
          "status": "implemented",
          "notes": "Vitest tests for dialog component"
        },
        {
          "type": "automation",
          "ref": "test/playbooks/capabilities/01-foundation/projects/create-project.json",
          "phase": "integration",
          "status": "implemented",
          "notes": "BAS workflow for full creation flow"
        }
      ]
    },
    {
      "id": "APP-PROJECT-LIST",
      "title": "Created projects appear in project list",
      "status": "in_progress",
      "prd_ref": "OT-P0-001",
      "validation": [
        {
          "type": "test",
          "ref": "api/project_service_test.go",
          "phase": "unit",
          "status": "implemented"
        },
        {
          "type": "automation",
          "ref": "test/playbooks/capabilities/01-foundation/projects/create-project.json",
          "phase": "integration",
          "status": "implemented",
          "notes": "Same workflow verifies project appears in list"
        }
      ]
    }
  ]
}
```

---

## Stage 7: Run Tests

### Execute Test Suite

```bash
# Run comprehensive tests
test-genie execute my-app --preset comprehensive
```

### Output

```
Test Genie - Executing tests for my-app

Phase: structure (15s)
  ✓ service.json exists
  ✓ UI smoke: Home page loads
  ✓ UI smoke: No console errors

Phase: dependencies (30s)
  ✓ Go runtime available
  ✓ Node.js runtime available
  ✓ PostgreSQL healthy

Phase: unit (60s)
  ✓ Go tests: 12 passed, 0 failed (coverage: 82%)
  ✓ Vitest tests: 24 passed, 0 failed (coverage: 78%)
  Requirements synced:
    - APP-PROJECT-CREATE: passed (2 tests)
    - APP-PROJECT-LIST: passed (1 test)

Phase: integration (120s)
  ✓ API health check passed
  ✓ UI accessible

Phase: business (180s)
  ✓ BAS workflow: create-project.json passed
  Requirements synced:
    - APP-PROJECT-CREATE: passed (automation)
    - APP-PROJECT-LIST: passed (automation)

Phase: performance (60s)
  ✓ Lighthouse: Performance 0.88, Accessibility 0.95

Summary:
  Total: 7 phases
  Passed: 7
  Failed: 0
  Duration: 4m 32s

Requirements:
  APP-PROJECT-CREATE: complete (3/3 validations passing)
  APP-PROJECT-LIST: complete (2/2 validations passing)
```

---

## Stage 8: View Results

### Dashboard

The Test Genie dashboard shows:

```
OT-P0-001: Project Management
├── APP-PROJECT-CREATE ✅
│   ├── API unit tests ✅
│   ├── UI unit tests ✅
│   └── E2E automation ✅
└── APP-PROJECT-LIST ✅
    ├── API unit tests ✅
    └── E2E automation ✅

Pass Rate: 100% (2/2 requirements)
Coverage: 3 layers (API, UI, E2E)
```

### Requirements Snapshot

**File**: `coverage/requirements-sync/latest.json`

```json
{
  "generated_at": "2025-12-02T14:30:00Z",
  "summary": {
    "total": 2,
    "complete": 2,
    "pass_rate": 1.0
  },
  "requirements": {
    "APP-PROJECT-CREATE": {
      "status": "complete",
      "validation_summary": {
        "total": 3,
        "passing": 3,
        "layers": ["API", "UI", "E2E"]
      }
    },
    "APP-PROJECT-LIST": {
      "status": "complete",
      "validation_summary": {
        "total": 2,
        "passing": 2,
        "layers": ["API", "E2E"]
      }
    }
  }
}
```

---

## Key Takeaways

### 1. Traceability

Every test traces back to:
- PRD operational target (OT-P0-001)
- Specific requirement (APP-PROJECT-CREATE)
- Test file and function

### 2. Multi-Layer Validation

P0 requirements validated across:
- API unit tests (service logic)
- UI unit tests (component behavior)
- E2E automation (full user flow)

### 3. Automation

The entire flow is automated:
- `[REQ:ID]` tags extracted automatically
- Validation entries created by sync
- Status derived from test results

### 4. Declarative E2E

BAS workflows are:
- JSON data structures
- Version controlled
- AI-friendly

---

## See Also

- [Requirement Flow Architecture](../concepts/requirement-flow.md) - How the system works
- [Testing Strategy](../concepts/strategy.md) - Three-layer approach
- [UI Automation with BAS](ui-automation-with-bas.md) - Workflow details
- [Requirements Sync](requirements-sync.md) - Auto-sync process
- [Scenario Unit Testing](scenario-unit-testing.md) - Writing unit tests
