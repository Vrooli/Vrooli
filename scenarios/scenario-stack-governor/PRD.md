# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Centralize and enforce tech-stack “rules of the road” so scenario generators and improvers don’t accidentally break core invariants (buildability, lifecycle correctness, harness wiring).
- **Primary users/verticals**: Vrooli maintainers, scenario authors, and agent workflows that generate/modify scenarios.
- **Deployment surfaces**: UI (rule catalog + toggles), API (run rules + fetch results), CLI (optional automation/CI hook), integration as an external rule pack for `scenario-auditor`.
- **Value promise**: Reduce repo-wide blast radius by catching stack drift early (e.g., Go workspace coupling, broken test harness configs) and explaining “why” so fixes are repeatable.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Rule enablement config | Persist which rules are enabled in a human-editable config file (and editable via UI/API).
- [ ] OT-P0-002 | Rule catalog UX | Provide an intuitive UI that lists each rule, explains what it checks and why it matters, and lets operators toggle rules.
- [ ] OT-P0-003 | Run rules + results | Run enabled rules on demand and return actionable, evidence-backed results.
- [ ] OT-P0-004 | Go workspace independence | Enforce that Go-based scenario CLIs build with `GOWORK=off` (no dependency on a repo-level `go.work`).

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Scenario-auditor integration | Expose this scenario as an external rule pack consumable by `scenario-auditor` (JSON contract + remediation hints).
- [ ] OT-P1-002 | Stack packs | Add rule packs for JavaScript/TypeScript (Vitest/Jest harness correctness), lifecycle invariants, and common “gotchas”.

### 🟢 P2 – Future / expansion ideas
- [ ] OT-P2-001 | Policy profiles | Support “profiles” (e.g., strict CI, local dev, generator mode) that enable different rule sets.
- [ ] OT-P2-002 | Guided remediation | Offer “fix it” suggestions and optional auto-patches for well-scoped rule violations.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: React + TypeScript UI, Go API; config-first rule packs that can be called by other scenarios.
- Data + storage expectations: File-backed config for enabled rules (no database required for MVP).
- Integration strategy: `scenario-auditor` consumes rule pack outputs; other scenarios can call the API/CLI for preflight checks.
- Non-goals / guardrails: Not a replacement for `scenario-auditor`; not a general CI system; no automatic package installs.

## 🤝 Dependencies & Launch Plan
- Required resources: none for MVP (rule runs operate on the repo filesystem + local toolchains).
- Scenario dependencies: `scenario-auditor` (integration target), `scenario-completeness-scoring` (quality telemetry).
- Operational risks: Rules that run builds can be slow; keep checks scoped and time-bounded, and make rule toggles explicit.
- Launch sequencing: MVP ships Go rule + config + UI toggles; then wire into `scenario-auditor`; then expand rule packs.

## 🎨 UX & Branding
- Look & feel: “control tower” clarity—rule cards with concise summaries, expandable “why this matters”, and obvious toggle/run affordances.
- Accessibility: Keyboard navigable toggles; readable contrasts; clear error messages.
- Voice & messaging: Educational and concrete—each rule explains failure modes and remediation.
- Branding hooks: Reuse Vrooli developer-tools styling conventions (Tailwind + shadcn primitives).

## 📎 Appendix (optional)
[Add references or research links here if needed.]
