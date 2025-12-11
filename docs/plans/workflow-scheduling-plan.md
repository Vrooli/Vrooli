# Workflow Scheduling Implementation Plan

**Goal**: Add cron-based workflow scheduling to browser-automation-studio with desktop tray integration, enabling users to automate recurring browser tasks even when the main window is minimized.

**Status**: Planning
**Created**: 2025-12-11
**Target Scenario**: `scenarios/browser-automation-studio`
**Dependencies**: `scenarios/scenario-to-desktop` (tray enhancements)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Phase 1: Foundation - API Schedule CRUD](#phase-1-foundation---api-schedule-crud)
4. [Phase 2: Scheduler Service](#phase-2-scheduler-service)
5. [Phase 3: UI Components](#phase-3-ui-components)
6. [Phase 4: Desktop Tray Integration](#phase-4-desktop-tray-integration)
7. [Phase 5: Notifications](#phase-5-notifications)
8. [Testing Strategy](#testing-strategy)
9. [Documentation Updates](#documentation-updates)
10. [File Manifest](#file-manifest)
11. [Reference Links](#reference-links)

---

## Executive Summary

### Problem Statement

Users want to run browser automation workflows on a schedule (daily checks, weekly reports, etc.) without manually triggering each execution. As a desktop Electron app, this requires careful consideration of app lifecycle—specifically, what happens when the user closes the window.

### Solution Approach

Implement a **hybrid scheduling system**:

1. **In-App Scheduler**: Cron-based scheduler runs in the Go API process while the app is open
2. **Tray Mode**: When schedules exist, minimize to system tray instead of quitting, keeping the scheduler alive
3. **Clear UX Messaging**: Communicate to users that schedules require the app to be running (in foreground or tray)

### Key Constraints

- **No OS-level scheduling**: Avoids complexity of platform-specific Task Scheduler/launchd/cron integration
- **Transparent state**: Users always know whether their schedules will run
- **Graceful degradation**: If app is closed, missed schedules are logged (not silently dropped)

### Effort Estimate

| Phase | Description | Effort |
|-------|-------------|--------|
| Phase 1 | API Schedule CRUD | 4-6 hours |
| Phase 2 | Scheduler Service | 4-6 hours |
| Phase 3 | UI Components | 6-8 hours |
| Phase 4 | Desktop Tray Integration | 3-4 hours |
| Phase 5 | Notifications | 2-3 hours |
| Testing | Unit + Integration | 4-6 hours |
| Docs | Updates | 1-2 hours |
| **Total** | | **24-35 hours** |

---

## Architecture Overview

### System Context

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         ELECTRON SHELL                                   │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  main.ts (scenario-to-desktop template)                           │  │
│  │  - ENABLE_SYSTEM_TRAY = true                                      │  │
│  │  - Tray IPC handlers for tooltip/menu updates                     │  │
│  │  - Desktop notification bridge                                     │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                    │                                     │
│                          IPC / preload bridge                            │
│                                    ▼                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  React UI (browser-automation-studio/ui)                          │  │
│  │  - SchedulesTab component                                          │  │
│  │  - ScheduleModal (create/edit)                                     │  │
│  │  - Tray update utilities                                           │  │
│  │  - Notification listeners                                          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                           HTTP / WebSocket
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         GO API SERVER                                    │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │ handlers/       │  │ services/       │  │ database/               │  │
│  │ schedules.go    │  │ scheduler/      │  │ repository_schedules.go │  │
│  │                 │  │ scheduler.go    │  │                         │  │
│  │ REST endpoints  │  │ service.go      │  │ CRUD for                │  │
│  │ for CRUD        │  │ cron.go         │  │ workflow_schedules      │  │
│  └─────────────────┘  │ notifier.go     │  └─────────────────────────┘  │
│           │           └─────────────────┘             │                  │
│           │                    │                      │                  │
│           └────────────────────┼──────────────────────┘                  │
│                                ▼                                         │
│                   workflow.ExecutionService                              │
│                   (existing execution pipeline)                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Boundary of Responsibility

| Component | Responsibility | Does NOT |
|-----------|---------------|----------|
| `handlers/schedules.go` | HTTP request/response, validation, routing | Business logic, cron parsing |
| `services/scheduler/` | Schedule lifecycle, cron management, execution triggering | HTTP concerns, UI state |
| `database/repository_schedules.go` | Persistence, queries | Business rules |
| `ui/features/schedules/` | User interaction, display | Schedule execution |
| `scenario-to-desktop` | Tray infrastructure, IPC | Schedule logic |

### Data Model

The `WorkflowSchedule` model already exists in `database/models.go:257-270`:

```go
type WorkflowSchedule struct {
    ID             uuid.UUID  `json:"id" db:"id"`
    WorkflowID     uuid.UUID  `json:"workflow_id" db:"workflow_id"`
    Name           string     `json:"name" db:"name"`
    Description    string     `json:"description,omitempty" db:"description"`
    CronExpression string     `json:"cron_expression" db:"cron_expression"`
    Timezone       string     `json:"timezone" db:"timezone"`
    IsActive       bool       `json:"is_active" db:"is_active"`
    Parameters     JSONMap    `json:"parameters,omitempty" db:"parameters"`
    NextRunAt      *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
    LastRunAt      *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
    CreatedAt      time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}
```

The database schema already exists in `initialization/storage/postgres/schema.sql`.

---

## Phase 1: Foundation - API Schedule CRUD

### Objective

Expose REST endpoints for creating, reading, updating, and deleting workflow schedules.

### Files to Create/Modify

#### 1.1 Repository Layer

**File**: `api/database/repository_schedules.go` (NEW)

```go
package database

// ScheduleRepository defines schedule-specific persistence operations.
// Implementations: PostgresRepository (production), MockRepository (tests).
type ScheduleRepository interface {
    CreateSchedule(ctx context.Context, schedule *WorkflowSchedule) error
    GetSchedule(ctx context.Context, id uuid.UUID) (*WorkflowSchedule, error)
    ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*WorkflowSchedule, error)
    UpdateSchedule(ctx context.Context, schedule *WorkflowSchedule) error
    DeleteSchedule(ctx context.Context, id uuid.UUID) error

    // Scheduler-specific queries
    GetActiveSchedules(ctx context.Context) ([]*WorkflowSchedule, error)
    UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRun time.Time) error
    UpdateLastRunAt(ctx context.Context, id uuid.UUID, lastRun time.Time) error
}
```

**Implementation requirements**:
- Add methods to existing `PostgresRepository` in `repository.go`
- Ensure `Repository` interface includes `ScheduleRepository`
- Add corresponding methods to mock repository for tests

#### 1.2 Handler Layer

**File**: `api/handlers/schedules.go` (NEW)

**Endpoints**:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/workflows/{id}/schedules` | Create schedule for workflow |
| `GET` | `/api/v1/workflows/{id}/schedules` | List schedules for workflow |
| `GET` | `/api/v1/schedules` | List all schedules (global) |
| `GET` | `/api/v1/schedules/{id}` | Get schedule by ID |
| `PATCH` | `/api/v1/schedules/{id}` | Update schedule |
| `DELETE` | `/api/v1/schedules/{id}` | Delete schedule |
| `POST` | `/api/v1/schedules/{id}/trigger` | Manually trigger schedule |
| `POST` | `/api/v1/schedules/{id}/toggle` | Toggle active/inactive |

**Request/Response contracts**:

```go
// CreateScheduleRequest for POST /api/v1/workflows/{id}/schedules
type CreateScheduleRequest struct {
    Name           string         `json:"name" validate:"required,min=1,max=255"`
    Description    string         `json:"description,omitempty"`
    CronExpression string         `json:"cron_expression" validate:"required,cron"`
    Timezone       string         `json:"timezone" validate:"timezone"`
    Parameters     map[string]any `json:"parameters,omitempty"`
    IsActive       *bool          `json:"is_active,omitempty"` // defaults to true
}

// UpdateScheduleRequest for PATCH /api/v1/schedules/{id}
type UpdateScheduleRequest struct {
    Name           *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
    Description    *string        `json:"description,omitempty"`
    CronExpression *string        `json:"cron_expression,omitempty" validate:"omitempty,cron"`
    Timezone       *string        `json:"timezone,omitempty" validate:"omitempty,timezone"`
    Parameters     map[string]any `json:"parameters,omitempty"`
    IsActive       *bool          `json:"is_active,omitempty"`
}

// ScheduleResponse for all schedule endpoints
type ScheduleResponse struct {
    *database.WorkflowSchedule
    WorkflowName  string `json:"workflow_name,omitempty"`
    NextRunHuman  string `json:"next_run_human,omitempty"` // "in 2 hours"
    LastRunStatus string `json:"last_run_status,omitempty"` // "success", "failed", "never"
}
```

#### 1.3 Route Registration

**File**: `api/main.go` (MODIFY)

Add schedule routes in the existing route registration block:

```go
// Schedule management
r.Post("/api/v1/workflows/{workflowID}/schedules", h.CreateSchedule)
r.Get("/api/v1/workflows/{workflowID}/schedules", h.ListWorkflowSchedules)
r.Get("/api/v1/schedules", h.ListAllSchedules)
r.Get("/api/v1/schedules/{scheduleID}", h.GetSchedule)
r.Patch("/api/v1/schedules/{scheduleID}", h.UpdateSchedule)
r.Delete("/api/v1/schedules/{scheduleID}", h.DeleteSchedule)
r.Post("/api/v1/schedules/{scheduleID}/trigger", h.TriggerSchedule)
r.Post("/api/v1/schedules/{scheduleID}/toggle", h.ToggleSchedule)
```

### Testing Seams (Phase 1)

| Seam | Interface | Mock Location |
|------|-----------|---------------|
| Repository | `ScheduleRepository` | `database/mock_repository.go` |
| Handler | HTTP handlers | `handlers/schedules_test.go` |

**Test file**: `api/handlers/schedules_test.go` (NEW)

Required test cases:
- Create schedule with valid cron expression
- Create schedule with invalid cron expression (400)
- Create schedule for non-existent workflow (404)
- List schedules with pagination
- Update schedule fields
- Delete schedule
- Toggle schedule active/inactive

---

## Phase 2: Scheduler Service

### Objective

Create a service that loads active schedules at startup, manages cron jobs, and triggers workflow executions.

### Files to Create

#### 2.1 Service Package Structure

```
api/services/scheduler/
├── doc.go              # Package documentation
├── interfaces.go       # Service interfaces (testing seam)
├── service.go          # SchedulerService implementation
├── cron.go             # Cron parsing and job management
├── notifier.go         # Notification dispatch
├── service_test.go     # Unit tests
└── README.md           # Architecture documentation
```

#### 2.2 Interfaces (Testing Seam)

**File**: `api/services/scheduler/interfaces.go` (NEW)

```go
package scheduler

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/vrooli/browser-automation-studio/database"
)

// SchedulerService manages cron-based workflow scheduling.
// The scheduler runs in-process and requires the API server to be running.
type SchedulerService interface {
    // Start loads active schedules and begins the cron scheduler.
    // Call this once during API server startup.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the scheduler, completing any in-flight executions.
    Stop(ctx context.Context) error

    // AddSchedule registers a new schedule with the cron system.
    // Called when a schedule is created via API.
    AddSchedule(schedule *database.WorkflowSchedule) error

    // UpdateSchedule modifies an existing schedule's cron expression or parameters.
    // Called when a schedule is updated via API.
    UpdateSchedule(schedule *database.WorkflowSchedule) error

    // RemoveSchedule unregisters a schedule from the cron system.
    // Called when a schedule is deleted or deactivated via API.
    RemoveSchedule(id uuid.UUID) error

    // TriggerNow executes a schedule immediately, outside its cron timing.
    // Used for manual triggers and testing.
    TriggerNow(ctx context.Context, id uuid.UUID) error

    // GetStatus returns scheduler health and job statistics.
    GetStatus() SchedulerStatus
}

// SchedulerStatus provides runtime information about the scheduler.
type SchedulerStatus struct {
    Running       bool      `json:"running"`
    ActiveJobs    int       `json:"active_jobs"`
    NextExecution time.Time `json:"next_execution,omitempty"`
    LastExecution time.Time `json:"last_execution,omitempty"`
    ErrorCount    int       `json:"error_count"`
}

// ExecutionTrigger is called when a scheduled job fires.
// This abstraction allows testing without running actual workflows.
type ExecutionTrigger interface {
    Execute(ctx context.Context, workflowID uuid.UUID, parameters map[string]any, triggerType string) (*database.Execution, error)
}

// NotificationSender dispatches notifications about schedule events.
type NotificationSender interface {
    SendSuccess(scheduleID uuid.UUID, scheduleName string, executionID uuid.UUID, duration time.Duration)
    SendFailure(scheduleID uuid.UUID, scheduleName string, err error)
    SendMissed(scheduleID uuid.UUID, scheduleName string, missedAt time.Time)
}
```

#### 2.3 Service Implementation

**File**: `api/services/scheduler/service.go` (NEW)

Key implementation details:

```go
package scheduler

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/robfig/cron/v3"
    "github.com/sirupsen/logrus"
    "github.com/vrooli/browser-automation-studio/database"
)

// schedulerService implements SchedulerService.
type schedulerService struct {
    repo       database.Repository
    executor   ExecutionTrigger
    notifier   NotificationSender
    cron       *cron.Cron
    jobs       map[uuid.UUID]cron.EntryID
    mu         sync.RWMutex
    log        *logrus.Logger
    ctx        context.Context
    cancel     context.CancelFunc
}

// New creates a new scheduler service.
// Dependencies are injected for testability.
func New(
    repo database.Repository,
    executor ExecutionTrigger,
    notifier NotificationSender,
    log *logrus.Logger,
) SchedulerService {
    return &schedulerService{
        repo:     repo,
        executor: executor,
        notifier: notifier,
        jobs:     make(map[uuid.UUID]cron.EntryID),
        log:      log,
        cron:     cron.New(cron.WithSeconds()), // 6-field cron for precision
    }
}
```

**Cron library**: Use `github.com/robfig/cron/v3` (mature, well-tested, supports timezones)

#### 2.4 Integration with Main

**File**: `api/main.go` (MODIFY)

```go
// In server setup, after repository and services are initialized:
schedulerSvc := scheduler.New(repo, executionService, notificationSvc, logger)
if err := schedulerSvc.Start(ctx); err != nil {
    logger.WithError(err).Fatal("Failed to start scheduler")
}
defer schedulerSvc.Stop(ctx)

// Pass to handler for trigger/status endpoints
h := handlers.NewHandler(
    // ... existing dependencies ...
    schedulerSvc,
)
```

### Testing Seams (Phase 2)

| Seam | Interface | Mock |
|------|-----------|------|
| Execution | `ExecutionTrigger` | `mock_executor.go` |
| Notifications | `NotificationSender` | `mock_notifier.go` |
| Cron | `cron.Cron` | Use real cron with short intervals in tests |

**Test file**: `api/services/scheduler/service_test.go` (NEW)

Required test cases:
- Start scheduler with existing active schedules
- Add schedule updates cron system
- Remove schedule clears cron entry
- Cron fires at correct time (use 1-second cron in test)
- Concurrent schedule modifications are thread-safe
- Graceful shutdown completes in-flight executions

---

## Phase 3: UI Components

### Objective

Build the user interface for managing schedules, following existing patterns in `ui/src/features/`.

### Files to Create

#### 3.1 Feature Structure

```
ui/src/features/schedules/
├── index.ts                    # Public exports
├── SchedulesTab.tsx            # Main tab component (for Dashboard)
├── SchedulesList.tsx           # List of schedule cards
├── ScheduleCard.tsx            # Individual schedule display
├── ScheduleModal.tsx           # Create/edit modal
├── ScheduleForm.tsx            # Form fields component
├── CronExpressionInput.tsx     # Cron builder/validator
├── hooks/
│   └── useSchedules.ts         # Data fetching hook
└── __tests__/
    ├── SchedulesTab.test.tsx
    ├── ScheduleModal.test.tsx
    └── CronExpressionInput.test.tsx
```

#### 3.2 Store

**File**: `ui/src/stores/scheduleStore.ts` (NEW)

```typescript
import { create } from 'zustand';
import { api } from '@/shared/api';

interface Schedule {
    id: string;
    workflow_id: string;
    name: string;
    description?: string;
    cron_expression: string;
    timezone: string;
    is_active: boolean;
    parameters?: Record<string, unknown>;
    next_run_at?: string;
    last_run_at?: string;
    workflow_name?: string;
    next_run_human?: string;
    last_run_status?: 'success' | 'failed' | 'never';
}

interface ScheduleStore {
    schedules: Schedule[];
    isLoading: boolean;
    error: string | null;

    // Actions
    fetchSchedules: (workflowId?: string) => Promise<void>;
    createSchedule: (workflowId: string, data: CreateScheduleInput) => Promise<Schedule>;
    updateSchedule: (id: string, data: UpdateScheduleInput) => Promise<void>;
    deleteSchedule: (id: string) => Promise<void>;
    toggleSchedule: (id: string) => Promise<void>;
    triggerSchedule: (id: string) => Promise<void>;
}

export const useScheduleStore = create<ScheduleStore>((set, get) => ({
    // ... implementation
}));
```

#### 3.3 Dashboard Integration

**File**: `ui/src/features/dashboard/Dashboard.tsx` (MODIFY)

Add "Schedules" tab alongside existing tabs (Home, Projects, Workflows, Executions, Exports):

```typescript
// In Dashboard tabs array
{
    id: 'schedules',
    label: 'Schedules',
    icon: CalendarClock,
    component: SchedulesTab,
}
```

#### 3.4 Cron Expression UI

The `CronExpressionInput` component should provide:
- Preset options: "Every hour", "Daily at 9am", "Weekly on Monday", etc.
- Custom mode with cron syntax input
- Human-readable preview of next 5 run times
- Validation feedback

### UI Testing Strategy

- **Unit tests**: Vitest + React Testing Library for component logic
- **Integration tests**: Test store ↔ API interaction with MSW mocks
- **E2E tests**: Playwright playbook for full schedule creation flow

---

## Phase 4: Desktop Tray Integration

### Objective

Enhance scenario-to-desktop templates to support dynamic tray updates, and enable tray mode for browser-automation-studio when schedules exist.

### 4.1 Shared Template Changes (scenario-to-desktop)

**File**: `scenarios/scenario-to-desktop/templates/vanilla/main.ts` (MODIFY)

Add IPC handlers for dynamic tray updates:

```typescript
// Add near other ipcMain handlers (around line 800+)

// === TRAY DYNAMIC UPDATES ===
// These handlers allow the renderer to update tray state based on app-specific logic

ipcMain.handle("tray:update-tooltip", (_event, tooltip: string) => {
    if (tray) {
        tray.setToolTip(tooltip);
    }
    return { success: true };
});

ipcMain.handle("tray:set-badge", (_event, count: number) => {
    // macOS dock badge
    if (process.platform === "darwin") {
        app.dock?.setBadge(count > 0 ? String(count) : "");
    }
    // Windows/Linux: Update tray icon (would need icon variants)
    return { success: true };
});

ipcMain.handle("tray:update-context-menu", (_event, items: TrayMenuItem[]) => {
    if (!tray) return { success: false };

    const dynamicItems: Electron.MenuItemConstructorOptions[] = items.map(item => ({
        label: item.label,
        enabled: item.enabled ?? true,
        click: () => {
            mainWindow?.webContents.send("tray-menu-action", item.action);
        }
    }));

    const baseMenu: Electron.MenuItemConstructorOptions[] = [
        { type: "separator" },
        {
            label: `Show ${APP_CONFIG.APP_DISPLAY_NAME}`,
            click: () => mainWindow?.show()
        },
        {
            label: "Quit",
            click: () => app.quit()
        }
    ];

    tray.setContextMenu(Menu.buildFromTemplate([...dynamicItems, ...baseMenu]));
    return { success: true };
});

interface TrayMenuItem {
    label: string;
    action: string;
    enabled?: boolean;
}
```

**File**: `scenarios/scenario-to-desktop/templates/vanilla/preload.ts` (MODIFY)

Expose tray methods to renderer:

```typescript
// Add to desktop API object
tray: {
    updateTooltip: (tooltip: string) => ipcRenderer.invoke("tray:update-tooltip", tooltip),
    setBadge: (count: number) => ipcRenderer.invoke("tray:set-badge", count),
    updateContextMenu: (items: TrayMenuItem[]) => ipcRenderer.invoke("tray:update-context-menu", items),
},
```

### 4.2 Scenario-Specific Configuration

**File**: `scenarios/browser-automation-studio/.vrooli/desktop-config.json` (NEW)

```json
{
    "proxy_url": "http://localhost:${UI_PORT}/",
    "server_url": "http://localhost:${UI_PORT}/",
    "api_url": "http://localhost:${API_PORT}/api",
    "app_display_name": "Vrooli Ascension",
    "deployment_mode": "bundled",
    "features": {
        "systemTray": true,
        "splash": true,
        "autoUpdater": true,
        "singleInstance": true,
        "menu": true,
        "devTools": true
    }
}
```

### 4.3 UI Tray Utilities

**File**: `ui/src/lib/desktop/tray.ts` (NEW)

```typescript
/**
 * Desktop tray integration utilities.
 * Only active when running in Electron (window.desktop available).
 */

interface TrayMenuItem {
    label: string;
    action: string;
    enabled?: boolean;
}

/**
 * Updates the system tray tooltip to show schedule information.
 * Called whenever schedules change.
 */
export async function updateTrayWithSchedules(schedules: Schedule[]): Promise<void> {
    if (!window.desktop?.tray) return;

    const activeSchedules = schedules.filter(s => s.is_active);
    const nextRun = activeSchedules
        .map(s => s.next_run_at)
        .filter(Boolean)
        .sort()[0];

    // Update tooltip
    const tooltip = nextRun
        ? `Vrooli Ascension\nNext scheduled run: ${formatRelativeTime(nextRun)}`
        : 'Vrooli Ascension';

    await window.desktop.tray.updateTooltip(tooltip);

    // Update context menu with upcoming schedules
    const menuItems: TrayMenuItem[] = activeSchedules
        .slice(0, 3)
        .map(s => ({
            label: `${s.name} - ${formatRelativeTime(s.next_run_at)}`,
            action: `view-schedule:${s.id}`,
        }));

    if (menuItems.length > 0) {
        menuItems.push({ label: 'View all schedules...', action: 'open-schedules-tab' });
    }

    await window.desktop.tray.updateContextMenu(menuItems);
}

/**
 * Sets the dock/taskbar badge for pending schedule count.
 */
export async function updateTrayBadge(pendingCount: number): Promise<void> {
    if (!window.desktop?.tray) return;
    await window.desktop.tray.setBadge(pendingCount);
}
```

---

## Phase 5: Notifications

### Objective

Notify users when scheduled workflows complete or fail, using desktop notifications when available.

### 5.1 WebSocket Events

**File**: `api/websocket/events.go` (MODIFY)

Add schedule-related event types:

```go
const (
    // ... existing events ...
    EventScheduleExecutionStarted   = "schedule.execution.started"
    EventScheduleExecutionCompleted = "schedule.execution.completed"
    EventScheduleExecutionFailed    = "schedule.execution.failed"
)

type ScheduleExecutionEvent struct {
    ScheduleID   uuid.UUID `json:"schedule_id"`
    ScheduleName string    `json:"schedule_name"`
    ExecutionID  uuid.UUID `json:"execution_id,omitempty"`
    Status       string    `json:"status"` // "started", "completed", "failed"
    DurationMs   int64     `json:"duration_ms,omitempty"`
    Error        string    `json:"error,omitempty"`
}
```

### 5.2 Notification Service

**File**: `api/services/scheduler/notifier.go` (NEW)

```go
package scheduler

// wsNotifier implements NotificationSender using WebSocket broadcast.
type wsNotifier struct {
    hub wsHub.HubInterface
    log *logrus.Logger
}

func NewWSNotifier(hub wsHub.HubInterface, log *logrus.Logger) NotificationSender {
    return &wsNotifier{hub: hub, log: log}
}

func (n *wsNotifier) SendSuccess(scheduleID uuid.UUID, scheduleName string, executionID uuid.UUID, duration time.Duration) {
    n.hub.Broadcast(wsHub.Message{
        Type: "schedule.execution.completed",
        Payload: map[string]any{
            "schedule_id":   scheduleID,
            "schedule_name": scheduleName,
            "execution_id":  executionID,
            "duration_ms":   duration.Milliseconds(),
            "status":        "completed",
        },
    })
}
```

### 5.3 UI Notification Handler

**File**: `ui/src/features/schedules/hooks/useScheduleNotifications.ts` (NEW)

```typescript
/**
 * Hook to handle schedule execution notifications.
 * Shows desktop notifications and updates UI state.
 */
export function useScheduleNotifications(): void {
    const { refetchSchedules } = useScheduleStore();

    useWebSocketEvent('schedule.execution.completed', (data) => {
        // Show desktop notification
        window.desktop?.notify(
            `${data.schedule_name} completed`,
            `Duration: ${formatDuration(data.duration_ms)}`
        );

        // Refresh schedule list to update last_run_at
        refetchSchedules();
    });

    useWebSocketEvent('schedule.execution.failed', (data) => {
        window.desktop?.notify(
            `${data.schedule_name} failed`,
            data.error || 'Unknown error'
        );

        refetchSchedules();
    });
}
```

---

## Testing Strategy

### Unit Tests

| Component | Test File | Coverage Target |
|-----------|-----------|-----------------|
| Repository | `database/repository_schedules_test.go` | CRUD operations, edge cases |
| Handlers | `handlers/schedules_test.go` | HTTP contracts, validation |
| Scheduler Service | `services/scheduler/service_test.go` | Job management, concurrency |
| UI Store | `stores/scheduleStore.test.ts` | State management |
| UI Components | `features/schedules/__tests__/` | Render, interactions |

### Integration Tests

| Scope | Description |
|-------|-------------|
| API → Scheduler | Creating schedule via API adds cron job |
| Scheduler → Execution | Cron trigger executes workflow |
| UI → API | Full CRUD flow with mock server |

### E2E Tests

**Playbook**: `test/playbooks/capabilities/07-scheduling/` (NEW)

```
07-scheduling/
├── 01-api/
│   ├── schedule-crud.json
│   └── schedule-trigger.json
└── 02-ui/
    ├── schedule-create-flow.json
    ├── schedule-edit-flow.json
    └── schedule-list-view.json
```

### Test Data Fixtures

**File**: `test/fixtures/schedules.json` (NEW)

```json
{
    "valid_daily_schedule": {
        "name": "Daily Health Check",
        "cron_expression": "0 9 * * *",
        "timezone": "America/New_York"
    },
    "valid_weekly_schedule": {
        "name": "Weekly Report",
        "cron_expression": "0 8 * * 1",
        "timezone": "UTC"
    },
    "invalid_cron": {
        "name": "Bad Schedule",
        "cron_expression": "not-a-cron"
    }
}
```

---

## Documentation Updates

### Files to Update

| File | Changes |
|------|---------|
| `api/docs/API.md` | Add schedule endpoints |
| `api/services/README.md` | Add scheduler to service diagram |
| `ui/src/features/README.md` | Document schedules feature |
| `PRD.md` | Update feature list |
| `docs/ENVIRONMENT.md` | Add scheduler-related env vars |

### New Documentation

**File**: `api/services/scheduler/README.md` (NEW)

Document:
- Architecture overview
- Cron expression format
- Timezone handling
- Missed execution behavior
- Testing approach

---

## File Manifest

### New Files

| Path | Description |
|------|-------------|
| `api/database/repository_schedules.go` | Schedule repository methods |
| `api/handlers/schedules.go` | Schedule HTTP handlers |
| `api/handlers/schedules_test.go` | Handler tests |
| `api/services/scheduler/doc.go` | Package documentation |
| `api/services/scheduler/interfaces.go` | Service interfaces |
| `api/services/scheduler/service.go` | Scheduler implementation |
| `api/services/scheduler/cron.go` | Cron job management |
| `api/services/scheduler/notifier.go` | Notification dispatch |
| `api/services/scheduler/service_test.go` | Service tests |
| `api/services/scheduler/README.md` | Architecture docs |
| `ui/src/features/schedules/index.ts` | Feature exports |
| `ui/src/features/schedules/SchedulesTab.tsx` | Main tab |
| `ui/src/features/schedules/SchedulesList.tsx` | List component |
| `ui/src/features/schedules/ScheduleCard.tsx` | Card component |
| `ui/src/features/schedules/ScheduleModal.tsx` | Create/edit modal |
| `ui/src/features/schedules/ScheduleForm.tsx` | Form fields |
| `ui/src/features/schedules/CronExpressionInput.tsx` | Cron builder |
| `ui/src/features/schedules/hooks/useSchedules.ts` | Data hook |
| `ui/src/features/schedules/hooks/useScheduleNotifications.ts` | WS hook |
| `ui/src/stores/scheduleStore.ts` | Zustand store |
| `ui/src/lib/desktop/tray.ts` | Tray utilities |
| `.vrooli/desktop-config.json` | Desktop feature config |
| `test/playbooks/capabilities/07-scheduling/` | E2E playbooks |
| `test/fixtures/schedules.json` | Test fixtures |

### Modified Files

| Path | Changes |
|------|-------------|
| `api/main.go` | Register scheduler service and routes |
| `api/database/repository.go` | Add ScheduleRepository to interface |
| `api/handlers/handler.go` | Add scheduler service dependency |
| `api/websocket/events.go` | Add schedule event types |
| `ui/src/features/dashboard/Dashboard.tsx` | Add Schedules tab |
| `ui/src/App.tsx` | Add notification listener |
| `scenarios/scenario-to-desktop/templates/vanilla/main.ts` | Tray IPC handlers |
| `scenarios/scenario-to-desktop/templates/vanilla/preload.ts` | Tray API exposure |

---

## Reference Links

### Existing Code Patterns

| Pattern | Reference File |
|---------|----------------|
| Service interfaces | `api/services/workflow/interfaces.go` |
| Handler structure | `api/handlers/workflows.go` |
| Repository pattern | `api/database/repository.go` |
| Feature folder structure | `ui/src/features/execution/` |
| Zustand store | `ui/src/stores/workflowStore.ts` |
| Desktop API usage | `ui/src/lib/storage/desktopStorage.ts` |

### External Dependencies

| Dependency | Purpose | Docs |
|------------|---------|------|
| `github.com/robfig/cron/v3` | Cron scheduling | https://pkg.go.dev/github.com/robfig/cron/v3 |
| scenario-to-desktop | Electron templates | `scenarios/scenario-to-desktop/README.md` |
| Bundle manifest | Desktop config | `docs/deployment/bundle-schema.desktop.v0.1.json` |

### Related Plans

| Document | Relevance |
|----------|-----------|
| `docs/plans/bundled-desktop-runtime-plan.md` | Desktop architecture context |
| `scenarios/browser-automation-studio/PRD.md` | Feature requirements |
| `scenarios/browser-automation-studio/api/services/README.md` | Service architecture |

---

## Acceptance Criteria

### Phase 1 Complete When:
- [ ] Schedule CRUD endpoints return correct responses
- [ ] Invalid cron expressions rejected with 400
- [ ] Schedules persist across API restarts
- [ ] All handler tests pass

### Phase 2 Complete When:
- [ ] Scheduler loads active schedules on startup
- [ ] Cron jobs fire at correct times (tested with 1-second intervals)
- [ ] Adding/removing schedules updates cron system
- [ ] Concurrent operations are thread-safe
- [ ] Graceful shutdown works correctly

### Phase 3 Complete When:
- [ ] Schedules tab displays in Dashboard
- [ ] Create schedule modal works with presets and custom cron
- [ ] Schedule list shows correct status and timing
- [ ] Toggle active/inactive works
- [ ] Manual trigger works

### Phase 4 Complete When:
- [ ] Tray enabled when app has active schedules
- [ ] Tray tooltip shows next scheduled run
- [ ] Tray context menu shows upcoming schedules
- [ ] Close-to-tray keeps scheduler running

### Phase 5 Complete When:
- [ ] Desktop notification on schedule completion
- [ ] Desktop notification on schedule failure
- [ ] UI updates when schedule executes

---

## Open Questions

1. **Missed executions**: If app was closed during scheduled time, should we:
   - Run immediately on next app open? (could be unexpected)
   - Log as missed and wait for next scheduled time? (current plan)
   - Let user configure per-schedule?

2. **Execution overlap**: If a scheduled workflow is still running when next trigger fires:
   - Skip the new trigger? (current plan)
   - Queue it?
   - Run in parallel?

3. **Timezone changes**: If user changes system timezone, should schedules:
   - Keep their original timezone?
   - Adapt to new system timezone?

---

## Changelog

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-11 | Claude | Initial plan creation |
