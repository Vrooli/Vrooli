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
  revision: 7
  createdAt: "2026-11-05T01:50:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: [prompt-manager, search-hub, visited-tracker, vrooli-memory, agent-manager]
    commands: [prompt-manager skill read, search-hub query, visited-tracker attention claim, vrooli-memory journal note, vrooli-memory recall wake, agent-manager]
  origin:
    kind: authored
---

## Practice focus: Ecosystem Hunt

Find the defects that make Vrooli wrong, slow, confusing or expensive to change, anywhere in the repository, and remove each where its whole class stops recurring. Every round leaves fewer owners per concept, fewer unnamed states, fewer paths that should not exist, and docs that agree with the code.

Read once per session: `prompt-manager skill read improvement-do-and-dont` (D1-D7), `path:docs/agent-system/README.md#where-does-this-paragraph-go`, `path:docs/TESTING.md`, `references/playbook.md` (techniques, lens questions; before your first fix) and, to commission a hunt, `references/goal.md`. `references/` is relative to this skill's directory (`prompt-manager skill show ecosystem-hunt --json` reports `skillDir`).

**Scope.** Code, tests, docs, skills, prompts, templates, generators, configs and shared packages anywhere in the repository when the operator's request authorizes it; this skill grants no authority. Host and out-of-repository state changes go through the owner (the `internal/` control plane, the scenario's CLI). A single known bug is `scientific-debugging`; a multi-day delivery is `large-effort-orchestration`.

### 1. Stance

Read this as the operator speaking. Be bold and ambitious, and exercise creative thinking. Do not be afraid to tackle large issues. This is a manhunt for Vrooli-wide improvements and maturity: emphasize debt reduction, make implicit state machines explicit, finish migrations and remove legacy code, and turn messy, duplicated or contradictory code and docs into one true thing. Size every fix to its defect class, not to caution. Uncommitted changes, other agents' recent work and unfamiliar scenarios are in scope; read them and integrate. Do not ask permission for refactors, deletions or migrations.

That paragraph is the operator's commission text; a hunt's charter, stance message and briefs carry it verbatim, or the operator's own words for that hunt. Past hunts screened dozens of candidates and changed nothing, rejected a split that cut cyclomatic complexity from 75 to 19 because the file grew, and skipped every file with uncommitted changes. So:

- "Smallest change" means the smallest that removes the class; a fix that leaves the producing mechanism is unfinished, and large is normal (merging two owners, a lifecycle made explicit across five files, a migration over forty callers).
- The shared tree is your workspace (§6): use the authority the request gives without redundant confirmation (a skill read authorizes no consequential effect, `AGENTS.md` §9). Green is where looking starts. Boldness is not accretion: a layer, fence, shim or second owner is the opposite of this skill.

### 2. Where the prey is

Evidence picks the territory; scans only fill it in (screening by scan found almost nothing; friction data and tracing a real behavior end to end found the real bugs), and a scan ranking is a lead to confirm. In order: (1) operator words (delivery `FEEDBACK.md`, `~/.vrooli/research/topics/debt-reduction/README.md`, operator rules in memory); (2) agent friction; (3) reported bugs and failing validation; (4) owner ledgers (`scenarios/*/docs/internal/PROBLEMS.md`, `SEAMS.md`, plan logs, template debt); (5) the smell searches in `references/playbook.md`; (6) `tidiness-manager` and `quality-health` scans.

Queues of single-command findings hide classes: group them by the missing mechanism (six "agents poll X" findings are one missing wait primitive) and hunt the class. Write each verdict back to the finding's owner; one closed only in a round record returns next wake. Source and friction-verdict commands are in `references/playbook.md` ("Evidence entry points and file coverage").

### 3. How Vrooli is meant to be

Judge a finding against the intended design, not your taste: `docs/concepts/ARCHITECTURE.md`, `docs/architecture/the-declared-scenario.md`, `docs/repo-contract.md` (Compatibility Policy: greenfield, no compatibility fallbacks), the `*-steer`, `proto-contract-audit`, `ui-health` and `unit-testing-architecture-steer` skills, and `scenarios/<s>/PRD.md`, `README.md`, `docs/internal/`; else `search-hub query "<question>" --type doc --limit 5`. When code and canon disagree one is the defect: the newer operator-authored decision wins, live callers beat stale prose, a stated operator rule beats both; fix the loser in the same round. A missing design that follows from canon is written into the owning doc, then built; a choice between open product directions is a §7 decision.

