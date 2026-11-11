# Actual-operator policy and storage integration review

This is an implementation-owner follow-up after retiring the dedicated-account
and privileged resource broker design. It records defects and source evidence;
it is not an independent clean pass or native hardware qualification. Retired
broker/capability research remains outside product source in the hashed history
archive.

## Findings and repairs

1. **Ordinary CLI admission was blocked by an obsolete supervisor-self gate.**
   The one-shot claim server still required its caller PID to equal the machine
   supervisor PID, even though public resource start/update now run directly as
   the operator. That gate was removed. Admission instead requires the exact
   current kernel PID, principal, incarnation, non-elevated security facts and
   resolved actual account to match the frozen owner. Fixtures admit an ordinary
   CLI without a supervisor observation and refuse foreign PID/principal/account,
   stale incarnation, elevated authority, unavailable token facts and cancellation.

2. **A global storage override split native owners into different namespaces.**
   Policy/descriptor/lock helpers selected `<global>/native/openrgb`, while the
   generic resource resolver selected another layout. All native mutable classes
   now resolve under the same native root. Linux and other resources retain their
   existing resolver. Unsupported per-class overrides fail early with global-root
   guidance; redundant exact native directories are accepted. Resolution creates
   no directories.

3. **Environment normalization lost custom capsule storage roots.**
   The closed wrapper could return to default storage after its environment was
   cleared. Capsule paths now determine the exact resource root from
   `launch/ID/capsule.json`; the owner descriptor is opened and OS-owner verified
   at that root. Material StateDir, bounded runtime role namespace and IPC endpoint
   must agree. Parsed owner policy retains only ephemeral root metadata, not a new
   persistent registry or descriptor field. Fixtures change the ambient global
   root and prove frozen material, endpoint and owner policy remain in the original
   namespace; cross-root material is refused.

4. **Long custom roots failed only when creating a native socket.**
   Explicit setup now validates the fixed claim filename against the existing
   endpoint byte bound before creating storage. Normal native directory layouts
   fit. An overlong Unicode root receives a shorter-global-root diagnostic.
   Read/status path resolution remains separate for retained-state inspection.

5. **Microsoft Entra operator SIDs were rejected as if they were service IDs.**
   Account validation now accepts canonical local/domain and Entra operator forms,
   including `S-1-12-1`, while rejecting SYSTEM, service, builtin/capability and
   malformed identities. Actual token identity and elevation remain separate
   kernel observations. The [Microsoft user-profile script](https://github.com/microsoft/universal-print-resources/blob/main/Scripts/Convert-UpPrinterToPerUser.ps1)
   recognizes both local/domain and Entra profile SID families.

6. **Linux v1 policy separation had a case-alias gap.**
   A case alias of the newly introduced `native_policy` field could evade the v1
   native-field presence check. The presence check now covers Go JSON's case
   aliases, including null values. Existing Linux field/encoding behavior is
   otherwise retained.

## Remaining integration obligations

- **Enrolled root versus changed ambient root:** an Enabled backend in root A
  must not be mistaken for missing state in root B after a global environment
  change. The accepted contract uses existing boot enrollment metadata to freeze
  root A and rejects mismatch before intent changes, quiescence or cleanup.
  Read-only enrolled-root accessor and ordinary action guards are being integrated
  with the operator-identity owner. A desired reconnect/disable fixture must prove
  no false Disabled or orphaned backend claim.
- **Native normal public update:** current cross-layer fixture proves initial
  boot/retirement/fresh claim/SDK/reconnect/disable. Generic update fixtures use
  Linux native references. A complete native V2 public journey still needs
  qualification, target primary and restoration roles with distinct runtime
  namespaces, empty launcher environment, retained policy publication, and an
  adopted original whose external source has disappeared. Source paths retain
  frozen policies and sealed rollback identity; these observations do not count
  as executing that native journey.
- **Native OS behavior:** actual macOS SDK/Seatbelt/IOKit/user-domain boot and
  actual Windows standard-token HID/Job/AF_UNIX behavior remain unqualified.
  The single Mac transport profile applies precise macUSPCIO and process-fork
  denial before the HID witness, with kernel verification before image handoff.
  The Windows derived tree omits the fixed PawnIO modules and retains immutable
  full DLL closure proof. Neither path claims arbitrary compromised-vendor
  owner-state isolation.

## Evidence

Host policy/transport and Linux permission fixtures passed (0.877s). Resource
transport/same-owner/A10 focused tests passed (0.735s), and the Windows resource
test crossbuild passed. New ordinary-owner and typed custom-root fixtures passed
(0.093s); the early endpoint/class-layout subset passed (0.055s). Linux v1 alias
regression and the final combined source checks are being rerun as integration
continues. No native payload, device, driver, ACL, service or privileged host
effect was performed for this review.

## Retained namespace and external detector follow-up

When native boot metadata is truly absent, an active lighting intent can now
locate only the current exact resource root through retained serving state,
frozen role/selection/native identity, original durable claim and witness.
This is metadata-only namespace evidence; it does not certify liveness or open
HID as the observer. Copied roots, changed home/account/machine/selection,
reused process identity, missing witness and ambiguous initial/update owners
are refused. Installed boot-descriptor evidence remains the preferred route.

External Terminal addition exposed a pre-existing overbroad observer rule:
all source-qualified controllers were required even when their detector was
disabled. Complete enrollment now applies to each enabled detector, before
any access open and again after access/generation proof. Extra attachments of
that detector still fail closed. Terminal-only settings cannot enable the
motherboard detector. This is implementation-owner follow-up, not an
independent clean review or actual hardware qualification.

Terminal rendering remains an explicit normal-path integration finding:
RGBController_AuraUSB initializes addressable zones with zero LEDs and
DeviceUpdateMode skips them. The complete lighting owner must configure and
resize the user-selected zone length before presets/Off can count as applied.
Source/HID admission is not visible-output proof. This isolated resource tree
does not contain the Light Desk SDK renderer; the finding was routed to the
parent for its existing scenario owner, rather than adding a private renderer.
