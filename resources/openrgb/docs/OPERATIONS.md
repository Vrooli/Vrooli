# OpenRGB operations

## Default behavior

Merely declaring this resource does not enable it. The root project manifest
does not depend on OpenRGB, and no default setup step installs it or inspects
hardware. Do not add OpenRGB to project `resources`, scenario dependencies,
or required host tools. `resource status openrgb` is read-only. Start requires explicit machine
opt-in, a qualified already-staged release and selected-device enrollment.
Ordinary start/restart reuses the native headless owner; it cannot acquire,
extract or upgrade an artifact.

## Linux amd64 lifecycle

Use Light Desk Settings or the typed control-plane workflow. Setup and device
selection are explicit; disabled status does not discover hardware. Replace the
uppercase device/plan/policy placeholders with identities returned by the owner:

```sh
vrooli runtime lighting setup --source managed --version 1.1 \
  --sha256 d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d --json
vrooli runtime lighting devices --json
vrooli runtime lighting permissions plan --device-id SELECTED_DEVICE_ID --json
vrooli runtime lighting permissions apply --plan-id REVIEWED_PLAN_ID --sudo-mode ask --json
vrooli runtime lighting setup --source managed --version 1.1 \
  --sha256 d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d --json
vrooli runtime lighting enable --source managed --version 1.1 \
  --sha256 d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d \
  --device-policy-id ENROLLED_POLICY_ID --json
vrooli runtime lighting status --json
vrooli runtime lighting disable --json
```

Setup and permission changes require disabled lighting intent; disable first
when lighting is active. The first setup may report installed metadata with `setupVerified=false` until
selected access is configured. Device choices represent physical hardware;
repeat `--device-id` for additional choices. Review the exact plan before its
maintained privileged apply. API/Settings calls return `needs_sudo` and the
fixed local command instead of prompting in a browser. Interrupted permission
changes retain the exact recovery plan and block enable until that transaction
is retried or rolled back. Completed enrollment can be replaced directly; a
replacement rollback restores the prior active grant and policy.

Enable records persistent desired-state intent and starts through the existing
native owner. Disable records disabled intent first, quiesces the writer,
revokes delayed launch claims, and drains exact owned backend/update processes.
A stop failure leaves intent disabled and reports unresolved cleanup; it never
marks a still-running physical backend as stopped. Repeat disable to reconcile
that retained cleanup. A temporary `vrooli resource stop openrgb` does not clear
lighting intent and can be resumed by the reconciler. The low-level resource
commands remain available for explicit maintenance; `resource disable openrgb`
also persists its resource opt-out before stop effects.

Managed setup selects and acquires the pinned official1.1 envelope through SDA
and the resource acquisition owner. Existing-source setup accepts declared1.1
compatibility metadata but launch additionally requires an exact qualified
release/payload profile and sealed closure; an arbitrary version-compatible
executable is not a launch proof. Source path/release/SHA are retained and
rechecked. Compatibility selection runs only `openrgb --version`, without
starting the server. Source changes during governed staging cannot redirect
managed acquisition into adopted-host execution or overwrite that source choice.

Explicit managed setup also prepares a content-addressed AppDir under the
operator's canonical Vrooli artifact storage. The preparation owner preserves
the verified AppImage envelope, extracts its pinned contents without running
OpenRGB, verifies the full bundle and serving ELF, and publishes a read-only
stage atomically. This step invokes only the verified archive runtime with
its fixed extraction-only flag. It runs as the resource owner. Other accounts cannot modify
the stage, and launch repeats the digest and permission checks because the
operator can deliberately change their own files. Startup verifies the
existing stage and performs no extraction or acquisition.

The resource does not silently upgrade at startup. The manifest's pinned
artifact is the acquisition baseline. Explicit uninstall stops the managed
service, removes only its Vrooli-owned artifact directory and records the
resource disabled. It leaves user profiles, other OpenRGB installs and
hardware state alone. Uninstall removes resource-owned managed artifacts;
when using the host source it leaves the adopted executable untouched.

The native update/rollback owner uses the declared `exclusive-stop-start`
strategy. Public admission, continuation and recovery require current explicit
opt-in and exact frozen original/target/rollback policy, catalog, artifact and
native-witness identities. It quiesces the bound writer before stopping the
original, consumes a one-time claim per replacement, verifies actual serving
identity/listener, and converges policy and lighting selection to the observed
serving role. Recovery uses retained inputs rather than a changed checkout.
Unknown builds remain unsupported by concrete release-profile checks. Disable
retires admitted update roles without restoring a backend; failed cleanup is
retained for retry. Source/native fixtures qualify these transitions; actual
OpenRGB host update, rollback and reboot remain separately unverified.