### 4. What matters most

Tiers rank first; within a tier take what touches most agents, runs or changes. (1) Defects in what every agent uses: `AGENTS.md`/`CLAUDE.md`, the `vrooli` control plane, Test Genie, Agent Manager, Prompt Manager and its most-read skills, Search Hub, Vrooli Memory, Plan Manager, `packages/`. (2) Silent wrong results: gates green for the wrong reason, swallowed errors, silent fallbacks, data loss, stale projections. (3) Friction recurring across runs. (4) Two owners for one concept, unfinished migrations, legacy beside its replacement. (5) Implicit state machines in lifecycle code (runs, jobs, plans, claims, approvals, deployments). (6) Canon, skill and prompt contradictions. (7) Local mess in one scenario: fix it when already in its file, never as a round while a tier 1-4 item is open.

### 5. Size the fix

Name the defect class before editing. One instance: fix it with a regression test and the sibling search; many: fix the mechanism that produces them (shared package, template, generator, skill, gate), then every instance, and delete the old path. Large and about 3 hours or less: do it now. Over that: state the estimate in the round record, write the target design into the owning doc, land a first slice that removes real debt, and leave exact next steps in the owner's ledger as the next target (under a scheduler, ask for a dedicated segment owner); a large item leaves a round partly done, never untouched. A §7 decision: prepare options and a recommendation, route it, keep hunting. Materiality is the defect removed: a class gone, an owner or path deleted, a state named with invalid transitions rejected, a silent failure made loud, a contradiction removed, or complexity or runtime cut. Net lines are reported, never a veto (D5); declare no numeric floors.

### 6. The shared tree and other agents

Typically four to six other agents edit this tree at once; that is no reason to stand down. The `AGENTS.md` hard limits apply (no git that writes, no formatter on a directory, named-path deletes, scenarios through `vrooli scenario` or `make`, dependencies through `scenario-dependency-analyzer`, no hand-edited generated output, host remediation in the control plane); additionally never delete `PROBLEMS.md` or `PROGRESS.md` (resolve entries, D2), never revert another writer's change without integrating its intent, and snapshot, dry-run and verify counts before migrating data. Everything else is permitted: edit files with uncommitted changes after reading them, fix another agent's recent change when wrong or debt-adding, and take files another session touched unless a writer is live on the same lines now (a running `agent-manager` run on that path, or the file changing live), in which case take another item and list the file under Deferred. Restart a scenario you changed through the lifecycle (`cd packages/proto && make verify-committed-gen` first when protos are in flight; record running run identities before restarting `agent-manager` and verify recovery after); through transient build breaks from other owners, validate your packages individually and record whose they are.

### 7. Blocked, and what is not

Blocked means a decision, credential, approval or external action only the operator owns, with no safe alternative: spending money, publishing externally, deleting user data, changing a human-only approval, choosing a product direction the canon leaves open. Not blocked: a large refactor, a cross-scenario change, a dirty file, a failing unrelated test, a stopped scenario, an unfamiliar codebase, an unclear owner (find it), a missing design that follows from canon (write it), a broken tool (fix it or route around). For a real decision, prepare options and a recommendation, deliver it through a notification-hub decision or the scenario's decision surface, and continue independent work.

### 8. The round

