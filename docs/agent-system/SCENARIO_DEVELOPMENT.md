# Scenario development under a grant

An operator or manager can authorize one work item to develop a scenario against a reviewed target. The agent chooses and implements successive improvements inside that grant (R3): it does not request a new item for each engineering decision and needs no human between sessions. This document owns the grant model, the work shapes and execution modes, the authority classes and the documentation-first method. It declares no runtime API; documentation adoption is not runtime readiness, so [what runs today](#implementation-status) is stated below.

A direct session may apply this method when the operator authorizes it. Record that authority; do not invent a receipt. Workflow agents keep their actual grants, and this document cannot widen them. A review request alone permits no implementation.

## Work shapes

Choose the cheapest shape that survives the work's real durability and authority needs; a plan is one shape, not the default. This is the one definition of each shape (R7). Use the first row that fits:

| Shape | Use when | Carries | Authoring skill |
|---|---|---|---|
| Discovered Action | The operation is mechanical, reversible and fully specified by the Action's contract | The Action | n/a |
| Bounded task | One session finishes it and the route is recoverable from the docs and code. Docs-first authoring is itself a bounded task; if the design, acceptance or owner is missing from the docs, the first assignment is that task. | A harness goal over the docs; no plan | n/a |
| Investigation | The deliverable is a report or verdict, not a change | A read-only harness goal; never a plan | n/a |
| Owner-backed campaign | The operator explicitly selected a continuous improvement loop, the scenario forbids Plan Manager plans, and an existing owner declaration or controller already holds boundary, evidence, qualification and recovery state | The scenario target and improve skill, a bounded active packet, an immutable Agent Manager declaration and a harness goal. No plan, no worker-authored controller, no second scheduler. | n/a |
| `mandate` plan | The target is documented, an improve skill and setpoint sensors exist, and the route should come from measured evidence | A Plan Manager plan holding target pointer, sensors, bands, scope, stop rules and arc; a goal whose finish line is the setpoint board | `adaptive-mandate-authoring` |
| `phased` plan | Route, phase order, handoffs or a sequential change boundary are the requirement | A Plan Manager plan with ordered phases and a goal pointing at it | `implementation-plan-authoring` |

If a campaign's owner pieces are missing, repair the owner or choose a plan shape. Use a plan family only when several independently executable plan-shaped units need family topology (Plan Manager's family decision). Long-running efforts with their own home follow [LONG_EFFORT_DESIGN.md](LONG_EFFORT_DESIGN.md).

## Grant, shape and mode

| Term | Meaning |
|---|---|
| Target | The scenario's own statement of what it must become: `PRD.md`, `requirements/`, `docs/concepts/`, the improve skill and its sensors. It lives with the scenario, never in a plan. |
| Development grant | Authorization to pursue one plan-backed item inside its acceptance globs, effect policy and aggregate limits. |
| Plan-backed item | One backlog item linked to one canonical Plan Manager plan, with acceptance pinned to the plan's content hash. The grant is made by the manager (R5) and recorded in the plan. Swarm Manager lists and tracks the item and launches an Agent Manager run only under that grant (and, with `continuation: until-allowance`, continuation children inside it); the run executes in Agent Manager. |
| Plan shape | `phased` or `mandate` (planned Plan Manager field; current plans are phased). |
| Execution mode | `sliced` (an Agent Manager workflow of bounded slices, each with a review) or `goal` (one Agent Manager run under a harness goal). Chosen at run time, independent of shape. |
| Finish line | The condition in `/goal` and `until`. Phased: every phase recorded finished with evidence. Mandate: every setpoint row in band and the evidence audit passes. The manager one loop up decides closure, not the agent doing the work (R5); a harness `met`, completed phases or a green wrapper alone proves nothing (R2). |
| Operator note | Free text on the item that reaches the agent verbatim: intent, reminders, authority beyond defaults. |

| | Sliced | Goal |
|---|---|---|
| Substrate | A fresh worker run per slice, bounded turns, an independent review per slice, a decision node, approval where a gate is manual | One Agent Manager run; the harness loops under `/goal`; the finish line also rides in the prompt for runners without native support |
| Context | Each slice starts cold from the plan reference and last handoffs | One warm conversation |
| Bounds | `max_slices`; aggregate limits | Aggregate limits only |
| Recommended for | Either shape | Mandates |

