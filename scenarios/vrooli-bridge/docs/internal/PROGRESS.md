# Progress — Vrooli Bridge

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/vrooli-bridge/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

> Historical entries may mention `vrooli-bridge:session`; that value was removed from the current transport contract on 2026-08-29.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| 2026-08-31 | Codex | Node capability readiness surfaces. | Archived |

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-12 | Codex (agi) | Plan proof pass and security boundary hardening. | Archived |
| 2026-08-11 | Codex (agi) | Remote session federation seam implemented, validation remains open. | Archived |
| 2026-08-11 | Codex (agi) | Acknowledged delivery, session wire/backends, onboarding handoff, operator navigation, and proof hardening. | Archived |
| 2026-07-14 | Claude (agi) | One-shot node onboarding plan — Phase 8 closeout (SSH first touch → paired ONLINE auto-starting node). | Archived |
| 2026-06-18 | Claude (agi) | Phase 4 — Privileged provisioning tier + cross-platform agent hardening (OT-P0-006/007). P0 COMPLETE. | Archived |
| 2026-06-18 | Claude (agi) | Phase 5 — P1 hardening (OT-P1-001/003/004/005/006). | Archived |
| 2026-06-18 | Claude (agi) | Phase 3 — Allowlisted dispatch + durable remote execution + audit (OT-P0-004/005/008). | Archived |
| 2026-06-18 | Claude (agi) | Phase 2 — One-touch bootstrap + mutual auth + atomic revocation (OT-P0-002). | Archived |
| 2026-06-18 | Claude (agi) | Phase 1 — Spine: registry + dial-out presence (OT-P0-001, OT-P0-003). | Archived |
| 2026-06-18 | Claude (agi) | Phase 0 — Foundations. | Archived |
| 2026-06-18 | Claude (agi) | Greenfield regeneration from `react-vite` (the prior doc-injection bridge was removed — see [`DECISIONS.md`](DECISIONS.md) superseded log). | Archived |

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

- **2026-08-11 — Acknowledged delivery, session wire/backends, onboarding handoff, operator navigation, and proof hardening.**: Plan validation remains open because the required cross-scenario baseline reports scenario-to-cloud/web-console as pre-existing/not-comparable, and web-console remote transport is not yet wired beyond target metadata.

- **2026-07-14 — One-shot node onboarding plan — Phase 8 closeout (SSH first touch → paired ONLINE auto-starting node).**: **DoD gap — live localhost durable-op SUCCEEDED: BLOCKED (not faked).** The onboarding feature is entirely uncommitted (local HEAD == origin/agi == e767613, `bootstrap/` absent on origin); the durable-op path builds the node from a pushed revision (phase-6 preflight), so it builds pre-feature tooling whose CLI rejects `pair redeem --state-dir` → op FAILED at pair-redeem. Resolving requires commit+push (forbidden here) — logged operator-pending. Mac-mini live onboard also operator-pending (real target + darwin gate).

- **2026-06-18 — Phase 3 — Allowlisted dispatch + durable remote execution + audit (OT-P0-004/005/008).**: Remaining suite reds (standards security-headers campaign, tidiness template `modeltest`/`no_prod_import` complexity, proto `shared_type_misplaced` Heartbeat/RunEvent) are pre-existing template/deferred debt — see PROBLEMS.md.

- **2026-06-18 — Phase 1 — Spine: registry + dial-out presence (OT-P0-001, OT-P0-003).**: **Remaining Phase 1:** node-agent live SSE dial loop (replace the Phase-0 stub), UI `features/fleet` node list, then full-suite requirements-sync flips BRG-P0-001/003 planned→passing.

- **2026-06-18 — done**: Orientation 5/8 — remaining 3 (scaffold-health `make test`, dependency-decisions service.json resources, example-domain-removed) are implementation-phase.
