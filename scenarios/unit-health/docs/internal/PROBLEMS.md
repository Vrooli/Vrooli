# Problems — Unit Health

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

### 2026-09-09 — Maturity quality codes reconciled

**Symptom:** The maturity spec contained scenario-level quality codes that were not all observable.

**Root cause:** The typed test-quality catalog is the source of truth for per-test observations. Useful rollups were missing, while some semantic claims could not be established by the available static evidence.

**Workaround:** Read `test_quality.results` for per-test detail and the bounded scenario rollup for workspace-level triage.

**Real fix:** Completed on 2026-09-09. `focused-test` and `skip-declaration` violations emit `TEST_SKIPPED_OR_ONLY`; `requirement-link` violations emit `TEST_UNTAGGED_REQUIREMENT`; non-observable legacy quality codes were retired from the spec and documentation.

**Owner:** unit-health-improve (`declared-vs-emitted` row).

**Refs:** `api/internal/validation/analyze.go`, `api/internal/validation/quality.go:16-18`, `.vrooli/test-genie.json`.

### 2026-09-09 — Calibration comparison runs outside the CLI manifest

**Symptom:** `test-quality-reference` remains a developer-only generator under `api/cmd/`, while calibration now has a governed RPC and CLI path.

**Root cause:** The CLI is thin over `ValidationService`; a command exists only as an RPC. The calibration harness was built as a developer tool before the skill-set contract required governed sensors.

**Workaround:** Use `unit-health calibrate run --partition development` or `unit-health calibrate corpus`; the harness tests remain focused regression coverage.

**Real fix:** Completed on 2026-09-09. `ValidationService.RunCalibration` loads the corpus, reports comparisons, and is exposed as `unit-health calibrate run` with the inventory command `unit-health calibrate corpus`.

**Owner:** unassigned (W1 obligation against unit-health).

**Refs:** `api/handlers/validation/calibration.go`, `cli/manifest.json`, `docs/internal/TESTING.md` §"Test-quality calibration".

### 2026-09-09 — No reviewed holdout partition exists

**Symptom:** Every rule in the catalog lists `reviewed-holdout` and `false-positive-budget` among its promotion prerequisites, and `holdout.go` hard-codes `PromotionAllowed = false`. No holdout label file exists and no false-positive budget value is recorded, so no rule can ever leave `advisory`.

**Root cause:** The corpus discipline was built (development-versus-holdout partition rejection, forbidden-observation lists) but the first holdout was never authored.

**Workaround:** None; rules stay advisory and never fail the `unit` phase.

**Real fix:** Author a reviewed holdout of at least `num[threshold]:30` cases for a rule (assertion-observation on the Go profile is the candidate), labelled before observations are inspected; set the budget in the catalog; run the comparison; record a dated owner decision. Tracked by the `holdout-agreement` and `rules-promoted` rows.

**Owner:** unit-health-improve.

**Refs:** `api/internal/testquality/calibration/holdout.go`, `api/internal/testquality/catalog.json`.

### 2026-09-09 — Sampled review labels are computed from metadata only

**Symptom:** `unit-health.test-quality-sample` sends only test id, framework, test kind, and static status to the AI Gateway and asks for labels such as `weak_oracle` or `missing_negative_case`. A model cannot judge an oracle it cannot see; the labels are triage order, not review evidence.

**Root cause:** The privacy invariant (never send source bodies) was applied by sending nothing, rather than a bounded, gated excerpt.

**Workaround:** Use the program for deterministic cohort selection and treat `observations` as advisory. Holdout labels remain the only path to calibration.

**Real fix:** Completed on 2026-09-09. `ValidationService.ReadTestBody` returns a redacted test-body excerpt with a hard 4096-byte ceiling and privacy-pattern refusal. `unit-health.test-quality-sample` sends those excerpts through the governed AI Gateway path and reports refused rows as `insufficient_context`; labels remain advisory and holdout labels remain the calibration authority.

**Owner:** unit-health-improve (`reviewed-evidence` row).

**Refs:** `.vrooli/program-runtime/test-quality-sample.py` `step_classify`, `api/internal/adapters/gotest/body.go`, `api/handlers/validation/handler.go`, `packages/proto/schemas/unit-health/v1/validation/validation.proto`.

