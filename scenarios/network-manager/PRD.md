# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Network Manager provides local-first internet management for home and small-office networks: health snapshots, DNS filtering, device inventory, safe optimization experiments, and integration hooks that other Vrooli scenarios can invoke.
- **Primary users/verticals**: Technical homeowners, families, homelab operators, small-office operators, managed-service consultants, and Vrooli agents that need a reusable network-control capability.
- **Deployment surfaces**: React/Vite UI for operators, Go API for product logic and integrations, Go CLI for agents/operators, Home Automation actions/events, and future resource adapters for managed DNS and router platforms.
- **Value promise**: Turns one-off network tuning and ad-blocking setup into a reusable Vrooli capability that diagnoses network quality, manages DNS policy, applies safe improvements with rollback, and becomes a monetizable lifestyle and small-office infrastructure feature.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Network health snapshot | Measure WAN reachability, gateway reachability, DNS latency, resolver behavior, IPv4/IPv6 status, packet loss, jitter, throughput, and local host/network facts in a repeatable report.
- [ ] OT-P0-002 | AdGuard Home resolver management | Manage an AdGuard Home resource as the first supported DNS/filtering backend, including setup status, upstream resolver configuration, conservative filtering defaults, and health checks.
- [ ] OT-P0-003 | Conservative DNS filtering controls | Provide allowlist, denylist, blocklist, pause/resume, and policy preview controls that prioritize low false-positive risk and clear rollback.
- [ ] OT-P0-004 | Device inventory | Discover and track LAN-visible clients with IP, hostname, MAC or stable identifier when available, resolver usage, group assignment, and confidence notes for randomized identities.
- [ ] OT-P0-005 | Safe optimization experiments | Run baseline/candidate/after measurements, score candidates by reliability-first metrics, and require approval before persistent configuration changes.
- [ ] OT-P0-006 | Capability adapter model | Represent host, resolver, router, and manual adapters as capability-reported interfaces so the scenario is OS-agnostic at the control-plane level.
- [ ] OT-P0-007 | Home Automation integration contract | Publish actions and events that Home Automation can consume for health checks, filtering toggles, device status, and outage or degradation signals.
- [ ] OT-P0-008 | Privacy-preserving defaults | Default to minimal retention, explicit query-log visibility, household-aware permissions, and no silent exposure of per-device browsing metadata.
- [ ] OT-P0-009 | Operator UI and CLI workflows | Expose the same core workflows through UI and CLI: snapshot, resolver status, device inventory, filtering change preview, optimization run, apply, rollback, and report export.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Household policy profiles | Add trusted, kids, guest, IoT, and work profiles with per-group DNS policy, filtering strength, schedule, and override behavior.
- [ ] OT-P1-002 | Scheduled access controls | Support bedtime, focus, guest-window, and temporary pause/resume policies for devices and groups.
- [ ] OT-P1-003 | Router adapter pilot | Add one explicit router adapter, preferably OpenWrt, OPNsense/pfSense, or UniFi based on the operator's first real deployment environment.
- [ ] OT-P1-004 | IPv6 and encrypted-DNS enforcement guidance | Diagnose IPv6 DNS bypasses, DoT/DoQ/DoH risks, and router/firewall rules, generating instructions or adapter-backed changes where supported.
- [ ] OT-P1-005 | Pi-hole adapter | Support Pi-hole as a second managed resolver backend for users who prefer a focused DNS sinkhole appliance.
- [ ] OT-P1-006 | Small-office audit mode | Add business-oriented change history, policy approval records, outage evidence, and exportable reports suitable for small-office operations.
- [ ] OT-P1-007 | Continuous monitoring | Schedule recurring health snapshots, detect regressions, and alert when resolver, gateway, ISP, or filtering failures appear.
- [ ] OT-P1-008 | Browser/endpoint DoH guidance | Generate browser and OS policy guidance for managed endpoints without attempting invasive TLS inspection.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Technitium adapter | Support Technitium DNS Server for DNS-heavy homelab and business users who need recursive/authoritative DNS, local zones, and advanced policies.
- [ ] OT-P2-002 | Multi-site management | Manage multiple home, small-office, or client networks with site profiles, comparisons, and fleet-level reporting.
- [ ] OT-P2-003 | Roaming-device filtering | Integrate VPN-back-home or endpoint-local filtering patterns so managed devices keep policy off-network.
- [ ] OT-P2-004 | Predictive network recommendations | Use historical measurements to suggest router placement, ISP escalation evidence, hardware replacement, or policy changes.
- [ ] OT-P2-005 | Advanced traffic/topology analysis | Add deeper topology discovery, bandwidth attribution, packet-capture workflows, and security scanning only where operator consent and platform support are explicit.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: React + Vite UI, Go API, Go CLI, Connect-RPC/proto for owned contracts, REST only for justified operational exceptions, and `vrooli-default` design kit.
- Data + storage expectations: Start with embedded local storage through the template's portable storage pattern. Store snapshots, device identities, policy state, experiment runs, approvals, rollback handles, retention settings, and exported reports. Keep query-level DNS logs retention-limited and configurable.
- Integration strategy: Treat AdGuard Home as the first managed resolver adapter. Model OS/host, resolver, router, and manual integrations behind capability-reported adapters. Home Automation consumes Network Manager actions/events rather than owning network state. The retired `network-tools` scenario is not an implementation baseline; recover ideas from git history only if a future audit justifies them.
- Non-goals / guardrails: P0 does not perform unsupported router writes, TLS interception, raw package installation, ISP account management, bypass-focused surveillance, hidden household monitoring, or risky automatic network changes. Unsupported platforms must receive read-only diagnostics and manual instructions instead of fake automation.

