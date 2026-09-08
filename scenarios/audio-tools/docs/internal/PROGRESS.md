# Progress — Audio Tools

This log keeps recent dated milestones. It does not certify current readiness.
Read [PROBLEMS.md](PROBLEMS.md) for tracked gaps and the owning Plan Manager
record for live execution status.

## Earlier history

The complete pre-cleanup log, including intermediate results and unresolved
handoffs, is preserved byte-for-byte at the protected runtime-home location:

```text
<runtime-home>/plan-artifacts/docs-history-20260907/scenarios/audio-tools/docs/internal/PROGRESS.md
```

SHA-256: `5756d0535ed5652dfd404e391eace8dcdfc971d12d492ace26c22b61b0c2b331`. Restore or read it before relying on an
older completion claim; archival does not close any outstanding work. The
[project preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup--2026-09-07)
records ownership and recovery.

## Recent milestones

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-08-18 | codex | docs/audit-resolution-refresh | Refreshed `plan-artifacts/docs-cleanup-20260907-final-txumdx73/docs/design/audio-reliability-audit-2026-08-09.html` so its resolution ledger and evidence delta distinguish the historical baseline from current state: the BAS/PipeWire device lane is automated and OS-path-scoped, current Audio Tools realtime evidence is named, Swarm Manager product-path credit remains pending, and native sherpa publication/target smoke remain explicit gaps. Removed stale active claims that all audio resources still require Docker or that the qualification instrument does not exist. |
| 2026-08-18 | codex | fix/native-release-reproducibility | Native sherpa release packaging now normalizes gzip metadata, detects tar's `--sort=ORDER` capability correctly, and has a sorted NUL-safe deterministic fallback for non-GNU tar (otherwise it fails closed). Identical target-native stages produce archive checksum `440f9d9738a2457761faa37545f1d28aa084db35d4a47bf7aa94595cedfe0032`; publication metadata includes `entry_path`. `bash -n`, the release-stage check, `make -C resources/sherpa-onnx check`, and two fresh signed release stages pass. Publication and target smoke gates remain intentionally open. |
| 2026-08-18 | codex | fix/swarm-search-registration | Repaired Swarm Manager's committed search descriptor by removing the obsolete `declared_at` field and adding required dense tuning for the initiative provider; the test now validates both intended providers and selects the records provider by id. The complete Swarm API package passes, and a governed restart is healthy with both providers updated in search-hub. |

## New entries

Append a dated milestone with its outcome, remaining constraint, and durable
owner reference. Store detailed command output with the producing owner.
