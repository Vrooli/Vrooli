# Ecosystem Hunt playbook

Debt-removal techniques for the lenses in `SKILL.md` §9. Each entry gives the signal, a
detection command, and the fix that removes the class. The commands are starting nets, not
verdicts: confirm each hit against the source and the intended design.

Search tools: `rg` (ripgrep) for text, `ast-grep` for syntax. Exclude generated and vendored
trees (`-g '!**/node_modules/**' -g '!**/dist/**' -g '!**/gen/**'`). If `rg` is absent, use
`grep -rnE`.

When a technique here proves itself on a real fix, add the evidence (one line: where, what it
removed). When a detection command produces mostly noise, sharpen it here. This file is the
hunt's shared method; keep it true.

---

## 1. Make implicit state machines explicit

The highest-yield technique in Vrooli. Lifecycles for runs, jobs, plans, claims, approvals,
deployments and scenario processes are often spread across booleans, nullable timestamps,
status strings, event rows and "is the process alive". Scans for size and clones do not see
them.

**Signals**
- Two or more booleans on one struct that describe progress (`Started`, `Done`, `Cancelled`).
- Nullable timestamps used as state (`CompletedAt == nil` means "running").
- Status strings compared ad hoc in several files instead of one switch on a named type.
- Retry, cancel or recovery logic that re-derives "where are we" from several fields.
- Two components that each decide whether the same thing is finished.

**Detect**
```bash
rg -n -g '*.go' 'status\s*==\s*"' scenarios internal packages -l     # ad hoc status compares
rg -n -g '*.go' '(Completed|Finished|Cancelled|Started)At\s+\*time\.Time' scenarios internal
rg -n -g '*.go' 'json:"is_[a-z_]+"' scenarios -l                       # boolean state flags
rg -n -g '*.sql' -g '*.go' "(completed|finished|cancelled)_at IS (NOT )?NULL" scenarios
```
Then pick one lifecycle and reconstruct it: list every variable that encodes its state,
every write site, every read site that branches on it.

Before using a fleet score as a conversion shortlist, verify the candidate's actual durable
write owner and whether it is an independent aggregate. A value nested in an in-memory
projection or memory-only store is not persisted just because it reaches a generic serializer;
several status fields nested in one persisted parent may be one owner, not several. Match hunt
closure to the same entity and conversion: a scenario mention or detector improvement is not
closure for every lifecycle in that scenario. Keep the score order, exclusions, source links and
exact hunt result in a register so the next owner can reproduce membership.

**Fix**
1. Enumerate the reachable combinations of the state variables. Most combinations are
   invalid; find which ones the code can still produce. Each one is a latent bug.
2. Name the states as one type (Go: a string type with constants and an exhaustive
   `switch`; TS: a discriminated union; SQL: a `CHECK` constraint).
3. Write the transition table: from-state, event, guard, to-state, side effect.
4. Give transitions one owner: a single function that validates the transition, persists
   it, and emits the event. Every other write site calls it.
5. Reject invalid transitions with a typed error. Do not ignore them.
6. Derive the old flags and timestamps from the state, then delete the stored copies.
   Keep a timestamp only as data ("when"), never as state ("whether").
7. Define the state after a crash at every step: what does recovery see and do?
8. Test the table: one case per valid transition and one per important invalid one.

Independent dimensions (for example "lifecycle" and "health") stay separate types; do not
flatten them into one machine with a product of states. Make illegal states
unrepresentable; parse, don't validate.

