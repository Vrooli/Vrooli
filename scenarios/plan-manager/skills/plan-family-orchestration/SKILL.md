---
name: "plan-family-orchestration"
description: "Coordinate a Plan Manager family through typed membership, claims, graph review, and runnable-frontier operations without inventing orchestration state outside Plan Manager."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["plan-manager","plan-family","graph","frontier","coordination"]
  icon: "git-branch"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["plan-manager"]
    commands: ["plan-manager families"]
  origin:
    kind: "authored"
---
## Tools focus: Plan Family Orchestration

Use Plan Manager as the only authority for family membership, resource claims,
graph revisions, human review, and runnable frontiers. Coordinate child plans
from typed family state; do not infer launch order from prose or local notes.

Before the first operation, read `prompt-manager skill read plan-manager` and
use its recall and per-attempt learning capture. Do not duplicate the record.

Required reading:
- `path:scenarios/plan-manager/docs/concepts/PLAN-MODEL.md`
- `path:scenarios/plan-manager/docs/reference/cli-commands.md`
- `prompt-manager skill read implementation-plan-execution`

### 1. Scope

In scope: deciding when work is a plan family; declaring members and claims;
proposing and reviewing the graph; reading the frontier; recording child state;
and routing divergence back through Plan Manager.

Out of scope: executing child implementation work; changing a child plan's
content; bypassing graph review; keeping a second family ledger; or launching
work from an unreviewed, stale, or cyclic graph.

### 2. Family decision

| Observable situation | Decision |
|---|---|
| One plan can own the outcome and validation boundary | Keep one plan. |
| Two or more plans can execute independently and have explicit dependencies or resource overlap | Create one family and register every child. |
| The proposed children are only phases of one atomic change | Keep one plan; phases are not family members. |
| Ownership or interaction is unknown | Create no launch order. Record claims, propose the graph, and require review. |

### 3. Typed orchestration loop

1. Read the family with `plan-manager families get <family>`.
2. Add or update every child with `families member-put`. Preserve each child
   plan id and source revision.
3. Record exclusive and shared resource claims with `families claim-put`.
4. Propose a graph with `families graph-propose` after membership or claims
   change. Treat the returned graph revision as immutable review input.
5. Require an authorized agent or operator `families graph-review` decision for that exact revision.
6. Read `families frontier`. Launch only when `launchable` is true, and launch
   only the returned batch.
7. Persist each child transition with `member-put`, then read the frontier
   again. A partial child failure must not erase independent runnable children.

Every mutation uses the latest family revision. A revision conflict means the
family changed: read it again and reassess; do not replay the stale mutation.

### 4. Divergence routing

| Evidence | Route |
|---|---|
| A child plan is wrong but the family graph remains valid | Use `implementation-plan-execution`; log the child divergence. |
| A dependency or resource interaction changed | Update claims or edges, propose a new graph revision, and require a new review. |
| The graph is cyclic | Correct plan boundaries or edges; never choose an arbitrary break. |
| A child is blocked by missing authority | Record the child state and blocker. Do not mark sibling work blocked unless the frontier says so. |

### 5. Output expectations

The family aggregate must contain the shared outcome, typed members, claims,
one current graph revision, its matching review, and the derived frontier.
Operator handoff must name the family id, graph revision, review decision,
runnable batch, and any non-runnable reason.

No governed program is declared for this skill yet. The current loop contains
agent or operator graph judgment and revision-sensitive mutations; each deterministic step
already has one typed Plan Manager command. Promote a repeated stable subset to
a program only after execution evidence fixes its inputs, joins, and stop rule.

### Troubleshooting & Edge Cases

- `launchable=false`: read the frontier reasons; fix the named graph, review,
  or member-state condition.
- Revision conflict: re-read the family and recompute the intended mutation.
- Review targets an older graph revision: propose or read the current graph and
  obtain a new review.
- Plan Manager unavailable: stop family mutations. Local notes are not a safe
  substitute for authoritative state.

### Current admission and topology review

Read `families frontier` for current admission candidates. The graph's proposed
batches describe a schedule, not permission to launch all batches. Running
members retain their claims and count against maximum concurrency. A running
prerequisite is a wait, not a cycle. Transition an unchanged pending member to
running with its execution identity and expected family revision before spawn.
A concurrent admission can win that revision; re-read after conflict.
Execution progress preserves graph review. Membership, claim, source-scope or
policy changes require a new review. Do not use commit movement as a scope change.

### Durable family execution

Link each pending member to its Plan Manager execution before dispatch. Select
one immutable supervision policy. Start the owner composition with
`agent-manager workflow start --owner plan-manager --key plan-manager/plan-family-drain --input-file <input.json> --idempotency-key <stable-key>`.
The input contains `family_id`, `policy_version`, optional `parent_run_id`, and
optional `role_ref` (default `code.default`). Retain the returned workflow ID.
Use `workflow execution-wait` once or inspect `workflow trace` for a failure.
Reuse the same start key after a disconnect.

Agent Manager persists the compiled owner schedule, child attempts and cohort
watches. Each child admission retains its attempt key. A terminal child does
not satisfy a dependency until Plan Manager reports its execution complete.
A new topology requires a new reviewed execution. Resume an existing executor
instead of starting a second executor against running members.