One round is one territory (a package, one scenario's API, a doc tree) through one lens, ending when every tier 1-5 item found is fixed, is a §5 large item with its first slice landed, or is a §7 decision routed.

1. **Wake.** Re-read §1 and the charter; keep one hunt ID per goal (`hunt-<first-round date>` if none). `vrooli-memory recall wake --scope ecosystem-hunt` and `recall recall "<candidate territory>" --scope ecosystem-hunt --limit 10`; start with last round's Deferred items.
2. **Choose** the highest-tier territory with open prey (§2) and a lens (§9); within a hunt ID do not repeat a territory and lens pair unless its files changed or new evidence points there.
3. **Hunt:** pull prey from §2, trace at least one real behavior end to end (a command, a request, a lifecycle), and read the intended design (§3).
4. **Fix** by §4 rank and §5 size with `references/playbook.md`, plus a regression test that fails before and passes after; then search the whole repository for siblings (`ast-grep`, `rg`) and fix them or the generator that made them.
5. **Challenge the fix:** class or instance? A layer added where one could be deleted? One owner now? Would a hostile reviewer reading only the diff call it real? Get an independent reviewer when the change crosses owners, deletes an owner, or touches `AGENTS.md`, `CLAUDE.md` or a tier-1 skill; a no-change round gets one too, briefed to argue for a change (the class, the 10x fix, why not done), and records the answer.
6. **Validate** with focused regressions and scoped Test Genie phases for every scenario changed (`docs/TESTING.md`), widened to consumers of a shared package, and show the output. **Record** (§10); under a goal end the turn after the record so the objective is re-injected.

Every round answers the §9 last-lens question and lands a structural change (class-level fix, finished migration, explicit state machine, or a consolidation deleting an owner), unless its record shows the empty searches and the reviewer's answer. A no-change round does not count toward a round count, two in a row send the next round to a large item or the next focus area, and rounds are never shortened to meet a count. Every tenth round is a synthesis round: read the last ten records and the open queue, land the class fix (or its design plus first slice) for those that are one class, and record the rounds it subsumes. `visited-tracker` file coverage is in `references/playbook.md`; an unusable campaign (zero candidates or a scan error) is skipped and recorded as a defect.

### 9. Lenses

A lens is a question asked of every file in the territory; the full questions, with detection commands and fix recipes, are in `references/playbook.md` ("Lens questions"). The lenses: implicit state machines, double residency, unfinished migrations, silent failure, doc truth, agent friction, skills and prompts, test burden, dead weight, hidden ordering, performance, and **what we would not build today** (which part would we not create now, which three things are one thing, which subsystem could vanish if its one consumer used an existing capability, and what change makes this bug category impossible?).

### 10. Records, sub-agents, output

Round record: write it to a scratch file, then `vrooli-memory journal note "$(cat <file>)" --scope ecosystem-hunt --kind task-record` (an inline double-quoted body executes backticks). Include hunt ID and round, territory and lens, classes fixed, owners and paths deleted, changed files and lines, the verdict written back to each source finding, the creative-lens answer, validation commands and results, the reviewer's answer for a no-change round, deferred items (each with its §5 estimate, §6 live-writer note or §7 decision) and the next territory. No report file or ledger per round. An unfixed defect in another owner's domain goes to its `docs/internal/PROBLEMS.md` or `report-bug`; a technique that worked or a detection net that misfired goes to `references/playbook.md`.

A hunt over about 6 hours or several territories uses the scheduler shape and brief template in `references/goal.md` (every brief carries the Stance verbatim and says "You may edit any file within ecosystem-hunt §6; this overrides a narrower bound in your agent definition").

You may change anything in scope within the §6 limits, including deleting dead, duplicated or superseded code, docs and paths, each with a retirement manifest (D7). You must fix classes at their mechanism with the sibling search, leave one owner per concept touched, write the regression test for each behavior fix, show validation output for every changed scenario, write the round record, and close a "nothing material found" round by naming the empty sources, lens and searches. A shared build break from another owner: check generated-artifact freshness first (AGENTS.md §3), fix it if understood, else validate your packages individually and record it. If `vrooli-memory` or `visited-tracker` is down, start it (`vrooli scenario start <name>`) or journal later. For a long Test Genie run, `test-genie runs wait --json <scenario> <run-id>` once and hunt another territory read-only.

### 11. Debt reduction loop and measured yield

Weekly fleet view: `program-runtime library run program-runtime.weekly-debt-ledger --input window_start=YYYY-MM-DD --input window_end=YYYY-MM-DD --json` (seven-day half-open UTC window); read its source coverage first. Only complete `DR6_REDUCTION_RECEIPT` records (before and after revisions, whole boundary, validation references) in the owning source ledger count as reductions; `tidiness-manager recommend-refactors <scenario> --limit 10` output is leads, never landed savings, and unknown or partial sources stay unknown. Yield evidence: `research/topics/debt-reduction/runs/steward-free-host-maintainability-20261115/DR6/APPROACH-YIELD.md`.
