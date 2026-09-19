# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario personal-planner`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose:** Personal Planner is a personal planning and focus application that helps people make realistic commitments, follow through on them, and plan better from what actually happens. It connects goals, projects, milestones, tasks, routines, commitments, calendars, focus sessions, and learning into one honest model of a person's time. It exists to improve visibility *before* commitments fail — replacing repeated optimistic over-commitment with a truthful picture of usable time, what one more task displaces, and why a forecast changed.

**Primary users / verticals:** Individual knowledge workers, founders, makers, students, and anyone who repeatedly over-commits and needs to see overload before it happens. R1 targets a single user with a private workspace; household and team collaboration are deliberately deferred (D03). Personal Planner also serves as the shared scheduling and availability owner for other Vrooli applications — the food-planning app ("Daily") and content-planning app ("Cadence") are the originating integration examples (D07).

**Deployment surfaces:** Web UI (the primary experience — the approved "Observatory" day/night design), Go API (the one authoritative command/query surface behind every caller), CLI/tools (bounded domain verbs at parity with the UI), and shared scheduling contracts consumed by other Vrooli scenarios. Progressive web-app install and desktop packaging follow the platform defaults.

**Value promise:** In a crowded market of calendars and to-do apps, Personal Planner differentiates on two things competitors miss: (1) truthful accounting — capacity, remaining effort, dependencies, delays, and explainable forecasts that never silently lie green; and (2) exceptional, calm, beautiful UX. Beauty is an explicit product requirement, not decoration: the coordinated day/night Observatory landscape is the wedge that earns repeated daily use, and repeated use is what makes the honest planning loop valuable.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Private persistent workspace | An individual user gets an isolated, identity-scoped workspace whose native tasks, events, and settings survive service and browser restart; a second identity cannot read them (plan REQ-01/REQ-21, INV-01).
- [ ] OT-P0-002 | Correct time semantics | Civil dates, instants, named timezones, DST transitions, all-day boundaries, and recurrence identity are stored as distinct concepts and never collapsed into one field (plan REQ-05, INV-03; RFC 5545).
- [ ] OT-P0-003 | Hybrid planning without double-counting | Fixed events, flexible timed sessions, date-assigned effort, and an unscheduled backlog coexist; splitting or moving an allocation conserves demand rather than cloning it (plan REQ-03/D02, INV-07).
- [ ] OT-P0-004 | Truthful interval-aware capacity | Capacity is both a budget and a set of intervals: overlapping busy events subtract their union, reserves count once, unknown estimates are listed separately, and no unqualified 'everything fits' claim is made (plan REQ-09, fixtures F01-F03).
- [ ] OT-P0-005 | Honest Today experience | A Today view answers 'what can I usefully do now?' with a next-action surface, scoped capacity, and commitment notices — fully usable manually before any automation, provider, or model is configured (plan REQ-01/REQ-23).
- [ ] OT-P0-006 | Observatory day/night experience | The approved Observatory visual direction ships with coordinated day and night landscapes, Auto/Day/Night controls, truthful time geometry, and WCAG 2.2 AA accessibility as a visual release gate (plan REQ-02/REQ-22/D08, §6).
- [ ] OT-P0-007 | One authoritative command surface | Every mutation flows through one application-command layer behind HTTP/CLI/tools/adapters with actor+workspace from authenticated context, idempotency keys, expected-revision checks, and atomic local writes (plan REQ-21, §20).

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Goals, milestones, commitments, dependencies | Native goals with explicit progress method, evidence-based milestones, explicit promises with full revision/acknowledgment history, and finish-to-start dependencies with waits and passive work (plan REQ-06/REQ-07/REQ-08, §9).
- [ ] OT-P1-002 | Deterministic proposal engine with review/apply/undo | A pure scheduling kernel returns explainable candidate placements that never write accepted records; apply is a separate revalidated, idempotent, atomic command with conditional undo (plan REQ-10, §11).
- [ ] OT-P1-003 | Explainable scenario forecasts | One shared personal resource model produces central/cautious scenario finishes (not false probabilities), with freshness, input fingerprints, risk state, and compact change explanations separating evidence from cause (plan REQ-11, §12).
- [ ] OT-P1-004 | Side-effect-free what-if | Add-a-task, reduce-scope, move-a-target, protect-a-day, and change-available-hours scenarios compare against the accepted baseline without writing sources, commitments, shares, or notifications (plan REQ-12, §11.5).
- [ ] OT-P1-005 | Durable focus sessions and correctable actuals | Four focus modes, at-most-one exclusive session per person across devices, server-durable timers, interruption capture, and manual/approximate actuals with preserved correction provenance (plan REQ-13/REQ-14, §13).
- [ ] OT-P1-006 | Daily/weekly review and evidence-linked learning | Planned-vs-actual review with visible coverage, selective carry-forward, and conservative descriptive recommendations that only change settings on explicit acceptance (plan REQ-15, §14).
- [ ] OT-P1-007 | Shared Vrooli scheduling contracts | Source registration, normalized scheduling-intent projection, and schedule command/query contracts let Daily/Cadence and agent work submit intent and read accepted schedules through one owner (plan REQ-16, §15).
- [ ] OT-P1-008 | Read-only external calendar integration | At least one real provider (Google fallback target) connects read-only, importing external obligations into capacity with honest freshness, incremental sync, and stale-busy behavior (plan REQ-17, §16).
- [ ] OT-P1-009 | Selective read-only sharing and notifications | Owners share chosen commitment fields to a recipient-bound viewer with server-side field masks and revocation, plus useful deduplicated notifications through the ecosystem owner (plan REQ-18/REQ-19, §17-18).
- [ ] OT-P1-010 | Portability and safe restore | Versioned native export, actuals CSV, calendar ICS, explicit permanent deletion, and validated import/preview that never reactivates shares, credentials, or a running timer (plan REQ-20, §23).
- [ ] OT-P1-011 | Measured performance and operator runbook | Bounded reference-scale datasets, p95 targets for Today/commands, coalesced forecasts, redacted diagnostics, and a real deployment/migration runbook with rollback (plan REQ-24, §23.6-23.8).

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Calibrated statistical forecasts | Probability-labeled completion forecasts with a documented method, held-out evaluation, and calibration reporting — only where enough relevant data exists (plan R2, §12.1).
- [ ] OT-P2-002 | Conversational planning assistant | An optional model-backed interface over the same bounded, validated commands — never a second scheduling engine and never required for the core loop (plan R2, §18.3).
- [ ] OT-P2-003 | Richer what-if and scheduling search | Multi-scenario comparison, advanced constraint search, and richer learning controls backed by measured evidence (plan R2).
- [ ] OT-P2-004 | Opt-in automatic flexible rescheduling | User-authorized automatic movement of flexible sessions within declared bounds, still preserving fixed commitments and promise history (plan R3).
- [ ] OT-P2-005 | Household and team collaboration | Shared household editing, team capacity/assignment, and multi-person availability aggregation as a deliberate product expansion beyond single-user R1 (plan R3, D03).
- [ ] OT-P2-006 | Two-way provider writeback | Optional provider event mutation with clear ownership and conflict semantics, well beyond R1's strictly read-only integration (plan R3, §2.3).
- [ ] OT-P2-007 | Commercial billing and subscription | Metered plans and billing if the product is taken to market, gated behind proven retention from the beautiful daily-use loop (plan R3, monetization).

