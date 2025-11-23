# Ecosystem Manager UI Migration Plan
## From Vanilla JS to React + TypeScript + Tailwind

**Created:** 2025-11-23
**Status:** Phase 0 - Planning
**Migration Strategy:** Phased Incremental Migration (Hybrid Approach)

---

## Executive Summary

This document provides a comprehensive plan for migrating the ecosystem-manager UI from vanilla JavaScript to the modern react-vite template stack. The migration will be executed in phases over multiple sessions, maintaining a working UI throughout the process.

### Current State
- **Framework:** Vanilla JavaScript (ES modules)
- **Styling:** Monolithic CSS (6,919 LOC)
- **Architecture:** Single class with modules (total ~10,000 LOC)
- **Build:** Vite
- **Dependencies:** sortablejs, @vrooli/api-base, @vrooli/iframe-bridge

### Target State
- **Framework:** React 18 + TypeScript
- **Styling:** Tailwind CSS + shadcn/ui components
- **Architecture:** Component-based with hooks
- **State Management:** @tanstack/react-query + React Context
- **Build:** Vite
- **Dependencies:** All current + lucide-react, clsx, tailwind-merge

### Migration Benefits
1. **Maintainability:** Component-based architecture vs 5,275 LOC monolithic file
2. **Type Safety:** TypeScript prevents runtime errors
3. **Developer Experience:** Hot reload, autocomplete, refactoring support
4. **Styling:** Utility-first CSS vs 6,919 LOC style.css
5. **Ecosystem Alignment:** Matches other scenarios (prd-control-tower, scenario-dependency-analyzer, etc.)
6. **Future Features:** Easier to add complex UI patterns

---

## Phase 0: Planning & Infrastructure (CURRENT PHASE)

### Objectives
✅ Document all existing features
✅ Map API surface
✅ Design component hierarchy
✅ Plan state management strategy
✅ Create migration roadmap
⏳ Set up parallel infrastructure

### Feature Inventory

#### 1. Kanban Board (Core Feature)
**Status Columns (7 total):**
- `pending` - Tasks waiting to start (visible by default)
- `in-progress` - Currently being worked on (labeled "Active" in UI, visible by default)
- `completed` - Implementation finished, awaiting verification (visible by default)
- `completed-finalized` - Fully delivered and closed (labeled "Finished" in UI, visible by default)
- `failed` - Attempts ended unsuccessfully (visible by default)
- `failed-blocked` - Stuck on external issues (labeled "Blocked" in UI, visible by default)
- `archived` - Hidden from active workflows (**hidden by default**, user can toggle visibility)

**Notes:**
- The `review` status exists in the JavaScript state but is not rendered/used in the current UI.
- Only the `archived` column is hidden by default; users can show it via the column visibility toggles in the filter panel.

**Features:**
- Drag-and-drop task movement between columns
- Column hide/show toggle (archived column hidden by default)
- Task count badges
- Real-time updates via WebSocket
- Horizontal scroll with touch/mouse support
- Task card rendering with status-specific styling
- Column visibility state persisted in AppStateContext

#### 2. Task Management

**Task Card Components:**
- Task ID with copy button
- Title and type/operation badges
- Priority indicator
- Target (for improver tasks)
- Notes preview (first 150 chars)
- Auto Steer profile indicator
- Execution count badge
- Elapsed time counter (for in-progress tasks)
- Action buttons (view details, delete)

**Task Creation Modal:**
- Type selector (resource/scenario)
- Operation selector (generator/improver)
- Title field with autofill support
- Priority dropdown
- Auto Steer profile selector with preview
- Target multi-select (for improver operations)
- Notes textarea
- Dynamic form validation

**Task Details Modal:**
- 5 tabs: Details, Logs, Prompt, Recycler History, Executions
- Editable fields (title, priority, notes, auto-steer profile, auto-requeue)
- Status transitions
- Execution history viewer
- Prompt preview
- Log viewer with auto-scroll
- Archive/delete actions
- Save changes functionality

#### 3. Filtering & Search

**Filter Panel:**
- Text search (across title and ID)
- Type filter (resource/scenario/all)
- Operation filter (generator/improver/all)
- Priority filter (critical/high/medium/low/all)
- Column visibility toggles (archived column hidden by default)
- Clear all filters button
- Filter state persisted in URL query params
- Active filter count badge

**Filter Behavior:**
- Combined AND logic
- Real-time application
- URL state synchronization
- Session persistence

#### 4. Settings Modal (7 Tabs)

**Tab 1: Processor Settings**
- Concurrent task slots (slider: 1-5)
- Refresh interval (number input: 5-300s)
- Max tasks (disabled - queue starvation prevention)
- Processor active toggle

**Tab 2: Agent Settings**
- Maximum turns (slider: 5-100, dynamic from backend)
- Allowed tools (text input, comma-separated)
- Skip permissions checkbox
- Task timeout (slider: 5-240 min, dynamic)
- Idle timeout cap (slider: 2-240 min)

**Tab 3: Display Settings**
- Theme selector (light/dark/auto)
- Live theme preview
- Condensed mode toggle

**Tab 4: Prompt Tester**
- Task type/operation selectors
- Title, priority, notes inputs
- Preview prompt button
- Assembled prompt display
- Token count and metadata
- Section breakdown viewer
- Copy to clipboard button

**Tab 5: Rate Limits**
- Current rate limit status display
- Reset rate limit button
- Countdown timer (when rate limited)

