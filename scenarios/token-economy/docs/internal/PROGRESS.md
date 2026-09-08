# Progress — Token Economy

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/token-economy/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.
The table retains selected dated milestones. Other milestones and their limitations
remain in the archive; consult it before resuming historical work.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.


## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-19 | codex | done | **Phase 11 — complete typed CLI surface and CLI-only earn-to-redeem proof.** Added generated-protobuf command groups for mints, holders, earning, grants, journal, catalog, and redemption. |
| 2026-08-19 | codex | done | **Phase 10 — compensating reversals and durable actor provenance.** Added one idempotent minter reversal RPC for mint, grant-credit, and redemption-debit events; every reversal requires a reason, links to the original, refuses a second compensation, preserves the original row, and projects the balance back to its pre-event value. The journal append choke point now stamps actor identity, operator/agent kind, runtime run id, and the shared `verified`/`unavailable`/`invalid`/`absent` verification status; product authentication supplies the operator subject while verified runtime claims remain distinguishable. |
| 2026-08-19 | codex | done | **Phase 9 — exactly-once redemption settlement, reservation, and approval queue.** Added complete redemption/reservation contracts and SQLite persistence, caller-required idempotency receipts, immediate and approval-gated settlement, minter approve/deny surfaces, and a durable pending queue. The redemption repository owns one transaction spanning inventory reservation, balance reservation/debit, redemption state, and append-only journal/audit events; retries return the first outcome, distinct keys remain independent, and denial releases both balance and inventory. Tests tagged `TKE-P0-009`/`TKE-P0-013` prove injected rollback leaves no redemption, reservation, stock change, or debit; pending reservations prevent double spend; generated holder/minter clients complete approval with no relay; and replay does not debit twice. Evidence: full API/CLI tests and Go lint pass; requirements, endpoints, proto generation, scoped proto health, and direct L3 architecture validation are clean; Test Genie run `20260820-012400-95e76145` passes 19/22 phases including unit, API, contracts, business, storage, security, proto, and architecture, failing only the established UI/workflow debt plus an unavailable docs provider. |
| 2026-08-19 | codex | done | **Phase 8 — minter-declared catalog and server-owned availability.** Expanded the stable catalog contract with token-denominated cost, optional window and quantity, explicit immediate/requires-approval posture, retained retirement, and no monetary field. MinterService now exposes catalog create/update/get/list/retire, HolderService browsing returns only currently available entries, and a direct redemption request for an unavailable entry fails at the catalog preflight before redemption can run. A redemption-owned `TKE-P0-013` descriptor test earns the catalog-posture slice while the full queue/settlement requirement remains honestly `in_progress` for Phase 9. Git Control Tower operation `2d69ea4c-e204-49d4-b275-a9d19e1a9da3` is fresh but `UNKNOWN` because the baseline provider was unavailable. |
| 2026-08-19 | codex | done | **Phase 7 — one earning contract for operators and programmatic adapters.** Added the stable, capability-minimal `EarningService.SubmitEarning` integration surface with its own `token-economy:earning` scope, leaving the two audience services unchanged. Git Control Tower operation `b16cc198-0379-4b74-bd91-9bc9d240d4ea` is fresh but `UNKNOWN` because its diff dispatch correctly refused to duplicate the already-active test run. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
