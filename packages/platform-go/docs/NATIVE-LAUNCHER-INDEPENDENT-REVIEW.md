# Independent native published launcher review

2026-11-11. Workspace: `/home/matthalloran8/Vrooli-native-lighting`.
Reviewer: native_launcher_adversarial. Scope: NEW same-host owned submission from the exact retained current-role launcher after the boot definition is removed; Darwin and Windows source, portable fixtures and shared proof contract. Full delivery charter/SPEC remain the destination. This report grants no native runtime or hardware qualification.

## Verdict

**Material finding NL-1 repaired and independently rechecked.** This pass is not a clean maturation pass because it found a material defect. The unchanged five desired-behavior counterexamples now pass, and an expanded independent check accepts two recorded Go-linked native Mach-O images. No further material source finding was identified in this bounded pass.

## Material finding NL-1: alternate Mach-O loader commands bypass closure checks

Given a native arm64 executable with a foreign loader path, when its load command is LC_LOAD_WEAK_DYLIB, LC_REEXPORT_DYLIB, LC_LOAD_UPWARD_DYLIB, LC_LAZY_LOAD_DYLIB or LC_LOAD_DYLINKER, the maintained-host closure verifier must reject the image. The pre-repair `validateDarwinBrokerMachO` admitted every case.

The new published launcher reader called this verifier on the held image. The pre-repair implementation used Go `debug/macho.File.ImportedLibraries`, which enumerates `*macho.Dylib` values. The standard parser constructs those only for LC_LOAD_DYLIB (0xc); other commands are raw load commands and are not inspected by the verifier. Therefore a correct SHA, root publication and native architecture do not establish the promised system-loader closure. This is a qualification defect; the demonstrated source issue does not depend on a malicious root, historical selection or changing the admitted account.

Retained desired-behavior fixture: `native-launcher-independent-review/adversarial_test.go.txt`; overlay: `native-launcher-independent-review/overlay.json`; observed pre-repair output: `native-launcher-independent-review/before.log`. All five subtests failed because the verifier returned nil. Native dyld execution was neither required nor attempted. Sent immediately to platform owner and Darwin producer; production files remain producer-owned.

Reproduce from `packages/platform-go`:

```sh
go test -overlay docs/native-launcher-independent-review/overlay.json -run TestIndependentNativeLauncher -v .
```

Repair: Darwin producer replaced the convenience inspection with `darwin_macho_closure.go`. The parser reads the held file directly, bounds command bytes/counts and universal tables, validates loader strings, accepts qualified system dylib forms and exactly `/usr/lib/dyld`, rejects embedded loader environment/unknown load forms, and inspects every universal slice.

Independent recheck: the original five counterexamples pass unchanged under `-race`; affected owner closure/publication/maintained-launch cases also pass (`native-launcher-independent-review/after.log`). An expanded overlay separately admits existing Go-linked amd64 and arm64 Mach-O test images without executing them (`expanded-after.log`, exact input identities in `recorded-go-images.sha256`). These image paths are retained diagnostic inputs in `/tmp`, not hermetic checked-in test dependencies and not native execution qualification. Final reviewed source identities are in `source-after.sha256`.

## Reviewed source contracts and positive observations

- Optional proof has closed version/profile/digest vocabulary. Canonical publisher path binds collection, lowercase digest and exact executable name; path substitutions fail before native opening. No directory search or latest pointer is introduced.
- Service-to-operation projection preserves proof, and the intent specification digest includes it. Legacy nil proof remains omitted in both persisted JSON and canonical digest. Owner golden fixtures cover exact old bytes and proof downgrade/substitution.
- Darwin's nonnil branch consults the published image directly; the nil branch retains installed-host validation. The closed capsule entrypoint remains enforced. Existing owned backend still admits durable intent before dispatch, refuses replay, keeps account/boot/label/invocation identity and does not turn uncertain submission into absence.
- Windows Start and independent early task bootstrap both call `openWindowsOwnedServiceSpecHost`; nonnil proof selects published image without installed task APIs. Bootstrap independently checks actual operator security, exact arguments/working directory, boot, mapped image, task instance and named Job custody. The root boot task is not the new proof branch's authority.
- Windows publication uses canonical ProgramFiles, verified parent handles and a final handle denying write/delete sharing. Digest and real PE import parsing use the held file. Parser rejects foreign machine/DLL forms, delay imports, CLR, foreign DLL names, out-of-section RVA reads, descriptor truncation, overlapping virtual mappings and missing terminators. It no longer relies on the unimplemented Go PE ImportedLibraries method.
- Both openers return descriptor custody through the common rewind/cancellation boundary. Darwin reader checks the named/opened/final object and sealed metadata before return.

