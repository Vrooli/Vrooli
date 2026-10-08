# Testing — Unit Health

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Common patterns and anti-patterns

| ✅ DO | ❌ DON'T |
|---|---|
| `mocks.FakeClock` for time-dependent assertions | `time.Sleep(150 * time.Millisecond)` then assert on a fuzzy match |
| A live server for streaming/socket behavior; a recorder for request/response mapping | Claiming a recorder proves transport behavior |
| Role/name queries for semantic controls; shared IDs where no suitable role exists | Coupling behavior tests to incidental editorial copy |
| `vi.mock("./api/health", async (importOriginal) => …)` inline at top of file | Helper-wrapped `vi.mock` (TDZ at hoist time) |
| `makeHealthResponse({ status: "degraded" })` for variants | Hardcoded literal payload in different tests |
| Per-method error knob (`PingErr error`) on fakes | Single global "fail mode" boolean across the fake |
| `var _ Pinger = (*sql.DB)(nil)` to lock the contract at compile time | Runtime "does this satisfy" check in init |
| Adding new seams to [`SEAMS.md`](SEAMS.md) at the same commit | "We'll document it later" |

## Scenario skill set and programs

The scenario declares its skills in `.vrooli/service.json` and owns its Program
Runtime contracts under `.vrooli/program-runtime/`:

| Artifact | Role | Read it when |
|---|---|---|
| `skills/unit-health/SKILL.md` (`prompt-manager skill read unit-health`) | usage | You need to read a scenario's assessment without over-claiming, choose plan-only versus executed reads, or route a repair |
| `skills/unit-health-improve/SKILL.md` (`prompt-manager skill read unit-health-improve`) | improve | Your task is Unit Health itself: the rule catalog, corpus, holdout, adapters, or the maturity contract |
| `unit-health.setpoint-read` | improve sensor board | `program-runtime library run unit-health.setpoint-read`; rows with no governed sensor carry their permanent reason instead of a reading |
| `unit-health.test-quality-sample` | bounded cohort selection | Deterministic review ordering for a scenario's tests; advisory only |
| `unit-health.mutation-pilot` | bounded mutation signal | Disposable owner-side Go mutations; valid killed/survived receipts only |

Program fixtures and their pytest doubles live under
`.vrooli/program-runtime/tests/`; run them with
`python3 -m unittest tests.test_setpoint_read tests.test_quality_sample` from
`.vrooli/program-runtime/`. Test Genie's `programs` and `skill-set` phases
validate the declarations and executed fixtures:
`vrooli scenario test unit-health --phases programs,skill-set`.

## Cross-references

- **Scenario flows**: [`FLOWS.md`](../concepts/FLOWS.md).

### Test-quality calibration

The canonical [authoring guide](../../../../docs/testing/UNIT-TEST-AUTHORING.md)
distinguishes behavioral evidence from static observations. Source keywords,
test names and `NoError` do not prove edge-case coverage. The versioned
[rule reference](../reference/test-quality-rules.md) is generated from the catalog.

From `api/`, run the focused harness tests with
`GOWORK=off go test ./internal/testquality/...`. The fixture inventory is
`internal/testquality/testdata/case-specification.json`; it retains all 81
authored cases. `development.json` owns parser expectations, `native-expected.json`
owns version-specific runner expectations, and `context-cases.json` owns
structured owner/reference inputs. Fixture availability is not certification.

The governed comparison command is `unit-health calibrate run --partition
development`; `unit-health calibrate corpus` returns the inventory, and the
committed reviewed comparison uses `--partition reviewed-holdout --holdout
assertion-observation-go-v1 --rule assertion-observation`. Missing
observations remain unknown; the RPC never derives results from expected
outcomes. The development corpus holds `num[sot]:42` of the `num[sot]:81` specified cases; the
remaining cases are either explicitly retired with a reason or remain
separately governed by native/context evidence. `case-specification.json` is
planned coverage, not certification. Native fixtures run separately through
`node internal/testquality/testdata/run-native.mjs <installed-node_modules> <output.json>`.
The runner uses installed dependencies offline in a disposable copy. Intentional
test failures are compared by exact identity and failure, not exit code alone.

Development fixtures must not be relabeled as reviewed holdouts. The first
reviewed holdout is `assertion-observation-go-v1`: `num[sot]:30` Go tests from
the selected scenarios, labels timestamped before observations, and a 0.05 false-positive
budget in the catalog. The current comparison is 30 labelled and observed,
zero unknown, within budget, and `promotion_allowed=false`; the catalog decision
is `requested`, so enforcement remains advisory. A matching result is scoped to
its version and claim, not proof of overall adequacy or permission to promote a
rule.

### Bounded sampled review

Use the read-only Program Runtime contract `unit-health.test-quality-sample`
for semantic review prioritization. Supply an exact scenario, workspace, source
identity, seed, and sample size. The contract sorts candidates deterministically,
prioritizes findings and unknowns, and keeps clean controls in the denominator.
A partial or unavailable cohort is not empty evidence. With `review_ai=true`,
the program calls `unit-health/validate/test-body` for each selected row. The
owner returns only a redacted test-body excerpt, capped at `num[threshold]:4096` bytes. A privacy
pattern or unavailable binding keeps the row in the denominator and reports
`insufficient_context`. AI labels are closed-vocabulary advisory observations;
they require an independent holdout label before calibration.

