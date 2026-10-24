# Progress — Treasury

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/treasury/docs/internal/PROGRESS.md`.
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
| 2026-08-19 | codex | partial | **Phase 8 evidence retention and ledger emission implemented.** Terminal settlement, a self-contained immutable attempt snapshot, and a Money Ledger outbox row now commit in one SQLite transaction; refusal, approval decline, expiry, settled, and definite-failure paths retain complete replay records. Comprehensive run `20260819-025813-5974ad08` completed all 22 phases with 19 passing; storage is clean, while the three remaining failures are the known governed Go-tidy mismatch, later-owned workflow proof, and branding. |
| 2026-08-18 | codex | partial | The onlyOperatorBeneficiaryCanBeRepresented invariant is now named at the SQLite enforcement sites and registered in docs/internal/INVARIANTS.md. Multiple books for the one operator remain valid. Comprehensive run 20260819-030935-f62a0310 completed all 22 phases with 19 passing; only the known dependency, later-owned workflow, and branding failures remain. |
| 2026-08-19 | codex | partial | **Phase 10 typed CLI and accessible operator approval queue delivered.** `TreasuryAdmin.ListApprovals` now exposes the local queue through generated Connect clients; the landing page reads and resolves real approvals with an in-memory operator credential, visible amount/counterparty/agent/mandate/expiry, non-colour status cues, live outcome announcements, and money-action accessible names. |
| 2026-08-19 | codex | partial | **Phase 11 real-transaction readiness is implemented up to the operator-owned facts.** The generated operator surface now creates and reads the single-beneficiary book, creates or updates budget caps without weakening independent gating/freeze controls, toggles gating, freezes/unfreezes, issues HMAC-signed mandates with the authorizer rebound to the verified operator, and revokes mandates idempotently. The remaining acceptance step is intentionally not fabricated: an operator must identify one already-budgeted recurring payment and either authorize a short-lived Agent Manager run or supply its active agent identity token before Treasury can record the real authorization, approval, manual settlement, evidence replay, and one Money Ledger event. |
| 2026-08-19 | codex | partial | **Phases 12–16 complete for all repository-owned work.** Added bidirectional x402 policy and adapter boundaries, standing-mandate recurrence and cancellation, durable book/budget/global freezes, computed headroom, operator controls, and a provider-neutral scoped-card contract with a sandbox-only Lithic adapter. The immutable generated-scenario baseline `20260818-232425-eff373c1` is structurally non-comparable because it stopped before phase results. P0 is complete; P1-001, P1-002, and P1-003 remain planned until their operator-owned live validations occur. P2 remains intentionally planned. |
## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
