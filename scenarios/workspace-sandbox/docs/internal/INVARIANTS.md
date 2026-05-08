# Workspace-Sandbox Invariants

This document is the **authoritative list** of behavioural invariants
the workspace-sandbox runtime depends on. Each invariant is given a
stable ID (`I-<DOMAIN>-<N>`); each ID MUST appear as a `t.Run` subtest
name somewhere in the test tree. The CI scan in
`scripts/check-invariants.sh` (run from the repo root) fails if any
documented ID is missing a matching test.

The list is the source of truth for what must always hold. Add an
entry every time you discover that a recurring bug fix had an
unwritten assumption — *naming the assumption* is the only way it can
be defended by the test suite the next time someone refactors.

## How to read this file

| Column | Meaning |
|---|---|
| **ID** | Stable identifier referenced by the test name (`t.Run("I-...", ...)`). |
| **Statement** | The behavioural promise. Plain English; no implementation details. |
| **Enforced by** | The `package::Symbol` (or file path) that owns the invariant. |
| **Tested by** | The `*_test.go` file + test name where the assertion lives. |

## SSE wire-format

### I-SSE-1 — every SSE log stream emits `event: end` exactly once, after `event: exit` if any

- **Enforced by:** `internal/sse::Writer` / `internal/handlers/process_logs.go`
- **Tested by:** `internal/handlers/process_sse_test.go::TestProcessLogsSSE_FrameOrdering`

### I-SSE-2 — SSE writers refuse construction without an `http.Flusher`

- **Enforced by:** `internal/sse::NewHTTPWriter`
- **Tested by:** `internal/sse/sse_test.go::TestHTTPWriter_RequiresFlusher`

## Home-overlay policy

### I-HOME-1 — `policy.DecideHomeOverlay` is pure (no I/O)

- **Enforced by:** `internal/policy/home_overlay.go::DecideHomeOverlay`
- **Tested by:** `internal/policy/invariants_test.go::TestInvariants/I-HOME-1`

### I-HOME-2 — `HomeOverlayOptional` + non-Present state ⇒ allowed + `HOME_OVERLAY_FALLBACK`

- **Enforced by:** `internal/policy/home_overlay.go::DecideHomeOverlay`
- **Tested by:** `internal/policy/home_overlay_test.go::TestDecideHomeOverlay_Matrix`
  (the `optional/...` rows; the test docstring tags I-HOME-2 explicitly)

## Mount lifecycle

### I-MOUNT-1 — `Service.Delete` returns ⇒ no fuse-overlayfs daemon remains for that sandbox UUID

- **Enforced by:** `internal/sandbox/service_lifecycle.go::Service.Delete` →
  `internal/sandbox/delete_daemon.go::Service.killDaemonsForSandbox`
- **Tested by:** `internal/sandbox/delete_daemon_lifecycle_test.go::TestDelete_Daemon_Lifecycle`

### I-MOUNT-2 — `Driver.Mount` is idempotent on already-mounted sandboxes

- **Enforced by:** `internal/driver/contract_test.go` per-driver matrix
- **Tested by:** `internal/driver/invariants_test.go::TestInvariants/I-MOUNT-2`

### I-MOUNT-3 — checkpointed sandboxes are unmounted but resumable

- **Enforced by:** `internal/sandbox/service_turn_checkpoint.go::Service.TurnCheckpoint` and `internal/sandbox/service_turn_checkpoint.go::Service.Resume`
- **Tested by:** `internal/sandbox/service_test.go::TestService_TurnCheckpoint_NoChangesTransitionsCheckpointed` and `internal/sandbox/service_test.go::TestService_Resume_CheckpointedTransitionsActive`

## Change detection

### I-CHANGE-1 — `ChangeTracker.GetChangedFiles` is deterministic for a given filesystem state

- **Enforced by:** `internal/driver/changedetect/walker.go::Walk`
  (sorts results via `sort.SliceStable`)
- **Tested by:**
  `internal/driver/changedetect/walker_contract_test.go::TestStrategy_DeterministicOrdering`

## Driver identity

### I-DRIVER-1 — a sandbox's `DriverID` is immutable after Create

- **Enforced by:** `internal/sandbox/service_lifecycle.go` (no Update path
  for `DriverID`); driver swap only affects new sandboxes via the slot
  hot-swap.
- **Tested by:** `internal/sandbox/invariants_test.go::TestInvariants/I-DRIVER-1`

## Audit emission

### I-AUDIT-1 — every state transition emits exactly one audit-log entry

- **Enforced by:** `internal/audit::Emitter` callers in
  `internal/sandbox/service_*.go`
- **Tested by:** `internal/sandbox/invariants_test.go::TestInvariants/I-AUDIT-1`

## How to add a new invariant

1. Add a row above with a fresh ID, the statement, the enforcement
   site, and the test reference.
2. Add the `t.Run("I-XXX-N", func(t *testing.T) { ... })` block in the
   referenced test file.
3. Run `scripts/check-invariants.sh`. Its job is to fail if the new ID
   has no matching `t.Run`.
