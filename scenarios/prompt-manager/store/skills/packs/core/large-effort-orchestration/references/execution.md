# Delivery execution reference

Read the section for the operation in progress. Agent Manager's
`path:scenarios/agent-manager/docs/reference/effort-supervision.md` owns CLI/RPC
semantics. The main skill owns product planning and acceptance judgment.

## Brief and dispatch

Store one working brief per admitted epoch at `epochs/E<n>.md`. Proposed work stays
unadmitted until source, owner, scope, native capacity and real conflicts are checked.
The orchestrator writes the brief/directives; workers append their progress.

```markdown
# E<n> — <product outcome>

- Mission: <whole product destination; why this outcome matters>
- Outcome: <one journey or replacement and the resulting user benefit>
- Kind: feature | refactor
- Started: <owner-observed ISO time at admission>
- Estimate: <meaningful work units>
- Expected size: <net runtime lines for feature work>
- Metric: <observable outcome and command/evidence that reads it>
- Targets: <required, occasional or unavailable platforms/builds>
- Workers: <actual returned run IDs>
- Status: admitted | in-progress | parked | accepted

## References
Exact requirements, selected original images and current evidence. Label historical observations.
## Modules and budgets
Owned files/modules, shared interfaces, peer ownership and necessary UI/API/refactor work.
## Deletion list
Superseded responsibilities and intended end states, or why no deletion is justified.
## Exit gate
Commands and affected journeys; served-original comparison and preserved-behavior checks.
## Non-goals
Actual exclusions; preserve unmet whole-product work in STATUS.
## Directives
## Slice log
```

Before dispatch, reconcile the current source/failure so already-fixed work is not reassigned.
Capture the dirty-scope anchor through `git-control-tower proposal anchor --effort <ref>
--epoch E<n> --scope '<scenario path>/**' --json`. Obtain the caller run from
`agent-manager run identity --json` (`claims.run_id`). Use existing authorized profiles.
Send the complete assignment in the task description:

```bash
agent-manager task create --title "<goal> E<n>" --scope-path <scenario path> --description "$(cat -- /absolute/worker-brief.md)"
agent-manager run create --task-id <returned-task> --profile-id <existing-profile> --parent-run-id <caller-run> --until "<short assigned completion predicate>" --model <canonical-profile-model-id> --effort medium --workload-key <goal>/E<n>
```

Keep the file substitution quoted. Set an existing task's description with `task update
<id> --description "$(cat -- /absolute/worker-brief.md)"`. Do not add `--prompt`: it replaces
the task description. Keep `--until` separate and below 2,048 characters. The worker receives
the exact repository-relative brief path, parent ID, worker card, applicable directives,
source versions, gates and resume set. Record returned IDs in the brief and STATUS.
Use a canonical profile model ID (for example `gpt-6-luna`), not a shorthand. Correct only a
deterministic spelling refusal within that profile's permissions. Reconcile an uncertain
dispatch before retrying. Never strip identity or change principal to pass admission.

## Worker progress

The [worker card](worker.md#progress-and-step-back) owns progress-line grammar, growth
budgets, step-back triggers and the worker's exact check command. Use that card when
interpreting the worker's result; do not create a second protocol here.

## Acceptance and progress

Independently inspect each required gate, affected journey, selected source version,
deletion end state/callers, growth budget and test-protection result. Inspect visual gates
against the exact served build and selected originals. Required targets pass or remain open
with an authorized equivalent/return condition. Run `epoch-check` for all admitted worker
IDs; unresolved directives prevent acceptance. Reuse current proof under `docs/TESTING.md`.
Retain partial outcomes and unknown attribution; never accept a full outcome on a partial slice.

Create the existing review proposal with `git-control-tower proposal create --effort <ref>
--epoch E<n> --subject '<scenario>: <outcome>' --body-file <body> --path <file> ...
--run <worker> --run <orchestrator> --json`. Describe outcome, removal/retention decisions,
checks, gaps and source identities. Operator Git approval is distinct from product acceptance.
Append the accepted outcome to LOG with exact worker UUIDs and the returned proposal ID:

```text
- E<n> | ACCEPTED <ISO time> | workers=<UUIDs> | yield=<delta> <destination unit> | gates=G1,G2 | <outcome and gctp-ID>
```

Record reopening under LOG `## Reopened` with `- E<n> | <later ISO time>`. Set each active
unit in STATUS `## Active` as `- unit: E<n> status: <state> workers: <UUIDs>`. Clear only that
unit on acceptance. Preserve stable IDs and independent worker attribution.

Three accepted outcomes below half their estimates, or a queue that cannot reach the
destination, is a planning finding. Explain and re-estimate in Forecast. Wake supervision
after a step-back, every third acceptance, repeated workaround, spend spike, planning
finding or new human decision: `prompt-manager team heartbeat-trigger effort-supervision
effort-supervisor`. Record the event first; use at most one event wake per schedule window.

## Steering, waits and decisions

Record operator/supervisor feedback verbatim in FEEDBACK, then translate applicable items
into brief directives: `- D<n> <ISO time> <instruction>`. Workers acknowledge directive IDs
in their progress log. Keep open feedback active until its actual condition is resolved.
Replace the single STATUS handoff through `agent-manager effort handoff set <home> --file
<handoff-file>` after a changed check-in. Name every open feedback ID. A no-change timer
wake writes nothing. An approved source update does not require replacing its native owner.

Use `agent-manager run continue <child> --message <text>` for a running interactive child.
It cannot take `--reinstall-goal`. For a terminal interactive child, a one-turn status follow-up
omits that flag and leaves the retained objective's verdict intact. Use the flag only to
explicitly reinstall the objective on an eligible terminal run. Preserve the amendment
agreement between brief, acknowledged directives and installed objective; use the recovery
reference for uncertain or refused continuation. Orchestrate scope covers native children
and established effort lineage, not arbitrary runs.

After completing due independent planning/review, use `agent-manager effort park <home>
--json`. The default is the verified caller; an explicit `--run` must match it. Inspect
the actual status and await handle. A child return, native timer or authorized wake resumes
the same owner. A raw feedback edit does not wake it. A neutral signal check-in consumes
its old wait; inspect the actual child/producer and re-park if still waiting. Do not poll.

When all remaining work waits on human decisions/external repair, retain the outcomes and
return conditions in STATUS. Raise genuine choices through `effort decision raise` using
the owner reference's Decisions contract. Record decision ID, actual publication and answer
states; STATUS text is not notification. Use one stable decision, viable options and an
evidenced recommendation. Do not invent answers/defaults, duplicate asks or decisions for
ordinary repair waits. Wake supervision once, then park with `--timeout 72h`. Native pressure
and timed stopping still apply; a long park does not extend the commissioned budget.

## Validation targets and installs

Record required/occasional/unavailable targets in STATUS. A missing target holds the affected
proof, not independent work. On the existing daily qualification opportunity, run the named
qualification against its routed fixture when due. Retain dated results and reopen affected
acceptance when current evidence fails or no longer applies. Do not invent a new shadow or
full-suite prerequisite. Use supported isolation/fixture owners for effects.

The orchestrator authorizes goal-only external tool installs only within existing authority;
record tool, size, reason and cleanup in STATUS. Project dependencies use Scenario Dependency
Analyzer. Remove large goal-only installs and stop external processes on closure. Tool brands
are not product acceptance gates; preserve the required behavior/proof and genuine permissions.
