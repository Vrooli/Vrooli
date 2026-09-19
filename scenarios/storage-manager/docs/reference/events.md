# Storage-manager events

Storage-manager publishes best-effort domain events through vrooli-events. A
delivery failure is logged and never prevents recovery or retention work.

| Event | Required payload |
|---|---|
| `storage.recovery.started` | `run_id`, `trigger`, `partition` |
| `storage.recovery.action` | `run_id`, `rung`, `provider_id`, `bytes_reclaimed`, `files_removed`, `duration_ms`, `free_before`, `free_after` |
| `storage.recovery.completed` | `run_id`, `status`, `reclaimed_bytes`, `action`, `reason` |
| `storage.writer.hot` | `root`, `bytes_per_hour`, `bytes`, `observed_at` |
| `storage.writer.cooled` | `root`, `bytes_per_hour`, `observed_at` |

The event source is `storage-manager`. Byte values are integer byte counts;
consumers must not parse human-readable audit messages.
