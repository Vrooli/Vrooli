# Progress — Data Backup Manager

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/data-backup-manager/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-26 | matthalloran8 | Scaffolded scenario from the `react-vite` template. | Archived |
| 2026-05-26 | matthalloran8 | Authored PRD and requirements: runtime-state backup with self-registration, six source kinds, kopia-backed encrypted destinations, many-to-many plans, in-process scheduler, and verified restore. | Archived |
| 2026-05-26 | matthalloran8 | Wrote companion `kopia` resource plan (`plan-manager plans get kopia-resource-implementation-validation-plan`) — the engine this scenario wraps. | Archived |
| 2026-05-26 | matthalloran8 | Locked architecture decisions (see `DECISIONS.md`): kopia wrap, Source/Destination/Plan model, encryption-on default, alert+block storage limits, verified-restore gate, separate-root rule, no n8n. | Archived |
| 2026-05-26 | matthalloran8 | Filled INTERNAL/OPERATIONS/BUSINESS docs to reflect the locked design. | Archived |
| 2026-05-26 | matthalloran8 | API+CLI implementation pass. | Archived |
| 2026-05-26 | matthalloran8 | Discovery domain (onboarding suggestions, Track B #1+#3). | Archived |

| 2026-06-03 | matthalloran8 | Coverage domain (default-coverage activation). | Archived |

| 2026-06-03 | matthalloran8 | First real backup to the Elements drive + capturer/transport fixes. | Archived |

| 2026-06-03 | matthalloran8 | Async run execution + persisted lifecycle + startup reconciliation (perf/observability plan, Phase 1). | Archived |

| 2026-06-03 | matthalloran8 | Bounded-concurrency target fan-out (perf/observability plan, Phase 2). | Archived |

| 2026-06-03 | matthalloran8 | In-place snapshot for filesystem sources (perf/observability plan, Phase 3). | Archived |

| 2026-06-03 | matthalloran8 | Per-target kopia overhead — Phase 4 investigated & descoped (perf/observability plan). | Archived |

| 2026-06-03 | matthalloran8 | Run metrics surface (perf/observability plan, Phase 5). | Archived |

| 2026-06-03 | matthalloran8 | Cadence / freshness view (perf/observability plan, Phase 6). | Archived |

| 2026-06-03 | matthalloran8 | Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike | Archived |

| 2026-06-03 | matthalloran8 | Generic snapshot audit (new `audits` domain) + DBM-UI-001 closed | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-05-26 — Discovery domain (onboarding suggestions, Track B #1+#3).**: Well-known scanner covers `~/.vrooli` runtime state (`plans`/`state`/`config`/`secrets.json` fs + `runtime.db` sqlite; scenario stores deferred via an additive `rootKind`).

- **2026-06-03 — First real backup to the Elements drive + capturer/transport fixes.**: `restores` `checksumDir` made symlink-safe; (3) `sources/sqlite.go` single-file artifact unrestorable into a directory target — now staged as a directory (`sqlite/snapshot.db`); (4) transport: `runs trigger`/`restores verify`/`restores restore` CLIs now use an unlimited-timeout Connect client and the API sets `WriteTimeout: 6h`, so multi-minute synchronous backup/restore RPCs no longer disconnect mid-kopia (client `Client.Timeout`/server 30s `WriteTimeout` were killing the exec'd kopia and wedging runs in PENDING). Filed separately: sync-execution/PENDING-reconciliation robustness gap and `resource-kopia repo stats --json` (breaks `destinations usage`).

- **2026-06-03 — Async run execution + persisted lifecycle + startup reconciliation (perf/observability plan, Phase 1).**: **Validated on the real Elements harness:** trigger returns in ~2s (pending); live `capturing`→`snapshotting` with outcomes accruing durably; full 16-target run completed 16/16 in ~45s; a hard `kill -9` mid-`capturing` left an orphan that startup reconciliation closed to `failed` (zero non-terminal runs remain); the migration preserved all 18 pre-existing runs and reconciled the 2 prior-session PENDING orphans.

- **2026-06-03 — Run metrics surface (perf/observability plan, Phase 5).**: **Dedup/physical-bytes deliberately deferred** (no silent drop — documented in PROBLEMS.md): kopia `snapshot create --json` exposes only logical size (empty `stats`, no uploaded bytes) in this build, and `repo stats --json` is separately broken, so a dedup ratio can't be computed reliably yet.

- **2026-06-03 — Cadence / freshness view (perf/observability plan, Phase 6).**: `next_scheduled_at` deferred with a note (needs a scheduler `lastFire` seam — see PROBLEMS.md).

- **2026-06-03 — Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike**: **Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike** (plan `data-backup-manager-deferred-follow-ups-...`). **(E)** Per-target overhead: spike measured ~0.2–0.5s connect/target with kopia's on-disk cache already persisting across spawns and targets already fanning out at concurrency 4 (~2s/run total) → **re-deferred** (kopia server mode is a large change and the per-target override-source identity likely can't survive its snapshot API; rationale in PROBLEMS.md, no scaffolding left).
