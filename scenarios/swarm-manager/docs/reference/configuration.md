# Configuration Reference

This document describes Swarm Manager's tunable levers.

**Implementation Reference:** [CODE: ui/src/config/index.ts]

## Overview

Swarm Manager exposes a focused configuration surface:
1. Few high-value knobs
2. Safe defaults
3. Bounded ranges
4. Clear behavioral impact

## Configuration Groups

### Data Fetching

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `retryCount` | 2 | 0-5 | Higher = more resiliency, slower hard-fail |
| `retryDelayMs` | 1000 | 500-5000 | Base delay between retries (exponential backoff) |
| `staleTimeMs` | 30000 | 5000-300000 | Freshness window for cached data |
| `cacheTimeMs` | 300000 | 60000-600000 | Retention for unused cache |
| `refetchOnWindowFocus` | true | boolean | Refetch on tab focus |

### Display Limits

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `backlogCardMaxTags` | 3 | 1-10 | Max backlog tags shown before truncation |
| `scenarioCardMaxTags` | 5 | 1-10 | Max scenario tags shown |
| `descriptionLineClamp` | 2 | 1-5 | Visible description lines before truncation |
| `defaultPageSize` | 20 | 10-100 | Default list page size |

### Workshop Auto-Execution (Settings API)

These settings control auto-execution triggers for the workshop refinement system. All are stored in `.vrooli/settings.json` and accessible via `GET/PUT /api/v1/settings`.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `auto_initialize_workshop` | true | boolean | Auto-spawn first workshop round on backlog item creation |
| `auto_advance_workshop` | true | boolean | Auto-spawn next round after save when item is not ready |
| `auto_cascade_workshop` | true | boolean | Auto-trigger dependent item workshops when a dependency becomes ready |
| `max_auto_rounds` | 10 | 0-50 | Maximum rounds before auto-advancement stops (0 = fully disabled) |

**Interaction notes:**
- When `auto_advance_workshop` is false, `max_auto_rounds` is irrelevant (no advancement occurs).
- When `auto_initialize_workshop` is false, newly created items require manual workshop initiation.
- When `auto_cascade_workshop` is false, dependency resolution does not auto-trigger downstream workshops.
- Setting all three booleans to false effectively "locks down" the swarm manager from all auto-execution.

### Execution Defaults (Settings API)

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `default_mode` | `"manual"` | manual, scheduled, yolo | Default execution mode for queued items |
| `default_delay_seconds` | 300 | 0+ | Default schedule delay in seconds |
| `auto_fixup` | false | boolean | Auto-re-run execution on review failure |
| `max_fixup_attempts` | 2 | 0-5 | Maximum fixup re-run attempts |

### Agent Behavior (Settings API)

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `agent_max_turns` | 60 | 5-200 | Maximum conversation turns per agent run |
| `agent_timeout_seconds` | 900 | 60-3600 | Agent run timeout |
| `agent_requires_approval` | true | boolean | Pause agent runs for human approval |

### UI Behavior

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `searchDebounceMs` | 300 | 100-1000 | Search debounce delay |
| `toastDurationMs` | 5000 | 2000-10000 | Toast visibility duration |
| `useSkeletonLoading` | true | boolean | Skeletons vs loading spinners |
| `enableKeyboardShortcuts` | true | boolean | Enables tab shortcut keys |
| `confirmDestructiveActions` | true | boolean | Safety confirmation dialogs |

### API Configuration

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `requestTimeoutMs` | 30000 | 5000-120000 | API timeout threshold |
| `apiVersion` | `"v1"` | string | Version prefix for API routes |

## What Is Not Configurable (By Design)

| Category | Reason |
|----------|--------|
| HTTP client internals | Prevent consistency regressions |
| Domain types/contracts | Runtime model must stay stable |
| Selector IDs | Test stability |
| Recommendation engine behavior | Recommendation subsystem removed from Swarm Manager |

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `VITE_API_BASE_URL` | Base URL for UI API requests | `http://localhost:15000/api/v1` |
| `API_PORT` | API server port | `15000` |
| `UI_PORT` | UI server port | `35000` |

## Testing with Custom Configuration

```typescript
vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0,
    retryDelayMs: 0,
    staleTimeMs: 0,
    cacheTimeMs: 0,
    refetchOnWindowFocus: false,
  },
  displayLimitsConfig: {
    backlogCardMaxTags: 3,
    scenarioCardMaxTags: 5,
    descriptionLineClamp: 2,
    defaultPageSize: 20,
  },
}));
```
