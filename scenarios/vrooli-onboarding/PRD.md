# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Transform Vrooli's manual setup process into a guided, user-friendly onboarding experience
- **Primary users/verticals**: Non-technical users, new developers, returning configuration managers
- **Deployment surfaces**: Web UI, CLI, REST API
- **Value promise**: Reduce time-to-first-working-agent from hours to under 10 minutes through guided setup

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Scenario-first configuration | The wizard shall select scenarios first, derive resources and host requirements from manifests, and keep system-required scenarios locked on.
- [x] OT-P0-002 | Authoritative operator state | The wizard shall atomically persist operator choices only to `.vrooli/operator-state.json`; it shall not generate service.json or treat database progress as configuration authority.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Credential and readiness guidance | The wizard shall present descriptor-driven credential status, host safeguards, provider availability, and actionable final readiness without rendering secret values.
- [x] OT-P1-002 | Re-enterable operator experience | The wizard shall reload effective operator state on entry and allow an operator to revise any non-deferred setup decision.

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Operating-mode controls | The wizard should expose manifest-recommended per-scenario auto-restart and operating-mode overrides.
- [x] OT-P2-002 | Deferred integrations contract | The integrations step shall clearly identify integration-hub as deferred, accept no fake bindings, and link to the owning capability.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go (API/CLI), React/TypeScript (UI)
- Data + storage expectations: Local file system (.vrooli/), HTTP endpoints for health checks
- Integration strategy: API-first architecture with thin CLI/UI layers
- Non-goals: Custom resource implementations, complex workflow automation

## 🤝 Dependencies & Launch Plan
- Required resources: manifest-derived scenarios, resources, host requirements, and credential descriptors
- Scenario dependencies: tunnel-manager, browser-automation-studio
- Operational risks: API key security, configuration file corruption, health check reliability
- Launch sequencing:
  1. Core API implementation
  2. CLI frontend
  3. Web UI deployment
  4. Integration validation

## 🎨 UX & Branding
- Look & feel: Clean, minimal interface with clear progress indicators
- Accessibility: WCAG 2.1 AA compliance, keyboard navigation support
- Voice & messaging: Friendly, instructional tone avoiding technical jargon
- Branding hooks: Standard Vrooli color scheme and component library

## 📎 Appendix
- Reference: GuidedTour implementation from browser-automation-studio
- Schema: service.json validation requirements
- Health check endpoint specifications
- **Configuration substrate**: [`/docs/configuration/`](../../docs/configuration/) — the source-of-truth contract this scenario implements. New configurability must be documented there before becoming a wizard step.
- **V2 wizard flow + wireframes**: [`docs/WIZARD_FLOW.md`](docs/WIZARD_FLOW.md) — step-by-step plan, UX sketches, and transcripts from the design conversation.
