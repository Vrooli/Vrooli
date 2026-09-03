# Product Requirements Document (PRD)

## 🎯 Overview

**Purpose:** Prompt Manager is Vrooli's permanent system for authoring, storing, discovering, and governing reusable skills, agents, teams, and typed actions. It keeps judgment in human-readable skills and team documents, deterministic execution in single-command action contracts, and generated Connect APIs as the shared transport for CLI, UI, scenarios, and Program Runtime.

The primary users are agents and operators composing repeatable work. Prompt Manager is deployed as a local-first scenario with API, CLI, and web UI surfaces. Its file-backed entities remain inspectable and portable, while optional indexed search and inference resources improve discovery and authoring without becoming correctness dependencies.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] prompt-manager-must-have-crud-operations-for-skills-via-api | Skill CRUD | Create, read, update, and delete skills through the supported API.
- [x] prompt-manager-must-have-agent-crud-operations-via-api | Agent CRUD | Manage agent appearance, SOUL, capabilities, and connectors.
- [x] prompt-manager-must-have-team-crud-with-roles-members-org-chart | Team CRUD | Manage teams, roles, members, and organization charts.
- [x] prompt-manager-must-have-pack-based-skill-organization | Skill packs | Organize skills into core, local, and draft packs.
- [x] prompt-manager-must-have-full-text-search-across-all-skills | Text search | Search skill names, descriptions, and content.
- [x] prompt-manager-must-have-cli-for-quick-skill-operations | Cross-platform CLI | Provide scriptable skill, agent, team, discovery, and action operations.
- [x] prompt-manager-must-have-web-ui-for-visual-skill-management | Web UI | Provide accessible visual management for Prompt Manager entities.
- [x] prompt-manager-must-have-file-based-storage | Inspectable storage | Persist entities in portable file-backed stores.
- [ ] trusted-experiment-receipt-signing | Trusted experiment evidence | Sign canonical audit and holdout receipts and reject untrusted evidence at production gates.

### 🟠 P1 – Should have post-launch

- [x] prompt-manager-should-have-semantic-search-using-qdrant-vector-database | Semantic search | Discover relevant skills through optional vector search.
- [x] prompt-manager-should-have-skill-analysis-and-pattern-extraction | Skill analysis | Extract useful patterns from skill content.
- [x] prompt-manager-should-have-skill-enhancement-suggestions | Skill enhancement | Suggest evidence-grounded improvements.
- [x] prompt-manager-should-have-usage-tracking-and-metrics | Usage evidence | Track skill use and effectiveness signals.
- [x] prompt-manager-should-have-tag-based-categorization | Tags | Categorize and filter skills with tags.
- [x] prompt-manager-should-have-team-membership-with-roles | Team membership | Manage member roles and status.
- [x] prompt-manager-should-have-pack-based-skill-organization | Pack precedence | Resolve enhanced pack organization deterministically.

### 🟢 P2 – Future / expansion

- [x] prompt-manager-nice-to-have-exportimport-functionality-complete-fixed-database-column-mismatch-tested-and-working | Import and export | Transfer skill data through supported formats.
- [ ] prompt-manager-nice-to-have-collaboration-features | Collaboration | Support multi-operator collaboration with explicit ownership.
- [ ] prompt-manager-nice-to-have-advanced-analytics-dashboard | Advanced analytics | Explain usage and effectiveness trends.
- [x] prompt-manager-nice-to-have-version-history-for-skills | Version history | Inspect and restore prior skill versions.
- [x] prompt-manager-nice-to-have-3d-world-visualization-for-agents | Agent world | Visualize agents and coordination state as a diorama where place is state: each agent stands at its desk when running, at the team table when a heartbeat is due, in the commons when idle; the HUD shows counts, the next heartbeat, a ticker and per-agent actions, and works without the canvas.

## 🧱 Tech Direction Snapshot

- **Preferred API:** Go domain services exposed through generated, proto-first Connect contracts. REST routes are retired slice by slice after consumer and parity checks; generated wire types remain outside domain and persistence models.
- **Preferred storage:** portable file-backed entity stores plus embedded SQLite where indexed state is appropriate. Path construction uses platform-aware APIs rather than shell-specific assumptions.
- **Preferred UI:** React and TypeScript with accessible keyboard, focus, and status behavior; 3D visualization is an enhancement, not the only management surface.
- **Preferred discovery:** deterministic text discovery first, with optional Qdrant and Ollama-backed semantic search. Discovery must preserve an honest no-match result.
- **Preferred execution:** typed Action contracts wrap exactly one Vrooli-controlled CLI command; skills retain judgment and must not conceal deterministic operational sequences in prose.
- **Non-goals:** arbitrary shell execution, permanent REST/Connect dual ownership, opaque binary-only storage, and treating optional AI resources as correctness requirements.

## 🤝 Dependencies & Launch Plan

Required dependencies are the local filesystem and scenario lifecycle. Embedded SQLite supports indexed metadata. Qdrant and Ollama are optional accelerators for semantic search, evaluation, and suggestions, with deterministic degradation when unavailable. Credential Authority supplies production receipt signing.

Launch through the Vrooli lifecycle so ports, process naming, dependencies, health checks, and generated contracts are governed. Sequence contract changes domain by domain: generate contracts, migrate adapters and consumers, prove parity, then retire the superseded route. Validate API, CLI, UI, workflow, business, security, dependency, and performance phases before promotion.

## 🎨 UX & Branding

The interface should feel fast, inspectable, and operator-focused: clear hierarchy, concise state labels, strong search affordances, and explicit evidence for destructive or governed operations. CLI output must remain stable and scriptable across supported platforms.

**Accessibility:** all primary management and approval flows must be keyboard operable, expose visible focus and semantic labels, avoid color-only status communication, and remain usable without the 3D canvas. Motion and dense visualization should respect reduced-motion and alternate-view preferences.

## 📎 Appendix

- Architecture boundaries and migration order: `docs/concepts/DOMAINS.md`
- Typed action contract: `docs/concepts/ACTIONS.md`
- Memory promotion model: `docs/concepts/MEMORY-PROMOTION.md`
- Durable architecture decisions: `docs/internal/DECISIONS.md`
- Current work and validation history: `PROGRESS.md`
- Public API contracts: `proto/vrooli/prompt_manager/v1/`
- Entity stores: `store/skills/`, `store/agents/`, `store/teams/`, and `store/actions/`
