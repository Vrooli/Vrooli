# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own every platform identity the fleet operates and every action taken as one — warming an account into algorithmic trust, keeping its cadence inside platform limits, watching its distribution for decay, and releasing approved content through whichever executor is available. One action queue per identity, because platforms count actions per account and not per workflow.
- **Primary users/verticals**: The operator running Vrooli's brand and persona-actor accounts across TikTok, Instagram, X, LinkedIn, YouTube, Threads, Bluesky, and Reddit; the `marketing-crew` producer that needs to know whether an account may carry a lane; `content-desk`, which hands off approved drafts and asks a single eligibility question.
- **Deployment surfaces**: UI (identity roster, today's due actions, warming progress, signal history), CLI (agent- and operator-facing verbs), API (Connect-RPC, called by `content-desk`), and a scheduled sweep that materializes due actions.
- **Value promise**: Makes multi-account operation survivable. Warming, cadence discipline, and distribution monitoring are the difference between an account that reaches its audience and one throttled permanently — and none of it is tracked anywhere today. It also gives `content-desk` the one answer it cannot compute for itself: whether this identity has earned the right to post this lane.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Identity registry | The system shall persist each platform identity with its platform, purpose tag, persona reference, lane grants, and lifecycle status.
- [ ] OT-P0-002 | Credential isolation | The system shall store only a vault reference for an identity and shall reject any write carrying a credential value.
- [ ] OT-P0-003 | Declarative platform descriptors | The system shall load per-platform formats, limits, cadence ceilings, and disclosure requirements from validated descriptors, so that adding a platform requires no code change.
- [ ] OT-P0-004 | Declarative warming programs | The system shall express a warming program as a validated descriptor carrying preconditions, session policy, phases, gates, graduation, maintenance, and provenance.
- [ ] OT-P0-005 | Precondition gate | The system shall refuse to start a warming program for an identity until every required precondition is recorded as satisfied.
- [ ] OT-P0-006 | Unified action queue | The system shall route every platform action for an identity — warming, engagement, and publishing alike — through one queue.
- [ ] OT-P0-007 | Cadence enforcement | The system shall refuse to schedule an action that would breach the identity's per-platform ceiling counted across all action kinds combined.
- [ ] OT-P0-008 | Session scheduling | The system shall group scheduled actions into sessions honouring the program's session count, duration, and minimum inter-session gap.
- [ ] OT-P0-009 | Recorded randomization | The system shall roll each action's count and time within the declared ranges and shall persist the roll and its seed.
- [ ] OT-P0-010 | Phase action guards | The system shall reject any queued action that the identity's active warming phase forbids.
- [ ] OT-P0-011 | Gate evaluation with terminal quarantine | The system shall evaluate a program gate against its declared measurement and shall support a terminal quarantine outcome that stops all scheduled actions for that identity.
- [ ] OT-P0-012 | Graduation by earned criteria | The system shall grant lane eligibility only when every graduation criterion passes, and shall never grant it on elapsed time alone.
- [ ] OT-P0-013 | Eligibility query | The system shall answer whether an identity is eligible for a lane, returning unknown on any internal failure and never returning eligible by default.
- [ ] OT-P0-014 | Idempotent release | The system shall accept an approved draft, return its post identifier and URL, and shall not create a second publish record when a release is retried.
- [ ] OT-P0-015 | Manual executor | The system shall render any queued action as an operator checklist item and shall accept completion with optional evidence.
- [ ] OT-P0-016 | Signal capture and baseline | The system shall record per-identity distribution metrics over time and shall maintain a rolling baseline per metric.
- [ ] OT-P0-017 | Flag on distribution decay | The system shall raise a flag carrying the evidence that triggered it, shall pause that identity's queue, and shall take no corrective action automatically.
- [ ] OT-P0-018 | Program provenance surfaced | The system shall carry each program's provenance and shall expose it wherever that program is displayed or applied.
- [ ] OT-P0-019 | Operator console | The system shall provide a UI for the identity roster, the day's due actions, warming progress, and signal history.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Browser executor | The system should execute queued actions through `browser-automation-studio`, binding each identity to its own session profile.
- [ ] OT-P1-002 | Portfolio separation | The system should enforce minimum posting separation across identities so that portfolio activity does not appear coordinated.
- [ ] OT-P1-003 | Asset uniqueness | The system should refuse to queue a post carrying an asset already published by another identity.
- [ ] OT-P1-004 | Disclosure enforcement | The system should require the platform's declared AI-content disclosure on any post from a persona-actor identity.
- [ ] OT-P1-005 | Retry classification | The system should classify execution failures as retryable or terminal per platform and should back off accordingly.
- [ ] OT-P1-006 | Observation capture | The system should append run outcomes to the program's observation log so that unvalidated defaults can be revised from evidence.
- [ ] OT-P1-007 | Publish metrics handoff | The system should return post performance to `content-desk` so that its ledger can answer how published work performed.
- [ ] OT-P1-008 | First-comment atomicity | The system should treat a post and its first comment as one unit of work, and should report partial completion rather than success when the comment does not land.
- [ ] OT-P1-009 | Per-platform post preview | The system should render how a post will appear on its target platform — truncation, media cropping, and disclosure placement — before it is released.
- [ ] OT-P1-010 | Environment liveness | The system should periodically verify that an identity's environment still presents the attested region and should flag divergence.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Platform API executors | The system may execute actions through official platform APIs where one exists and the identity qualifies.
- [ ] OT-P2-002 | Cannibalization detection | The system may detect audience overlap across persona identities operating in the same niche.
- [ ] OT-P2-003 | Window optimization | The system may propose posting windows derived from an identity's observed reach rather than from a declared window.
- [ ] OT-P2-004 | Multi-operator handoff | The system may assign manual actions to a named operator and track completion per assignee.
- [ ] OT-P2-005 | Deliberate re-release | The system may release a previously published draft again as a distinct release with its own record, so that cadence can be held without producing new material.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC proto contracts, Go CLI over cli-core primitives, React + TypeScript + Vite UI — the standard react-vite scenario shape, matching `content-desk` and `asset-studio`.
- Data + storage expectations: SQLite in-process for identities, queue, sessions, signals, and program state. Platform and warming-program descriptors are versioned JSON files under `data/`, validated against schemas and seeded at boot; the descriptor file is the source of truth and the table is a cache.
- Integration strategy: `vault` for credentials through its `content` CLI, never a stored token. `browser-automation-studio` for the browser executor, including per-identity session profiles. `content-desk` calls in for release and eligibility. `asset-studio` answers asset-uniqueness. No direct platform SDK at P0.
- Non-goals / guardrails: Not a content authoring tool — copy, claims, and approval belong to `content-desk`. Not an asset renderer — that is `asset-studio`. Not a strategy surface — channel priority and account purpose are canon in `docs/marketing/strategy/CHANNELS.md`. The scenario does not provision the environment an identity runs in (device, proxy, region); it records and checks it. It never asserts that an account has been penalized — it reports measurements and flags.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite in-process; `vault` for credential references.
- Scenario dependencies: `content-desk` (release handoff and eligibility, bidirectional), `browser-automation-studio` (P1 executor and session profiles), `asset-studio` (P1 asset-uniqueness lookup), `vrooli-events` (run correlation, automatic through api-core).
- Operational risks: Every warming threshold shipped at P0 is unvalidated operator folklore rather than platform documentation, so the programs are hypotheses and the observation log is how they stop being hypotheses. Browser automation of account actions is against most platforms' terms of service and enforcement is account suspension, which makes the executor choice a recorded operator decision per platform rather than a default. Distribution decay is inferred from measurements and can never be confirmed, so a flag is evidence and not a verdict. The environment an identity depends on — device fingerprint, residential proxy, region consistency — sits upstream of this scenario, and a leak there degrades distribution in ways nothing here can detect or repair. Warming spans weeks, so a defect in gate evaluation surfaces slowly and costs an account rather than a retry.
- Launch sequencing: identities, platform descriptors, and credential isolation first; then the unified queue with cadence enforcement and the manual executor, which is the point at which the scenario is usable end to end without any automation; then warming programs, plan generation, and session scheduling; then signals and baselines, which must precede gates because a gate is evaluated against a measurement this scenario has to already be collecting, and which need history before they mean anything; then gates, quarantine, and graduation; then the eligibility query and release handoff that `content-desk` is already written against; then the operator console. The browser executor is deliberately last of the P1 set — it is the only piece carrying terms-of-service exposure, and everything above it must work manually first.

## 🎨 UX & Branding

- Look & feel: Vrooli Operational Console per root `DESIGN.md` — calm, dense, technical, slate neutrals with blue primary and cyan technical emphasis; light, dark, and system modes first-class. The signature surfaces are an identity roster showing warming stage and health at a glance, a day view of due actions that reads as a work list rather than a dashboard, and a signal history where the baseline is visible behind the series.
- Accessibility: WCAG AA contrast in both themes, visible focus states, 44px touch targets, no status conveyed by colour alone, reduced motion respected, and full keyboard reachability for completing an action and recording evidence.
- Voice & messaging: Operational, and non-committal about inference. A flag reads "reach below 40% of 14-day baseline for 3 consecutive posts", never "shadowbanned". A warming program always displays its provenance and confidence beside its numbers, so nobody mistakes folklore for platform documentation.
- Branding hooks: Inherits the vrooli-default design kit; replace the generic PWA icons when product branding exists.

## 📎 Appendix

- Counterpart contract: `scenarios/content-desk/docs/concepts/INTEGRATIONS.md` specifies this scenario's boundary from the other side — exactly two questions cross it, eligibility never fails open, and release is idempotent.
- Canon: `docs/marketing/strategy/CHANNELS.md` (per-platform rules, account purpose tags, and the rule that credentials and handles never live in canon) and `docs/marketing/strategy/patterns/ai-ugc-personas.md` (persona disclosure and the banned-claim set).
- Predecessor: `social-media-scheduler` was retired 2026-07-28. It had not compiled since 2025-09-08 and had no consumers; nothing was migrated.