## 🧱 Tech Direction Snapshot

**Preferred stacks / frameworks:** Follow the Vrooli react-vite scenario shape — Go API behind generated Connect-RPC proto services, React + TypeScript + Vite UI over the react-component-library shell, and a Go CLI at command parity with the UI. The deterministic scheduling/capacity/forecast logic lives in a pure, side-effect-free Go kernel that accepts an immutable normalized snapshot and returns candidate placements with reasons; persistence, auth, HTTP, and rendering stay outside it (plan §22).

**Data + storage expectations:** Start with the scenario-owned SQLite substrate (schema co-located with each domain via SchemaProvider/EnsureSchemas). Store authoritative current state plus revisioned history for meaningful edits and an append-only change/outbox log; read models are disposable projections, never a second authority. Temporal and effort types are explicitly modeled (Instant, CivilDate, LocalTime, TimeZoneId, opaque Revision, Effort/point/range, TemporalExtent) and never coerced. A relational engine with transactions, unique constraints (e.g. the partial-unique one-exclusive-session guard), and explicit indexes is the fallback if a shared resource is ever justified — no graph DB or event-sourcing platform for R1 (plan §19).

**Integration strategy:** Reuse over rebuild — shared Vrooli owners first (scenario-authenticator for identity, notification-hub for delivery, the existing event transport, credential storage for provider tokens), then a scenario CLI/resource, then direct API. Personal Planner is the single scheduling authority: other scenarios submit normalized scheduling intent and read accepted schedules through contracts; a shared database is never the integration API. External calendars are strictly read-only in R1 (plan §3, §15, §16).

**Non-goals / guardrails:** No two-way external sync, automatic promise renegotiation, employee monitoring, screenshot/keystroke capture, medical/energy conclusions, billing, group resource optimization, or unrestricted autonomous agents in R1 (plan §2.3). Guard equally against the two failure modes: a generic calendar with extra tabs (missing effort/commitment/delay/capacity/learning) and an all-purpose project-management platform that duplicates domain owners. The app must be fully useful with no AI provider configured; a missing model never breaks the core workflow (plan §2.4, D09).

## 🤝 Dependencies & Launch Plan

**Required resources:** None mandatory for R0 — SQLite is in-process. A relational resource (e.g. Postgres) is adopted only if measured scale or the partial-unique concurrency guard justifies it, recorded through the Scenario Dependency Analyzer, not hand-edited.

