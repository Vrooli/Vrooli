# Product Requirements Document (PRD)

## 🎯 Overview
- **Purpose**: Vrooli Onboarding is the permanent operator surface for deciding what a Vrooli install runs and under what permissions. It projects manifest declarations into one guided workflow, commits operator decisions through a single typed authority, applies them to the host, and reports honest readiness — on a workstation, a desktop bundle, or a remote VPS.
- **Primary users/verticals**: First-run operators installing Vrooli; returning operators changing a running install; agents and remote coordinators (vrooli-bridge, scenario-to-cloud) configuring a host they cannot see.
- **Deployment surfaces**: Web UI, interactive CLI, non-interactive CLI, REST API, bundled desktop app.
- **Value promise**: An operator reaches a configured, applied, verifiably ready install in under ten minutes on any deployment tier, and never has to hand-edit a JSON file to do it.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Scenario-first capability selection | When an operator selects a scenario, the wizard shall resolve its transitive scenario and resource dependencies from manifests and shall show the resulting stack before the operator continues.
- [ ] OT-P0-002 | Non-destructive operator-state writes | When a surface commits a choice, the system shall merge only the changed fields into operator state and shall preserve every field it does not own.
- [ ] OT-P0-003 | Single operator-state write authority | The system shall expose one typed operator-state service, and every writer shall write operator state only through it.
- [ ] OT-P0-004 | Manifest-driven operator surface | The wizard shall render every operator-controllable choice from manifest declarations and shall hold no hard-coded inventory of scenarios, resources, categories, tools, safeguards, or safeguard config fields.
- [ ] OT-P0-005 | Deployment-tier parity | While the wizard runs from a desktop bundle, every step shall resolve its data from the bundled catalog and no step shall return an error.
- [ ] OT-P0-006 | Apply and completion | When the operator confirms setup, the system shall install opted-in host tools, apply opted-in safeguards, enable selected resources, start selected scenarios, report a per-item result, and record a durable completion marker.
- [ ] OT-P0-007 | Credential provisioning without disclosure | If a credential value is submitted, then the system shall relay it to the credential authority and shall not write it to operator state, logs, URLs, browser storage, or any response body.
- [ ] OT-P0-008 | Surface parity across UI, CLI, and API | The wizard shall be completable through the UI, an interactive CLI, and a non-interactive CLI or API call, and the three shall produce identical operator state for identical choices.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Actionable readiness report | The readiness report should probe declared credentials, host tools, host safeguards, and resource reachability, and should name a remediation for every item that is not ready.
- [ ] OT-P1-002 | Re-enterable setup | When an operator re-opens the wizard on a configured install, it should load committed state, resume at the first unsatisfied step, and allow revision of any decision that is not deferred.
- [ ] OT-P1-003 | Descriptor-complete credential guidance | Each credential card should present the declared purpose and the obtain link carried by the credential descriptor.
- [ ] OT-P1-004 | Remote and headless onboarding | The non-interactive surface should let vrooli-bridge and scenario-to-cloud configure a remote host or VPS with no hand-edited file.
- [ ] OT-P1-005 | Deployment union export | The wizard should export the union of scenarios, resources, tools, and safeguards implied by a selection, for bundle, VPS, and bridge targets to consume.
- [ ] OT-P1-006 | Accessible, themed operator experience | The wizard should meet WCAG 2.1 AA, should be operable by keyboard alone, and should render correctly in light and dark themes.
- [ ] OT-P1-007 | Declared-surface honesty | The published endpoint, CLI, and requirement contracts should match the running code, and drift should fail a test rather than wait for review.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Per-scenario operating mode | The wizard may expose the manifest-recommended auto-restart choice per scenario and may persist operator overrides.
- [ ] OT-P2-002 | Deferred integrations contract | If a scenario declares an integration before integration-hub ships, then the integrations step shall present it as deferred and shall create no placeholder binding.
- [ ] OT-P2-003 | Goal intake and profiles | The wizard may pre-select a stack from a named profile once a second concrete profile exists.
- [ ] OT-P2-004 | Configuration discovery | The wizard may publish its configuration surface to search-hub so an operator may find a setting by intent rather than by path.

## 🧱 Tech Direction Snapshot
- Preferred stacks: Go for the API, CLI, and the shared control-plane state service; React + TypeScript + Vite for the UI, consuming the shared component library and design tokens.
- Data + storage expectations: No scenario-owned database. `.vrooli/operator-state.json` is the sole durable record and is owned by a control-plane service, not by this scenario. Credential values live only in the credential authority.
- Integration strategy: Manifest-derived read models and a field-scoped write API. Every other surface — CLI, bridge, cloud, desktop — is a client of the same service and evaluator.
- Non-goals: Authoring `service.json`; owning connector or OAuth lifecycles (integration-hub); implementing host remediation itself (the control plane owns detection and repair); replacing `secrets-manager` for credential lifecycle.

## 🤝 Dependencies & Launch Plan
- Required resources: none. The scenario reads manifests and delegates storage.
- Scenario dependencies: secrets-manager and web-console (system-required peers); browser-automation-studio for journey evidence; experience-manager for the UX contract; plan-manager and test-genie for validation.
- Control-plane dependencies: the operator-state service, the credential authority, and the host-requirement resolver.
- Operational risks: silent operator-state field loss; deployment-tier drift between repo and bundle catalogs; a wizard that records intent but never applies it; declared contracts that drift from the running router.
- Launch sequencing:
  1. State integrity and deployment-tier parity.
  2. Wizard contract completion, including apply and completion.
  3. Configuration-substrate consolidation onto one write authority.
  4. CLI and API surface parity.
  5. Remote and bundled onboarding through vrooli-bridge.
  6. Journey evidence, coverage floors, and contract-drift gates.

## 🎨 UX & Branding
- Look & feel: Calm, dense, instrument-like. Progressive disclosure over dialogs. Consequences of a choice are visible on the same screen as the choice.
- Accessibility: WCAG 2.1 AA, complete keyboard operation, visible focus, announced step changes, and reduced-motion support.
- Voice & messaging: Plain operator language. Every non-ready state names its cause and the next action. No apology copy and no unexplained failure.
- Branding hooks: The shared design-token set and the Vrooli component library; light and dark themes are both first-class.

## 📎 Appendix
- **Configuration substrate**: [`/docs/configuration/`](../../docs/configuration/) — the source-of-truth contract this scenario implements. New configurability is documented there before it becomes a wizard step.
- **Wizard flow + wireframes**: [`docs/WIZARD_FLOW.md`](docs/WIZARD_FLOW.md) — the step-by-step implementation contract.
- **UX contract**: [`experience/`](experience/) — page, component, and journey specs validated by experience-manager.
- **State schema**: [`/.vrooli/schemas/operator-state.schema.json`](../../.vrooli/schemas/operator-state.schema.json).
- Operational targets carry RFC 2119 obligation by tier: P0 is `shall`/`must`, P1 is `should`, P2 is `may`. Checkboxes are flipped by requirement sync from passing evidence, never by hand.
