# Windows submission fresh C review

Independent reviewer `windows_submission_fresh_c`, 2026-11-11. Read delivery CHARTER/SPEC/STATUS/FEEDBACK, product-resumption/NATIVE-LIGHTING-GAP-AUDIT-20261111.md and docs/TESTING.md. Scope: current Windows task-owned maintained capsule submission, custody, observation, stop/release and the real OpenRGB retirement callers. This is a finding-bearing source/fixture pass, not a clean component pass or product acceptance.

## C1: old-boot submission strands normal retirement

**Material source finding, repaired by the producer and independently rechecked.** `lookupOwnedOperation` reads the protected retained payload and calls `windowsRef(p)`. The ref therefore retains the payload's original BootID after reboot; it does not claim that the task action restarted. This is correct identity preservation for an on-demand task with no triggers.

The resource cleanup adapter does not account for that Windows behavior:

- `retireOpenRGBInitialWith` revokes the domain fence, performs Lookup, and asks `retireOpenRGBNativeBootWith` to handle boot cleanup.
- `retireOpenRGBNativeBootWith` sees equal original and lookup BootIDs and returns `false, nil` when there is no prior native boot retirement. It never reaches a new-boot cleanup branch. `reconcileOpenRGBServingBoot` has the same equal-BootID early return for a serving publication.
- Initial cleanup with no persisted quiescence then calls `StopOwnedService`. `cleanupManagedUpdateRole` likewise calls Stop after the boot adapter declines.
- `stopOwnedOperation` reads the actual current boot and returns ErrOperationIdentity when it differs from the frozen payload boot, before revocation or task cancellation. Repeating normal disable/recovery cannot change these facts. A rootless interrupted/uncertain submission is sufficient: no PID reuse or physical hardware is needed to trigger the failure.

Thus reboot between reservation/submission and retirement can strand the exact ledger/task, preventing normal opt-out or a fresh governed start. Direct Release is not the normal initial/uncertain-role cleanup path because the domain needs a quiescence receipt first. A previously retained live-root claim has the same Stop gate.

`windows-submission-fresh-c/check_cold_boot.py` projects the **actual current complete stopOwnedOperation function body**, replacing only platform effects with hermetic fakes. It supplies an exact retained old-boot payload/ref, a different current boot, absent Job and successful exact-task cancellation. The desired assertion is that this safely absent frozen registration can be retired. Initial observed execution exits 1 with `platform: owned operation identity mismatch`; no native effects run. The caller chain above is source-traced, not a real Windows runtime journey.

Repair must preserve original intent/claim identity, never manufacture a new-boot action ref, and never inspect or kill an old PID merely because its numeric PID exists now. A cold-boot branch needs exact maintained no-trigger task metadata, durable revocation, authoritative exact task cancellation and fresh instances/state/Job absence, followed by normal retirement. A current named Job or unknown native response remains unresolved. Parent and producer received the finding before production changes.

## Other assessed boundaries

Normal caller startup uses Scheduler Run, not direct caller CreateProcess. The trusted task action retains image/Job custody and waits for its SDK child; the SDK inherits its image lock without inheriting the Job handle. Prepare joins the bounded kill-on-close Job before durable root/custody publication and vendor admission. Complete payload and RunIssued precede Run; uncertain submission prohibits replay. Mapped image, task/GUID/engine-birth relation, same actual user/non-elevation, exact arguments/cwd and boot checks remain explicit. Environment normalization occurs before vendor launch. Stop uses exact native IRegisteredTask.Stop and requires actual S_OK rather than treating S_FALSE as cancellation. Retained empty Jobs cannot establish namespace absence. No fresh issue was established in these source paths during this pass.

Natural exit with no persisted terminal receipt remains OutcomeUnknown, not success. Native S4U/profile, engine topology, nested Job inheritance, task XML/DACL normalization, queue cancellation, HID access and physical output remain target qualifications; fixtures cannot close them. Review retains the accepted same-user trusted maintained host/pinned-vendor threat model.

## Changed, verified, remaining and unverified

Changed: this report and the adjacent hermetic source-projection script only. No production code, dependency, native task/Job/logon/token/device/service/root/elevation, host state, target dispatch or Git operation was performed. No background process, service or model was started.

Verified: independent source path above and initial deterministic failing counterexample (`python3 packages/platform-go/docs/windows-submission-fresh-c/check_cold_boot.py`, exit 1). Prior review reports were read after the fresh lifecycle trace to distinguish earlier repaired findings. No broad unchanged suite was rerun.

Remaining: native Windows recovery qualification and a genuinely fresh clean review. A finding-bearing pass never counts toward consecutive clean passes.

Unverified: real Windows reboot/normal disable recovery, task queue/Stop behavior, root/SDK image and native handle semantics, caller-close survival, logged-out boot, resume/controller return and visible Off/static. Full destination remains open.

## Repair and independent recheck

The producer added the production-used `windowsOwnedRetirementTaskRoot` decision and explicit old-boot Stop/Release paths. With a different current boot, initial exact Job absence is mandatory; old PID/start/GUID authority is cleared only for task cleanup. Revocation precedes exact task cancellation; Stop checks fresh Job absence again and returns quiescence with the unchanged original ref. It never opens or terminates a Job in this branch. Release skips old PID liveness on proven boot mismatch, cancels/deletes the exact verified task, rechecks namespace absence and removes the original ledger. Same-boot behavior preserves its root/GUID and existing live-role checks. No new-boot ref or alternate architecture was introduced.

The retained Python fixture now projects the actual complete **Stop and Release** bodies with native boundaries replaced by fakes. It passed these desired assertions:

- Rootless interrupted old-boot admission can stop and retire.
- A retained-root old-boot admission never queries the potentially reused numeric PID (the fake panics if called), and never opens/terminates an old/new Job.
- The task cancellation receives no obsolete root/GUID authority.
- A present initial named Job and a Job appearing after cancellation both refuse quiescence.
- Release performs `absence → revoke → exact task stop → fresh absence → task deletion → ledger deletion`.

`python3 packages/platform-go/docs/windows-submission-fresh-c/check_cold_boot.py` exits 0 after the repair (one top-level projected test). The initial version exited 1 before it. This is source-control-flow evidence; native functions are deliberately faked and no Windows operation executes.

Focused portable selection from packages/platform-go:

```sh
go test -run '^TestWindows(OwnedRebootRetirement|TaskAdmission|TaskRejects|OwnedTaskHasNoBoot|OwnedPayloadBounds)' -v .
```

Observed PASS, seven top-level tests, no skips: payload bound; durable once-Run admission; generic/substituted launch refusal; no boot/retry/implicit lifetime task policy; serialized escape bounds; reboot retirement refusing old-root/new-Job authority; foreign XML attribute namespace refusal. No broad unchanged checks were run. The producer separately owns native cross-build evidence; this reviewer does not claim having run it.

Observed repaired source hashes (shared mutable source, not isolation):

- `operation_windows.go`: `3d7084eeb5fbb8756c8472b7cf58fde51221ff4ad9ebb10062347e3e3ddce5c3`
- `windows_owned_task.go`: `e542592436ef8d3ea8bdb3c0b4487ed042c276470a936bd2161b6892677e0cb2`

C1 is closed at the source/fixture level. Initial/managed resource caller integration is source-traced, not an executed real normal-disable/reboot journey. Native queue and task cancellation behavior, actual reboot recovery and the full product destination remain open.