The `deployment.profiles.desktop` entry stays unsupported on every platform:
this resource is declared `host-required` and has no signed desktop-bundle
artifact contract. That declaration is separate from the control-plane
resource lifecycle, which currently integrates Linux amd64 only.

## Compatibility and permissions

The acquisition path is pinned to OpenRGB 1.1. Upstream's release page lists
SDK version 6 for 1.1. Host adoption accepts that declared release only; other
versions are rejected with the reported version. This is a release-version
compatibility gate, not proof that arbitrary builds or device sets are
compatible. SDK protocol negotiation and actual device readiness are reported
separately by runtime status.
On 2026-11-10, `vrooli resource source openrgb host` inspected the existing
`/usr/bin/openrgb`, found `0.9+ (1.0rc1)`, rejected it, and left the source
choice unset. No backend was started by this compatibility check.

The published Linux amd64 artifact is a desktop AppImage. It is staged only
after explicit install and is run with `--server --server-host 127.0.0.1`;
OpenRGB's upstream README documents its headless server mode. The resource
sets `XDG_CONFIG_HOME=${RESOURCE_CONFIG_DIR}` before process startup and passes
`--config ${RESOURCE_CONFIG_DIR}/OpenRGB`. OpenRGB creates its settings manager
before parsing `--config`, so the XDG override keeps the constructor's early
default-settings writes away from the host user's config. The resource writes
a versioned, resource-owned detector catalog to `OpenRGB/OpenRGB.json` before
launch. The checked-in catalog disables every detector; only an explicit
machine-local policy can enable a detector after binding it to an exact USB HID
interface or PCI topology and I2C adapter. Missing enrollment prevents headless
launch; the host-neutral catalog remains all-disabled. The ASUS USB HID `0b05:19af` route is
preferred over SMBus in fixtures, but this host's exact interface/topology is
unqualified. GPU identity uses PCI topology and adapter name, never a transient
`/dev/i2c-N` number. A first pre-fix investigation
start wrote constructor defaults to the ambient user config before this
isolation was added; the original bytes were not backed up, so that file was
not restored byte-for-byte. Evidence records both hashes and the deviation.

The follow-up read-only recovery search found no authentic original copy in the
OpenRGB config directory, its profiles/logs, targeted user backup and snapshot
locations, or the resource-owned configs. `Off.orp` is a July 2025 binary device
profile, not a backup of `OpenRGB.json`; the two resource-owned JSON files were
created during this work and are not recovery sources. The config resides on an
ext4 root filesystem; no `snapper` or `timeshift` command or snapshot directory
was found. Do not overwrite this file with a regenerated/default config or
derive an alleged original from a profile. The safe choices are to preserve the
current file, restore only from a separately verified external backup if one is
available, or have the operator review and adjust current preferences in
OpenRGB. Exact restoration remains unverified and unavailable from the searched
sources.

A bounded offline reconstruction candidate was also tested. Pinned OpenRGB 1.1
`ResourceManager.cpp` inserts missing `Server.default_host`, `Server.default_port`,
`Server.all_controllers`, and `Server.legacy_workaround` values before saving. The
previously installed OpenRGB 0.9+ (1.0rc1) binary contains the latter two setting
names but not the first two. Removing only `default_host` and `default_port`
from the current JSON while preserving every other value produced an 84,004-byte
candidate with SHA-256
`e982b63d0ba3d35bd0845c236bde439290ac4f45b0af44f9f6711f2d25fcf742`, which does
not match the saved original length/hash. It was rejected and never written to
the ambient file. No retained transcript or tool output contains the pre-image
JSON bytes or a full key/value dump. Exact recovery remains unavailable.

The current manifest routes the process environment's `XDG_CONFIG_HOME` to
`${RESOURCE_CONFIG_DIR}` and passes `--config ${RESOURCE_CONFIG_DIR}/OpenRGB`;
OpenRGB's early constructor write and later CLI-selected settings path are both
inside that resource-owned root. `TestOpenRGBUsesPinnedIsolatedDetectorAllowlist`
asserts both paths, and the later isolated operated launch left the ambient
file's hash and timestamp unchanged. No OpenRGB process is running now.

