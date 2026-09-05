# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario swarm-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Operator-facing command center for collaborating with coding agents at scale. Swarm Manager automates how an operator completes software work — create a backlog item, workshop one evolving implementation plan until the operator accepts it, execute the accepted plan through governed agent workflows, review the evidence, and turn verified outcomes into follow-up work. Above the item layer, Goals capture higher-level intent (with milestones and acceptance criteria) so the system tracks true progress, proposes new work, and keeps the operator thinking at the project level instead of re-prompting agents task by task.
- **Primary users / verticals**: Vrooli operators managing autonomous change work; session and workflow agents acting on their behalf; developers extending the scenario ecosystem.
- **Deployment surfaces**: CLI, API (Connect-RPC + REST), UI (React + Vite), and agent surfaces (agent sessions, agent-manager workflow contracts).
- **Value promise**: Working through Swarm Manager is at least as effective as prompting a coding agent directly, with the next recommended action always explicit at both the item and the goal level.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Backlog work intake | The system shall let operators and agents create backlog items (idea, research, fix, execute, chore) directly, from grounded capture proposals at `suggested`, and from accepted proposals; captures may also propose a goal or milestone through the same decision rail
- [ ] OT-P0-002 | One evolving plan per item | Each backlog item shall carry exactly one evolving implementation plan that is authored and revised in place through workshop rounds
- [ ] OT-P0-003 | Plan workshop loop | When a workshop round runs, the system shall present typed proposals and open decisions and shall change the plan only through operator-approved responses
- [ ] OT-P0-004 | Explicit plan acceptance gate | The system shall block execution queueing until the operator accepts the current plan revision, and shall clear acceptance whenever the plan changes
- [ ] OT-P0-005 | Strategy-selectable execution | When an accepted item is run, the system shall let the operator select an execution strategy from a declared strategy registry
- [ ] OT-P0-006 | Phased slice execution | While a phased execution runs, the system shall implement and validate one plan slice at a time with access to prior-slice handoffs, per-slice review, and bounded correction turns
- [ ] OT-P0-007 | Independent post-run review | When an execution completes, the system shall produce an independent review verdict over the deliverable, baseline diff, and test evidence as advisory input to the operator
- [ ] OT-P0-008 | Operator-owned terminal decisions | The system shall reach a terminal item status (completed, failed, needs follow-up) only through an explicit operator review decision
- [ ] OT-P0-009 | Typed follow-up proposals | When review or execution surfaces further work, the system shall record typed proposals (follow-up run, correction, new backlog item) for operator decision in a single inbox
- [ ] OT-P0-010 | Item next-action projection | The system shall compute one recommended next action per backlog item using the same preflight checks that gate execution
- [ ] OT-P0-011 | Goals with milestones | The system shall model goals as operator intent statements with owned milestones, each carrying acceptance criteria over member backlog items
- [ ] OT-P0-012 | Goal planning loop | When goal planning runs, the system shall clarify goal intent and emit typed proposals (milestones, acceptance criteria, item assignments, new backlog items) into the operator's decision inbox
- [ ] OT-P0-013 | Milestone review on completion | When all member items of a milestone reach terminal status, the system shall review the milestone's acceptance criteria against repository evidence and propose follow-up items for unmet criteria
- [ ] OT-P0-014 | Governed workflow execution | The system shall run all agentic work through declared agent-manager workflows whose prompts are canon-conformant contract skills with typed result schemas

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Goal next-action chaining | The system should compute one recommended next action per goal, chaining into the top-priority member item's next action when no goal-level decision is pending
- [ ] OT-P1-002 | Goal progress and velocity | The system should report per-goal progress, ETA bands, scope-creep history, and completion velocity against the goal's trajectory
- [ ] OT-P1-003 | Goal-scoped discovery | When goal discovery runs, the system should identify missing or at-risk work inside the goal's scope and propose backlog items to close the gaps
- [ ] OT-P1-004 | Swarm management sessions | The system should provide conversational agent sessions primed with an operator-loop skill that can drive the full create, workshop, accept, execute, review cycle
- [ ] OT-P1-005 | One-shot execution strategy | The system should offer a one-shot full-plan execution strategy alongside the phased drain in the strategy registry
- [ ] OT-P1-006 | Live execution progress | While an execution runs, the system should surface live slice progress, workflow state, and cancellation controls
- [ ] OT-P1-007 | Unified work activity feed | The system should present one per-item feed spanning runs, reviews, proposals, operator decisions, and status changes
- [ ] OT-P1-008 | Execution policy modes | The system should enforce manual, scheduled, and yolo execution policies with configurable delays, queue caps, and circuit breakers

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Portfolio analytics | The system may provide cross-goal portfolio analytics covering throughput, cost, and trend dashboards
- [ ] OT-P2-002 | Green-path auto-accept policies | The system may support operator-configured auto-accept policies for low-risk reviews with clean evidence
- [ ] OT-P2-003 | Mid-run steering | The system may provide an operator channel to steer a running execution beyond slice-approval signals

## 🧱 Tech Direction Snapshot

- **Preferred stacks**: Go API (Connect-RPC + REST), React + Vite UI, Go CLI as a thin API wrapper.
- **Agentic substrate**: all agent runs flow through agent-manager declared workflows; workflow prompts are prompt-manager contract skills; the swarm-transitions registry is the single catalog of transitions.
- **Data**: file-backed domain stores under the runtime data root; events in the shared event log; plans owned by plan-manager and referenced by plan_ref.
- **Integration strategy**: shared workflows over direct APIs; proposals are the only mutation path for agent-suggested changes; operator decisions are the only path to terminal statuses.
- **Non-goals**: no parallel plan variants per item, no agent-side direct graph mutation, no bespoke per-scenario execution runtimes, no reintroduction of operating modes or initiative entities.

## 🤝 Dependencies & Launch Plan

- **Required scenarios**: agent-manager (workflow runtime), plan-manager (plan authority), prompt-manager (skills), test-genie (validation evidence), git-control-tower (baseline diffs), search-hub (recall).
- **Required resources**: postgres/qdrant/redis per service.json; local agent runners via agent-manager.
- **Risks**: agent-manager runtime health gates all execution features; contract-skill edits require workflow reconciliation to take effect; goal-layer apply paths must stay wired end-to-end or proposals silently strand.
- **Sequencing**: item loop first (intake, workshop, acceptance, execution, review), then goal layer (planning, milestone review, discovery), then portfolio analytics.

## 🎨 UX & Branding

- **Experience promise**: the next recommended action is always visible and truthful — the UI never recommends an action the server would refuse. Disabled actions always show the reason; categorically inapplicable actions are hidden.
- **Editing model**: focused Drawer/BottomSheet overlays for all editing; no inline forms.
- **Look/feel**: design-token layer (design-tokens.css) with light/dark parity; calm information-dense surfaces; Plan board is the primary surface, Graph secondary.
- **Accessibility**: WCAG 2.1 AA color contrast, full keyboard reachability for decision flows, motion kept subtle.
- **Voice**: plain, direct, evidence-first — surfaces state what happened and what to do next, never marketing tone.

## 📎 Appendix

- Operator journeys narrative: docs/concepts/OPERATOR-JOURNEYS.md
- Operating model authority: docs/concepts/TARGET-OPERATING-MODEL.md
- Transition catalog: docs/reference/transition-catalog.md
- Supersedes the 2026-01-27 PRD (idea-backlog/five-tab era); regenerated 2026-07-23 after the operating-mode retirement and the goals+milestones merge.
