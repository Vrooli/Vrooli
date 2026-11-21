# Windows submission fresh E review

Independent reviewer `windows_submission_fresh_e`, 2026-11-11. Scope: the current isolated native-lighting tree's maintained Windows owned-role lifecycle and its initial/update capsule callers. Read the delivery CHARTER/SPEC/STATUS/FEEDBACK, native lighting gap audit, testing policy and writing-standards. The operator's full portable, headless and physical delivery destination remains active.

## Result

No new material source finding was established. This fresh pass is a clean **component source/fixture review**, following fresh D on the same observed stable production source. Together D and E meet two consecutive clean reviews for this component's source scope only. They do not qualify Windows execution, headless boot, lighting hardware, full-product acceptance or closure.

The pass independently traced production lifecycle and capsule source, then read earlier review/repair reports to distinguish already repaired conditions. It retained the accepted trusted maintained host, pinned vendor and same actual user model. It did not introduce hostile owner duplicate-task actors or change that model.

## Source assessment

Normal Start refuses generic service executables before effects. Initial and update callers use the maintained early capsule action. The protected complete intent/specification/boot/account ledger, create-only UUID-specific S4U/LUA task registration and durable RunIssued barrier precede one Run. Admission, registration, Run or acknowledgement uncertainty retains the original intent; it cannot authorize replay. An empty pre-Payload key is recoverable only after its values/subkeys and native Job namespace are independently absent.

The task action itself is the waiting wrapper. Its early typed capsule read precedes submission preparation; preparation checks current boot, actual user/non-elevation, approved immutable installed-equivalent image, exact argv/cwd and stable singleton scheduler GUID/engine-birth relation. It joins its own bounded kill-on-close Job before durable root/custody publication, owner IPC, device witness or vendor execution. Environment normalization occurs in this trusted wrapper. Start waits for pending custody to complete its exact flush/ready barrier and independently verifies live root, birth image and held Job. The caller has no direct wrapper CreateProcess fallback and its scenario Job is unchanged. Actual scheduler ancestry, S4U/profile behavior and caller-close survival remain native qualification conditions.

Lookup preserves frozen identity without claiming liveness. Inspect distinguishes populated, root alive, proven Running and terminal outcome. Native custody verification pins process birth before interpreting inherited image/Job handles and closes query leases. A partial prepared record or missing durable terminal receipt does not invent healthy operation or SDK success. Same-boot reused numeric PIDs do not regain root authority. Root/terminal reader schemas are trusted producer records with bounded reads and subsequent identity checks; observed bytes alone are not liveness evidence. Capsule transport separately uses strict bounded typed frames and rejects unexpected fields/operation shape before admitting a claim.

The owner observes the actual peer and exact root, requires Running/custody and confirms Lookup/incarnation before material disclosure and once-claim. It consumes its attempt before the durable callback, so an uncertain response provides no second admission. The Windows vendor child is created suspended with exact explicit environment/argv/cwd under the actual standard user; its inherited handle list contains its own image lock, excluding the Job handle. Its native birth/image/security/Job facts enter the durable resource claim before Resume. Failed or uncertain claim aborts it while suspended. The wrapper closes owner IPC before Resume and waits for the child beyond the bounded admission context; it does not exit as a relay when admission ends.

Stop and Release serialize against task submission/preparation, preserve unknown native errors, and revoke admission before exact task cancellation. Registered-task Stop accepts exact native S_OK and runs on every retirement attempt. Empty instance collections do not make visible queued/running state quiescent. Release retains its ledger until task cancellation/deletion and named Job absence are established; retained empty Jobs remain present. Cold-boot retirement keeps the old ref/claim, requires Job absence before and after cancellation, and does not query or signal the old PID. A newly occupied same-name Job refuses retirement. Narrow missing-task classification excludes credential/configuration failures.

No new roster, account pivot, capability-token bootstrap, root broker, parent-containment weakening or parallel recovery owner was found. Existing pure identity/policy helpers support these checks without requiring Windows effects. Public Linux cgroup/legacy identity validation remains separate from platform-tagged native identity; this pass found no source path widening the Linux legacy contract. Existing producer module/race/cross-build results were read as prior evidence, not rerun or promoted to native proof. No production change justified repeating those unchanged broad checks.

## Independent desired-behavior checks

`windows-submission-fresh-e/check_inspect.py` extracts the complete **current actual inspectOwnedOperation body** and compiles it in a temporary Go overlay, renaming only native boundaries to hermetic fakes. It does not copy an older algorithm or execute Windows APIs. This differs from D's Release projection and C's cold-boot Stop/Release projection.

Given retained identity, these desired outcomes passed:

