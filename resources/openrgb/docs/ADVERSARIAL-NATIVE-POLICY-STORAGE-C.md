# Independent native policy and storage review C

## Result and applicability

No new material source finding in this component scope. This is a fresh independent pass after B, against unchanged reviewed production files. It is not a whole-product clean review or hardware qualification. No production source changed.

Read the main delivery CHARTER, SPEC, STATUS and FEEDBACK before substantive work. All source reads, fixtures and writes used the isolated `/home/matthalloran8/Vrooli-native-lighting` tree. The accepted model is the trusted pinned SDK under the actual standard non-elevated operator, matching Linux trust. It does not require a dedicated account, root RPC, restricted token or isolation from a hostile process sharing the operator account. Native IDv2, one-shot exact capsule claims and device generations remain required.

The reviewed production scope is exactly:

- `internal/hostreqkit/openrgb_native_{darwin,darwin_nocgo,other,policy,transport,windows}.go`
- `internal/resources/openrgb_device_policy_native.go`, `openrgb_policy_union.go`, `openrgb_native_capsule_files.go`, `openrgb_native_claim_owner.go`
- `internal/resources/openrgb_native_owner{,_darwin,_other,_windows}.go`
- `internal/resources/openrgb_native_storage{,_darwin,_enrollment,_other,_retained,_windows}.go`

The SHA256 of sorted `path<TAB>file-SHA256<LF>` entries for those 20 files is `e105e71e660d714d87148486f812f966cdf65ac948e265366600f2ba11fff919`. All 20 hashes were unchanged at final verification. Adjacent capsule/material, frozen-role, witness and existing fixture code was read to trace call contracts; launcher execution, platform owner implementation, public update tests and Light Desk rendering were not independently reviewed here.

## Independent challenges

- Inspected offline retained `AsusAuraUSBControllerDetect.cpp:148`, `:450–454` and `:557` under `.vrooli/work/openrgb-headless-source-qualification/source`. The Terminal callback opens its HID path and builds `AuraAddressableController`; its registration is unguarded HID VID/PID `0b05:1889`. Of the five enabled Motherboard registrations, only `19af` requires the PU pair. Native qualification preserves those distinctions, including unrestricted Terminal usage values. This supports the source transaction footprint only; the known initial-zero-zone/resize issue belongs to the Light Desk owner and remains open.
- Challenged detector scope with a Terminal-only policy while a disabled Motherboard attachment changes path and generation between enumerations. Only Terminal opens; the unrelated detector does not invalidate or widen the enabled scope. Existing B completeness tests independently reran for missing members of mixed enabled classes and an enabled attachment appearing during access.
- Changed account and machine identity after a successful HID open. Neither case emits a policy/generation witness or verified scope. Changed OS enumeration order without changing devices: stable enrollment and generation digest remain the same. Changed an attachment generation: enrollment stays stable and the generation digest changes.
- Traced Windows token elevation, enabled administrator groups and principal queries; Darwin non-root identity; exact claim-owner PID, incarnation, kernel security principal and account resolution. SID shape is structural validation, never proof of access. Reran ordinary and Entra owner fixtures. Darwin process-fork/macUSPCIO denial and Windows exact six-module omission remain narrow transport claims, not general same-account isolation.
- Challenged retained storage with a catalog/controller that no longer qualifies for launch. Exact structural policy/artifact/selection/witness metadata still locates the old namespace for revocation. A changed process birth, IDv1 native identity, or missing context/generation/claim digest fails. This deliberately proves metadata location, not process liveness. The production path reads and validates the consumed original claim before calling the locator helper.
- Challenged frozen owner paths after ambient root changes. Canonical files under the frozen root remain accepted; sibling-prefix, parent and textual traversal aliases fail without filesystem creation. Malformed retained enrollment values fail on both platform branches. Existing fixtures reran root A→B rejection, default-root changes, missing boot enrollment with active intent, capsule/runtime namespace derivation, explicit environment checks and foreign-state materialization rejection before filesystem effects.
- Traced owner descriptor type/size/OS-owner checks, same-file verification, strict duplicate/unknown/trailing JSON rejection and private-path policy. Capsule readers derive their root from exact `launch/ID/capsule.json`, then compare owner, endpoint and runtime namespace. Materialization derives the root from the frozen bounded runtime role rather than ambient environment. Existing native ACL/materialization B fixture remains unchanged and explicitly skips on Linux.

## Changed

- `internal/hostreqkit/openrgb_native_policy_adversarial_c_test.go`: three independent desired-behavior fixtures for disabled detector churn, post-open identity change, and deterministic generation with enumeration reorder/replacement.
- `internal/resources/openrgb_native_storage_adversarial_c_test.go`: retired-catalog retained locator with five identity/witness mutations; frozen owner, path aliases and malformed enrollment guards.
- This report. No dependency, generated output, Linux legacy policy bytes or production source changed.

## Verified

All validation commands set `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` to the isolated tree.

1. `go test ./internal/hostreqkit ./internal/resources -run TestAdversarialC -count=1 -v`: PASS, 0.003s and 0.051s.
2. `go test -race ./internal/hostreqkit ./internal/resources -run 'Test(Adversarial[BC]|Native(HID|External|Operator|OpenRGB|Enrolled|Global|Unknown|Retained|Capsule|Endpoint|ClaimOwner))' -count=1`: PASS, 1.019s and 6.395s. Native B ACL materialization remains a Linux skip.
3. Final `go test ./internal/hostreqkit ./internal/resources -run 'Test(AdversarialC|OpenRGBNativePolicy|OpenRGBNativeHIDPolicy|OpenRGBNativeTransportClaim)' -count=1`: PASS, 0.003s and 0.283s. This includes existing one-shot claim/uncertainty behavior and Linux/native wire separation fixtures.
4. `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/resources` and `GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/resources`: PASS. Temporary binaries removed after compilation. Darwin no-CGO intentionally rejects HID execution; this does not compile the CGO IOKit implementation.
5. Final source fingerprint comparison: all 20 reviewed production files unchanged.

## Remaining and unverified

No material source finding requires resolution in this scope. Native ACL/publication, Darwin CGO compile/link and execution, Windows SetupAPI/token behavior, device generations on real replug, HID access, SDK rendering, physical Terminal output and boot/resume remain unverified here. Execute the retained B native-only ACL/materialization fixture on ordinary macOS and Windows accounts. The Terminal zero-zone/resize issue remains with its assigned owner. Two component-clean reviews can count only for this unchanged component scope; full SPEC acceptance remains open.

No host install, hardware probe, native SDK invocation, device effect, root action, service action, dependency mutation or Git operation occurred. Hermetic local transport fixtures were closed by their existing cleanup hooks.
