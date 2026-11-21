# Windows independent submission maturation

Reviewer: `windows_submission_maturation`, 2026-11-11. Scope: current maintained Windows owned-service submission in the isolated native-lighting tree. Read the delivery CHARTER/SPEC/STATUS/FEEDBACK, native lighting gap audit and testing policy. Applied writing-standards for this report. This component review does not close the product destination.

## Result

**One fresh material source defect found, repaired by the producer and independently rechecked. This is a finding-and-repair pass, not a clean maturation pass.**

### M1 — admitted payload can exceed its own recovery reader

Given a valid maintained service intent with a 40,000-byte APPDATA value, `validateWindowsPayload` accepts it because the shared environment bound is 64 KiB. `prepareWindowsOperatorPayload` marshals and durably writes that payload without enforcing a serialized bound. The scheduler can then register the task, persist RunIssued and Run. However, `readWindowsOperatorPayload` rejects JSON larger than 32,768 bytes. The wrapper cannot bootstrap; lookup, stop and release cannot read their own durable intent. An accepted input therefore strands its native registration and ledger behind ErrOperationIdentity.

This also occurs below 32,768 input bytes through JSON escaping. The independent fixture uses 17,000 tabs in APPDATA, producing 34,961 JSON bytes. The 40,000 plain-byte case produces 40,961 bytes in the observed test workspace. Both passed validation before the repair and failed the required recoverability assertion. The complete serialized payload, including escaping, paths and metadata, must fit the reader before the reservation/native-effect boundary. The producer and parent received the finding before any requested repair.

Independent fixture: `windows-submission-maturation/adversarial_test.go.txt`, loaded by adjacent `overlay.json`. Initial command from packages/platform-go:

```sh
go test -overlay docs/windows-submission-maturation/overlay.json -run '^TestFreshSubmission' -v .
```

Observed exit 1; both plain and escaped oversized admitted records fail. No native operation was executed.

## Fresh source assessment

The normal caller submits through Scheduler COM rather than creating a direct descendant. The registered action is the trusted maintained host itself, and its Windows capsule branch waits on the SDK child. The action retains its exact image and Job handles; the suspended SDK child inherits the image lock but excludes the Job handle. The wrapper joins its own bounded kill-on-close Job before custody publication and before vendor/device/owner-claim effects. The caller's scenario Job implementation is unchanged.

The protected intent reservation and durable RunIssued barrier precede one native Run. An uncertain acknowledgment never authorizes a replay. Bootstrap checks the current boot, same actual user and non-elevation, singleton running task/GUID, action birth relation, mapped immutable approved bytes, exact argv/cwd and root-once publication. Explicit environment normalization occurs in trusted code before vendor admission. The resource owner independently consumes the claim once before child resume. Stop/release serialize against Run/bootstrap; revocation precedes scheduler control. Fresh registered-task Stop is called on every retirement attempt, rather than inferring cancellation from a disabled flag or empty running-instance list. Release requires exact Job absence, root quiescence, task deletion and confirming missing-task observation before ledger deletion.

Natural exit without a persisted terminal receipt remains OutcomeUnknown; it is not fabricated as success. Reboot does not allow old intent replay. Retained empty Jobs are quiescent but not absent. No additional relay, account pivot, restricted-token broker or direct launch fallback was found in this path. Legacy native identity and persisted contracts remain separate from Linux's public/opaque identity.

