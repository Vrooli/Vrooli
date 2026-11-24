---
name: "cognitive-load-reduction"
description: "Simplify code to reduce mental overhead for developers"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","simplification"]
  tags: ["skill","audit-technique"]
  icon: "brain"
  status: "active"
  revision: 43
  createdAt: "2025-01-15T00:00:00Z"
  updatedAt: "2026-02-04T13:13:54Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Steer focus: Cognitive Load Reduction

Prioritize **making the code-craft layer of `scenarios/{{TARGET}}/` match the template's canonical shape**: file layout, function shape, naming, comment discipline, and local state flow. This skill governs the local readability that remains after structural audits have placed code correctly. The destination is "this scenario's code reads like the template's code," verified through `tidiness-manager` and routed to existing docs.

Required reading (architectural context — set the stage before this skill applies):
- `prompt-manager skill read architecture-lenses` — folder/domain shape and product vocabulary.
- `prompt-manager skill read resilience-lenses` — lifecycle and state-machine ownership.
- `prompt-manager skill read architecture-lenses` — when indirection is justified.
- `prompt-manager skill read architecture-lenses` — when named decision helpers earn their indirection cost.
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Required reading (programmatic enforcement substrate):
- `prompt-manager skill read tidiness` (or `tidiness-manager --help`) — the scenario that runs light + smart scans and is the primary destination as this skill becomes more programmatic over time.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — surfaces, domain map, intended structure, any scenario-local vocabulary discipline.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — boundary registry; the source of truth for which indirections are justified.
- `scenarios/{{TARGET}}/docs/internal/INVARIANTS.md` — state ownership and invariants.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — accepted code-craft debt and complexity hotspots.

Optional context:
- `docs/scenario-qa/methods/audit/cognitive-load-reduction.md` — when this lens applies, when it backfires, what the qa-contrarian challenges.

---

### 1. Scope Boundaries

This skill is **post-architectural**. Structural problems are handed off to the audit that owns them; cog-load only owns what remains after the structural audits have placed code correctly.

**In scope:**
- file shape inside a domain folder (canonical ordering of types, constructors, methods, helpers)
- function shape (length, nesting depth, complexity, guard-clauses-first, happy-path-top ordering)
- local naming (variables, parameters, function names) in domain vocabulary
- comment discipline (why-not-what; removing comments that restate code)
- local state flow inside a function (predictable inputs/outputs, no surprising mutations)
- routing findings to existing docs (ARCHITECTURE, SEAMS, INVARIANTS, PROBLEMS) by lens
- proposing tidiness-manager check enhancements when a recurring manual finding could become programmatic

**Out of scope (hand off):**
- folder organized by tech bucket instead of domain → `architecture-lenses`
- temporal/lifecycle complexity (retries, cancellation, state machines) → `resilience-lenses`
- whether an abstraction earns its keep across domains → `architecture-lenses`
- whether a named decision helper earns its indirection cost → `architecture-lenses`
- public API surface, transport, contracts → `api-steer`
- dead code, stale config, abandoned files → `code-cleanup`
- new product features, behavior changes, performance refactors
- creating standalone `*_AUDIT.md` or dedicated `COGNITIVE-LOAD.md` files — findings route to existing docs

If the structural audits have not yet been run on this scenario, note that as a prerequisite gap rather than fixing it inside this skill.

---

### 2. Cognitive Load Maturity Model

Assess each surface (`api/`, `ui/`, `cli/`) independently. A scenario may be Level 3 in API code-craft and Level 1 in UI code-craft.

