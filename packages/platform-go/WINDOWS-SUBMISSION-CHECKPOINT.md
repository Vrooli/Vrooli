# Windows independent submission checkpoint

Outcome owner: Windows ordinary-operator role submission lifetime, source only in this package. Full charter/spec/status/feedback/gap audit and docs/TESTING.md were read. Parent brief's operator stance/full destination remain active. No task/account/logon/service/device/host effects or Windows native test invocation occurred.

## Changed

The direct caller CreateProcess route is replaced with a create-only, UUID-specific same-user S4U/LUA task action whose exact host/argv/cwd are the existing trusted capsule wrapper. No relay, account pivot, broad privilege, scenario Job weakening, separate roster or public native identity change. Source uses installed immutable maintained host bytes as sole image authority; sealed resource launcher copies can differ by path/inode. Removed obsolete global Configure API and retired service verb/stub. Parent owns root/capsule integration outside this package.

PrepareWindowsOwnedServiceSubmission(ctx,intent) binds protected frozen nonrevoked RunIssued intent, exact mapped sealed host/args/cwd/SID/nonElevation/boot and stable singleton task GUID/action relation. It creates/joins its own exact bounded named Job, pins image, persists self root with internal TaskInstance, normalizes frozen environment and publishes existing post-flush custody before resource witnesses/vendor execution. No CLI inherited Job/environment bootstrap remains. API rejects nil/canceled contexts before cached state.

Intent reserve, task registration, durable RunIssued and sole Run are ordered through production-used effects. Every interrupted stage is nonreplayable. Task lifetime policy has no triggers/retries/idle/battery/network/time cutoff. Custom canonical data roots are supported. Arbitrary service/executable shapes fail before effects. Serialized payload validation includes JSON escaping/metadata overhead and shares the reader's 32768-byte bound before reservation.

Stop durably revokes under the same mutex, disables the exact task, calls native IRegisteredTask.Stop and only acknowledges S_OK, confirms no queued/running/active instance, then validates/terminates exact Job. Release refuses established live roles/retained Jobs before mutation, revokes pending admission, cancels task, confirms absence, deletes task and confirms missing before ledger deletion. Credential/configuration/permission errors never become absence. A late bootstrap cannot pass revocation or missing ledger to vendor/HID work.

Native Stop deliberately queries IRegisteredTask and invokes SDK-vtable slot23 directly. Automation Invoke can hide S_FALSE; generic taskHRESULT remains unchanged because CoInitializeEx permits S_FALSE. SDK IID9c86f320-dee3-4dd1-b972-a303f26b061e and full25-slot C vtable were verified from Microsoft's SDK metadata header.

Old-boot retirement: after an independently observed boot change, Stop/Release require exact named Job namespace absence, discard obsolete task-instance/PID authority for task cancellation, and never query/terminate the old PID. Any newly occupied same-name Job fails closed. The frozen old claim/ledger remains until exact task cancellation/deletion and absence complete; no old intent replays. Fresh C review found the previous boot-mismatch refusal would strand ordinary postboot cleanup; repair and focused race fixture PASS. C independently projected complete Stop/Release into hermetic fakes and proved rootless/retained-root oldboot cleanup, no oldPID query/GUID propagation, foreign/new Job refusal and exact release ordering; actual cleanup callers were source-traced. This is another finding-and-repair pass, not clean acceptance.

## Verification

- Latest normal go test -count=1 ./... PASS4.383s incl rlimitexec.
- Root platform package race PASS8.630s, earlier9.002s; focused Windows task/custody/admission/serialized-bound race PASS1.042s, zero skips.
- Full ./... race NOT green: untouched rlimitexec TestRlimitShimAppliesNprocAndAsBeforeExec helper exits66, twice; ordinary focused fixture PASS. Report knw-1794401862033614384 via report-bug skill.
- go vet ./... PASS.
- Six no-CGO test crossbuilds Linux/Darwin/Windows amd64/arm64 PASS. Windows targets recompiled after native harness and bound changes; exact native Stop repair Windows amd64/arm64 crossbuilds PASS.
- Independent first pass repaired F1 wrong HRESULT absence and F2 queued dispatch cancellation; actual-source stop harness PASS.
- Latest serialized-bound/native Stop S_OK focused race PASS1.032s; actual-source queued Stop harness PASS.
- Work-record journal e524c82d-10c6-4992-a834-e845a77e409f records this source outcome.
- Independent maturation repaired M1 oversized serialized durable payload and M2 native Stop S_FALSE ACK loss. M1/M2 final independent recheck passed34 focused tests with zero skips, native Stop syscall-projection harness, and Windows amd64 crossbuild. These are finding-and-repair passes, not clean acceptance passes.
- Explicit opt-in windows_owned_task_live_test.go SOURCE compiles: requires prepared isolated real capsule/immutable host; submitter in outer kill-on-close Job, >=2 role processes, outer close role survival, separate fresh task-stop/custody-loss modes and real owner cleanup. NOT executed.

## Remaining / unverified

Fresh independent clean passes after all repairs; parent integrated resource/root crossbuild and persistent-source classification. Native Windows S4U/batch-logon/profile policy, task DACL/export canonicalization, engine/action topology, queued Run/Stop behavior, actual outer Job-close lifetime, task-stop/custody-loss recursive quiescence, crash/reboot/logged-out/headless/hardware/SDK outcomes remain OPEN. Models/builds never count as native execution. Windows ARM compilation does not imply OpenRGB ARM artifact support.

