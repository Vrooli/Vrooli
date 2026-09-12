---
name: "team-coordination-leader-led"
description: "Supplemental guidance for teams with an explicit coordinating lead"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["agent"]
  tags: ["coordination","team","leader-led"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill"]
  origin:
    kind: "authored"
---
# Coordination Guidance (Leader-Led)

This is supplemental guidance referenced from the generated Operating Policy. You are operating inside a leader-led team. The Operating Policy will tell you whether you are the lead or a reporting member, and whether execution is single-process or multi-process.

## Operating Model

- The lead sets priorities, resolves ambiguity, and synthesizes outcomes.
- Reporting members should execute within their assigned scope and surface blockers early.
- In single-process runtime mode, coordination happens through the active session and subagent workflow.
- In multi-process runtime mode, coordination can use durable inbox messaging and persisted team state.

## Storage Model

- Continue with handoff when the next run needs continuity.
- Observe with knowledge entries for evidence, findings, snapshots, and friction.
- Propose with decisions when a durable surface should change.
- Operate only the team working state named in your generated Storage Map.
- Use task boards or inboxes only when the Operating Policy enables them.

## Default Behavior

1. Read the Operating Policy and org context before acting.
2. If you are the lead, delegate intentionally and synthesize results.
3. If you are a reporting member, execute your assigned work and report material blockers or findings upward through the enabled coordination surface.
4. Persist only the information the next run or teammate will actually need.

---

## Gated delegation pipeline contract

This is the shared scaffolding for every `leader-*` pipeline skill (`leader-explore-plan-implement`, `leader-research-analyze-plan`, `leader-triage-investigate-resolve`). Each of those skills cites this section for the phase shape, gate/rework semantics, delegation template, and convergence checklists, and adds only its own phase-to-leaf mapping, entry table, and domain-specific gate/rework criteria. Do not restate this contract inside a pipeline skill — reference it.

### Phase shape

A leader pipeline is an ordered sequence of phases separated by gates:

```
┌─────────┐  GATE 1  ┌─────────┐  GATE 2  ┌─────────┐
│ Phase A │ ───────▶ │ Phase B │ ───────▶ │ Phase C │
└─────────┘          └─────────┘          └────┬────┘
     ▲  REWORK ◀──────────┘  REWORK ◀───────────┘
     └── (a later phase reveals an earlier phase was wrong)
```

Every phase has the same four parts:

1. **Entry criteria** — what must be true before the phase starts (usually the prior gate passed).
2. **Leader actions** — read the phase's leaf skill; decide to perform the work personally or delegate it; run or review the work.
3. **Delegation** — when delegating, send the delegation message (template below).
4. **Artifacts** — the named deliverable the phase produces; the next gate checks it.

Each pipeline maps its phases onto **leaf skills** (the methodology each phase invokes) and enters at any phase when its inputs already exist (partial entry). The pipeline's own skill provides that phase-to-leaf mapping and the entry work table.

### Gate semantics

A gate is a checklist that must pass before the next phase begins. If a gate fails, return to the phase that owns the missing artifact and redo it with a scope targeting the specific gap. Gate checklists verify that the phase's artifact exists, is evidence-backed, and is sufficient for another agent to continue from alone. Each pipeline defines its own per-gate criteria; the pass/fail-then-return rule is universal.

### Rework semantics

Rework is cheaper than proceeding on a wrong assumption. When a later phase surfaces a signal that an earlier phase was wrong, return to the earlier phase rather than pressing forward. Each pipeline provides a rework-trigger table mapping `signal → phase → action`. The universal rule: sunk cost is never a reason to keep a wrong artifact alive.

### Delegation message template

Every delegation carries enough context that the assignee needs no back-channel:

```
I need you to [task] using the [methodology] methodology.

Context: [findings / report / triage location or summary]
Scope: [specific deliverable; in-scope / out-of-scope]
Acceptance criteria: [objective, pass/fail]
Dependencies: [what must precede/follow this work]
Required reading: prompt-manager skill read [leaf-skill(s)]

Deliver: [expected artifact and format]. Report back when complete or blocked.
```

### Convergence checklists

**Delegation sufficiency** — before delegating any phase, verify: the assignee has all required context (findings, plan, triage report); acceptance criteria are objective; required reading is named; the deliverable format is specified; blocking dependencies are identified; check-in cadence is set.

**Pipeline completion** — before declaring a pipeline complete, verify: every phase's artifact exists (produced or pre-existing at partial entry); the final Definition of Done passes; any rework loops are recorded for future reference.

### Shared anti-patterns

| Anti-pattern | Why it fails | Better |
|---|---|---|
| Skipping an early phase | Later phases build on wrong assumptions; rework cascades | Invest in the early phase; it saves downstream rework |
| Delegating without context | Assignee re-discovers what the leader already knows | Include findings, references, and required reading every time |
| No phase gates | Bad assumptions propagate unchecked | Run each gate checklist before proceeding |
| Rework avoidance | Sunk-cost thinking keeps a wrong artifact alive | Return to the owning phase the moment a rework signal appears |
| Leader implements instead of delegates | Leader becomes the bottleneck; defeats the pipeline | Leaders coordinate and synthesize; members execute |
