# Configuration Reference

This document describes the tunable levers available in Swarm Manager. These settings allow operators to customize behavior without modifying code.

**Implementation Reference:** [CODE: ui/src/config/index.ts]

## Overview

Swarm Manager's control surface is designed around a few key principles:

1. **Fewer, well-chosen knobs** - Only settings that meaningfully affect behavior are exposed
2. **Clear impact descriptions** - Each lever describes what happens when you change it
3. **Safe defaults** - All defaults work well for common usage
4. **Bounded values** - Extreme but valid values produce degraded (not broken) behavior

## Configuration Groups

### Data Fetching

Controls how data is fetched and refreshed across the UI.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `retryCount` | 2 | 0-5 | Higher = more reliable on flaky networks but delays error feedback |
| `retryDelayMs` | 1000 | 500-5000 | Base delay between retries. Uses exponential backoff |
| `staleTimeMs` | 30000 | 5000-300000 | How long before cached data is considered stale |
| `cacheTimeMs` | 300000 | 60000-600000 | How long to keep unused data in cache |
| `refetchOnWindowFocus` | true | boolean | Whether to refetch when the window regains focus |

**Common adjustments:**
- **Slow network**: Increase `retryCount` and `retryDelayMs`
- **Real-time data**: Decrease `staleTimeMs` and enable `refetchOnWindowFocus`
- **Reduce server load**: Increase `staleTimeMs` and `cacheTimeMs`

### Display Limits

Controls truncation and pagination of displayed items.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `ideaCardMaxTags` | 3 | 1-10 | Max tags shown on idea cards before "+N more" |
| `scenarioCardMaxTags` | 5 | 1-10 | Max tags shown on scenario cards |
| `descriptionLineClamp` | 2 | 1-5 | Lines shown for descriptions before truncation |
| `defaultPageSize` | 20 | 10-100 | Items per page in paginated lists |

**Common adjustments:**
- **Dense information**: Increase `ideaCardMaxTags` and `descriptionLineClamp`
- **Cleaner cards**: Decrease limits for simpler appearance
- **Large datasets**: Adjust `defaultPageSize` based on typical list sizes

### Recommendation Engine

Controls the behavior of the recommendation engine.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `minimumCompletenessThreshold` | 0 | 0-100 | Only recommend for scenarios above this score |
| `maxActiveRecommendationsPerScenario` | 5 | 1-20 | Prevents recommendation overload |
| `yoloModeDelayMs` | 5000 | 1000-60000 | Safety delay before auto-approving in YOLO mode |
| `yoloModeAllowedPriorities` | [3, 4, 5] | array of 1-5 | Which priorities can auto-execute in YOLO mode |

**Common adjustments:**
- **Focus recommendations**: Increase `minimumCompletenessThreshold` to target mature scenarios
- **Safety first**: Increase `yoloModeDelayMs` and restrict `yoloModeAllowedPriorities`
- **Full autonomy**: Set `yoloModeAllowedPriorities` to `[1, 2, 3, 4, 5]` (use with caution)

### Insights Engine

Controls the self-improvement and pattern detection engine.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `minimumCompletedScenarios` | 3 | 1-10 | Minimum data before generating insights |
| `patternWindowSize` | 50 | 10-200 | Number of recent actions to analyze |
| `confidenceThreshold` | 0.7 | 0.5-0.95 | How confident before surfacing an insight |

**Common adjustments:**
- **More insights**: Lower `confidenceThreshold` (may include noise)
- **Fewer, reliable insights**: Raise `confidenceThreshold`
- **Detect longer patterns**: Increase `patternWindowSize`

### UI Behavior

Controls general user interface behaviors.

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `searchDebounceMs` | 300 | 100-1000 | Delay before search triggers after typing |
| `toastDurationMs` | 5000 | 2000-10000 | How long notifications stay visible |
| `useSkeletonLoading` | true | boolean | Show skeleton placeholders vs spinners |
| `enableKeyboardShortcuts` | true | boolean | Enable power-user keyboard shortcuts |
| `confirmDestructiveActions` | true | boolean | Require confirmation for destructive actions |

**Common adjustments:**
- **Power users**: Decrease `searchDebounceMs`, enable shortcuts
- **Safety**: Keep `confirmDestructiveActions` enabled
- **Accessibility**: Adjust `toastDurationMs` for reading speed

### API Configuration

API-related settings (most API config comes from environment variables).

| Lever | Default | Range | Impact |
|-------|---------|-------|--------|
| `requestTimeoutMs` | 30000 | 5000-120000 | How long to wait before timing out |
| `apiVersion` | "v1" | string | API version prefix for all requests |

**Common adjustments:**
- **Slow operations**: Increase `requestTimeoutMs` for long-running tasks
- **API migration**: Update `apiVersion` when server supports new version

## What is NOT Configurable (By Design)

Some values are intentionally kept internal:

| Category | Reason |
|----------|--------|
| HTTP cache policies | Internal optimization, changing could break consistency |
| Component styling | Use Tailwind theme for visual customization |
| Type definitions | Domain model shouldn't vary at runtime |
| Selector IDs | Breaking selectors would break tests |

These decisions were made to keep the control surface focused and reduce the chance of misconfiguration.

## Environment Variables

The following environment variables are recognized:

| Variable | Description | Example |
|----------|-------------|---------|
| `VITE_API_BASE_URL` | Base URL for API requests | `http://localhost:15000/api/v1` |
| `API_PORT` | API server port | `15000` |
| `UI_PORT` | UI server port | `35000` |

## Testing with Custom Configuration

For testing, the configuration module can be mocked:

```typescript
vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0,        // Disable retries for faster test failures
    retryDelayMs: 0,
    staleTimeMs: 0,
    cacheTimeMs: 0,
    refetchOnWindowFocus: false,
  },
  displayLimitsConfig: {
    ideaCardMaxTags: 3,
    scenarioCardMaxTags: 5,
    descriptionLineClamp: 2,
    defaultPageSize: 20,
  },
}));
```

## Future Considerations

The following levers may be added in future iterations:

- **Theme customization**: Brand colors, fonts
- **Notification preferences**: Email, Slack, webhook integrations
- **Rate limiting**: Max concurrent operations
- **Audit settings**: Log verbosity, retention

These are documented here for planning purposes but are not yet implemented.