**Tab 6: Recycler Settings**
- Enabled for selector (off/resources/scenarios/both)
- Recycle interval (30-1800s)
- Model provider (ollama/openrouter)
- Model name selector with custom input
- Completion threshold (1-10)
- Failure threshold (1-10)
- Info cards explaining recycler behavior

**Tab 7: Auto Steer**
- Profile list with cards
- Template gallery
- Profile editor (separate modal)
- Create/edit/duplicate/delete actions

#### 5. Auto Steer Profile Management

**Profile Features:**
- CRUD operations
- Phase management
- Condition builder
- Tag system
- Template-based creation
- Profile preview in task creation

**Condition Builder:**
- Visual tree editor
- AND/OR logic groups
- Field, operator, value inputs
- Nested conditions
- JSON export/import

#### 6. Process Monitoring

**Features:**
- Running process count indicator
- Dropdown with process details
- Live elapsed time counters
- Per-task process info
- Terminate process action

#### 7. System Logs

**Log Viewer Modal:**
- Log level filter (all/info/warning/error)
- Real-time log streaming
- Auto-scroll toggle
- Timestamp display
- Level-based color coding

#### 8. Recycler Testing

**Recycler Testbed (in Settings):**
- Custom output text input
- Preset suite runner
- Test results table
- Performance metrics
- Success/failure indicators

#### 9. Execution History

**Features:**
- Per-task execution list
- Execution detail viewer
- Prompt file viewer
- Output file viewer
- Metadata display
- Filtering by status/date

#### 10. Floating Controls

**Control Bar:**
- Queue processor status/toggle button
- Processor remaining count badge
- Filter panel toggle with active count
- Next refresh countdown timer
- Process monitor dropdown
- System logs button
- Settings button
- New task button (primary action)

#### 11. Real-time Updates

**WebSocket Integration:**
- Task status changes
- Process start/complete notifications
- Queue status updates
- Rate limit notifications
- Automatic UI refresh

#### 12. Additional Features

**Toast Notifications:**
- Success/error/info/warning messages
- Auto-dismiss
- Positioned top-right

**Loading States:**
- Global loading overlay
- Skeleton screens
- Spinner indicators

**Rate Limit Handling:**
- Overlay with countdown
- Manual resume option
- Automatic pause/resume

**Keyboard Shortcuts:**
- `Ctrl+N` - New task
- `Ctrl+F` - Open filters
- `Ctrl+S` - Open settings (when in forms)
- `Esc` - Close modals/panels

**Responsive Design:**
- Mobile-optimized kanban (vertical scroll)
- Tablet breakpoints
- Desktop optimizations
- Touch-friendly controls

---

## API Surface Documentation

### HTTP Endpoints (via ApiClient)

#### Task Management
```typescript
GET    /api/tasks                    - List tasks (with filters)
GET    /api/tasks/{id}               - Get task details
POST   /api/tasks                    - Create task
PUT    /api/tasks/{id}               - Update task
DELETE /api/tasks/{id}               - Delete task
PUT    /api/tasks/{id}/status        - Update task status
GET    /api/tasks/{id}/logs          - Get task logs
GET    /api/tasks/{id}/prompt        - Get task prompt config
GET    /api/tasks/{id}/prompt/assembled - Get assembled prompt
GET    /api/tasks/active-targets     - Get active improvement targets
```

#### Queue Management
```typescript
GET  /api/queue/status              - Get processor status
POST /api/queue/trigger             - Trigger manual processing
POST /api/queue/start               - Start processor
POST /api/queue/stop                - Stop processor
POST /api/queue/reset-rate-limit    - Reset rate limit
GET  /api/processes/running         - Get running processes
POST /api/queue/processes/terminate - Terminate process
```

#### Settings
```typescript
GET  /api/settings                           - Get settings
PUT  /api/settings                           - Update settings
POST /api/settings/reset                     - Reset to defaults
GET  /api/settings/recycler/models?provider - Get recycler models
```

#### Discovery
```typescript
GET /api/resources              - List available resources
GET /api/scenarios              - List available scenarios
GET /api/resources/{name}/status - Get resource status
GET /api/scenarios/{name}/status - Get scenario status
GET /api/operations             - Get available operations
GET /api/categories             - Get available categories
```

#### Logs & Execution
```typescript
GET /api/logs?limit=500                              - Get system logs
GET /api/executions                                  - Get all execution history
GET /api/tasks/{id}/executions                       - Get task executions
GET /api/tasks/{id}/executions/{exec_id}/prompt     - Get execution prompt
GET /api/tasks/{id}/executions/{exec_id}/output     - Get execution output
GET /api/tasks/{id}/executions/{exec_id}/metadata   - Get execution metadata
```

#### Recycler & Testing
```typescript
POST /api/recycler/test           - Test recycler
POST /api/recycler/preview-prompt - Preview recycler prompt
POST /api/prompt-viewer           - Preview assembled prompt
```

#### Auto Steer
```typescript
GET    /api/auto-steer/profiles      - List profiles
POST   /api/auto-steer/profiles      - Create profile
GET    /api/auto-steer/profiles/{id} - Get profile
PUT    /api/auto-steer/profiles/{id} - Update profile
DELETE /api/auto-steer/profiles/{id} - Delete profile
GET    /api/auto-steer/templates     - List templates
```

#### Maintenance
```typescript
POST /api/maintenance/state - Set maintenance mode
```

### WebSocket Events