- Unrecorded natural exit stays OutcomeUnknown with ExitKnown false.
- Reused PID birth is not RootAlive; a changed boot never queries the old PID.
- Populated but merely prepared submission is not Running.
- Claimed metadata still requires native verification; failure cannot produce Running.
- Exact verified live custody produces Running without inventing an exit.
- Exact terminal evidence distinguishes success and nonzero exit; foreign terminal/custody is an error.
- A foreign caller root ref is rejected before further native observation.
- Each unavailable Lookup, payload, population, boot, root-birth, root, custody, native-verification or terminal boundary returns its error and stops subsequent boundary reads. Both populated and empty paths are exercised.

Command from the isolated tree:

```sh
python3 packages/platform-go/docs/windows-submission-fresh-e/check_inspect.py
```

Observed PASS: two top-level tests, 26 subtests, zero skips; projected package execution 0.006s. These are current source control-flow checks with faked native observations, not native Windows behavior.

Focused existing portable checks from `packages/platform-go`:

```sh
go test -count=1 -run '^TestWindows(OwnedTaskHasNoBootRetryOrImplicitLifetimeLimit|TaskArgumentsPreserveTrailingSlashesAndQuotes|TaskActionIdentityDoesNotAssumeEngineIsAction|TaskInstanceReplacementInvalidatesObservation|TaskAbsenceRejectsCredentialAndManagerFailures|TaskStopRequiresNativeSOK|OwnedInterruptedSubmissionNeverReplaysOrInventsCustody|OwnedRefBindsExactContainment|OwnedTerminalEvidenceBindsExactRoot|OperatorSecurityProofBindsActualUserAndRejectsElevation|PrePayloadInterruptionIsRecoverableOnlyWithNativeAbsence)$' -v .
```

Observed PASS: all eleven named top-level tests executed, zero skips, 0.006s. This checks exact task policy/argv, engine/action separation, instance replacement, error versus absence, native cancellation acknowledgement, interruption/no-replay, native containment/terminal identity, actual-user/non-elevation proof and unpublished-slot recovery. It executes no live native qualification test.

## Observed source identity

These hashes identify observed bytes in shared mutable source, not an isolated runtime artifact. The first six match fresh D's observations.

| File | SHA256 |
| --- | --- |
| operation_windows.go | 3d7084eeb5fbb8756c8472b7cf58fde51221ff4ad9ebb10062347e3e3ddce5c3 |
| windows_owned_task_windows.go | 692c70b4d7c9bb0b69410fddb5ae41462f7b367d01f9bee584c9f165e7103ba6 |
| windows_owned_task.go | e542592436ef8d3ea8bdb3c0b4487ed042c276470a936bd2161b6892677e0cb2 |
| windows_owned_contract.go | d545019064e427c8797bcfaeecf9160435c8e35917e29ebf859046a8faccfb69 |
| windows_owned_custody_windows.go | 360a31bee19e7fdbf649cc89a3cf25a1bbbc13ceef8f21ce9e1486146f3bb6fb |
| process_image_windows.go | 5fae405a5e1bc774f06188536068e6ae8342ed806030cf285dfedd737ff097ca |
| internal/resources/openrgb_native_capsule_exec.go | dcf3be2749da32c85b2e994f86226c9c89e2a03403edd058f49ef8a3634dd549 |
| internal/resources/openrgb_native_capsule_transport.go | 261043e8c2518270b24ba8c9ccacc4f8320c9035f60e66d6ad3d21a89f500533 |

## Changed, verified, remaining and unverified

Changed: this report and its adjacent independent hermetic fixture only. No production source, main checkout, Git operation, dependency, native task, Job, account, token, device, service, root/elevation, host configuration or target dispatch was changed. No long-running resource, service or model was started. The parent owns the aggregate journal. Required discovery succeeded with source-ledger.scopes degraded by a timeout; that limits recall completeness, not this source review's directly inspected boundaries.

Verified: the source paths described above, 26 new current-Inspect projection subcases and eleven selected portable tests. No new material component source defect was found.

Remaining: authorized native platform qualification and full destination acceptance. D/E's consecutive source-clean result is deliberately limited to the Windows submission component; it cannot satisfy the charter's full-product adversarial passes or fixed full-destination challenger.

Unverified: real task engine/action topology, Task Scheduler XML/DACL normalization, queued/uncertain Run cancellation, S4U batch-logon/profile/session-zero behavior, nested Job limits/custody and exact inherited handles, ordinary outer caller close survival, wrapper/owner death and recovery, real natural exit, reboot/normal-disable recovery, logout/headless boot, selected HID access, SDK execution, resume/replug/device return and physical Off/static. The opt-in native qualification fixture was read as source only and never executed.
