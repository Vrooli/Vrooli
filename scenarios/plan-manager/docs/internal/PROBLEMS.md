# Problems — Plan Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-25 — Documentation-first; no runtime code yet — RESOLVED 2026-06-27

**Status:** RESOLVED. All product domains (`plans`, `authoring`, `execution`, `validation`, `log`) are now implemented as real vertical slices and the fenced `notes` example domain has been removed (see PROGRESS.md). The historical entry is retained for context.

**Symptom:** The scenario validates its PRD/requirements/docs but has no real domains; the fenced `notes` example domain is still present, so the `example-domain-removed` orientation step fails by design.

**Root cause:** Intentional — this session implemented Gates 1–3 + 5b (PRD, requirements, concept/business/ops docs) only. The first vertical slice (`plans`) and `vrooli scenario detemplate` are future work.

**Workaround:** N/A — expected state for a documentation-first handoff.

**Real fix:** Build the `plans` slice (Gate 6) beside the example, then `vrooli scenario detemplate plan-manager` (Gate 7).

**Owner:** unassigned (next implementation session).

**Refs:** `docs/START-HERE.md` Gates 6–7; `requirements/` modules.

### 2026-06-25 — Legacy `~/.vrooli/plans` coexistence is unspecified in code

**Symptom:** plan-manager will share the home store with existing markdown plan files, but the adoption/coexistence path is documented (DATA.md) not built.

**Root cause:** Storage decision (scenario-owned logic over the durable home store) is new; the migration/coexistence step is deferred to the `plans` slice.

**Workaround:** Treat existing markdown plans as import sources; do not destructively migrate.

