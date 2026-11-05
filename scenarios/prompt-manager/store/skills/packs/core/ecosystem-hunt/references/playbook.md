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

## 2. One owner per concept (double residency)

**Signals**: the same enum, regex, constant, port, path, schema or decision appears in two or
more places; two CLI commands or two endpoints do the same job; two docs state the same rule;
a client re-implements a server decision.

**Detect**
```bash
rg -n -F '<distinctive literal>' scenarios packages internal docs   # every home of one value
rg -N --no-filename -g '*.md' -g '!**/evidence/**' '^#{2,3} \S' docs scenarios/*/docs | sort | uniq -c | sort -rn | head -30   # repeated headings: boilerplate or copied rules
tidiness-manager scan <scenario> --type tidiness                     # clone signals, as leads
```

**Fix**: choose the owner with `docs/agent-system/LAYERS.md` (truth in Plan of Record,
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

For a contradictory doc set: pick the one home per `docs/agent-system/LAYERS.md`, merge the
true content into it, turn the others into one-line links or delete them, and fix inbound
links. `knowledge-observatory-maintenance` owns the move and retirement mechanics.

## 6. Fix agent friction at the owner

**Signals**: agents run `--help` before a command, guess flags, retry with different syntax,
or misread output. Friction digests and Agent Manager command-failure measures show these.

**Fix**: change the owner, not the skill prose. Make the error message name the correct
invocation. Accept the obvious spelling. Make the default output answer the common question.
When the same clarification appears in several skills, move it into the CLI and delete it
from the skills (`docs/agent-system/PROMOTION_LADDER.md`). Verify by running the bad
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
`docs/agent-system/DEPRECATION_POLICY.md`.

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

## 10. Fix the generator, not the copies

**Signals**: the same defect in many scenarios with the same file layout; code under a
template's tree; generated shells or sidecars.

**Fix**: find the template, generator or shared package (`template-manager`,
`packages/`, `cli-core`, `api-core`). Fix it there, then regenerate or update every copy
through the owner command. Never hand-edit generated output (AGENTS.md §3).

---

## Related skills for depth

`domain-clarity` (concept compression), `cognitive-load-reduction`, `seam-discovery-and-enforcement`,
`invariant-discovery-and-enforcement`, `temporal-flow-audit`, `idempotency-replay-safety-hardening`,
`error-semantics-recovery-path-design`, `storage-steer`, `scientific-debugging`,
`documentation-health`, `knowledge-observatory-maintenance`. Load one when its lens is the
round's lens; do not load them all.
