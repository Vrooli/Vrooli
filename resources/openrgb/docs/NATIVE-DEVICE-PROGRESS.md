# Native selected-device integration checkpoint

Native OpenRGB follows the same authority model as Linux: a trusted pinned
vendor backend, bounded qualified detection/control, and the actual operator.
It does not require a sandbox against arbitrary vendor compromise. Explicit
privileged setup publishes immutable code. Ordinary launch creates no account
and uses no continuous privileged resource broker.

## Current source

- Linux policy schema 1 and serialized bytes remain unchanged. Linux amd64 and
  arm64 permission owners use the same sysfs/stat/ACL/udev implementation. ARM
  admits only the qualified Aura motherboard HID scope; its I2C transaction
  qualification remains open.
- Native policy schema 2 binds machine identity, actual operator uid:/sid:,
  exact admitted source artifact, detector catalog and every matching selected
  HID interface. Metadata discovery does not certify device access. The fresh
  owned wrapper must open all selected interfaces under its actual non-root or
  non-elevated execution token and recheck generation before consuming its claim.
- Native mutable paths retain their canonical machine resource namespace, so
  candidate/claim records are not silently relocated. Owner descriptor version
  2 binds that namespace to the actual logical-home OS owner. Old dedicated
  account descriptors fail explicitly instead of being rebound. Files are
  private to the actual operator; same-user ACLs are not called vendor isolation.
- A global storage override consistently routes all native mutable classes to
  its native/openrgb namespace. Per-class values can only repeat those exact
  canonical native directories; other layouts fail early with global-root
  guidance. Setup checks the fixed IPC filename against its byte bound before
  creating directories. Read/status resolution stays available for inspection.
- The closed wrapper derives its resource root from the exact capsule
  launch/ID/capsule.json path and binds material StateDir/runtime/IPC to that
  namespace. Parsed owner policy retains that root as ephemeral metadata, so
  environment normalization cannot silently return it to the default root.
- Windows uses a pinned derivation omitting exactly six PawnIO module files.
  All remaining DLLs, including PawnIOLib, remain in the verified immutable
  closure. Transport verification requires module absence; permission errors
  or unknown failures do not count as absence.
- Darwin's inherited transport facet denies the exact pinned macUSPCIO
  privileged user client and process forking before device observation and execution. This bounds
  upstream unconditional bus initialization without requiring backend root.
- One-shot capsule/claim, native process identity/Job containment, finite all-
  matching HID selection, generation, configuration and transport bindings
  remain mandatory. Default opt-out performs no native hardware probing.
- A10 retains the existing lighting operation lock through ordinary native
  start/restart operations. Initial Prepare/Claim independently rechecks the
  exact Enabled version/artifact/policy selection.

The additional full-restriction capability-token, device-ACL and planned owner-
state confinement machinery was introduced during investigation and did not
come from a product requirement. It has been removed from production source,
with code, source findings and fixture evidence archived under
`.vrooli/work/native-portability-history/superseded-sameuser-owner-isolation-permissions.*`.
No protected permission receipt store or live native ACL effects were created.

## External HID controller footprint

Native policies also support the external **ASUS ROG AURA Terminal**, USB
`0b05:1889`. The exact release registers `ASUS ROG AURA Terminal` without an
OS guard and without a usage/interface filter. Discovery therefore enrolls
all present HID interfaces matching this detector. An enabled detector must
have every matching attachment enrolled; a different disabled detector does
not widen the opened scope. Settings enable only the distinct selected names.

