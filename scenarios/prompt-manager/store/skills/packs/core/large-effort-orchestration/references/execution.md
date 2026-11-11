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
- Scenario: <exact scenario slug; required for receipt-backed journey gates>
- Kind: feature | refactor
- Started: <owner-observed ISO time at admission>
- Estimate: <optional planning estimate; omit when not useful>
- Expected size: <optional source-size estimate; never an execution cap>
- Metric: <observable outcome and command/evidence that reads it>
- Targets: <required, occasional or unavailable platforms/builds>
- Workers: <actual returned run IDs>
- Status: admitted | in-progress | parked | accepted

## References
Current product design target/version, applicable amendment/refinement IDs, exact requirements,
selected original images and current evidence. Label historical observations.
## Ownership and dependencies
Owned files/modules, shared interfaces, peer ownership and necessary UI/API/refactor work.
## Deletion list
Superseded responsibilities and intended end states, or why no deletion is justified.
## Exit gate
Commands and affected journeys; served/current-target comparison, expected differences linked
to adopted amendment/refinement IDs, and preserved-behavior checks on applicable media/themes.
## Non-goals
Actual exclusions; preserve unmet whole-product work in STATUS.
## Directives
## Slice log
```

Before dispatch, reconcile the current source/failure so already-fixed work is not reassigned.
Release terminal peers' file ownership; an integrated outcome includes its shared
composition and fixture dependencies. Explicit scope amendments preserve real live-peer
conflicts, not stale ownership. Select model/reasoning through the existing profile and
role policy for this outcome, not a hard-coded recipe.
Capture the dirty-scope anchor through `git-control-tower proposal anchor --effort <ref>
--epoch E<n> --scope '<scenario path>/**' --json`. Obtain the caller run from
`agent-manager run identity --json` (`claims.run_id`). Use existing authorized profiles.
Send the complete assignment in the task description:

The native `--until` is a kernel of 600 characters or fewer (`harness-goal-authoring` §2a): it
requires the assigned product outcome and its demonstrated behavior/fidelity, quotes about 150
characters of the charter's stance (`Stance: "<excerpt>"`), names the brief path to re-read after
compaction, and tells the worker to end its turn after each slice-log line. The brief carries everything else. "Produce comparisons", "run checks" or "return an honest report"
are evidence duties, not completion predicates. A genuine unresolved mismatch against the
current target is unfinished work; an implemented verified amendment/refinement is an
expected difference, not a reason to undo an improvement. Follow `experiential-ui-design` §2a.
When a retained predicate makes that mistake, correct it on an eligible native
continuation and keep the same outcome/owner; do not count a partial report as delivery.

```bash
agent-manager task create --title "<goal> E<n>" --scope-path <scenario path> --description "$(cat -- /absolute/worker-brief.md)"
agent-manager run create --task-id <returned-task> --profile-id <existing-profile> --parent-run-id <caller-run> --until "<assigned user-visible behavior and demonstrated acceptance condition>" --model <canonical-profile-model-id> --effort <permitted-reasoning> --workload-key <goal>/E<n>
```

Add `--charter <absolute brief or charter path>` so the child's lineage header names the file to
re-read after compaction. Wait with `agent-manager run wait --children <caller-run> --since
<observed_at>`, not repeated `run get`. Exit 6 means Agent Manager is unreachable: continue with
a native sub-agent and the same brief, never stall. Child depth is capped by
`runExecution.maxChildDepth` (`POLICY_CHILD_DEPTH_EXCEEDED`). Only an Agent Manager-launched
run can be a parent.

Keep the file substitution quoted. Set an existing task's description with `task update
<id> --description "$(cat -- /absolute/worker-brief.md)"`. Do not add `--prompt`: it replaces
the task description. Keep `--until` separate from the task description and within the kernel
size; a longer objective is dropped at compaction (`LONG_EFFORT_DESIGN.md` §4a). The worker receives
the exact repository-relative brief path, parent ID, worker card, applicable directives,
source versions, applicable amendment/refinement IDs, gates and resume set. Verify receiving
when feedback changes an active assignment; reconcile its eligible native predicate through
the owner rather than leaving obsolete acceptance live. Record returned IDs in the brief and STATUS.
Use a canonical profile model ID (for example `gpt-6-luna`), not a shorthand. Correct only a
deterministic spelling refusal within that profile's permissions. Reconcile an uncertain
dispatch before retrying. Never strip identity or change principal to pass admission.

## Receipt-backed browser gates

For an admitted exact case set, declare `G2 journey: bas/cases/one.json,bas/cases/two.json @ workflow-health`.
After execution, append `gate G2=receipt workflow-health:<run-id>[,<run-id>...]` in
the existing slice log. Use the same owner run IDs after an unrelated change;
`agent-manager effort epoch-check <brief> --acceptance --json` derives coverage,
success and current dependency applicability from the provider. Newer supplied
case failures take precedence over older passes. Manual counts cannot qualify
this gate. Missing/stale evidence names the affected cases to rerun.

Workflow Health's [runbook](../../../../../../../../scenarios/workflow-health/docs/operations/RUNBOOK.md#reusing-current-browser-evidence)
owns dependency declarations and conservative fallback behavior. Review their
completeness before adopting narrow scopes. Unchanged browser evidence can retain
its original capture/review provenance; a new build does not require recapturing
every image. Still verify the applicable original, target amendments and any
material uncertainty. This replaces repeated bookkeeping, not visual review,
initial served-build qualification or independent product acceptance.

## Worker progress

The [worker card](worker.md#progress-and-step-back) owns progress-line grammar, advisory
estimate reviews, step-back triggers and the worker's exact check command. Use that card when
interpreting the worker's result; do not create a second protocol here.

## Acceptance and progress

Independently inspect each required gate, affected journey, selected source version,
deletion end state/callers, justified source growth and test-protection result. Inspect visual gates
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
finding, new human decision or destination-met claim: `prompt-manager team heartbeat-trigger effort-supervision
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

