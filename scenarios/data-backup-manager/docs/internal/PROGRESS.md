# Progress — Data Backup Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-26 | matthalloran8 | done | Scaffolded scenario from the `react-vite` template. Standard three-surface (API/CLI/UI) layout; example `notes` domain still present and slated for removal. |
| 2026-05-26 | matthalloran8 | done | Authored PRD and requirements: runtime-state backup with self-registration, six source kinds, kopia-backed encrypted destinations, many-to-many plans, in-process scheduler, and verified restore. |
| 2026-05-26 | matthalloran8 | done | Wrote companion `kopia` resource plan (`docs/plans/kopia-resource-plan.md`, repo root) — the engine this scenario wraps. |
| 2026-05-26 | matthalloran8 | done | Locked architecture decisions (see `DECISIONS.md`): kopia wrap, Source/Destination/Plan model, encryption-on default, alert+block storage limits, verified-restore gate, separate-root rule, no n8n. |
| 2026-05-26 | matthalloran8 | done | Filled INTERNAL/OPERATIONS/BUSINESS docs to reflect the locked design. |
| 2026-05-26 | matthalloran8 | done | **API+CLI implementation pass.** Removed the `notes` example domain (API/CLI/proto/gen). Authored proto for targets/destinations/plans/runs/restores (+ shared `sources` SourceKind) and regenerated. Built the KopiaEngine + CommandRunner + sources.Capturer seams (wrapping `resource-kopia` and the source resource CLIs). Implemented all five Connect-RPC domains + the health backup-posture rollup; per-domain SQLite schema; idempotent registration; encryption-on/separate-root/alert+block destination rules; many-to-many plans + in-process scheduler; run fan-out with partial-failure isolation and storage-cap block (never evict); verified-restore gate (false-verified prevented). CLI command per RPC + self-registration. All P0 requirement validations green; `make endpoints`/proto-parity/seam gates pass (25 endpoints/25 CLI commands). UI (DBM-UI-001) and P1/P2 are explicit follow-ups (see PROBLEMS). |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
