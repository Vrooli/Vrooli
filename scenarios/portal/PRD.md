# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Portal is the chat-first front door to the Vrooli ecosystem, giving operators one place to talk with LLMs, coding agents, and ecosystem capabilities.
- **Primary users/verticals**: Vrooli operators, builders, and agents who need a low-friction control surface for discovery, chat, and scenario readiness.
- **Deployment surfaces**: UI, API, CLI health/status, and BAS workflows.
- **Value promise**: Operators can start from conversation, see capability readiness honestly, and let Vrooli surface relevant scenarios, docs, skills, records, and commands without blocking the chat path.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Healthy Portal scaffold | Portal starts through the scenario lifecycle, survives restart, exposes API/UI/CLI health, and remains healthy when optional dependencies are unavailable.
- [ ] OT-P0-002 | Chat-ready contract foundation | Portal owns typed Connect contracts and local surfaces for chat, message tree, integration status, and search suggestion seams.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | OpenRouter chat path | Portal streams OpenRouter completions with model choice, web-search toggle, usage capture, and missing-key degradation.
- [ ] OT-P1-002 | Agent-mode bridge | Portal can hand agent conversations to agent-manager and surface session status without making agent-manager required for boot.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Passive ecosystem search | Portal offers omnibox suggestions and PASSIVE search attachments from search-hub without delaying LLM sends.
- [ ] OT-P2-002 | Readiness ladder controls | Portal measures optional integration health, applies OFF/PASSIVE/FULL-reserved mode policy, and exposes override/status UI.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: React/Vite UI, Go API/CLI, Connect/proto contracts, SQLite through api-core storage.
- Data + storage expectations: Scenario-local SQLite under the data directory; declarative schema for persisted chat state when the chat domain lands.
- Integration strategy: Optional scenario dependencies first; every dependency fails soft and reports readiness instead of blocking boot.
- Non-goals / guardrails: No agent-inbox/app-monitor migration, no FULL pre-LLM short-circuit in v0, no REST feature endpoints beyond operational health.

## 🤝 Dependencies & Launch Plan
- Required resources: None for v0 boot; local storage only.
- Scenario dependencies: search-hub, agent-manager, and prompt-manager are optional; OpenRouter is configured by `OPENROUTER_API_KEY`.
- Operational risks: OpenRouter key absence, unavailable optional scenarios, stale generated proto/UI artifacts, and template residue.
- Launch sequencing: Keep the scaffold green, land contracts, add storage/chat, wire OpenRouter, then add readiness/search behavior.

## 🎨 UX & Branding
- Look & feel: Quiet operational workspace with light/dark support, dense navigation, and restrained Vrooli Portal branding.
- Accessibility: Keyboard navigable shell, labeled navigation regions, typed selectors for workflows, and UI-health/a11y tests.
- Voice & messaging: Direct, operational, and honest about degraded integrations.
- Branding hooks: Managed Vrooli Portal display name, PWA icons, `/public/` assets, theme colors, and CLI description.

## 📎 Appendix
- Execution plan: `portal-v0-core-scaffold-chat-transplant-readiness-registry.md` in the operator plan store.
