# Windows submission fresh D review

Independent reviewer `windows_submission_fresh_d`, 2026-11-11. Reviewed the current isolated native-lighting source against delivery CHARTER/SPEC/STATUS/FEEDBACK, the native lighting gap audit and docs/TESTING.md. Read writing-standards for this report. The full cross-platform, headless and physical product destination remains active.

## Result

No new material source finding was established in this fresh pass. This is a clean **component source/fixture review** after the three earlier finding-bearing passes and their repairs. It is not native Windows execution evidence, a clean full-product acceptance pass, or product closure.

The review started with the current lifecycle source and resource capsule callers. Earlier review reports were then read to distinguish existing repairs from fresh findings. The accepted same-operator, immutable maintained host and pinned vendor model was retained. Hypothetical malicious owner replacement tasks were not added to the threat model.

## Source assessment

Initial and managed-update specs select the same closed maintained capsule entrypoint. The early dispatcher performs a bounded typed capsule/owner-file read, then calls Windows submission preparation before owner IPC, device witness, vendor process creation or once-claim. The caller does not create a direct wrapper child. It reserves complete protected intent, registers a create-only invocation-specific S4U/LUA task, persists RunIssued and makes one Run call. Any admitted/uncertain stage is nonreplayable.

The trusted native task action establishes its own custody. Preparation checks frozen identity, current boot, actual same user/non-elevation, mapped approved image bytes, exact arguments/cwd and singleton task/GUID/engine-birth relation. It creates and joins its bounded kill-on-close Job before durable root and custody publication. Registry visibility alone cannot replace the custody flush barrier. Environment normalization precedes vendor work. The wrapper retains image and Job handles while waiting for the SDK child. The suspended child inherits its read-only image lock and Job membership; the explicit inherited handle list excludes the Job handle. Child native identity/security/containment facts are bound into the durable owner claim before Resume. An uncertain claim response aborts the still-suspended child.

The owner transport observes the actual peer, exact root incarnation, current Running/custody evidence and a confirming lookup before material disclosure and claim. The server consumes its attempt once before the durable claim callback, and the wrapper closes that session before vendor resume. A failed/uncertain claim does not provide a retry path. Stop serializes revocation against submission/preparation before exact task cancellation. Release retains its ledger until authoritative task retirement and Job namespace absence complete. The production code does not alter caller Job containment or introduce account pivots, relay lifetime, direct launch fallback or capability-token bootstrap.

Same-boot release distinguishes a retained empty Job from absent namespace and checks an established root's exact birth before mutation. A reused numeric PID does not retain the old root's authority. Cold-boot retirement preserves the original ref/claim, requires exact Job absence, clears obsolete task-instance/PID authority only for verified task cancellation, and never signals an old PID or adopts a new Job under the same name. Missing-task classification remains narrow; credential/permission/configuration errors remain errors. Native registered-task Stop accepts exact S_OK, while the general COM initialization classifier retains its separate success semantics. Natural exit without a durable terminal record remains OutcomeUnknown.

The native lifecycle uses one protected frozen ledger, one task action and one Job custody owner. No additional roster or parallel recovery owner was found. The portable public ServiceRef and Linux legacy identity paths were not widened by this task-submission repair. Existing source helpers separate pure identity/policy decisions from native effects, permitting focused counterexamples without operating the host.

## Independent desired-behavior checks

`windows-submission-fresh-d/check_release.py` extracts the **actual complete current releaseOwnedOperation body**, renames native boundary calls and compiles it into a temporary overlay with hermetic fakes. It does not copy an older algorithm or run Windows APIs. Unlike the previous cold-boot harness, these cases exercise same-boot release and its interruption boundaries.

Given a same-boot retained role, when any of initial Job observation, root birth observation, revocation, task Stop, final Job observation, task deletion or ledger deletion returns an unknown error, then Release must return that error and perform no later boundary effect. All seven cases passed. The separate safety cases passed:

- A retained empty named Job prevents any retirement mutation.
- An exact live root prevents revocation/deletion even if the Job is reported absent.
- A reused PID with a different birth does not prevent retirement of the old registration.
- A Job appearing after task cancellation prevents task/ledger deletion.
- A caller ref with a different root PID is rejected before native retirement effects.

Command from the native-lighting tree:

```sh
python3 packages/platform-go/docs/windows-submission-fresh-d/check_release.py
```

Observed PASS: two top-level tests, twelve subtests, zero skips; projected package execution 0.005s. This establishes current source control flow with faked native boundaries. It does not establish task cancellation or Job semantics on Windows.

Also executed from `packages/platform-go`:

```sh
go test -count=1 -run '^TestWindows(OwnedRebootRetirement|TaskAdmission|TaskRejects|OwnedPayloadBoundsSerialized|Custody)' -v .
```

Observed PASS: eight top-level tests, zero skips, 0.005s. The actual selected tests cover custody birth/Job binding, the pending/flush/ready barrier and paused-flush visibility, durable once-Run ordering, closed entrypoint refusal, escaped serialized payload bounds, cold-boot retirement and foreign XML attribute namespace refusal. No broad unchanged module suite or native live fixture was rerun.

## Observed source identity

The isolated tree remains shared mutable source; these hashes identify observed bytes, not an immutable runtime qualification:

| File | SHA256 |
| --- | --- |
| operation_windows.go | 3d7084eeb5fbb8756c8472b7cf58fde51221ff4ad9ebb10062347e3e3ddce5c3 |
| windows_owned_task.go | e542592436ef8d3ea8bdb3c0b4487ed042c276470a936bd2161b6892677e0cb2 |
| windows_owned_task_windows.go | 692c70b4d7c9bb0b69410fddb5ae41462f7b367d01f9bee584c9f165e7103ba6 |
| windows_owned_custody_windows.go | 360a31bee19e7fdbf649cc89a3cf25a1bbbc13ceef8f21ce9e1486146f3bb6fb |
| process_image_windows.go | 5fae405a5e1bc774f06188536068e6ae8342ed806030cf285dfedd737ff097ca |
| windows_owned_contract.go | d545019064e427c8797bcfaeecf9160435c8e35917e29ebf859046a8faccfb69 |

## Changed, verified, remaining and unverified

Changed: this report and its adjacent hermetic source-projection fixture only. No production source, main checkout, dependency, Git state, task, Job, account, token, service, device, root/elevation, native dispatch or host configuration was changed. No long-running resource, service or model was started. The parent owns the aggregate work-record journal.

Verified: the source paths assessed above, the twelve new same-boot Release counterexamples and eight focused existing portable tests. Existing producer cross-build evidence was read, not independently rerun or promoted to runtime proof.

Remaining: another fresh independent component pass is needed for two consecutive clean component source reviews. Full acceptance remains governed by the complete product denominator and its independent review/challenger gates.

Unverified: real Windows S4U/batch-logon/profile behavior, exported XML/DACL normalization, EnginePID/action ancestry, uncertain Run and queued Stop cancellation, nested Job membership and exact handle semantics, outer caller Job-close survival, owner-death/fence recovery, cold boot/reboot recovery, logged-out/headless lifecycle, selected HID access, SDK execution, device return/resume and visible Off/static. Those native and physical outcomes remain OPEN. The opt-in native outer-Job/reboot fixtures were not executed.