## 🤝 Dependencies & Launch Plan

- Required resources: AdGuard Home should become the first managed resolver resource or adapter dependency. The scenario should otherwise remain local-first and avoid new resources until a domain needs them.
- Scenario dependencies: Home Automation is a consumer integration in P0. The legacy `network-tools` live scenario has been retired in favor of this greenfield replacement.
- Operational risks: Networking changes can lock users out, break DNS, expose sensitive query metadata, create family/privacy trust issues, or vary heavily by OS/router/ISP. The product must favor preview, approval, rollback, and explicit capability reporting.
- Launch sequencing: First ship read-only health snapshot and device inventory, then AdGuard Home management, then conservative filtering controls, then approved optimization experiments, then Home Automation actions/events. Router writes and household policy profiles follow after the core loop is measured and trusted.

## 🎨 UX & Branding

- Look & feel: Calm operational control center, not a marketing page. Dense but readable diagnostics, status timelines, device tables, policy controls, and before/after experiment comparisons.
- Accessibility: WCAG 2.1 AA, keyboard-first controls, visible focus, screen-reader labels for charts and status indicators, non-color-only status, and accessible confirmation flows for risky actions.
- Voice & messaging: Precise, plain-spoken, evidence-backed. Prefer "This change reduced median DNS latency by 18ms" over vague claims like "optimized."
- Branding hooks: `Network Manager` should read as a trustworthy local infrastructure utility within the Vrooli ecosystem. Visual identity should support home and small-office profiles without becoming playful or alarmist.

## 📎 Appendix

**Ecosystem fit**: Network Manager is a product scenario plus reusable infrastructure capability. It serves direct UI, programmatic API/CLI, Home Automation action/event consumers, and future resolver/router resource adapters. Its compound-value seams are network health snapshots, optimization run ledgers, device inventory, policy profiles, adapter capability reports, and Home Automation events. It fits the lifestyle bundle as a personal/home-network headliner and the Business Suite as a small-office local IT depth feature.

**Resolver decision**: AdGuard Home is the first resolver target because it best fits the productized control-plane use case: integrated encrypted-DNS support, client-specific policy, network-wide filtering, and an API-friendly management surface. Pi-hole remains a priority follow-up for focused DNS sinkhole users. Technitium is deferred until advanced DNS platform features are needed.
