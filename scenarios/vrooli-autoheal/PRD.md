# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario vrooli-autoheal`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Keep the operator-declared Vrooli supervision set available across reboots, crashes, and partial failures without contradictory heal actions.
- **Primary users/verticals**: Vrooli operators, system administrators, DevOps engineers, and automated agents.
- **Deployment surfaces**: CLI tick/loop/status commands, API health and history surfaces, a web dashboard, and OS watchdog integrations.
- **Value promise**: One operator declaration drives cross-platform resource and scenario supervision, while durable evidence makes outages and repairs auditable.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | CLI tick command | When an operator requests one cycle, vrooli-autoheal shall bootstrap dependencies and execute the due health checks once.
- [ ] OT-P0-002 | CLI loop command | While loop mode is active, vrooli-autoheal shall run due checks at the configured interval until graceful shutdown.
- [ ] OT-P0-003 | Platform detection | The supervisor shall detect Linux, Windows, macOS, WSL, and the host capabilities needed by platform-specific checks.
- [ ] OT-P0-004 | Health check registry | The supervisor shall register and execute typed health checks with intervals, platform filters, and shared action handling.
- [ ] OT-P0-005 | Core bootstrap | When starting from cold state, the supervisor shall bootstrap its durable store, core resources, and critical scenarios.
- [ ] OT-P0-006 | Resource health checks | The supervisor shall monitor and repair every resource in the computed supervision set without treating a serving resource as failed.
- [ ] OT-P0-007 | Scenario health checks | The supervisor shall monitor and repair every scenario in the computed supervision set.
- [ ] OT-P0-008 | OS watchdog installer | The control plane shall idempotently install and verify the platform watchdog that keeps vrooli-autoheal running.
- [ ] OT-P0-009 | Health result persistence | The supervisor shall durably store bounded health, action, outage, and incident records for operator queries.
- [ ] OT-P0-010 | CLI status command | When an operator requests status, vrooli-autoheal shall report the last-known health summary and degraded dependencies.
- [ ] OT-P0-011 | Single supervision authority | The supervisor shall derive resource and scenario targets from one operator-declared, database-free dependency closure with attribution.
- [ ] OT-P0-012 | Cross-check heal interlock | If one check requests a destructive action against a target another check started inside the configured window, the supervisor shall refuse and durably record the action.
- [ ] OT-P0-013 | Availability ledger | When a supervised member becomes unavailable and later recovers, the supervisor shall persist one outage interval and the matching repair actions.
- [ ] OT-P0-014 | Bounded retry and escalation | If a supervised member exceeds its configured consecutive-failure budget, the supervisor shall suspend retries and raise a durable incident until explicitly resumed.
- [ ] OT-P0-015 | Storage retention | The supervisor shall enforce retention and size bounds on its durable history and use one canonical database location.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Infrastructure checks | The supervisor should monitor network connectivity, DNS resolution, and time synchronization.
- [ ] OT-P1-002 | System resource checks | The supervisor should monitor disk, swap, zombie-process, and port-exhaustion conditions.
- [ ] OT-P1-003 | Remote access health | Where remote access is configured, the supervisor should monitor and repair the platform-specific service.
- [ ] OT-P1-004 | Docker daemon health | Where Docker is required, the supervisor should monitor and repair an unresponsive daemon.
- [ ] OT-P1-005 | Cloudflared tunnel health | Where a Cloudflare tunnel is configured, the supervisor should monitor its service and connectivity.
- [ ] OT-P1-006 | Health history window | The supervisor should expose retained health and availability history for dashboards and trend analysis.
- [ ] OT-P1-007 | Web UI dashboard | The web UI should present current health, recent events, repair actions, outages, and incidents accessibly.
- [ ] OT-P1-008 | Configurable check intervals | The supervisor should schedule each check using its configured interval and avoid running it before it is due.
- [ ] OT-P1-009 | Graceful shutdown | When receiving SIGINT or SIGTERM, the supervisor should stop scheduling work and shut down cleanly.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Certificate expiration monitoring | Where certificates are configured, the supervisor may warn before expiration.
- [ ] OT-P2-002 | Display manager health | Where a display manager is configured, the supervisor may monitor its health on supported platforms.
- [ ] OT-P2-003 | Webhook notifications | The supervisor may deliver incident notifications to configured webhook destinations.
- [ ] OT-P2-004 | Custom check plugins | The supervisor may load externally defined health checks through a governed plugin contract.
- [ ] OT-P2-005 | AI root-cause analysis | Where an approved model is available, the supervisor may analyze failure history and suggest causes.
- [ ] OT-P2-006 | Mobile status app | The product may provide mobile status and push-notification surfaces.

## 🧱 Tech Direction Snapshot

- **Preferred stacks/frameworks**: Go for the API and control-plane integrations; React and TypeScript for the dashboard.
- **Data expectations**: SQLite is the canonical bounded local history store; supervision-set computation remains database-free.
- **Integration strategy**: Read operator state and dependency manifests through the control plane; use governed Vrooli resource and scenario lifecycle commands; keep host remediation in the control plane.
- **Non-goals**: Do not replace metrics or paging products, supervise arbitrary non-Vrooli processes, or add a second target declaration.

## 🤝 Dependencies & Launch Plan

- Required capabilities: the Vrooli control plane, scenario-dependency-analyzer, operator state, and filesystem process ownership records.
- Optional resources: Redis for real-time delivery and Ollama for future analysis; neither is required to compute the supervision set.
- Risks: circular self-healing, platform-specific watchdog behavior, retry storms, stale process identity, and unbounded history.
- Launch sequence: establish ownership-authoritative classification; add the heal interlock; publish one supervision set; consume it in onboarding and autoheal; add durable outages, incidents, retention, and host-schedule hygiene.

## 🎨 UX & Branding

- Look and feel: calm, status-led, and operationally dense without hiding degraded states.
- Accessibility: meet WCAG AA, preserve high contrast, keyboard navigation, and screen-reader descriptions for health and incident state.
- Voice: precise and reassuring; distinguish healthy, serving-but-degraded, unavailable, refused, suspended, and escalated outcomes.
- Branding: use Vrooli shield and heartbeat motifs with semantic theme tokens rather than raw status colors.
