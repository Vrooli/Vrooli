# OpenRGB operations

## Default behavior

Merely declaring this resource does not enable it. The root project manifest
does not depend on OpenRGB, and no default setup step installs it or inspects
hardware. Do not add OpenRGB to project `resources`, scenario dependencies,
or required host tools. `resource status openrgb` and `resource start openrgb`
are explicit operations; they may observe the SDK or access controller
hardware.

## Linux amd64 lifecycle

Use only the control-plane resource owner for lifecycle and artifact changes:

```sh
vrooli resource enable openrgb
vrooli resource source openrgb managed
vrooli resource install openrgb
vrooli resource start openrgb
vrooli resource status openrgb --json
vrooli resource stop openrgb
vrooli resource disable openrgb
vrooli resource uninstall openrgb
```

Enable records operator intent. Choose one source before installing. `managed`
downloads the pinned official OpenRGB 1.1 AppImage through the managed-resource
acquisition owner and verifies the SHA-256 in `resource.json`. `host` adopts
only an `openrgb` executable found on the current `PATH` whose `--version`
output contains the declared 1.1 release. The selected absolute path, release,
and SHA-256 are persisted in operator state. Compatibility selection executes
only `openrgb --version`; it does not start the backend or enumerate hardware.
Install and start recheck the same file identity. A changed file must be
selected again. Start launches the managed backend with the SDK
server bound to loopback. Stop ends the managed process. Disable prevents
future `start-all` selection and stops a managed service before persisting the
disabled choice. If stop fails, disable fails and leaves the resource enabled
so the operator can recover it. Uninstall is a separate explicit operation
and does not remove an independently installed system package.

The resource does not silently upgrade at startup. The manifest's pinned
artifact is the acquisition baseline. Explicit uninstall stops the managed
service, removes only its Vrooli-owned artifact directory and records the
resource disabled. It leaves user profiles, other OpenRGB installs and
hardware state alone. Uninstall removes resource-owned managed artifacts;
when using the host source it leaves the adopted executable untouched.

The generic managed-service update owner currently supports stateless rolling
updates; that strategy is unsafe for OpenRGB because two processes may access
the same controllers at once. Deliberate version update/rollback remains
unsupported until a hardware-exclusive stop, replace, start, health-check and
rollback transaction is available. Do not edit the version pin as a substitute
for that recovery path.

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
a versioned, resource-owned detector asset to `OpenRGB/OpenRGB.json` before
launch. The readable asset disables every listed detector except the ASUS Aura
motherboard and ASUS TUF RTX 4070 Ti SUPER GPU detectors. Future launches with
this XDG override use the resource-owned config. A first pre-fix investigation
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
The generator pins the release tag and commit and requires exactly the two
evidence-backed ASUS entries enabled. The asset explicitly keeps `ENE SMBus
DRAM` disabled; its detector writes to SMBus while remapping memory modules.

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

Health checks only establish that the local SDK TCP endpoint accepts
connections. They do not prove hardware discovery, compatibility, controller
readiness, or light output.

OpenRGB's upstream udev guidance states that AppImage users need generated
udev rules to access RGB devices as an unprivileged user. This resource does
not install system-wide rules or grant broad device access automatically.
Where neither a narrow workflow nor a pre-existing host permission grants
access, a healthy SDK endpoint may still have no usable hardware access. No logged-out boot or
headless device-permission qualification has been performed. On the inspected
host, `/dev/i2c-0` through `/dev/i2c-15` show `root:root` mode `0660`, with a
pre-existing named ACL granting `matthalloran8` read/write on all 16 devices.
The managed process runs as UID 1000 with no effective capabilities. No ACL or
permission change was made by this resource. This host ACL grants wider I2C
access than the detector map; detector filtering bounds OpenRGB's supported
device-specific functions but does not change the OS-level ACL.

The OpenRGB SDK is a raw TCP protocol, not an authenticated API. It must remain
loopback-only. Never publish port 6742 through a proxy, LAN bind, container
port mapping, or firewall rule. Remote lighting calls must pass through a
Vrooli-authenticated API and transport.

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
