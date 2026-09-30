# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Provide a deterministic, tiny generated-app fixture for Android and iOS delivery-ramp conformance.
- **Primary users/verticals**: Scenario-to-android, scenario-to-ios, BAS, and device-control maintainers.
- **Deployment surfaces**: React/Vite UI, lifecycle-managed scenario test, and BAS flow fixture.
- **Value promise**: Cross-platform validation has a stable app contract instead of scenario-specific demo logic.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Deterministic fixture | The app renders stable selectors for state, route, connectivity, notification, and response.
- [x] OT-P0-002 | BAS-compatible flow | The smoke flow can navigate, enter text, submit, and assert the result through the selector manifest.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Persistence | A submitted value survives reload through local storage.
- [x] OT-P1-002 | Explicit state transitions | Connectivity and notification states are visible without network or backend dependencies.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Native shell parity | Add equivalent iOS and Android native-shell assertions as ramps mature.
- [ ] OT-P2-002 | Fixture variants | Add focused permission, migration, and deep-link variants without changing the base contract.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: React, TypeScript, Vite, and the standard Vrooli lifecycle wrapper.
- Data + storage expectations: browser local storage only; no external resource or network dependency.
- Integration strategy: selector manifest and BAS flow are the shared contract; ramps own device execution.
- Non-goals / guardrails: this fixture does not own Android/iOS device control, signing, release distribution, or host repair.

## 🤝 Dependencies & Launch Plan
- Required resources: none.
- Scenario dependencies: browser-automation-studio for the reviewed BAS flow contract.
- Operational risks: keep selectors and state transitions deterministic; do not add backend calls to the base fixture.
- Launch sequencing: UI test → BAS smoke flow → Android/iOS ramp conformance chapters.

## 🎨 UX & Branding
- Look & feel: compact fixture surface with clear state labels and responsive layout.
- Accessibility: semantic labels and stable test IDs for every conformance target.
- Voice & messaging: explicit state, route, and result text; no timing-dependent copy.
- Branding hooks: none; this is a test fixture, not a product surface.

## 📎 Appendix
[Add references or research links here if needed.]