Prior evidence: Template Manager drift status had no "unknown" state; adding a first-time
unverified state removed contradictory status/flag output (2026-11-04).
Prior evidence: Swarm Manager Agent Session and proposal statuses had scattered direct writes;
one transition table per persisted status plus store-boundary enforcement removed the
bypasses and stopped stale run snapshots from overwriting proposal review (2026-11-12).
Prior evidence: Swarm Manager review rounds now validate Gathering→Complete/Failed at the
single file-store boundary and reject persisted terminal reversals (2026-11-12).
Prior evidence: Client Rehearsal coding jobs now route stage and evaluation results through one
transition owner; the direct failed-then-strict-pass writes and separate evaluation state guard
were removed. Transition and invalid-state matrix tests pass; production is +22 net lines and burden-19
top-20 membership remains unassigned (debt-reduction run `steward-free-host-maintainability-20261115/DR2`).
Prior evidence: Scenario-to-Cloud deployment results now persist without changing status; operation completion is the sole terminal projector. A SQLite integration regression catches the success path persisting failed before attempting deployed; the handler reset, bool-selected result branch and duplicate status writers were deleted. Production +144/−60 (+84 net); owner unit/API Test Genie `20261116-042320-2d8f7f81` passes (debt-reduction run `steward-free-host-maintainability-20261115/DR2`).
Prior evidence: Agent Manager workflow executions now route runtime status intents through
one domain graph and reject illegal persisted edges at the status+version CAS boundary; the
table and child-cancellation propagation have an exhaustive invalid-state regression (2026-11-12).
Prior evidence: Code Facts index jobs now use a typed transition graph and SQLite source-state
CAS; removing the stored cancellation boolean eliminated a second durable representation (2026-11-12).
Prior evidence: React Component Library component-test sweeps checkpoint in place only while
`running`; blocked/failed runs must resume through `running`, and completed snapshots cannot be
rewritten. The SQLite owner rejects invalid edges without changing persisted results, while the
handler checkpoints progress and records failed/blocked/complete outcomes through the same
boundary. This closes a terminal same-state save loophole; no obsolete independent flags existed
and no production line reduction is claimed (debt-reduction run `steward-free-host-20261114`).
Prior evidence: Swarm Manager proposal decision publication is a persisted outbox nested in a
proposal, not a projection of proposal review status. Its immutable event snapshot can remain
pending across publish failures, become published only after a durable receipt, or remain
legacy-unresolved until an exact-terms reissue preserves the earlier uncertainty. The store
owns these edges; no replaced flags existed to delete (2026-11-12).
Prior evidence: Agent Manager transcript-index files now use one persisted status graph at
the SQLite upsert boundary; invalid or unknown transitions cannot enter the durable file row,
and same-state checkpoint writes preserve crash recovery metadata (2026-11-12).
Prior evidence: Plan Manager validation operations now use one `queued → running → terminal` owner; receipt projection calls the transition method and the SQLite transaction compare-and-swaps the prior status, so stale queued receipts and terminal resurrection are rejected. Focused validation and HTTP seam packages pass; Test Genie unit/API run `20261114-132156-be4ebd47` passed (unit L2→L3 retains `MISSING_INJECTABLE_SEAM`; API L1→L2 retains three unstructured-logging findings). During the same owner pass, Plan Manager deleted its duplicate local `FakeDoer` implementation and tests (87 implementation lines and 88 duplicate test lines) and reused `api-core/apihttptest.FakeDoer`; the lifecycle enforcement added 70 net production lines, so this is a test-support reduction rather than a production-line reduction (debt-reduction run `steward-free-host-20261114`).
Prior evidence: Swarm Manager goals now use a typed status graph and reject illegal existing-
record transitions at the atomic JSON store boundary; archive validates before dropping
targeted work, while achievement retains its independent milestone-evidence gate (2026-11-12).
Prior evidence: Agent Inbox tool calls already had a typed status graph and SQLite transition
guards, but restart review found a missing durable step: approval is persisted before tool
execution while `running` is only set in memory. Startup recovery now includes persisted
`approved` calls and cancels them with an explicit unknown-outcome reason. It does not replay
the tool because a crash after an external side effect but before result persistence could
duplicate that effect. This is at-most-once recovery with an uncertain user-visible outcome;
record that limit rather than claiming exactly-once execution (2026-11-14).
Adjudication: Scenario-to-Desktop PlatformResult.Status is only owned by an in-memory build
store; BuildStage persists one final StageResult.Details snapshot after synchronous build
completion, and crash recovery does not restore the platform transitions. Do not count this
as a persisted lifecycle conversion. The inventory's nearby bundle-resource sink is unrelated
candidate flow. The same scenario's pipeline Status.Status is the persisted lifecycle: each
store update is atomically written to the pipeline file and pending/running states are marked
failed after restart. (2026-11-14)
Prior evidence: Scenario-to-Desktop pipeline Status.Status now uses a typed transition owner;
idle/pending/running transitions, failure, cancellation, terminal reversal rejection and
unknown state rejection have matrix coverage. Existing stale-pipeline recovery continues to
move interrupted pending/running records to failed through that owner (2026-11-14).
Prior evidence: Scenario-to-Desktop investigations now route pending/running/completed/failed/
cancelled writes through a domain transition graph under the JSON store lock. Findings and error
helpers no longer assign terminal status directly; invalid edges leave the stored record intact.
Focused domain/persistence and full API Go tests pass; the current unit/API Test Genie run is
still in progress, so do not count its owner phases green yet (2026-11-14).
Prior evidence: Browser Automation Studio recording sessions now validate `active -> closed`
through one domain transition owner and SQLite status CAS; unknown states and reopening are
rejected, repeated close remains idempotent, and the unused derived `IsActive` branch is gone
(2026-11-14). The top-ranked `NavigationSession.Status` candidate was in-memory; its apparent
durable sink belonged to an unrelated `ExportIndex`, so trace the exact aggregate to storage
before admitting an inferred lifecycle (debt-reduction run `steward-free-host-20261114`).
Prior evidence: Swarm Manager's durable `BacklogItem.Status` mutations now pass through one
actor-specific edge graph shared by PATCH, review decision, retry, recovery, follow-up,
archive/reset, goal archival, import, proposal, and execution callers. The former known-status-pair
predicate and generic patch status field were removed. A production assignment scan found no
current writer bypassing the graph. The low-level JSON `SaveItem` API still accepts a full model
without actor identity, so a future caller could bypass by saving a pre-mutated value. Swarm
Manager owner unit Test Genie passed in run `20261114-114107-34214fcb` (L2→L3, with the existing
injectable-seam warning); no production line reduction is claimed (debt-reduction run
`steward-free-host-20261114`).
Prior evidence: Swarm Manager review `RequestThread.Status` is persisted inside review-round JSON and has its own pending→fulfilled/dismissed lifecycle, distinct from the parent round. Creation, evidence settlement, dismissal and fulfilled-evidence projection now route through named child-state operations; invalid, terminal-reversal and replay tests cover recovery. Exact candidate terms had no matching entry in the complete 112-entry hunt export. Owner Test Genie result is recorded in debt-reduction run `steward-free-host-20261114`.

