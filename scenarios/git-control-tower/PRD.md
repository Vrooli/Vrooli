# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario git-control-tower`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Git Control Tower is the permanent evidence-backed Git advisory and human control plane for repository changes.
- **Primary users / verticals**: Operators, reviewers, and advisory agents that need bounded repository evidence, drafts, reviews, provenance, and safe human controls.
- **Deployment surfaces**: Go API and CLI, React UI, governed programs, scenario skills, and host-neutral collaboration consumers.
- **Value promise**: Agents may inspect and prepare advisory artifacts; repository and external publication mutations remain verified human actions.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Health check endpoint | The system shall validate database connectivity, Git availability, repository access, and readiness with a structured JSON response.
- [ ] OT-P0-002 | Repository status API | The system shall return current branch, tracking status, staged, unstaged, untracked, conflict, and bounded scope information.
- [ ] OT-P0-003 | File diff endpoint | The system shall return a structured diff for an explicit file path relative to the selected immutable revision.
- [ ] OT-P0-004 | Stage and unstage operations | The system shall stage or unstage explicit paths or bounded scopes only through verified human authority.
- [ ] OT-P0-005 | Commit composition API | The system shall validate a human-authored conventional commit message and create a commit only after exact preconditions pass.
- [ ] OT-P0-006 | Push and pull status | The system shall report push or pull readiness and behind-remote safety conditions without granting agents repository authority.
- [ ] OT-P0-007 | SQLite audit logging | The system shall record mutating operation decisions and outcomes durably without recording credentials.
- [ ] OT-P0-008 | CLI command parity | The system shall expose bounded CLI commands corresponding to supported API read and human-control operations.
- [ ] OT-P0-009 | Durable validation evidence operations | When validation requests behavioral or source-comparison evidence, the system shall durably capture or compare the declared collection with parent receipt attribution and idempotency.
- [ ] OT-P0-010 | Safe advisory artifacts | The system shall produce immutable change subjects, bounded evidence bundles, grounded summaries, findings, and editable commit, PR, and release drafts without actuating repository or host writes.
- [ ] OT-P0-011 | Human-only repository authority | If an agent, anonymous caller, forged header, expired intent, wrong repository, or replayed intent requests a repository mutation, then the system shall refuse before invoking a writer.
- [ ] OT-P0-012 | Truthful durable advisory execution | The system shall durably identify advisory jobs, preserve exact evidence and validation receipts, report partial or unavailable inputs explicitly, and support idempotent recovery.
- [ ] OT-P0-013 | Safe validation admission | The system shall admit only tests and delegated workflows whose subprocess and fixture effects are proven repository-safe, with unsafe historical paths reported as degraded.
- [ ] OT-P0-014 | Operator workspace and controls | The system shall provide a diff-first workspace with explicit subject identity, evidence inspection, draft editing, human action previews, and exact precondition controls.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Branch operations | The system should list, create, and switch branches with uncommitted-change detection and verified human authority.
- [ ] OT-P1-002 | AI-assisted commit messages | The system should generate commit-message suggestions from bounded evidence with a deterministic fallback when inference is unavailable.
- [ ] OT-P1-003 | Conflict detection and reporting | The system should detect active merge conflicts and potential conflicts from local and remote state without performing an agent merge.
- [ ] OT-P1-004 | Change preview | The system should preview commit impact with bounded line, affected-scope, and deployment-risk evidence.
- [ ] OT-P1-005 | Trailer and work-reference resolution | The system should parse supported Vrooli trailers, preserve unknown keys, resolve current work references, and disclose unresolved links without rewriting history.
- [ ] OT-P1-006 | Pending and committed AI provenance | The system should connect agent runs, sandbox receipts, exact content, and committed history while preserving mixed, legacy, private, and unknown attribution states.
- [ ] OT-P1-007 | Shared provenance search | Where a provenance provider is available, the system should expose a registered, scoped, privacy-preserving search corpus with explicit experimental or unavailable standing.
- [ ] OT-P1-008 | Mention-driven advisory requests | The system should normalize mention events, deduplicate edited deliveries, persist request and reply state, refuse stale revisions, and expose durable delivery outcomes.
- [ ] OT-P1-009 | Host-neutral collaboration | The system should expose provider-neutral change, review, check, release, and publish capability semantics without owning provider authentication or SDK connectors.
- [ ] OT-P1-010 | Governed skills and programs | The system should publish validated usage, feature, improve, provenance, review, and draft skills plus governed programs with explicit contracts and failure semantics.
- [ ] OT-P1-011 | Measured improvement | The system should preserve missing observations, apply anti-gaming floors, and produce setpoint reads that distinguish unavailable evidence from measured improvement.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Web UI dashboard | The system may provide a visual diff viewer, interactive human staging, branch management, and service-health monitoring.
- [ ] OT-P2-002 | Worktree management | The system may list and manage worktrees through the bounded WorktreeService contract.
- [ ] OT-P2-003 | Stash operations | The system may save, apply, and drop stashes through verified human controls.
- [ ] OT-P2-004 | Connected PR and release workspace | Where a verified host connection exists, the system may provide connected pull-request, release-authoring, activity, and publication views with capability-specific unavailable states.
- [ ] OT-P2-005 | Expanded provider integrations | The system may add provider-owned adapters for external collaboration operations while preserving the host-neutral GCT contract.
- [ ] OT-P2-006 | Operational evidence hardening | The system may add measured concurrency, retention, recovery, and independent corpus receipts when the owning validation infrastructure provides them.

## 🧱 Tech Direction Snapshot

- **Preferred**: Go domain services and generated contracts for API/CLI boundaries, React for the operator workspace, SQLite for scenario-owned durable state, and typed governed programs for repeated workflows.
- Evidence uses immutable records, explicit subject identity, bounded scopes, and uncertainty-preserving status values.
- Integration Hub owns provider authentication and adapters; Git Control Tower owns neutral consumers and human authority.
- Non-goals: agent repository mutation, caller-asserted authority, hidden host repair, raw CLI parsing as an API, and automatic PR/release publication.

## 🤝 Dependencies & Launch Plan

- Required local dependencies include Test Genie for owned validation, Workspace Sandbox and Agent Manager for provenance, Search Hub for scoped provenance search, and optional Integration Hub providers.
- Launch order is safe admission and contract reconciliation, authority enforcement, immutable evidence and jobs, provenance and advisory workflows, governed skills/programs, then connected collaboration surfaces.
- Missing providers, historical baselines, or independent corpora must remain explicit unavailable/degraded evidence and must not become passing claims.

## 🎨 UX & Branding

- **Accessibility**: WCAG AA contrast, keyboard-complete controls, semantic labels, and explicit status text for unavailable, partial, stale, refused, and verified states.
- Look and feel: dark, calm, terminal-inspired, diff-first, evidence-forward, and operator-readable.
- Drafts preserve human edits during regeneration; source identity and revision remain visible across navigation.
- Voice is precise and uncertainty-preserving: never imply provider connection, attribution, validation, or authority that was not observed.

## 📎 Appendix

- Canonical maturity plan: `~/.vrooli/plans/git-control-tower-advisory-maturity-20260905.md`.
- Detailed acceptance dossier and security posture live under `docs/internal/`.
