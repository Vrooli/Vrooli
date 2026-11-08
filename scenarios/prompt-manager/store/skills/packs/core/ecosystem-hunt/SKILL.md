---
name: ecosystem-hunt
description: "Hunt bugs, debt, friction and inconsistency across all of Vrooli in rounds, and remove each defect class at its root: rotate territory and lens so ground is not re-hunted, pull prey from friction and failure evidence before scans, size every fix to its defect class rather than to caution, and leave the ecosystem measurably simpler. For open-ended 'improve Vrooli' and debt-reduction mandates, not for one known bug."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [debt-reduction, bug-hunt, tidiness, state-machines, migrations, docs, ecosystem, maturity]
  icon: crosshair
  status: active
  revision: 6
  createdAt: "2026-11-05T01:50:00Z"
  updatedAt: "2026-11-06T23:45:00Z"
  requires:
    scenarios: [prompt-manager, search-hub, visited-tracker, vrooli-memory, agent-manager]
    commands: [prompt-manager skill read, search-hub query, visited-tracker attention claim, vrooli-memory journal note, vrooli-memory recall wake, agent-manager]
  origin:
    kind: authored
---

## Practice focus: Ecosystem Hunt

Find the defects that make Vrooli wrong, slow, confusing or expensive to change — anywhere
in the repository — and remove each one at the level where its whole class stops recurring.
Every round leaves the ecosystem with fewer owners per concept, fewer states nobody named,
fewer paths that should not exist, and docs that agree with the code.

Required reading, once per session:
- `prompt-manager skill read improvement-do-and-dont` — real fix versus metric gaming (D1–D7).
- `path:docs/agent-system/LAYERS.md` — every concept has one home; "no double residency".
- `path:docs/TESTING.md` — validation scope.
- `references/playbook.md` — the debt-removal techniques with detection commands. Read it
  before your first fix.
- `references/goal.md` — when you commission a hunt: the stance message, kernel, charter and
  segment brief.

Paths under `references/` are relative to this skill's directory
(`prompt-manager skill show ecosystem-hunt --json` reports it as `skillDir`).