**Client → Server:** (Currently minimal, mostly receive-only)
```typescript
// Connection established automatically
```

**Server → Client:**
```typescript
{
  type: "task_status_update",
  task_id: string,
  status: TaskStatus,
  ...metadata
}

{
  type: "process_started",
  task_id: string,
  process_id: string,
  agent_id: string,
  start_time: string
}

{
  type: "process_completed",
  task_id: string,
  ...metadata
}

{
  type: "queue_status_update",
  active: boolean,
  slots_used: number,
  ...metadata
}

{
  type: "rate_limit_notification",
  retry_after: number,
  ...metadata
}
```

---

## React Component Hierarchy

```
App.tsx
├── Providers
│   ├── QueryClientProvider (@tanstack/react-query)
│   ├── AppStateProvider (React Context)
│   └── ThemeProvider (React Context)
│
├── Layout
│   ├── FloatingControls
│   │   ├── ProcessorStatusButton
│   │   ├── FilterToggleButton
│   │   ├── RefreshCountdown
│   │   ├── ProcessMonitor
│   │   │   ├── ProcessMonitorToggle
│   │   │   └── ProcessMonitorDropdown
│   │   │       └── ProcessCard[]
│   │   ├── LogsButton
│   │   ├── SettingsButton
│   │   └── NewTaskButton
│   │
│   └── ToastContainer
│       └── Toast[]
│
├── KanbanBoard
│   └── KanbanColumn[] (7 columns)
│       ├── ColumnHeader
│       │   ├── ColumnTitle
│       │   ├── TaskCount
│       │   └── HideColumnButton
│       └── ColumnContent
│           └── TaskCard[]
│               ├── TaskCardHeader
│               │   ├── TaskId (with copy)
│               │   ├── TaskBadges
│               │   └── PriorityIndicator
│               ├── TaskCardBody
│               │   ├── TaskTitle
│               │   ├── TaskTarget
│               │   ├── TaskNotes
│               │   └── AutoSteerIndicator
│               └── TaskCardFooter
│                   ├── ElapsedTimer
│                   ├── ExecutionCount
│                   └── TaskActions
│
├── FilterPanel
│   ├── SearchInput
│   ├── TypeFilter
│   ├── OperationFilter
│   ├── PriorityFilter
│   ├── ColumnVisibilityToggles
│   └── ClearFiltersButton
│
└── Modals
    ├── CreateTaskModal
    │   ├── TaskForm
    │   │   ├── TypeSelector
    │   │   ├── OperationSelector
    │   │   ├── TitleInput
    │   │   ├── PrioritySelect
    │   │   ├── AutoSteerProfileSelect
    │   │   │   └── ProfilePreview
    │   │   ├── TargetMultiSelect (improver only)
    │   │   └── NotesTextarea
    │   └── FormActions
    │
    ├── TaskDetailsModal
    │   ├── TabNavigation
    │   └── TabContent
    │       ├── DetailsTab
    │       │   ├── EditableFields
    │       │   └── StatusActions
    │       ├── LogsTab
    │       │   └── LogViewer
    │       ├── PromptTab
    │       │   └── PromptPreview
    │       ├── RecyclerTab
    │       │   └── RecyclerHistory
    │       └── ExecutionsTab
    │           ├── ExecutionList
    │           └── ExecutionDetailView
    │
    ├── SettingsModal
    │   ├── SettingsTabs (desktop)
    │   ├── SettingsMobileSelect
    │   ├── SettingsForm
    │   │   ├── ProcessorTab
    │   │   │   └── ProcessorFields
    │   │   ├── AgentTab
    │   │   │   └── AgentFields
    │   │   ├── DisplayTab
    │   │   │   └── DisplayFields
    │   │   ├── PromptTesterTab
    │   │   │   ├── TesterConfig
    │   │   │   └── TesterResults
    │   │   ├── RateLimitsTab
    │   │   │   └── RateLimitManager
    │   │   ├── RecyclerTab
    │   │   │   ├── RecyclerFields
    │   │   │   └── RecyclerTestbed
    │   │   └── AutoSteerTab
    │   │       ├── ProfileList
    │   │       │   └── ProfileCard[]
    │   │       └── TemplateGallery
    │   │           └── TemplateCard[]
    │   └── SettingsActions
    │
    ├── AutoSteerProfileEditorModal
    │   ├── ProfileBasicInfo
    │   ├── PhaseList
    │   │   └── PhaseEditor[]
    │   ├── TagEditor
    │   └── ProfileActions
    │
    ├── ConditionBuilderModal
    │   ├── ConditionTree
    │   │   └── ConditionNode[] (recursive)
    │   │       ├── LogicOperator
    │   │       ├── FieldSelect
    │   │       ├── OperatorSelect
    │   │       ├── ValueInput
    │   │       └── AddRemoveButtons
    │   └── BuilderActions
    │
    ├── SystemLogsModal
    │   ├── LogLevelFilter
    │   ├── LogViewer
    │   │   └── LogEntry[]
    │   └── AutoScrollToggle
    │
    └── LoadingOverlay
        └── Spinner
```

### Shared/Reusable Components (shadcn/ui + custom)