Prior evidence: Deployment Manager readiness reviews now own their persisted status graph (`agent_review → approved → promoted → superseded`) and evaluation refresh separately. SQL evaluation replacement compares the observed status and rolls back stale writes; unknown stored statuses fail closed, and promoted reviews cannot be reopened. Domain/repository invalid-state, terminal-reversal and stale-write tests were added. Focused tests and Test Genie remain unrun pending the shared host's 120 GiB free-space floor; record this conversion as unverified until those owner phases pass (debt-reduction run `steward-free-host-20261114`).

Prior evidence: Compute Manager `intent.Record.State` now routes new and persisted writes through one owner with a state compare-and-swap; Tunnel Manager `exposure.Lease.Status` uses a transition graph and full-snapshot CAS. Both include invalid-state coverage and caller migration; their owner validations remain unrun under the same disk admission floor. Do not count either as a green conversion or a supported top-20 member until owner tests and exact hunt closure are verified (debt-reduction run `steward-free-host-20261114`).

Prior evidence: Web Console session metadata now has one SQLite transition/CAS owner and the matching in-memory graph; archived/recovered-session behavior is preserved and invalid/terminal tests were added. Brand Manager logo candidates now use a persisted transition graph, expected-state CAS and one transaction for picked-candidate replacement; the generic status update and separate superseding writers were removed, and Connect reports invalid transitions as failed precondition and stale CAS as aborted. Git Control Tower proposal state writes from services and transactional combine/publication now share one graph while retaining revision CAS and publish-failure recovery. Exact current hunt records were checked for each field; these owner tests and Test Genie phases are unrun under the shared 120 GiB floor, so record all three as runtime-unverified and do not rank/count them as green conversions yet (debt-reduction run `steward-free-host-20261114`).

