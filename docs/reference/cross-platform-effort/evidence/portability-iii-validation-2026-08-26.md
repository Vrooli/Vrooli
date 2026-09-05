# Portability truth chain III validation — 2026-08-26

This is the implementation validation record. The final plan handoff is in
[`portability-iii-final-validation-2026-08-26.md`](portability-iii-final-validation-2026-08-26.md).

## Implemented and verified

- Declaration validation rejects circular platform evidence, invalid
  `not_applicable`/`not_implemented` shapes, and undated policy exclusions.
- Situation classification is per host OS and preserves the underlying
  declaration when a policy applies to another OS.
- Conformance uses `go vet` across six cells: Linux, macOS, and Windows at
  amd64 and arm64. Vet style diagnostics are warnings; package load failures
  remain hard findings.
- The portability AST rules run from `make check` through
  `make lint-portability`; all 87 current exceptions have a reason and review
  date.
- SDA carries named source-derived platform verdicts with
  `sda_source_verdict`.
- The grid/protobuf projection carries resource architecture claims and the
  measured skip budget, or an explicit unavailable reason.
- The platform-support renderer is a typed Connect client and has `--check`
  mode; it does not shell out to `vrooli`.
- The declaration-only resource gate is active and returns zero findings for the
  migrated resource set.
- `kokoro` has measured Linux amd64 CPU and CUDA composed artifacts:
  `deb0268971cae48f88c089f90bf0937c9c321d17f56a1277ed54b1730a241ab8` and
  `db8375fc431b025f16217596d52973acca595e681c2864cbfeab253a0c075d57`.
- `speaker-verification` has one measured Linux amd64 CUDA artifact,
  `46cae27aef0907cc04b0bcb3e07446775764a8d2a2521e9d96c032606c1a4bef`,
  whose server retains CPU fallback behavior.
- Docker/Compose deployment files were removed from Kokoro, Kyutai STT, and
  Speaker Verification; all three manifests use `managed-service`.
- Native Kokoro and Speaker Verification starts were exercised through the
  managed-service control plane on Linux amd64. Both reached healthy CUDA
  serving with `mode_drift=false`; see the resource-specific evidence files
  for artifact digests, endpoint responses, and the one real isolation defect
  found and fixed.
- The live `infrastructure-manager` owner now decodes the object-valued policy
  schema, and a source control-plane forced restart rebuilt the stale linked
  shared-package snapshot. `vrooli capability ledger --json` is healthy again:
  52 capabilities, 29 resource rows with 87 OS claims, and a measured skip
  budget of 155 per OS. The CLI's canonical JSON field names are
  `acquisition_kind` and `skip_budget`.
- Lifecycle install freshness now includes declared shared-package output
  digests, so a changed governed file dependency invalidates the consumer
  install state instead of allowing a stale pnpm snapshot to survive.

## Validation results

| Check | Result |
|---|---|
| `go test ./internal/deployability` | pass |
| `go test ./internal/resources` | pass after native bundling/fleet assertions were updated |
| `go test ./...` in `packages/binaryfetch` | pass |
| infrastructure-manager portability/handler tests | pass |
| scenario-dependency-analyzer deployment tests | pass |
| `make acquisition-schema-check` | pass |
| `make fleet-contract-check` | pass after allowing managed-service host-fact target selection |
| `make lint-portability` | pass |
| lifecycle shared-package freshness regression | pass; `GOWORK=off go test ./internal/lifecycle -run 'TestInstallInputDigest...'` proves a changed governed output invalidates the consumer install gate |
| six-cell `make cross-compile` | 1794 targets; 146 hard findings, 20 vet warnings |
| `go run ./cmd/vrooli capability conformance --json` | 1794 targets; 86 hard findings, 72 warnings; failures are outside the migrated resource/grid changes and remain recorded for follow-up |
| `vrooli scenario test audio-tools` | server-owned run `20260826-175649-6c56343f`; 12/24 phases passed, with native service checks passing and unrelated standing failures elsewhere |
| `vrooli capability ledger --json` | pass after source control-plane restart; 52 capabilities, 29 resource rows/87 OS claims, skip budget 155 per OS; contract check passed for two architecture cells per OS |
| `vrooli capability fleet --json` | pass; source-derived blocked rows are present and explainable |
| `vrooli scenario test infrastructure-manager` | server-owned run `20260826-183526-20346877` terminal `FAIL`; portability presentation reached clean L1, while standing UI/branding/proto findings remain |
| `vrooli scenario test scenario-dependency-analyzer` | server-owned baseline admission was saturated; deployment package tests pass locally |

## Explicitly incomplete

The native resource installs, artifact digest measurements, first-run model
warmups, and direct health checks are complete on the current Linux amd64 host.
The live consumer run completed but is red on standing scenario debt; its native
resource checks passed.

Docker-stopped startup remains unproven because this session could not
authenticate for `sudo systemctl stop docker`; Docker remained active with the
pre-existing `postgis-main` container. This is an environment-authority gap,
not evidence that either native service requires Docker.

The three migrated resource trees have no remaining Docker/Compose files. Four
unrelated repository files remain under `resources/unstructured-io` and
`resources/claude-code/sandbox`; the plan's literal repository-wide zero-file
check was not satisfied, and this run did not widen the migration boundary to
those unrelated resources.

The live ledger was initially unavailable for two independently observable
reasons: infrastructure-manager was running an API built against the old
string-valued policy schema, and its UI's linked react-component-library
snapshot lacked `useLocale/1.0.1`. Both are now repaired at their owning
boundaries; the owner restarts cleanly and the live ledger is available. The
Docker-stopped resource proof remains the only environment-authority gap in
the native migration evidence.

The complete repository conformance gate is also red on unrelated dirty-tree
modules and pre-existing test-only portability failures; the durable Plan
Manager findings identify those paths and outputs.
