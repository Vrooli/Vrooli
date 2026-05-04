# React-Vite Template Fitness — Iteration 1 (cli-core declarative surface + canonical-domain refactor)

**Plan file**: `/home/matthalloran8/.claude/plans/thanks-for-the-postgres-wise-hearth.md`
**Authored**: 2026-05-04
**Iteration**: 1 of N (multi-iteration optimization; see §1.2)
**Owner**: meta-optimization team (toolchain-validator)
**Supersedes**: prior plan in this file (reference-pattern-fitness lens establishment — landed)

---

## 1. Purpose

### 1.1 What this plan is

The first iteration of a multi-iteration program to maximize the **reference-pattern fitness** of `templates/scenarios/react-vite/` — i.e. its quality *as a copy source* for every scenario the template will generate. The lens, its sub-lenses, and the prior audit findings are codified at [`docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md). This plan implements the substrate fix the lens flagged as the highest-multiplier finding (Tier-1 #1 in the worked example) and establishes the **measurement harness** every future iteration runs against.

This plan delivers two coupled deliverables:
1. **A measurement harness** that gives every future iteration the same yardstick. Without it, "did this iteration improve fitness?" is a vibe, not a measurement.
2. **The first substrate change**: cli-core grows a declarative arg-schema + proto-typed API call + envelope-aware error helpers, and the react-vite template's notes domain is rewritten against the new surface. This is the *canonical demonstration* of the new substrate, not an exhaustive sweep of every fitness finding.

### 1.2 What this plan is NOT

This plan **does not** put the template in its final state. The reference-pattern-fitness audit surfaced six tiered findings; this iteration addresses two of them (Tier-1 #1 in cli-core; Tier-1 #2 in-template via `lib/api.ts`) and explicitly defers four (Tier-1 #3 route-vs-endpoint parity test; Tier-2 #4 `cli_commands_seed.json` triple source; Tier-2 #5 `Repository.Create` DTO; Tier-3 #6 description style). Closing those is the work of iterations 2..N, each of which will:
- declare its hypothesis
- re-run the measurement harness
- compare against the baseline frozen here
- write the new baseline into the harness's results table
- propose what iteration N+1 should attack

The agent executing this plan must understand: **shipping iteration 1 is success even if the final per-domain line counts are still high.** The final state is the destination of iterations N..M, not iteration 1.

### 1.3 Why this matters

The react-vite template is the canonical scaffold for every scenario Vrooli generates. The reference-pattern-fitness audit found that adding a domain to a generated scenario costs ~250 lines of pure infrastructure boilerplate per domain — `apiError`/`decodeEnvelope`/`formatX`/protojson decode-ribbon in the CLI, `decodeApiError`/`if (!res.ok)` ribbon in the UI lib. Multiplied across N future scenarios × M domains per scenario, this is the largest known ongoing tax in the codebase that's *fixable at the substrate layer*. The right home for the fix is `cli-core` (substrate, cross-scenario) plus a small in-template helper (`ui/src/lib/api.ts`).

---

## 2. Hard rules

These rules constrain every phase. Repeat them in the Definition of Done; failures here invalidate the iteration.

### 2.1 Iteration discipline

- **The plan delivers iteration 1, not the final state.** Do not silently expand scope to address Tier-1 #3 / Tier-2 #4–5 / Tier-3 #6. They have explicit homes in iterations 2..N.
- **Measurement harness is required before any substrate change is shipped.** Phase A and B (harness + baseline) gate Phase C (cli-core change). If baseline numbers are not recorded, do not start the substrate change.
- **The hypothesis must be written before the change, not after.** Phase A includes a `HYPOTHESIS.md` declaring "we believe X causes Y; if we do Z, the metric for scenario N will move from A to B." Phase E grades whether the hypothesis was right. A correct-but-no-improvement result is a *valid* iteration outcome — it tells us cli-core wasn't the bottleneck and saves iteration 2 from wasted work.

### 2.2 Greenfield (template side)

Templates are regenerated, not migrated. Existing scenarios already generated from older templates are out of scope — they catch up per scenario when each is next touched.

In template-side code (`templates/scenarios/react-vite/**`):
- **No compatibility shims**, deprecation wrappers, or "old + new path" branches. The template ships one shape; the old per-handler `flag.NewFlagSet` style is deleted, not aliased.
- **No `// Deprecated:`, `// legacy`, `// compat`, `type Old…= New…` markers.**
- **Failed approaches surface immediately** — if a phase fails its checklist, fix the substrate; do not paper over.

### 2.3 Additive only (cli-core side)

cli-core is consumed by other scenarios and the prompt-manager runtime; existing call sites must continue to work. Therefore:
- **Do not modify exported function signatures** of existing cli-core surface (`Command.Run`, `RenderListReport`, etc.).
- **Do not remove exported symbols** even if internal callers stop using them in this iteration.
- **New surface lands alongside old surface.** `cliapp.Command` gains optional fields; `Command.Run` keeps its `func([]string) error` shape; the new declarative path is opt-in via a new field (e.g. `RunCtx func(cliapp.RunContext) error`).
- The greenfield rule applies *within the template*; it does not apply *inside cli-core itself*. cli-core's existing API stays stable so non-template consumers don't break mid-iteration.

### 2.4 Testable rule statement

A `git diff` of the post-iteration template against pre-iteration must contain zero occurrences of `Deprecated:`, `// legacy`, `// compat`, `// backwards-compat`, `type Old…= New…`. cli-core's diff may contain *additions* but no *deletions* of exported symbols.

---

## 3. Required reading

Run before executing any phase:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read reference-pattern-fitness
```

Reference files to read top-to-bottom:

- [`docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md) — strategic-canon home for the audit lens this iteration operates under. The four sub-lenses (per-replica cost, drift surface map, contract location audit, coordinated-edit count) are the metrics the harness records.
- [`docs/plans/cmd-thin-cli-overhaul-plan.md`](../../docs/plans/cmd-thin-cli-overhaul-plan.md) — the analogous declarative refactor for the root vrooli CLI. Phases 2 (declarative command-definition), 3 (shared arg-spec), 4 (unified help generation), 5 (collapse handler boilerplate) are the *exact pattern shape* this iteration applies to cli-core. **The agent should mirror `internal/cli/commandtree`'s shape into cli-core, not invent a new model.** Specifically:
  - `internal/cli/commandtree/commandtree.go` → cli-core's `cliapp.Spec` analogue (already partially exists as `Command` + `SubcommandGroup`).
  - `internal/cli/commandtree/args.go` → cli-core's new `cliapp.ArgSchema` + parser.
  - `internal/cli/commandtree/action.go` → cli-core's new `cliapp.RunContext` + shared action pipeline.
- [`packages/cli-core/cliapp/app.go`](../../packages/cli-core/cliapp/app.go) — the surface the agent extends. Confirms `Command.Run func([]string) error` is the existing dispatcher signature that must stay supported.
- [`packages/cli-core/cliutil/httpclient.go`](../../packages/cli-core/cliutil/httpclient.go) — `APIError` type and `ParseAPIError` already exist. The new envelope helper composes on top of these; do not replace them.
- [`templates/scenarios/react-vite/cli/domains/notes/handlers.go`](../../templates/scenarios/react-vite/cli/domains/notes/handlers.go) — current 178-line handler is the canonical refactor target. Read end-to-end before designing the substitute.
- [`templates/scenarios/react-vite/cli/domains/notes/handlers_test.go`](../../templates/scenarios/react-vite/cli/domains/notes/handlers_test.go) — tests that must continue passing after the refactor (test seam stays; only the implementation under it changes).
- [`templates/scenarios/react-vite/ui/src/lib/notes.ts`](../../templates/scenarios/react-vite/ui/src/lib/notes.ts) — current 142-line client. Half is shared infrastructure misfiled in the notes domain; the refactor moves it to `lib/api.ts`.
- [`templates/scenarios/react-vite/ui/src/lib/api.ts`](../../templates/scenarios/react-vite/ui/src/lib/api.ts) — 44-line file that will grow to host shared `protoFetch`, `ApiError`, `decodeApiError`. Today's `fetchHealth` uses a different (less typed) error path than notes; the refactor unifies them.

---

## 4. Problem statement

### 4.1 What the reference-pattern-fitness audit established

A 2026-05-04 audit of `templates/scenarios/react-vite/` (recorded in [`docs/agent-system/REFERENCE_PATTERN_FITNESS.md`](../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md) "Worked example") surfaced six tiered findings. The two this iteration addresses:

**Tier-1 #1 — CLI per-domain client boilerplate.**
[`cli/domains/notes/handlers.go`](../../templates/scenarios/react-vite/cli/domains/notes/handlers.go) is 178 lines for one CRUD domain. Of those, ~50 lines are pure infrastructure that every future domain will copy: `apiError` (lines 146–164), `decodeEnvelope` (166–178), the `flag.NewFlagSet` ribbon per handler (~6 lines × 3 = ~18), the `protojson.Marshal → core.Request → protojson.Unmarshal` ribbon per method (~10 lines × 3 = ~30), and `formatNote` (132–137; legitimate domain code, stays).

Replication factor: every scenario × every domain. For a hypothetical scenario with 3 domains, that's ~150 lines of boilerplate; 5 domains, ~250.

Substrate home: cli-core. The substrate exists; the helpers don't.

**Tier-1 #2 — UI per-domain lib boilerplate.**
[`ui/src/lib/notes.ts`](../../templates/scenarios/react-vite/ui/src/lib/notes.ts) is 142 lines for the same domain. The pattern `if (!res.ok) throw await decodeApiError(res); const json = await res.json(); return fromJson(Schema, json, { ignoreUnknownFields: true });` repeats 3× (once per method). `ApiError` and `decodeApiError` *live in `lib/notes.ts`* despite being domain-agnostic — when `lib/tasks.ts` lands, it will either import them from notes (coupling) or duplicate them (drift, already drifted: `lib/api.ts::fetchHealth` throws plain `Error` instead of typed `ApiError`).

Replication factor: every scenario × every UI domain.

Substrate home: in-template. A `lib/api.ts` `protoFetch<Req,Resp>(...)` helper plus relocating `ApiError`/`decodeApiError` to `lib/api.ts` is the fix. No external substrate needed.

### 4.2 What measurement currently looks like (anecdotally)

When this plan was authored, the audit produced **anecdotal** numbers based on file inspection:
- "Add new endpoint to existing domain" — ~14 places touched, ~100 lines of boilerplate
- "Change request shape" — ~6 places touched, mostly genuine domain work
- "Add new domain end-to-end" — ~20 files created, ~500 lines including boilerplate

These numbers are **not** the baseline this iteration freezes. Phase B re-derives them with a deterministic recipe that any future agent can re-run and get the same number. Anecdotes don't survive iteration N+3.

### 4.3 The risk we are pricing

Without this iteration, the template stays in a state where:
- Adding a domain feels worse than it should (cognitive load + boilerplate)
- Pattern-style drift accumulates (every scenario evolves its own apiError variant)
- "Why is the template like this?" becomes a recurring question

With this iteration:
- The substrate exists; future iterations have the platform to compose richer fixes on top
- The measurement harness exists; future iterations can show their improvement as a number
- The first scenario regenerated from the new template is the proof point

---

## 5. Scope

### 5.1 In scope

**Phase A — measurement harness.** New directory at the location decided in §6.4 with:
- `README.md` — what the harness is, who runs it, how to run it
- `SCENARIOS.md` — the 6 frozen workflow scenarios (§5.3) and their measurement recipes
- `METRICS.md` — the metric definitions
- `STOPPING_RULE.md` — when the multi-iteration program declares done
- `BASELINE.md` — empty in this phase; populated in Phase B
- `RESULTS.md` — empty in this phase; populated in Phase E
- `HYPOTHESIS.md` — populated in Phase A before substrate work begins

**Phase B — baseline.** Run the 6 scenarios against the *current* template tree (pre-substrate-change) using the recipes from `SCENARIOS.md`. Record numbers in `BASELINE.md`. Each scenario's recipe must be deterministic — the executing agent records the exact commands, branch names, and artifacts.

**Phase C — cli-core additive surface.** New files in `packages/cli-core/cliapp/`:
- `argschema.go` — `ArgSchema`, `Flag`, `Positional` types; descriptive only.
- `runcontext.go` — `RunContext` interface; methods `Flag(name) string`, `BoolFlag(name) bool`, `Positional(name) string`, `Args() []string` (raw fallback), `Render*(report)` methods that route to JSON or human based on global `--json`, `Core() *ScenarioApp` for direct API client access when needed.
- `parser.go` — internal parser that turns `(ArgSchema, []string) → RunContext + error`. Handles boolean flags, valued flags, required/optional positionals, repeated positionals, `--help`, unknown-option/missing-value/arity errors. Mirrors `internal/cli/commandtree/args.go` semantics.
- `helpgen.go` — generates `--help` output from `ArgSchema` + `Command` metadata. Mirrors `internal/cli/commandtree/help_*.go` shape.
- `call.go` — `Call[Req, Resp proto.Message](app *ScenarioApp, method, path string, req Req) (Resp, error)`. Marshals `req` via `protojson`, calls `app.Request`, decodes envelope on non-2xx (returning a typed `*cliutil.APIError` already populated by `ParseAPIError`), unmarshals 2xx body via `protojson`. Generic, works for any proto message.
- `envelope.go` — exports `DecodeEnvelope(body []byte) (*errorsv1.ErrorEnvelope, bool)` (currently scenario-local) and a thin `WrapAPIError(action string, err error, body []byte) error` that produces the same envelope-aware error string the scenario `apiError` produces today.

**Extended fields on `cliapp.Command`** (additive, optional):
- `Args ArgSchema` — schema describing flags/positionals.
- `RunCtx func(RunContext) error` — alternate handler signature; when set, cli-core dispatches via the parser path. When unset, the existing `Run func([]string) error` path stays in effect.
- `LongDescription string` (optional) — multi-line free-form text for `--help`.

Tests for every new file (next to it; `_test.go`). Style matches existing cli-core tests (stdlib + the existing testify usage where present in `cliapp` tests).

**Phase D — react-vite template refactor.** Notes domain rewritten against the new surface:
- `cli/domains/notes/handlers.go` shrinks from 178 → ~80 lines. `apiError` and `decodeEnvelope` deleted (now in cli-core). Each handler uses `cliapp.Call[Req, Resp]` and `RunContext.Flag`/`Positional`. `formatNote` stays (it's domain code).
- `cli/domains/notes/register.go` grows from 44 → ~75 lines. Each `Command` gains an `Args` schema and the `RunCtx` field.
- `cli/domains/notes/handlers_test.go` updated to drive handlers through the `RunCtx` path. Existing tests must keep their assertions (output strings, captured request bodies) — only the entry point changes.
- `ui/src/lib/api.ts` grows from 44 → ~110 lines. Hosts `ApiError`, `decodeApiError`, `protoFetch<Req, Resp>(method, path, opts)`. `fetchHealth` rewritten on top of `protoFetch` so the template ships only one error path.
- `ui/src/lib/notes.ts` shrinks from 142 → ~50 lines. Each function calls `protoFetch` and adds the missing-field guard. Re-exports `ApiError` from `./api` for compatibility with existing imports.
- `ui/src/lib/notes.test.ts`, `ui/src/lib/api.test.ts` updated where assertions reference the old shape; behavior coverage unchanged.

**Documentation updates** (template-side, supporting the refactor):
- `docs/internal/REPLACING-NOTES.md` — section "5. CLI domain" gets a reduced template showing the `Args` + `RunCtx` shape and the `cliapp.Call[Req,Resp]` pattern.
- `docs/internal/SEAMS.md` — new row for the `protoFetch` seam in `lib/api.ts`; new row for `cliapp.RunContext` if it's worth surfacing.
- `docs/concepts/ARCHITECTURE.md` — "CLI thin-wrapper" subsection gains one paragraph about the declarative arg-schema; one line about the `protoFetch` pattern.

**Phase E — re-measurement.** Run the same 6 scenarios against the post-substrate tree. Record numbers in `RESULTS.md`. Compare against `BASELINE.md`. Grade `HYPOTHESIS.md`. Write `ITERATION_2_PROPOSAL.md` summarizing what to attack next.

### 5.2 Out of scope

- **Tier-1 #3** — route-vs-endpoint parity test in `module_test.go`. Iteration 2 candidate; in-template fix; orthogonal to this iteration's substrate work.
- **Tier-2 #4** — `cli_commands_seed.json` triple-source-of-truth. Iteration 3 candidate; needs cli-core to expose a dump-commands binary or equivalent first (separate substrate change).
- **Tier-2 #5** — `RepositoryCreateInput` DTO replacing the partial-Note shape. Iteration 2 candidate; in-template; doesn't depend on this iteration.
- **Tier-3 #6** — `endpoints.go` description-style convention. Iteration 4+ paint; gets a PR when someone has a spare hour.
- **Migration of existing generated scenarios** to the new cli-core surface. Per the per-scenario greenfield rule from prior reference-react-vite passes; each scenario catches up when next touched. The template ships clean; old scenarios stay on their forked copy.
- **Cursor pagination on `/api/v1/notes`** — still tracked as a follow-up from earlier passes.
- **Auth on notes endpoints** — per-scenario concern; unchanged.
- **Adding new resources to the template** (redis, prom, etc.) — not in this audit.

### 5.3 The 6 frozen measurement scenarios

These are the canonical workflows the harness measures. They are frozen at iteration 1 and may not be redefined by iteration N without an explicit "scenario revision" entry in `SCENARIOS.md`.

| # | Scenario | Recipe summary |
|---|---|---|
| 1 | Add a new endpoint to an existing domain (`notes update` PATCH `/api/v1/notes/{id}`) | Implement on a clean branch off the post-iteration tree; measure files-touched, central-registry edits, lines-added (gofmt'd, prettier'd) across `cli/`, `api/`, `ui/`, plus a yes/no "could a junior do this from `REPLACING-NOTES.md` alone?" |
| 2 | Add a new domain end-to-end (`tasks` CRUD: list, create, get) | Same recipe; replicates a full vertical-stack walkthrough. |
| 3 | Add an optional field to a request shape (`tags []string` on `CreateNoteRequest`) | Same recipe; small change; tests the "change shape" cost. |
| 4 | Add a new error-code path (`409 conflict` for duplicate title in service + handler + UI surface) | Same recipe; tests that error-handling extension is cheap. |
| 5 | Delete a domain (`notes` per `REPLACING-NOTES.md`) | Same recipe; measures the cleanup tax — counterpart to #2. |
| 6 | Rename a CLI flag (`--title` → `--name` on `notes create`) | Same recipe; tests that breaking changes are localized. |

Each scenario's full recipe — including exact `git checkout` commands, the PR target, the `git diff --stat` invocation, the `cloc` / `tokei` invocation for non-test line counts, and the central-registry-edit definition — lives in `SCENARIOS.md`. This plan does not duplicate the recipes; the harness owns them.

### 5.4 The 4 metrics per scenario

Per [`REFERENCE_PATTERN_FITNESS.md`](../../docs/agent-system/REFERENCE_PATTERN_FITNESS.md) §"The four sub-lenses":

| Metric | Source sub-lens | What it measures |
|---|---|---|
| **Per-replica cost** | per-replica cost | Lines added (non-test, post-formatter) across `cli/`, `api/`, `ui/`. Proxy for "how much is each new copy paying for boilerplate." |
| **Drift surface count** | drift surface map | Number of places where information must agree but only convention enforces it (not types, not CI checks). E.g., `endpoints.go::Path` vs `r.HandleFunc`'s path string. |
| **Contract location** | contract location audit | For each non-trivial precondition/invariant introduced by the workflow, where does it live: type signature (best), CI check (good), code comment (debt), nowhere (worst). |
| **Central-registry edits** | coordinated-edit count | Number of files touched *outside the domain folder* — i.e., places where a shared registry/spec/seed must be updated. The most architecturally meaningful number. |

A 5th meta-metric is recorded but not optimized: **"could a junior do this from `REPLACING-NOTES.md` alone?"** as yes/no with a reason. This is the counterweight against gaming the four numerical metrics.

---

## 6. Current technical context

### 6.1 cli-core surface today

Located at [`packages/cli-core/`](../../packages/cli-core/) as a separate Go module. Scenarios pin via `replace` directive in their own `go.mod` (so cli-core changes propagate immediately to anything regenerated; no version-bump dance).

Public surface relevant to this iteration:

- `cliapp.ScenarioApp` — facade for scenario CLIs. Constructed via `NewScenarioApp(opts ScenarioOptions)` or `NewStandardScenarioApp(opts StandardScenarioOptions)`.
- `cliapp.Command` — `{ Name, Description string; Run func([]string) error }`. Run takes raw positional args; no built-in parsing.
- `cliapp.SubcommandGroup` — `{ Name, Description string; Subcommands []Command; NeedsAPI bool }`.
- `cliapp.RenderListReport(io.Writer, ListReport) error`, `RenderMutationReport`, `RenderOperationalReport`, `PrintReportJSON` — output contract renderers.
- `ScenarioApp.Get(path, query) ([]byte, error)`, `Request(method, path, query, body) ([]byte, error)`, `GetRoot`, `RequestRoot` — HTTP wrappers.
- `cliutil.APIError` (struct with StatusCode, Message, Code, Category, Details, Recovery, RecoveryHint, AutoFix, ManualSteps, RawResponse). `cliutil.ParseAPIError(statusCode, data) *APIError` does envelope detection.
- `cliutil.ParseInterspersed(fs *flag.FlagSet, args []string) error` — reorders args so flags come before positionals; called per handler today.

Tests live at `packages/cli-core/cliapp/*_test.go` and `packages/cli-core/cliutil/*_test.go`. Style: stdlib `testing.T` with mixed `require` (testify) usage in some files. Match the surrounding file's style when adding tests.

**No existing arg-spec, parser, help generator, envelope-helper, or proto-typed call helper.** Greenfield additions.

### 6.2 Template CLI surface today

`cli/domains/notes/handlers.go` (178 lines):
- `handlers` struct holds `*cliapp.ScenarioApp`.
- Three handlers (`list`, `create`, `get`) — each is `func(args []string) error`.
- `create` builds a `flag.FlagSet`, parses `--title` and `--body`, validates required, marshals proto request via `protojson.Marshal`, calls `core.Request`, unmarshals response, calls `RenderMutationReport`.
- `get` parses positional id, calls `core.Get`, unmarshals, calls `RenderListReport`.
- `list` calls `core.Get`, unmarshals, calls `RenderListReport`.
- `apiError(action, err, body)` — wraps an error with envelope decode (40 lines).
- `decodeEnvelope(body)` — decodes `errorsv1.ErrorEnvelope`, returns `(*ErrorEnvelope, bool)`.
- `formatNote(*notesv1.Note) string` — domain-specific row formatter (legitimately stays).

`cli/domains/notes/register.go` (44 lines):
- `Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup` — declares `Name`, `Description`, `NeedsAPI: true`, `Subcommands: []cliapp.Command{ {Name, Description, Run: h.list}, {Name, Description, Run: h.create}, {Name, Description, Run: h.get} }`.

`cli/domains/notes/handlers_test.go` (216 lines):
- `fakeAPI(t, status, body)` — `http.Handler` + recording struct with mutex.
- 8 tests covering list/create/get success and error paths plus `Register` wiring.

### 6.3 Template UI surface today

`ui/src/lib/api.ts` (44 lines):
- `fetchHealth(): Promise<HealthResponse>` — does its own fetch + ok-check + `fromJson`. Throws plain `Error("API health check failed: 401")` on non-2xx — different shape than notes' `ApiError`.

`ui/src/lib/notes.ts` (142 lines):
- `class ApiError extends Error` — held here; logically shared.
- `async function decodeApiError(res)` — held here; logically shared.
- `listNotes()`, `createNote(input)`, `getNote(id)` — three functions, each ~25 lines repeating the `fetch → !res.ok → fromJson → guard` ribbon.

### 6.4 Measurement harness location decision

Two viable locations:

1. **Recommended**: `scenarios/prompt-manager/store/teams/meta-optimization/notebook/template-fitness/react-vite/2026-05-04/`. The fitness PoR doc explicitly names this as the canonical home for fitness audits. Tracked in git, citable from future docs, durable across machines/iterations. The notebook already has retention conventions (date-stamped subfolders).
2. **Alternative requested by user**: an untracked path under `~/.claude/template-fitness/react-vite/iteration-1/`. Forces cleanup-by-disuse. Cost: machine-bound; iteration N+1 must re-baseline if the machine is wiped.

This plan recommends location #1 for iteration durability, and adds an explicit cleanup checklist item to the *final* iteration's Definition of Done (whichever iteration concludes the program). The agent executing this plan should use location #1 unless the user adjudicates otherwise. If the user prefers #2, the only change is `mkdir -p $LOCATION` in Phase A; everything else in this plan stays.

---

## 7. Target end state for iteration 1

After this plan executes, the following are true:

### 7.1 Measurement harness

1. The harness directory exists at the location chosen in §6.4 with the seven files listed in §5.1 Phase A.
2. `BASELINE.md` contains a row per scenario × per metric, with numbers derived by the recipes in `SCENARIOS.md`. Each cell cites the exact command and branch used to measure.
3. `HYPOTHESIS.md` declares the iteration's hypothesis ("we believe per-replica cost on scenarios 1, 2, 3 is dominated by CLI handler boilerplate and UI lib boilerplate; if we add cli-core's declarative surface and `lib/api.ts::protoFetch`, scenario 1 per-replica cost drops by ≥40%, scenario 2 by ≥35%, scenario 3 by ≥10%") in advance of substrate work.
4. `RESULTS.md` contains the post-substrate measurements alongside the baseline, plus a "hypothesis grade" section (right / partially right / wrong, with reason).
5. `ITERATION_2_PROPOSAL.md` is written, naming the next finding(s) to attack and why.

### 7.2 cli-core additive surface

6. New files exist in `packages/cli-core/cliapp/`:
   - `argschema.go`, `argschema_test.go`
   - `runcontext.go`, `runcontext_test.go`
   - `parser.go`, `parser_test.go`
   - `helpgen.go`, `helpgen_test.go`
   - `call.go`, `call_test.go`
   - `envelope.go`, `envelope_test.go`
7. `cliapp.Command` has new optional fields `Args ArgSchema`, `RunCtx func(RunContext) error`, `LongDescription string`. Existing `Run func([]string) error` field unchanged.
8. cli-core's existing tests still pass. New tests cover:
   - ArgSchema parser: no positional / one required / one optional / repeated positionals / boolean flag / valued flag / required-flag-missing / unknown-option / `--help`.
   - `Call[Req,Resp]`: success path; non-2xx with envelope; non-2xx without envelope; transport error; marshal failure.
   - `RunContext`: typed accessors; `Render*` routing JSON vs human based on global `--json`.
   - `helpgen`: deterministic output for representative `Command` + `ArgSchema` shapes; fixture-based.
9. No existing exported symbol in `cliapp` or `cliutil` is removed or has its signature changed.

### 7.3 react-vite template refactor

10. `templates/scenarios/react-vite/cli/domains/notes/handlers.go` is ≤ 90 lines, contains no `apiError` or `decodeEnvelope` helpers, uses `cliapp.Call[Req,Resp]` for every method, uses `RunContext` accessors instead of `flag.FlagSet`.
11. `templates/scenarios/react-vite/cli/domains/notes/register.go` declares `Args ArgSchema` per command and binds `RunCtx`. Pre-existing `Run` field is not used (greenfield: only one path in the template).
12. `templates/scenarios/react-vite/cli/domains/notes/handlers_test.go` exercises the `RunCtx` path. Captured-request and rendered-stdout assertions are unchanged in intent.
13. `templates/scenarios/react-vite/ui/src/lib/api.ts` exports `ApiError`, `decodeApiError`, `protoFetch<Req,Resp>`. `fetchHealth` is rewritten on top of `protoFetch` so health and notes share one error path.
14. `templates/scenarios/react-vite/ui/src/lib/notes.ts` is ≤ 60 lines. Each function is ≤ 8 lines. `ApiError`, `decodeApiError` deleted from this file.
15. `templates/scenarios/react-vite/ui/src/lib/api.test.ts` and `ui/src/lib/notes.test.ts` cover the new surface. Specifically: `protoFetch` envelope-decode tests; `protoFetch` proto-marshal/unmarshal tests; `notes.ts` per-method tests still pass.
16. `templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md` step 5 ("CLI domain") is updated. The "Steps to add your domain" walkthrough reflects the new declarative shape and shows fewer / shorter code blocks.
17. `templates/scenarios/react-vite/docs/internal/SEAMS.md` includes the `protoFetch` row.

### 7.4 End-to-end validation gate

18. A throwaway scenario generated from the post-iteration template passes the gate in §9.5. Zero residue after cleanup.

### 7.5 Greenfield-rule self-audit

19. `git diff` of the template tree contains zero occurrences of `Deprecated:`, `// legacy`, `// compat`, `// backwards-compat`, `type Old…= New…`. cli-core's diff shows additions only (no deletions of exported symbols).
20. `grep -rn "apiError\|decodeEnvelope" templates/scenarios/react-vite/cli/` returns zero matches (helpers fully migrated to cli-core).
21. `grep -rn "class ApiError\|async function decodeApiError" templates/scenarios/react-vite/ui/src/lib/notes.ts` returns zero matches (helpers fully migrated to `lib/api.ts`).

---

## 8. Implementation strategy

Five phases. Phases A → B → C → D → E run strictly in order. Phase B gates Phase C; Phase D gates Phase E.

### Phase A — Measurement harness

**Skill alignment**: implementation-plan-authoring §5C; reference-pattern-fitness PoR doc.

#### A.1 Choose location

Default to `scenarios/prompt-manager/store/teams/meta-optimization/notebook/template-fitness/react-vite/2026-05-04/` per §6.4. If the user has communicated the alternative untracked path before this phase starts, use that and update `README.md` to point at it.

#### A.2 Create the seven files

- `README.md`: title, what the harness is, who runs it (every iteration's executing agent), how to run it (`./run-baseline.sh` or a documented sequence of commands), retention policy ("delete after the multi-iteration program closes; see iteration N's Definition of Done").
- `SCENARIOS.md`: the 6 scenarios from §5.3, each with:
  - Goal (one sentence)
  - Recipe: exact `git` commands, file paths to touch, success criteria
  - Measurement command: the `git diff --stat`, `tokei`, etc. invocation
  - Central-registry-edit definition: the list of files counted as "central" (the modules registry, the `cli_commands_seed.json`, `App.tsx`, `domains.go`, etc. — enumerate explicitly so the count is unambiguous)
- `METRICS.md`: the 4 metrics from §5.4 plus the meta-metric, with the exact computation formula for each.
- `STOPPING_RULE.md`: "Stop when (a) no measured scenario has more than 2 central-registry edits AND adding a new domain (scenario 2) costs ≤ 200 lines, OR (b) consecutive iterations achieve <20% improvement on the same metric (diminishing returns)." Plus a "hard stop" of 6 iterations regardless.
- `BASELINE.md`: empty table with the headers from §5.4, ready for Phase B to fill in.
- `RESULTS.md`: empty table, ready for Phase E.
- `HYPOTHESIS.md`: written below as part of A.3.

#### A.3 Write the iteration-1 hypothesis

Populate `HYPOTHESIS.md` with the testable claim from §7.1 #3. Be explicit about:
- Which scenarios the hypothesis predicts to move
- The numerical threshold for "the hypothesis was right" (≥40%, ≥35%, ≥10%)
- What a "wrong" outcome would tell us (e.g., "if scenario 1 per-replica cost drops by <20%, the cli-core helpers were not the bottleneck and iteration 2 should look at higher layers — service layer, output contract, etc.")

#### A.4 Phase A validation

- All seven files exist at the chosen location.
- `README.md` is self-contained — an agent with no prior context can read it, find `SCENARIOS.md`, and run the measurement.
- `HYPOTHESIS.md` is testable.
- No code has been changed in `packages/cli-core/` or `templates/scenarios/react-vite/`. Phase A is purely additive and outside those trees.

### Phase B — Baseline measurement

**Gate**: Phase A must be complete.

#### B.1 Run all six scenarios against the *current* (pre-substrate) template tree

Each scenario is implemented on a separate temporary branch off the current head. The agent does not commit the implementation — it implements, measures, records, and discards the branch. (Or commits to a `baseline-scenario-N` branch that is preserved for the iteration but not merged.)

For each scenario:
1. Check out a clean branch.
2. Implement per the recipe in `SCENARIOS.md`.
3. Run the four measurement commands (one per metric).
4. Record numbers in `BASELINE.md`.
5. Discard / archive the branch.

#### B.2 Record the meta-metric

For each scenario, the agent records the yes/no judgment for "could a junior do this from `REPLACING-NOTES.md` alone?" with a one-sentence reason. This is the gameable-metric counterweight from §5.4.

#### B.3 Phase B validation

- `BASELINE.md` has a complete row per scenario × per metric.
- Every cell cites the command that produced it.
- No code in `packages/cli-core/` or template prod code has been mutated outside scenario-implementation branches that are now discarded.

### Phase C — cli-core additive surface

**Gate**: Phase B complete; baseline frozen.

This phase mirrors `internal/cli/commandtree`'s shape into cli-core. The agent should read `internal/cli/commandtree/{commandtree,args,action,help_*}.go` for design reference but write *new* code in cli-core (greenfield-within-cli-core: don't copy code; copy *patterns*).

#### C.1 `argschema.go`

Types:
```go
package cliapp

// ArgSchema describes the positionals and flags a Command accepts.
// Used by the parser to produce a RunContext and by helpgen to emit
// --help. Empty schema = command takes no args (still parsed for --help).
type ArgSchema struct {
    Positionals []Positional
    Flags       []Flag
}

type Positional struct {
    Name        string  // for help and RunContext lookup
    Description string
    Required    bool
    Repeated    bool    // last-positional only; mutually exclusive with multiple positionals after
}

type Flag struct {
    Name        string  // canonical name (without leading --)
    Aliases     []string // e.g. ["t"] for -t shorthand
    Description string
    Required    bool
    Default     string  // for valued flags
    Bool        bool    // if true, no value taken; presence = true
}
```

Test: schema validation — duplicate names rejected at validation time, repeated-positional-must-be-last enforced.

#### C.2 `runcontext.go`

```go
type RunContext interface {
    // Typed accessors. Panics if the name isn't declared in the ArgSchema.
    Flag(name string) string
    BoolFlag(name string) bool
    Positional(name string) string
    Positionals(name string) []string // for repeated

    // Raw fallback for handlers that need the leftover args.
    Args() []string

    // Render helpers route to JSON or human based on the global --json flag.
    RenderList(report ListReport) error
    RenderMutation(report MutationReport) error
    RenderOperational(report OperationalReport) error

    // Direct API client access for handlers that need to call beyond Call[Req,Resp].
    Core() *ScenarioApp

    // Stdout / stderr writers — handler tests can inject these.
    Stdout() io.Writer
    Stderr() io.Writer
}
```

Concrete impl is unexported. Constructed by the parser. Test: each accessor; Render routing JSON vs human; injection of writers for tests.

#### C.3 `parser.go`

`func parseArgs(schema ArgSchema, args []string, globals *GlobalOptions) (RunContext, error)` — internal.

Behavior parity with `internal/cli/commandtree/args.go`:
- Boolean flags accept `--name` (no value); valued flags accept `--name=value` or `--name value`.
- `--help` / `-h` short-circuits to a "help requested" sentinel handled by the dispatcher (returns a typed `ErrHelpRequested` so the dispatcher prints help and exits 0).
- `--` ends flag parsing; remaining args are positionals.
- Unknown options return a typed error through `cliutil`'s policy.
- Required-flag-missing returns a typed error citing the flag name.
- Positional arity errors (too few / too many) return typed errors.

Test: each shape from §C.1; one fixture-based test that locks the error messages so future regressions are visible.

#### C.4 `helpgen.go`

`func renderHelp(cmd Command, w io.Writer) error` — generates `Usage: <name> <args>`, Description, Options block, Positionals block, LongDescription. Uses only cmd metadata + ArgSchema; no hand-authored strings.

Test: golden files in `helpgen_testdata/` for representative shapes (no args, only flags, only positionals, mixed, with LongDescription).

#### C.5 `call.go`

```go
func Call[Req, Resp proto.Message](
    app *ScenarioApp,
    method, path string,
    req Req,
) (Resp, error) {
    // 1. Marshal req via protojson if non-nil.
    // 2. Call app.Request(method, path, nil, body).
    // 3. On error, route through WrapAPIError so the returned error
    //    carries the envelope's code+message when present.
    // 4. On success, allocate a new Resp, protojson.Unmarshal into it.
    // 5. Return Resp + nil.
}
```

The generic constraint is `proto.Message` from `google.golang.org/protobuf/proto`. The `Resp` allocation uses `proto.Clone` of the type's zero value or reflection — pick the cleanest pattern; `internal/cli/commandtree`'s analogous code (if any) shows what works at the same Go version.

Test: success with proto fixtures (use `errorsv1.ErrorEnvelope` itself as a proto fixture since it's already imported); 4xx with envelope; 4xx without envelope; transport error; nil request (pass-through).

#### C.6 `envelope.go`

```go
// DecodeEnvelope attempts to decode body as an ErrorEnvelope. Returns
// the decoded envelope and true on success; nil and false on failure
// (empty body, non-JSON, missing code field).
func DecodeEnvelope(body []byte) (*errorsv1.ErrorEnvelope, bool)

// WrapAPIError produces a human-readable, envelope-aware error string
// for failed API calls. action is a short verb phrase ("create note",
// "list tasks") used as the leading context.
func WrapAPIError(action string, err error, body []byte) error
```

`WrapAPIError`'s logic mirrors the scenario's current `apiError` — first try the body, then fall back to `cliutil.APIError.RawResponse`, then the underlying error. The point of moving it is *not* to change behavior; it's to delete it from every domain.

Test: the four branches — body decodes / cliutil.APIError.RawResponse decodes / neither decodes / err is nil.

#### C.7 Dispatcher integration

The existing `App.Run` and `ScenarioApp` dispatchers call `Command.Run(args)`. Extend them to:
- If `Command.Args` is non-empty OR `Command.RunCtx` is set: call `parseArgs(Args, args, globals)` to build a `RunContext`, then call `Command.RunCtx(ctx)`. Handle `ErrHelpRequested` by calling `renderHelp(cmd, stdout)` and returning 0.
- Otherwise: existing path — call `Command.Run(args)`.

Test: dispatcher routes correctly based on which field is populated; help-requested sentinel triggers help; unknown-option reaches the user.

#### C.8 Phase C validation

```bash
cd packages/cli-core && go test ./... -race
```

All existing tests still pass. New tests cover everything in §C.1–C.7. No exported symbol in cliapp or cliutil has been removed or had its signature changed (`go vet` + manual diff inspection).

### Phase D — react-vite template refactor

**Gate**: Phase C complete and green.

This phase rewrites the notes domain (CLI + UI) against the new substrate. The greenfield rule applies: the old apiError / decodeEnvelope / per-handler FlagSet code is *deleted*, not aliased.

#### D.1 Rewrite `cli/domains/notes/handlers.go`

Replace the file with handlers using `cliapp.Call[Req,Resp]`, `RunContext.Flag`, `RunContext.Positional`, and the existing `RenderMutation` / `RenderList` patterns. Delete `apiError` and `decodeEnvelope` entirely. `formatNote` stays.

Reference shape (representative; final lines may vary slightly):

```go
func (h *handlers) create(ctx cliapp.RunContext) error {
    resp, err := cliapp.Call[*notesv1.CreateNoteRequest, *notesv1.CreateNoteResponse](
        ctx.Core(), http.MethodPost, "/notes",
        &notesv1.CreateNoteRequest{
            Title: ctx.Flag("title"),
            Body:  ctx.Flag("body"),
        },
    )
    if err != nil {
        return err
    }
    if resp.Note == nil {
        return fmt.Errorf("server returned no note")
    }
    return ctx.RenderMutation(cliapp.MutationReport{
        Result:      []string{fmt.Sprintf("Created note %s.", resp.Note.Id)},
        Changes:     []string{formatNote(resp.Note)},
        NextCommand: []string{
            fmt.Sprintf("`notes get %s` — show this note", resp.Note.Id),
            "`notes list` — show all notes",
        },
    })
}
```

#### D.2 Rewrite `cli/domains/notes/register.go`

Each `Command` declares its `Args ArgSchema` and binds `RunCtx`:

```go
{
    Name:        "create",
    Description: "Create a note",
    Args: cliapp.ArgSchema{
        Flags: []cliapp.Flag{
            {Name: "title", Required: true, Description: "Note title"},
            {Name: "body", Description: "Note body"},
        },
    },
    RunCtx: h.create,
},
```

The `Run` field is left unset on each command (greenfield: one path only).

#### D.3 Update `cli/domains/notes/handlers_test.go`

Existing tests construct `*handlers` and call `h.list(args)` directly with raw `[]string`. Update them to either:
- Build a `RunContext` directly via a cli-core test helper (preferred), OR
- Build the full Command + ArgSchema and dispatch via the cli-core dispatcher (more end-to-end, better coverage).

The captured-request and rendered-stdout assertions (`require.Contains(t, out, "Found 2 note(s).")`, etc.) are unchanged in intent. Update the call site only.

If cli-core's test infrastructure doesn't expose a `NewTestRunContext` helper, add one in `packages/cli-core/cliapp/runcontext.go` (under a `// Exported for tests in scenarios that use this package` comment) — this is part of Phase C scope; pull it forward if Phase D discovers the gap.

#### D.4 Rewrite `ui/src/lib/api.ts`

Add `ApiError` (moved from notes.ts), `decodeApiError` (moved from notes.ts), and `protoFetch`:

```ts
export class ApiError extends Error { /* unchanged from current notes.ts */ }
async function decodeApiError(res: Response): Promise<ApiError> { /* unchanged */ }

