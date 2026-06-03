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
| 2026-05-26 | matthalloran8 | done | **Discovery domain (onboarding suggestions, Track B #1+#3).** New `discovery` domain: proto `DiscoveryService` (ListTargetSuggestions/ListDestinationSuggestions/DismissSuggestion) + Go core + Connect handler + CLI group (`discovery targets|destinations|dismiss`) + UI Suggestions panel (Overview onboarding centerpiece). Cross-platform `internal/sysmounts` volume seam backed by `gopsutil/v3/disk` (the one place gopsutil is imported) + fixture-tested per-OS removable classifier. Well-known scanner covers `~/.vrooli` runtime state (`plans`/`state`/`config`/`secrets.json` fs + `runtime.db` sqlite; scenario stores deferred via an additive `rootKind`). Suggestions are derived (only `discovery_dismissals` persists) with stable sha256 ids; accepting reuses existing RegisterTarget/CreateDestination (no accept RPC); scanners strictly read-only (never read `secrets.json` contents). Discovery computes its own protected-path set (runtime root + dest locations + target locators), wider than the destinations `protectedRoot` (D4). All gates green: go build/test, proto generate, `make endpoints`, tsc/eslint/175 vitest/strings:check. Track B #2 (self-registration lifecycle hook) intentionally not built. |

| 2026-06-03 | matthalloran8 | done | **Coverage domain (default-coverage activation).** New `coverage` domain closes the discovered-vs-registered gap: proto `CoverageService` (GetCoverageReport/AcceptDefaultTargets) + Go core composing discovery + targets + plans + runs + restores seams (owns no scanner logic, reads no file contents, persists nothing) + Connect handler + CLI group (`coverage report|accept-defaults` with `--include-sensitive`/`--dry-run`) + UI coverage banner (Overview/Targets/Plans, `features/backup-coverage/`). `AcceptDefaultTargets` bulk-registers non-sensitive discovered durable targets (idempotent, locators only); sensitive credential/token targets require explicit opt-in. Added a plan coverage guard: `CreatePlanRequest`/`UpdatePlanRequest` gain `allow_incomplete_coverage`; `plans create/update` reject with `failed_precondition` while non-sensitive recommendations are unregistered (plans service consults a discovery-backed `CoverageGuard` seam; no construction cycle — a suggestions-only coverage instance backs the guard). UI maps the rejection to an explicit "proceed with incomplete coverage" control. All gates green: api `go test ./...`, cli `go test ./...`, `make endpoints` (29 endpoints), ui type-check/lint/186 vitest. Docs updated (RUNBOOK first-backup flow, cli-commands, api-endpoints, FLOWS, PROBLEMS resolved entry). |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
