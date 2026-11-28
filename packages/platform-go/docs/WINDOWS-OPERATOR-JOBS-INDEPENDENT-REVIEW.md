# Windows actual operator Job ownership — independent source review

Date: 2026-11-11. Reviewer: jobs_review. Scope: the selected actual-operator, non-elevated trusted pinned-vendor contract in platform-go. This review does not reinstate the rejected restricted-token, alternate-account or private-desktop candidate. Native Windows runtime and hardware qualification remain OPEN.

Initial verdict: four material findings. The repair recheck below supersedes their open dispositions; the initial pass remains historical and does not become a clean fresh pass retroactively.

## Repair recheck — 2026-11-11

Re-read CHARTER.md and reviewed the repaired production call sites and deterministic regressions under the same selected trusted actual-operator contract. Current disposition: WJ-R1 through WJ-R4 are addressed at the source level. No new material finding was identified in this repair recheck. This is a recheck by the original reviewer, not another fresh independent reviewer/pass and not native Windows qualification.

| Finding | Current source disposition | Evidence limit |
| --- | --- | --- |
| WJ-R1 | `windows_owned_submission.go:54` requests QUERY_VALUE and SET_VALUE; the custody writer uses it at `windows_owned_custody_windows.go:101`. | Deterministic access-mask assertion and call-site inspection; real registry handle access remains a Windows check. |
| WJ-R2 | `commitWindowsOwnedCustody` at `windows_owned_submission.go:65` writes pending custody, completes flush successfully, then publishes a digest-bound Ready marker. `readWindowsOwnedCustody` at `windows_owned_custody_windows.go:128` ignores pending data without Ready and rejects mismatched marker/data. | The completion marker may remain unflushed: its visibility proves the prior custody flush completed. Losing the marker after a crash conservatively leaves the submission unclaimed; it does not authorize replay. Paused/failed-flush deterministic fixtures exercise the source-used commit boundary. Actual registry flush/visibility/crash ordering remains unoperated. |
| WJ-R3 | `operation_windows.go:627` verifies the exact protected ledger key, zero values/subkeys and independent native Job namespace absence. Missing Payload maps to NotFound only for that state at line 657. Start retains its named submission mutex while the existing empty slot is rechecked, removed and recreated at lines 598–612. Value-bearing, subkey-bearing or retained-Job states do not gain cleanup/replay authority. | Predicate fixtures prove the source-used state decision; code inspection checks native wiring and mutex ownership. They do not execute Windows registry create/delete/reopen, competing readers, or process interruption. |
| WJ-R4 | Shared `validateWindowsFrozenText` at `windows_owned_submission.go:91` is called by root payload validation and by child preparation before custody/process effects (`process_image_windows.go:110`). | Invalid argv/key/value/path/cwd cases and valid multilingual/non-BMP inputs pass deterministic validation tests. Native UTF-16 launch observation remains open. |

The two stale security/SCM comments identified below were also updated. Reviewed the new empty-slot recovery against existing no-replay paths: any Payload value keeps the original no-replay behavior; retained empty Jobs remain namespace presence; the normal persisted-intent-before-dispatch ordering still holds. Reviewed marker publication against cached custody: the process-local cache is assigned only after the source-used commit returns, and readers continue to independently verify exact native custody. The repair adds no account pivot, privilege requirement, arbitrary-vendor sandbox or fallback process launch.

Reviewer independently ran the four exact repair regressions with `go test -race -count=1 -v -run '^(TestWindowsCustodyRequiresSuccessfulFlushBeforeReady|TestWindowsCustodyPausedFlushCannotBeAcknowledgedByVisibility|TestWindowsPrePayloadInterruptionIsRecoverableOnlyWithNativeAbsence|TestWindowsChildFrozenInputRejectsLossyUTF16Handoff)$' .` in platform-go: PASS, package 1.013s. All four tests and their listed subcases executed without skips. This is Linux-host deterministic source-boundary proof, not execution of the Windows registry/process APIs. The earlier metadata crash-stage tests still retain their narrower scope; they must not be reported as operated native fault injection.

Changed in this recheck: this report only. Verified: repair source wiring and focused race regressions. Remaining: fresh independent maturation and the declared native qualification matrix. Unverified: Windows registry permissions and loaded SID hive/S4U, actual suspended birth/custody transfer, native interruption and last-close behavior, recursive stop/release/absence, scheduler ancestor Jobs, SDK/device output and logout/boot/reconnect/resume. No native effect, service, dependency or target dispatch was performed; no resource remains running from this review.

## Initial material findings — retained history

### WJ-R1 — P1: custody flush uses a handle without the required access right

`windows_owned_custody_windows.go:101` opens the ledger with `registry.SET_VALUE`; `windowsOperatorLedgerKey` adds only `READ_CONTROL` (`operation_windows.go:547`). The wrapper then calls `RegFlushKey` on that handle at `windows_owned_custody_windows.go:113` through `operation_windows.go:709`.

Microsoft explicitly requires the key to have been opened with `KEY_QUERY_VALUE` for [RegFlushKey](https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regflushkey). `SET_VALUE | READ_CONTROL` does not include that right. Thus the documented native contract does not permit this flush; the wrapper can write visible custody and then fail before establishing its in-memory custody. Cross-compilation and pure metadata fixtures cannot catch this native handle-access defect.

Request `QUERY_VALUE | SET_VALUE` for this write-and-flush path. Add a focused access-mask assertion through a source-used registry boundary, and qualify actual custody write/flush on Windows with the real least-privilege handle. Do not add privileges or widen the operator account.

### WJ-R2 — P1: Start accepts visible custody before its durability barrier succeeds