Prior evidence: Agent Manager `Task.Status` had a tested domain graph that durable writers bypassed with SQL status allowlists. Task creation, attempt claim/cancel and Run-driven status projection now validate against the graph; projections use exact observed-state CAS, reads reject unknown persisted values, and the in-memory caller uses the same transition method. A missing `needs_review → running` continuation edge is now explicit and remains fenced by exact-current-Run admission. Regression tests are authored but owner validation is unrun below the shared disk floor; do not count this as green or assign top-20 membership yet (debt-reduction run `steward-free-host-20261114`).

## 2. One owner per concept (double residency)

**Signals**: the same enum, regex, constant, port, path, schema or decision appears in two or
more places; two CLI commands or two endpoints do the same job; two docs state the same rule;
a client re-implements a server decision.

**Detect**
```bash
rg -n -F '<distinctive literal>' scenarios packages internal docs   # every home of one value
rg -N --no-filename -g '*.md' -g '!**/evidence/**' '^#{2,3} \S' docs scenarios/*/docs | sort | uniq -c | sort -rn | head -30   # repeated headings: boilerplate or copied rules
tidiness-manager recommend-refactors <scenario> --limit 10            # ranked reduction candidates
```

**Fix**: choose the owner with `docs/agent-system/README.md#where-does-this-paragraph-go` (truth in Plan of Record,
judgment in skills, state in scenarios, shared code in `packages/`). Move every consumer to
the owner. Delete the other copies. Write a retirement manifest: each deleted path and
symbol, with the search that proves it is gone. Do not leave a fence or a sync job between
two owners (`improvement-do-and-dont` D4).

## 3. Finish migrations; delete the legacy path

Vrooli is greenfield: new structure lands without compatibility fallbacks
(`docs/repo-contract.md` §"Compatibility Policy").

**Signals**: `legacy`, `deprecated`, `compat`, `fallback`, `old`, `v1`, `transitional` in
code or docs; a feature flag that is always on; a reader that accepts two formats; docs that
describe both the old and new way.

**Detect**
```bash
# comments that admit a legacy path (~220 files; the bare word matches thousands)
rg -n -i -g '*.go' -g '*.ts' -g '*.tsx' -g '!**/node_modules/**' -g '!**/gen/**' -g '!*_test.go' '//\s*(legacy|deprecated|compat(ibility)? (shim|path|layer)|transitional|TODO.*(migrat|remove))' scenarios packages internal -l
rg -n -i -g '*.md' '^#+ .*(legacy|deprecated|migration)' docs scenarios -l
```
Narrow further by territory; then read each hit's callers.

**Fix**: list the callers of both paths. Move every caller to the new path. When real data
exists, migrate it with an out-of-tree script (`storage-steer`): snapshot the data, dry-run,
apply, and verify counts after. Delete the old path,
its flag, its tests and its docs. Write the retirement manifest. A migration that only adds
the new path is unfinished (D7).

## 4. Make silent failures loud

**Signals**: an error that becomes a log line and a default; an empty `catch`; `|| true`;
`2>/dev/null` on a command whose failure matters; a parse failure that returns zero; a gate
that passes when its input is missing.

**Detect**
```bash
ast-grep run -l go -p 'if err != nil { $L.Warn($$$) }' scenarios internal packages
rg -n -g '*.go' '_\s*=\s*err\b' scenarios internal packages
rg -U -g '*.ts' -g '!**/node_modules/**' '\.catch\(\(\)\s*=>\s*(\{\s*\}|undefined|null)\)' scenarios packages
rg -n -g '*.sh' '\|\|\s*true' scripts scenarios internal
```

**Fix**: return a typed error to the caller that can act on it. Make a gate fail or report
"unknown" when its input is absent; never "pass". Keep a fallback only when it is a designed,
documented degradation with a visible signal. Add a test for the failure path.

## 5. Make docs true, once

**Signals**: a command or flag in a doc that the CLI rejects; a path that does not exist; the
same rule in two docs with different wording; dated narrative ("on 2026-09-14 we…") inside a
current-instruction doc; a doc that describes a removed design.

