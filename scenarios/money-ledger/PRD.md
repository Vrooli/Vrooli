# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Hold where money is and where it went as an auditable journal, admit every source through one contract, and compute financial position rather than asserting it. Replaces a runway formula and snapshot schema that were written as prose and re-executed by hand.
- **Primary users/verticals**: Any operator with money moving through more than one place — a solo founder, a freelancer, a household, a small business. The Vrooli `monetization` team is the first user, not the shape of the product.
- **Deployment surfaces**: Go API (Connect-RPC), typed Go CLI, React operator console, and adapters that pull money events from external sources.
- **Value promise**: Every money event — a Stripe charge, a bank transaction, a hand-typed cash sale — lands in one journal in one shape, carrying how much it can be trusted. Position is a query over that journal and can never be stale.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Accounts and books | Accounts belong to exactly one book (an accounting entity); queries scope to one book or span several, and inter-book movement is modelled rather than lost
- [x] OT-P0-002 | Signed postings journal | Money events are dated, signed postings against accounts, stored append-only with an audit trail
- [x] OT-P0-003 | Derived balances | Balances, cash flow, burn and net position are computed at read time and never stored, so a stale figure is structurally impossible
- [x] OT-P0-004 | The money-event contract | Exactly one typed contract admits events into the journal; every source is an adapter satisfying it, with no privileged path
- [x] OT-P0-005 | Provenance and basis on every event | Each event carries its adapter, external id, fetch time, and a basis of `authoritative`, `derived`, or `operator-asserted`
- [x] OT-P0-006 | Manual and file adapters are first-class | Operator entry and file import are ordinary adapters, not a degraded fallback, because several real revenue sources have no API at all
- [x] OT-P0-007 | Idempotent ingestion | Re-running an adapter over an overlapping window produces no duplicate postings

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Position and runway | Runway, burn, and inflow/outflow over a window are computed from the journal with every input's source and age visible
- [x] OT-P1-002 | Goals with thresholds | Financial goals are declared as thresholds with a sustain window and evaluated against position; "default-alive" is one instance rather than the only rule
- [x] OT-P1-003 | Statements | Income-versus-expense over a period and assets-minus-liabilities at a date, as queries over the same journal
- [x] OT-P1-004 | Adapter health is honest | An adapter that cannot run is reported unavailable with a reason and an age; it is never reported as zero
- [ ] OT-P1-005 | Commerce adapter | The Landing Page Business Suite adapter lands subscription and charge events, proving the contract against a real upstream
- [ ] OT-P1-006 | Tax categorisation | Events carry deductibility category tags, so an accountant gets a clean export

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Reconciliation | Match journal entries against an authoritative source statement and surface unmatched entries on both sides
- [x] OT-P2-002 | Operator console | Journal, position, goals and adapter health with explicit loading, empty, partial, stale and error states
- [ ] OT-P2-003 | Valuation accounts | Holdings whose value changes without a transaction — investments, crypto — as an account kind plus a valuation event
- [ ] OT-P2-004 | Adapter extraction | Adapters move to their own scenario when credential rotation, independent sync clocks, or a third live adapter justifies the boundary
- [ ] OT-P2-005 | Cost-basis lots | An event may reference an acquisition lot so a per-unit margin is knowable for resale, once a resale capability exists to need it
- [ ] OT-P2-006 | Recurring and expected events | An expected future event is an ordinary event whose basis is projected, so runway becomes forward-looking without introducing a second concept
- [ ] OT-P2-007 | Rule-based categorisation | Declared rules assign categories at ingest; a rule never writes an amount, and any model-assisted suggestion is a proposal carrying a derived basis
- [ ] OT-P2-008 | Event attachments | An event may carry a receipt or statement document as the evidence layer under the categorised export
- [ ] OT-P2-009 | Per-currency reporting | Balances and statements report each currency separately and never convert between them, so multi-currency holding works without an FX engine

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API behind Connect-RPC with generated proto contracts, typed Go CLI mirroring each service, React + TypeScript + Vite operator console.
- Data + storage expectations: SQLite via `api-core/storage`. Accounts and signed postings are modelled from the first commit — the double-entry *data shape*, without the accounting reporting UX — because the shape is nearly free now and painful to retrofit. Every amount carries a currency; there is no FX engine. This dataset is **not regenerable**: it cannot be recomputed from anything else, and its backup policy must say so.
- Integration strategy: one inbound contract, many adapters. Adapters may only emit events through that contract; they never write balances and never touch offers. Credentials, where an adapter needs them, are held through the platform's secret store and never in scenario config.
- Non-goals / guardrails: no tax computation or filing, ever — categorise, never compute; no payroll; no invoicing or accounts-receivable chasing; no accrual accounting (cash basis is correct for this user); no FX gain/loss engine; **no direct bank credential storage** — aggregator APIs, file import, or nothing.