All edits remain isolated to packages/platform-go; no Git/dependency/host effects. No long-lived resource/service/model was started.

## Primary sources

- https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs
- https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks
- https://learn.microsoft.com/en-us/windows/win32/taskschd/tasksettings-executiontimelimit
- https://learn.microsoft.com/en-us/windows/win32/api/taskschd/nf-taskschd-iregisteredtask-stop
- https://raw.githubusercontent.com/microsoft/win32metadata/main/generation/WinSDK/RecompiledIdlHeaders/um/taskschd.h

## Reopened retained publisher admission — 2026-11-11

Parent explicitly authorized NEW update/restart submission from the exact retained current role's maintained launcher+SHA after boot-task removal. Charter reread; full original stance/destination unchanged. Common NativeLauncherProof/spec/digest/public API/persistent classification is parent-owned. Source-only Windows implementation remains in this isolated package; no root-source/Git/dependency/native effects.

Implemented native_owned_launcher_windows.go private hooks: canonical KnownFolderProgramFiles/Vrooli/OpenRGB/launchers collection, complete root-owned non-reparse broker ancestry pin and final image ACL/no-write/delete lock; independently validate exact SHA, native PE architecture/static loader closure and rewind. Both Start AND Prepare now select OpenPublishedNativeLauncher with nonnil frozen proof; nil uses installed approved-copy branch. Failure never falls back, enumerates historical versions, consults a removed boot task or reads an old role ledger. The parent's common getter proves exact collection/SHA/vrooli.exe path before native access.

Important source defect repaired in the reused Windows broker verifier: Go1.25 debug/pe.ImportedLibraries is TODO and returns nil,nil. The previous allowed DLL loop therefore checked no actual dependencies. native_owned_launcher_pe.go now parses bounded PE import descriptors/name RVAs from the SAME held image FD; verifies amd64/arm64 PE32+ executable shape; accepts genuine absent imports; refuses nonnative/DLL/managed/delay-load images, unknown/path-bearing DLL names, malformed/unterminated/wrapping/overlapping/truncated import evidence. Existing broker import wrapper and installed-copy branch reuse the same parser; there is no duplicate loader policy. Primary PE format: https://learn.microsoft.com/en-us/windows/win32/debug/pe-format#the-idata-section . Native image hash/ACL/ABI does not imply actual Windows runtime qualification.

Portable native opener fixtures replace only PinParents/OpenImage effects; actual SHA and PE parser remain real. They prove retained canonical copy admission without any boot actor, tamper/foreign ABI/unknown DLL refusal, ACL failure propagation without fallback, parent handle release, failed image close and offset0. ActualPE fixtures cover real binary layouts rather than fake DLL name lists. Both real Go-crossbuilt Windows test executable artifacts also passed the parser through docs/windows-published-launcher/built_pe_test.go.txt overlay (0.024s); these artifacts were read, not executed.

Focused PE fixtures/race PASS1.019s; retained-copy/ACL/tamper fixtures PASS0.003s. Windows amd64+arm64 current test crossbuilds PASS. Final normal module/race/vet result below when collected. No native qualification tests ran. No root publication, account/logon/task/service/device operations, downloads, or external execution occurred. Parent retains combined fresh review and integration; this source handback does not close the full destination or count as a clean review pass.

Reopened final checks: normal go test -count=1 ./... PASS8.774s (all subpackages); root package go test -race -count=1 . PASS14.025s; go vet ./... PASS; Windows amd64/arm64 current crossbuilds PASS. Source was shared with parent common-field work during checks, so these observations cover the current combined package rather than an immutable snapshot.

### Retained proof codec integration repair

Parent integration audit found that validateWindowsPayload reconstructed OwnedServiceSpec without NativeLauncher, so its service digest comparison rejected every real proof-bearing NewServiceIntent before publisher opening. Added TestWindowsProofBearingServiceAdmissionAndLedgerCodec and observed FAIL with specification-changed before the fix, then PASS after copying NativeLauncher. The test uses the real proof-bearing NewServiceIntent/processSpec, validates actual Windows payload admission, and roundtrips the strict production registry JSON decoder. Changed/dropped proof, foreign intent and trailing bytes are refused. The unchanged native JSON read logic now lives in portable decodeWindowsOperatorPayloadJSON(raw,intent), directly called by readWindowsOperatorPayload for both lookup and task bootstrap. The test covers that real codec rather than a getter-only or retired base64 environment fixture.

Production conversion audit: OwnedServiceSpec.processSpec preserves NativeLauncher; Windows validation's reverse service-spec construction was the only omitted production copy. Identity-only windowsRef projections intentionally use Intent/digest and observer identity without reconstructing executable specs. The common frozen digest binds proof; parent owns common schema/compatibility. Test fixture reconstruction now preserves NativeLauncher too. Windows amd64/arm64 current test crossbuilds and vet PASS; focused proof/Windows payload race results are retained with this owner handoff. Native/root effects remain absent; combined fresh review/native qualification remain open.

Codec repair final verification: focused Windows/proof race PASS1.066s with actual regression selected and no skips; Windows amd64/arm64 test crossbuilds PASS; vet PASS. Before-fail/after-pass evidence retained in the regression's observed output. No additional persistent fields/constants or common-file changes were introduced by this fix.