**Real fix:** Implement non-destructive adoption of existing `~/.vrooli/plans` records when building the `plans` domain.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md` (Migrations And Compatibility); `internal/app/plans` (existing store).

### 2026-06-25 — `prd-control-tower prd generate` (LLM path) returns HTTP 500

**Symptom:** `prd-control-tower prd generate plan-manager --publish` fails with `api error (500)`.

**Root cause:** Environmental (the generate LLM endpoint), not the scenario. Same behavior was seen for meta-optimization-manager.

**Workaround:** Hand-author `PRD.md` to the canonical template and validate with `prd-control-tower prd validate` (status healthy).

**Real fix:** None on this scenario's side; revisit if the generate endpoint is restored.

**Owner:** unassigned (prd-control-tower).

**Refs:** `PRD.md`; `prd-control-tower prd validate plan-manager --json`.

### 2026-06-27 — Full Test Genie remains blocked by scenario-health gates

**Symptom:** `vrooli scenario test plan-manager` run `20260627-173227-a56f3f9f` completed with 13/17 phases passing. The failing phases were standards, architecture, performance, and tidiness. Baseline diff run `20260627-174053-d1c3e988` resolved the current work as `preexisting`: structure/workflows/visuals unchanged, inherited standards failure, and unit cleared.

Update 2026-07-01: the workspace-aware root CLI/hygiene hardening pass cleared the standards and performance gates individually. `test-genie execute plan-manager standards --json` passed as run `20260701-115627-497b8f37`; `test-genie execute plan-manager performance --json` passed as run `20260701-120820-8216ef58`. The remaining managed blockers are architecture and tidiness: architecture run `20260701-120906-d2c673c7` still gates on seven deterministic layering findings, and tidiness run `20260701-120950-4bede63a` still reports 12 error-level maintainability findings.

**Root cause:** Mixed residual scenario-health debt outside the hardening plan's feature path. The standards and performance slices had narrow fixes: Plan Manager API error responses now stamp security headers directly, and the shared scenario server template gzip-compresses large static assets. Architecture remains campaign-scale boundary debt around undeclared `planmodel`/`planproto` shared-kernel ownership and handler/internal sibling-domain imports. Tidiness remains campaign-scale maintainability debt across duplicated code, long files, and high-complexity functions in existing API/CLI/UI files; the hardening-local `ParseRegressionAnchorBlock` complexity finding was addressed by splitting the parser into helpers.

**Workaround:** Use the narrower validated gates for readiness decisions: API `go test ./...`, CLI `go test ./...`, UI `pnpm type-check`, `pnpm test`, `pnpm lint`, `pnpm test:coverage`, requirements validation, `cli-health`, static `ui-health`, live `ui-health`, and `git-control-tower baseline diff` classification.

**Real fix:** Resolve or explicitly campaign the remaining scenario-health gates: declare or restructure `planmodel`/`planproto` as owned shared kernels so architecture layering is deterministic-clean, then finish tidiness refactors for duplicated code/complexity hot spots. Test Genie's latest architecture and tidiness runs both recommend campaign tracking rather than ad hoc fixes.

**Owner:** unassigned.

**Refs:** `coverage/runs/20260627-173227-a56f3f9f/phase-results/{architecture,performance,tidiness,standards}.json`; `coverage/runs/20260701-115627-497b8f37/phase-results/standards.json`; `coverage/runs/20260701-120820-8216ef58/phase-results/performance.json`; `coverage/runs/20260701-120906-d2c673c7/phase-results/architecture.json`; `coverage/runs/20260701-120950-4bede63a/phase-results/tidiness.json`; `git-control-tower baseline diff status --scenario plan-manager --name plan-manager-hardening-readiness --run 20260627-174053-d1c3e988`.

### 2026-06-28 — docs-first contract ahead of code (structured-rendered-plans)

**Symptom:** `PLAN-MODEL.md`, `DATA.md`, `FLOWS.md`, and `ARCHITECTURE.md` now
describe the professional plan structure (problem_statement, target_outcome,
assumptions, technical_approach, validation_strategy, final_validation_commands,
risks_hazards, prohibited_approaches, work_posture/source/detail,
import_provenance, preserved_legacy_sections, and phase affected_areas/steps/
expected_outputs/validation/handoff_notes/risks_hazards) plus the automatic
Greenfield/Brownfield posture and a fixed render order. The proto/model/renderer/
parser/wizard implement these incrementally across Phases 2–8 of the
`plan-manager-structured-rendered-plans` plan.

**Root cause:** Intentional. The docs lock the contract first (Phase 1) so
implementation cannot drift into legacy-section cloning. Until Phase 8 golden
tests land, treat `PLAN-MODEL.md` as the source of truth for section names.

**Status:** Tracked by the plan's phases; resolved when golden tests assert the
renderer/parser/wizard match this document.

### 2026-06-28 — Authoring response contract too heavy for small/local agents — RESOLVED

**Symptom:** Small/local-model authoring was failing on adoption blockers: every
mutation echoed the full `AuthoringSession` (blowing the context window); a bad
accepted relevant-context item found in `author preview` could only be fixed by
deleting the phase/session; free-form phase `relevant_context` prose was turned
into executable `prompt-manager skill read …` commands with malformed argv;
`References` was effectively optional in the skeleton; `author preview` rendered a
different work posture than finalize for brownfield scenarios; degraded anchor
autofill gave no recovery path; direct `phase add/update` lacked the canonical
phase fields; `plans update` dropped import lineage; and a misplaced `--auto-start`
produced a bare "unknown option".

**Root cause:** The wizard contract was designed around a full-session echo and a
prose-to-command heuristic; preview did not apply the plans-domain posture
derivation; and several CLI surfaces lagged the canonical phase/governance model.

**Resolution (known behavior now):**

- **Focused responses.** Normal authoring mutations return a compact
  `AuthoringProgress` + an `AuthoringMutationSummary` + the single changed object
  + violations + the next guided step — never the accumulated session. Full state
  is read explicitly via `GetSession` (`author get-session`); the UI uses
  read-after-write. `ContinueAuthoring` returns the single current work item;
  `PreviewPlan` / `plans render` return full markdown.
- **Accepted-context recovery.** `UpdateRelevantContextItem` /
  `RemoveRelevantContextItem` (`author context-update` / `context-remove`) edit or
  drop one accepted item by id before finalize; removal recomputes structure
  violations so a resulting gate is reported with its recovery action.
- **Free-form prose → notes only.** `author phase-submit --field relevant_context`
  records prose as notes (`NO_CONTEXT:` preserved); executable setup context must
  use typed `context-submit` / candidate acceptance.
- **References is a mandatory skeleton section**, satisfiable by a `NO_CODE_REFS:`
  reason, communicated from the start.
- **Preview posture parity.** `author preview` stamps the same posture derivation
  as finalize via the `PosturePreparer` seam (see [`SEAMS.md`](SEAMS.md)).
- **Anchor recovery NextActions.** The `regression_anchor` guided step carries a
  retry, a `git-control-tower baseline snapshot …` command, and a verbatim
  fallback anchor block template.
- **Direct phase CLI parity.** `phase add/update` expose `--affected-areas`,
  `--steps`, `--expected-outputs`, `--validation`, `--risks-hazards`,
  `--handoff-notes`; `phase update` is full-replace.
- **Import provenance durability.** `plans update` preserves `import_provenance`
  and `preserved_legacy_sections` when omitted.
- **Global-flag placement.** Global flags (e.g. `--auto-start`) must precede the
  subcommand; a misplaced one now yields a clear placement hint.

**Refs:** `packages/proto/schemas/plan-manager/v1/authoring/authoring.proto`;
`api/handlers/authoring/{connect_handler.go,convert.go,module.go}`;
`api/internal/authoring/progress.go`; `cli/manifest.json`;
[`../concepts/FLOWS.md`](../concepts/FLOWS.md) (authoring response contract);
[`../reference/cli-commands.md`](../reference/cli-commands.md).

### 2026-06-28 — GCT / Test Genie have no native path-scoped baseline

**Symptom:** A plan whose change boundary includes non-scenario paths
(`packages/proto/**`, `docs/**`, root tooling) has no pass/fail regression
oracle for those paths. Plan Manager derives a scenario baseline **oracle** per
affected scenario but can only emit an **informational** `git diff --stat -- <repo
paths>` for the non-scenario allow globs — it is never treated as a verdict.

**Root cause:** `git-control-tower baseline snapshot/diff` requires
`--scenario --name`; there is no path-scope baseline command. `test-genie execute`
requires a scenario positional argument (`--scenario-path` only relocates a
scenario's physical placement, it does not test arbitrary repo paths). Both are
scenario-keyed substrate.

**Workaround:** Plan Manager consumes the substrate honestly: scenario baseline
checks are oracles, repo/path diffs are informational and labelled as such in the
rendered anchor, validation tiers, and execution reminders. `acceptance_deny`
globs render as pre-execution constraints (validation does not parse actual
changed paths, so it never claims a deny rule was verified).

**Real fix:** Native multi-path / path-scoped baselines in git-control-tower and
arbitrary path-scoped suites in test-genie. When they exist, Plan Manager can
promote the repo/path diffs from informational to oracle. This is deferred
substrate work, **not** a Plan Manager blocker.

**Owner:** unassigned (cross-scenario: git-control-tower + test-genie).

**Refs:** `api/internal/validation/service.go` (`deriveScope`, `isOracleCommand`);
`api/internal/planmodel/boundary.go` (`BoundaryAnchorCommands`);
[`../concepts/PLAN-MODEL.md`](../concepts/PLAN-MODEL.md) (Change Boundary).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
