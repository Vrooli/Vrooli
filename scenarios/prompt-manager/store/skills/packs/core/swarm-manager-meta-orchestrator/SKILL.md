# Meta-Orchestrator: Vision to Backlog Pipeline

## Purpose

Translate a high-level vision into a reviewed, dependency-aware backlog import plan for Swarm Manager, then create the items only after the user approves.

This skill supports long, iterative planning before creation. If the user wants to front-load context, avoid premature workshop auto-spawn, or work one cluster at a time, stay in planning mode until they explicitly ask to create items.

**Required reading:**
- `prompt-manager skill read swarm-manager-backlog-tools`

## Related Skills

- `swarm-manager-initiative-context` — how to load an initiative's members, upstream, and downstream in one call. Required reading for any sub-step that needs to understand what's already in an initiative or adjust it post-research.

## Post-Planning Reorganization

The meta-orchestrator's job is the **initial shape** of the backlog — clustering a vision into initiatives and items, declaring dependencies, and kicking off workshops. **Research-driven reorganization** (deleting obsolete items, reprioritizing siblings, updating initiative `depends_on`) happens *after* research items conclude, not here. The research-conclusion-authoring and swarm-manager-process-research skills own that flow.

If you are revisiting a backlog that already exists (rather than building a new one from a vision), read `swarm-manager-initiative-context` for the current state of each initiative before proposing changes.

## Canonical Contract

When shaping backlog items, use the real Swarm Manager contract:

- `kind`: `idea | research | fix | execute | chore`
- `name`: kebab-case item id
- `title`
- `description`
- `priority`: `1-10`
- `effort`: `XS | S | M | L | XL`
- `research_target`: only for research items
- `depends_on`: `["kind/name", ...]`
- `initiative`: per-item initiative name
- `acceptance_allow`
- `acceptance_deny`

Do not use `scope`. It is not part of the backlog contract.

### Initiative metadata

Initiative metadata is supplied separately in the batch-create request's top-level `initiatives` array. Each entry accepts:

