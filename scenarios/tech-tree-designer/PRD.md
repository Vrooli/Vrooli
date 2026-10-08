# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario tech-tree-designer`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Provide Vrooli's ecosystem design environment: understand observed software, curate the intended capability horizon, and author bounded proposal workspaces spanning the repository.
- **Primary users / verticals**: Engineering agents, operators reviewing plans, scenario/resource authors, and ecosystem designers.
- **Deployment surfaces**: Local operator UI, typed API and CLI, and governed agent consumers. Remote and native-host claims require separate qualification.
- **Value promise**: Preserve proposed target artifacts alongside concise plans, reduce author-to-executor information loss, and connect bottom-up development with a revisable top-down capability model.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Live interface graph | When current interfaces are requested, TTD MUST expose SDA-derived scenario nodes and proto/Go import relationships with source provenance.
- [ ] OT-P0-002 | Contract-first planning | When planned proto contracts are edited, TTD MUST validate them against applicable schemas and refuse materialization when validation fails.
- [ ] OT-P0-003 | Distinct ecosystem views | When ecosystem information is displayed, TTD MUST distinguish observed facts, intended capabilities, and proposal-local changes with provenance and uncertainty.
- [ ] OT-P0-004 | Repository-wide proposal workspaces | When a proposal is created, TTD MUST retain scoped changes across existing or new scenarios, resources, shared packages, and project artifacts without publishing them.
- [ ] OT-P0-005 | Revision-bound design review | When a proposal is reviewed, TTD MUST expose the exact artifact and authored-graph revision, material changes, validation limitations, and plan references.
- [ ] OT-P0-006 | Recoverable owner-directed application | When authorized changes are applied, TTD MUST recheck relevant bases and owner grants, preserve unrelated edits, and retain idempotent per-entry recovery receipts.
- [ ] OT-P0-007 | Draft authority and isolation | If a draft requests ungranted effects or unsafe content paths, TTD MUST refuse the operation and identify the unmet boundary without publishing draft skills or programs.
- [ ] OT-P0-008 | Bounded large-ecosystem operation | When browsing or drafting at scale, TTD MUST bound query, response, compute and storage work and report truncation, freshness and unsupported limits explicitly.
- [ ] OT-P0-009 | Evidence-bound development target | When an authorized agent develops TTD, the scenario MUST expose its target and measurement gaps and require applicable evidence before claiming required outcomes satisfied.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Capability ontology | When long-term capabilities are curated, TTD SHOULD expose hierarchy, fulfillment mappings, coverage and focus views beside the observed ecosystem.
- [ ] OT-P1-002 | Production planning UI | When users inspect or edit designs, TTD SHOULD provide accessible graph and artifact views with loading, empty, partial, stale, error and recovery states.
- [ ] OT-P1-003 | Bottom-up learning and fulfillment | When implementation evidence changes, TTD SHOULD distinguish contribution from verified fulfillment and support reviewed revisions to the intended capability model.
- [ ] OT-P1-004 | Proposal comparison and lifecycle | When proposals evolve, TTD SHOULD support alternative revisions, amendment, abandonment and retention without losing referenced reviewed content.
- [ ] OT-P1-005 | Qualified draft experiments | Where explicitly authorized draft execution is available, TTD SHOULD link isolated experiment results to exact proposal inputs and expose runtime and effect limitations.
- [ ] OT-P1-006 | Cross-surface agent and operator parity | When a design operation is available, TTD SHOULD expose equivalent typed API and CLI behavior and discoverable operator navigation without UI-only business rules.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Rich graph source | TTD MAY enrich observed graph views through SDA and other qualified owner interfaces without duplicating their scanners or declaring unsupported coverage.
- [ ] OT-P2-002 | Strategic analysis | After deterministic design workflows are qualified, TTD MAY suggest capability decompositions and opportunities with provenance, uncertainty and explicit human disposition.
- [ ] OT-P2-003 | Civilization-scale design horizon | TTD MAY evolve a sector-based long-term capability horizon toward civilization-scale simulation and coordination without claiming exhaustive coverage or granting execution authority.

## 🧱 Tech Direction Snapshot

- **Preferred approach**: Keep Go/Connect API and CLI, React UI and domain-owned storage; extend existing graph, planning and ontology boundaries rather than create another plan manager.
- **Data**: Retain immutable reviewed revision manifests and content identities independently of ephemeral compute; use bounded derived caches and proposal deltas, not full-fleet copies.
- **Integrations**: SDA owns observed interface discovery; TTD owns design bundles and authored capability relationships; Plan Manager owns plans; Swarm owns authorization; qualified Workspace Sandbox and artifact owners provide diff/apply operations; the control plane owns runtime setup.
- **Non-goals**: No private repository scanner, Git engine, sandbox engine, execution scheduler or general-purpose IDE. Approval never rewrites observed facts or proves implementation. Optional strategic AI is not a prerequisite.

## 🤝 Dependencies & Launch Plan

- **Resources**: Embedded SQLite for current product metadata. Additional infrastructure must earn an explicit owner contract; no blanket resource expansion.
- **Scenario dependencies**: SDA supplies the implemented observed-interface source. General proposals require qualified owner operations for storage, workspaces, validation, plan references and Swarm authority; these integrations remain development targets.
- **Degradation**: Source unavailability retains provenance and any applicable cached/planned context; it never becomes a healthy empty ecosystem. Unsupported draft operations expose their missing capability.
- **Risks**: Relevant-base conflicts, partial multi-owner application, draft publication leaks, false capability fulfillment and unbounded whole-repository work.
- **Sequencing**: Establish the three-view model and scoped draft identity; qualify revision/review/application and scale; prove documentation-first work across multiple target kinds; then mature alternatives, experiments and strategic analysis.
- **Readiness**: New targets are unverified. Numeric performance cohorts and execution allowances require the documented qualification decisions before the full development mandate can be accepted.

## 🎨 UX & Branding

- **Look and feel**: A design workspace first, with graph-backed context and optional intended-capability navigation; precise labels, readable artifacts and progressive disclosure.
- **Accessibility**: Keyboard-operable navigation and editing; textual alternatives to graph geometry; status never communicated only by color; compact/mobile review retains warnings and decisions.
- **Voice**: Distinguish observed, proposed, approved, applied, and verified. Explain partial coverage and unavailable evidence in operator language.
- **Review**: Surface target, permission, interface, deletion and scope changes without opening every file. Link to Swarm for authorization rather than duplicating its decision system.