The pinned [detector source](https://codeberg.org/OpenRGB/OpenRGB/src/commit/8e16de27858cf2be6af54a33ce003df74cb9fd3e/Controllers/AsusAuraUSBController/AsusAuraUSBControllerDetect.cpp)
opens the enumerated HID path and constructs AuraAddressableController and
RGBController_AuraUSB. Its base controller requests firmware/configuration via
65-byte HID reports, and control uses direct-color/effect HID reports. This
closure contains no driver installation, raw bus access or firmware flashing.
The exact four retained native/ARM payloads contain the registration literal;
that corroborates compiled registration rather than actual device behavior.

Pinned source SHA256: AsusAuraUSBControllerDetect.cpp
`a72307848971bfde043fa404e30a7061a7d9921f3ac5efa46e326a9123710221`;
AsusAuraAddressableController.cpp
`2590d7f4618202e0b77343004b9bf019c5fb9a1c9e1eb02aaf66b542316e17d8`;
AsusAuraUSBController.cpp
`c2a69b1195faf7093b59b3af59e18748abfd9f2d70e960e0a1fee497bc1c2805`;
RGBController_AsusAuraUSB.cpp
`bfe8fa92f3ef2b23f8a3199bdc84284c6cd397728a48e18b440c38b390f4dde4`.
The upstream Terminal controller starts addressable zones at zero LEDs; its
mode update skips zero-size zones. A meaningful configured render therefore
requires the lighting owner to persist a user-selected zone length and issue
the supported SDK zone resize before rendering. Its advertised range is
0–500 LEDs per addressable zone. The pinned SDK resize path does not enforce
that bound itself; the client must validate the advertised range, then
reobserve zone/LED counts before reporting success. Resize changes role-local
configuration through SaveConfiguration, not device firmware. Detection/access alone must
not acknowledge applied Off/static output for an unconfigured empty zone.
That normal rendering composition is an open integration check with the
Light Desk owner. Actual Terminal access/output and its upstream
malformed-response behavior remain unqualified. No vendor executable or HID transport was run here.

## Primary transport evidence

Pinned OpenRGB commit 8e16de27858cf2be6af54a33ce003df74cb9fd3e
[start_pawnio](https://codeberg.org/OpenRGB/OpenRGB/src/commit/8e16de27858cf2be6af54a33ce003df74cb9fd3e/i2c_smbus/Windows/i2c_smbus_pawnio.cpp)
checks module existence before opening/reading it, then opens PawnIO; any open
error returns before module loading or caller port/IMC transactions. Source SHA256:
fff0a51f08ff2db9cce6d042b9fe3d74f0ded4bfff44e65fee768c93dbc184e0.

Offline inspection of the exact vendor payload confirms PawnIOLib.dll is a normal
PE import loaded before main. Its SHA256 is
c6e7c2f54643ef9cb51b9a3ec7b0efc9e204803c206f33014e39f281546ea9b4.
Its pawnio_open export uses native \Device\PawnIO with GENERIC_READ|GENERIC_WRITE,
share flags 7 and open options 0. Module omission does not replace immutable DLL
closure and loader-search validation. No payload was executed for this evidence.

## Verification and open qualification

Focused native transport/storage and A10 regressions passed before this removal
(0.756s). Native policy/transport fixtures passed (0.003s). Linux ARM permission
fixtures passed (0.999s). Custom-root/per-class/capsule ambient-change metadata fixtures also passed (0.055s). The retained-root recovery and ordinary-CLI owner checks passed (0.209s);
external Terminal/selection regressions passed in hostreqkit (0.010s) and
resources (0.235s). Scoped native/storage race checks passed (hostreqkit 1.017s, resources 3.100s).
Windows amd64 and Darwin arm64 no-CGO resource test crossbuilds passed;
Terminal exact settings freeze passed (0.035s); historical
capability/ACL fixtures are not qualification of the simpler design.

Actual Windows HID access, Job/process behavior, native boot/logon and hardware
output remain unqualified. Actual Darwin SDK/Seatbelt, IOKit access, user-domain
boot availability, logout/reboot/resume and hardware output remain unqualified.
No native payload or device was executed/probed by this owner. The source and
hermetic fixtures do not replace these native qualification gates.

Retired root-broker and dedicated-account history is also preserved outside
product source under `.vrooli/work/native-portability-history`, with content
hashes. It is not qualification of this runtime design.

## Fresh component review

Independent B found no new material source finding in the adopted native
policy/storage/Terminal scope. Its new counterexamples passed, including mixed
enabled detectors, denied actual access, matching attachment appearance during
proof, frozen custom roots with empty ambient environment, and rejected
foreign-state writes. Combined race checks passed (1.022s/7.138s); Windows amd64
and Darwin arm64 no-CGO test crossbuilds passed. The real native disposable
ACL/materialization test is explicitly skipped on Linux and remains unqualified.
Sequential independent C also found no new material source finding against
unchanged production source (20-file scope digest
`e105e71e660d714d87148486f812f966cdf65ac948e265366600f2ba11fff919`).
C added post-open identity-change, disabled-detector churn, deterministic
generation and retired-catalog cleanup counterexamples. Combined B/C race
checks passed (1.019s/6.395s); Windows amd64 and Darwin amd64 no-CGO test
crossbuilds passed. B+C count as two clean component source passes only. They
cannot substitute for whole integration or native hardware qualification.