The wrapper publishes `Custody` at `windows_owned_custody_windows.go:110`, then flushes at line 113 and establishes its process-local held state at lines 116–117. Meanwhile `operation_windows.go:240–249` treats a matching readable record plus live handle verification as sufficient to return success at line 268. Neither `readWindowsOwnedCustody` nor `verifyWindowsOwnedCustody` establishes that the writer finished its flush.

The [RegFlushKey documentation](https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regflushkey) explicitly says registry modifications are visible to other processes before flushing, and that the flush returns only after the hive data reaches disk. Schedule: block/fail the wrapper flush after SetStringValue; let Start read and independently verify the still-live wrapper and Job; Start returns successful even though the wrapper has not completed, and may subsequently fail, the required ACK durability barrier. WJ-R1 makes a real failure at this boundary especially plausible, but correcting the access mask alone does not close the race.

Make the receiving success path independently establish durability of the exact accepted custody value before returning, or add a separate completion handshake ordered after the wrapper's successful flush. Keep the durable record binding and native handle checks. Exercise a source-used injected persistence boundary that pauses/fails between custody publication and completed flush: Start must not report completed durable custody from visibility alone. Include native interrupted-write/flush qualification.

### WJ-R3 — P2: interruption between key creation and Payload publication strands the invocation

`prepareWindowsOperatorPayload` creates the invocation key at `operation_windows.go:593`, then publishes Payload at line 606. [RegCreateKeyExW](https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regcreatekeyexw) creates a key with no values. Process termination after creation, or a SetStringValue error, therefore leaves an empty invocation slot even though this attempt never reached native Job/process dispatch.

`readWindowsOperatorPayload` maps the missing Payload to `ErrOperationIdentity` (`operation_windows.go:618–620`). Start refuses it at lines 77–78. Lookup fails, so both Stop's required initial inspection (line 385) and Release's required inspection (line 469) fail before cleanup. The absence path also refuses identity errors through `windowsOwnedReplacedSlot` (`owned_service_release_windows.go:42–56`). No canonical retirement route for this reachable pre-dispatch state exists in the reviewed code. The advertised crash fixtures begin with an already valid payload (`windows_owned_submission_test.go:44–58`), so they omit this boundary.

Model and recover this pre-publication state explicitly under the existing submission mutex. Preserve no replay/foreign-metadata refusal. A cleanup path must independently establish the exact operator slot and native namespace absence, or publish the payload atomically through an appropriate native storage boundary. Add the key-created/Payload-missing and SetStringValue-failed stages to lifecycle recovery fixtures. Merely returning identity failure leaves durable lifecycle recovery incomplete.

### WJ-R4 — P2: child handoff still permits lossy invalid UTF-8 conversion

The root payload now rejects invalid UTF-8 in `windows_owned_contract.go:42–51`. The SDK child factory uses `validateOwnedProcessSpec` at `process_image_windows.go:120–122`, which checks NUL, sizes and shape but does not validate UTF-8 (`operation.go:250–307`). It then converts argv and environment with `windows.UTF16PtrFromString` / `UTF16FromString` at `process_image_windows.go:150–158`. Invalid UTF-8 such as an environment value containing byte `0xff` is accepted by that validator and converted to a replacement character by the Go Windows conversion. Thus the actual explicit child environment/arguments can differ from the bytes accepted by the handoff. An invalid environment key can also normalize into a duplicate after the pre-conversion duplicate check.

Share a Windows launch-input validator between wrapper and child paths and reject malformed input before custody/process effects. Cover child argv and environment values/keys, plus valid non-ASCII round trips. The root-only `TestWindowsOwnedFrozenInputsRejectInvalidUTF8` does not exercise the child path.

## Non-material cleanup

`windows_owned_custody_windows.go:94` still describes a fully restricted vendor, and `process_image_windows.go:333` still calls the ledger an SCM root record. Update these comments to the selected trusted actual-operator Job contract. `windowsOwnedAccountName` remains used by the immutable-code reader's legacy virtual-SID branch; any removal should account for that caller rather than deleting it in isolation.

## Evidence and limits

Read CHARTER.md, SPEC.md, STATUS.md, FEEDBACK.md, the native lighting gap audit, docs/TESTING.md, WINDOWS-OPERATOR-JOBS-CHECKPOINT.md, windows_owned_backend.md, all requested source files, relevant shared validation/identity/storage helpers, and submission/security/contract fixtures. Checked Microsoft primary documentation for the registry and Job semantics. Source observations describe the mutable isolated source tree and do not claim an exact frozen revision.

Existing owner-reported module/race/cross-build/vet receipts retain their source-only scope. This reviewer did not rerun them: no implementation changed, and the material gaps concern omitted native persistence/handle boundaries or child validation. The existing crash-stage test proves metadata classification and replay refusal; it does not execute registry creation, flush ordering, suspended process creation, ACK transport or cleanup. Native execution is required in addition to the proposed deterministic boundary regressions.

Recall used `search-hub query "Windows OwnedService operator Job custody review" --type library,record,skill,doc --limit 5`; source-ledger.scopes was degraded by a timeout. The supported CLI automatically rebuilt its installed cache binary; no repository source was changed by that read. The writing-standards skill was read for report placement/style. No main-workspace edits, Git mutations, service launches, target dispatch, dependency acquisition, logon/LSA/device/grant effects or native Windows execution occurred. No resource or service was started. The sole review artifact is this report; the implementation owner retains work-record/checkpoint operations.

Changed: this report only. Verified: the four source/API-contract findings above and existing fixture applicability. Remaining: implement findings and run focused regressions, then a fresh independent review. Unverified: actual Windows custody/last-close/stop/release, interrupted native stages, loaded profile/S4U, ancestor Jobs, device/SDK output, logout/boot/reconnect/resume and hardware qualification.