### 2026-09-09 — Mutation signal is now governed

**Symptom:** The improve board had a mutation-signal row but no owner-side generator or runner, so it could not distinguish killed, surviving, invalid, equivalent, out-of-contract, infrastructure-failure, and unknown outcomes.

**Root cause:** Mutation classification existed before a disposable workspace and bounded execution path was wired to it.

**Workaround:** None; the row remained pending telemetry.

**Real fix:** Completed on 2026-09-09. `RunMutationPilot` and `unit-health mutation pilot` use deterministic `go/ast` operators, cache-only copies, the bounded executor, and a governed program. Two comparable 20-mutant runs derived the 0.95 setpoint floor.

**Owner:** unit-health-improve (`mutation-signal` row).

**Refs:** `api/internal/testquality/mutation/operators.go`, `api/internal/testquality/mutation/workspace.go`, `api/handlers/validation/handler.go`, `.vrooli/program-runtime/mutation-pilot.py`, `skills/unit-health-improve/SKILL.md` §2/§4.

### 2026-09-09 — Testutil finding and projection check disagree about the cli workspace

**Symptom:** A self-validation reports `TEST_UTIL_MISSING` for the `cli` workspace ("6 test files; no testutil/ package found") while the projection check `cli testutil.root` for `cli/internal/testutil` passes in the same response.

**Root cause:** The projection check correctly saw the declared directory, but the architecture analyzer intentionally counted only a non-test Go file as a test-utility package. The directory contained only its import-ban guard, so the finding was correct for an empty root.

**Workaround:** None. The CLI now has a shared stdout-capture helper consumed by the smoke tests.

**Real fix:** Completed on 2026-09-09. The shared helper makes the root a real package, C081 pins the analyzer behavior, and self-validation no longer reports `TEST_UTIL_MISSING`.

**Owner:** unit-health.

**Refs:** `api/internal/validation/architecture.go:149`, projection checks in `api/internal/adapters/`.

### 2026-09-09 — Test Genie does not declare unit-health as a dependency

**Symptom:** Test Genie shells its `unit` phase to `unit-health validate scenario <name> --execution --json`, but `scenarios/test-genie/.vrooli/service.json` lists no provider scenario under `dependencies.scenarios` (only agent-inbox and agent-manager). The skill-set applicability trigger "multiple scenarios depend on it" cannot see the relationship.

**Root cause:** Test Genie's provider scenarios are discovered through phase descriptors, not the dependency graph, and none of them is declared.

**Workaround:** The improve role is selected by operator request (2026-09-08), which does not need the trigger.

**Real fix:** Test Genie declares its validation providers (unit-health, quality-health, and the others) as optional scenario dependencies in one change. This is Test Genie's edit, not this scenario's.

**Owner:** test-genie.

**Refs:** `scenarios/test-genie/docs/phases/unit/README.md`, `.vrooli/test-genie.json`.

**Resolution update (2026-09-09):** Unit Health now declares Test Genie and
Agent Manager as optional `try_start` consumers for its composed setpoint child
programs. Test Genie's own provider-dependency declaration remains an owner-side
follow-up and is not edited by this scenario.

### 2026-09-09 — Calibration corpus retains explicit retired cases

**Symptom:** The calibration denominator contains cases that cannot be assessed
by the currently supported static or native profiles.

**Root cause:** The case specification is broader than the evidence kinds owned
by the current adapters; deleting those identities would shrink the denominator.

**Workaround:** Use `unit-health calibrate corpus` and inspect each non-empty
`retired_reason`; retired cases remain in the specified denominator and do not
count as implemented.

**Real fix:** Completed on 2026-09-09 for the current corpus. The `num[sot]:81` identities
remain in `case-specification.json`; 42 are implemented and 39 are retained with
explicit reasons. A new adapter or evidence kind may move a case from retired to
implemented only through a new calibration record.

**Owner:** unit-health-improve.

**Refs:** `api/internal/testquality/testdata/case-specification.json`,
`api/internal/testquality/testdata/development.json`, `unit-health calibrate corpus`.

### 2026-09-09 — Docs gate retains an external command defect and partial CLI metadata

