# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Turn a `deployment-manager` profile + `scenario-dependency-analyzer` graph into a deployable “mini Vrooli” bundle, then deploy it to cloud targets (VPS first).
- **Primary users/verticals**: Vrooli operator deploying scenarios to production infrastructure.
- **Deployment surfaces**: `deployment-manager` orchestration plus equivalent `scenario-to-cloud` UI, CLI, and API surfaces. The UI is a required professional operator surface for P0; it cannot be treated as optional evidence.
- **Value promise**: Repeatable deployments with strong preflight checks, explicit manifests, and predictable health verification.

## 🎯 Operational Targets

Operational targets are tracked via `requirements/` modules and auto-updated by the test suite.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Cloud Manifest Export | Export a deployment manifest from deployment-manager that fully defines the cloud bundle + target config for one scenario.
- [ ] OT-P0-002 | Mini-Vrooli Bundle Build | Build an immutable bundle containing the required Vrooli core, declaration-derived scenario/resource/tool/safeguard/credential closure, target control-plane artifacts, and no unrelated packages.
- [ ] OT-P0-003 | VPS Preflight | Validate SSH/DNS/ports/OS/network and fail fast with actionable errors before copying artifacts.
- [ ] OT-P0-004 | VPS Install + Setup | Copy bundle to the VPS, run Vrooli setup, and write minimal config needed for this deployment mode.
- [ ] OT-P0-005 | Deploy + Start | Start required resources, allocate target-owned ports from the reviewed deployment contract, start the scenario, and verify health through the configured endpoint and observed target identity.
- [ ] OT-P0-006 | Inspect + Logs | Provide a standard way to fetch status + logs over SSH for the deployed scenario/resources.
- [ ] OT-P0-007 | Authenticated Management Authority | Every management API, CLI, WebSocket and relay entry point authenticates the actor and authorizes deployment, target, environment and effect under the shared policy before any target effect.
- [ ] OT-P0-008 | Typed Identities + One Executable Plan | Machine, deployment, release and operation identities are explicit and noninterchangeable; preview, policy and apply consume one typed plan digest.
- [ ] OT-P0-009 | Durable Fenced Operations | Deployment operations persist intent before effects, survive client and owner restarts, refuse stale fences and reconcile unknown remote results.
- [ ] OT-P0-010 | Declaration-Derived Closure + Shared Configuration | Dependency, tool, safeguard and credential closure derives from component declarations and the analyzer; machine configuration flows through setup/v1 selection and onboarding.
- [ ] OT-P0-011 | Shared Reach + Target-Local Effects | Enrolled machines are reached through Bridge/nodereach with typed target-local control-plane actions; cloud keeps no private SSH inventory or host-repair scripts.
- [ ] OT-P0-012 | Verified Immutable Release + Safe Activation | One release identity binds bundle, native CLI, closure and configuration; releases are verified, staged and activated with rollback eligibility and protected data.
- [ ] OT-P0-013 | Backup, Schema Transition + New-Host Restore | Declared data bindings have consistent encrypted recovery points; restore onto a fresh host is proven with measured RTO/RPO, separately from code rollback.
- [ ] OT-P0-014 | Credential Lifecycle | Credentials are distributed, rotated, revoked and recovered through the credential authority with versioned consumer acknowledgement and zero value leakage.
- [ ] OT-P0-015 | Truthful Health + Governance Binding | Health observations carry target, release, configuration and freshness; Deployment Manager binds approval and publication to the exact candidate and target receipts.
- [ ] OT-P0-016 | Professional Operator Surfaces | API, CLI and UI expose the same typed state, actions and recovery next-steps with keyboard, narrow-viewport and interruption-resume support.
- [ ] OT-P0-017 | Certified Support Matrix | Ubuntu 24.04 on Linux amd64 and arm64 is certified for the minimal fixture through QEMU and authorized real-VPS lanes; other combinations are classified compatibility-only or unsupported with reasons.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Multi-Environment Profiles | Support multiple environment configs (staging/prod) with separate domains/targets.
- [x] OT-P1-002 | Target Scaffolds | Add scaffolds for future targets (k8s, Railway, etc.) without requiring P0 deploy support.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Rollback | Superseded by OT-P0-012: predecessor retention and compatibility-qualified rollback are launch requirements.
- [ ] OT-P2-002 | Updates | Delta updates instead of full tarball upload.
- [ ] OT-P2-003 | Managed Services Swaps | Optional swaps to managed Postgres/Redis/etc.
- [ ] OT-P2-004 | Bastion/Zero-Trust Access | Support jump hosts and stricter SSH models.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go for packager logic + CLI; minimal UI for operator visibility (non-blocking).
- Data + storage expectations: Postgres-backed deployment, operation, release and evidence records behind domain-owned repositories; leased SQLite for routed test runs.
- Integration strategy: scenario-to-cloud owns deployment desired state and execution; deployment-manager governs release approval and publication records; scenario-dependency-analyzer supplies the dependency graph; vrooli-bridge/nodereach supplies machine identity and reach; vrooli-onboarding/setup supplies machine configuration; the credential authority supplies credential custody; delivery-ramp-go supplies evidence dispositions.
- Non-goals / guardrails: No managed service swaps, no Kubernetes/multi-region, no delta uploads, no billing or mobile packaging in P0. Rollback, backup and restore are launch requirements, not exclusions.

## 🤝 Dependencies & Launch Plan

- Required resources: postgres (deployment, operation and release records).
- Scenario dependencies (conceptual): `deployment-manager`, `scenario-dependency-analyzer`, `vrooli-bridge`, `vrooli-onboarding`, `secrets-manager`, `data-backup-manager`, `test-genie`, `vrooli-autoheal`.
- Operational risks: DNS/ports for Let’s Encrypt, declaration-derived bundle determinism, target-owned port allocation, and idempotency.
- Launch sequencing:
  1) Implement manifest export + validation in deployment-manager.
  2) Implement bundle builder + stripper in scenario-to-cloud.
  3) Implement VPS preflight + deploy + start + verify.
  4) Enforce shared authentication, typed identities, one executable plan and durable operations.
  5) Adopt shared reach, onboarding configuration, immutable releases, backup/restore and credential lifecycle.
  6) Bind health and evidence to Deployment Manager; certify the Ubuntu 24.04 amd64/arm64 minimal fixture on QEMU and an authorized real VPS.
  7) Validate `landing-page-business-suite` as an additional hosted consumer, never as the only fixture.

## 🎨 UX & Branding

- Look & feel: operator-first, dense, status-heavy, with clear plan/steps and copy-pasteable commands.
- Accessibility: keyboard order, focus restoration, status announcements, contrast, reduced motion and narrow viewports are launch requirements for deployment and recovery flows.
- Voice & messaging: explicit about risks, prerequisites, and what is/isn’t automated.
- Branding hooks: none in P0.