**Stopping and resuming.** Trust the agent's verdict: a run that ends `complete`, `blocked` or `abstained` has stopped, and follow-up is a manager decision, never an automatic relaunch. Do not trust an interruption (usage window, timeout, crash, lost session with no result): it was cut off. Continuation is permitted only under `continuation: until-allowance`, while allowance remains; the sweeper starts a fresh child (never the same session), and three resumes with no Plan Manager progress halt the item for the manager. Under `continuation: manual` every stop is a stop. If Swarm's implementation requires human approval only because it cannot yet represent an already-authorized assignment, use a qualified Agent Manager route inside that assignment's fixed scope, acceptance and budget, record that the Swarm receipt was unavailable and which guarantees were reduced, and never fabricate an approval; a genuinely owner-granted effect (new scope, production access, paid spend, private data) pauses. Details: Swarm's [continuation contract](../../scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md#continuation).

**Item fields.** `plan_ref` and `plan_acceptance` (acceptance pins the plan hash; editing acceptance globs or scope policy invalidates it). `execution_mode` (`sliced` default). `execution_limits`: tokens, wall seconds, turns, charge, children, retries, one aggregate across resumes; unknown usage is not zero. `scope_policy`: `fixed`, or `extend-with-record` (the agent appends globs with `plan-manager exec boundary-extend` and a reason before the edit; `acceptance_deny` still refuses). `acceptance_allow` and `acceptance_deny` stay narrow (the target scenario, its shared packages, protos, docs), since a repository-wide allow list makes `fixed` meaningless. `operator_note` is bound verbatim. Runner, model and effort are per-run preferences, not pins.

**The goal message** is composed for plan-backed items, never typed ad hoc. Slots in order: destination, proof (the check whose output must appear in the transcript), sources, boundary, dials, blocked, budget, handoff, non-goals, then the operator note verbatim. `harness-goal-authoring` owns the slots and proof standard. Keep it concise; durable design belongs in the docs and plan.

## Implementation status

As of 2026-11-16, Swarm Manager persists `execution_mode` and `operator_note` on plan-backed items ([backlog fields](../../scenarios/swarm-manager/api/internal/backlog/types.go)); the [mode registry](../../scenarios/swarm-manager/api/internal/transitions/execution_mode.go) accepts `sliced` and `goal`, and [goal launch](../../scenarios/swarm-manager/api/internal/execution/service_control.go) creates one Agent Manager goal run. Plan Manager has no `shape` field yet ([model](../../scenarios/plan-manager/docs/concepts/PLAN-MODEL.md#implementation-status)). Legacy `execution_strategy` values migrate as `phased-plan-drain` and `adaptive-improvement` to `sliced`, `goal-session` to `goal`. The retired `contract-development` route is registered but never launch-ready. Remaining runtime work belongs to its owners: the shape field and mandate checks, same-session interruption resume, native/fallback qualification and derived skill applicability checks. Prompt changes alone do not deliver these.

## Retiring a scenario or capability

Deleting is a deliverable (R8), and these facts guard it so a retirement never removes what is still used (R6). Skip the windows only with evidence the item is dead.

- **Windows.** Propose retirement of a skill after 90 days with no active reference, an agent after 90 days with no heartbeat or invocation, a team after 120 days empty or disabled with no open work.
- **A skill is a candidate only when all three reachability classes are zero over the window:** static reference (`prompt-manager graph orphaned-skills`), discovery (`skill-usage` `returned`) and read (`skill-usage` `demandReads`, agent-member reads only).
- **Roadmap check before filing.** Any active director-swarm initiative naming it, any candidate or active SKU needing it (`offer-desk offers catalog-list` and `catalog-edges`), any other team's member file referencing it, or it being the only coverage of a needed capability each stops the retirement.
- **Archive path.** Set `status: archived`, keep 30 heartbeats so a member can ask to un-archive, move the files to `store/archived/<type>/<id>/`, and delete after 180 heartbeats.
- **Deadband.** A threshold states the target, never the current reading.

## Owners

| Owner | Owns | Does not own |
|---|---|---|
| Target scenario | Product contracts, architecture, requirements, domain state | Private dependency internals, copied evidence ledgers |
| Plan Manager | Canonical plans, rationale, dependencies, review references | Draft file storage, harness interpretation |
| Swarm Manager | The candidate-work list and plan-backed item bookkeeping | Running the work (Agent Manager does), or a duplicate plan |
| Agent Manager | Execution resolution, continuity, budgets, cancellation, recovery | Product acceptance criteria |
| Tech Tree Designer | Design graph and draft-bundle experience | A second plan manager or Git implementation |
| Prompt Manager / Program Runtime | Discoverable judgment and typed bounded compositions | Product state or self-issued effect grants |
| Test Genie and evidence providers | Observations, measurements, provenance | Permission to redefine the target |

Use Git Control Tower for Git and evidence operations. Applying source does not authorize commit, merge, push, release or deployment. Layers follow [the three-speed stack](SKILL_AUTHORING.md#the-three-speed-capability-stack).

## Target artifact contract

Reference existing artifacts; create no parallel registries. `PRD.md` (product intent and targets), `requirements/` (outcome obligations, status derived from evidence), `experience/` where adopted, `docs/concepts/` (architecture, ownership, interfaces), `docs/internal/` (seams, test methods, measurements, limitations), scenario `skills/`, `.vrooli/program-runtime/` and `.vrooli/service.json`. Include resource contracts, shared packages and protos when the target reaches them, and separate stable obligations from replaceable mechanisms.

Each required outcome needs a target, an evidence source or an explicit measurement gap, a validity rule and a repair route. Unknown targets stay undecided and unknown readings stay unverified; neither satisfies completion, and publishing a target does not assert its behavior exists. The improve role's setpoint rows point back to approved outcomes and are not a second source of targets. Separate required acceptance rows from diagnostic signals: a signal becomes a gate only when the approved contract says so, and an unreadable row does not remove a required outcome. Setup is assessed as **declared**, **semantically reviewed** or **exercised** (program fixtures plus a fresh-agent trial), reported separately and never as runtime status enums.

## Authorization and change classification

Identify in the instruction or work context: work identity and approved revision; permitted paths, dependency repairs and scope-extension policy; allowed effects (runtime changes, external calls, spending, publication); aggregate limits across continuations; required evidence and acceptance policy; checkpoint, cancellation and amendment procedures. An omitted budget is not unlimited authority. Both modes use terminal usage to admit later work and cannot contain an active run in flight, and all observed usage reduces one aggregate; exhaustion authorizes no new dispatch. The granted `execution_limits` (tokens, wall seconds, turns, charge, children, retries, and `max_slices` in sliced mode) are limits and stop dispatch when reached; planned pace is a friction signal (R2), not a limit. Spending money stays inside bright line 1. A historical approval gains nothing from a new default.

Discovered goals are candidate context, not additions to the mandate; approved priorities are preserved. Resolve conflicts before the affected action without blocking independent authorized work.

| Proposed action | Disposition |
|---|---|
| Choose an in-scope implementation, experiment, refactor or next repair | Do it and keep the rationale in the engagement |
| Repair a required sensor, binding, test, skill or program in scope | Build it through its owning method only if it is a reusable instrument (one command, used by a gate, reused later); otherwise record the row unverified and continue |
| Repair an agent-system core capability in scope | Use the owner's write and validation route; no separate promotion decision |
| Repair an explicitly authorized dependency | Work at its owner and validate the affected contract |
| Correct documentation to express approved intent | Update it; preserve contradictory evidence |
| Change an outcome, relax a floor, remove a required route or platform, or exceed the grant | Request an amendment first; continue independent work |
| Discover unrelated work or an ungranted dependency repair | Record it at the existing owner |

Initial approval binds reviewed changes, not every later edit. The agent may improve guidance and measurement code but cannot redefine protected acceptance semantics, and the approved target stays recoverable.

## Development loop

Read the target and evidence, choose an authorized intervention, implement, verify affected outcomes, checkpoint, repeat; return evidence when every required outcome is satisfied, and request an amendment when the work needs new authority. Skills: `scenario-skill-setup` (owed roles; links targets to sensors and repair routes), `<scenario>-improve` (which concern, how measured), `scenario-improvement-campaign` (successive repairs under one engagement), `goal-loop`, `scenario-work-ladder` (locates a broken layer; not a queue or permission gate), plan authoring and execution skills.

Observation-only use reads and recommends; it cannot infer permission to curate, file work, start services or implement. Checkpoint target identity, intervention, evidence, unmet obligations, owner waits and remaining budget in the owner's durable log and the shared Memory contract, with no duplicate ledgers.

Completion requires applicable evidence for every required outcome and the caller's acceptance policy. A green wrapper, all readable rows passing, two cached reads or a harness `complete` is insufficient. Sample counts come from the outcome contract. Revise the owner of a rule and then its callers, and delete obsolete instructions in the same change: an exception appended beneath a contradictory rule resolves nothing (R7).

## Documentation-first authoring

Author concrete target artifacts for changes to intent, interfaces, architecture, experience, skills and validation. Keep rationale, dependencies, risks and rollout in the plan; do not duplicate artifact contents in plan prose.

| Capability and grant | Authoring location |
|---|---|
| Qualified Tech Tree Designer draft operations | Owner-issued draft identity and workspace; link the revision to review |
| No draft path; operator permits canonical doc and skill edits | Edit those sources and list the changed files; preserve current-versus-target status |
| No draft path; proposals only | Preserve content in Plan Manager's returned artifact directory; record the missing integration; do not mutate canonical source |
| Caller forbids Plan Manager writes | Use its destination or response contract; report a missing destination |

Check artifact-kind, workspace, review and apply capabilities, not only service health; never invent draft commands or use live materialization as preview. Draft bundles need source bases, proposed paths, explicit additions and deletions, revision identity, links and evidence, and exclude secrets and runtime data. Unapproved skills and programs stay out of live discovery. Executable drafts need effect authority: a filesystem draft does not isolate ports, databases, credentials, network or billing, so use owner-managed test runtimes. Bind approval to exact changes, compare bases with destinations before application, preserve unrelated edits, surface conflicts and partial application as durable receipts, and use generators and domain publication operations for their artifacts. Review must expose material target, permission, budget, interface and platform changes without opening every file.

Owner specifications: [Swarm admission and amendments](../../scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md#plan-backed-execution), [Agent execution and accounting](../../scenarios/agent-manager/docs/concepts/ARCHITECTURE.md#target-goal-mode-run-contract), [Plan review references](../../scenarios/plan-manager/docs/concepts/ARCHITECTURE.md#target-artifact-first-review), [TTD revision and application](../../scenarios/tech-tree-designer/docs/concepts/ARCHITECTURE.md#target-extension-artifact-first-design-bundles). Validation scope and waits: [TESTING.md](../TESTING.md).
