## Steer focus: Cognitive Load Reduction

Prioritize **making the code-craft layer of `scenarios/{{TARGET}}/` match the template's canonical shape**: file layout, function shape, naming, comment discipline, and local state flow. This skill governs the local readability that remains after structural audits have placed code correctly. The destination is "this scenario's code reads like the template's code," verified through `tidiness-manager` and routed to existing docs.

Required reading (architectural context — set the stage before this skill applies):
- `prompt-manager skill read screaming-architecture-audit` — folder/domain shape and product vocabulary.
- `prompt-manager skill read temporal-flow-audit` — lifecycle and state-machine ownership.
- `prompt-manager skill read seam-discovery-and-enforcement` — when indirection is justified.
- `prompt-manager skill read decision-boundary-extraction` — when named decision helpers earn their indirection cost.
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Required reading (programmatic enforcement substrate):
- `prompt-manager skill read tidiness-manager` (or `tidiness-manager --help`) — the scenario that runs light + smart scans and is the primary destination as this skill becomes more programmatic over time.

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
- folder organized by tech bucket instead of domain → `screaming-architecture-audit`
- temporal/lifecycle complexity (retries, cancellation, state machines) → `temporal-flow-audit`
- whether an abstraction earns its keep across domains → `seam-discovery-and-enforcement`
- whether a named decision helper earns its indirection cost → `decision-boundary-extraction`
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

### 3. Finding-Routing Table (No New Docs)

Every cog-load finding belongs to one of the existing audit docs. Never create a standalone cog-load report or a new file for these findings.

| Finding | Routes to | Lens that ultimately owns it |
|---|---|---|
| Folder organized by tech bucket instead of domain | `ARCHITECTURE.md` | `screaming-architecture-audit` (hand off) |
| File named `*_helper.go`, `*_utils.go`, `common/`, `manager/` | `ARCHITECTURE.md` vocabulary section | `screaming-architecture-audit` (hand off) |
| Function exceeds length/complexity/nesting budgets | `PROBLEMS.md` "Complexity hotspots" with `path:` link | Cog-load (this skill) |
| Buried happy path inside error pyramid | `PROBLEMS.md` "Control-flow hotspots" | Cog-load (this skill) |
| Wrapper with one call site that adds no domain meaning | `SEAMS.md` (justify or collapse) | `seam-discovery-and-enforcement` (hand off) |
| Named-decision helper challenged on indirection grounds | `SEAMS.md` cross-reference | `decision-boundary-extraction` (hand off) |
| Ambient package-level state, hidden mutation, `init()` magic | `INVARIANTS.md` (ownership) + `PROBLEMS.md` if not yet fixed | Cog-load (this skill) |
| Restateful comments (`// loops through items`) | Fix in code; no doc entry | Cog-load (this skill) |
| Ambiguous local variable names (`x`, `tmp`, `data`, `obj`) | Fix in code; no doc entry | Cog-load (this skill) |
| Recurring manual finding that could be a programmatic check | Open a tidiness-manager backlog entry | Tidiness-manager (capability promotion) |

Rule: if the finding is structural, **stop here and hand off** to the audit that owns it. Do not paper over a screaming-architecture or seam-discipline problem with a local cog-load patch.

---

### 4. The Code-Craft Dimensions

Each dimension below is a property of the code's local readability. For every dimension, this section names: signs of divergence from template shape, the rule, and the current programmatic check (if any) so you know what `tidiness-manager` already catches versus what still needs human reading.

#### 4.1 File shape

**Signs of divergence:** files named `helpers.go`, `utils.go`, `common.go`, `base.go`; files mixing unrelated types; files where ordering is by edit history rather than dependency.

**Rule:** Within a Go file, the canonical order is: types → constants → constructors → exported methods (in dependency order, mutators before readers) → unexported helpers (only when reused 2+ times in this file). Within a TypeScript module, the order is: types → constants/hooks → primary export → sub-helpers. Files are named after the domain noun, not after the mechanic.

**Programmatic check:** banned-file-name check in tidiness-manager light scan. Within-file ordering is currently human-read; flag as a tidiness-manager enhancement candidate when you fix it manually.

#### 4.2 Function shape

**Signs of divergence:** functions over 60 lines; nesting deeper than 4 levels; happy path indented deeper than guard clauses; multiple unrelated responsibilities in one body; long sequential `if/else` chains where a switch or table lookup is clearer.

