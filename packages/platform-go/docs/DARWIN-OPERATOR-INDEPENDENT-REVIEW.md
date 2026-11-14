# Darwin actual-operator ownership independent review

2026-11-11. Reviewer: `/root/native_lighting_portability/native_owned_services/darwin_operator_maturation`.
Workspace: `/home/matthalloran8/Vrooli-native-lighting`. This review is source and hermetic fixture evidence only.

## Scope and decision

Read the delivery CHARTER, SPEC, STATUS, FEEDBACK, native-lighting gap audit,
`docs/TESTING.md`, and `DARWIN-OPERATOR-JOBS-CHECKPOINT.md`. Reviewed the actual
operator design: boot daemon runs as the operator; role jobs use exact nonce
labels in `user/UID`; immutable maintained capsule entrypoint; trusted pinned
vendor; per-process fork denial and selected transport restriction. This review
does not introduce another account, home sandbox, generic executable authority,
or permanent elevated runtime.

**Initial fresh pass found DOR-1. The producer repaired it and the independent
original-counterexample recheck passed.** A subsequent fresh bounded source pass
found no additional material issue in the scope listed below. This does not
constitute full product/native acceptance or two full-destination clean passes.

## DOR-1: retirement discards a changed manager root after revocation

Given an exact role receipt whose root is PID42/birth999, when `stop` finishes
its initial inspection and revokes durable files, and launchd's same role label
then identifies PID77/birth777, the next snapshot must reject that changed root
before dispatching `bootout`. The current code checks only `present` at
`darwin_owned_backend.go:386` and boots out the new root at line393. It returns
`NativeQuiescent=true` for the old receipt.

The same transition affects `release` at lines439–445. Given a naturally exited
original root and a newly associated root after revocation, release boots out
the new root, removes the record, and returns success. These are two triggers
of one retirement boundary defect. Initial pre-revocation inspection does not
validate the manager association observed after revocation I/O.

The independent fixture wraps only `darwinOwnedStore.Revoke`, performs the real
isolated revocation, then changes the fixture's exact-label manager association.
It asserts identity failure and zero additional native effects. Both assertions
failed on first run. This models a changed association and does not claim that
launchd supports an atomic compare-birth-and-bootout operation. A source fix
should preserve the exact-label mechanism and reject an observed mismatch;
numeric inspect-then-kill is not a remedy.

Fixture: `review-fixtures/darwin-operator-revocation-counterexamples.go.txt`.
Overlay: `/tmp/darwin-operator-review-overlay.json` maps the fixture to a virtual
`independent_darwin_operator_review_test.go` in the platform package. Reproduce:

```sh
go test -overlay=/tmp/darwin-operator-review-overlay.json -run '^TestIndependentDarwin' -count=1 .
```

First observation: both `TestIndependentDarwinStopRejectsRootChangedDuringRevocation`
and `TestIndependentDarwinReleaseRejectsRootStartedDuringRevocation` FAIL; each
recorded one exact-label bootout, nil error, and the first returned quiescence.
Material finding sent to parent and implementation owner before report completion.

### Repair and independent recheck

The producer factored `inspectAssociation`, which returns a fresh complete
manager/kernel-root observation and presence flag. Both retirement paths now
call it after revocation and before bootout. The original independent two-test
overlay passed under race detection: `ok github.com/vrooli/platform-go 1.090s`.
This is a fix recheck, distinct from a fresh pass.

The fresh follow-up pass inspected the maintained admission guard, domain-only
recovery and dispatch phase, typed query unknown/absence semantics, private
metadata/ACL validation, immutable publication, same-PID image replacement,
kernel observation brackets, fork-denial verifier, and current retirement code.
No further material source finding was identified within that scope. New
independent post-revocation probes cover same-PID/new-unique-birth, boot change,
manager outage with retained intent, and legitimate same-birth exec. The complete
four-test overlay (including three uncertainty subcases) passed with `-race`:
`ok github.com/vrooli/platform-go 1.274s`. Existing focused Darwin tests passed
normally in 2.305s and with race detection in 3.431s. These execute portable fixture
logic on Linux; Darwin-tagged CGO/runtime code was inspected, not executed.

Rechecked backend SHA256:
`fa5f2576980ee91afcbd3621158902af9faff41a7cc465d4bed7722acf1f2bb8`.
The workspace remains shared and mutable; the digest identifies the observed
file after validation, not an immutable whole-run source snapshot.

## Verified boundaries and qualification limits

Source inspection retains closed admission before persistence/native effects,
unknown-query preservation, durable `DispatchPrepared` before role submission,
no adoption of a role discovered after domain-only recovery, bounded positive
root settle, stable unique birth across exec, transient exec-version observation
brackets, revoked capsule refusal, user/root ACL ownership, and the single-link
immutable rename publisher. These are inspected mechanisms, not native execution
qualification. Existing producer fixtures are not counted as independent proof.

Real Apple SDK/CGO compilation and linking, private flavor17 ABI support,
ServiceManagement dictionary shape and actual user-domain targeting from the
boot daemon, first boot before login, sandbox runtime/fork negatives, native
HID/IOKit behavior, logged-out boot/recovery, reconnect/resume and physical
OpenRGB output remain OPEN. CGO0 cross-builds cannot close these rows.

Changed: this report and isolated retained fixture only. Verified: initial
counterexample failures, producer repair independently rechecked, fresh bounded
source pass and additional probes as above. Remaining: independent review across
the integrated resource/boot owners and target qualification. Unverified: all target/runtime/hardware
claims. No source edits, Git operations, account/service/device/root effects,
dependency work, or target dispatch occurred. No resource/service/model was
started by this reviewer.

Work-record journal: `02f55779-470d-4744-a186-b450644a7b4f`.