Microsoft documents [nested Job inheritance](https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs), supporting the original caller-lifetime problem and the need for scheduler-owned ancestry. [RegisteredTask.Stop](https://learn.microsoft.com/en-us/windows/win32/taskschd/registeredtask-stop) stops all task instances, but its documentation does not by itself qualify every pending-queue transition. [Security contexts](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks) require batch-logon privilege for S4U; the implementation must report denial without silently granting rights. [EnginePID](https://learn.microsoft.com/en-us/windows/win32/taskschd/runningtask-enginepid) identifies the task engine, not an independently documented per-action identity. The current parent/birth/GUID association is therefore a native qualification condition. This review uses the specified trusted same-operator/pinned-vendor model; hypothetical malicious owner duplicate tasks are outside it.

## Changed, verified, remaining and unverified

Changed: this report and independent overlay fixture only. No production source was edited by this reviewer. No native task, Job, logon, token, service, device, root or dependency action, target dispatch or Git operation occurred. No background resource or model was started.

Verified: fresh source review, deterministic initial failure of the M1 recovery invariant, and passing affected recheck after repair.

Remaining: another fresh review after the repair is required for a clean component pass. Full product requires its separate acceptance and consecutive clean review gates.

Unverified: actual Windows normal caller-close survival; scheduler topology, EnginePID parent association, task XML/DACL normalization, queue/uncertain-Run cancellation, batch-logon/session-zero operation, natural exit/reboot, native Job/image/access semantics, logged-out boot, selected HID access and physical lighting. Cross-compilation and portable tests cannot prove these.

## Repair and independent recheck

The producer added `windowsOwnedPayloadMaxJSONBytes = 32768` and checks `json.Marshal(p)` in `validateWindowsPayload`; the actual reader uses the same constant. `prepareWindowsOperatorPayload` independently calls the validator before registry access. Both original oversized inputs are now rejected before submission, including the JSON-escaping case. The producer also retained ordinary-payload acceptance coverage. No reader widening or compatibility bypass was introduced.

Commands executed after the repair, from packages/platform-go:

```sh
go test -overlay docs/windows-submission-maturation/overlay.json -run '^Test(FreshSubmission|WindowsOwned|WindowsCustody|WindowsTask)' -v .
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -overlay docs/windows-submission-maturation/overlay.json -c -o /tmp/vrooli-windows-submission-maturation-amd64.test.exe .
python3 docs/windows-submission-review/check_stop.py
```

Observed: the independent M1 fixture and focused selection pass; Windows amd64 test cross-compilation exits 0; actual-source Stop harness passes. Full focused output is `/tmp/vrooli-windows-submission-maturation-focused.log` (ephemeral evidence). The cross-build does not execute Windows-only tests. The focused selection includes the producer's concurrent explicit AllowHardTerminate policy repair, which this reviewer did not originate. Source remains mutable shared work; hashes below identify observed files, not immutable isolation.

Final observed source SHA256:

- `windows_owned_contract.go`: `d545019064e427c8797bcfaeecf9160435c8e35917e29ebf859046a8faccfb69`
- `operation_windows.go`: `8bee9c29e45ff2d906fcaaba62e4264276f56bf040f38414a2017a5378d84c38`
- `windows_owned_task.go`: `a3631baef0a9e0fb64c86dcf88878e4ef95e2cb2386fee933a20dd3a83c12816`
- `windows_owned_task_windows.go`: `942ce07638c3e7cb20cb62a89be22a4d619bb2b278a1f8d33d415e04bd1de6ad`
- `windows_owned_custody_windows.go`: `360a31bee19e7fdbf649cc89a3cf25a1bbbc13ceef8f21ce9e1486146f3bb6fb`
- `process_image_windows.go`: `5fae405a5e1bc774f06188536068e6ae8342ed806030cf285dfedd737ff097ca`

Focused selection executed 33 top-level tests; no skipped test lines.

## M2 — native Stop S_FALSE cannot prove cancellation (parent-origin finding, repaired)

After the M1 recheck, the parent identified an additional exact-API condition. Microsoft's native [IRegisteredTask::Stop documentation](https://learn.microsoft.com/en-us/windows/win32/api/taskschd/nf-taskschd-iregisteredtask-stop) distinguishes S_OK from S_FALSE: S_FALSE means that the user cannot successfully stop task instances. The registered-task Stop wrapper calls `taskDispatch.invoke`, whose shared `taskHRESULT` treats every nonnegative HRESULT as nil. If Automation Invoke supplies HRESULT 1, cancellation inability therefore becomes successful cancellation. A disabled/empty snapshot can then finish retirement using this insufficient proof.

`windows-submission-maturation/check_stop_status.py` projects the actual current registered-task Stop body and actual HRESULT classifier, with only the native Automation boundary faked to deliver S_FALSE. Observed initial FAIL: `S_FALSE cancellation inability became success`. This establishes current source information loss. It does not establish how a particular native Automation server maps the underlying method's successful-status HRESULT: IDispatch may itself normalize it. Merely checking Invoke's outer HRESULT is not enough to claim native Stop returned S_OK without a documented propagation guarantee. The narrow repair should retain the native IRegisteredTask::Stop result, accepting only exact S_OK; shared COM initialization still legitimately uses other success statuses.

The parent and producer were notified before repair. M2 was subsequently repaired and independently rechecked below. The earlier 33-test observations remain valid for their observed source and do not create a clean maturation pass.

### M2 repair and final recheck

The producer now queries native IRegisteredTask explicitly and calls Stop directly, accepting only HRESULT zero. This avoids both the shared classifier and any Automation HRESULT normalization. Independent review checked the IID and 25-slot vtable layout against [Microsoft's SDK header](https://raw.githubusercontent.com/microsoft/win32metadata/main/generation/WinSDK/RecompiledIdlHeaders/um/taskschd.h): Stop is slot 23, after SetSecurityDescriptor; Release is slot 2. The direct call passes flags zero and releases the queried native interface and original dispatch interface.

The current `check_stop_status.py` projects the actual repaired native wrapper, substituting only QI/native syscall effects. It checks the exact IID, Stop slot/zero flags, refusal of S_FALSE, other positive statuses and access denial, acceptance of S_OK, and both reference releases. Observed PASS. The original failing projection had demonstrated the generic HRESULT conversion loss; the retained harness now follows the production repair rather than preserving obsolete production code.

Final rerun: 34 focused top-level tests, zero skips, PASS (0.011s; `/tmp/vrooli-windows-submission-maturation-final.log`); Windows amd64 test cross-compilation PASS. M1's independent original counterexamples remain fixed. The repair's Windows-only syscalls were compiled, not executed. Both M1 and parent-origin M2 are closed at the source/fixture level; native qualification stays open. This finding-and-repair pass remains ineligible as a clean maturation pass.

M2 final source observation: `windows_owned_task_windows.go=692c70b4d7c9bb0b69410fddb5ae41462f7b367d01f9bee584c9f165e7103ba6`; `windows_task_contract.go=20db4e565b725cf1c9e47a70fbab0d7ad90ab25b463392a33654c0d8404dd134`.