Pinned 1.1 source makes the platform distinction explicit. In
`ResourceManager::SetupConfigurationDirectory` (`ResourceManager.cpp:1109-1144`),
Linux/macOS/BSD builds use `XDG_CONFIG_HOME` when present (otherwise
`$HOME/.config`) and append `OpenRGB`; Windows uses `%APPDATA%` and ignores
`XDG_CONFIG_HOME`. The `--config` pre-detection parser eventually calls
`SetConfigurationDirectory` (`cli.cpp:1461-1474`), but `ResourceManager::get()`
constructs the manager while handling that option, before the explicit path is
applied. Therefore the environment override must be present before startup.
The current Linux resource is configured that way; the same source path would
isolate macOS if its runtime were qualified. Windows remains explicitly
unsupported and would require an `APPDATA=${RESOURCE_CONFIG_DIR}` override
before it could be considered isolated. `TestOpenRGBUsesPinnedIsolatedDetectorAllowlist`
asserts Linux support, macOS/Windows unsupported declarations and absence of an
unqualified Windows override. `TestOpenRGBFirstLaunchSeedsOnlyResourceOwnedConfig`
uses a temporary ambient file containing a user preference, seeds the manifest
config through the managed config writer, confirms constructor and CLI paths
resolve to the same resource-owned file, and byte-compares the ambient file
before/after. This is fixture/source-path proof on Linux; it does not claim
native macOS/Windows startup qualification.

OpenRGB fills missing detector entries from each detector's enabled-by-default
setting, so a sparse allowlist would not be safe. The pinned source audit found
1,948 active static detector keys across platform source files; the asset's
original 1,954 entries include six names found only in commented-out calls and
removed by OpenRGB at startup. The earlier Linux run retained 1,931 names
after pruning, consistent with additional platform-specific omissions. The two
QMK/Vial dynamic detectors are registered separately and do not participate in
`Detectors.detectors`. The checked-in asset is generated and checked against
the exact static key set by
`python3 resources/openrgb/scripts/generate-detectors.py --check --source /tmp/openrgb-release-1.1`.
The generator pins the release tag and commit and requires every entry to be
disabled. A machine-local selection may enable only detector names bound to an
exact qualified device policy. The fixture prefers the ASUS USB HID detector
for `0b05:19af`; this host's exact HID interface remains unqualified. No GPU
choice is inferred from `/dev/i2c-N`; it requires PCI topology and exact
adapter identity. `ENE SMBus DRAM` remains disabled because its detector writes
to SMBus while remapping memory modules.

This selection bounds device detectors, but does not mean zero I2C access.
OpenRGB's Linux backend enumerates and opens available `/dev/i2c-*` interfaces;
an earlier operated run logged 15 interfaces, including motherboard, NVIDIA, and AMDGPU
display buses. The ASUS Aura motherboard detector filters candidates to motherboard SMBus
buses whose PCI subsystem vendor is ASUS or unknown (`0`/`0xffff`), then checks
only addresses `0x40`, `0x4e`, and `0x4f`. If a candidate responds,
`TestForENESMBusController` performs bounded reads and `ENERegisterRead` issues
an SMBus word write to command `0x00` with a source-defined register selector
before reading register `0x81`; this is a nonzero write during detection. The
pinned detector source shows no lighting or mode write in this detection
function. The selected GPU detector matches its PCI IDs and checks only fixed
address `0x67`. HID devices are enumerated, but their disabled detectors do not
run. The ENE DRAM detector that remaps memory modules remains disabled. No
`--i2c-tools` or ad hoc address scan is used. These are bounded upstream
transactions, not zero bus access, and source inspection does not establish
electrical safety or prove motherboard detection. The OpenRGB SDK remains
available for detected devices.

