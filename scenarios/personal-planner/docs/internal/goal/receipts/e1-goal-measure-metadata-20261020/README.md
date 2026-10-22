# E1 measurable goal metadata slice receipt

Directive: `ORCH-E1-GOAL-MEASURE-METADATA-20261020`  
Worker task/run: `a1a95f2b-b012-43a9-b4f7-c10fddb046be` / `655ab3a8-da4f-4d1e-a859-f3159b085491`  
Parent: `5d07e651-eaa6-48ec-89b4-5dc359059193`  
Effort: `personal-planner-I18`

## Behavior delivered

- Goal domain and SQLite now retain an optional typed currency measure: amount in USD minor units, currency unit/code, `after_costs_before_tax` basis, one-time/once or recurring/month period, and the goal's target date. Goal and measure inserts share one transaction.
- `GET/POST /api/v1/goals` carries that representation for create/read. Measured-goal responses report `progress_state: unknown` and omit `progress_basis_points`; manual progress updates for measured goals are rejected until observation support exists. Generic goals continue to expose and update their prior progress values.
- Supported CLI `goals create` and `goals list` use the scenario REST representation and show measure, basis, recurrence, target date, and unknown progress. Other goal progress/milestone operations remain on their current Connect routes.
- Goal creation UI accepts and has create/read regressions for both one-time and recurring monthly USD targets and target date. Read cards show its target meaning and an explicit unknown-progress state; they do not show the generic percent slider for measured goals.

No income observations, supporting evidence, provider integration, forecast changes, Proto changes, generated artifacts, or product-data writes were made. Test storage is in-memory SQLite; UI transport is mocked; CLI transport uses a deterministic local HTTP test server.

## Baseline and current identities

- Source HEAD at baseline: `52097258290d1bff0191319242d4948e214af9e3`.
- Exact admitted-path baseline: `baseline-original/manifest.json`, SHA-256 `16f8fd4b0fe9579819e2ffd901311f3f8085c4dfa3ea06772788d0b70e4a7d98`. It includes exact source bytes/hashes and the explicit absence of the new CLI handler test. These bytes include earlier dirty Planner goal target-date work and are the correct comparison point for this slice.
- Current path hashes, byte counts, line counts, and baseline deltas: `current-identities.txt`.
- Net admitted-path delta: +288 runtime lines and +332 test lines. This exceeds the slice estimate of +180 runtime/+140 tests because the generated goal Connect message is out of bounds, requiring the existing UI/CLI create/read to use a scenario REST representation. The admitted file and feature boundary did not widen. The authoritative worker epoch-check reports cumulative growth of +1,706 runtime and +2,145 test lines against the +2,100 runtime ceiling, leaving 394 runtime lines. The measured pre-slice runtime growth is +1,418; the epoch-check did not step back on growth. Test-line growth exceeds the epoch's 2,100 expected-size comparison by 45 lines and is disclosed; the worker-card growth trigger is runtime-line based. No further growth is claimed or authorized by this receipt.
- The required E1 ledger append was separately snapshotted before that write: `epoch-ledger-prewrite/manifest.json`, SHA-256 `03e9a643fb53be7aea6857c8cc95ae5a91b83f4224f4ecd9745459822af82cdd`.

## Focused validation

Full command outputs are retained beside this receipt:

- `api-domain-handler.log`: `cd scenarios/personal-planner/api && go test ./internal/goals ./handlers/goals -count=1` — PASS.
- `cli-goals.log`: `cd scenarios/personal-planner/cli && go test ./domains/goals -count=1` — PASS. The HTTP transport test creates both examples, independently reads them, checks unknown progress and checks CLI rendering.
- `ui-goals-tests.log`: `cd scenarios/personal-planner/ui && pnpm exec vitest run src/api/goals.test.ts src/pages/GoalsPage.test.tsx` — PASS, 2 files / 18 tests.
- `ui-typecheck.log`: `cd scenarios/personal-planner/ui && pnpm exec tsc --noEmit` — PASS, exit 0 with empty stdout/stderr.
- `cli-manifest-check.log`: manifest JSON parses — PASS.
- `cli-evidence-prewrite/` and `cli-evidence-current.txt`: exact original/current scenario CLI primitive-evidence artifact identities. It was refreshed through `UPDATE_CLI_EVIDENCE=1 go test . -run TestPrimitiveEvidenceArtifactCurrent -count=1`; only its manifest hash and `goals create`/`goals list` primitive identities changed. `cli-primitive-evidence.log` then passes both artifact freshness and primitive consistency tests. No `packages/proto` or Proto-generated output was changed by this slice.
- `scoped-diff-check.log`: scoped `git diff --check` — PASS, exit 0 with empty output.
- `epoch-check.txt`: required worker-card epoch check and its unmodified output.

## Remaining acceptance gaps

- Independent review is pending; this slice and E1 are not accepted.
- E1's full calendar/planning journey, seven-phase scenario gate, visual criteria, storage/backup qualification, and other listed acceptance gates remain open and unchanged.
- Existing generated Connect `GoalsService` messages cannot carry or return measure metadata under the no-Proto boundary. Existing generic Connect goal behavior remains available; the selected UI/CLI measured-goal create/read path uses the scenario REST representation. Connect readback of the new measure is therefore not claimed.
- This slice represents target definitions only. Actual progress and evidence remain unknown; it does not establish income or measure outcomes. Only USD is currently accepted by this representation.
