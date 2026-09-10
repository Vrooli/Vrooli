# Decisions — Tech Tree Designer

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-14 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

## 2026-09-09 — ecosystem design contract

The operator approved a documentation-first expansion, not runtime implementation in this task.

| Decision | Consequence | Revisit trigger |
|---|---|---|
| Separate observed, intended and proposal-local views | Sources own facts; drafts never impersonate implementation | Qualified source contract changes |
| Support repository-wide artifact scope | Scenarios, resources, shared packages and root documentation can share one proposal | New artifact kind needs owner qualification |
| Retain immutable concrete artifacts beside concise plans | Execution consumes reviewed bytes rather than reconstructing intent | Measured owner limitations require explicit fallback |
| Delegate one apply experience to multiple owners | Per-entry receipts and recovery; no global atomicity promise | Proven stronger owner transaction support |
| One approved mandate supports bounded iteration | Swarm owns approval/acceptance; no approval item per ordinary improvement | Material scope or authority amendment |
| Bottom-up work informs a revisable top-down horizon | Contribution differs from verified fulfillment | New applicable evidence |
| Qualify numerical budgets before acceptance | Missing measurements are gaps, not invented guarantees | Approved cohort baselines and thresholds |
| Preserve historical foundation records | Three unsupported completion assertions remain visible evidence debt | Evidence-owner reconciliation with real tests/attestations |

## Development review proposal — ecosystem-design-v1

**State: proposed; not approved, accepted or launched.** This packet prepares one
Swarm backlog item, “Develop Tech Tree Designer's selected ecosystem-design
contract.” It is not a second execution control plane. The generic goal is:
“Read TTD START-HERE and tech-tree-designer-improve. Develop the selected retained
contract within its grants and aggregate limits. Continue successive in-scope
repairs; return owner evidence for every required outcome or an explicit unmet
boundary.” The Swarm owner must bind the exact reviewed artifact bytes before
approval; a branch name or this revision label alone is insufficient.

### Proposed selection and boundaries

Select all nine P0 outcomes. Include P1-001, P1-002 and P1-006 for ontology,
operator usability and typed-surface parity. Defer P1-003/004/005 and all P2;
their rows remain visible and explicitly excluded, not passed. Include any
retention, recovery or identity behavior needed by P0 even when a richer P1
lifecycle is deferred. Do not defer a safety prerequisite under an optional label.

Incorporate PRD, requirements modules, experience files, DESIGN, START-HERE,
architecture/data/domains/flows/integrations, SECURITY, PERFORMANCE, TESTING,
RUNBOOK and these decisions. Skills and program sources are mutable methods;
selected outcomes, numeric floors, accepted exclusions and grants are protected.
The machine-readable preview input is [development-proposal.json](development-proposal.json) beside this
document. It is a review specimen, not an acceptance resolver or retained approval.

Proposed implementation scope is TTD. Cross-owner repair may cover SDA query
contracts, Workspace Sandbox artifact isolation, Plan Manager revision references,
Swarm evidence/authority integration and Test Genie measurement adapters **only
when the reviewed grant names those owners and operations**. Generated artifacts
use their owning generators. Shared primitives must not be reimplemented in TTD.
The concurrent Swarm launch-readiness plan retains its own work; do not edit it
or claim its phases completed from TTD setup evidence.

Proposed effects: local source edits, isolated test data, governed lifecycle and
scoped validation; no commit/push/deploy, production data writes, paid inference,
external network experiments, draft publication or destructive retention. A new
dependency uses Scenario Dependency Analyzer. Proposed development ceiling:
1,000,000 aggregate tokens, 8 aggregate agent-hours, USD 25 total metered spend,
one active agent and no delegated children. These values require operator review
and a qualified accounting path; they do not permit spending in this setup task.
Use metered cancellation unless the selected harness proves a hard ceiling.
Unknown usage does not replenish the budget. Review the budget if measured
scope will not fit; do not quietly omit required outcomes.

### Retention, recovery and deployment proposal

Retain reviewed and applied revisions, referenced bytes and receipts while any
live plan, approval, unresolved apply or recovery pin refers to them. Propose
30 days for unreferenced abandoned drafts and 7 days for rebuildable caches.
Deletion requires an explicit operator-approved policy, reference check and
auditable receipt; no automatic destructive action is authorized by this text.
The existing 5 GiB manifest alarm is not a GC grant. Refuse new retention work
when capacity cannot safely hold it, while preserving existing pins.

Propose daily encrypted consistent backups, RPO ≤24 h and RTO ≤1 h for a local
deployment; require a restore exercise including immutable bytes and unresolved
owner receipts before claiming support. RPO allows loss since the last backup;
it does not meet a zero-data-loss deployment. Such a deployment needs a separately
approved stronger profile. Never replay canonical effects merely after restoring
metadata. RUNBOOK remains honest that this procedure is not implemented.

Initial support is one local operator. Remote/multi-user identity and hosted
deployment are excluded. The local boundary is not an authentication bypass:
owner grants and isolated draft effects still apply. Draft execution experiments
are deferred until explicit runtime/network/billing allowances and isolation
are qualified. Documentation-only bundles must work without executing drafts.

### Evidence and launch gate

TESTING.md owns the 18-outcome inventory and falsification protocols. Register
acceptance resolvers through the Swarm owner when actual owner evidence exists;
do not fabricate resolver IDs here. Every selected outcome needs its tested
product revision, target snapshot, cohort, observed time, verdict, validity and
durable receipt. Setpoint inventory and setup tests are never product receipts.
Preview field completeness is not launch or acceptance readiness.

Before launch: review concrete owner adapters and grants; approve the selected
cohorts/bands and retained artifact snapshot; verify registered resolvers and
budget enforcement; qualify the Swarm/Agent Manager launch path. Then create,
approve and start the one development item through Swarm. This setup does none
of those approval/start operations. Material target, scope, floor, effect or
budget changes require amendment; ordinary implementation choices do not.

Final acceptance: given the approved retained selection, when all selected
outcomes have applicable passing owner evidence and protected regressions pass,
then Swarm may present the work for operator acceptance. Unknown evidence,
unsettled usage, unresolved effects or a failed protected case prevents completion.