**Scenario dependencies:** scenario-authenticator (identity, private-workspace isolation, and the recipient-bound viewer identity for sharing — required); notification-hub (reminder and risk-notice delivery with quiet hours and receipts — optional, degrades to in-app only); the shared event transport (reuse envelope and receipts, never start a private bus); credential storage owner (read-only provider tokens, never in logs/bundles/exports). Daily and Cadence are *consumers* of Personal Planner's scheduling contracts, not upstream dependencies; where a consumer scenario is absent, ship the contract + fixture adapter and keep that live-integration gate explicitly open (plan §15.8, §3).

**Operational risks:** duplicate scheduling owners writing the same schedule (mitigate with the P00 owner map and a single write authority); attractive visuals concealing wrong arithmetic (mitigate with shared domain values and fixtures F01-F03/F14); tracking friction driving abandonment (manual path, optional wrap-up, brief reviews); forecasts appearing more certain than inputs (scenario labels, unknown/stale states); private causes leaking through sharing (server-side DTOs, redaction, network-level tests). Full register: plan §28.3.

**Launch sequencing:** P00 repository discovery and owner map -> P01 durable foundation + Observatory shell (R0 begins) -> P02 temporal model + manual hybrid planning + shared-contract foundation (R0 checkpoint) -> P03 goals/milestones/commitments/dependencies -> P04 capacity + honest Today -> P05 proposal engine + forecasts -> P06 focus + actuals -> P07 review + learning -> P08 real source + read-only calendar integration -> P09 sharing + notifications -> P10 portability + reliability + CLI parity -> P11 verified, polished R1 with migration/cutover evidence. R1 is the finish line; R2/R3 are expansion (plan §26).

## 🎨 UX & Branding

**Look & feel:** The approved 'Observatory' direction (D08). A small hillside observatory overlooks a valley, lake, and layered mountains; the same viewpoint renders as warm sunlit terrain by day and dark silhouettes with restrained settlement lights at night. The scene sits mostly below the working area, creating a sense of place without ever competing with text, inputs, or the timeline. Expressive editorial serif headings (Fraunces-class) pair with a highly readable UI sans and tabular numerals for times and capacity. Restrained radii, subtle day shadows, night surface/border separation; no neon, glass, decorative gradients, or baked-in text. Auto/Day/Night is user-controlled; Auto uses editable transition hours (default 07:00/19:00) and defers transitions during a focus session. Working label 'Planner'/'Observatory' is replaceable configuration, not an approved permanent product name.

**Accessibility:** Target WCAG 2.2 AA and treat it as a release gate. Verify >=4.5:1 normal-text contrast in both appearances and in the brightest/darkest landscape regions, relevant non-text contrast, and target-size rules (~44px touch targets where practical). Every drag has a keyboard and menu alternative; timer start/stop needs no precise pointer; meaningful timer-state changes are announced without narrating every second. Visible focus, sensible focus return from dialogs, semantic headings, reflow, text zoom to 200%, honored reduced-motion, and no color-only status. Time geometry is truthful: position and width represent real time; a 45-minute block never occupies 90 minutes to fit its label. Review Today in both appearances, a busy week, an empty workspace, a long title, overlapping fixed events, a commitment at risk, a running focus session, a narrow phone, and 200% zoom (plan §6.7-6.8, §24.14).

**Voice & messaging:** Calm, honest, and precise. The product distinguishes facts, estimates, forecasts, user reports, and AI suggestions at all times, and never claims an item is objectively the best use of a life. Copy explains what happened, what is still safe, and the next action. No motivational slogans or streak-shaming in the working UI by default.

**Branding hooks:** Keep the seeded PWA/brand assets (web manifest, service worker, icon set, og-image, brand-manager marker) valid and replace generic placeholder icons by selecting a brand through brand-manager once a permanent name is chosen. Source colors (e.g. Cadence purple, Daily green) stay semantically stable across themes even as shades shift for contrast.

## 📎 Appendix

**Source material:** This charter is derived from the operator-supplied 'Adaptive Personal Planner - Complete Vrooli Implementation Plan' (v1.0, 2026-09-18) and two approved Observatory mockups (day and night). The plan's accepted decisions D01-D09, invariants INV-01-INV-16, requirements REQ-01-REQ-24, worked fixtures F01-F16, acceptance cases T01-T36, and packages P00-P11 are the authoritative product contract; this PRD's operational targets map onto them.

**Standards referenced:** RFC 5545 (iCalendar date/recurrence/exception semantics), WCAG 2.2 (accessibility), Google Calendar incremental sync and recurring-events guides, and Microsoft Graph event delta queries (for a future adapter). These inform interoperability/accessibility behavior only; product choices such as reserve percentage, focus defaults, forecast horizons, and learning thresholds are recommendations, not findings from those sources.

**Legacy note:** The unused, undeveloped `calendar` scenario was removed before this scenario was generated; there is no live scheduling data to migrate. If any such data appears later, follow plan §3.3 (stable ID mapping, dry-run counts, provenance marking, rollback accounting) rather than treating old scheduled events as actual activity.
