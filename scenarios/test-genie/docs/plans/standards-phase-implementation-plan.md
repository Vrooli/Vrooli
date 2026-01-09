# Standards Phase (Opt-Out) — Implementation Plan

## Context (What Exists Today)

- **Test-genie phase registry & ordering**: Go-native phases are registered in `scenarios/test-genie/api/internal/orchestrator/phases/catalog.go` (`NewDefaultCatalog()`).
- **Default presets**: `scenarios/test-genie/api/internal/orchestrator/suite_execution.go` (`defaultExecutionPresets`).
- **CLI phase allowlist**: `scenarios/test-genie/cli/internal/phases/phases.go` (`AllowedPhases`).
- **Scenario-auditor CLI wrapper**: `scenarios/scenario-auditor/cli/scenario-auditor` provides `audit --standards-only --json`, and can auto-start the scenario-auditor service.
  - **Known issue for machine parsing**: `audit --json` currently prints an additional JSON payload before the final summary JSON, which makes it awkward to consume programmatically.

## Goal

Add a new Test Genie phase named **Standards** (canonical key: `standards`) that runs scenario-auditor standards rules as part of `test-genie execute`, **enabled by default** (opt-out via `--skip standards` or per-scenario config).

## Proposed Default Semantics (Confirm Before Coding)

- **Phase name**: `standards` (display: “Standards”).
- **Scope**: standards-only (no security scan) by default.
- **Ordering**: place **immediately after `structure`** (fast fail on standard/config issues).
- **Default timeout**: 60s (overrideable via per-phase timeout config).
- **Default failure threshold**: fail on `critical` or `high` findings (configurable).

## Implementation Steps (Detailed Checklist)

### 0) Confirm rollout + UX details

- [ ] Confirm phase ordering: after `structure` (recommended) vs after `docs`.
- [ ] Confirm which presets include `standards` by default:
  - [ ] Option A (true opt-out): add to `quick`, `smoke`, and `comprehensive`.
  - [ ] Option B (lighter default): add to `smoke` and `comprehensive` only.
- [ ] Confirm gating policy:
  - [ ] Default: fail on `critical|high`.
  - [ ] Alternate: fail on `critical` only for initial rollout.

### 1) Make `scenario-auditor audit --json` reliably consumable

**Goal**: One command, one JSON object on stdout.

File: `scenarios/scenario-auditor/cli/scenario-auditor`

- [ ] Adjust `cmd_audit()` so that when `--json` is specified:
  - [ ] Do **not** print the “combined” JSON block to stdout (currently emitted before summary JSON).
  - [ ] Emit **only** the final summary JSON payload to stdout (currently printed later).
  - [ ] Keep progress logs on stderr (they already are).
  - [ ] Preserve `--all` behavior for humans (full payload).
- [ ] Validate exit codes:
  - [ ] `audit` returns non-zero when the standards job fails.
  - [ ] `audit` returns distinct non-zero (or at least non-zero) when it times out.

Acceptance tests (manual):

- [ ] `scenario-auditor audit test-genie --standards-only --timeout 60 --json | jq .` succeeds and parses.
- [ ] `scenario-auditor audit test-genie --standards-only --timeout 0 --json` fails non-zero (timeout path) and still prints parseable JSON if possible (optional).

### 2) Add the new Go-native phase to test-genie

#### 2.1 Define canonical phase name

File: `scenarios/test-genie/api/internal/orchestrator/phases/types.go`

- [ ] Add a new phase constant: `Standards Name = "standards"`.

#### 2.2 Implement the phase runner

File (new): `scenarios/test-genie/api/internal/orchestrator/phases/phase_standards.go`

Behavior:

- [ ] Execute scenario-auditor via CLI:
  - [ ] Command: `scenario-auditor audit <scenario> --standards-only --timeout <seconds> --json`
  - [ ] Ensure runtime is bounded by the phase timeout (use `exec.CommandContext`).
- [ ] Parse the JSON summary payload and convert into Test Genie observations:
  - [ ] Total violations, highest severity
  - [ ] Severity distribution (by_severity)
  - [ ] Top rule aggregates (by_rule) and/or top violations
  - [ ] Artifact path (if provided)
- [ ] Determine pass/fail from parsed summary:
  - [ ] Fail if `highest_severity` >= configured `fail_on` (default: high)
  - [ ] Fail if the command returns non-zero (job failed/timeout)