```
components/ui/
├── button.tsx                  - shadcn button
├── input.tsx                   - shadcn input
├── select.tsx                  - shadcn select
├── textarea.tsx                - shadcn textarea
├── checkbox.tsx                - shadcn checkbox
├── slider.tsx                  - shadcn slider
├── badge.tsx                   - shadcn badge
├── card.tsx                    - shadcn card
├── dialog.tsx                  - shadcn dialog (modal base)
├── tabs.tsx                    - shadcn tabs
├── tooltip.tsx                 - shadcn tooltip
└── toast.tsx                   - shadcn toast

components/custom/
├── Modal.tsx                   - Base modal wrapper
├── TagMultiSelect.tsx          - Target multi-selector
├── IconButton.tsx              - Icon-only button variant
├── LoadingSpinner.tsx          - Loading indicator
├── EmptyState.tsx              - Empty list placeholder
├── ErrorBoundary.tsx           - Error boundary wrapper
├── Kbd.tsx                     - Keyboard shortcut display
└── StatusBadge.tsx             - Status-colored badge
```

---

## State Management Strategy

### Global State (React Context + @tanstack/react-query)

#### 1. AppStateContext (src/state/AppContext.tsx)
```typescript
interface AppState {
  // Filter state
  filters: {
    search: string;
    type: TaskType | '';
    operation: OperationType | '';
    priority: Priority | '';
  };

  // Column visibility (archived column is hidden by default)
  columnVisibility: Record<TaskStatus, boolean>;

  // UI state
  isFilterPanelOpen: boolean;
  activeModal: string | null;

  // Settings (cached from react-query)
  cachedSettings: Settings | null;
}

// Default column visibility state
const defaultColumnVisibility = {
  pending: true,
  'in-progress': true,
  completed: true,
  'completed-finalized': true,
  failed: true,
  'failed-blocked': true,
  archived: false, // Hidden by default
};
```

#### 2. ThemeContext (src/state/ThemeContext.tsx)
```typescript
interface ThemeState {
  theme: 'light' | 'dark' | 'auto';
  resolvedTheme: 'light' | 'dark';
  setTheme: (theme: Theme) => void;
}
```

#### 3. WebSocketContext (src/state/WebSocketContext.tsx)
```typescript
interface WebSocketState {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  send: (message: unknown) => void;
}
```

### Server State (@tanstack/react-query)

#### Query Keys Structure
```typescript
// src/lib/queryKeys.ts
export const queryKeys = {
  tasks: {
    all: ['tasks'] as const,
    lists: () => [...queryKeys.tasks.all, 'list'] as const,
    list: (filters: TaskFilters) => [...queryKeys.tasks.lists(), filters] as const,
    details: () => [...queryKeys.tasks.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.tasks.details(), id] as const,
    logs: (id: string) => [...queryKeys.tasks.detail(id), 'logs'] as const,
    prompt: (id: string) => [...queryKeys.tasks.detail(id), 'prompt'] as const,
    executions: (id: string) => [...queryKeys.tasks.detail(id), 'executions'] as const,
  },
  settings: ['settings'] as const,
  processes: ['processes'] as const,
  autoSteer: {
    profiles: ['autoSteer', 'profiles'] as const,
    templates: ['autoSteer', 'templates'] as const,
  },
  // ... etc
};
```

#### Key Queries
```typescript
// Tasks
useQuery({ queryKey: queryKeys.tasks.list(filters), queryFn: () => api.getTasks(filters) })

// Task details
useQuery({ queryKey: queryKeys.tasks.detail(id), queryFn: () => api.getTask(id) })

// Settings
useQuery({ queryKey: queryKeys.settings, queryFn: () => api.getSettings() })

// Running processes
useQuery({
  queryKey: queryKeys.processes,
  queryFn: () => api.getRunningProcesses(),
  refetchInterval: 5000 // Poll every 5s
})
```

#### Key Mutations
```typescript
// Create task
useMutation({
  mutationFn: (data: CreateTaskInput) => api.createTask(data),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all })
})

// Update task
useMutation({
  mutationFn: ({ id, updates }: UpdateTaskInput) => api.updateTask(id, updates),
  onSuccess: (_, variables) => {
    queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(variables.id) })
    queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() })
  }
})

// Save settings
useMutation({
  mutationFn: (settings: Settings) => api.updateSettings(settings),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.settings })
})
```

### Local Component State (useState/useReducer)

Use local state for:
- Form inputs (controlled components)
- Modal open/close
- Accordion/dropdown expand/collapse
- Temporary UI state (e.g., "show more")

---

## Migration Phases Breakdown

### ✅ Phase 0: Planning & Infrastructure (CURRENT)
**Estimated Time:** 1-2 sessions
**Status:** In Progress

**Deliverables:**
- [x] Feature inventory (this document)
- [x] API surface documentation
- [x] Component hierarchy design
- [x] State management strategy
- [ ] Parallel infrastructure setup
  - [ ] Create `ui-new/` directory
  - [ ] Copy react-vite template
  - [ ] Configure Vite for dual UIs
  - [ ] Add UI toggle mechanism (e.g., `/new` route or feature flag)
  - [ ] Set up basic layout shell
  - [ ] Install dependencies
  - [ ] Configure Tailwind
  - [ ] Set up shadcn/ui

**Exit Criteria:**
- Working react-vite skeleton accessible at `/new` or via feature flag
- Both old and new UIs can be accessed
- Migration plan approved

---

### Phase 1: Core Kanban & Task Cards
**Estimated Time:** 2-3 sessions
**Status:** Not Started

**Objectives:**
- Implement basic kanban board layout
- Create task card components
- Implement drag-and-drop
- Integrate with task list API

