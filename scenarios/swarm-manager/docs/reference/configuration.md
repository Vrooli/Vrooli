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

### Insights Engine

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `minimumCompletedScenarios` | 3 | 1-10 | Minimum completed scenarios before insights |
| `patternWindowSize` | 50 | 10-200 | Action window used for pattern detection |
| `confidenceThreshold` | 0.7 | 0.5-0.95 | Confidence required to surface insight |

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