- [ ] Failure classification + remediation:
  - [ ] Missing `scenario-auditor` binary → `missing_dependency`, remediation: install/ensure `scenario-auditor` CLI exists and scenario can start.
  - [ ] Timeout → `timeout`, remediation: increase phase timeout or reduce scan scope.
  - [ ] Standards violations above threshold → `misconfiguration`, remediation: rerun targeted scan and fix violations.

Notes:

- [ ] Keep phase deterministic and file-based (no external network calls beyond local services).
- [ ] Ensure stdout/stderr from the auditor are captured in the phase log file.

#### 2.3 Register the phase in the catalog

File: `scenarios/test-genie/api/internal/orchestrator/phases/catalog.go`

- [ ] Register `Standards` with:
  - [ ] Weight/order: after `structure` (recommended).
  - [ ] Default timeout: 60s (or slightly higher if needed).
  - [ ] Description: “Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config).”
  - [ ] Optional: should be **false** (since this is opt-out).

#### 2.4 Include in presets (opt-out)

File: `scenarios/test-genie/api/internal/orchestrator/suite_execution.go`

- [ ] Add `"standards"` to `defaultExecutionPresets` per confirmed strategy:
  - [ ] quick preset: include or exclude (per decision)
  - [ ] smoke preset: include
  - [ ] comprehensive preset: include

#### 2.5 Update CLI allowlist and selection

File: `scenarios/test-genie/cli/internal/phases/phases.go`

- [ ] Add `"standards"` to `AllowedPhases`.

### 3) Configuration knobs (minimal but useful)

Start with environment variables (fast, avoids schema churn), and optionally expand later.

- [ ] `TEST_GENIE_STANDARDS_FAIL_ON` (default `high`)
- [ ] `TEST_GENIE_STANDARDS_LIMIT` (default `20`)
- [ ] `TEST_GENIE_STANDARDS_MIN_SEVERITY` (default `medium` for display)

Also support per-scenario opt-out via existing testing config mechanism:

- [ ] `.vrooli/testing.json`: `{"phases":{"standards":{"enabled":false}}}`

### 4) Automated tests (avoid regressions)

Test-genie:

- [ ] Add unit tests for:
  - [ ] Parsing valid summary JSON into observations
  - [ ] Failing when highest severity exceeds threshold
  - [ ] Handling missing `scenario-auditor` binary (classification + remediation)
- [ ] Update existing suite execution tests that assert phase lists:
  - [ ] `scenarios/test-genie/api/internal/orchestrator/suite_execution_test.go` (expected phase ordering includes `standards`).
- [ ] Update CLI parsing tests:
  - [ ] `scenarios/test-genie/cli/commands_test.go` (ensure `--phases standards` is accepted).

Scenario-auditor:

- [ ] Add/adjust a small CLI behavior test if one exists (or manual verification) to ensure `audit --json` is single JSON on stdout.

### 5) Documentation updates

Test Genie docs (update “10-phase” → “11-phase”):

- [ ] `scenarios/test-genie/docs/phases/README.md` (insert Standards into pipeline, update tables).
- [ ] `scenarios/test-genie/README.md` (update phase list).
- [ ] `scenarios/test-genie/docs/reference/presets.md` (update phase lists per preset).

Add a short “Opt-out” snippet:

- [ ] Include in docs: `test-genie execute X --skip standards`
- [ ] Include in docs: `.vrooli/testing.json` phase disable example

### 6) Rollout guidance (practical ops)

Because this is opt-out, it will surface pre-existing violations immediately.

- [ ] Decide if initial gating should be `critical` only for a short grace period.
- [ ] Provide a bulk opt-out recipe for legacy scenarios (documentation only; avoid mass-edit scripts unless explicitly requested).
- [ ] Consider a global phase toggle entry in `.vrooli/test-genie-phase-toggles.json` as an emergency brake (optional; depends on desired governance posture).

## Validation Commands (Post-Implementation)

- `cd scenarios/test-genie && make build`
- `test-genie execute test-genie --phases standards --fail-fast`
- `test-genie execute <scenario> --preset smoke --skip standards` (ensures opt-out works)
- `scenario-auditor audit <scenario> --standards-only --timeout 60 --json | jq .`

## Deliverables

- New `standards` phase runner registered in test-genie
- Defaults updated so `standards` is included in presets (opt-out)
- Scenario-auditor CLI emits parseable JSON for `audit --json`
- Tests and docs updated accordingly