Choose the wait from the worker's execution lifetime:

| Observed worker lifetime | Wait |
| --- | --- |
| Durable owner survives the parent executor exiting | Use the native park below. |
| Harness-local collaboration worker stops when the parent executor exits | Keep the parent executor alive. Use the harness's long collaboration wait through the worker's return. |
| Lifetime or live owner is unknown | Qualify the actual owner handle and shutdown contract before choosing a wait. |

Check all live fallback workers before native parking, including workers outside the native
child list. A task-start marker or first tool call does not complete their work. If a fallback
already stopped, use the recovery reference to resume it or transfer its retained context to
a permitted durable owner. Preserve its source and evidence. Do not duplicate a live owner.

Producer CLI waits can also park the inherited run. While executor-bound workers are live,
run these waits with `VROOLI_ASYNC_WAIT_MODE=inline` (for example,
`VROOLI_ASYNC_WAIT_MODE=inline vrooli scenario wait <scenario> --json`). This selects the
normal blocking producer wait without changing identity or authorization. Keep the returned
tool handle and wait on it once. The default `auto` retains native parking for durable workers.

After completing due independent planning/review, and with no live executor-bound worker,
use `agent-manager effort park <home>
--json`. The default is the verified caller; an explicit `--run` must match it. Inspect
the actual status and await handle. A child return, native timer or authorized wake resumes
the same owner. A raw feedback edit does not wake it. A neutral signal check-in consumes
its old wait; inspect the actual child/producer and re-park if still waiting. Do not poll.

When a genuine operator choice gates work, record the exact outcome and return condition in
STATUS, then raise one stable decision through `agent-manager effort decision raise` under
the enrolled effort. Give viable options and an evidence-backed recommendation. Make the Hub
question self-contained, with the absolute path to the current STATUS and a compact digest of
other open choices; tell the operator to answer in Notification Hub. STATUS is a navigation
projection, not the answer channel.

After raising, retain the decision ID and exact event/publication state. Confirm that Notification
Hub created the matching ask and that its delivery receipt exists before claiming it reached the
operator. A published event without a Hub ask or delivery receipt is pending/unavailable, not
sent. On every wake, read the canonical effort decision and matching Hub ask; after an answer or
dismissal, run `agent-manager effort decision sync <home>` and verify the verbatim result in
FEEDBACK.md and removal from `## Needs operator`. Do not solicit an answer in chat, duplicate an
existing decision, or surface a terminal decision again after compaction.

If the owner route cannot represent a required direct notification, use Notification Hub only
after qualifying the exact operation and recipient. Direct sends need `notification-sending`;
direct asks need `notification-decision-asking`. Request a missing exact grant through the
canonical authorization owner before promising delivery. Do not treat an ordinary repair wait
as a human decision. When all remaining work waits on human decisions/external repair, wake
supervision once, then park with `--timeout 72h`. Native pressure and timed stopping still apply;
a long park does not extend the commissioned budget.

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
