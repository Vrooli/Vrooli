# Utils Unification Notes

Consolidated duplicated utility code across the system-monitor UI and API into shared, well-organized modules. Pure refactoring — no business logic, API, or feature changes.

## New Shared Modules

### UI (`ui/src/shared/`)

| Module | Exports | Replaces |
|--------|---------|----------|
| `utils/formatters.ts` | `formatDurationSeconds`, `formatDurationElapsed`, `formatChartTime`, `formatTimestampDisplay`, `formatWindowLabel`, `formatOptionalNumber` | Inline formatters in 6+ components |
| `utils/colors.ts` | `getRiskLevelColor`, `getHealthColor`, `getStatusColor` | Inline ternary chains in 10+ components |
| `utils/timestamps.ts` | `toIsoString`, `sortByTimestamp` | `tsToIso` in useSystemMonitor, inline sorting |
| `utils/chartData.ts` | `buildSingleSeriesData`, `combineDiskSeries` | Pure data functions extracted from metricHelpers.tsx |
| `api/proto-converters.ts` | `statusEnumToString`, `protoToAgentState` | Inline converters in useInvestigationAgents.ts |

### Go API (`api/internal/`)

| Package | Exports | Replaces |
|---------|---------|----------|
| `healthutil` | `NewResult`, `WithError`, `MarkConnected` | 20+ repetitive map literals in health.go |
| `maputil` | `GetFloat64`, `GetInt`, `GetInt64`, `GetFloat64Slice`, `GetString`, `GetBool` | Unexported helpers in metric_values.go |

### `convert/` Package Split

The 665-line `convert.go` monolith was split into domain files (same package, same function signatures):

| File | Content |
|------|---------|
| `metrics.go` | Metric snapshot/history conversions |
| `investigations.go` | Investigation & trigger conversions |
| `reports.go` | Report conversions |
| `settings.go` | Settings conversions |
| `scripts.go` | Script conversions |
| `enums.go` | All enum mapping helpers |

## Dependency Direction

```
UI Components
  └── features/*/components/*.tsx
        ├── shared/utils/formatters.ts
        ├── shared/utils/colors.ts
        ├── shared/utils/chartData.ts
        └── features/metrics/components/MetricRenderHelpers.tsx (JSX only)

UI Hooks
  └── features/*/hooks/*.ts
        ├── shared/utils/timestamps.ts
        └── shared/api/proto-converters.ts

Go Handlers
  └── internal/handlers/health.go
        └── internal/healthutil/

Go Services
  └── internal/services/metric_values.go
        └── internal/maputil/
```

## Eliminated Duplications

- **Duration formatting**: 4 inline implementations → 1 shared `formatDurationSeconds`
- **Risk level colors**: 10+ inline ternary chains → 1 shared `getRiskLevelColor`
- **Em-dash rendering**: Mixed `'\u2014'`, `'--'`, `'—'` → consistent `'—'` via `formatOptionalNumber`
- **Chart time formatting**: Inline `formatAxisTime` → shared `formatChartTime`
- **Proto timestamp conversion**: Inline `tsToIso` → shared `toIsoString`
- **Health result maps**: 20+ repetitive map constructions → 3 helper functions
- **Map value extraction**: 6 unexported helpers → exported `maputil` package