**Detect**
```bash
rg -n -o -g '*.md' '`(vrooli|test-genie|agent-manager|prompt-manager|plan-manager) [a-z-]+( [a-z-]+)?' docs scenarios/*/docs | sort -u   # then run each with --help
rg -n -o -g '*.md' 'path:[^ )`]+' docs | head                                    # machine-readable refs to check
rg -n -g '*.md' '^\s*(On|As of) [0-9]{4}-[0-9]{2}-[0-9]{2}' docs
```

**Fix**: code is the truth for behavior; operator-authored canon is the truth for intent
(`SKILL.md` §3). Rewrite the loser. Keep each assertion in one home and link to it from
others. Write in the present tense. Remove history from instruction docs; retain it only
where it carries an unresolved obligation. Run the scenario's docs phase after editing.

For a contradictory doc set: pick the one home per `docs/agent-system/README.md#where-does-this-paragraph-go`, merge the
true content into it, turn the others into one-line links or delete them, and fix inbound
links. `knowledge-observatory-maintenance` owns the move and retirement mechanics.

## 6. Fix agent friction at the owner

**Signals**: agents run `--help` before a command, guess flags, retry with different syntax,
or misread output. Friction digests and Agent Manager command-failure measures show these.

**Fix**: change the owner, not the skill prose. Make the error message name the correct
invocation. Accept the obvious spelling. Make the default output answer the common question.
When the same clarification appears in several skills, move it into the CLI and delete it
from the skills (`docs/agent-system/SKILL_AUTHORING.md#promotion-ladder`). Verify by running the bad
invocation before and after.

Prior evidence: fixing `scenario setup --json` rejection and capacity-release syntax at the
shared owner removed repeated guessing across runs (2026-11).

## 7. Delete dead weight

**Signals**: exported symbols with no caller; files no build includes; config keys nothing
reads; scripts no doc or Makefile names; tests for removed behavior; skills nobody reads.

**Detect**
```bash
rg -n -w '<SymbolName>' scenarios packages internal          # per candidate
prompt-manager graph orphaned-skills                          # unreferenced skills
```

`orphaned-skills` shows missing references, not missing reads. Before retiring a skill,
search the corpus for its id and check Agent Manager usage; follow
`docs/agent-system/RULES.md`.

**Fix**: delete it, with a retirement manifest. Do not comment it out or keep it "in case".

## 8. Enforce hidden ordering

**Signals**: comments like "must be called before", "call X first", "after init"; setup that
works only in one order; a struct that is invalid until a second method runs.

**Fix**: make the order structural — a constructor that returns a ready value, a builder
whose final step is the only way to get the usable type, or a state type that only exposes
valid next operations (typestate). Delete the comment once the code enforces it.

## 9. Remove redundant work

**Signals**: a query inside a loop (N+1); repeated filesystem scans per request; the same
expensive read in several handlers; polling where an event or a single wait exists.

**Fix**: measure first (benchmark, timing, `performance-health`). Batch the read, cache at the
owner with explicit invalidation, or replace polling with the owner's wait. Report the
before and after numbers.

Prior evidence: Agent Manager's effort board used sequential per-run state and event
accounting reads; a page-level run/event batch removed that fan-out while preserving per-run
metering and isolated corrupt streams (hunt-2026-11-05 Round 80).

## 10. Fix the generator, not the copies

**Signals**: the same defect in many scenarios with the same file layout; code under a
template's tree; generated shells or sidecars.

**Fix**: find the template, generator or shared package (`template-manager`,
`packages/`, `cli-core`, `api-core`). Fix it there, then regenerate or update every copy
through the owner command. Never hand-edit generated output (AGENTS.md §3).

## 11. Separate nested durable lifecycles by transition owner

**Signals**: one persisted aggregate contains several status-bearing child records, and each
child advances under different runtime events, retries, or recovery metadata.

**Fix**: do not collapse these into one candidate solely because they share a JSON row or
repository. Trace the writer and recovery path for each child. Count children independently
only when each has its own transition boundary and behavior; keep parent phase/progress fields
out of the child lifecycle. Conversely, a result label copied from a parent or immutable
artifact remains one owner, not another lifecycle. Add invalid-edge checks at the durable
update boundary and preserve same-state updates only when they carry explicit recovery
metadata.

Prior evidence: Swarm Manager's execution record embeds one finalization parent and separate
per-scenario restart, health, and review steps. Restart retries, health probes and asynchronous
review polling advance independently; their shared JSON persistence does not make them one
state machine. Their update helpers now reject illegal edges and keep an active child step
consistent when its parent fails (debt-reduction run `debt-crypto-models-20261112`).

