# Portability truth chain III final validation — 2026-08-26

This report records the final executable evidence for the plan
`portability-truth-chain-iii-make-the-grid-falsifiable-make`. It distinguishes
implemented gates from repository-wide gates that remain red. The baseline
used for comparison is
[`portability-iii-baseline-2026-08-26.md`](portability-iii-baseline-2026-08-26.md).

## Result

The resource/grid projection, declaration contract, generated platform-support
document, and portability lint gates are implemented and locally verified. The
plan is not fully green: the repository-wide conformance gate reports 86 hard
findings and 72 warnings across 1794 targets, and the server-owned scenario
validation remains red on standing scenario debt. Those findings are not
silently converted into a pass.

## Twelve-measure comparison

| Measure | Baseline | Current evidence | Result |
|---|---:|---:|---|
| Capabilities | unavailable; plan audit expected 45 | 52 capabilities in `vrooli capability ledger --json` | improved and available |
| `no_work_required` situations | 38 (audit record) | 38 | unchanged |
| `real_peer_nobody_wired` situations | 1 (audit record) | 5 | improved |
| Policy cells that can express work | 126 `no_work_required`, 8 `no_equivalent_ever`, 0 other | 126 `no_work_required`; surviving `no_equivalent_ever` entries carry rationale/review metadata | contract preserved |
| Safeguard cells with circular evidence | 23 files | 0 matches | pass |
| Conformance targets | 598 | 1794 across six OS/architecture cells | expanded; gate still red |
| Modules failing non-Linux test cross-compile | 4 | full conformance still reports hard findings in unrelated dirty-tree modules | not closed |
| Unguarded shell-out sites | 19 plan-inventory sites | 25 syntactic findings, each dated in the allowlist | lint gate pass; inventory widened |
| Unguarded kernel-filesystem files | 53 plan-inventory files | 80 syntactic matches across 35 files, each dated in the allowlist | lint gate pass; inventory widened |
| Scenarios blocked/degraded by OS | 9 scenarios / 19 rows | 19 rows from `vrooli capability fleet --json` | preserved |
| Resources using `compose-service` | 2 | 0 | pass |
| Resources surfaced in the ledger | absent | 29 resource rows; 87 OS claims | pass |

The current ledger also reports a skip budget of 155, with Linux, macOS, and
Windows budgets all at 155, ratchet direction `down`, and
`last_run_within_budget: true`. The CLI uses canonical protobuf JSON names:
`acquisition_kind` and `skip_budget`.

## Executed validation

| Command or check | Evidence |
|---|---|
| `go test ./internal/deployability/... ./internal/resources/...` | pass |
| `(cd scenarios/infrastructure-manager/api && go test ./internal/portability/... ./handlers/portability/...)` | pass |
| `make lint-portability` | pass; all current findings have dated allowlist entries |
| `make -C packages/proto generate` | pass; Go and TypeScript stubs regenerated |
| `go run ./internal/deployability/cmd/platform-support --check` | pass |
| `go run ./internal/deployability/cmd/platform-support` | pass; `docs/reference/platform-support.md` regenerated |
| `vrooli capability ledger --json` | pass; 29 resources, two architecture cells per OS, skip budget measured |
| ledger contract `jq` check | pass; `projection_contract=pass` |
| `vrooli capability fleet --json` | pass; 19 blocked rows |
| `GOWORK=off go run ./cmd/vrooli capability conformance --json` | 1794 targets, 86 hard findings, 72 warnings; not a repository-wide pass |
| `GOWORK=off go run ./cmd/vrooli capability conformance --declarations-only --json` | pass; zero declaration findings |
| `vrooli scenario test audio-tools` | server-owned run `20260826-175649-6c56343f`; 12/24 phases passed, native resource checks passed |
| `vrooli scenario test infrastructure-manager` | server-owned run `20260826-183526-20346877`; terminal `FAIL` on standing UI/branding/proto debt |
| `make check` | exits 2 at `cross-compile`; the broad Go package portion passed, then the repository-wide conformance output reported the recorded unrelated hard findings |
| `vrooli scenario test scenario-dependency-analyzer` | server-owned run `20260826-195423-00d86ad0`; terminal `FAIL`, 16/21 phases passed; portability passed at L1 |
| `vrooli scenario test infrastructure-manager` | server-owned run `20260826-195121-9a10b025` was launched; its attached waiter ended before terminal JSON, so no pass is claimed. The prior terminal run `20260826-183526-20346877` was `FAIL` on standing UI/branding/proto debt. |

## Resource and UI evidence

The live ledger now exposes one row for each of the 29 resources. Each row
contains its driver, acquisition kind, three host-OS claims, and both amd64 and
arm64 architecture cells. Missing or unreadable skip-budget input is represented
as unavailable with a reason rather than zero. The infrastructure-manager board
renders these rows and the skip-budget verdict.

The API and component UI tests pass. The page test remains blocked by the
checkout's pre-existing duplicate TanStack Query versions (`5.100.9` and
`5.59.0`), which produce separate `QueryClient` contexts. No dependency change
was made because dependency changes are outside this plan's authority.

## Negative fixtures

The durable declaration fixtures are in
`internal/deployability/testdata/declarations/` and are exercised by
`TestDeclarationFixturesStayExecutable`:

- `not_applicable-with-mechanism.json` fails the closed-cell mechanism rule.
- `not_implemented-without-mechanism.json` fails the open-cell mechanism rule.
- `circular-evidence.json` fails the circular-evidence rule.

The remaining negative conformance fixtures are retained by the existing
deployability/resource test suites. The full seven-fixture Gate 2 replay is not
claimed here because the repository-wide test admission/conformance gate stayed
red.

## Deferred findings and authority gaps

- Full conformance remains red on unrelated dirty-tree modules, including
  missing generated helpers and platform-specific implementations. The durable
  Plan Manager log records the paths and findings.
- The infrastructure-manager scenario run has prior terminal evidence red on
  standing UI, branding, and proto findings even though the portability
  presentation and focused API tests pass. The newer run was not assigned a
  verdict after its waiter ended before terminal JSON.
- Docker-stopped startup was not proven because this session could not
  authenticate for `sudo systemctl stop docker`. Native managed-service starts
  and health checks passed while Docker was still active.
- The baseline collection remained partial because Test Genie admission was
  saturated and `api-library` did not produce a phase result. No baseline diff
  is represented as clean when that evidence is absent.
- The repository still contains active unrelated Docker/Compose fixtures under
  `resources/claude-code/sandbox`; the migrated resource trees themselves have
  no remaining Docker/Compose deployment files.

## Source evidence

- [`portability-iii-validation-2026-08-26.md`](portability-iii-validation-2026-08-26.md)
- [`portability-iii-baseline-2026-08-26.md`](portability-iii-baseline-2026-08-26.md)
- [`portability-iii-runtime-lint-audit-2026-08-26.md`](portability-iii-runtime-lint-audit-2026-08-26.md)
- [`portability-iii-kernel-fs-audit-2026-08-26.md`](portability-iii-kernel-fs-audit-2026-08-26.md)
- [`portability-iii-kokoro-migration-2026-08-26.md`](portability-iii-kokoro-migration-2026-08-26.md)
- [`portability-iii-speaker-verification-migration-2026-08-26.md`](portability-iii-speaker-verification-migration-2026-08-26.md)