| Level | Name | What exists | Where it's verified |
|---|---|---|---|
| 0 | Untouched | No cog-load lens applied; code-craft hotspots live only in agent memory. | — |
| 1 | Routed | Findings are filed into the correct existing doc by lens (ARCHITECTURE / SEAMS / INVARIANTS / PROBLEMS); structural findings are handed off to the audit that owns them, not patched here. | `grep` those docs for entries dated this pass. |
| 2 | Template-shape match — naming and layout | Banned-name check clean (no `manager`/`helper`/`util`/`common`/`data`/`process`/`handle`/`do`/`misc`/`base`/`core` as bare suffixes/prefixes); folders, files, and exported names use domain vocabulary; per-file ordering matches the template's canonical order. | `tidiness-manager scan {{TARGET}} --type light` returns no naming or layout violations. |
| 3 | Template-shape match — function shape | Function length, nesting depth, and complexity within template budgets (`funlen ≤ 60`, `gocognit ≤ 15`, `nestif ≤ 4`, `gocyclo ≤ 12`; UI `complexity ≤ 10`, `max-lines-per-function ≤ 60`, `max-depth ≤ 4`); guard-clauses-first + happy-path-top ordering applied; restateful comments removed. Exceptions are recorded against `PROBLEMS.md` with rationale. | `tidiness-manager scan {{TARGET}} --type light` clean on these checks; `--type smart` raises no hotspots in these dimensions. |
| 4 | Template-shape match — state and indirection | No ambient/package-level mutable state outside `config/`; service dependencies passed explicitly via constructors; every retained abstraction has a justified entry in `SEAMS.md` or has been collapsed. | `tidiness-manager scan {{TARGET}} --type smart` raises no ambient-state or unjustified-indirection hotspots; `INVARIANTS.md` and `SEAMS.md` align with the code. |
| 5 | Continuous tidiness | An ongoing tidiness-manager campaign covers this scenario; new code is gated on the same checks; recurring manual findings have been promoted to tidiness-manager checks. | `tidiness-manager campaigns` shows an active campaign for `{{TARGET}}` and recent runs are green. |

Do not treat the level as a score to inflate. Use it to identify the next concrete move: route findings, then run the next tidiness-manager scan, then close the highest-impact hotspots.

---

### 3. Finding routing (no new docs)

Never create a standalone cog-load report; route each finding to an existing doc. Structural findings stop here and are handed off, never patched locally. Tech-bucket folders and `*_helper.go`, `*_utils.go`, `common/`, `manager/` names go to `ARCHITECTURE.md` (vocabulary section) and `architecture-lenses`. A one-call-site wrapper with no domain meaning, and a named-decision helper challenged on indirection, go to `SEAMS.md` (`architecture-lenses` §2, §6). Owned here: over-budget functions and buried happy paths go to `PROBLEMS.md` ("Complexity hotspots", "Control-flow hotspots") with a `path:` link; ambient package-level state, hidden mutation and `init()` magic go to `INVARIANTS.md` (ownership) plus `PROBLEMS.md` until fixed; restateful comments and ambiguous local names (`x`, `tmp`, `data`, `obj`) are fixed in code with no doc entry; a recurring manual finding that could be a check becomes a tidiness-manager backlog entry.

---

### 4. The code-craft dimensions

For each, the tidiness-manager light scan (`golangci-lint` and `eslint` configs) catches what is marked; the rest is human or smart-scan reading.

