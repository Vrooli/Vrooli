# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario content-desk`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own the editorial production path for marketing content as a permanent capability — one place where a draft is written against a campaign, its factual claims are verified against re-runnable evidence, its post type and review gates are enforced mechanically, and what actually shipped is recorded as a queryable ledger. It replaces per-file JSONL bookkeeping and honesty rules that exist only as prose.
- **Primary users/verticals**: The marketing-crew producer agent drafting against active campaigns; the marketing-contrarian scoring drafts against per-type failure modes; the operator, who is the only actor that can approve publication; and any scenario that needs publish history, coverage, or subject-familiarity state.
- **Deployment surfaces**: CLI (the agent-facing write and read verbs), API (Connect-RPC), and UI (a draft workbench and a publish ledger).
- **Value promise**: Makes the honesty doctrine mechanical instead of aspirational — a draft cannot reach approved while it carries an unverified claim — and turns campaigns into a work surface so agent capability comes from data rather than per-member configuration.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Campaign record and lifecycle | The system shall persist campaigns with theme, audiences, channels, hypotheses, and status, and shall refuse to activate a campaign that carries no supporting evidence reference
- [ ] OT-P0-002 | Artifact slot budget | The system shall bound each campaign by a declared artifact slot count and shall refuse to accept a draft beyond that budget
- [ ] OT-P0-003 | Draft lifecycle enforcement | The system shall persist each draft with an explicit status and shall reject any transition not declared in the lifecycle contract
- [ ] OT-P0-004 | Shared claim library | The system shall store factual claims as first-class records that many drafts may cite, rather than as annotations owned by a single draft
- [ ] OT-P0-005 | Evidence strength | The system shall record each claim's evidence as either a citation or a re-runnable check with an expected result, and shall distinguish the two
- [ ] OT-P0-006 | Verification gate | The system shall refuse to move a draft to approved while any claim it cites is unverified
- [ ] OT-P0-007 | Operator-only approval | The system shall accept approval only from an operator identity and shall record who approved which draft and when
- [ ] OT-P0-008 | Post-type registry and activation gate | The system shall hold the post-type registry with activation state and shall refuse approval of any draft whose post type is not active
- [ ] OT-P0-009 | Review scoring | The system shall record a review run against the post type's declared failure modes with a per-mode verdict and supporting evidence
- [ ] OT-P0-010 | Publish ledger | The system shall record what shipped — draft, channel, URL, post identifier, series, and prior post — and shall serve that history as a query surface
- [ ] OT-P0-011 | Subject familiarity state | The system shall answer whether a named subject has already been introduced to a given audience
- [ ] OT-P0-012 | Narration ledger | The system shall answer what has already been narrated about a given scenario so that a later draft advances rather than repeats it
- [ ] OT-P0-013 | Idempotent state import | The system shall import existing marketing team state from its current file storage without duplicating any item on re-run
- [ ] OT-P0-014 | Operator workbench | The system shall provide a UI to work the draft queue, edit a draft, resolve claims, read review output, and approve

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Claim re-verification sweep | The system should re-run stored checks on a schedule and should mark a claim stale when its result changes
- [ ] OT-P1-002 | Published-post contamination report | The system should list every published post that cites a claim which has become stale or refuted
- [ ] OT-P1-003 | Novelty claim expiry | The system should require a search date on novelty claims and should return them to unverified once that date passes a configured age
- [ ] OT-P1-004 | Coverage reporting | The system should report marketing coverage by campaign, lane, channel, and SKU, including staleness
- [ ] OT-P1-005 | Account eligibility check | The system should consult the scheduler for whether an account is eligible for a lane before a draft targeting that account is approved
- [ ] OT-P1-006 | Publish handoff contract | The system should hand an approved draft to the scheduler and should record the returned URL and post identifier against the ledger
- [ ] OT-P1-007 | Assisted claim extraction | The system should propose claims extracted from a draft body for operator confirmation, as a cross-check on author self-reporting
- [ ] OT-P1-008 | Federated retrieval registration | The system should register drafts and publish history as a search-hub provider so they are reachable from federated query

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Publish performance ingestion | The system may ingest per-post engagement telemetry once social accounts and measurement sources exist
- [ ] OT-P2-002 | Per-channel variant derivation | The system may derive channel-specific variants from a single approved draft
- [ ] OT-P2-003 | Rich-media asset linkage | The system may link drafts to rendered image and video assets produced by a future asset-production scenario
- [ ] OT-P2-004 | Outcome-driven hook promotion | The system may propose hook-library promotions from measured post outcomes rather than editorial taste

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC proto contracts, Go CLI over cli-core primitives, React + TypeScript + Vite UI — the standard react-vite scenario shape. The draft lifecycle is modelled as a Level 5 formal flow through flow-verifier, because its illegal transitions are the product.
- Data + storage expectations: SQLite in-process for campaigns, drafts, claims, evidence, review runs, and the ledger. No external storage resource. Credentials are never stored here in any form.
- Integration strategy: ai-gateway for assisted claim extraction at P1 only, so the P0 path works with inference unavailable; social-media-scheduler for publishing and account eligibility; search-hub for federated retrieval; vrooli-events receipts arrive automatically through api-core with no integration work. Third-party packages are governed exclusively through Scenario Dependency Analyzer.
- Non-goals / guardrails: This scenario does not generate copy — generation prompts stay in the paired post-type skills so they can change without a deploy. It does not publish to platforms, does not own account identity, warming, or credentials, does not own visual identity or rendered assets, and does not own strategic marketing canon, which remains operator-curated and decision-gated. Approval is never automated.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite in-process. No external resource dependency at P0.
- Scenario dependencies: social-media-scheduler (publish handoff and account eligibility, P1); ai-gateway (assisted claim extraction, P1, optional); search-hub (federated retrieval registration, P1); prompt-manager (read-only — team, decisions, and the paired post-type skills); vrooli-events (run correlation, automatic).
- Operational risks: Operator approval becomes the throughput bottleneck once agents are freer and campaigns supply a backlog, which is a constraint on attention rather than capacity and degrades review quality before it shows up as a queue; the claim taxonomy is uncalibrated until a real draft passes through it, so what counts as a claim may be wrong in both directions; author self-reported claims are only as honest as the author, and the compensating control is that finding undeclared claims is an explicit charge of the review pass; a stored check can pass while the claim it supposedly proves is false, so evidence strength is a floor and not a guarantee; and retiring the prior scenario leaves references in six skill files plus one marketing catalogue document that must be resolved rather than left dangling.
- Launch sequencing: One post published manually end to end first, to calibrate the claim taxonomy and the workbench against real friction rather than an imagined queue. Then the ledger and the idempotent import, which produce value from existing state immediately and prove the transport. Then drafts with the formal lifecycle. Then claims and the verification gate. Then the post-type registry and its activation gate. Then review scoring. Then the operator workbench. The team restructure that this scenario enables is deliberately not part of this sequence: it is a separate, decision-gated change that should follow observed behaviour.

## 🎨 UX & Branding

- Look & feel: Vrooli Operational Console per root DESIGN.md — calm, dense, technical, slate neutrals with blue for primary commands and cyan for technical emphasis; light, dark, and system modes are first-class. The signature surface is a three-pane desk: queue, draft editor, and a permanently docked inspector holding claims and review verdicts, because the inspector is the reason the scenario exists and must not be hidden behind a modal. The second surface is a dense publish ledger with series linkage and coverage staleness.
- Accessibility: WCAG AA contrast in both themes, visible focus states, 44px touch targets, no status conveyed by colour alone, reduced motion respected, and full keyboard reachability for queue, edit, claim resolution, and approval. On mobile the panes become stepwise panels with preserved context rather than a shrunken desk.
- Voice & messaging: Precise and operational. A disabled approval control always states its reason inline — an unresolved claim, an inactive post type, an exhausted slot budget — because a dead control with no explanation is the failure mode this surface exists to prevent. Claims are shown with their evidence and its strength, never as a bare boolean.
- Branding hooks: Inherits the vrooli-default design kit; replace the generic PWA icons when product branding exists. Keep the seeded web app manifest, service worker, maskable icons, relative install asset URLs, and safe-area tokens valid.