## 🤝 Dependencies & Launch Plan
- Required resources: none beyond SQLite for P0. A secret store is required before any adapter that authenticates to a third party.
- Scenario dependencies: Landing Page Business Suite as the first non-trivial adapter source (P1). Offer Desk reads this scenario for its actuals join; that direction is one-way and this scenario has no knowledge of offers.
- Operational risks: adapter surfaces drift and break constantly, which is why they sit behind one contract and report unavailability honestly; holding third-party credentials is a real liability, which is why aggregator APIs and file import are preferred over anything that requires a password; an append-only journal makes corrections harder by design, so the reversing-entry workflow must exist before real data lands.
- Launch sequencing: (1) books, accounts, postings, audit trail; (2) the money-event contract with manual and file adapters; (3) position and goals; (4) the commerce adapter, proving the contract against a real upstream; (5) reconciliation and console.

## 🎨 UX & Branding
- Look & feel: the generated `vrooli-default` design language, unmodified. Financial figures are dense and compared across rows, so tabular numerals and consistent alignment matter more here than in a typical operator tool.
- Accessibility: WCAG 2.1 AA. Money is never communicated by colour alone — an outflow carries a sign and a label, not just a red tint. Every chart has a table equivalent.
- Voice & messaging: exact and unhedged. An unavailable adapter says which one, when it last succeeded, and what is therefore missing. A figure derived from an operator-asserted value says so next to the number.
- Branding hooks: none beyond the design kit. Replace the generic PWA icons when product branding exists.

## 📎 Appendix

**Provenance.** Specified before generation from an audit of `docs/monetization/evidence/FINANCIAL_MODEL.md`, which stated a runway formula, a default-alive threshold with a sustain window and a buffer multiple, and an eleven-field per-heartbeat snapshot schema including deltas and three named flags — all as prose an agent re-executed by hand each cycle, over a hand-edited inputs file. That is a calculator written as a document.

**The generalisation decision.** The contract is deliberately about the shape of a money event rather than about any upstream's API. A landing page, a payment processor, a marketplace, a brokerage and a person typing a number all produce the same thing: a dated, signed, attributed event with provenance. Standardising that is what makes the scenario usable by an operator whose sources are nothing like this one's, and it is why no upstream is named in any P0 target.

**Sibling scenario.** Offer Desk owns what should be sold and the gates that move an offer through its lifecycle. This scenario owns what actually happened. Neither can state "this offer is active and has earned nothing" alone.

**Amendment 2026-08-13 — P2 expansion only.** `OT-P2-006` through `OT-P2-009` were appended after a competitive scan of the personal- and small-business-finance landscape. Nothing above P2 was touched, no existing target was altered, and no narrative section was rewritten — the amendment adds to the future/expansion list, which is what that list is for. Each new target has a matching requirement in `requirements/04-expansion/` and a stated market rationale in `docs/business/GO-TO-MARKET.md`. The scan's rejected features are recorded there too, so a later reader can tell a decision from an omission. Rationale for amending rather than regenerating: regeneration would discard the provenance and generalisation reasoning above, which is the most load-bearing content in this document.