**Scope.** In: code, tests, docs, skills, prompts, templates, generators, configs and shared
packages anywhere in the repository when the operator's current request authorizes those
changes. This skill supplies a hunt method; reading it does not expand the request or grant
authority by itself. Follow the actual change authority and the owner of each effect. Host
and out-of-repository state is changed only through its owner (the control plane in
`internal/`, the owning scenario's CLI). A single known bug with no hunt around it belongs
to `scientific-debugging`; a multi-day product delivery belongs to
`large-effort-orchestration`.

---

### 1. Stance

This section is the most important one in the skill. Read it as the operator speaking.

Be bold and ambitious, and exercise creative thinking. Do not be afraid to tackle large issues. This is a manhunt for Vrooli-wide improvements and maturity — emphasize debt reduction: make implicit state machines explicit, finish migrations and remove legacy code, and turn messy, duplicated or contradictory code and docs into one true thing. Size every fix to its defect class, not to caution. Uncommitted changes, other agents' recent work and unfamiliar scenarios are in scope; read them and integrate. Do not ask permission for refactors, deletions or migrations.

That paragraph is the operator's commission text. A hunt's charter, stance message and briefs
carry it verbatim, or the operator's own words for that hunt. This is a manhunt, not a tidy-up of
whatever you happened to open.

Past hunts failed in a specific way, and you will drift into the same failure unless you
resist it on purpose. They screened dozens of candidates and changed nothing. They rejected
a split that cut cyclomatic complexity from 75 to 19 because the file grew. They excluded
every file with uncommitted changes. They closed rounds with "bounded partial closeout, no
broad success claim" after running no tests. Each decision looked prudent. Together they
made the hunt a machine for concluding there was nothing to do, while the implicit
lifecycles, duplicate ledgers and half-finished migrations stayed.

So this skill changes the defaults:

- **Size the change to the defect, not to your comfort.** When another skill says "smallest
  change", read it as "the smallest change that removes the defect class", never "the
  smallest edit near the symptom". A fix that removes one instance and leaves the mechanism
  that produced it is unfinished.
- **Large is normal.** Merging two owners, making a lifecycle explicit across five files,
  finishing a migration across forty callers, rewriting a contradictory doc set into one —
  these are the expected unit of work, not exceptions that need permission.
- **The shared tree is your workspace.** Uncommitted changes, files another agent touched
  yesterday, and scenarios you have never opened are all in scope. Read what is there,
  integrate with it, and fix it if it is wrong. §6 lists the few real limits.
- **Use the authority already given by the current request.** Do not ask for redundant
  confirmation on necessary, reversible changes within that scope. A skill read does not
  authorize a consequential effect or a change outside the request; follow `AGENTS.md` §9,
  route a genuine operator-owned decision as in §7, and continue independent work.
- **Green is where looking starts.** A passing suite says the checks you have pass. It says
  nothing about the states no check covers.

Boldness is not accretion. A "bold" change that adds a layer, a fence, a compatibility shim
or a second owner is the opposite of this skill. The ambition is a simpler system: fewer
concepts, fewer paths, fewer places to remember. Measure it by what is gone and what can no
longer go wrong, not by how much you wrote.

---

### 2. Where the prey is

Evidence picks the territory; scans only fill it in. Scan-driven screening found almost
nothing in past hunts; friction data and tracing a real behavior end to end found the real
bugs.

| Order | Source | What it gives you |
|---|---|---|
| 1 | Operator words: delivery `FEEDBACK.md` files, `~/.vrooli/research/topics/debt-reduction/README.md`, operator rules in memory | Stated pain, direction, rejected approaches |
| 2 | Agent friction | Commands that fail, help lookups, retries, guessed syntax |
| 3 | Reported bugs and failing validation | Defects already seen and not fixed |
| 4 | Owner debt ledgers: `scenarios/*/docs/internal/PROBLEMS.md`, `SEAMS.md`, plan logs, template debt | Accepted-but-unpaid debt |
| 5 | Smell searches in `references/playbook.md` | Defects nobody reported |
| 6 | Scans: `tidiness-manager`, `quality-health` | Size, clone, complexity, lint leads |

A scan ranking is a lead, not a verdict; scan metrics have been stale before. Confirm
against the source.

Queues of single-command findings (friction, help lookups) hide classes. Before choosing,
group the queue by the missing mechanism: six "agents poll X" findings are one missing
wait/watch primitive; five "guessed command shape" findings are one discoverability defect.
Hunt the class. Write each verdict back to the finding's owner so the queue drains; a
finding closed only in a round record returns in the next wake. For Agent Manager friction,
run `agent-manager usage friction-dismiss <fingerprint>` after a fix or a reviewed no-change
verdict, and `agent-manager usage friction-report <fingerprint>` for an owner defect you did
not fix (it files to the scenario-qa bug inbox). Pass the same `--since` you listed with.

#### 2a. Source commands

Verified 2026-11-04. Counts change; the commands are the point.

| Source | Command | Notes |
|---|---|---|
| Team knowledge inboxes | `prompt-manager team knowledge-list <team> --topic-prefix=<declared-prefix> --last 1000` | Read the owning team's `topics.json` and linked taxonomy to identify the prefix, status, severity, triage rules, and owner action. Use `docs/agent-system/TOPICS.md` for the current topic registry. The default listing is 20 entries; set `--last` for a wider cohort. Prioritize entries by the owning taxonomy's severity and triage rules, then close resolved items through the owner command. |
| Agent Manager friction | `agent-manager usage friction --status open`; `agent-manager usage opportunities`; program `agent-manager.friction-digest` | Tool findings ranked by calls and runs affected. |
| Plan Manager logs | `plan-manager log list --type bug_report`; `plan-manager log list --type finding --triage candidate` | Many bug reports show sync failed: they never reached the inbox. |
| Swarm captures | `swarm-manager captures list` | Failed classifications are prey too. |
| Test Genie | `test-genie fleet status`; `test-genie runs findings --scenario <s>` | High failure rates are first-class leads. |
| Template debt | `template-manager debt list` | Inherited debt: fix the template (playbook §10). |
| Tidiness and quality | `tidiness-manager scenarios list`; `tidiness-manager issues list --scenario <s>`; `quality-health audit run <s>` | Leads only. |
| Doc health | `knowledge-observatory knowledge-base health --scenario-name <s> --skip-external-links` | Per scenario. |
| Readiness | `git-control-tower review summary <s>` | Code-quality violations per scenario. |
| Prior work | `search-hub query "<territory> <lens>" --type record,library --limit 5`; `vrooli-memory recall recall "<query>" --scope agent-memory --limit 10` | Build on prior fixes and no-gos instead of repeating them. |

Several CLIs rebuild themselves on first call and print build noise first. That is not an
error.

---

### 3. How Vrooli is meant to be

Judge a finding against the intended design, not against your taste.

| Question | Truth lives in |
|---|---|
| Platform shape, control plane, resources versus scenarios | `docs/concepts/ARCHITECTURE.md`, `docs/concepts/GLOSSARY.md` |
| What a well-formed scenario declares and contains | `docs/architecture/the-declared-scenario.md`, `docs/repo-contract.md` |
| Where a piece of guidance or state belongs | `docs/agent-system/LAYERS.md`; three-speed stack in `docs/agent-system/SKILL_AUTHORING.md` |
| How APIs, CLIs, storage, contracts and UIs should look | `api-steer`, `cli-steer`, `storage-steer`, `interoperability-steer`, `proto-contract-audit`, `ui-health`, `unit-testing-architecture-steer` |
| Migration and compatibility | `docs/repo-contract.md` §"Compatibility Policy" (greenfield, no compatibility fallbacks); `docs/agent-system/PROMOTION_LADDER.md` |
| Testing; dependencies | `docs/TESTING.md`; `docs/package-governance.md` |
| One scenario's intent and known gaps | `scenarios/<s>/PRD.md`, `README.md`, `docs/internal/` |
| Anything else | `search-hub query "<question>" --type doc --limit 5` |

When code and canon disagree, one of them is the defect. The newer operator-authored
decision wins, live callers beat stale prose, and a stated operator rule beats both. Fix the
loser in the same round. Do not leave a note that "they disagree".

When the intended design is missing and follows from existing canon, write it into the
owning doc, then implement it. When it requires choosing between product directions the
canon leaves open, that is a §7 decision.

---

### 4. What matters most

Tiers rank first. Within a tier, take the item that touches the most agents, runs or future
changes, and makes them most wrong.

| Tier | Prey | Why it ranks here |
|---|---|---|
| 1 | Defects in what every agent uses: `AGENTS.md`/`CLAUDE.md`, the `vrooli` control plane, Test Genie, Agent Manager, Prompt Manager and its most-read skills, Search Hub, Vrooli Memory, Plan Manager, `packages/` | One fix improves every future session |
| 2 | Silent wrong results: gates green for the wrong reason, swallowed errors, silent fallbacks, data loss, stale projections | Wrong answers that look right |
| 3 | Friction that recurs across runs | Measured, repeated cost |
| 4 | Two owners for one concept; unfinished migrations; legacy paths beside their replacement | Every future change is made twice or remembered |
| 5 | Implicit state machines in lifecycle code: runs, jobs, plans, claims, approvals, deployments | Hidden invalid states and recovery gaps |
| 6 | Canon, skill and prompt contradictions | Agents act on the wrong instruction |
| 7 | Local mess in one scenario | Real, but lowest reach |

Fix a tier-7 item when you are already in its file. Do not spend a round on tier 7 while a
tier 1–4 item is open.

---

### 5. Size the fix

Name the defect class before you edit. Then pick the row.

| Situation | Action |
|---|---|
| One instance, clear fix | Fix it, add the regression test, run the sibling search (§8 step 5). |
| Many instances across scenarios | Fix the mechanism that produces them (shared package, template, generator, skill, gate), then every instance. Delete the old path. |
| Large, and you estimate it fits in about 3 hours of work | Do it now. Large is the expected unit. |
| Your estimate is more than about 3 hours | State the estimate in the round record. Write the target design into the owning doc. Land a first slice that removes real debt. Make the remaining slices the next round's or segment's target, with exact next steps in the owner's ledger; under a scheduler, ask for a segment owner dedicated to it. A large item leaves the round partly done, never untouched, and is not dropped after its first slice. |
| Needs a §7 decision | Prepare options and a recommendation, route it, and keep hunting. |

Materiality is the defect removed: a class gone, an owner or path deleted, a state named and
its invalid transitions rejected, a silent failure made loud, a contradiction removed, or
complexity or runtime measurably cut. Net lines are a signal to report, never a veto
(`improvement-do-and-dont` D5). Do not predeclare numeric floors.

---

### 6. The shared tree and other agents

Typically four to six other agents edit this tree at once. That is the normal working
condition, not a reason to stand down.

Hard limits — operator rules, and the only limits in this section:
- No git command that writes (stash, checkout, commit, reset, worktree). Prefer filesystem
  and CLI evidence over git reads.
- No formatter on a directory. Pass the exact files you edited.
- Delete only explicitly named paths, one per `rm`. No globs, no `rm -rf` on unnamed trees.
- Never delete `PROBLEMS.md` or `PROGRESS.md`; resolve their entries (D2).
- Do not revert or overwrite another writer's change without reading it and integrating its
  intent.
- Manage scenarios through `vrooli scenario` / `make`; never run binaries directly.
- Route dependency changes through `scenario-dependency-analyzer`.
- Never hand-edit generated output; regenerate through the owner command.
- Host remediation belongs in the control plane (`internal/`). Do not revoke node
  privileges. Use elevation only in project setup.
- Before migrating stored data: snapshot it, dry-run the migration, and verify counts after.

Everything else is permitted:
- Edit files with uncommitted changes. Read the current content first, then integrate.
- Fix another agent's recent change when it is wrong, incomplete or adds debt.
- Take on files another session touched. If a writer is live on the same lines now (a
  running `agent-manager` run on that path, or the file changing under you), take another
  item, list the file under Deferred in the round record, and start the next round there.
- Restart a scenario you changed through the lifecycle. Run
  `cd packages/proto && make verify-committed-gen` first when protos are in flight. When you
  restart `agent-manager`, record the running run identities first and verify their recovery
  after (AGENTS.md §9).
- Expect transient build breaks from other owners. Validate your packages individually and
  record which failure belongs to whom.

---

### 7. Blocked, and what is not

Blocked means a decision, credential, approval or external action that only the operator
owns, with no safe alternative: spending money, publishing externally, deleting user data,
changing an explicitly human-only approval, or choosing between product directions the
canon leaves open.

Not blocked: a large refactor, a cross-scenario change, a dirty file, a failing unrelated
test, a stopped scenario, an unfamiliar codebase, an unclear owner (find it), a missing
design that follows from canon (write it), a broken tool (fix it or route around it).

For a real §7 decision: prepare the options and your recommendation, deliver it through the
owner path (a notification-hub decision, or the scenario's decision surface), and continue
with independent work.

---

### 8. The round

One round is one territory hunted through one lens. It ends when every tier 1–5 item found
in it is fixed, or is a §5 large item with its first slice landed, or is a §7 decision
routed.

```
wake ─▶ evidence picks territory × lens ─▶ hunt ─▶ rank (§4) ─▶ fix the class (§5)
  ▲                                                                   │
  │                                                                   ▼
record ◀─ validate ◀─ challenge the fix ◀─ sibling search and fan-out
```

1. **Wake.** Re-read §1 and the hunt's charter first, every round. Every hunt has a hunt ID: the one the goal names, or `hunt-<date of its first
   round>` (for example `hunt-2026-11-05`), kept for the whole goal. Run
   `vrooli-memory recall wake --scope ecosystem-hunt` for recent rounds, and
   `vrooli-memory recall recall "<candidate territory>" --scope ecosystem-hunt --limit 10`
   before choosing a territory. Start with any Deferred item from the last round. Read
   recent records for coverage and deferred items, never as the template for a round's
   size: the charter's exemplars set the expected unit.
2. **Choose.** Let the §2 evidence pick the highest-tier territory with open prey, and a
   lens (§9). Within the current hunt ID, do not repeat a territory × lens pair unless its
   files changed or new evidence points there. Rounds from earlier hunts are context —
   what was fixed, deferred or rejected and why — never a reason to skip a territory. A
   territory is one package, one scenario's API, one doc tree. File-level coverage is
   optional (§8a).
3. **Hunt.** Pull the territory's prey from §2. Trace at least one real behavior end to end
   — a command an agent runs, a request, a lifecycle — not only reading files. Read the
   intended design (§3).
4. **Fix.** Rank by §4. Size each fix by §5. Use `references/playbook.md`. Write the
   regression test that fails before and passes after.
5. **Sibling search and fan-out.** For every defect fixed, search the whole repository for
   the same pattern (`ast-grep`, `rg`) and fix the siblings, or fix the generator that made
   them. One finding should become a fleet sweep.
6. **Challenge the fix.** Did I fix the class or the instance? Did I add a layer where I
   could have deleted one? Is there now one owner? Would a hostile reviewer reading only the
   diff call this real (`improvement-do-and-dont` §2)? Get an independent reviewer pass when
   the change crosses owners, deletes an owner, or touches root instructions (`AGENTS.md`,
   `CLAUDE.md`) or a tier-1 skill — regardless of size. A round that ends with no change
   also gets a reviewer, briefed to argue for a change: what is the class here, what would
   the 10× fix be, and why is it not being done? Record its answer.
7. **Validate.** Run focused regressions and scoped Test Genie phases for every scenario you
   changed (`docs/TESTING.md`); widen to the consumers of a shared package you changed. Show
   the output.
8. **Record** (§10). Under a goal, end the turn after the record (`references/goal.md`), so
   the objective is re-injected before the next round.

Every round also answers one creative-lens question (§9, last row) for its territory and
records the answer. Every round lands a structural change: a class-level fix, a finished
migration, an explicit state machine, or a consolidation that deletes an owner. The only
exception is a round whose record shows the searches that came up empty and the step-6
reviewer's answer. A no-change round does not count toward a round count, and two in a row
send the next round to a large item (§5) or the next focus area. Do not shorten rounds to
meet a round count.

#### 8a. File coverage with visited-tracker (optional)

Most rounds trace behavior across files and need no file-level bookkeeping; the round
record is enough. Use `visited-tracker` only when it pays for itself:
- several hunters split one large territory and need exclusive claims, or
- the lens is a file-by-file sweep (for example doc truth across one doc tree) and you
  want to know which files are left.

Rules, from its current behavior:
- Tag: `eh-<hunt-id>-<territory-slug>`. A new hunt ID starts fresh coverage; reuse an
  older hunt's tag only when the goal says to continue that hunt. Review credit is not
  per lens, so add `-<lens>` to the tag for a second sweep lens on the same territory.
- Run `visited-tracker attention ensure-campaign --location "$(pwd)/<territory path>" --tag <tag> --patterns '<glob>' --max-files <files + 50%>`
  from the repository root. Repeat `--patterns` once per glob; a comma-separated value
  matches nothing. `--max-files` is fixed at creation (ceiling 10000) and cannot change
  later, so leave headroom.
- Run `visited-tracker attention preview --campaign-id <id> --limit 20` right after
  ensure. Zero candidates or a scan error means the campaign is unusable: path components
  named `data`, `build`, `tmp`, `out`, `dist`, `coverage` or `node_modules` anywhere in the
  absolute path are silently excluded, and a broken symlink aborts the scan. Do not debug
  it inside the hunt; skip file coverage for that territory and record the defect.
- An ensure "conflict" means the tag exists with other patterns or limits. Use a new tag.
- Claim with a fresh `--request-id` (a UUID) every time; a reused ID replays the old claim
  even when it has expired.
- Complete `--outcome reviewed --evidence <summary>` before you edit the file. Completing
  after an edit fails because the file's hash changed; the edited file then returns as
  unreviewed, which is correct.
- `coverage`, `least-visited` and the percentage in `campaigns list` count only legacy
  visits and ignore reviews. Read progress from `attention preview` instead.
- Once every file is reviewed, claims keep returning reviewed files. Stop the sweep when
  preview shows no unreviewed candidates, not when claims run out.

#### 8b. Synthesis round

Every tenth round, hunt the records instead of a territory. Read this hunt's last ten round
records and the open queue, and ask which of them are one class: the same missing mechanism,
the same owner confusion, the same absent primitive. Land the class-level fix, or write its
design into the owning doc and land the first slice (§5). Record which rounds it subsumes.

---

### 9. Lenses

A lens is a question you ask of every file in the territory. Each finds defects the others
miss. Detection commands and fix recipes are in `references/playbook.md`.

| Lens | The question |
|---|---|
| Implicit state machines | Which lifecycle is spread across booleans, nullable timestamps, status strings and process existence, with no named states and no transition owner? |
| Double residency | Which concept, decision, constant or doc assertion has two homes? |
| Unfinished migrations | Which old path still runs beside its replacement? |
| Silent failure | Where does an error become a default, a log line or a "green"? |
| Doc truth | Which doc says something the code does not do, says it twice, or describes the past? |
| Agent friction | Which command, help text or error made an agent guess, retry or fail? |
| Skills and prompts | Which skill, goal or prompt contradicts canon, restates another, or steers agents toward caution, ceremony or the wrong owner? |
| Test burden | Which tests pin implementation instead of behavior, hardcode cases a generator should make, or pass for the wrong reason? |
| Dead weight | Which code, flag, config, test or doc has no live caller or reader? |
| Hidden ordering | Which steps must happen in an order nothing enforces? |
| Performance | Which hot path does N+1 reads, repeated scans or redundant work? |
| What we would not build today | Knowing what we know now, which part of this territory would we not create? Which three things here are one thing? Which subsystem could disappear if its one real consumer used an existing capability? What change would make this category of bug impossible? |

---

### 10. Records

| What | Where |
|---|---|
| Round record | Write it to a scratch file, then `vrooli-memory journal note "$(cat <file>)" --scope ecosystem-hunt --kind task-record`. An inline double-quoted body executes any backticks in it and silently drops the command names. |
| File coverage, when used | `visited-tracker` per §8a |
| Defect in another owner's domain that you did not fix | That owner's `docs/internal/PROBLEMS.md`, or `prompt-manager skill read report-bug` |
| A technique that worked or a detection net that misfired | `references/playbook.md` |

Round record: hunt ID and round number; territory and lens; classes fixed; owners and paths deleted; changed files
and lines; source findings and the verdict written back to each; the creative-lens answer;
validation commands and results; the step-6 reviewer's answer for a no-change round;
deferred items, each with its §5 estimate, §6 live-writer note or §7 decision; next
territory. Do not create a report file or ledger per round.

---

### 11. Running with sub-agents

A hunt longer than about 6 hours, or one that covers several territories at once, uses the
scheduler shape in `references/goal.md`: a root that does no hunting, over segment owners of
about 10 rounds each, which spawn their own hunters and reviewers. This follows
`harness-goal-authoring` §5 F and G and `docs/agent-system/LONG_EFFORT_DESIGN.md`.

- Every brief is generated from the charter with `fork_turns=none`, carries the Stance
  verbatim, and says "You may edit any file within ecosystem-hunt §6; this overrides a
  narrower file bound in your agent definition."
- Briefs say spawning is expected; a segment owner that hunts alone serializes the hunt.
- Hunters own a territory for the whole round, often more than an hour. Do not split a round
  into minute-long workers.
- Two hunters never hold the same territory, unless a segment owner splits a large one
  through §8a claims. When a hunter's sibling search lands in another hunter's active
  territory, it sends the hit to that hunter; otherwise it fixes it.
- Use the harness's real roles (`references/goal.md` §"Harness vocabulary").
- Count concurrency slots: every waiting ancestor holds one.

For a multi-day delivery-shaped effort, use `large-effort-orchestration` instead.

---

### 12. Output expectations

You may change anything in scope within the §6 hard limits, including deleting dead,
duplicated or superseded code, docs and paths, each with a retirement manifest
(`improvement-do-and-dont` D7).

You must:
- fix defect classes at their mechanism, and run the sibling search for each
- leave one owner for each concept you touch
- write or update the regression test for each behavior fix
- show the validation output for every changed scenario
- write the round record after every round
- end any "nothing material found" round by naming the sources, lens and searches that came
  up empty

### Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| Smallest-edit syndrome | The mechanism survives; the bug returns elsewhere | Name the class; fix the mechanism and the siblings |
| Screen-and-reject | Predeclared floors select for "nothing to do" | Materiality is the defect removed (§5) |
| Dirty-file avoidance | The worst debt sits where agents are working | Read, integrate, fix (§6) |
| File-and-forget | A big item becomes a ledger line nobody pays | Design, first slice, exact next steps |
| Redundant confirmation | Adds friction when the current operator request already authorizes the needed change | Check the request and owner rules once; ask only for a §7 decision |
| Re-hunting home ground | Same three scenarios, diminishing returns | Wake first; evidence picks the territory |
| Bold-by-accretion | New layers, shims and fences grow the debt | Delete an owner; simplify the path |
| Hedged closeout | Hides what was and was not done | State changed, verified, remaining, unverified |
| Stopping at green | Uncovered states stay broken | Green starts the challenge step |
| Latest round as template | Each smaller round sets the size of the next | Charter exemplars set the unit; recent records are for coverage |
| One finding per round, forever | The queue never drains and the class survives | Group findings into classes (§2); synthesis round (§8b); write verdicts back |
| One long turn | The objective is omitted after compaction and rounds shrink | End the turn after each round's record |

### Troubleshooting & Edge Cases

| Symptom | Response |
|---|---|
| `vrooli-memory` or `visited-tracker` is down | Start it with `vrooli scenario start <name>`. If it stays down, keep the round records in the session and journal them when it returns. Keep hunting. |
| `attention preview` says the pattern matches more files than the campaign limit | The limit is fixed at creation. Use a new tag with a larger `--max-files`, or narrow `--location` or `--patterns`. |
| A visited-tracker command fails in a way §8a does not cover | Skip file coverage for this territory, record the failure in the round record, and keep hunting. visited-tracker is advisory. |
| A shared build breaks from another owner's edit | Check for a generated-artifact freshness failure first (AGENTS.md §3). Fix it if understood; otherwise validate your packages individually and record the foreign failure. |
| A Test Genie run is long | Wait once with `test-genie runs wait --json <scenario> <run-id>`; hunt another territory read-only meanwhile. |
| Every candidate in a territory looks fine | Trace one real agent task through it and ask the creative-lens questions. If both find nothing, record the searches and move on. |