interface ProtoFetchOptions<Req, Resp> {
  requestSchema?: GenSchema<Req>;
  request?: Partial<Req>;
  responseSchema: GenSchema<Resp>;
}

export async function protoFetch<Req, Resp>(
  method: string,
  path: string,
  opts: ProtoFetchOptions<Req, Resp>,
): Promise<Resp> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const init: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  };
  if (opts.requestSchema && opts.request) {
    const reqMsg = create(opts.requestSchema, opts.request);
    init.body = toJsonString(opts.requestSchema, reqMsg);
  }
  const res = await fetch(url, init);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(opts.responseSchema, json, { ignoreUnknownFields: true });
}

// fetchHealth rewritten on top of protoFetch — health and notes share one path.
export async function fetchHealth(): Promise<HealthResponse> {
  return protoFetch("GET", "/health", { responseSchema: ResponseSchema });
}
```

Type details (`GenSchema`, `Partial<Req>`) match `@bufbuild/protobuf`'s generated-schema typing; the agent should verify against the actual generated `.pb.ts` types and adjust the generic constraints. The shape above is illustrative.

#### D.5 Rewrite `ui/src/lib/notes.ts`

Each function becomes a `protoFetch` call plus the missing-field guard. ApiError is re-imported from `./api` (single source of truth). Re-export `ApiError` from notes.ts so existing callers in `features/notes/` don't break.

#### D.6 Update tests

- `ui/src/lib/api.test.ts` — add coverage for `protoFetch` (success path, 4xx with envelope, 4xx without envelope, missing required schema field). The existing `fetchHealth` tests should still pass.
- `ui/src/lib/notes.test.ts` — assertions unchanged. If they `vi.mock` `lib/api`, the mock surface gains `protoFetch`.

#### D.7 Update template docs

- `docs/internal/REPLACING-NOTES.md` — step 5 ("CLI domain") replaced with the new declarative shape. Step 8 (UI) references `protoFetch`.
- `docs/internal/SEAMS.md` — new row under UI: `lib/api.ts::protoFetch` — substrate seam, no production-vs-test split since it's a thin fetch wrapper.
- `docs/concepts/ARCHITECTURE.md` — "CLI thin-wrapper, domain-organized" subsection gains a paragraph about the declarative arg-schema; "UI feature-shaped" subsection notes that lib/ files use `protoFetch` for proto-typed I/O.

#### D.8 Phase D validation

```bash
cd templates/scenarios/react-vite/api && go test -race ./...
cd templates/scenarios/react-vite/cli && go test -race ./...
cd templates/scenarios/react-vite/ui && pnpm install --ignore-workspace && pnpm strings:check && pnpm lint && pnpm test:coverage && pnpm build
```

All green. Coverage gates from prior passes hold (API ≥ 75%, UI ≥ 85%).

### Phase E — Re-measurement

**Gate**: Phase D complete and green.

#### E.1 Run the same 6 scenarios against the post-substrate template tree

Same recipes as Phase B. Same commands. Record numbers in `RESULTS.md` next to the baseline cells.

#### E.2 Grade the hypothesis

In `RESULTS.md`, add a "Hypothesis grade" section:
- For each predicted scenario × metric movement: did the prediction land in the predicted range?
- Overall: right / partially right / wrong.
- One paragraph reasoning about *why*. If wrong, what does the result tell us about where the actual cost lives?

#### E.3 Write `ITERATION_2_PROPOSAL.md`

Name the next finding to attack. Default candidates (unranked here; the agent ranks based on what Phase E observed):
- Tier-1 #3 — `module_test.go` route-vs-endpoint parity test.
- Tier-2 #5 — `RepositoryCreateInput` DTO.
- Tier-2 #4 — `cli_commands_seed.json` triple-source — but this needs cli-core to expose a dump-commands binary first; if iteration 2 tackles this, half its work is in cli-core.

The proposal includes a hypothesis for iteration 2.

#### E.4 Phase E validation

- `RESULTS.md` complete with hypothesis grade.
- `ITERATION_2_PROPOSAL.md` written.
- The user can read both files and decide whether to greenlight iteration 2.

---

## 9. Contract decisions

These are the load-bearing design choices future readers may second-guess.

### 9.1 cli-core's new declarative path is opt-in via `RunCtx`, not a breaking change

`Command.Run` keeps its existing signature `func([]string) error`. New `Command.RunCtx func(RunContext) error` field opts into the parser path. Within the template, only `RunCtx` is used (greenfield); existing scenarios that haven't migrated keep using `Run`. Cost: cli-core has two dispatch paths internally. Benefit: zero breakage for non-template consumers.

### 9.2 `Call[Req, Resp]` uses generics (Go 1.18+); proto.Message is the constraint

The cleaner alternative — `Call(app, method, path, reqProto.Message, respFactory func() proto.Message) (proto.Message, error)` — works without generics but forces every caller to write `respFactory := func() proto.Message { return &notesv1.CreateNoteResponse{} }` and a type assertion. Generics give us `cliapp.Call[*notesv1.CreateNoteRequest, *notesv1.CreateNoteResponse](...)` returning a typed `*notesv1.CreateNoteResponse` directly. cli-core already uses Go 1.22; generics are fine.

### 9.3 `RunContext.Flag(name)` panics on undeclared name; does not return `(string, bool)`

Rationale: the schema is the source of truth. If `RunContext.Flag("titel")` is called but `"titel"` isn't in the ArgSchema, that's a programming error visible at the first invocation — better as a panic than as a silent empty string. Tests cover the schema-validation pre-flight at parser time, so the panic path triggers only on programmer typos.

### 9.4 `protoFetch` belongs in `lib/api.ts`, not a dedicated `lib/protoFetch.ts`

The lib/ folder has 3 files today; adding a fourth for one helper is overkill. `lib/api.ts` is currently minimal and becomes the natural home for shared API infrastructure. If lib/ grows beyond ~5 files later, splitting becomes worthwhile; today, single-file shared infra is the right shape.

### 9.5 The measurement harness lives in tracked git (default), not an untracked path

Per §6.4. The fitness PoR doc names the notebook location explicitly. Durability across iterations beats auto-cleanup-by-disuse. Cleanup is encoded as a checklist item in the *final* iteration's Definition of Done — not by burying the harness on one machine.

If the user prefers the untracked alternative, the only mechanical difference is path; everything else (recipes, metrics, hypothesis grading) is identical.

### 9.6 Hypothesis is testable, not aspirational

`HYPOTHESIS.md` predicts numerical movements (≥40%, ≥35%, ≥10%) per scenario. A "wrong" outcome — e.g., scenario 1 only drops 15% — is a *valid result*, not a failure. It tells us cli-core wasn't where the cost lived; iteration 2 should look elsewhere (service layer? output contract? proto code?). Hypothesis-grading is what makes the iterative loop convergent.

### 9.7 The template ships only the new path; old generated scenarios catch up per-scenario

Greenfield rule. The template's `cli/domains/notes/handlers.go` post-iteration uses `RunCtx`; agents who regenerate from the new template get the new shape. Existing generated scenarios that still use `Run` keep working (cli-core supports both); they migrate when next touched.

---

## 10. Testing plan

### 10.1 cli-core unit tests (Phase C)

Per §C.1–C.7. All in `packages/cli-core/cliapp/*_test.go`. Style: match adjacent files (stdlib + testify mixed).

Coverage targets:
- `argschema.go`: 100% — small surface, all branches reachable.
- `parser.go`: ≥ 90% — every shape from §C.3 plus error paths.
- `runcontext.go`: ≥ 95% — typed accessors, JSON-routing.
- `helpgen.go`: ≥ 90% — fixture-driven; the golden-file diff is the load-bearing assertion.
- `call.go`: ≥ 95% — success + 3 error paths.
- `envelope.go`: 100% — small.

### 10.2 Template unit/integration tests (Phase D)

- `cli/domains/notes/handlers_test.go` — same 8 tests; updated entry point.
- `ui/src/lib/api.test.ts` — gains `protoFetch` coverage.
- `ui/src/lib/notes.test.ts` — entry-point updates only.

Coverage holds prior gates: API ≥ 75%, UI ≥ 85%.

### 10.3 Measurement-harness self-validation (Phase A/B/E)

The harness itself is documentation, but Phase B's act of running every recipe *is* the validation that the recipes are deterministic. If Phase B records a number that varies across runs, the recipe in `SCENARIOS.md` is broken — fix the recipe before proceeding.

### 10.4 End-to-end gate (post-Phase D)

A throwaway scenario generated from the post-iteration template:

```bash
vrooli scenario generate react-vite \
  --id template-fitness-iteration-1-smoke \
  --display-name "Template Fitness Iteration 1 Smoke" \
  --description "End-to-end verification for the template-fitness iteration-1 plan"

cd scenarios/template-fitness-iteration-1-smoke

# API gates.
( cd api && go vet ./... && go build ./... && CGO_ENABLED=0 go build ./... && go test -race ./... )

# CLI gates — this is where the new substrate is exercised.
( cd cli && go vet ./... && go build ./... && CGO_ENABLED=0 go build ./... && go test -race ./... )

# UI gates.
( cd ui && pnpm install --ignore-workspace && pnpm strings:check && pnpm type-check && pnpm lint && pnpm test:coverage && pnpm build )

# Manifest validations (unchanged from prior passes).
jq . docs/manifest.json > /dev/null
jq . .vrooli/endpoints.json > /dev/null

# Runtime smoke — proves the new CLI handlers actually work end-to-end.
vrooli scenario start template-fitness-iteration-1-smoke
API_PORT=$(vrooli scenario status template-fitness-iteration-1-smoke --json | jq -r '.ports.api')
template-fitness-iteration-1-smoke notes list                         # default human output
template-fitness-iteration-1-smoke notes create --title hi --body bye # mutation report
template-fitness-iteration-1-smoke notes create --title ""            # exits non-zero with envelope-aware error

# --help works (proves helpgen is wired)
template-fitness-iteration-1-smoke notes --help
template-fitness-iteration-1-smoke notes create --help

# Cleanup throwaway. The canonical, fully-correct cleanup snippet — covering
# both naming conventions (dash form for schemas/go/typescript, underscore
# form for python) and both typescript output dirs (gen/typescript and
# gen/typescript/js) — lives in the harness README:
#   scenarios/prompt-manager/store/teams/meta-optimization/notebook/template-fitness/react-vite-template/2026-05-04/README.md
# Future iteration plans should link to that snippet rather than inline a
# variant; the inline form below misses Python-underscored gen and the
# gen/typescript/js mirror, which is how iteration 1 accumulated 62
# orphaned-file deletions before they were caught.
SCENARIO=template-fitness-iteration-1-smoke
SCENARIO_UNDERSCORE="${SCENARIO//-/_}"
vrooli scenario stop "$SCENARIO" 2>/dev/null || true
rm -rf "scenarios/$SCENARIO" \
       "packages/proto/schemas/$SCENARIO" \
       "packages/proto/gen/go/$SCENARIO" \
       "packages/proto/gen/typescript/$SCENARIO" \
       "packages/proto/gen/typescript/js/$SCENARIO" \
       "packages/proto/gen/python/$SCENARIO_UNDERSCORE"
( cd packages/proto && make generate )
git status packages/proto/   # confirm only intended deletions, nothing else
```

Every step must succeed first try.

---

## 11. Rollout / Validation Checklist

- [ ] Required-reading skills loaded (§3).
- [ ] **Phase A — harness**:
  - [ ] Location chosen per §6.4 (default: notebook path).
  - [ ] `README.md`, `SCENARIOS.md`, `METRICS.md`, `STOPPING_RULE.md`, `BASELINE.md` (empty), `RESULTS.md` (empty), `HYPOTHESIS.md` exist.
  - [ ] `HYPOTHESIS.md` predicts numerical movement per scenario.
  - [ ] No code changes outside the harness directory yet.
- [ ] **Phase B — baseline**:
  - [ ] All 6 scenarios run against the pre-substrate template.
  - [ ] `BASELINE.md` complete with 4 metrics × 6 scenarios + meta-metric.
  - [ ] Each cell cites the command that produced it.
- [ ] **Phase C — cli-core**:
  - [ ] `argschema.go`, `runcontext.go`, `parser.go`, `helpgen.go`, `call.go`, `envelope.go` (+ tests for each) exist.
  - [ ] `Command` gains `Args`, `RunCtx`, `LongDescription` fields.
  - [ ] Existing exported symbols unchanged (signature & names).
  - [ ] `go test ./packages/cli-core/...` green; no regressions.
  - [ ] Coverage targets per §10.1 met.
- [ ] **Phase D — template refactor**:
  - [ ] `cli/domains/notes/handlers.go` ≤ 90 lines; `apiError`/`decodeEnvelope` deleted.
  - [ ] `cli/domains/notes/register.go` declares `Args` per command and uses `RunCtx`.
  - [ ] `cli/domains/notes/handlers_test.go` updated; existing 8 tests pass.
  - [ ] `ui/src/lib/api.ts` exports `protoFetch`, `ApiError`, `decodeApiError`; `fetchHealth` rewritten on top.
  - [ ] `ui/src/lib/notes.ts` ≤ 60 lines; `ApiError`/`decodeApiError` deleted from this file (re-export still allowed).
  - [ ] `docs/internal/REPLACING-NOTES.md` step 5 updated; `docs/internal/SEAMS.md` row added; `docs/concepts/ARCHITECTURE.md` updated.
  - [ ] All template tests green; coverage gates hold.
- [ ] **Phase E — re-measurement**:
  - [ ] All 6 scenarios run against the post-substrate template.
  - [ ] `RESULTS.md` complete with measurements + hypothesis grade.
  - [ ] `ITERATION_2_PROPOSAL.md` written.
- [ ] **End-to-end gate** (§10.4) — every step passes on a throwaway scenario, zero residue.
- [ ] **Greenfield self-audit**:
  - [ ] `git diff` of template tree contains no `Deprecated:`, `// legacy`, `// compat`, `// backwards-compat`, `type Old…= New…`.
  - [ ] cli-core diff: additions only; no exported-symbol deletions or signature changes.
  - [ ] `grep -rn "apiError\|decodeEnvelope" templates/scenarios/react-vite/cli/` → empty.
  - [ ] `grep -rn "class ApiError\|async function decodeApiError" templates/scenarios/react-vite/ui/src/lib/notes.ts` → empty.

---

## 12. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| The hypothesis is wrong (cli-core wasn't the bottleneck; per-replica cost barely moves) | medium | Iteration design treats this as a *valid* outcome, not a failure. `HYPOTHESIS.md` says explicitly what a wrong result tells us. Iteration 2 redirects. |
| cli-core's existing consumers break despite the additive contract | low | §C.8 test gate runs full cli-core suite. Manual diff inspection in Definition of Done confirms no signature changes. The `replace` directive means in-repo consumers (other scenarios under `scenarios/`) recompile against the new code; their tests catch any silent breakage. |
| Generic `Call[Req, Resp]` runs into Go's protobuf reflection edge cases | medium | Test `call.go` against actual proto fixtures (`errorsv1.ErrorEnvelope` is already imported). If reflection is flaky, fall back to the factory-function shape (§9.2 alt) — the call-site cost is one extra `func() proto.Message {…}` per handler, still net positive vs today. |
| `RunContext.Render*` methods don't have access to the global `--json` flag because cli-core's parsing happens at a layer above the handler | medium | The dispatcher passes `globals *GlobalOptions` into `parseArgs`; the constructed `RunContext` closes over it. The same plumbing exists today for `app.Request`'s base URL. Verify in Phase C.2. |
| Help output drifts from the existing scenario's status command help shape | low | `helpgen` is fixture-tested. The `template-fitness-iteration-1-smoke` end-to-end gate runs `notes --help` and `notes create --help` so a regression surfaces. |
| Recipes in `SCENARIOS.md` are non-deterministic — Phase B numbers vary across runs | medium | Phase B's act of recording the numbers IS the recipe-validation gate. If a number varies, the recipe is broken; fix the recipe before proceeding. Lock formatter versions (gofumpt, prettier) in the recipe. |
| The notebook path is ambiguous (multiple `react-vite` audits have happened on different dates) | low | Date-stamped subfolders (`2026-05-04/`) are the convention. This iteration creates a new dated folder; future iterations reuse the same folder (since the harness IS the iteration tracker, not per-iteration scratch). |
| Phase D test refactor accidentally weakens assertions | medium | Pre-iteration assertions (`require.Contains(t, out, "Found 2 note(s).")`, etc.) are listed in §D.3 and must round-trip. Add a "test-coverage diff" check: every assertion present pre-Phase D must still be present post-Phase D, even if its surrounding setup changed. |
| The user changes their mind and prefers the untracked harness path | low | §6.4 calls this out. The mechanical difference is one `mkdir` invocation; nothing else in the plan changes. The agent should ask if they have access to the user; otherwise default to the notebook path. |

---

## 13. Non-goals / Prohibited patterns

- **No backwards-compatibility shims in the template.** Greenfield rule (§2.2). No `type oldHandler = func([]string) error` aliases, no `// Deprecated: use RunCtx` markers.
- **No exported-symbol deletions in cli-core.** Existing scenarios still consume the old surface.
- **No new fitness findings addressed beyond §5.1's two.** Tier-1 #3 / Tier-2 #4 / Tier-2 #5 / Tier-3 #6 are explicitly deferred.
- **No new measurement scenarios beyond the 6 in §5.3.** Frozen at iteration 1.
- **No JSON Schema for `HYPOTHESIS.md` / `RESULTS.md`**; markdown is enough until tooling forces the question.
- **No `// TODO`s left behind** unless they reference a tracked iteration-N proposal.
- **No `cli_commands_seed.json` shape changes.** That's iteration 3+ work.
- **No `--help` content rewrites for commands outside notes.** Only the notes domain uses the new declarative path in this iteration; status/configure stay on the existing path.
- **No documentation files outside what §5.1 enumerates.** Specifically: do not write a new architecture-decision-record file unless one is requested.

---

## 14. Definition of Done

The plan is done when **all** of the following are true:

1. **Phase A–E checklist (§11) is fully ticked.**
2. **End-to-end gate (§10.4) passes** on a throwaway-generated scenario with zero residue after cleanup.
3. **The harness exists and is discoverable.** `BASELINE.md` and `RESULTS.md` are populated; `HYPOTHESIS.md` is graded; `ITERATION_2_PROPOSAL.md` is written.
4. **The greenfield self-audit (§7.5) passes.** No legacy/compat markers in the template; only additions in cli-core.
5. **`git diff` of `templates/scenarios/react-vite/cli/domains/notes/handlers.go`** shows the file shrinking from 178 lines to ≤ 90 lines, with `apiError` and `decodeEnvelope` deleted.
6. **`git diff` of `templates/scenarios/react-vite/ui/src/lib/notes.ts`** shows the file shrinking from 142 lines to ≤ 60 lines, with `class ApiError` and `async function decodeApiError` deleted (now in `lib/api.ts`).
7. **All test gates green.** cli-core suite + template (api/cli/ui) suites + end-to-end gate.
8. **The plan's "Deviations during execution" section (§15) is populated** (with `(none)` if there were truly none — explicit so iteration 2's agent doesn't wonder).

If any item above is false, iteration 1 is not done.

---

## 15. Deviations during execution

Each entry: **file or step** — **what changed** — **why**.

(Populated during execution. Agent: do not leave this section empty at completion; if there were no deviations, write `(none)` here so iteration 2's agent doesn't wonder.)

---

## 16. Iteration handoff

After this plan completes, the next agent picking up template-fitness work:

1. Reads `RESULTS.md` and `HYPOTHESIS.md` to understand what iteration 1 attempted and how it landed.
2. Reads `ITERATION_2_PROPOSAL.md` for the recommended next attack.
3. Authors `iteration-2-plan.md` (or its analogue) following the same shape:
   - Phase A — adjust the harness (only if STOPPING_RULE.md or scenario revisions require it; otherwise reuse).
   - Phase B — baseline = iteration 1's RESULTS.md numbers (no re-baselining unless the substrate change is large enough to demand it).
   - Phase C — substrate change (or in-template change, depending on what iteration 2 attacks).
   - Phase D — apply to template.
   - Phase E — re-measure; grade hypothesis; write `ITERATION_3_PROPOSAL.md`.
4. Re-runs the same 6 measurement scenarios so RESULTS.md grows a column per iteration.
5. **For the end-to-end gate's smoke-scenario cleanup**: link to the canonical recipe in the harness README ("Smoke-scenario cleanup recipe" section). Do NOT copy the inline snippet from this plan's §10.4 verbatim — the iteration-1 polish pass found the inline form was missing Python-underscored gen and the `gen/typescript/js/` mirror, leaving 62 orphan-file deletions to be retroactively cleaned. The harness recipe is the source of truth; future plans cite it.

The `STOPPING_RULE.md` adjudicates whether iteration N+1 should happen at all.

---

## Appendix A — Why iteration 1 picks Tier-1 #1 + #2 specifically

The reference-pattern-fitness audit produced 6 findings. Iteration 1 picks the two with:
- **Largest replication factor**: Tier-1 #1 (CLI per-replica cost) and Tier-1 #2 (UI per-replica cost) both scale per-domain × per-scenario. Tier-1 #3 (route-vs-endpoint drift) is per-domain only — smaller blast radius.
- **Smallest substrate-change risk**: Tier-1 #1's home is cli-core, which has stable additive-only conventions and a clean test surface. Tier-2 #4's home is also cli-core but requires *removing* a source of truth (`cli_commands_seed.json`), which is a structurally bigger change.
- **Cleanest stories for the harness**: Per-replica cost is the easiest sub-lens metric to operationalize. Drift surface count and contract location are auditor-judgment-heavy; central-registry edits are low-cardinality. Picking the most-numerical metric for iteration 1 lets us validate the harness's measurement protocol before we lean on the harder metrics.

---

## Appendix B — File touch summary

For Phase D (template), the files modified:

| Path | Change |
|---|---|
| `templates/scenarios/react-vite/cli/domains/notes/handlers.go` | Rewrite |
| `templates/scenarios/react-vite/cli/domains/notes/register.go` | Rewrite |
| `templates/scenarios/react-vite/cli/domains/notes/handlers_test.go` | Update test entry points |
| `templates/scenarios/react-vite/ui/src/lib/api.ts` | Rewrite (grow) |
| `templates/scenarios/react-vite/ui/src/lib/notes.ts` | Rewrite (shrink) |
| `templates/scenarios/react-vite/ui/src/lib/api.test.ts` | Add protoFetch coverage |
| `templates/scenarios/react-vite/ui/src/lib/notes.test.ts` | Update entry points |
| `templates/scenarios/react-vite/docs/internal/REPLACING-NOTES.md` | Update step 5 |
| `templates/scenarios/react-vite/docs/internal/SEAMS.md` | Add `protoFetch` row |
| `templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md` | Update CLI subsection |

For Phase C (cli-core), the files added:

| Path | Change |
|---|---|
| `packages/cli-core/cliapp/argschema.go` | New |
| `packages/cli-core/cliapp/argschema_test.go` | New |
| `packages/cli-core/cliapp/runcontext.go` | New |
| `packages/cli-core/cliapp/runcontext_test.go` | New |
| `packages/cli-core/cliapp/parser.go` | New |
| `packages/cli-core/cliapp/parser_test.go` | New |
| `packages/cli-core/cliapp/helpgen.go` | New |
| `packages/cli-core/cliapp/helpgen_test.go` | New |
| `packages/cli-core/cliapp/helpgen_testdata/` | New (golden files) |
| `packages/cli-core/cliapp/call.go` | New |
| `packages/cli-core/cliapp/call_test.go` | New |
| `packages/cli-core/cliapp/envelope.go` | New |
| `packages/cli-core/cliapp/envelope_test.go` | New |
| `packages/cli-core/cliapp/app.go` | Edit: add `Args`, `RunCtx`, `LongDescription` to `Command` struct |
| `packages/cli-core/cliapp/dispatcher.go` (or wherever Run dispatch lives) | Edit: route on `Args`/`RunCtx` presence |

For Phase A (harness), the files added at the location chosen in §6.4:

| Path | Change |
|---|---|
| `<harness-root>/README.md` | New |
| `<harness-root>/SCENARIOS.md` | New |
| `<harness-root>/METRICS.md` | New |
| `<harness-root>/STOPPING_RULE.md` | New |
| `<harness-root>/HYPOTHESIS.md` | New |
| `<harness-root>/BASELINE.md` | New (populated in Phase B) |
| `<harness-root>/RESULTS.md` | New (populated in Phase E) |
| `<harness-root>/ITERATION_2_PROPOSAL.md` | New (populated in Phase E) |
