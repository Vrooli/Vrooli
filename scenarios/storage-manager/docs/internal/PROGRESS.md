# Progress — Storage Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-08 | codex | partial | Phase 1 started: generated storage-manager from the supported `react-vite` template, authored the PRD and requirements through `business-health wizard`, removed the starter requirement module, recorded ecosystem fit as a meta-scenario/interface-enabler, and captured fallback regression anchor `7b01bc3140c1baf470021ae2c6cb536eb34be1dc` after the named baselines were missing. |
| 2026-07-08 | codex | partial | Phase 2 started: added cleanup provider metadata/contracts, safety tiers, side-effect seams, fake filesystem/process/Docker clients, no-real-cleanup drift tests, and seam/idempotency invariants. Validation: `go test ./...` passes in `scenarios/storage-manager/api`; requirements validation remains clean. |
| 2026-07-08 | codex | partial | Phase 3 started: added conservative provider registry, filesystem-backed trash/tmp/cache providers, Docker provider excluding volumes, journal and command metadata providers, owner-scenario delegation provider, policy profiles, fake journal/owner clients, provider tests, and provider reference docs. Validation: `go test ./...` passes in `scenarios/storage-manager/api`. |
| 2026-07-08 | codex | partial | Phase 4 started: added CleanupService proto/API/CLI surface for provider catalog, policy profile, deterministic plan creation, approval/idempotency-gated apply, and redacted audit listing. Added orchestration service tests for stable plan IDs, policy/version gates, apply replay, and audit redaction. Refreshed generated proto artifacts, endpoint metadata, and CLI primitive evidence. Validation: `go test ./...` passes in `scenarios/storage-manager/api` and `scenarios/storage-manager/cli`. |
| 2026-07-08 | codex | partial | Phase 5 vertical slice: removed starter Notes runtime/docs, added the cleanup dashboard console, reconciled generated proto/CLI/docs/experience assets, moved architecture metadata under `.vrooli/`, fixed dependency governance drift through the analyzer, hardened logger/preflight/API seams, and raised UI coverage. Validation: `vrooli scenario test storage-manager` passed as `20260708-053406-dcb10f14`. |
| 2026-07-08 | codex | partial | Phase 6 integration slice: converted autoheal's Docker `prune` recovery action into a storage-manager handoff that makes zero executor calls, replaced broad autoheal disk/Docker cleanup guidance with storage-manager preview/apply flow, and documented adjacent ownership boundaries for system-monitor observation and workspace-sandbox owner hooks. Validation: `go test ./internal/checks/infra` passes in `scenarios/vrooli-autoheal/api`; storage-manager docs/unit phases pass; `vrooli scenario test storage-manager` passed as `20260708-134611-42a1a034`. Residual outside-scope validation: broader autoheal API tests still fail in `api/internal/systemevents`, and autoheal docs-health has pre-existing contract drift. |
| 2026-07-08 | codex | partial | Phase 6 owner-hook hardening: registered disabled-by-default owner-scenario providers for `workspace-sandbox-retention`, `test-genie-run-retention`, and `web-console-sessions`; added tests proving owner hooks stay `safe_with_owner` and delegate only through `ScenarioProviderClient`; regenerated `.vrooli/endpoints.json` to remove stale starter Notes endpoints. Validation: `go test ./...` passes in storage-manager API/CLI; `proto-health validate scenario storage-manager --json` passes; `vrooli scenario test storage-manager` passed as `20260708-163312-69eddc26`. Adjacent docs validation blockers were filed to scenario-qa as `knw-1783528605319723546`. |
| 2026-07-08 | codex | partial | Phase 6 system-monitor handoff slice: implemented system-monitor's read-only `MetricsService.GetDiskDetail` path and added storage-manager remediation notes when disk pressure is high, keeping system-monitor observational while storage-manager owns cleanup policy/apply/audit. Validation: `go test ./...` passes in `scenarios/system-monitor/api`; `vrooli scenario test storage-manager` passed as `20260708-184901-fd1a5be4`. |
| 2026-07-08 | codex | partial | Phase 7 hardening slice: cleared storage-manager's unit-health advisory debt by removing the skipped API e2e runtime branch and adding observable assertions to type-alias/a11y tests; cleared ui-health component-standard warnings by moving `HealthCard` into the standard component inventory and using adopted `Input`, `Select`, and `EmptyState` primitives in the cleanup dashboard. Validation: `go test ./...` passes in `scenarios/storage-manager/api`; `pnpm test:coverage` and `pnpm build` pass in `scenarios/storage-manager/ui`; `unit-health validate scenario storage-manager` reports no findings; `ui-health validate scenario storage-manager` reports only live-runtime unavailable info when the UI is not running; `vrooli scenario test storage-manager --json` passed as `20260708-185734-d2c7a514`. Scenario-auditor current equivalent checks: `scenario-auditor scenarios health storage-manager` reported health score 100.0; `scenario-auditor standards scan storage-manager --wait` completed as `standards-78e77968-6194-4e4b-bbc2-42b015886aaf` with overbroad/stale standards findings filed to scenario-qa as `knw-1783537291399114281`. |

| 2026-10-08 | codex | done | Repaired temporary scan starvation with resumable strict directory cursors, indivisible oversized entries, identity/age/activity rechecks and v2 preview rejection; added explicitly granted temporary-only scheduled recovery and its typed CLI configuration; aligned owner-budget approval and classified HTTP failures; exposed incomplete/no-progress retention; bounded fleet acceptance fixtures and repaired census nesting and dashboard controls. A read-only control-plane process observer was prepared; its original installer was superseded by the setup-managed broker integration below. Evidence: `~/.vrooli/plan-artifacts/disk-investigation-20261008/`. |
| 2026-10-08 | codex | done | Moved temporary observation into the existing setup-managed privilege broker; removed the standalone helper/installer and scenario sudo execution. Setup now honors elevation consent while preserving the owner's parent session, and status reports the live capability registry. Broker transport verifies the root peer, bounds waiting and payloads, and matches receipts. Retained host /tmp when TMPDIR is relocated, required whole-host activity proof with explicit root coverage, and repaired dropped managed-log subjects. Root/platform and affected storage packages pass race tests; live setup dry run succeeds. Host broker upgrade remains the normal project setup operation. |

2026-10-08 — Automatic safe temporary cleanup enabled after the operator
clarified the expected behavior. Guarded whole-tree temporary cleanup defaults
to automatic authority in the balanced profile; conservative mode remains
disabled, and explicitly persisted operator choices are preserved. The live
temporary policy uses three-day retention and its fifteen-minute schedule.
The first scheduled run completed as
`recovery-3099782-1791483003371684670-70`, reporting 2,462,419,240 bytes
reclaimed and `target_met`. Focused provider/policy/orchestrator
race tests and Test Genie `20261008-200928-b2515a6e` contracts/unit pass.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map


2026-10-16 — Repaired zero-byte temporary recovery starvation. An expired
empty file no longer pins the resumable cursor ahead of large reclaimable
trees. Added optional exact preview item IDs to cleanup apply, including
unknown/ambiguous-ID rejection before effects and replay binding to the plan
and normalized selection. The CLI preserves repeated item IDs over the actual
protobuf transport. Policy, owner, activity, age and identity checks remain
active. Generated protobufs, endpoint metadata and CLI primitive evidence were
updated through their owners. Negative-before-positive regressions pass;
Test Genie contracts/unit passed as `20261017-030800-d792a6b0`. Canonical
lifecycle rollout completed; operator-authorized cleanup receipts are in
`~/.vrooli/plan-artifacts/disk-investigation-20261016/`.