Independent execution of the focused owner regressions with `-race` passed; exact selected names/subtests and output are retained in `native-launcher-independent-review/focused.log`. The selection covers shared proof/legacy bytes, Darwin new-role creation after boot removal, image substitution/ACL fixtures and actual Windows PE closure/ABI rejection. Passing these tests did not expose NL-1; the additional fixture did.

## Changed, verified, remaining and unverified

Changed: this report, retained independent overlay fixture/configuration and logs, under `packages/platform-go/docs` only. No production code, Git operation, dependencies, native task/Job/account/service, root state, dispatch or device effects.

Verified: the selected portable focused tests passed under the race detector; five independent alternate Mach-O loader counterexamples fail before repair and pass afterward; two real Go-linked Mach-O input formats pass afterward; source paths above were inspected. Search Hub recall returned useful owner records with an explicit agent-manager semantic-retrieval degradation; no capability absence was inferred.

Remaining: fresh independent maturation following this repaired finding, and parent-owned resource retained-current-role integration acceptance. Concurrent resource wiring was not represented as an integration defect. This report cannot close the full goal or count as a clean pass.

Unverified: actual Darwin CGO/SDK/Seatbelt behavior, Windows native ACL/Task Scheduler/S4U/Job/loader behavior, logged-out/boot/restart/update hardware operation and physical lighting. Cross-compilation and fixtures cannot establish those outcomes. No resource/service/model was started by this reviewer.

## Fresh bounded followup: shared metadata and dylib-use policy split

The Darwin producer requested a fresh review after moving the parser to `native_owned_macho.go`, exporting `ReadMachOLoaderMetadata(reader, size)` for the vendor artifact consumer, and adding macOS 15 `dylib_use_command` support. The charter was reread. This is a separate followup to the finding-bearing pass above.

**No material source finding identified in this bounded followup.** The shared reader reports loader facts for executable/dylib/bundle images; it does not grant trust. The maintained-host wrapper independently enforces executable type, native architecture, system dependency roots, exact system dynamic linker, system RPaths and no embedded dyld environment. The split preserves NL-1's rejection policy.

Independent new probes construct alternate load/weak command bytes directly. For both commands and each known flag combination 0–15, they verify exact command/flags/vendor-path metadata, then verify the maintained-host policy rejects the vendor dependency. For dylib and bundle file types, metadata admission remains distinct from maintained-host executable admission. Unknown high flag bits and reads past the explicit supplied extent are refused. A separate environment probe proves the shared reader reports the exact embedded environment string while the maintained-host wrapper rejects it. The original five NL-1 counterexamples and two recorded Go-linked native image format inputs also pass.

Apple's current [cctools loader header](https://raw.githubusercontent.com/apple-oss-distributions/cctools/main/include/mach-o/loader.h) confirms the alternate encoding's marker, name offset, two command IDs and four defined flag bits. The code's known mask matches that primary definition. The shared reader retains those modifier facts for the consumer instead of discarding them.

Evidence: `native-launcher-independent-review/shared-policy-followup.log` (independent probes PASS, race1.021s); `shared-policy-owner-followup.log` (independent probes plus affected owner tests PASS, race1.099s); final parser identity `shared-policy-followup.sha256`. The retained overlay now includes the new independent probes. Existing owner syntax/universal-table tests were inspected alongside production bounds checks.

Changed: appended this followup and added retained independent probes/logs/source identity under docs only. Verified: the shared policy split, supported alternate metadata preservation, explicit extent, unknown-flag refusal and maintained-host rejection behavior above. Remaining/unverified: full vendor bundle consumer closure, actual macOS SDK/dyld execution, native Windows/runtime/controller/hardware and full-goal maturation. This source-only bounded clean followup is not a full-destination clean pass or qualification. No native effect, dependency installation, process execution of foreign images, service/model or device action occurred.