- **File shape.** Divergence: `helpers.go`, `utils.go`, `common.go`, `base.go`, files mixing unrelated types, ordering by edit history. Go order: types, constants, constructors, exported methods (dependency order, mutators before readers), unexported helpers (only when reused twice in the file). TypeScript: types, constants and hooks, primary export, sub-helpers. Name files after the domain noun. Banned file names are checked; within-file ordering is not.
- **Function shape.** Divergence: over 60 lines, nesting over 4, happy path indented deeper than guards, mixed responsibilities, long `if/else` chains where a switch or table is clearer. Order: guard clauses first, fetch dependencies (wrap errors with the operation name), happy path unindented, one return. Budgets (checked): Go `funlen <= 60`, `gocognit <= 15`, `nestif <= 4`, `gocyclo <= 12`; TS `complexity <= 10`, `max-lines-per-function <= 60`, `max-depth <= 4`. Genuine exceptions (orchestrating handlers, generated translation tables, exhaustive enum switches) carry `//nolint:<rule> // <reason>` and a `PROBLEMS.md` entry under "Accepted code-craft exceptions".
- **Naming.** Banned bare prefixes and suffixes (`manager`, `helper`, `util`, `common`, `data`, `process`, `handle`, `do`, `misc`, `base`, `core`; list lives in tidiness-manager, with scenario extensions in an `ARCHITECTURE.md` "Vocabulary discipline" subsection). Names state intent and domain, not mechanics (`selectEligibleNotes`, not `filterList`) so a reader can predict behavior without opening the thing. Whether a verb like `process` is real domain vocabulary is a human or smart-scan call.
- **Comments.** Answer why, not what. Remove comments that restate code, name the obvious from the signature, point at callers or describe removed branches, and commented-out code. Keep non-obvious decisions, hidden constraints, invariant references, bug workarounds and surprises. Test: would removing it confuse a reader who has the code in front of them? Do not delete to cut density; twenty earned comments beat two restateful ones. Detection is a smart-scan candidate.
- **Local state flow.** Divergence: package-level `var` outside `config/`, `init()` registering globals, surprising receiver mutation, hidden ambient inputs. Output is determined by parameters and constructor-passed dependencies; mutate at named seams (`s.repo.Save(...)`), never `s.cache[k] = v` mid-flow. Lint partly covers package-level mutables; hidden dependencies stay manual.
- **Scattering and co-location.** If deleting the capability should delete the code, it belongs in that domain's folder; single-domain helpers stay local, helpers for two or more unrelated domains move to `internal/<substrate>/` or `lib/` with a seam entry. Cross-domain scattering is a screaming-architecture finding; cog-load owns a single domain fragmented across files for no reason (five 30-line files where two 80-line files read better).

---

### 5. Working with `tidiness-manager`

`tidiness-manager scan {{TARGET}} --type light` is the fast static pass (L2 and L3 verification); `--type smart` is the AI-assisted pass for L3 and L4 hotspots lint cannot catch; `issues {{TARGET}}` lists tracked issues; `campaigns` starts or checks a campaign (`tidiness-manager --help` has the current surface). Trust scan output for what is enforced, and record a manual fix that could be programmatic (for example removing restateful comments by reading) as an enhancement candidate so recurring findings migrate into checks. If tidiness-manager is unavailable, fall back to `golangci-lint run` and `npm run lint` and note the gap in `PROBLEMS.md`.

### 6. Safe refactoring

You may split long functions into units that each name a real domain operation, collapse wrappers that add no domain meaning, reorder code within a file, improve names and comments, and adjust tests to match clarified behavior without weakening coverage. You must preserve observable behavior, keep public interfaces stable where feasible (updating all call sites if not), never delete or loosen meaningful tests, prefer incremental steps over large rewrites, and never reverse an `architecture-lenses` outcome on local-readability grounds alone. Substantial redesign is recorded in `PROBLEMS.md`, not half-done here.

### 7. Audit workflow

Confirm prerequisites (skim `ARCHITECTURE.md`, `SEAMS.md`, `INVARIANTS.md`; if the structural audits were not run and the scenario is visibly fragmented, hand off); run the light scan and fix L2 and L3 violations first; read smart-scan results for L4; walk the §4 dimensions, function shape and naming first; route each finding through §3; note tidiness-manager enhancement candidates; update docs via `knowledge-observatory-tools`; re-run the scan and confirm the §2 level genuinely moved.

---

### **8. Output Expectations**

By the end of this loop, the scenario should:
- have advanced one level on the §2 ladder for at least one surface (`api/`, `ui/`, or `cli/`)
- have its cog-load findings filed into existing docs by lens — not collected in a private cog-load file
- have visibly tighter function shape and clearer local naming where they were divergent
- have any structural findings handed off to the audit that owns them, not papered over locally
- leave at least one concrete enhancement candidate for tidiness-manager when manual work could become programmatic

Avoid:
- creating standalone cog-load report files or new docs under `docs/internal/`
- bulk renames or reformatting that don't measurably reduce divergence from template shape
- deleting comments that earn their place (the goal is signal-per-comment, not silence)
- reversing prior audit outcomes (`architecture-lenses`) on local-readability grounds without engaging their reasoning
- claiming maturity advancement that the corresponding tidiness-manager scan cannot verify