**Rule:** Function bodies follow this order:
1. **Guard clauses first** — return early on invalid input or impossible state.
2. **Fetch dependencies** — pull what you need; wrap errors with operation name.
3. **Happy path** — the longest, most prominent branch, unindented.
4. **Return** — single return type, single value-construction site where possible.

Budgets (enforced by tidiness-manager light scan):
- Go: `funlen ≤ 60`, `gocognit ≤ 15`, `nestif ≤ 4`, `gocyclo ≤ 12`.
- TS/UI: `complexity ≤ 10`, `max-lines-per-function ≤ 60`, `max-depth ≤ 4`.

Exceptions are allowed when the natural shape genuinely exceeds a budget (typically: handlers that orchestrate many distinct steps, generated-code translation tables, exhaustive switches over a domain enum). Document each exception with `//nolint:<rule> // <one-sentence reason>` and add an entry to `PROBLEMS.md` under "Accepted code-craft exceptions" with the rationale.

**Programmatic check:** budgets enforced by tidiness-manager light scan (golangci-lint + eslint configs). Guard-clause-first and happy-path-top ordering are currently human-read; smart-scan candidate.

#### 4.3 Naming and intent signaling

**Signs of divergence:** `manager`, `helper`, `util`, `utils`, `common`, `data`, `process`, `handle`, `do`, `misc`, `base`, `core` as bare suffixes/prefixes; local variables named `x`, `tmp`, `obj`, `result`, `val` when a domain noun is available; function names that describe mechanics (`filterList`, `processItems`) instead of intent (`selectEligibleNotes`, `markNotesArchived`); abbreviations that aren't industry-standard.

**Rule:** Names communicate **intent and domain**, not mechanics. The banned-name list lives in `tidiness-manager` as the programmatic default; scenarios that need additional bans (e.g., domain words that are too generic locally) add a "Vocabulary discipline" subsection to `ARCHITECTURE.md`. A name should let the reader predict what the thing does without opening it.

**Programmatic check:** banned-name list enforced by tidiness-manager light scan. Domain-vocabulary judgments (is `process` a verb the domain actually uses, or noise?) remain smart-scan + human territory.

#### 4.4 Comment discipline

**Signs of divergence:** comments that restate the code they sit next to (`// loop through items`, `// return the result`); commented-out code left in place "just in case"; comments describing what the *caller* does ("used by the upload flow"); documentation comments that explain the obvious from the signature.

**Rule:** Comments should answer **why**, not **what**.
- **Remove** comments that only restate the code, name the obvious from the signature, point at callers, or describe a now-removed branch.
- **Keep** comments that document a non-obvious decision, a hidden constraint, an invariant reference, a workaround for a specific bug, or behavior that would surprise a future reader.
- **Test for removability:** ask "would removing this comment confuse a future reader who already has the code in front of them?" If no, remove. If yes, keep — it earns its place.

Do not delete comments simply to reduce comment density. The goal is signal-per-comment, not absence of comments. A scenario with twenty earned comments is in better shape than one with two restateful comments and silence elsewhere.

**Programmatic check:** none today. Restateful-comment detection is a strong tidiness-manager smart-scan candidate; flag the opportunity when you do this manually.

#### 4.5 Local state flow

**Signs of divergence:** package-level `var` declarations outside `config/`; `init()` functions that register globals; functions that mutate receiver state in surprising ways; functions whose output depends on hidden ambient context the caller can't see; shared mutable maps/slices passed implicitly.

**Rule:** A function's output is fully determined by its parameters (plus the receiver's *explicitly-passed-at-construction* dependencies). State mutation happens at well-named seams — `s.repo.Save(...)`, not `s.cache[k] = v` sprinkled mid-flow. Constructors take dependencies as named arguments; no global registries that things sneak into.

**Programmatic check:** ambient-state detection is a tidiness-manager smart-scan candidate; partially covered today by lint rules against package-level mutables, but the deeper "hidden dependency" patterns remain manual.

#### 4.6 Scattering and co-location

**Signs of divergence:** a single behavior requires opening five-plus files in sequence to follow; helpers for one domain live in a generic `utils` package; sibling files that should be in one domain folder are split across the repo.

