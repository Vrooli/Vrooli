# Operator-account backend dependency assessment

Historical decision input, 2026-11-11. The supervisor subsequently approved the actual-operator backend and fixed six-module Windows derivation described in [the current profile contract](NATIVE-RELEASE-PROFILES.md). The following assessment preserves the facts and alternatives at the earlier decision point; its statements about unchanged dedicated-account source and proposed transformations are historical. I/J cover that earlier source, not the subsequent implementation or the full product.

An actual-operator backend can retain the existing artifact, constructor and library safety properties. Root-owned code publication can remain an explicit initial setup operation. The present Windows six-module ACL scheme, however, depends on distinct operator and runtime principals. A standard token with the same user SID cannot receive a different allow-list through that SID alone.

| Boundary | What can remain | Dependency of an operator-account design |
| --- | --- | --- |
| Serving code and libraries | Complete pinned native payload, Qt/plugin closure and machine-owned immutable ancestors | Keep the code outside operator-writable state. The operator and backend can both have RX; neither receives code replacement or ACL authority. Administrator maintenance remains trusted and explicit. |
| Constructor configuration | Private XDG_CONFIG_HOME and APPDATA set before construction, explicit CLI config child, disabled automatic profiles and remote clients | Bind HOME, working directory and config to the actual operator's canonical resource state. Continue inspecting normalized/frozen config. Do not import ambient application preferences. |
| OS prerequisites | Linux provider/version checks, macOS15 shared-cache checks, Windows10 1809 plus six signed Microsoft14.44 providers/export checks and separate UCRT | These checks do not depend on a dedicated identity. Each real native account/environment still needs runtime qualification. |
| Mutable state and capacity | Existing operator-owned ledger, ordinary user acquisition and private resource state | Same-user processes can access those paths without a second account traversing private HOME. This removes that particular cross-account storage requirement; it does not waive process, claim, lifecycle or authorization checks. |
| Launch image | Immutable complete launcher/image and exact digest | Preserve an admitted open-image boundary. The current no-path helper proves a current system-supervisor slot; its caller contract must be adapted and reviewed if the managed host becomes an operator daemon. Do not replace it with arbitrary executable path lookup. |
| Windows PawnIO isolation | Current source stops before driver calls when its executable-adjacent module cannot be read | Current ACL/receipt code requires a distinct provisioned service SID and intentionally rejects identical principals. It cannot be re-labelled as a same-user standard-token proof. |

## Unchanged official bundle and a standard token

The accepted pinned OpenRGB source review establishes the module-file check/read before `pawnio_open` and module-load/execute IOCTLs. If the same SID can read every module, that early stop disappears. Non-administrator status, an ordinary Job object, an absent registry entry, or a driver being absent at one instant does not replace it.

A driver-denial approach would need the exact backend token, the exact endpoint and requested access, all alternate bus paths, and a maintained denial during startup and device return. An initial absent-device observation has a creation race. A denial test that opens a driver may itself enter the driver's create dispatch; it must not be represented as passive metadata or as proof of no driver open. A standard-token check must also rule out an elevated linked token being used as the backend. No such replacement isolation mechanism is implemented or qualified by this assessment.

Current upstream [PawnIOLib source](https://github.com/namazso/PawnIO/blob/master/PawnIOLib/PawnIOLib.cpp) corroborates an `NtOpenFile` request for read/write followed by separate load/execute operations. This mutable upstream source is not a substitute for the pinned OpenRGB1.1 call-chain evidence. The exact pinned module path and every fallback must remain the basis of a production change. Optional dynamic driver-library loading, if reached before a proposed denial, also needs provenance/search-path proof.

## Candidate: an explicitly derived module-free HID serving profile

A deterministic, governed serving derivation could omit exactly the six pinned PawnIO data modules from the machine-owned serving directory while retaining all 16 PE images unchanged. With a proven executable-adjacent-only lookup and the existing missing-file branch, the immutable absence would stop the bus path before driver open regardless of driver installation or its ACL. This preserves the currently selected HID scope; it does not claim Windows RAM/I2C support.

This must be declared as a derived serving profile, with the complete official envelope and original tree verified first, an explicit six-file transformation, a separately pinned complete derived tree, unchanged payload/library hashes, and immutable ancestry. It must not silently accept an incomplete upstream bundle or reuse the full upstream tree digest. Existing-source adoption must still verify its complete original native closure before derivation. The original installation remains untouched. All six omitted paths must be absent and have no alias/fallback. Production code, recovery, docs, tests, independent reviews and native startup proof would be required before use.

An offline read-only research overlay verified the original retained Windows tree, checked all six exact module hashes/sizes, then computed the canonical candidate record digest without modifying or publishing a bundle:

- Official envelope SHA256: `63291726c6fc84eb479ef2ee752e97e1187f6ce9c49fd27eee40ded588a8172a`.
- Original tree SHA256: `062cd360105968221df877550e8e1c6f4a2cf850df2a70b93d6fde9af62bdbe9` (22 regular files).
- Candidate derived tree SHA256: `c2f9c9df90ef023a39b2d3c5bb06ec0645ebc64d43e21f8333e53e9721f39caf` (16 retained PE images; six fixed modules omitted).
- Unchanged payload SHA256: `e26ae71162795a8abcfb90c05c1572a00ba06ca63b886aca57392a0138963b1e`.
- Research-only overlay test: PASS0.042s. `/tmp/openrgb-operator-hid-profile-research-overlay.json`; source `/tmp/openrgb-operator-hid-profile-research_test.go`. The first draft used the wrong handwritten field name and was corrected before the successful measurement. These temporary files are corroboration, not a production publication or promised durable evidence service.

This candidate would allow both operator and backend to hash every serving byte and could remove the blocked-leaf receipt branch. It requires an explicit strategy/profile decision and new proof. No production source or profile constant was changed for it.

## Other options and qualification limits

A Windows restricted token with an additional restricting SID can, in principle, distinguish a same-user child from the unrestricted operator at access checks. That is a different mechanism from a standard token or Job containment. It would require typed token/ACL/process-creation support and native proof; no new root runtime broker is intrinsically implied, and none was implemented here. The module-free immutable profile appears simpler for the existing HID-only native scope.

macOS launchd can be configured with an operator identity, but actual logged-out boot, environment, session and selected-device access still need observation. A Windows ordinary-account SCM service needs its maintained credential/logon route, or an explicitly supported equivalent lifetime owner; neither an interactive user's current token nor a Job alone proves logged-out boot persistence. The existing platform owner must assess the supported route before claiming this behavior. This assessment does not declare that capability unavailable or request a new secret.

The artifact/constructor/library boundary therefore permits simplification, but changing identity does not itself prove PawnIO isolation, headless service lifetime, logged-out boot, device access, Off/static/effects, recovery or resume. Those remain full destination obligations. Preserve the current source and evidence until the supervisor selects and validates the replacement design.
