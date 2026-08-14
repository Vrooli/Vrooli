# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Hold every sellable thing an operator offers as a record with a state, evaluate the conditions that should move it, and refuse the transitions that should not happen. Replaces a hand-maintained markdown catalog whose lifecycle nothing could enforce.
- **Primary users/verticals**: The Vrooli `monetization` team as its single instrument; any solo operator, freelancer, or small business that sells more than one thing and forgets to revisit the ones that are waiting on a condition.
- **Deployment surfaces**: Go API (Connect-RPC), typed Go CLI, React operator console, and a `space --projection offers --json` read verb for fleet consumers.
- **Value promise**: An offer that is waiting on a condition stops being something a human has to remember. The condition is declared once, evaluated on a schedule, and surfaces itself when it fires.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Typed offer graph | Offers, variants, channels, and revenue lines as typed nodes with one shared status vocabulary and typed edges
- [x] OT-P0-002 | Enforced lifecycle | An illegal status transition is refused by the API rather than discouraged by prose
- [x] OT-P0-003 | Mandatory revisit trigger | A node cannot enter or remain in `candidate` without a machine-evaluable revisit trigger
- [x] OT-P0-004 | Trigger evaluation | Scheduled evaluation can move a node to `trigger-met`, a state the source documents describe and nothing can currently reach
- [x] OT-P0-005 | Operator-gated promotion | Agents may only propose promotion; the transition to `active` requires an operator-role call
- [ ] OT-P0-006 | Verified import | The source catalog imports with per-source-file counts verified before any source file is deleted
- [x] OT-P0-007 | Append-only audit trail | Every state change records actor, timestamp, prior value, and reason; corrections are new entries, never edits

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Deliverable membership validation | Offer-to-deliverable membership is validated against the live deliverable set; a dangling member is a reported finding
- [x] OT-P1-002 | Actuals join | Reads Money Ledger to report earned-versus-intended per offer; an unavailable ledger is stated with a reason, never reported as zero
- [x] OT-P1-003 | Ranked board | One surface ranks fired triggers, blocked offers, and active offers earning nothing, with each source degrading independently
- [ ] OT-P1-004 | Projection verb | `space --projection offers --json` serves the owning team's obligation cells under the fleet denominator contract
- [x] OT-P1-005 | Trigger authoring aids | Declared triggers are validated at write time and dry-runnable against current facts before being saved

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Operator console | Graph view, offer detail, and promotion actions with explicit loading, empty, partial, stale, and error states
- [ ] OT-P2-002 | Compliance obligations | Per-offer legal and platform obligations tracked with review dates, so a disclosure or terms requirement surfaces before launch rather than after
- [ ] OT-P2-003 | Multiple offer books | More than one operator's offer graph in a single instance, scoped and queryable independently
- [ ] OT-P2-004 | Scenario-authored triggers | Triggers that read another scenario's live state directly rather than an operator-supplied fact

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API behind Connect-RPC with generated proto contracts, typed Go CLI mirroring each service, React + TypeScript + Vite operator console.
- Data + storage expectations: SQLite via `api-core/storage`. State is small, local, and durable — records, a trigger registry, and an append-only audit log. This is the one dataset in the scenario that is not regenerable.
- Integration strategy: Read Money Ledger over a typed client with a short deadline for the actuals join. Never own money, entitlements, billing, or a customer relationship. Expose a `space --projection` verb so fleet consumers read this scenario rather than re-deriving it.
- Non-goals / guardrails: No pricing or billing (Landing Page Business Suite owns commerce). No marketing execution (Content Desk). No accounting or balances (Money Ledger). No agent-initiated promotion to `active`, ever. No strategy decisions — the scenario records a decision, it never makes one.

## 🤝 Dependencies & Launch Plan
- Required resources: none beyond SQLite. No shared resource is justified by any P0 target.
- Scenario dependencies: Money Ledger for the actuals join (P1, optional at runtime and degrades to an honest unavailable). Landing Page Business Suite is read only indirectly, through Money Ledger, so this scenario has no commerce coupling.
- Operational risks: the source catalog's cross-references are already 19% broken, so the import must reconcile rather than trust; a trigger language that is too expressive becomes an unmaintainable rules engine, so the first version deliberately admits only declared facts and comparison operators; the team must be paused during cutover because a running team writes to the surfaces the importer reads.
- Launch sequencing: (1) offer graph and lifecycle enforcement; (2) triggers and evaluation — the first capability that does something no prose could; (3) verified import and source retirement; (4) actuals join and board; (5) operator console.

## 🎨 UX & Branding
- Look & feel: the generated `vrooli-default` design language, unmodified. This is a low-density operator tool read a few times a week, not a dashboard — favour legible type and generous spacing over information density.
- Accessibility: WCAG 2.1 AA. Status is never communicated by colour alone; every status carries a text label, and the graph view has a table equivalent.
- Voice & messaging: plain and specific. A refused transition says which rule refused it and what would satisfy it. A fired trigger states the condition and the fact that satisfied it.
- Branding hooks: none beyond the design kit. Replace the generic PWA icons when product branding exists.

## 📎 Appendix

**Provenance.** This scenario was specified before generation from an audit of `docs/monetization/`, which held four separate lifecycles across 22 hand-maintained markdown files: an offer lifecycle (`idea → candidate → trigger-met → active → shipped → retired`), a delivery-tier lifecycle, and status fields on revenue lines and channels. Two defect classes motivated the build — operational state held in prose, and rules stated as prose that nothing could enforce. `trigger-met` had never been reached by any record because nothing evaluated a trigger.

**Sibling scenario.** Money Ledger owns accounts, money events, balances, and financial position. The division of responsibility is that this scenario holds *what should earn* and Money Ledger holds *what actually happened*; neither can state "this offer is active and has earned nothing" without the other.
