# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Transform Vrooli's manual setup process into a guided, user-friendly onboarding experience
- **Primary users/verticals**: Non-technical users, new developers, returning configuration managers
- **Deployment surfaces**: Web UI, CLI, REST API
- **Value promise**: Reduce time-to-first-working-agent from hours to under 10 minutes through guided setup

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Resource Configuration Wizard | Complete guided setup flow for coding agents and AI providers with 100% validation coverage
- [x] OT-P0-002 | Configuration State Management | Generate and validate service.json with all dependencies resolved and proper schema compliance

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Health Monitoring Dashboard | Real-time resource status visualization with health checks and status indicators
- [x] OT-P1-002 | Progress Persistence System | Save and resume capability for partial onboarding progress across sessions

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Smart Flow Optimization | Intelligent setup order suggestions while maintaining flow flexibility
- [x] OT-P2-002 | User-Friendly Documentation | Context-aware help content with plain language descriptions replacing technical terms

### 🔵 V2 Rework – Configuration substrate alignment (planned)
The wizard's first iteration grouped operator choices around resources. The v2 rework inverts the model so operators select scenarios first (capabilities), and resources, secrets, host tools/safeguards, and integrations are derived from that selection. The configuration substrate this rework consumes is documented in [`/docs/configuration/`](../../docs/configuration/); see [`docs/WIZARD_FLOW.md`](docs/WIZARD_FLOW.md) for the flow and wireframes.

- [ ] OT-V2-001 | Scenarios-first wizard flow | Replace the resources-first flow with scenarios → resources → secrets → integrations → host → operating-mode → validation. System-required scenarios render as locked-on per `service.system_required`.
- [ ] OT-V2-002 | Operator state persisted to operator-state.json | Wizard writes choices to `.vrooli/operator-state.json` per [`operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json). Manifests remain the source of declarative truth; this file holds operator choices only.
- [ ] OT-V2-003 | Host tools/safeguards step with risk indicator | New step rendering `risk` field on safeguards and opt-in toggles writing to `host_tools` and `host_safeguards` in operator-state.
- [ ] OT-V2-004 | Per-scenario auto-restart toggle | "Keep running" toggle per scenario, defaulting from `runtime.auto_restart_default`, override stored in operator-state.
- [ ] OT-V2-005 | Re-enterable from any step | Wizard is idempotent and re-enterable; not a one-shot. Adding a scenario or resource later re-enters at the relevant step with prior state pre-loaded.
- [ ] OT-V2-006 | Final validation report | Terminal step runs full health-probe pass and shows green-light or actionable error list.
- [ ] OT-V2-FEATURE-COMPLETE | Wizard covers every documented integration | **Feature-complete when every integration documented in [`/docs/configuration/integrations/`](../../docs/configuration/integrations/) has a wizard step.** This is the explicit acceptance criterion: the configuration docs are the contract; the wizard is the implementation.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go (API/CLI), React/TypeScript (UI)
- Data + storage expectations: Local file system (.vrooli/), HTTP endpoints for health checks
- Integration strategy: API-first architecture with thin CLI/UI layers
- Non-goals: Custom resource implementations, complex workflow automation

## 🤝 Dependencies & Launch Plan
- Required resources: `vrooli resource status --json`, service.json, secrets.json
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