**Rule:** If deleting the capability should delete the code, the code belongs with that capability's domain folder. Helpers used by exactly one domain stay domain-local. Helpers used by 2+ unrelated domains move to `internal/<substrate>/` (Go) or `lib/` (TS), with a justified seam entry.

**Note:** Most scattering findings are actually screaming-architecture findings — hand them off. Cog-load owns the case where a single domain's own code is fragmented across files within the same folder for no reason (e.g., five 30-line files where two 80-line files would read more naturally).

**Programmatic check:** cross-domain scattering is screaming-architecture's territory. Within-domain over-fragmentation is currently human-read.

---

### 5. Working with `tidiness-manager`

Tidiness-manager is the substrate that makes this skill verifiable today and increasingly programmatic over time.

**Primary commands** (consult `tidiness-manager --help` for current surface):

```bash
# Run the fast static-analysis pass; treat as L2/L3 verification.
tidiness-manager scan {{TARGET}} --type light

# Run AI-assisted deeper analysis; surfaces L3/L4 hotspots that lint cannot catch.
tidiness-manager scan {{TARGET}} --type smart

# List currently-tracked issues for this scenario.
tidiness-manager issues {{TARGET}}

# Start or check a campaign.
tidiness-manager campaigns
```

**Two-way relationship.** When you read a tidiness-manager scan, trust the output as the source of truth for what's currently enforced. When you make a manual cog-load fix that *could* be programmatic but isn't yet (e.g., you removed three restateful comments by reading the file), record that as a tidiness-manager enhancement candidate. Over time, recurring manual findings should migrate into tidiness-manager's check surface — that is how this skill becomes thinner and more deterministic.

**When tidiness-manager is unavailable** (not running, not yet built for this stack), fall back to the underlying tools directly (`golangci-lint run`, `npm run lint`) and note the gap in `PROBLEMS.md` so the scenario can graduate to tidiness-manager coverage later.

---

### 6. Safe Refactoring Guidelines

You may:
- split overly long functions into smaller, well-named units when each piece names a real domain operation
- collapse wrappers that add no domain meaning at the call site
- reorder code to bring related pieces closer together within a file
- improve naming, comments, and lightweight documentation
- adjust tests to match clarified behavior (without weakening coverage)

You must:
- preserve observable behavior and user-facing workflows
- keep public interfaces stable where feasible; if you change them, update all call sites consistently
- avoid deleting meaningful tests or relaxing their strictness to make changes pass
- avoid large, high-risk rewrites in a single loop; prefer incremental steps that clearly improve clarity
- never reverse a `decision-boundary-extraction` or `seam-discovery-and-enforcement` outcome on local-readability grounds alone — engage the prior audit's reasoning first

If you discover an area that needs substantial redesign, record the candidate in `PROBLEMS.md` rather than partially rewriting it here.

---

### 7. Audit Workflow

1. **Confirm prerequisites.** Skim `ARCHITECTURE.md`, `SEAMS.md`, `INVARIANTS.md`. If the structural audits have not been run and the scenario is visibly fragmented, hand off — cog-load on a structurally broken scenario produces local polish that obscures the real problem.
2. **Run tidiness-manager.** Start with `tidiness-manager scan {{TARGET}} --type light`. Treat any L2/L3 violations as the first work to address.
3. **Read smart-scan results** (`--type smart`) for L4 dimensions and patterns lint cannot reach.
4. **Walk the code-craft dimensions** in §4, focusing on the highest-impact divergences first: usually function shape and naming.
5. **Route each finding** through §3's table. Structural findings → hand off. Local findings → fix or record in the appropriate existing doc.
6. **Note enhancement candidates** for tidiness-manager when you find yourself doing the same manual check repeatedly.
7. **Update existing docs** via `knowledge-observatory-tools`: complexity hotspots into `PROBLEMS.md`, ambient-state findings into `INVARIANTS.md`, justified-or-collapsed indirection into `SEAMS.md`, vocabulary discipline into `ARCHITECTURE.md` when scenario-local extensions are needed.
8. **Verify maturity advancement.** Re-run the relevant tidiness-manager scan; confirm the level on the §2 ladder genuinely moved.

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
- reversing prior audit outcomes (`decision-boundary-extraction`, `seam-discovery-and-enforcement`) on local-readability grounds without engaging their reasoning
- claiming maturity advancement that the corresponding tidiness-manager scan cannot verify
