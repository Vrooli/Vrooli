# Darwin submitted-root uncertainty and retirement

E1 source repair, 2026-11-11. This is source/fixture evidence, not an operated
Darwin launchd failure or native hardware qualification.

## Counterexample and cause

The independent whole-lifecycle E OS facet retained a failing actual-backend/file
store fixture in `.vrooli/work/reviews/native-lifecycle-e-os`. Its modeled native
bootstrap launches the trusted wrapper but loses the response. The manager later
has no label association while the modeled wrapper remains alive. Lookup has no
kernel root identity and returns a valid rootless native ref.

Previously `claimAlive` returned false for a zero PID. Stop treated manager
absence plus that false result as full extinction, Release removed the persisted
intent, and Released returned true. `absent` already preserved this same-boot
uncertainty, but a successful rootless lookup bypassed it. Initial resource cleanup
before a durable claim or NativeRef therefore could persist quiescence and
Stopped from this false native receipt. The capsule rejects rootless execution
before vendor/device work; the demonstrated defect concerns trusted-wrapper
extinction and lost ownership uncertainty, not demonstrated vendor execution.

## Existing-owner repair

`darwinOwnedBackend.rootExtinct` centralizes the facts required after exact manager
absence:

| Durable/current evidence | Extinction result |
| --- | --- |
| Exact rooted ref, same origin boot | Observe that exact kernel birth/unique ID; a surviving orphan remains nonquiescent. A caller-supplied different boot cannot override the persisted origin boot. |
| Intent never reached DispatchPrepared | No private bootstrap was admitted; rootless cleanup remains supported. |
| Actual current boot differs from the persisted origin boot | The old origin boot is extinct; no old PID signal or reused-process query is required. |
| DispatchPrepared, same boot, no root identity | Unknown (`ErrOperationOutcomeLost`); manager absence cannot manufacture the missing root witness. |

Stop still durably revokes the role, rechecks association and uses only exact
launchd label bootout where that label is present. It does not signal sampled
PIDs, process groups or all processes of the operator. After label absence it
requires the extinction proof before NativeQuiescent. Release requires the same
proof before removing the native record. Released propagates unresolved extinction
while the record remains, and returns false for retained quiescent metadata so the
normal resource caller performs actual Release/Remove rather than skipping it.
After completed removal, manager/activation absence and the retained exact rooted
birth (where present) resolve a lost release acknowledgement through the existing
release contract.

No persisted field/format, broker, roster, account, root wrapper, arbitrary exec
fallback, or UID-wide discovery/termination was added. The current persistent
whole-source classifier already covers this native backend; no shape migration
is needed.

## Normal recovery

The resource execution fence is revoked before cleanup. A submitted/unclaimed
attempt whose native stop is unknown remains submitted, without NativeRef,
QuiescentAt or StoppedAt. Durable lighting intent remains Disabling; a retry must
not claim successful disable or launch a replacement while extinction is unknown.

If a later exact manager observation recovers the root, ordinary rooted cleanup
can drain it through its exact label and kernel identity. If the origin boot ends,
ordinary cleanup can establish old-boot extinction without touching a reused PID.
Never-dispatched setup failures still retire normally. Lost release replies reuse
the persisted exact quiescence/ref and verify completed metadata removal.

If the wrapper exits without ever yielding an exact root witness and the manager
association never returns, the existing observations cannot prove same-boot
extinction. Its fence/intent must remain unresolved until independent proof is
available or the origin boot ends. No timeout, unknown query, PID search or manual
metadata erasure is represented as that proof. This narrow uncertain state does
not make the useful normal native lifecycle unsupported.

## Validation

All checks use the isolated source with explicit VROOLI_ROOT,
VROOLI_SOURCE_ROOT and a private VROOLI_PROTO_ARTIFACT_ROOT. No native service,
device, process signal, vendor executable or host/account change was performed.

- `darwin_owned_uncertainty_test.go` exercises the real native backend/file store
  with modeled native edges: lost bootstrap/live unknown orphan, unknown natural
  exit, repeated stop/release/released, retained revoked metadata, exact-root
  recovery without redispatch, never-dispatched cleanup, changed-boot cleanup,
  lost release acknowledgement and forged origin-boot refusal.
- `internal/resources/openrgb_native_rootless_retirement_test.go` composes the
  actual durable lighting intent, initial attempt and exec fence through public
  Disable/Reconcile, with the native stop boundary modeled explicitly. It checks
  repeated Disabling and revoked/unclaimed retention, then exact-root or
  changed-origin-boot recovery and lost release acknowledgement, with no starts.
- Final focused Darwin race selection: PASS4.254s. Public rootless-disable
  fixture: PASS3.253s. Broader initial/boot/public-native recovery race selection:
  PASS81.054s. Final persistent codec/whole-source checks: PASS0.682s; full Linux CLI
  compile: PASS0.010s. Complete ordinary platform module and all six CGO0 root test
  compilations are refreshed after the final source refinement.
- Independent E OS aftercheck reran the unchanged original counterexample and
  new direct Release/passive Released/retry/revocation boot-change/new-root/
  exact-root recovery/actual-ledger-removal probes on final backend SHA256
  `0fa06cf4d41cc85a0edb8b4c3c17dc9d6a64fee2482758a59a4577182594c2af`:
  race PASS1.942s. Its independent normal resource fixture race PASS3.859s;
  isolated proto owner verifier exit0. Before/after final source hashes were
  stable. Receipt: `.vrooli/work/reviews/native-lifecycle-e-os/RESULT.md` and
  `repair-evidence.sha256`. No further material finding or production edit.
  The E finding-bearing whole pass does not become a clean whole-product review
  because this repair passes.

Actual absent-manager/live-unclaimed-wrapper reachability on modern launchd,
Darwin CGO/link/boot/user-domain behavior, Seatbelt/no-fork execution, SDK/HID and
physical lighting remain unqualified. No source fixture establishes those facts.