**Tasks:**
1. Create KanbanBoard component with 7 columns
2. Implement KanbanColumn component
3. Build TaskCard component with all sub-components
4. Integrate @dnd-kit/core for drag-and-drop
5. Connect to useQuery for task lists
6. Handle drag-drop mutations
7. Implement WebSocket integration for real-time updates
8. Add task count badges
9. Implement column hide/show (with archived column hidden by default)

**Components to Build:**
- `KanbanBoard.tsx`
- `KanbanColumn.tsx`
- `TaskCard.tsx`
- `TaskCardHeader.tsx`
- `TaskCardBody.tsx`
- `TaskCardFooter.tsx`
- `TaskBadges.tsx`
- `PriorityIndicator.tsx`
- `ElapsedTimer.tsx`

**State/Hooks:**
- `useTasks(filters)` - Query hook
- `useTaskDragDrop()` - Drag-drop logic
- `useWebSocket()` - WebSocket integration
- `useAppState()` - Column visibility

**API Integration:**
- `GET /api/tasks` with status filter
- `PUT /api/tasks/{id}/status` on drag-drop

**Testing:**
- [ ] Tasks load and render correctly
- [ ] Drag-drop updates task status
- [ ] Real-time updates work
- [ ] Column counts update
- [ ] Column hide/show works
- [ ] Archived column hidden by default
- [ ] Column visibility state persists

**Exit Criteria:**
- All 7 columns render with tasks
- Drag-drop works smoothly
- Real-time updates functional
- Basic styling matches design intent

---

### Phase 2: Task Creation & Details
**Estimated Time:** 2-3 sessions
**Status:** Not Started

**Objectives:**
- Build task creation modal
- Implement task details modal
- Handle form validation
- Integrate with task CRUD APIs

**Tasks:**

#### 2A: Create Task Modal
1. Build CreateTaskModal with shadcn Dialog
2. Implement TypeSelector (resource/scenario)
3. Implement OperationSelector (generator/improver)
4. Build form fields (title, priority, notes)
5. Implement AutoSteerProfileSelect with preview
6. Build TargetMultiSelect for improver operations
7. Add form validation (zod or react-hook-form)
8. Integrate createTask mutation
9. Handle success/error states

**Components:**
- `CreateTaskModal.tsx`
- `TaskForm.tsx`
- `TypeSelector.tsx`
- `OperationSelector.tsx`
- `AutoSteerProfileSelect.tsx`
- `ProfilePreview.tsx`
- `TargetMultiSelect.tsx`

#### 2B: Task Details Modal
1. Build TaskDetailsModal with tabs
2. Implement DetailsTab with editable fields
3. Build LogsTab with log viewer
4. Implement PromptTab with preview
5. Build RecyclerTab with history
6. Implement ExecutionsTab with history list
7. Add save/delete/archive actions
8. Handle auto-requeue toggle

**Components:**
- `TaskDetailsModal.tsx`
- `TaskDetailsTabs.tsx`
- `DetailsTab.tsx`
- `LogsTab.tsx`
- `PromptTab.tsx`
- `RecyclerTab.tsx`
- `ExecutionsTab.tsx`
- `ExecutionList.tsx`
- `ExecutionDetailView.tsx`

**State/Hooks:**
- `useCreateTask()` - Mutation hook
- `useUpdateTask()` - Mutation hook
- `useDeleteTask()` - Mutation hook
- `useTaskDetails(id)` - Query hook
- `useTaskLogs(id)` - Query hook
- `useTaskPrompt(id)` - Query hook
- `useTaskExecutions(id)` - Query hook
- `useAutoSteerProfiles()` - Query hook

**API Integration:**
- `POST /api/tasks`
- `GET /api/tasks/{id}`
- `PUT /api/tasks/{id}`
- `DELETE /api/tasks/{id}`
- `GET /api/tasks/{id}/logs`
- `GET /api/tasks/{id}/prompt/assembled`
- `GET /api/tasks/{id}/executions`

**Testing:**
- [ ] Task creation works for all types/operations
- [ ] Form validation prevents invalid submissions
- [ ] Target multi-select works correctly
- [ ] Task details load and display
- [ ] All tabs function correctly
- [ ] Edit/save/delete work
- [ ] Error handling works

**Exit Criteria:**
- Can create tasks via modal
- Can view and edit task details
- All tabs functional
- CRUD operations work correctly

---

### Phase 3: Filtering & Search
**Estimated Time:** 1-2 sessions
**Status:** Not Started

**Objectives:**
- Implement filter panel
- Add search functionality
- Sync filters with URL
- Apply filters to task lists

**Tasks:**
1. Build FilterPanel component
2. Implement SearchInput with debounce
3. Add TypeFilter dropdown
4. Add OperationFilter dropdown
5. Add PriorityFilter dropdown
6. Build ColumnVisibilityToggles (with archived column hidden by default)
7. Add ClearFiltersButton
8. Sync filter state with URL query params
9. Update task queries to use filters
10. Add active filter count badge

**Components:**
- `FilterPanel.tsx`
- `SearchInput.tsx`
- `FilterSelect.tsx` (reusable)
- `ColumnVisibilityToggles.tsx`
- `FilterActiveCount.tsx`

**State/Hooks:**
- `useFilters()` - Custom hook for filter state
- `useDebounce(value, delay)` - Debounce hook
- `useQueryParams()` - URL sync hook