**Symptom:** `vrooli scenario test unit-health --phases docs` stays at L0 with
`broken_command_snippet` for `test-genie registry build` in `num[sot]:4` locations.
The latest run also reports `num[sot]:16` informational `partial_command_snippet`
observations because the CLI metadata cannot expose the argument schema for
`vrooli scenario status unit-health --json`.

**Root cause:** The snippet validator grades against CLI manifests. The actual
owner command `test-genie registry build --scenario scenarios/unit-health -q`
exits successfully and regenerates the tracked registry, but the validator's
manifest view rejects the valid `build` positional. The CLI metadata for
`vrooli scenario status` is also incomplete. The prompt-manager form was not
reproduced by the latest run. Neither issue is a unit-health behavior defect.

**Workaround:** Keep the valid owner command documented and use the direct
scenario-scoped invocation while the validator metadata is repaired. The
status snippets are real commands and are retained while their argument schema
remains unavailable. The latest run has no missing-link, placeholder, or
content-contract findings.

**Real fix:** Test Genie and knowledge-observatory owners should repair the
command registration/manifest and schema publication. The Test Genie issue is
filed as scenario-qa bug `knw-1788932400803018741`; the schema issue is filed
as `knw-1788951916709719458`. The earlier prompt-manager observation remains
recorded as `knw-1788932450678799631` but was not reproduced.

**Owner:** test-genie and knowledge-observatory.

**Refs:** command exit 0; run `20260909-110950-4eb02a27`; prior run
`20260909-054102-236d24a9`.

### 2026-09-09 — Full closeout retains only filed external validator observations

**Symptom:** The final executed validation is passed and at the top local rung.
The composed board still keeps `unknown-share` and `fleet-adoption` out of band.
The full Test Genie owner-phase run passes unit, programs, skill-set, proto, and
structure; its docs phase retains only the filed external validator
observations permitted by the plan.

**Root cause:** The reviewed Quality Health native profile intentionally emits
the ESLint-owned syntax checks but not `skip-declaration`; Unit Health
therefore needed a calibrated source-bound Vitest projection for skip-only,
non-empty, and conditional declarations. That projection now makes the live
skip rule clean for the current UI corpus. The repository-wide documentation
validator now reports no missing-link, placeholder, content-contract, or
number finding. The command-snippet finding for `test-genie registry build`
and the status-schema partials are the filed external observations; the
prompt-manager form is retained in the ledger historically but was not
reproduced by the current validator run.

**Workaround:** Treat the remaining out-of-band readings as advisory and start
the next improvement cycle with fleet child-receipt evidence. Do not convert
missing child evidence or unknown observations into zeroes.

**Deferral:** Deferred on 2026-09-09 to a follow-up child-receipt repair and
external-validator repair owned by the respective owners. This plan records
the evidence and does not lower thresholds or edit external manifests. The
owner-side low-coverage findings and the
skip-declaration source projection were repaired with focused tests; the final
owner validation reports zero blocking findings and `skip-declaration`
unknown-share 0.0; the docs phase remains qualified only by the filed
external observations above.

**Owner:** unit-health for coverage and execution completeness; test-genie and
knowledge-observatory for the command-validator defects; documentation owners
for any future docs-quality regression.

**Refs:** `evidence-2026-09-09/self-validation-final.json`,
`evidence-2026-09-09/test-genie-final.txt`,
`knw-1788932400803018741`, `knw-1788951916709719458`,
`knw-1788932450678799631`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Starter-domain residue (`notes` CRUD examples from the react-vite template) across the docs tree | Resolved 2026-09-09: every doc now describes the real `validation` + `health` + `runhistory` + `testquality` contract; `cli-commands.md`, `maturity.md`, `api-endpoints.md`, `DOMAINS.md`, `DATA.md`, `FLOWS.md`, `START-HERE.md`, `QUICKSTART.md` and the lighter mentions were rewritten. Kept here as the record that the drift existed. | none remaining | done |

_2026-06-17: `SEAMS.md` and `ARCHITECTURE.md` (the deferred hardening files) were fully rewritten from the `notes` starter domain to the real `validation` + `health` + `runhistory` + `discovery`/`executor` seams. While doing so it surfaced that the same `notes` starter residue pervades the rest of the docs tree (row above) — a larger cleanup than that hardening pass, tracked here rather than silently half-done._

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
