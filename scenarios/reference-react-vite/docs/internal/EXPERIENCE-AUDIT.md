# Experience Architecture Audit - 2026-03-11

This document captures the experience architecture audit findings for the reference-react-vite scenario.

## Scenario Purpose

**Purpose Statement**: This scenario helps users manage tasks and projects so they can track work progress and stay organized.

The reference-react-vite scenario is a golden reference implementation demonstrating:
- Full-stack React + Vite + Go architecture
- Task and project management CRUD operations
- Best practices for UI/API integration

## Core Personas & Key Jobs

### Persona 1: First-Time User
**Primary jobs**:
1. Understand what this app does → Dashboard provides overview
2. Create first task → Quick action button on Dashboard + form on Tasks page
3. Explore navigation → Clear nav with Dashboard/Tasks/Projects

### Persona 2: Returning User
**Primary jobs**:
1. Check current task status → Dashboard shows pending/completed counts
2. Continue where left off → Recent tasks on Dashboard
3. Cycle task status → Click status icon in task row

### Persona 3: Project Organizer
**Primary jobs**:
1. Create and manage projects → Projects page with CRUD
2. View project status → Color-coded cards with status indicators
3. Organize tasks by project → (Future: task-project association UI)

## Current vs. Ideal Flows

### Flow 1: Create First Task (First-Time User)

**Current Flow**:
1. Land on Dashboard (/)
2. See "No tasks yet" message if empty, or recent tasks
3. Click "New Task" quick action OR navigate to Tasks
4. Fill form on Tasks page
5. Submit to create task

**Ideal Flow**: Same - Direct access via Dashboard quick action is good.

**Status**: Aligned

### Flow 2: Resume Work (Returning User)

**Current Flow**:
1. Land on Dashboard
2. See stats (pending count, completed count)
3. See 5 recent tasks
4. Click "View all tasks" to see full list
5. Find specific task and update status

**Ideal Flow**: Same - Dashboard provides good at-a-glance status and recent items.

**Status**: Aligned

### Flow 3: Manage Project (Project Organizer)

**Current Flow**:
1. Navigate to Projects page
2. See grid of project cards
3. Create new project via form
4. Cycle project status via icon click

**Ideal Flow**: Same - Direct and clear.

**Status**: Aligned

## Friction Points Analysis

### Mechanical Friction (Too Many Clicks/Inputs)

| Location | Friction | Severity | Status |
|----------|----------|----------|--------|
| Dashboard quick actions | Open same page (Tasks/Projects) for "New Task" and "New Project" | Low | Acceptable |
| Status cycling | Single-click cycles through states | None | Good UX |
| Delete confirmation | Required dialog prevents accidents | None | Intentional |

**Assessment**: Mechanical friction is minimal. Status cycling is efficient (single click).

### Cognitive Friction (Must Remember/Guess)

| Location | Friction | Severity | Status |
|----------|----------|----------|--------|
| Status icons | Icons are intuitive (circle, clock, checkmark) | None | Good |
| Priority labels | Labels are visible on task rows | None | Good |
| Navigation | 3 clear nav items with icons | None | Good |

**Assessment**: Cognitive load is low. Icons and labels are clear.

### Discoverability Friction (Important Capabilities Buried)

| Capability | Discoverability | Severity | Status |
|------------|-----------------|----------|--------|
| Status cycling | Icon is clickable but no tooltip until hover | Low | Acceptable |
| Delete action | Trash icon visible in each row/card | None | Good |
| Health indicator | Visible in header | None | Good |

**Assessment**: Key actions are visible. Status toggle has title attribute for hover.

## Navigation Section

### Navigation Mental Model

The app has a 6-route structure with detail drill-downs:
```
Dashboard (/) ────────┐
    │                 │
    ├──► Tasks (/tasks)
    │        ├──► Task CRUD (inline)
    │        └──► Task Detail (/tasks/:id)
    │                 └──► Notes CRUD (inline)
    │
    └──► Projects (/projects)
             ├──► Project CRUD (inline)
             └──► Project Detail (/projects/:id)
                      └──► Task links to /tasks/:id
```