Prior evidence: Swarm Manager dispatch intents now use a typed `submitting -> unknown -> acknowledged`
recovery graph (with direct `submitting -> acknowledged` and idempotent same-state recovery
updates). The durable store rejects unknown states, invalid edges and changed immutable owner
identity; runner callers no longer assign raw state strings. Lost-start recovery still joins only
the original owner and never resubmits. Focused transitionrun and transitionrunner Go packages
pass. The first Test Genie attempt exposed that the owner reply adds execution/run provenance
after the reservation write; the durable identity check was narrowed to permit that binding only
on acknowledgement. Corrected Swarm Manager unit run `20261114-122456-ff8601f1` passed (450s,
82 observations; standing `MISSING_INJECTABLE_SEAM` warning remains; debt-reduction run
`steward-free-host-20261114`).

## 12. Model derived recovery states without persisting extra flags

**Signals**: a stored state and timestamps jointly determine whether a retry/probe is currently
allowed, or a compatibility state remains readable although writers no longer create it.

**Fix**: keep the persisted lifecycle small. Make the transition owner accept only declared
edges, validate unknown values at the durable boundary, and derive time-dependent effective
states from the stored state plus the injected clock. Preserve legacy persisted states only with
explicit recovery semantics and tests. Do not add a second persisted state or boolean for a view
that can be computed from the existing state and time.

Prior evidence: Agent Buyer Range `trials.Record.Status` is stored in canonical trial JSON, but its accepted `prepared|observed|incomplete` value is fixed at insert and SQLite triggers reject both update and delete. `Get`/`List` only rehydrate that original result. A status field plus a timestamp and durable sink therefore do not establish a mutable lifecycle; verify the owner can progress and recover the record before converting it (debt-reduction run `steward-free-host-20261114`).

Prior evidence: AI Gateway ProviderHealth persists `open` with `CooldownUntil`; routing derives
`half_open` after the cooldown rather than storing a second flag. The transition owner supports
legacy `half_open` records but current runtime only writes closed/open. Invalid state values are
rejected by the repository and transition graph (debt-reduction run `debt-crypto-models-20261112`).

---

## 13. Make persisted child-state policy one owner

**Signals**: a durable record has one repository writer, but legal state is duplicated between a model predicate and a literal persistence condition; reads accept unknown stored values.

**Fix**: put the allowed edge graph on the domain transition owner, route the write through it, compare the exact observed state at the durable boundary, and fail closed on unknown persisted values. Preserve same-outcome idempotency and represent retries as new linked records when history is append-only. Add invalid-edge, stale-write, unknown-state and retry-history coverage.

Prior evidence: Vrooli Bridge `EnrollmentAttempt.State` had terminal logic in both `Terminal()` and a SQL `IN ('created','running')` guard. `AttemptState` and `EnrollmentAttempt.TransitionState` now own terminal outcomes; creation is insert-only, scans reject unknown values, and completion uses exact-state CAS while preserving same-result idempotency and linked retry records. Invalid, stale, terminal and error-mapping regressions were authored. Source changes are +63 production, +62 tests and +5 docs; Go/Test Genie remain unrun below the host free-space floor, so runtime is unverified (debt-reduction run `steward-free-host-20261114`).

## Evidence entry points and file coverage

Several CLIs rebuild on first call and print build noise, which is not an error.