**Testing:**
- [ ] Search filters tasks correctly
- [ ] Type/operation/priority filters work
- [ ] Combined filters work (AND logic)
- [ ] URL updates with filters
- [ ] Clear filters resets state
- [ ] Active count displays correctly
- [ ] Column visibility toggles work correctly
- [ ] Archived column hidden by default

**Exit Criteria:**
- All filters functional
- Search works with debounce
- URL state sync works
- Filter panel UX polished

---

### Phase 4: Floating Controls & Process Monitoring
**Estimated Time:** 1-2 sessions
**Status:** Not Started

**Objectives:**
- Build floating control bar
- Implement process monitor
- Add processor status toggle
- Implement refresh countdown

**Tasks:**
1. Build FloatingControls component
2. Implement ProcessorStatusButton with toggle
3. Add FilterToggleButton
4. Build RefreshCountdown timer
5. Implement ProcessMonitor dropdown
6. Build ProcessCard components
7. Add LogsButton
8. Add SettingsButton
9. Add NewTaskButton
10. Handle processor start/stop mutations

**Components:**
- `FloatingControls.tsx`
- `ProcessorStatusButton.tsx`
- `FilterToggleButton.tsx`
- `RefreshCountdown.tsx`
- `ProcessMonitor.tsx`
- `ProcessMonitorToggle.tsx`
- `ProcessMonitorDropdown.tsx`
- `ProcessCard.tsx`

**State/Hooks:**
- `useQueueStatus()` - Query hook with polling
- `useRunningProcesses()` - Query hook with polling
- `useToggleProcessor()` - Mutation hook
- `useTerminateProcess()` - Mutation hook

**API Integration:**
- `GET /api/queue/status`
- `POST /api/queue/start`
- `POST /api/queue/stop`
- `GET /api/processes/running` (poll every 5s)
- `POST /api/queue/processes/terminate`

**Testing:**
- [ ] Processor toggle works
- [ ] Process monitor shows running tasks
- [ ] Terminate process works
- [ ] Countdown timer accurate
- [ ] Polling intervals correct

**Exit Criteria:**
- Floating controls fully functional
- Process monitoring works
- Processor control works
- All buttons functional

---

### Phase 5: Settings Modal (Tabs 1-3)
**Estimated Time:** 2 sessions
**Status:** Not Started

**Objectives:**
- Build settings modal shell
- Implement Processor, Agent, Display tabs
- Handle settings save/load

**Tasks:**
1. Build SettingsModal with tabs
2. Implement ProcessorTab with fields
3. Implement AgentTab with fields
4. Implement DisplayTab with theme preview
5. Add settings form validation
6. Integrate settings query/mutation
7. Add reset to defaults
8. Implement mobile tab selector

**Components:**
- `SettingsModal.tsx`
- `SettingsTabs.tsx`
- `ProcessorTab.tsx`
- `AgentTab.tsx`
- `DisplayTab.tsx`
- `SettingsActions.tsx`

**State/Hooks:**
- `useSettings()` - Query hook
- `useSaveSettings()` - Mutation hook
- `useResetSettings()` - Mutation hook
- `useTheme()` - Theme context hook

**API Integration:**
- `GET /api/settings`
- `PUT /api/settings`
- `POST /api/settings/reset`

**Testing:**
- [ ] Settings load correctly
- [ ] All fields editable
- [ ] Theme preview works
- [ ] Save persists changes
- [ ] Reset restores defaults
- [ ] Mobile selector works

**Exit Criteria:**
- Settings modal functional
- First 3 tabs complete
- Save/reset work
- Theme toggle works

---

### Phase 6: Settings Modal (Tabs 4-7)
**Estimated Time:** 3 sessions
**Status:** Not Started

**Objectives:**
- Implement Prompt Tester tab
- Implement Rate Limits tab
- Implement Recycler tab with testbed
- Implement Auto Steer tab

**Tasks:**

#### 6A: Prompt Tester Tab
1. Build TesterConfig form
2. Implement preview button
3. Build TesterResults display
4. Add section breakdown viewer
5. Add copy to clipboard
6. Integrate prompt preview API

**Components:**
- `PromptTesterTab.tsx`
- `TesterConfig.tsx`
- `TesterResults.tsx`
- `PromptPreview.tsx`
- `SectionBreakdown.tsx`

#### 6B: Rate Limits Tab
1. Build rate limit status display
2. Add reset button
3. Implement countdown timer
4. Handle rate limit notifications

**Components:**
- `RateLimitsTab.tsx`
- `RateLimitStatus.tsx`

#### 6C: Recycler Tab
1. Build recycler settings form
2. Implement model provider/name selectors
3. Add threshold inputs
4. Build recycler testbed
5. Implement preset suite runner
6. Add test results table

**Components:**
- `RecyclerTab.tsx`
- `RecyclerFields.tsx`
- `RecyclerTestbed.tsx`
- `PresetSuite.tsx`
- `TestResults.tsx`

#### 6D: Auto Steer Tab
1. Build profile list view
2. Implement profile cards
3. Build template gallery
4. Add create/edit/delete actions
5. Link to profile editor modal

**Components:**
- `AutoSteerTab.tsx`
- `ProfileList.tsx`
- `ProfileCard.tsx`
- `TemplateGallery.tsx`
- `TemplateCard.tsx`

**State/Hooks:**
- `usePromptPreview()` - Mutation hook
- `useResetRateLimit()` - Mutation hook
- `useRecyclerTest()` - Mutation hook
- `useAutoSteerProfiles()` - Query hook
- `useAutoSteerTemplates()` - Query hook

