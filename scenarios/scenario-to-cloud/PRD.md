# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Turn a `deployment-manager` profile + `scenario-dependency-analyzer` graph into a deployable “mini Vrooli” bundle, then deploy it to cloud targets (VPS first).
- **Primary users/verticals**: Vrooli operator deploying scenarios to production infrastructure.
- **Deployment surfaces**: `deployment-manager` orchestration, `scenario-to-cloud` CLI, `scenario-to-cloud` API (UI is optional and non-blocking for P0).
- **Value promise**: Repeatable deployments with strong preflight checks, explicit manifests, and predictable health verification.

## 🎯 Operational Targets

Operational targets are tracked via `requirements/` modules and auto-updated by the test suite.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Cloud Manifest Export | Export a deployment manifest from deployment-manager that fully defines the cloud bundle + target config for one scenario.
- [ ] OT-P0-002 | Mini-Vrooli Bundle Build | Build a tarball containing only required Vrooli core + required scenarios/resources (from analyzer) + all `packages/`, plus `vrooli-autoheal`.
- [ ] OT-P0-003 | VPS Preflight | Validate SSH/DNS/ports/OS/network and fail fast with actionable errors before copying artifacts.
- [ ] OT-P0-004 | VPS Install + Setup | Copy bundle to the VPS, run Vrooli setup, and write minimal config needed for this deployment mode.
- [ ] OT-P0-005 | Deploy + Start | Start required resources, start the scenario with fixed ports (UI 3000, API 3001, WS 3002), and verify health via HTTPS.
- [ ] OT-P0-006 | Inspect + Logs | Provide a standard way to fetch status + logs over SSH for the deployed scenario/resources.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Multi-Environment Profiles | Support multiple environment configs (staging/prod) with separate domains/targets.
- [ ] OT-P1-002 | Target Scaffolds | Add scaffolds for future targets (k8s, Railway, etc.) without requiring P0 deploy support.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Rollback | Keep previous release + one-command rollback.
- [ ] OT-P2-002 | Updates | Delta updates instead of full tarball upload.
- [ ] OT-P2-003 | Managed Services Swaps | Optional swaps to managed Postgres/Redis/etc.
- [ ] OT-P2-004 | Bastion/Zero-Trust Access | Support jump hosts and stricter SSH models.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go for packager logic + CLI; minimal UI for operator visibility (non-blocking).
- Data + storage expectations: File-based state in `.vrooli/` (P0). No database required for P0.
- Integration strategy: deployment-manager owns profiles and calls scenario-to-cloud; scenario-dependency-analyzer provides dependency graph; scenario-to-cloud performs bundle/build/deploy.
- Non-goals / guardrails: No rollback, no backups, no managed service swaps, no multi-host fleets in P0.

## 🤝 Dependencies & Launch Plan

- Required resources: none for scenario-to-cloud itself (P0).
- Scenario dependencies (conceptual): `deployment-manager`, `scenario-dependency-analyzer`, `vrooli-autoheal`.
- Operational risks: DNS/ports for Let’s Encrypt, “mini Vrooli” bundle determinism, fixed-port overrides, and idempotency.
- Launch sequencing:
  1) Implement manifest export + validation in deployment-manager.
  2) Implement bundle builder + stripper in scenario-to-cloud.
  3) Implement VPS preflight + deploy + start + verify.
  4) Validate against `landing-page-business-suite`.

## 🎨 UX & Branding

- Look & feel: operator-first, dense, status-heavy, with clear plan/steps and copy-pasteable commands.
- Accessibility: basic keyboard navigation and readable contrast; UI is secondary to CLI in P0.
- Voice & messaging: explicit about risks, prerequisites, and what is/isn’t automated.
- Branding hooks: none in P0.