### Navigation Integrity Verification

| Control | Label | Actual Destination | Honest? |
|---------|-------|-------------------|---------|
| Nav "Dashboard" | Dashboard | / (Dashboard page) | Yes |
| Nav "Tasks" | Tasks | /tasks | Yes |
| Nav "Projects" | Projects | /projects | Yes |
| "New Task" quick action | New Task | /tasks (navigates to Tasks page) | Yes |
| "New Project" quick action | New Project | /projects (navigates to Projects page) | Yes |
| "View all tasks" | View all tasks | /tasks | Yes |
| "Create your first task" (empty state) | Create your first task | /tasks | Yes |

**Assessment**: All navigation labels accurately describe destinations.

### Back/Forward Coherence

- **Browser back/forward**: Works correctly with react-router-dom
- **In-app back buttons**: None needed (flat navigation structure)
- **Close buttons (dialogs)**: ConfirmDialog has cancel/close that dismisses modal

**Assessment**: Navigation is flat (3 pages), no deep hierarchies requiring back navigation.

### Keyboard Shortcuts

| Shortcut | Action | Works? | Scoped? |
|----------|--------|--------|---------|
| Ctrl/Cmd+K | Global search (relays to host) | Yes | App-wide |
| Ctrl/Cmd+S | Save (relays to host) | Yes | App-wide |
| Escape | Close dialogs | Yes | Dialog-local |

**Assessment**: Centralized in `useKeyboardShortcuts.ts`, properly scoped, input-aware.

### Edge Cases

| Scenario | Behavior | Status |
|----------|----------|--------|
| Direct link to /tasks | Loads Tasks page correctly | Good |
| Direct link to /projects | Loads Projects page correctly | Good |
| Browser refresh | State recovered via React Query refetch | Good |
| API offline | Health indicator shows "offline", error states appear | Good |

## Improvements Implemented

### This Phase (18.3)

1. **Project Detail Page**: Added `/projects/:id` route with full project view, edit capability, and task association
2. **Task Association UI**: Project detail shows all tasks linked to the project with clickable links
3. **Navigation Enhancement**: Projects list now links to detail page via name click and chevron icon

### Phase 18.2

1. **Task Detail Page**: Added `/tasks/:id` route with full task view, edit capability, and notes management
2. **Notes UI**: Implemented notes CRUD (previously API-only)
3. **Navigation Enhancement**: Tasks list now links to detail page via title click and chevron icon

### Phase 18

1. **Documentation**: Created this EXPERIENCE-AUDIT.md to document personas, flows, and friction analysis.

### Previously Implemented (Earlier Phases)

- Dashboard with stats and recent tasks (Phase 11)
- Quick action buttons for new task/project (Phase 11)
- Status cycling via icon click (Phase 11)
- Delete confirmation dialog (Phase 12)
- Loading/error/empty states (Phase 11)
- Keyboard shortcuts with iframe relay (Phase 9)

## Recommendations for Future Phases

### Low Effort, High Impact

1. ~~**Notes UI**: Currently API-only, could add UI for task notes~~ ✅ Implemented in Phase 18.2
2. **Task-Project Association**: Allow assigning tasks to projects in UI (create task form)
3. **Filter/Sort Controls**: Add filter by status, sort by priority

### Medium Effort

1. ~~**Task Detail Page**: `/tasks/:id` for full task view with notes~~ ✅ Implemented in Phase 18.2
2. ~~**Project Detail Page**: `/projects/:id` showing associated tasks~~ ✅ Implemented in Phase 18.3
3. **Search**: Global search across tasks and projects

### Already Good (No Changes Needed)

- Navigation structure (3 clear pages)
- Status cycling UX (single-click)
- Dashboard overview (stats + recent)
- Error/loading/empty states
- Keyboard shortcut handling

---

*Last updated: 2026-03-11 by Scenario Improver (Phase 18.3: Project Detail & Task Association Implementation)*