- `name`: kebab-case initiative id (referenced by items' `initiative` field)
- `title`
- `description` (optional)
- `status`: `active | completed` (default `active`)
- `priority`: `1-10` (optional; `0` means unprioritized — same scale as items)
- `depends_on`: `["<initiative-name>", ...]` (optional)

`depends_on` on an initiative takes **bare initiative names**, not `kind/name` (that form is for item-level deps). Every entry must resolve to another initiative in the same batch or an already-existing initiative on disk. Batch apply is topologically ordered, so it is safe to declare a dependent initiative before its dependency in the `initiatives` array.

## Scope

**In scope**
- parse messy input into candidate work items
- discuss the work with the user before creation
- preserve visual/theme clusters when clarifying
- inspect the codebase when existing scenarios or seams are involved
- identify dependencies, initiative groupings, and likely item splits
- preview and create backlog items through Swarm Manager CLI
- update existing backlog items instead of duplicating them when appropriate

**Out of scope**
- managing workshop rounds after creation
- executing the work
- making code changes outside Swarm Manager itself

## Workflow

### Phase 1: Intake

1. Read the full input.
2. If images or whiteboards are involved, transcribe with confidence flags.
3. Check existing backlog for likely duplicates.
4. Break the vision into candidate work clusters before forcing item boundaries.
5. Present the candidate clusters/items back to the user.

Use the user's grouping when it is meaningful:
- whiteboard columns
- color-coded regions
- numbered lists that appear thematic rather than priority-based
- user-directed "let's start with this section first"

### Phase 2: Clarify In Planning Mode

This phase may last many turns.

The default pattern is:
1. work one cluster at a time
2. clarify intent, value, and dependencies
3. note what is still unknown
4. keep a running planned-item list without creating anything

When the user says not to create items yet, do not create them.

Ask clarifying questions in small batches:
- prefer 2-5 related questions
- preserve the current cluster/theme
- ask confirmation-style questions when possible

Good planning outputs:
- major sub-initiatives
- provisional backlog items
- **initiative priorities** (1-10) where sequencing matters
- **cross-initiative dependency chain** (which initiative unblocks which)
- item-level dependency chain within each initiative
- implementation-order hypothesis
- what should be left for workshop agents to discover later

When two initiatives have a clear sequencing relationship, prefer `depends_on` on the later initiative over flattening the order into item-level deps. Item-level `depends_on` expresses "this item needs that item done first"; initiative-level `depends_on` expresses "this whole work stream needs that whole work stream landed first." Conflating them buries the structural story.

### Phase 3: Inspect Existing Code Before Finalizing Items

If the work touches scenarios, APIs, CLIs, or skills that already exist, inspect the codebase before finalizing the backlog plan.

Minimum expectations:
- read the relevant scenario docs/PRDs if they exist
- inspect existing APIs, stores, handlers, and command flows
- identify whether a requested capability already partially exists
- decide whether the work is:
  - extension of an existing scenario
  - cross-scenario integration work
  - greenfield replacement

Do not finalize backlog items from whiteboard text alone when code already exists and that inspection can materially improve the plan.

### Phase 4: Shape Backlog Items

Split work into backlog items when that improves:
- ownership
- dependency clarity
- scenario locality
- execution order

Common split triggers:
- one whiteboard bullet spans multiple scenarios
- API and UI work should be separated
- research is needed before execution
- a greenfield replacement needs separate research/spec/runtime items

Useful item patterns:
- `research` for architecture audits, signal extraction, and contract definition
- `execute` for feature/runtime implementation
- `fix` for targeted bug/risk correction
- `chore` for archiving/removing legacy structures
- `idea` for greenfield specification or concept-definition work when implementation is intentionally deferred

### Phase 5: Preview Before Create

Before creating, present the planned items to the user in review form.

The review should include:
- item title
- kind
- initiative
- priority
- effort
- dependencies
- a short description

Then preview the import with the CLI:

```bash
cat > /tmp/meta-orch-items.json <<'EOF'
{
  "items": [
    {
      "kind": "research",
      "name": "desktop-release-control-plane-audit",
      "title": "Audit desktop release control plane",
      "description": "Trace the real desktop release flow across deployment-manager, scenario-to-desktop, LPBS, and prompt-manager deployment skills.",
      "priority": 1,
      "effort": "M",
      "initiative": "desktop-release-governance",
      "acceptance_allow": [
        "scenarios/deployment-manager/**",
        "scenarios/scenario-to-desktop/**",
        "scenarios/landing-page-business-suite/**",
        "scenarios/prompt-manager/**"
      ]
    }
  ],
  "initiatives": [
    {
      "name": "desktop-release-governance",
      "title": "Desktop Release Governance",
      "description": "Shared release-control, traceability, and LPBS-delivery work for desktop monetization.",
      "status": "active",
      "priority": 1
    },
    {
      "name": "desktop-release-telemetry",
      "title": "Desktop Release Telemetry",
      "description": "Ship post-release signal so governance decisions have evidence.",
      "status": "active",
      "priority": 2,
      "depends_on": ["desktop-release-governance"]
    }
  ]
}
EOF

swarm-manager backlog batch-create --file /tmp/meta-orch-items.json --preview
```

Preview is the default safety gate for large imports. Use it before creation unless the user explicitly wants immediate creation and the plan is already tightly reviewed.

### Phase 6: Create

Only create after user approval.

```bash
swarm-manager backlog batch-create --file /tmp/meta-orch-items.json
```

Workshop round 1 auto-initializes on item creation. Do not manually trigger it.

If the session produced significant architecture context, upload it so workshop agents inherit the planning context. You can upload to either a backlog item or the initiative itself:

**Upload to a backlog item** (for item-specific context):
```bash
swarm-manager backlog file-upload --kind <kind> --name <name> --path orchestration-summary.md --stdin <<'EOF'
# Meta-Orchestrator Summary

## Source
[what the planning session covered]

## Decisions Made
- ...

## Dependency Notes
- ...

## Unresolved Questions Deferred To Workshop
- ...
EOF
```

**Upload to the initiative** (for cross-item strategic context):
```bash
swarm-manager initiatives file-upload --name <initiative> --path orchestration-summary.md --stdin <<'EOF'
# Initiative Context

## Strategic Rationale
[why this initiative exists and what success looks like]

## Cross-Item Decisions
[decisions that affect multiple items in this initiative]

## Sequencing Notes
[implementation order rationale, dependency reasoning]
EOF
```

Prefer initiative-level uploads when the context spans multiple items. Prefer item-level uploads when context is specific to one item.

## Duplicate Handling

Check duplicates at two levels:

1. **Within the session**
   If the same concept appears in multiple clusters, prefer one item with multiple dependents.

2. **Against the existing backlog**
   If an item already exists, update it with the new context instead of creating a duplicate unless the user clearly wants a separate item.

## Greenfield Replacement Pattern

When an existing scenario should not be evolved in place:

1. create a research item to extract reusable signal from the legacy implementation
2. optionally create a chore item to archive/remove the legacy scenario
3. create new greenfield spec/runtime items built from first principles

Do not frame this as "adapt the old scenario" if the actual intent is replacement.

## Questioning Rules

- Preserve user grouping by default.
- Prefer cluster-by-cluster clarification over giant cross-cutting interrogations.
- Ask only what is needed to improve the plan materially.
- If the user has already given all they know, stop interrogating and move to codebase inspection plus planning.
- It is acceptable to leave some ambiguity for workshop agents as long as the backlog descriptions capture what is already known.

## Output Template Before Creation

Use a concise review structure like:

```text
Planned backlog import:

Initiative: desktop-release-governance (priority 1)
- research/desktop-release-control-plane-audit
- execute/deployment-manager-approval-gate-surfaces
- execute/deployment-manager-visual-validation-approval-flow

Initiative: desktop-release-telemetry (priority 2, depends_on: desktop-release-governance)
- research/telemetry-signal-audit
- execute/wire-release-telemetry-dashboard

Initiative: emulator-platform (priority 5)
- research/emulator-extraction-plan
- execute/build-vrooli-emulator-linux-first

Initiative order:
1. desktop-release-governance
2. desktop-release-telemetry (unblocks only after governance lands)
3. emulator-platform (independent)

Item order within desktop-release-governance:
- research/desktop-release-control-plane-audit
- execute/deployment-manager-approval-gate-surfaces
- execute/deployment-manager-visual-validation-approval-flow
```

Show per-initiative priority and depends_on when set. Keep the two layers separate: the "Initiative order" reflects `depends_on` on the initiatives themselves; "Item order within X" reflects item-level `depends_on` inside a single initiative.

## Anti-Patterns

- do not create items before the user approves
- do not use `scope`
- do not skip codebase inspection when the target scenarios already exist
- do not flatten everything into one initiative if the work naturally separates
- do not manually trigger workshop round 1
- do not ask one tiny question at a time for long planning sessions
- do not preserve a legacy scenario by inertia when the actual need is a greenfield replacement
- do not flatten initiative sequencing into item-level `depends_on` when cross-initiative ordering is what's actually being expressed — use initiative `depends_on` instead
- do not put `kind/name` values in an initiative's `depends_on`; that form is only valid on items

## Troubleshooting

| Problem | Response |
|---------|----------|
| User wants to discuss only, not create | Stay in planning mode and produce a planned-item list only |
| Vision is large and messy | Group into clusters first, then clarify one cluster at a time |
| Existing code contradicts the whiteboard assumption | Surface the mismatch explicitly and adjust the backlog plan |
| Many items span multiple scenarios | Split by seam and link with `depends_on` |
| The user wants context preserved for spawned workshop agents | Add `orchestration-summary.md` after creation |
