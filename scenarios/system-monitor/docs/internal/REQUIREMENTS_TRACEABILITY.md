# Observability Hardening Traceability

| Contract obligation | Evidence | Current proof |
|---|---|---|
| One cycle groups delayed collectors | `api/internal/repository/cycle_test.go`, `api/internal/repository/sqlite/cycle_test.go` | Memory and SQLite preserve one row per cycle with caller-owned time. |
| Failed and unsupported values are explicit | `api/internal/services/metric_values_test.go`, cycle tests | Typed states survive persistence and conversion. |
| Charts exclude unavailable values | `ui/src/features/monitoring/hooks/useMetricHistory.ts`, hook-surface test | Only measured oneof values enter chart series. |
| Reports exclude unavailable values | `api/internal/services/report.go`, report tests | Statistics use measured-state filters. |
| Existing scalar fields are compatibility projections | `packages/proto/schemas/system-monitor/v1/metrics/metrics.proto` | Oneof MetricValue fields are authoritative. |
| Persistent cycle schema is indexed | `api/internal/repository/sqlite/repository.go` | `cycle_id` and `observed_at` columns and indexes are declared in the fresh greenfield schema. |
| Scheduler owns cycle identity | `api/internal/services/monitor.go` | Collection creates one cycle envelope and persists it atomically. |

The remaining plan gates require race, cross-build, lifecycle, retention,
performance, and comprehensive Test Genie evidence. They are not claimed by
this focused contract slice.