**API Integration:**
- `POST /api/prompt-viewer`
- `POST /api/queue/reset-rate-limit`
- `POST /api/recycler/test`
- `GET /api/auto-steer/profiles`
- `GET /api/auto-steer/templates`

**Testing:**
- [ ] Prompt tester works
- [ ] Rate limit reset works
- [ ] Recycler test works
- [ ] Auto Steer CRUD works

**Exit Criteria:**
- All 7 settings tabs complete
- All features functional
- Settings fully migrated

---

### Phase 7: Auto Steer Advanced Features
**Estimated Time:** 2 sessions
**Status:** Not Started

**Objectives:**
- Build Auto Steer Profile Editor
- Implement Condition Builder
- Add phase management

**Tasks:**
1. Build AutoSteerProfileEditorModal
2. Implement profile basic info editor
3. Build phase list/editor
4. Add tag editor
5. Build ConditionBuilderModal
6. Implement condition tree (recursive)
7. Add AND/OR logic groups
8. Integrate profile CRUD mutations

**Components:**
- `AutoSteerProfileEditorModal.tsx`
- `ProfileBasicInfo.tsx`
- `PhaseList.tsx`
- `PhaseEditor.tsx`
- `TagEditor.tsx`
- `ConditionBuilderModal.tsx`
- `ConditionTree.tsx`
- `ConditionNode.tsx` (recursive)
- `LogicOperator.tsx`
- `FieldSelect.tsx`
- `OperatorSelect.tsx`
- `ValueInput.tsx`

**State/Hooks:**
- `useCreateProfile()` - Mutation hook
- `useUpdateProfile()` - Mutation hook
- `useDeleteProfile()` - Mutation hook
- `useDuplicateProfile()` - Mutation hook

**API Integration:**
- `POST /api/auto-steer/profiles`
- `PUT /api/auto-steer/profiles/{id}`
- `DELETE /api/auto-steer/profiles/{id}`

**Testing:**
- [ ] Profile editor works
- [ ] Condition builder works
- [ ] Phase management works
- [ ] Tag editor works
- [ ] CRUD operations work

**Exit Criteria:**
- Auto Steer fully functional
- Condition builder working
- Profile editor polished

---

### Phase 8: System Logs & Additional Modals
**Estimated Time:** 1 session
**Status:** Not Started

**Objectives:**
- Build System Logs modal
- Implement log viewer
- Add log filtering

**Tasks:**
1. Build SystemLogsModal
2. Implement log level filter
3. Build LogViewer component
4. Add auto-scroll toggle
5. Implement log streaming
6. Add timestamp formatting

**Components:**
- `SystemLogsModal.tsx`
- `LogLevelFilter.tsx`
- `LogViewer.tsx`
- `LogEntry.tsx`

**State/Hooks:**
- `useSystemLogs(limit)` - Query hook with polling

**API Integration:**
- `GET /api/logs?limit=500`

**Testing:**
- [ ] Logs load and display
- [ ] Filtering works
- [ ] Auto-scroll works
- [ ] Timestamps formatted

**Exit Criteria:**
- System logs modal complete
- Log viewer functional
- Filtering works correctly

---

### Phase 9: Polish & Optimization
**Estimated Time:** 1-2 sessions
**Status:** Not Started

**Objectives:**
- Refine responsive design
- Add loading states
- Implement error boundaries
- Optimize performance
- Add accessibility features
- Polish animations

**Tasks:**
1. Audit mobile responsiveness
2. Add skeleton loaders
3. Implement error boundaries
4. Add React.memo where needed
5. Optimize re-renders
6. Add ARIA labels
7. Test keyboard navigation
8. Polish animations/transitions
9. Optimize bundle size
10. Add loading overlays

**Components:**
- `ErrorBoundary.tsx`
- `SkeletonTaskCard.tsx`
- `LoadingOverlay.tsx`
- Various optimizations to existing components

**Testing:**
- [ ] Mobile UX smooth
- [ ] Loading states polished
- [ ] No unnecessary re-renders
- [ ] Accessibility audit passes
- [ ] Keyboard navigation works
- [ ] Bundle size acceptable

**Exit Criteria:**
- App feels polished
- Performance optimized
- Accessibility compliant
- Mobile-friendly

---

### Phase 10: Final Migration & Cleanup
**Estimated Time:** 1 session
**Status:** Not Started

**Objectives:**
- Remove old UI
- Update service.json
- Update documentation
- Final testing

**Tasks:**
1. Copy any missing features from old UI
2. Delete old UI files (app.js, modules/, styles.css, index.html)
3. Move `ui-new/` contents to `ui/`
4. Update `.vrooli/service.json` lifecycle scripts
5. Update README
6. Remove old dependencies (sortablejs)
7. Final end-to-end testing
8. Deploy and verify

**Testing:**
- [ ] All features work
- [ ] No regressions
- [ ] Production build works
- [ ] Service lifecycle works

**Exit Criteria:**
- Old UI removed
- New UI is the default
- All tests passing
- Documentation updated
- Migration complete! 🎉

---

## Risk Assessment & Mitigation

### High Risks

**1. Feature Parity Gaps**
- **Risk:** Missing features during migration
- **Mitigation:** This comprehensive feature inventory; checklist tracking
- **Contingency:** Keep old UI accessible via feature flag during transition

