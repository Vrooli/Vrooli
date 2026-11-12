# Independent Windows owned submission review

Reviewer: `windows_submission_adversarial`, 2026-11-11. Scope: caller-independent Windows maintained role submission in `packages/platform-go`. Read the delivery CHARTER/SPEC/STATUS/FEEDBACK, native lighting gap audit and `docs/TESTING.md`. This report does not close R3, native lighting, or the full delivery goal.

Work record: `be169776-9874-4c38-9061-e319ed57c0c0`.

## Result

The first pass found two material unknown-to-absence defects. F1 and F2 are repaired and independently rechecked. F2 first had a retained failing actual-source counterexample after an initial partial repair. This is a finding-and-repair pass, not a fresh clean maturation pass. Source/build evidence does not establish Windows runtime, logged-out boot, device permissions or physical light behavior.

### F1 — task credential failure incorrectly proved task absence (repaired)

Given an existing task whose security database/account observation returns HRESULT `0x8004130f`, stop/release must retain unknown task state and the ledger. The original `taskNativeErrorMissing` accepted `hr == 0x80070002 || hr == 0x8004130f`. Microsoft defines the latter as `SCHED_E_ACCOUNT_INFORMATION_NOT_SET`, not a missing task. Both `stopWindowsOwnedTask` and `deleteWindowsOwnedTask` consumed that result as absence, allowing retirement without the required native proof. This also affected existing generic task callers.

