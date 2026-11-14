# Windows actual-operator Jobs fresh adversarial pass B

Reviewed 2026-11-11T12:07:56.078541+00:00. Independent read-only source review by `jobs_final_review`. Source tree: `/home/matthalloran8/Vrooli-native-lighting/packages/platform-go`.

## Decision

**Initial pass found one material source-contract defect, WJ-B1. The owner repaired it and the independent follow-up below verifies the repair; no additional material finding was established.** The original pass remains a finding-bearing pass. A repair recheck does not replace a separately scoped fresh clean pass. Windows runtime, hardware and full destination acceptance remain OPEN.

The selected security contract trusts the pinned vendor under the actual operator's standard non-elevated token. This review does not reinstate an arbitrary compromised-same-SID vendor sandbox, alternate account, private desktop or additional token privileges.

## WJ-B1 — reject foreign observer principal before publishing a Windows invocation

Severity: medium; accepted-input lifecycle/recovery defect. **Disposition: repaired and independently rechecked.** The following describes the original source and failing counterexample.

`windows_owned_contract.go:46` validates the observer's SID syntax, while `:58` validates the native account independently. Neither this boundary nor `validateServiceIntent` requires a nonempty observer to equal the actual account. Consequently a correctly re-digested invocation with account `sid:S-1-5-21-100-200-300-1001` and observer `sid:S-1-5-21-100-200-300-1002` passes the exact production predispatch payload validator.

The receipt created from that accepted payload is invalid under the canonical actual-operator identity contract: `native_owned_identity.go:58` rejects a nonempty observer different from the account. `windowsRef` copies both fields without reconciling the discrepancy. `operation_windows.go:264-268` returns that receipt after custody, and `lookupOwnedOperation` also reconstructs it from durable metadata. `owned_service.go:125-129,160-178` reject this receipt before Inspect, Stop or Release reaches the Windows owner. Repeating Lookup does not repair the frozen mismatch. Even a reserved, never-born invocation with these accepted fields therefore cannot be retired through those public lifecycle operations. This does not establish any observed native orphan; it establishes incompatible admission and recovery contracts before native execution is available.

Desired correction: enforce the canonical nonempty-observer-equals-actual-account rule at the Windows predispatch payload boundary, before `prepareWindowsOperatorPayload` or native creation. Preserve rejection at receipt validation. Add a regression that creates a fresh intent from each candidate specification, rejects a different valid SID, and accepts empty/same-account observer. A changed specification retaining an old digest only exercises digest refusal and misses this defect.

Reproduction used a temporary Go overlay of the existing fixture test file, without changing repository source. It recomputed the intent with `windowsOwnedFixtureIntent`, called `validateWindowsPayload`, then required `windowsRef(p).ValidateIdentity()` to succeed for every admitted payload. Exact observed result:

```text
=== RUN   TestWindowsFreshPassObserverAdmissionProducesValidReference
predispatch accepts frozen observer "sid:S-1-5-21-100-200-300-1002" for account "sid:S-1-5-21-100-200-300-1001", but its reference is rejected: platform: owned operation identity mismatch
--- FAIL: TestWindowsFreshPassObserverAdmissionProducesValidReference (0.00s)
FAIL github.com/vrooli/platform-go 0.003s
SOURCE_UNCHANGED_OVERLAY_EXIT=1
```

The temporary overlay directory was removed on completion. No failing regression was installed into source because this assignment is read-only.

## Additional challenges and proof limits

Read the canonical CHARTER/SPEC/STATUS/FEEDBACK, native gap audit, `docs/TESTING.md`, `windows_owned_backend.md`, checkpoint, every Windows source file named in the brief, and source-used contract/submission/security fixtures. Followed public validation into `owned_service.go`, `operation.go`, `native_owned_identity.go`, and maintained broker path/image validation.

