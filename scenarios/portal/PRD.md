# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Portal is the chat-first front door to the Vrooli ecosystem, giving operators one place to talk with LLMs, coding agents, and ecosystem capabilities.
- **Primary users/verticals**: Vrooli operators, builders, and agents who need a low-friction control surface for discovery, chat, and scenario readiness.
- **Deployment surfaces**: Browser UI, API, CLI, governed programs, and a native companion packaged through Scenario-to-Desktop.
- **Value promise**: Operators can start from conversation, see capability readiness honestly, and let Vrooli surface relevant scenarios, docs, skills, records, and commands without blocking the chat path.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Healthy Portal scaffold | Portal starts through the scenario lifecycle, survives restart, exposes API/UI/CLI health, and remains healthy when optional dependencies are unavailable.
- [x] OT-P0-002 | Chat-ready contract foundation | Portal owns typed Connect contracts and local surfaces for chat, message tree, integration status, and search suggestion seams.

- [ ] OT-P0-003 | Federated target and surface identity | When an operator selects a target, Portal shall resolve owner-scoped identity and attached topology, preserve partial inventory, and expose fresh capability evidence without credentials or transport authority in descriptors.
- [ ] OT-P0-004 | Optional providers and equivalent routes | When a provider is absent, denied, stale, or lost, Portal shall remain usable and resolve only routes that preserve target, account, authority, data policy, and outcome; uncertain effects shall be reconciled before retry.
- [ ] OT-P0-005 | Native companion workspace parity | When the companion changes between hidden, pill, palette, and expanded modes, Portal shall preserve conversation, branch, attachments, selected surface, and active run identities in the ordinary Portal workspace.
- [ ] OT-P0-006 | Explicit context capture and accessible interaction | When the operator requests context or voice input, Portal shall provide bounded capture, geometry-preserving annotation, explicit permission recovery, keyboard access, and at-most-once finalized voice submission.
- [ ] OT-P0-007 | Isolated interactive surfaces | When an owner surface is embedded, Portal shall enforce authenticated origin and session binding, bounded messages, navigation policy, and isolation from native privileges while retaining conversation state on failure.
- [ ] OT-P0-008 | Governed task composition and measurable reuse | When a task is executed or reused, Portal shall invoke owner-governed operations with explicit acceptance checks and report measured effort, comparable cohorts, failures, and missing telemetry without inventing evidence.
- [ ] OT-P0-009 | Assistant migration and evidence-backed delivery | When Assistant capture is replaced or Portal support is claimed, Portal shall reconcile retained records and links, route capture once to its current owner, and tie platform and commercial claims to attributable acceptance receipts.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | OpenRouter chat path | Portal streams OpenRouter completions with model choice, web-search toggle, usage capture, and missing-key degradation.
- [x] OT-P1-002 | Agent-mode bridge | Portal can hand agent conversations to agent-manager and surface session status without making agent-manager required for boot.

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Passive ecosystem search | Portal offers omnibox suggestions and PASSIVE search attachments from search-hub without delaying LLM sends.
- [x] OT-P2-002 | Readiness ladder controls | Portal measures optional integration health, applies OFF/PASSIVE/FULL-reserved mode policy, and exposes override/status UI.
- [x] OT-P2-003 | One context brief path | Portal gates trust-labelled current-turn context once and serves the same persisted verdict to LLM, agent, and canary-verified external-harness consumers.

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