The producer now delegates to portable `windowsTaskHRESULTMissing`, which accepts only typed `0x80070002`. Independent fixtures sweep all `0x80041300..0x800413ff`, access denial, RPC outage, unspecified failure and queued-success codes; none may establish absence. The Windows-only existing test's stale expectation was also routed to the producer. [Microsoft's HRESULT definitions](https://learn.microsoft.com/en-us/windows/win32/taskschd/task-scheduler-error-and-success-constants) support the distinction.

### F2 — queued dispatch uncertainty disappears on stop retry (repaired)

Given a task snapshot with queued State 2 and an empty running-instance collection, stop must not establish task quiescence. The original code returned success for any empty collection. The producer added a guard before and after disabling the task, which fixes the single-call counterexample. However, disabling can change the next visible state to Disabled (1). A repeated stop/release then sees State 1 and an empty collection and returns success without preserving the unresolved prior queue observation.

`windows-submission-review/check_stop.py` projects the actual current `stopWindowsOwnedTask` function body into a Linux Go test and substitutes only native snapshot/control functions. The first fake snapshot is State 2/empty; disable changes it to State 1/empty. The first stop returns NotQuiescent; the second returns nil. Observed FAIL: `retry erased unresolved queue: <nil>`. This requires durable uncertainty or documented/native proof that disabling cancels every prior queued Run. The fixture does not assert that a particular Windows release actually retains a queue after disable; that API semantic remains a native qualification obligation. Revocation still independently denies delayed vendor admission.

The final repair performs UUID-exact registered-task Stop after disable on **every** attempt, preserves cancellation errors, then refuses queued/running empty snapshots. Microsoft documents that [RegisteredTask.Stop stops all instances](https://learn.microsoft.com/en-us/windows/win32/taskschd/registeredtask-stop); this supplies the missing cancellation effect instead of inferring it from disabling. The extended actual-source fixture now passes: initial/retry cancellation errors remain failures; every attempt cancels the exact task after disable; queued post-Stop snapshots remain NotQuiescent; success requires cancellation plus disabled/empty observation. Native pending-queue behavior remains unexecuted, but the source no longer erases uncertainty without a fresh cancellation effect.

## Source assessment

The task action is the existing immutable maintained host and exact capsule argv. Caller submission contains no direct CreateProcess fallback. Create-only registration, the protected frozen ledger and the durable pre-Run marker prohibit replay after uncertain registration/run. Same actual SID, non-elevation, boot identity, mapped image, observed argv/cwd, root-once admission and singleton task observation precede custody. The trusted wrapper joins its own bounded kill-on-close Job, pins its image and publishes post-flush custody before vendor code. Vendor child launch retains its image handle without the Job handle. The parent scenario containment implementation was not weakened.

Stop/release serialize against Run/bootstrap and persist revocation before task control. Revocation prevents a delayed wrapper from reaching vendor code even if the scheduler dispatches late. Task errors remain errors after F1. Retirement requires scheduler disable/instance stop plus exact Job absence and root quiescence, task deletion and a confirming missing-task result before ledger removal. No relay, account pivot, capability token or root broker was added by this submission path.

### Qualification limits retained

1. EnginePID is documented as the task engine, not an action PID or a guarantee of engine isolation. Current association checks a stable singleton GUID plus the exact approved image/capsule/SID/boot and birth-ordered native parent relation. A deliberately same-user duplicate action with identical capsule argv and a shared engine could mimic that relation. The producer explicitly trusts same-user code; independent ordinary invocations use distinct capsule/intent UUIDs. Do not claim undocumented per-instance engine isolation. [EnginePID documentation](https://learn.microsoft.com/en-us/windows/win32/taskschd/runningtask-enginepid).
2. `GetInstances` documents currently running instances; it does not explicitly promise visibility of every scheduler queue state. A native queued Run→disable/registered-task Stop/delete test must establish pending-dispatch quiescence. Source now performs explicit cancellation and refuses visible queued/running state with an empty collection (F2). Revocation provides a separate pre-vendor refusal guarantee. [GetInstances documentation](https://learn.microsoft.com/en-us/windows/win32/taskschd/registeredtask-getinstances).
3. S4U requires batch-logon privilege. Registration or runtime denial must stay explicit; this implementation must grant nothing. Actual session-zero image access, scheduler XML/DACL normalization, process-parent relation, unprivileged API access and Job behavior remain native tests. [Security contexts](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks).

The caller Job defect is real under Microsoft's documented recursive Job inheritance. Delegating launch to the scheduler removes that caller ancestry by design; only an actual nested normal caller close test establishes the operated result. [Nested Jobs](https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs).

## Independent fixtures and evidence

`windows-submission-review/adversarial_test.go.txt` contains two independent portable tests, loaded with the adjacent Go overlay. The tests require scheduler failure to remain unknown and reject added boot/logon/restart actions, elevation, image drift, finite lifetime and battery/network/idle gates. No production source was edited by this reviewer.

From `packages/platform-go`:

```sh
go test -overlay docs/windows-submission-review/overlay.json -run '^TestIndependentSubmission' -v .
go test -overlay docs/windows-submission-review/overlay.json -run '^Test(IndependentSubmission|WindowsTaskAbsence|WindowsOwned|WindowsCustody|WindowsTask)' -v .
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -overlay docs/windows-submission-review/overlay.json -c -o /tmp/vrooli-windows-submission-review-amd64.test.exe .
python3 docs/windows-submission-review/check_stop.py
```

Observed: both independent tests and all 12 task-policy subcases PASS; the wider focused selection executed 29 top-level tests with zero skips and passed in 0.008s. Windows amd64 test cross-compilation exited 0. Full focused output is `/tmp/vrooli-windows-submission-review.log` (ephemeral execution evidence). Cross-compilation did not execute Windows-only tests. No task/account/token/service/device/root operation or native target dispatch occurred. No service, model or background process was started.

The source was mutable shared work. At the above observation, SHA256 values were `windows_owned_task.go=890d0d221940432de8503c97187f44361fe987afd3ac11578dbaff671acd5359`, `windows_owned_task_windows.go=b5dfe2906c4fa2fbd2cd97a0d0dd6f22008c132931e26b217404990366643b33`, `operation_windows.go=c908bbcd7ed7118bb86f5514b08b7a8bc142b3a9cff9ed8a968ce92b101b13fc`, `windows_owned_custody_windows.go=360a31bee19e7fdbf649cc89a3cf25a1bbbc13ceef8f21ce9e1486146f3bb6fb`, `windows_task_contract.go=c585f7c92a4a61c9febbd593568e0e30c7a04afebeb8b37e10a904fde1ea9568`, `task_scheduler_windows.go=00ec44d3115e76536e2d90eb53a44fbbd8efbd71724e86afd453f4b90979e914`. These fingerprints describe observations, not immutable suite isolation.

Remaining: finish producer integration and independently recheck affected changes; perform a genuinely fresh review after repairs; execute authorized Windows runtime tests for normal caller close, queued/uncertain dispatch, natural exit, reboot, exact Job/image/account/custody, stop/release and real lighting. Full product and two consecutive clean passes remain open.

Final affected recheck: actual-source stop harness PASS (0.003s); Windows amd64 cross-compilation PASS; focused portable selection rerun PASS with 33 top-level tests, zero skips (0.010s; `/tmp/vrooli-windows-submission-review-final.log`). Final source observation: `windows_owned_task_windows.go=942ce07638c3e7cb20cb62a89be22a4d619bb2b278a1f8d33d415e04bd1de6ad`, `operation_windows.go=2439c6b266677c16d3957b6a5ad5d4f7c12137992b9327f5c0156c79251f9028`, `windows_owned_task.go=3065c60bd880492e3bf697fcdb52c18fa65375d892871e4046faa936a66b98a8`. These supersede the corresponding initial hashes above. The EnginePID hostile same-user duplicate scenario is outside this bounded review's adversary model; no further source work was requested for it.