Challenged loss of submitter before and after suspended birth, root publication and custody; reader visibility before flush; abort-only reconciliation; root death and PID reuse across inspection/termination; transient query-handle last-close effects; empty retained Jobs versus absent namespace; release against a live/reused root; frozen UTF8 handoff; suspended child account checks and handle inheritance. No additional concrete unsafe dispatch, false healthy state, wrong-incarnation termination or false absence was established. The source closes temporary Job query/termination leases on the inspected returns; refuses adoption from pending custody; binds known roots to incarnation/image; preserves unknown exit status. Native races remain unoperated. A transient failure after root death can require reconciliation; the reviewed paths do not turn that failure into successful ownership or permission to replay.

Microsoft's [Job Objects documentation](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects) confirms that kill-on-last-close depends on closing the final Job handle and that nested jobs include descendants. This supports treating every duplicate/query handle as a real lifetime lease; it does not prove this implementation's target behavior.

Fresh ordinary fixture execution:

```text
go test -race -count=1 -v -run 'TestWindows(Custody|Owned|ChildFrozen)|TestWindowsOperator' ./...
PASS github.com/vrooli/platform-go 1.047s
```

Named output confirms actual execution of payload/account/observer-digest tests, all five interrupted-submission metadata cases, custody hash binding, failed/paused flush gates, bootstrap-key refusal, invalid UTF8 root/child input tests, actual-user security facts and boot-policy fixtures. Other module subpackages reported no tests selected and provide no additional behavioral evidence. These are Linux-executed production-helper fixtures; Windows-tagged native API bodies were read, not executed.

Inspected existing owner receipts `/tmp/windows-operator-jobs-independent-repairs-module.log` (platform module PASS4.056s) and `...-crossbuilds.json` (six CGO0 targets, Windows amd64/arm64 test compilation and Windows amd64 vet, all exit0). These remain owner-produced source/build evidence, not independent native execution. The fresh counterexample explains why passing existing fixtures did not establish compatible admission and receipt validation.

## Changed, verified, remaining, unverified

Changed: this report only. Repository product source untouched. Temporary overlay/test build artifacts were private; the overlay was removed. No main-workspace changes, Git operations, dependency acquisition, target dispatch, services, logon/LSA, devices, permissions or grants. No long-lived process/resource/model was started; test processes completed.

Verified: WJ-B1 by a failing production-helper counterexample; selected existing fixtures pass under race; source ordering and refusal paths above reviewed. Shared source is mutable: this report does not claim an isolated exact-version snapshot.

Remaining: applicable fresh clean-pass accounting; WJ-B1 repair and independent recheck are complete. Native Windows suspended birth/image replacement, loaded actual SID hive under S4U, custody ACK/crash stages, last-close/recursive stop/release, scheduler ancestor Jobs, detached executor maintenance, actual non-elevated token, HID/Qt/SDK and logged-out boot/resume/physical output remain unverified. No full H/W/R/L/Q destination closure is implied.

## Independent WJ-B1 repair recheck — 2026-11-11T12:09:14.573488+00:00

Read the revised `validateWindowsPayload` and `TestWindowsOwnedObserverAdmissionAlwaysProducesCanonicalReceipt` in `windows_owned_submission_test.go:234`. The production boundary now rejects every nonempty observer unequal to `Spec.NativeAccount`, while the existing account/intent binding and digest validation remain in force. `startOwnedOperation` invokes this validator before `prepareWindowsOperatorPayload` or process creation. Earlier mutex acquisition and account/boot observations do not publish the durable invocation or dispatch a process.

The regression recomputes a fresh intent for each empty, same-account and different-valid-SID specification. It explicitly expects `ErrOperationIdentity` for the foreign observer and requires `windowsRef(p).ValidateIdentity()` for both admitted cases, so it exercises the original mismatch rather than stale-digest rejection. The repaired comparison matches the canonical receipt validator. No regression established in the surrounding admission ordering or account checks.

Independent execution completed exit0 with all three subcases visibly selected:

```text
go test -race -count=1 -v -run '^TestWindowsOwnedObserverAdmissionAlwaysProducesCanonicalReceipt$' .
PASS github.com/vrooli/platform-go 1.013s
```

The owner's `/tmp/windows-operator-jobs-pass-b-repair.log` also records the three-case PASS1.013s; the independent execution above is separate. No product source changed during this recheck. Only this report was updated. Native Windows APIs/runtime/hardware remain unexecuted, and no resource remains running.