The contract emits one bounded envelope with policy version, seed, source
identity, denominator, selected rows, excluded count, controls, and limitations.
It never includes a full source file or an unredacted credential. Provider
routing must use the governed AI Gateway typed classification path. Labels stay
advisory and do not establish behavioral adequacy.

Attach a sampled cohort to an executed receipt with
`--reviewed-cohort <seed> --reviewed-source-identity <identity>
--reviewed-observation-count <observed>`. The response reads
`evidence_stages.reviewed=supplied` only when all required values identify at least
one observed label. The setpoint board then reads `reviewed-evidence` in band.

Holdout labels are supplied independently to the governed calibration RPC after
rule and prompt versions are
frozen. The report retains missing, stale, malformed, and uncertain rows as
unknown and never promotes a rule automatically.

Mutation pilots run with `unit-health mutation pilot <scenario> --package <p>
--seed <seed>` or the governed `unit-health.mutation-pilot` program. The owner
copies the workspace below the scenario cache prefix, applies one `go/ast`
operator per copy, and removes each copy after the bounded executor returns.
Record the owner run and source identity. Classify each receipt as killed,
survived, invalid, equivalent, out_of_contract, infrastructure_failure, or
unknown. Only valid killed/survived receipts contribute to `kill_rate`; a
surviving valid mutant is a review signal and does not establish a defect or
change the shared tree. The current comparable floor is 0.95 from two 20-mutant
runs recorded in `unit-health-improve` §4.

### Scoped self-test classifications

`AppShell.a11y.test.tsx` awaits `expectNoA11yViolations` through the local
test-utils re-export of the shared api-base helper. The helper owns the axe
assertion. Do not add a duplicate local assertion merely to satisfy source
inspection. Native assertion observation is limited to the supported installed
Vitest version; it does not prove all accessibility behavior.

`TestExecutorHelperProcess` is a child-process entrypoint, not an independent
behavior test. The Go adapter requires both its opt-in environment guard and a
matching typed launcher before classifying it as not applicable. Parent executor
tests assert the child output, exit status, cancellation and capture behavior.
An unmatched helper-looking function receives no such exemption.

Type declarations, including the `commonStage` alias in handler tests, are not
runtime test cases. The adapter's type-declaration regression requires the actual
behavior test to remain in the inventory while excluding aliases and structs.

`TestEndToEndGoWorkspaceThroughValidate` conditionally skips only when the Go
toolchain is unavailable. `repoRootForTest` conditionally skips when the shared
companion registry is unreachable. Native skipped outcomes are not passes.
Neither prerequisite check is permission for unconditional placeholder tests.
The separate Go skip-declaration calibration covers retained C013/C014: a sole
direct skip statement is a placeholder violation, while a conditional declaration
remains unknown/not-executed. This rule is advisory and does not infer runtime
outcomes. The Vitest C034 skip-only fixture is covered by the source calibration
profile; delegated and dynamic declarations remain outside that profile, and
an absent row is not a clean skip assessment.

The companion-registry anti-drift test compares current shared-package exports
with `.vrooli/test-companions.json`. Its explicit regeneration mode intentionally
fails after writing: review the changed export inventory, then rerun the gate
without regeneration enabled. Do not remove exports or weaken the test to hide
projection drift.

### Requirement traceability evidence

Validation reads current declarations from Test Genie's
`GET /api/v1/scenarios/<name>/requirements?view=registry`. This view uses the
maintained import discovery/parser, rejects partial registries, and does not
borrow cached execution status. Old server response shapes, malformed registries
and unavailable owners remain unknown. Integration/business-only responsibilities
do not create artificial unit-tag obligations; unspecified phases remain unknown.

The native Vitest handoff uses the existing reporter's tag extractor, including
suite inheritance, comma-separated IDs and configured patterns. The shared Go
test-tag parser is checked against the reporter's default grammar. Go declaration
comments and literal subtest names produce static links, not passing observations.
Tags from a parent apply to its subtests; sibling-only tags do not leak.

Supported Go execution uses `go test -json -count=1`, with an independent bounded
8 MiB capture rather than the 8 KiB display tail. The adapter matches package and
test identity and keeps pass, fail, skip and unknown separate. Overflow, missing
run events, ambiguous rewritten subtest names and unsupported command forms do
not establish current passes. Captured bytes are ephemeral; persisted trace links
retain command identity. Cached assessments cannot become current-run evidence.

`TestRetainedRequirementContextCalibration` checks the six retained requirement
contexts. Focused native Go, reporter, service, protobuf and CLI tests validate
their separate boundaries. These checks establish linkage and observed outcomes,
not behavioral adequacy or requirement completion. Test Genie remains the owner
of requirement status derivation.

- **Seams definition + adding new seams**: [`SEAMS.md`](SEAMS.md).
- **Skill bundle for testing-related work** (load before substantial test changes):
  ```bash
  prompt-manager skill read seam-discovery-and-enforcement
  prompt-manager skill read test
  prompt-manager skill read unit-testing-architecture-steer
  ```
- **Test runner used by CI and `vrooli scenario test`**: see
  `.github/workflows/test.yml` and `packages/cli-core/cmd/scenario_test.go`.
- **Why no inline mocks in `*_test.go` files**: the testutil package
  is the single source of fake behavior. Inline mocks in tests
  fragment the contract; when the interface grows a method, every
  inline mock has to be updated. One mock in `mocks/`, one update.
