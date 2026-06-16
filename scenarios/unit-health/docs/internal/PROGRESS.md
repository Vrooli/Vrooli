# Progress — Unit Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-16 | agi | done | Phase 0 + Phase 1 (product contract). Generated from react-vite. Authored PRD, 5 requirements modules, `.vrooli/maturity.json` (L0–L5 + 24 finding codes), `docs/reference/maturity.md`, planned-domain map in DOMAINS.md, service.json deps. API builds green. Example `notes` domain intentionally left in place (compiling scaffold) — see handoff below. |
| 2026-06-16 | agi | done | Phase 2 (Proto/API/CLI foundations). Authored `unit-health/v1/validation/validation.proto` (`ValidationService.ValidateScenario`, full domain message set: TestSurface/TestWorkspace/ExecutionPlan/CommandResult/CoverageTarget/ValidationFinding/Diagnostic/MaturitySummary/ValidationCounts + `common.v1.MaturityAssessment`); regenerated proto (Go/TS/Py + descriptor). Deleted `notes` proto. Swapped `notes`→`validation` across API (`handlers/validation` + `internal/validation` skeleton service + registry + main.go, dropped the measures block), CLI (`domains/validate`, manifest `validate scenario`, gen-endpoints seed), and UI (`api/validation.ts` + `features/validation/ScenarioValidationWorkbench`, deleted `features/notes`/`api/notes.ts`/`NotesPage`). Added `maturity-go` dep to api/go.mod; tidied cli/go.mod. Added missing `RESTExceptionPayloads` type + health-endpoint `proto_payloads` (template was older than quality-health's). **example-domain-removed gate PASSES.** Validation: api+cli `go test ./...` green; UI lint+test+build green; `proto-health validate scenario unit-health` passed (L5, 0 err); `cli-health validate scenario unit-health` passed (L5, 0 err); live `unit-health validate scenario unit-health` (human + --json) returns a schema-valid, honestly-degraded stub assessment. |

## Phase 3 Handoff (next agent — start here)

**State:** Proto/API/CLI/UI foundations are landed and green. The `validation`
domain replaced the template `notes` example everywhere; the
`example-domain-removed` gate passes. `unit-health validate scenario <name>`
works end-to-end (human + `--json`) and returns a **schema-valid stub**: the
`internal/validation.Service.Validate` body currently returns a single honestly-
degraded Response with no findings (status `degraded`, maturity L0,
degraded_reason "engine not yet implemented"). Phase 0 parity checklist +
contract sketch: `~/.vrooli/plans/unit-health-PHASE0-handoff.md`.

**Where the engine plugs in:** `api/internal/validation/service.go`. The
`Response` struct already mirrors `validation.proto` field-for-field, and
`api/handlers/validation/handler.go` does a flat copy + builds the shared
`common.v1.MaturityAssessment` via `maturity-go/assessment.BuildProtoAssessment`
(finding `Code` → `.vrooli/maturity.json` mapping; coverage-dimension codes get
`FINDING_SOURCE_COVERAGE`, the rest `FINDING_SOURCE_STANDARDS`). So Phase 3+
work is almost entirely inside `service.go`: add seam fields (Code Facts
discoverer, executor, analyzers), populate `Surfaces`/`Workspaces`/`Plan`/
`Findings`/etc., and the wire/CLI/UI surfaces light up automatically.

**Phase 3 steps (per plan §7 Phase 3):**
1. Implement a Code Facts client (`code-facts facts describe scenario:<name>
   --include surfaces,parse_units --json`); degrade explicitly when unavailable.
   Model the client shape on `quality-health/api/internal/surfaces`
   (`CodeFactsClient`/`Discoverer`/`Locator` seam).
2. Build the target/workspace model + canonical-framework resolution (Go `go
   test`, React/Vite `vitest`, Python `pytest`→unittest degraded) and the
   degraded/noncanonical detectors (Jest, missing test/coverage script, pm
   mismatch, unsupported parse unit).
3. Produce the dry-run ExecutionPlan + discovery/config-gap findings. Add tests
   with fake Code Facts reports.

**Maturity-spec note:** `.vrooli/maturity.json` is loaded and parsed at module
startup via `assessment.ParseSpec` (in `handlers/validation/module.go`
`loadMaturitySpec`) and is already exercised live (proto-health/cli-health pass
L5; the stub assessment builds cleanly). Real current/next-level computation
arrives with real findings in Phase 5's maturity assessor.

**Known pre-existing scaffolding debt (NOT introduced by Phase 2; shared with
sibling quality-health):**
- Full `vrooli scenario test unit-health` STANDARDS phase is red (~32 high) —
  the fleet-wide freshly-generated-template standards campaign, not this work.
- `dependencies` phase: governance-review warnings for the scenario's deps
  (need recording in approved-dependency memory) — template baseline.
- `proto` phase WARNINGS only (errors=0 now): `errors`/`health` protos still
  carry `@template react-vite/example` (sibling does too); `errors` domain has
  no handler dir. Accepted scaffold state.
- `docs/manifest.json` nav still lists the old `notes` CRUD domain in a couple
  of entries (lines ~884/924/1416); the `bas/cases/routed-database/` case still
  references `@selector/notes.list`. These are doc/UI-test scaffolding — clean
  up when the UI is built out (Phase 6) and during doc passes; smoke/playbooks
  currently pass regardless.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
