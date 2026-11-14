# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario lighting-control`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: provide an opt-in, local Vrooli capability to enroll host lighting controllers and keep lights in a desired safe state across service, backend, and device lifecycle changes. Primary users: the local machine operator and authorized local scenarios. Deployment surfaces: scenario UI, CLI, authenticated local API, and headless local reconciliation.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Backend and enrolled hardware control | The scenario shall report OpenRGB availability and expose stable enrolled controllers and zones with truthful per-zone capabilities.
- [ ] OT-P0-002 | Durable manual Off and static control | When an operator selects Off or static color and brightness, the scenario shall persist the desired policy and apply only operations supported by the enrolled zone.
- [ ] OT-P0-003 | Headless desired-state reconciliation | When the scenario, backend, or device returns, the scenario shall reconcile the persisted desired policy without frame-rate device scans.
- [ ] OT-P0-004 | Real-input lighting policies | When configured, the scenario shall derive load, work, health, and combined policies from real system-monitor and active-agent sources with freshness fallback, priority, hysteresis, cap, expiry, and static fallback.
- [ ] OT-P0-005 | Operable desktop and mobile experience | The UI shall provide accessible, truthful hardware enrollment, policy, preset, and recovery journeys on desktop and mobile.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Operator diagnostics | The scenario should explain backend readiness, input provenance and staleness, policy precedence, and controller limitations through its UI, CLI, and API.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Future provider extension | The scenario may add another lighting backend after a separate contract and qualification; the current release shall make no such provider claim.

## 🧱 Tech Direction Snapshot

Preferred: React/Vite UI with proto-first Go Connect API and Go CLI generated from the same API contracts. Store desired policy and enrollment locally per machine through the scenario's supported storage contract. Consume the optional, locally bound OpenRGB resource; do not install, upgrade, or own its process. Consume system-monitor metrics and Agent Manager active-run state through their real APIs. Keep device identity independent from discovery order and bus index; report supported actions per zone. Non-goals: firmware-level persistence, undocumented bus writes, remote unauthenticated SDK access, duplicate telemetry or active-agent inventories, and unsupported platform/hardware claims.

## 🤝 Dependencies & Launch Plan

OpenRGB is an optional local resource, enabled and operated by its resource owner. System-monitor and Agent Manager are optional scenario dependencies for automatic policies; manual Off/static control remains available when they are absent. Missing or stale inputs use explicit safe fallback behavior. Launch with no default install, process, or hardware probe. Start the resource only after explicit enablement and keep SDK access local. Validate backend-absent and no-device operation before hardware control. Risks include backend permissions, unstable vendor identity fields, source outages, effect differences, and physical output that cannot be proven by SDK acknowledgement alone.

## 🎨 UX & Branding

Use a calm light-desk concept: make backend readiness, enrolled hardware, and the effective policy legible without a gaming-dashboard aesthetic. Desktop uses a wide simultaneous view; mobile uses a focused card stack and drill-in with Off within thumb reach. Accessibility target: WCAG 2.2 AA, with keyboard and screen-reader support, strong contrast, reduced motion by default, and clear state labels. Use plain recovery guidance. Never imply a command succeeded physically unless observed hardware evidence supports that claim.