Source basis: pinned [`ENESMBusControllerDetect.cpp`](https://gitlab.com/CalcProgrammer1/OpenRGB/-/raw/release_1.1/Controllers/ENESMBusController/ENESMBusControllerDetect.cpp)
defines the address list, `ENERegisterRead`, the motherboard bus/vendor filter,
and the GPU detector registrations.

The resource health check performs only the OpenRGB SDK protocol v6 negotiation
and confirms the server-name response. It does not request controller count or
data, perform discovery, or prove controller compatibility or light output.
OpenRGB 1.1 logs a `recv_select failed receiving magic` error when these short
health-check connections close after negotiation; this is upstream behavior
for a client disconnect, not evidence of a malformed request. Readiness and
liveness each perform a negotiation, so one status evaluation can produce two
such log lines.

OpenRGB's [upstream udev guidance](https://openrgb.org/udev.html) says AppImage
users need generated udev rules to access RGB devices as an unprivileged user.
In the pinned 1.1 source, that generated file grants `uaccess` to every
`/dev/i2c-*` node and `/dev/port`, then adds rules for all registered HID
detectors. That is broader than this resource's bounded detector allowlist, so
the resource does not install those rules or run OpenRGB as root. A fresh host
may therefore report a ready loopback SDK while OpenRGB cannot access a useful
controller. The maintained control-plane permission owner provides typed
physical-device discovery, plan, apply and rollback. It grants persistent
numeric-owner access only to selected routes and preserves unrelated ambient
ACLs. Every accessible route matching an enabled detector must belong to the
selected physical hardware; one GPU selection includes all of that card's
supported I2C adapters. Privileged activation and logged-out boot qualification
remain unverified; the upstream generated rule is not that maintained workflow.
On the inspected host, no permission change was made. Its existing
host, `/dev/i2c-0` through `/dev/i2c-15` show `root:root` mode `0660`, with a
pre-existing named ACL granting `matthalloran8` read/write on all 16 devices.
The managed process runs as UID 1000 with no effective capabilities. This
host's pre-existing ACL grants wider I2C access than the detector map;
detector filtering bounds OpenRGB's supported device-specific functions but
does not change the OS-level ACL.

The OpenRGB SDK is a raw TCP protocol, not an authenticated API. It must remain
loopback-only. Never publish port 6742 through a proxy, LAN bind, container
port mapping, or firewall rule. Remote lighting calls must pass through a
Vrooli-authenticated API and transport.

Headless launch uses the existing native managed-service owner and a durable
one-time pre-exec claim. Explicit machine opt-in, selected static device access,
source-qualified artifact/catalog and a complete sealed stage are required.
The launcher establishes final service confinement, verifies selected-node
access and exact native identity there, durably consumes the claim, then writes
its frozen private settings and executes the verified payload FD. Readiness
rechecks the serving executable, argv, environment, cwd, settings scope,
security context and exact loopback listener. Failed or uncertain claims cannot
be replayed. Generic launch/adoption paths cannot bypass this owner.

The pinned AppImage envelope digest is
`d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d`; its
verified embedded ELF digest is
`48739d378cfcd27e8b8c8cfc6d820364dddf31f0a9819065be1a281a4ee98dab`.
Explicit setup stages the full verified extraction closure and retains the
governed envelope for rollback. Serving uses the direct sealed ELF in the same
native root, rather than inferring ownership from an AppRun descendant.
Unknown release/payload profiles remain unsupported. Device-free operated
fixtures qualify this payload and SDK behavior; they do not qualify real
selected-device access, actual native service confinement, boot, or physical
lighting. Real managed update/rollback remains separately unverified.

### Managed settings and runtime normalization

Managed settings disable saved `Client.clients`, export local controllers only,
disable OpenRGB's six automatic profile events, and enable its supported
`Detectors.hid_safe_mode`. This selects finite detection instead of autonomous
HID-arrival callbacks. Light Desk owns desired
output; OpenRGB cannot apply a competing startup, resume or shutdown profile.
Selected device generations are rechecked; an actual node replacement first
quiesces the writer, then replaces the exact owned backend and rebinds once.
Unchanged unsupported controls do not trigger restart loops. Unrelated
display/tuning preferences are retained. The pinned runtime starts
saved clients even in server mode and applies service startup/shutdown profiles;
those settings therefore cannot expand the local managed operation.

OpenRGB1.1 rewrites its JSON while initializing, before SDK readiness. The
verified device-free run pruned1948 all-disabled detector entries to1931,
retained all enforced network/profile settings, and kept identical settings
through serving and shutdown. Frozen settings include the exact constructor
defaults and canonical object ordering; large integer preferences are preserved.
The consumed witness keeps the original frozen config digest. Live observation
allows omitted disabled detector keys, never added/enabled detectors, saved
clients, dynamic HID registrations or automatic profiles. Admission/stop
normalize the harmless disabled-key pruning and formatting against the retained
frozen settings; unrelated config drift remains an error. Runtime-normalized
bytes do not replace the consumed launch contract or rollback digest.

## Source and release record

- Upstream release: [OpenRGB 1.1](https://openrgb.org/releases.html), published
  2026-10-16; SDK protocol version 6.
- Asset: `OpenRGB_1.1_x86_64_8e16de2.AppImage`, linked from the official
  [release page](https://openrgb.org/releases.html).
- The release-tagged upstream [CLI source](https://gitlab.com/CalcProgrammer1/OpenRGB/-/raw/release_1.1/cli.cpp)
  defines `--server`, `--server-host` (default `127.0.0.1`) and `--server-port`
  (default `6742`).
- SHA-256 observed from the release asset: `d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d`.
- Runtime license declared by upstream source: GPL-2.0-or-later.
- Upstream describes OpenRGB as capable of standalone or client/headless
  server operation and warns that it uses reverse-engineered hardware
  protocols with nonzero device risk.

The digest authenticates bytes fetched from the linked upstream asset; no
separate upstream signature or checksum was verified. The operated copy was
acquired through the governed resource installer and verified against this
digest.
