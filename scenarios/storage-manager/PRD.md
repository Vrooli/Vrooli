# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario storage-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own Vrooli's durable cleanup policy and orchestration surface: scan reclaimable host and scenario storage, produce preview-first cleanup plans, enforce safety tiers and operator policy, and record immutable audit history for every apply attempt.
- **Primary users/verticals**: Vrooli operators, local self-hosters, SRE-style agents, system-monitor and vrooli-autoheal workflows that need a safe remediation path, and owner scenarios that expose cleanup hooks for private data.
- **Deployment surfaces**: Go CLI (`storage-manager` scan, plan, policy, apply, audit, provider), Connect-RPC API for scenario integrations, and a React operational UI for disk pressure, provider estimates, policy controls, plan review, apply status, and audit history.
- **Value promise**: Replaces ad hoc shell cleanup with a testable, auditable, policy-gated capability that can reclaim disk space without letting tests or automation mutate real host cleanup targets.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Provider safety contract | Define cleanup providers with Estimate, Preview, Apply, Verify, safety tier, owner metadata, supported platforms, privilege requirements, irreversible effects, and test substitutes.
- [x] OT-P0-002 | No-real-cleanup test seams | Route every host effect through injected filesystem, process, Docker, journal, clock, and scenario-provider seams, with fakes and drift tests proving tests cannot delete files or run cleanup commands against the real host.
- [x] OT-P0-003 | Policy-gated scan and plan | Produce deterministic scans and cleanup plans from provider estimates plus operator policy, including conservative defaults, provider versions, policy versions, approval requirements, and blocked Conditional or Forbidden actions.
- [x] OT-P0-004 | Replay-safe apply and audit | Apply only an approved exact plan with matching policy and provider versions plus idempotency key, return no-op on safe replay, and persist immutable redacted audit events.
- [x] OT-P0-005 | CLI and API control surface | Expose scan, plan, provider catalog, policy get/set/list, apply, and audit list/show through stable CLI and API contracts with mutating commands marked confirmation-required.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Conservative built-in providers | Implement preview-first providers for Trash, tmp, language/build caches, journald and apt metadata, Docker pruneable data, and Vrooli-owned retention hooks with dangerous cleanup disabled by default.
- [ ] OT-P1-002 | Operator cleanup console | Deliver a dense React UI for disk pressure overview, provider estimates, policy editing, plan review, approval state, apply results, and audit history with loading, empty, and error states.
- [ ] OT-P1-003 | Adjacent scenario handoff | Document or wire system-monitor and vrooli-autoheal remediation paths through storage-manager while owner scenarios retain private deletion logic through provider hooks.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Scheduled policy execution | Add opt-in scheduled cleanup runs for eligible Safe and SafeWithOwner providers with explicit dry-run and audit retention controls.
- [ ] OT-P2-002 | Fleet cleanup intelligence | Aggregate cleanup estimates, provider health, and reclaim outcomes across scenarios to guide capacity planning and future storage governance.
- [ ] OT-P2-003 | Extended provider marketplace | Support registered scenario-owned providers with declarative metadata, contract validation, and reusable action discovery for future cleanup capabilities.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API using the repository's Connect-RPC patterns, Go CLI on cli-core manifests, React + Vite + TypeScript UI using the vrooli-default design system, and SQLite for local durable policy and audit state.
- Data + storage expectations: SQLite stores policy profiles, provider catalog snapshots, scans, plans, apply attempts, approvals, audit events, and redacted errors. Provider observations and previews are deterministic records, not implicit shell output.
- Integration strategy: storage-manager is a meta-scenario/interface-enabler. Other scenarios call it through CLI or Connect, while scenario-private cleanup remains implemented by owner scenarios through provider hooks. system-monitor observes disk pressure; vrooli-autoheal requests remediation; storage-manager governs policy and audit.
- Non-goals / guardrails: no real cleanup during tests; no direct `rm`, Docker prune, journal vacuum, apt clean, or language cache cleanup outside typed seams; no default Docker volume pruning, live database deletion, model cache deletion, or scenario data deletion; no broad replacement of workspace-sandbox garbage collection internals.

## 🤝 Dependencies & Launch Plan

- Required resources: none beyond the local host toolchain and scenario-owned SQLite database. Docker and journald providers degrade to unavailable or privilege-required states when their clients are not present.
- Scenario dependencies: system-monitor for disk pressure signals, vrooli-autoheal for remediation requests, workspace-sandbox and owner scenarios for private cleanup hooks, test-genie for scenario validation, cli-health and ui-health for interface governance.
- Operational risks: cleanup actions are irreversible if policy gates fail; mitigated by preview-first contracts, conservative defaults, approval requirements, idempotency keys, immutable audit, and fake-only tests. Privilege-dependent providers may be unavailable; mitigated by explicit privilege status and skipped apply. Integration dependencies may be absent; mitigated by optional clients and documented handoff commands.
- Launch sequencing: scaffold scenario and business contract; implement safety seams and fake drift tests; add conservative providers; add policy, planning, apply, and audit persistence; build CLI/API/UI; document adjacent handoff; run scenario and health validation before enabling any real apply path.

## 🎨 UX & Branding

- Look & feel: restrained operational console with compact tables, clear safety-tier badges, provider status, policy gates, preview diffs, and audit timelines. The interface should feel like infrastructure control software, not a marketing page.
- Accessibility: WCAG AA contrast, keyboard-operable controls, stable `data-testid` selectors, responsive layouts, explicit loading/empty/error states, and controls whose labels remain visible at mobile and desktop widths.
- Voice & messaging: factual, safety-first, and remediation-forward. Every cleanup action must state what would be cleaned, why it is allowed or blocked, which owner/provider is responsible, and whether approval or privileges are missing.
- Branding hooks: use vrooli-default tokens and lucide-style operational icons; safety tier, approval state, and reclaim estimate are the primary visual anchors.

## 📎 Appendix

- Source plan: `~/.vrooli/plans/storage-manager-scenario-implementation.md`.
- Related docs: `scenarios/system-monitor/PRD.md`, `scenarios/vrooli-autoheal/docs/reference/checks/system-disk.md`, `scenarios/workspace-sandbox/docs/reference/configuration.md`, and `docs/TESTING.md`.
- Ecosystem fit: meta-scenario and interface-enabler serving programmatic CLI/Connect, direct UI, and agentic action discovery; local cleanup primitives are not paid or gated.
