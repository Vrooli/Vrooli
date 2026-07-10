# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario experience-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Experience Manager is the permanent experience-axis authority: it owns the machine-checkable UX spec contract (a root `experience/` folder in every scenario declaring what each page must communicate), the form-based authoring and workshop studio for those specs, and validation of built UIs against declared experience intent. It gives the experience track the same design-first, testable, target-rollup rigor that business-health gives the business-logic track.
- **Primary users/verticals**: Scenario authors designing UX before building it, internal engineering agents scaffolding and validating scenario UIs, operators tracking fleet-wide experience debt, and the test-genie suite as the delegated `experience` phase consumer.
- **Deployment surfaces**: Go Connect API, `experience-manager` CLI, operational React UI (Fleet, Scenario Explorer, Evidence, Studio, Findings), Test Genie delegated validation provider via `.vrooli/test-genie.json`, and BAS-driven capture automation.
- **Value promise**: UX stops being trapped after generation. Every scenario's experience intent becomes a validated contract: specs are authored cheaply in a visual workshop, claims are checked deterministically against what the built UI actually exposes, and unproven experience promises are visible debt instead of silent drift.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Experience spec contract | Validate `experience/` (index, pages, journeys, components) against `.vrooli/schemas/scenario-experience-spec.schema.json` plus parser checks: unique IDs, resolvable references (routes, PRD OTs, DESIGN.md states, bindings, component examples), enforcement-tier rules, and open-world semantics, with a frozen finding-code vocabulary.
- [x] OT-P0-002 | Scenario validation provider | Implement `ScenarioValidationService` for the canonical `experience` phase with findings, metrics, maturity ladders, and fix preview/apply, discovered by Test Genie through `.vrooli/test-genie.json` with presence-keyed applicability.
- [x] OT-P0-003 | Structure reconciliation | Drive Browser Automation Studio to capture a screenshot plus accessibility tree per declared page/state and per reusable component example, check machine-tier claims against the captured tree, and persist per-claim evidence; BAS unavailability yields skipped findings, never failures, and component reconciliation remains advisory until promoted.
- [x] OT-P0-004 | Authoring studio | Form-based spec authoring with live validation and CLI parity, satisfying the round-trip property: a spec saved through the studio re-validates to zero contract findings.
- [x] OT-P0-005 | Self-spec dogfood | Experience Manager ships its own `experience/` folder at L3+ depth and validates green, serving as the first real proof of the spec schema on its own surfaces.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Render and workshop | Deterministic wireframe rendering of any spec page with claim annotations, side-by-side variant compare with promote-to-spec, and an optional AI-image render mode composing image-tools.
- [x] OT-P1-002 | BAS case scaffolding | Derive `bas/cases` stubs from spec entries (routes to navigation, claims to assertions, bindings to selectors) for workflow-health governance, with spec-to-case reference-integrity findings in both directions.
- [x] OT-P1-003 | Deterministic autofix | Preview and apply mechanical fixes for binding drift, index normalization, missing state stubs, and finding-doc stubs with accounting of fixed versus remaining findings.
- [x] OT-P1-004 | Manual attestation ledger | Manual-tier claims are attested with an expiry; expired attestations surface as findings so human-verified experience promises decay honestly instead of staying green forever.
- [x] OT-P1-005 | Fleet sweep | Compute-on-read sweep of spec coverage and depth across all scenarios, scored worst-first so experience debt is visible and prioritized fleet-wide.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Perception tier | Pixel-side parsing and saliency-based importance scoring checked against declared communication priorities, running advisory before ever gating and quarantined off the CI hot path, providing the promotion path from aspirational to machine claims.
- [ ] OT-P2-002 | Journey coherence | Runtime journey validation (steps executable, states reachable, friction budget respected) built on workflow-health execution rather than a second browser harness.
- [ ] OT-P2-003 | Search leaf | Specs and claims discoverable through search-hub so agents can answer which pages promise a given experience outcome.

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go Connect API, manifest-bound CLI, react-vite React UI on the vrooli-default design kit.
- **Spec artifact**: JSON files under a root `experience/` folder per scenario, validated by `.vrooli/schemas/scenario-experience-spec.schema.json` (JSON schema for artifacts; proto/Connect only for service endpoints). Claim model is open-world (anything unclaimed is free), uses WAI-ARIA roles as the element vocabulary, separates stable intent from volatile grounding bindings, and carries per-claim enforcement tiers: machine (gates), manual (attested with expiry), aspirational (advisory, never rejected).
- **Parser home**: internal package (`api/internal/spec`); no shared Go package until a second in-repo consumer actually exists — cross-scenario access goes through the CLI/API.
- **Data storage expectations**: SQLite for studio sessions, attestations, and capture-evidence references; the spec itself stays in-repo as reviewable files.
- **Integration strategy**: BAS is the single capture engine (screenshot + accessibility tree per page/state); Test Genie integration is entirely declarative via `.vrooli/test-genie.json`; DESIGN.md's required UX-state contract feeds state-coverage checks as an input.
- **Non-goals**: third-party surface fingerprinting, photo/receipt extraction, deciding the pixel-perception engine's home, absorbing workflow-health, and owning DESIGN.md validation (it is consumed, not owned).

## 🤝 Dependencies & Launch Plan

- **Resources**: none required in v1; qdrant/ollama/reranker enter only with the P2 search leaf.
- **Scenario dependencies**: browser-automation-studio (capture engine; degraded = reconciliation skipped, never failed), image-tools (optional AI-image rendering via its openrouter-image provider), workflow-health (file-level seam: scaffolded `bas/cases` stubs land under its governance), business-health (sibling axis; validates this scenario's own PRD/requirements).
- **Prerequisite feature**: BAS must add accessibility-tree capture per execution step alongside screenshots — filed as a cross-scenario request before reconciliation work starts.
- **Risks**: spec brittleness is the make-or-break — claims that encode pixel incidents instead of intent will flap on every restyle. Mitigation: intent-level claim vocabulary, the intent/bindings split, and a hard spike gate before broad rollout: the schema must express business-health's Matrix page, web-console's terminal surface, and this scenario's own Studio naturally, or the vocabulary gets fixed first.
- **Sequencing**: v1 is deliberately zero-ML (contract + provider + reconciliation + studio + dogfood), then P1 workshop/scaffolding/fleet, then the P2 perception tier. The `experience` phase launches with presence-keyed applicability (spec absence is fleet-sweep debt, not a failed suite) and promotes to broader gating only after the fleet has specs. Template scaffolding of `experience/` and start/orient doc updates ship as implementation-plan rollout work, not as scenario capabilities.

## 🎨 UX & Branding

- **Look and feel**: dense operational console on the vrooli-default kit — data-forward, keyboard-friendly, built for operators scanning fleet state and drilling into evidence. The Studio (split-pane spec form beside a live annotated wireframe, N-variant compare) is the one creative surface and receives the design investment.
- **House principle — evidence-first**: no green check without a click-through to its evidence; every satisfied claim links to the capture region and accessibility-tree node that proved it.
- **Honest tiers**: aspirational claims are visually distinct from machine-verified ones everywhere they appear; the UI never dresses an unproven promise as assurance.
- **Accessibility**: WCAG AA contrast targets, full keyboard navigation, and RTL-aware layouts; i18n ships en/ja/ar like sibling health scenarios. As the scenario that judges other UIs' communication, its own surfaces must clear the bar it enforces.
- **Voice**: plain, specific, non-alarmist — findings state what diverged from the declared intent and what unlocks the next maturity level, mirroring business-health's ladder language.
