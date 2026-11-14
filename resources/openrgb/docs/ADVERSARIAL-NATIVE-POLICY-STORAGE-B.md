# Independent native policy and storage review B

Scope: isolated `/home/matthalloran8/Vrooli-native-lighting`; native HID selection, policy union, operator identity, storage enrollment/retained namespace, owner descriptor and capsule file materialization. Read the delivery CHARTER, SPEC, STATUS and FEEDBACK before review. This review does not cover the parent's launcher execution or lifecycle/update review and does not certify the full product.

## Result

No new material source finding in this scope after independent counterexamples. The already reported Terminal addressable-zone rendering issue remains assigned outside this review. No production source changed.

The reviewed model is the pinned trusted SDK running as the actual ordinary operator, matching Linux's trust model. It does not promise isolation from a hostile process sharing that operator identity. Darwin's narrow process-fork/macUSPCIO restriction and Windows's derived six-module omission are separate transport limits; neither is a hardware-output qualification.

## Source checks and counterexamples

- Offline pinned source `8e16de27858cf2be6af54a33ce003df74cb9fd3e`, retained at `.vrooli/work/openrgb-headless-source-qualification/source/AsusAuraUSBControllerDetect.cpp:450` and `:557`, registers five Motherboard identities and Terminal `0b05:1889`. Only motherboard `19af` has the PU predicate. `internal/hostreqkit/openrgb_native_policy.go:128` and `:316` preserve that distinction. New mixed-device fixtures exercise nonzero unrestricted usage pairs as well as the constrained motherboard interface.
- `internal/hostreqkit/openrgb_native_policy.go:250` checks all attachments reached by each enabled detector before any access callback. Mixed Terminal/Motherboard enrollment succeeds; omission of an enabled detector's present interface fails before access. Appearance of a matching interface during successful access invalidates the proof at `:302`.
- Metadata-only observation leaves `CurrentScopeVerified=false` at `internal/hostreqkit/openrgb_native_policy.go:311`. An actual denied open remains denied despite a valid policy/account. Windows identity queries elevation, group authority and current user rather than treating SID shape as token proof (`openrgb_native_windows.go:30`). Claim admission separately checks PID, incarnation, kernel principal and account (`internal/resources/openrgb_native_claim_owner.go:30`). These checks do not substitute for actual HID access.
- Enrolled root A rejects ambient root B with an exact restoration diagnostic (`internal/resources/openrgb_native_storage_enrollment.go:60`). Missing boot enrollment while active requires retained serving evidence, rather than assuming absence (`:30`). Existing retained fixtures independently passed mutations for copied root B, foreign home/account/machine/selection/artifact, reused PID, missing witness, absent native policy and ambiguous owners. Retained evidence locates the namespace only; `internal/resources/openrgb_native_storage_retained.go:81` does not claim a live process or grant device access.
- Capsule namespace derivation and material checks retain the frozen root when ambient resource/HOME variables are empty. Config and runtime directories remain under the frozen custom root. A foreign StateDir fails materialization before reading an owner or creating directories (`internal/resources/openrgb_native_capsule_files.go:157`). Missing, empty, wrong-token and duplicate explicit environments fail material validation.

## Changed

- `internal/hostreqkit/openrgb_native_policy_adversarial_b_test.go`: mixed enabled-detector completeness, metadata versus access denial, and matching attachment appearing during access.
- `internal/resources/openrgb_native_storage_adversarial_b_test.go`: foreign-StateDir rejection before filesystem effects; frozen custom-root material with empty ambient metadata; missing/ambiguous explicit environment rejection; a disposable native-only real ACL/materialization fixture for subsequent native execution.
- This report.

## Verified

Every command set both `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` to the isolated tree.

1. `go test ./internal/hostreqkit ./internal/resources -run TestAdversarialB -count=1 -v`: pass (hostreqkit 0.003s, resources 0.100s). The actual native ACL/materialization fixture explicitly skips on Linux; four platform-independent adversarial fixtures pass.
2. `go test -race ./internal/hostreqkit ./internal/resources -run 'Test(Native(HID|External|Operator|OpenRGB|Enrolled|Global|Unknown|Retained|Capsule|Endpoint|ClaimOwner)|AdversarialB)' -count=1`: pass (1.022s and 7.138s), including existing retained-root and ordinary/Entra claim-owner counterexamples. This race run preceded addition of the native-only materialization fixture; the final fixture was subsequently compiled for both targets below.
3. Windows amd64, CGO disabled: `go test -c ./internal/resources` passes.
4. Darwin arm64, CGO disabled: `go test -c ./internal/resources` passes. This does not compile/qualify the CGO IOKit implementation.

Recall discovery returned results with degraded `source-ledger.scopes` (deadline); it was not treated as a complete evidence inventory. The source/fixture conclusions above come from direct isolated-tree reads and tests.

## Remaining and unverified

Execute `TestAdversarialBNativeCustomRootMaterializationWritesOnlyFrozenTree` on actual ordinary macOS and Windows accounts to qualify platform ACL/publication behavior. Native IOKit/SetupAPI/token behavior, kernel enforcement, HID opens, SDK rendering, physical output and boot/resume remain unqualified here. Cross-compilation and metadata fixtures establish none of those claims. No resource, service, root operation, installation, hardware probe or device effect was performed. Temporary cross-build test artifacts were removed. Parent should include this review in its consolidated work record.
