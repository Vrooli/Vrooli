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

### Plan Workshop

Plan Workshop has no automatic initialization, auto-advance, or readiness settings. Operators explicitly open a review for a backlog item or milestone, submit one response, and decide whether to accept the resulting valid candidate plan. Execution checks the accepted canonical plan on the server.

### Execution Defaults (Settings API)

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `default_mode` | `"manual"` | manual, scheduled, yolo | Default execution mode for queued items |
| `default_delay_seconds` | 300 | 0+ | Default schedule delay in seconds |
| `auto_fixup` | false | boolean | Auto-re-run execution on review failure |
| `max_fixup_attempts` | 2 | 0-5 | Maximum fixup re-run attempts |

### Backlog Auto-Filer (Settings API)

The backlog auto-filer converts programmatic maintenance findings into backlog
items under an operator-governed policy. It is disabled by default and runs in
the swarm-manager API process. The fix-before-feature gate remains separate:
when it sees queued feature work, it can wake the auto-filer early, but it does
not bypass the policy below.

Implementation references: [CODE: api/internal/autofiler/sweeper.go],
[CODE: api/internal/autofiler/filer.go], [CODE: api/internal/settings/model.go].

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `auto_filer.enabled` | false | boolean | Master switch. When false, filing is skipped; reconciliation remains policy-owned when cycles run. |
| `auto_filer.mode` | `"suggest"` | suggest, auto_add | `suggest` creates items in the `suggested` status for operator accept/dismiss; `auto_add` files directly into the normal backlog flow. |
| `auto_filer.strategy` | `"feature_pending"` | feature_pending, importance | Selects target scenarios from queued feature work or fleet importance ranking. |
| `auto_filer.max_open_auto_filed` | 10 | 1-100 | Cap for currently open items created by the auto-filer. Human-filed backlog items do not count against this cap. |
| `auto_filer.velocity_window_days` | 7 | 1-90 | Trailing window used by the velocity brake. |
| `auto_filer.min_velocity_transitions` | 1 | 1-1000 | Minimum forward backlog status transitions required in the velocity window before filing new findings. |
| `auto_filer.interval_minutes` | 30 | 1-1440 | Background sweep interval. Feature-queue events can wake a cycle before the next tick. |
| `auto_filer.goal_name` | `"automated-maintenance"` | non-empty string | Reserved goal that receives every auto-filed item as an explicit target. |

Operator notes:
- Dismissing a suggested item archives it and records its `finding_ref`, so the
  same stable finding is not suggested again.
- Accepting a suggested item is a normal backlog status update to `backlog` or
  `ready`; accepted and in-progress items are not auto-closed by later
  reconciliation.
- Resolved findings auto-archive untouched `suggested` items and annotate
  accepted items instead of deleting operator-visible history.
- The status surface is available through `swarm-manager autofiler status` and
  the Settings → Execution tab.
- Operators can force one immediate governed cycle with
  `swarm-manager autofiler run-now` or the Settings → Execution "Run now"
  button; this does not bypass disabled mode, the velocity brake, caps, or
  dismissal memory. The manual request is bounded so unavailable external
  review/scoring dependencies return a degraded status instead of blocking the
  operator indefinitely.

### Agent Behavior (Settings API)

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `agent_max_turns` | 600 | 5-1000 | Maximum conversation turns per agent run |
| `agent_timeout_seconds` | 900 | 60-3600 | Agent run timeout |

### UI Behavior

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `searchDebounceMs` | 300 | 100-1000 | Search debounce delay |
| `toastDurationMs` | 5000 | 2000-10000 | Toast visibility duration |
| `useSkeletonLoading` | true | boolean | Skeletons vs loading spinners |
| `enableKeyboardShortcuts` | true | boolean | Enables tab shortcut keys |
| `delete_confirmation_levels.session` | "simple" | "none"/"simple"/"strong" | Session delete confirmation level |
| `delete_confirmation_levels.scenario` | "strong" | "none"/"simple"/"strong" | Scenario delete confirmation level |
| `delete_confirmation_levels.backlog` | "simple" | "none"/"simple"/"strong" | Backlog item delete confirmation level |
| `delete_confirmation_levels.milestone` | "strong" | "none"/"simple"/"strong" | Milestone delete confirmation level |
| `delete_confirmation_levels.capture` | "none" | "none"/"simple"/"strong" | Capture delete confirmation level |
| `delete_confirmation_levels.backlogFile` | "simple" | "none"/"simple"/"strong" | Backlog file delete confirmation level |

Delete confirmation is keyed by entity type (an open map, not fixed fields). `none` deletes immediately, `simple` shows an OK/Cancel dialog, and `strong` requires typing the entity's name (with a copy button) to confirm. Levels are configurable per entity type under **Settings → General → Delete Confirmation** and persist in `config/settings.json`. The set of deletable entity types is defined by the UI registry (`ui/src/lib/deletable-entities.ts`), mirrored in the API (`api/internal/settings/model.go`).

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