**2. State Management Complexity**
- **Risk:** Incorrect state synchronization, race conditions
- **Mitigation:** Use battle-tested @tanstack/react-query; clear query key structure
- **Contingency:** Extensive testing of state mutations

**3. WebSocket Integration**
- **Risk:** Real-time updates fail or cause performance issues
- **Mitigation:** Reuse existing WebSocketHandler module pattern
- **Contingency:** Fall back to polling if WebSocket unstable

**4. Drag-Drop Performance**
- **Risk:** Janky drag-drop with large task lists
- **Mitigation:** Use optimized @dnd-kit library; virtualize long lists if needed
- **Contingency:** Simplify animations or limit visible tasks

### Medium Risks

**5. Auto Steer Complexity**
- **Risk:** Condition builder is complex to re-implement
- **Mitigation:** Break into small components; leverage recursive patterns
- **Contingency:** Ship without condition builder initially if needed

**6. Responsive Design**
- **Risk:** Desktop-focused design doesn't translate to mobile
- **Mitigation:** Mobile-first Tailwind approach; test on real devices
- **Contingency:** Simplified mobile layout if full feature set too complex

**7. Theme System**
- **Risk:** Theme switching causes flicker or broken styles
- **Mitigation:** Use Tailwind dark mode; cache preference in localStorage
- **Contingency:** Default to dark mode only initially

### Low Risks

**8. Bundle Size**
- **Risk:** Large bundle from React + dependencies
- **Mitigation:** Code splitting, tree shaking, lazy loading
- **Contingency:** Acceptable for desktop-first app; optimize later

---

## Session Continuity Plan

Each session should end with:
1. **Progress Update:** Update this document with completed items
2. **Next Steps:** Clear list of 3-5 tasks for next session
3. **Blockers:** Document any issues encountered
4. **Commit Message:** Detailed commit describing work done

Each session should start with:
1. **Review Previous:** Read last session's notes
2. **Verify State:** Check current branch, run dev server
3. **Plan Work:** Identify 3-5 tasks from roadmap
4. **Time Box:** Set realistic scope for session

---

## Technology Stack Reference

### Current (Legacy)
```json
{
  "framework": "Vanilla JavaScript",
  "modules": "ES Modules",
  "styling": "CSS (6,919 LOC monolithic)",
  "drag-drop": "sortablejs",
  "build": "Vite",
  "utilities": ["@vrooli/api-base", "@vrooli/iframe-bridge"]
}
```

### Target (React)
```json
{
  "framework": "React 18",
  "language": "TypeScript",
  "styling": "Tailwind CSS",
  "components": "shadcn/ui",
  "state": "@tanstack/react-query + React Context",
  "drag-drop": "@dnd-kit/core",
  "icons": "lucide-react",
  "build": "Vite",
  "utilities": [
    "@vrooli/api-base",
    "@vrooli/iframe-bridge",
    "clsx",
    "tailwind-merge",
    "class-variance-authority"
  ]
}
```

---

## Appendix: Code Patterns

### API Hook Pattern
```typescript
// src/hooks/useTasks.ts
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/lib/queryKeys';
import { api } from '@/lib/api';

export function useTasks(filters: TaskFilters) {
  return useQuery({
    queryKey: queryKeys.tasks.list(filters),
    queryFn: () => api.getTasks(filters),
    staleTime: 30000, // 30 seconds
  });
}
```

### Mutation Hook Pattern
```typescript
// src/hooks/useCreateTask.ts
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/queryKeys';
import { api } from '@/lib/api';

export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTaskInput) => api.createTask(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}
```

### WebSocket Hook Pattern
```typescript
// src/hooks/useWebSocket.ts
import { useContext, useEffect } from 'react';
import { WebSocketContext } from '@/state/WebSocketContext';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/queryKeys';

export function useTaskUpdates() {
  const { lastMessage } = useContext(WebSocketContext);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!lastMessage) return;

    if (lastMessage.type === 'task_status_update') {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
      queryClient.invalidateQueries({
        queryKey: queryKeys.tasks.detail(lastMessage.task_id)
      });
    }
  }, [lastMessage, queryClient]);
}
```

---

## Success Criteria Checklist

### Phase 0 (Current)
- [x] Feature inventory complete
- [x] API documentation complete
- [x] Component hierarchy designed
- [x] State management strategy defined
- [ ] Parallel infrastructure running

### Overall Migration
- [ ] All features from old UI implemented
- [ ] No regressions in functionality
- [ ] Performance equal or better
- [ ] Mobile responsiveness improved
- [ ] Type safety enforced
- [ ] Bundle size < 1MB gzipped
- [ ] Lighthouse score > 90
- [ ] Zero console errors/warnings
- [ ] Documentation updated
- [ ] Old UI removed

---

## Next Steps (After Phase 0)

1. **Set up parallel infrastructure:**
   ```bash
   cd scenarios/ecosystem-manager
   mkdir ui-new
   cp -r ../../scripts/scenarios/templates/react-vite/ui/* ui-new/
   cd ui-new && pnpm install
   # Configure vite.config.ts for different port
   # Add route toggle in server.js
   ```

2. **Verify both UIs work:**
   - Old UI: http://localhost:36110
   - New UI: http://localhost:36111 (or /new route)

3. **Start Phase 1:** Begin with KanbanBoard component

---

**Document Version:** 1.0
**Last Updated:** 2025-11-23
**Next Review:** After Phase 0 completion
