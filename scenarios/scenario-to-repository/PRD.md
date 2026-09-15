# Product Requirements Document (PRD)

> **Template Version**: 2.0  
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`

## 🎯 Overview

- **Purpose**: Turn an admitted Vrooli scenario snapshot into a reviewable, deterministic, buildable source distribution with explicit closure, privacy, verification, and human-publication standing.
- **Primary users/verticals**: Scenario authors sharing a capability, release reviewers approving an exact artifact, and maintainers inspecting or rebuilding a focused source package.
- **Deployment surfaces**: Connect/API and CLI for automation, a responsive operator UI for source contents and review handoff, and governed programs for repeatable preparation and verification.
- **Value promise**: A recipient can see exactly what source is included, why it is included, what remains external, and whether the exact archive was independently verified without implying bidirectional synchronization or automatic Git publication.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Deterministic source assembly | Given equal admitted inputs and recipe, two exports have identical ordered manifests and archive digests.
- [ ] OT-P0-002 | Fail-closed source safety | Private/history content, escaping links, unresolved closure obligations, secret-like content, unsafe archive paths, and automated publication attempts are refused with redacted reasons.
- [ ] OT-P0-003 | Exact verification standing | Verification binds to the artifact digest and reports passed, failed, or unavailable without substituting a different run.
- [ ] OT-P0-004 | Honest source contract | The export identifies runtime requirements, canonical provenance, supported build assumptions, exclusions, and the monorepo as canonical.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Durable distribution lifecycle | Restart and duplicate requests preserve one distribution identity and retain artifact, verification, destination, and publication standing.
- [ ] OT-P1-002 | Exact governance handoff | Deployment readiness binds policy, closure, recipe, artifact, verification, and destination tuple; changed inputs invalidate readiness.
- [ ] OT-P1-003 | Reviewable update semantics | Source, recipe, and destination changes are shown separately; independent destination edits block overwrite preparation.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Measured improvement loop | Closure accuracy, reproducibility, private-content escape, clean-build, documentation, and failure-recurrence sensors retain comparable evidence.
- [ ] OT-P2-002 | Connector activation | A separately owned host connector can consume a human activation packet and return destination read-back evidence without moving publication authority here.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go domain core and Connect-RPC contracts, SQLite-backed domain repositories, React/Vite operator UI, and Vrooli lifecycle/test-genie ownership.
- Data + storage expectations: Domain-owned repositories hide SQLite; immutable export artifacts and receipts live under the scenario data namespace; no Git history is copied or mutated.
- Integration strategy: Scenario Dependency Analyzer owns source closure and dependency proposals; delivery-ramp-go owns provider-neutral evidence identity; Deployment Manager owns exact-candidate readiness; Integration Hub owns live host adapters.
- Non-goals / guardrails: No bidirectional sync, history-preserving extraction, force-push, autonomous commit/push/tag, host SDK, credential lifecycle, or claim of live publication from a fake connector.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite only for durable local records; no database, credential, or host resource is silently bundled into source output.
- Scenario dependencies: `scenario-dependency-analyzer`, `deployment-manager`, `git-control-tower`, `test-genie`, and `program-runtime` through their supported contracts.
- Operational risks: Private source leakage, incomplete local-module closure, archive nondeterminism, stale approval, destination drift, and unavailable host connectors.
- Launch sequencing: contract and closure → policy and deterministic archive → clean verification → durable orchestration → exact governance and human handoff → UI, learning, and maturation evidence.

## 🎨 UX & Branding

- Look & feel: Calm, dense operational console with source tree, closure reasons, policy findings, exact digests, gate status, and a separate publication handoff card.
- Accessibility: Keyboard-complete workflows, semantic status text in addition to color, responsive tree/detail navigation, 44px touch targets, and explicit loading, empty, error, stale, unavailable, and retry states.
- Voice & messaging: Precise and honest. Say “prepared,” “verified,” “awaiting human publication,” and “unavailable” distinctly; never say “published” after a button click without destination read-back.
- Branding hooks: Preserve `vrooli-default` tokens and shell; use cyan/blue for technical identity, amber for review attention, green only for evidenced success, and red for refused or failed obligations.

## 📎 Appendix

- Product label: **Source repository**.
- Canonical ownership: the Vrooli monorepo remains authoritative for development and contribution intake.
- Delivery format is orthogonal to runtime tier, operating system, host provider, and publication destination.