- Team inboxes: `prompt-manager team knowledge-list <team> --topic-prefix=<declared-prefix> --last 1000` (prefix, severity and triage rules are in the team's `topics.json` and `scenarios/prompt-manager/docs/internal/TOPICS.md`; close resolved items through the owner command).
- Agent friction: `agent-manager usage friction --status open`, `usage opportunities`, program `agent-manager.friction-digest`; write verdicts back with `usage friction-dismiss <fingerprint>` (after a fix or reviewed no-change verdict) or `usage friction-report <fingerprint>` (owner defect you did not fix; files to the scenario-qa bug inbox), passing the same `--since` you listed with.
- Bugs and findings: `plan-manager log list --type bug_report` (many show sync failed and never reached the inbox) and `--type finding --triage candidate`; `swarm-manager captures list` (failed classifications are prey); `test-genie fleet status`, `runs findings --scenario <s>`.
- Debt: `template-manager debt list` (fix the template, §10); `tidiness-manager scenarios list|issues list|recommend-refactors <s> --limit 10`, `quality-health audit run <s>` (read `tidiness` for detector precision; scanner output is not a reduction estimate); `knowledge-observatory knowledge-base health --scenario-name <s> --skip-external-links`; `git-control-tower review summary <s>`.
- Prior work: `search-hub query "<territory> <lens>" --type record,library --limit 5`; `vrooli-memory recall recall "<query>" --scope agent-memory --limit 10`.

File coverage with visited-tracker is optional; use it when several hunters split one large territory (exclusive claims) or the lens is a file-by-file sweep. Tag `eh-<hunt-id>-<territory-slug>` (add `-<lens>` for a second lens); `visited-tracker attention ensure-campaign --location "$(pwd)/<territory path>" --tag <tag> --patterns '<glob>' --max-files <files + 50%>` (one `--patterns` per glob; `--max-files` is fixed at creation; if `attention preview` says the pattern exceeds it, use a new tag with a larger `--max-files` or narrower `--location` or `--patterns`), then `attention preview --campaign-id <id> --limit 20`. Zero candidates or a scan error means the campaign is unusable (paths containing `data`, `build`, `tmp`, `out`, `dist`, `coverage` or `node_modules` are excluded, and a broken symlink aborts the scan): skip coverage for that territory and record the defect. Claim with a fresh UUID `--request-id`; complete `--outcome reviewed --evidence <summary>` before editing the file. Read progress from `attention preview`, not `coverage` or `campaigns list`, which ignore reviews, and stop when preview shows no unreviewed candidates.

## Related skills for depth

`architecture-lenses` (concept compression), `cognitive-load-reduction`, `resilience-lenses`, `storage-steer`, `scientific-debugging`,
`documentation-health`, `knowledge-observatory-maintenance`. Load one when its lens is the
round's lens; do not load them all.

Prior evidence: Plan Manager `CandidateRevision.State` is a persisted independent child lifecycle. Replace permissive upsert with insert-only pending creation, a pending→applied/discarded/expired graph and transactional compare-and-swap; apply the canonical plan and applied candidate in the same SQLite transaction. Invalid-state, terminal-rewrite and injected rollback tests pass. Test Genie unit/API `20261114-134808-c22b3fd2` passed on the final single-statement CAS tree (unit L2→L3 retains `MISSING_INJECTABLE_SEAM`; API L1→L2 retains unstructured-logging findings). Earlier run `20261114-133622-3d19b19e` passed the first implementation before the 21-line store simplification. Production source changed +93/−20 and tests +79. The conditional single-statement CAS removed 21 lines from the initial store transition implementation while preserving error classification; overall production code remains +73 net, so this conversion improves transition safety and atomicity but does not claim net LOC reduction (debt-reduction run `steward-free-host-20261114`).

## Lens questions

Each lens is a question asked of every file in the territory (SKILL.md §9 names them).

- **Implicit state machines:** which lifecycle hides in booleans, nullable timestamps, status strings and process existence, with no named states or transition owner?
- **Double residency:** which concept, decision, constant or doc assertion has two homes?
- **Unfinished migrations:** which old path still runs beside its replacement?
- **Silent failure:** where does an error become a default, a log line or a "green"?
- **Doc truth:** which doc says what the code does not do, says it twice, or describes the past?
- **Agent friction:** which command, help text or error made an agent guess, retry or fail?
- **Skills and prompts:** which contradicts canon, restates another, or steers toward caution, ceremony or the wrong owner?
- **Test burden:** which tests pin implementation, hardcode cases a generator should make, or pass for the wrong reason?
- **Dead weight:** what has no live caller or reader?
- **Hidden ordering:** which steps need an order nothing enforces?
- **Performance:** which hot path does N+1 reads or repeated scans?
- **What we would not build today:** which part would we not create now, which three things are one thing, which subsystem could disappear if its one real consumer used an existing capability, and what change makes this category of bug impossible?
