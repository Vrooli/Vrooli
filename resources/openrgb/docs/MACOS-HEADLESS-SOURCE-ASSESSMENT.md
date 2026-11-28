# macOS headless source assessment

The pinned release has a non-GUI SDK-server path that avoids constructing
`QApplication`. This makes a daemon launch a source-supported candidate, but
does not qualify LaunchDaemon, WindowServer independence, Seatbelt, HID/TCC,
reconnect, or actual SDK operation on either Mac architecture.

The official GitHub `release_1.1` tag resolves to commit
`8e16de27858cf2be6af54a33ce003df74cb9fd3e`, matching the release asset commit.
Files were read through the official GitHub connector and retained with their
Git blob identities in
`.vrooli/work/openrgb-headless-source-qualification/manifest.json`.

## Startup and constructor path

- [cli.cpp](https://github.com/CalcProgrammer1/OpenRGB/blob/8e16de27858cf2be6af54a33ce003df74cb9fd3e/cli.cpp#L1414)
  initializes return flags to zero. `--server` selects the server path and is
  not counted among the configuration-only arguments. Explicit GUI options or
  an invocation containing only configuration options select the GUI. The
  maintained invocation includes `--server` and refuses GUI arguments.
- [startup.cpp](https://github.com/CalcProgrammer1/OpenRGB/blob/8e16de27858cf2be6af54a33ce003df74cb9fd3e/startup/startup.cpp#L66)
  constructs `QApplication` only inside `RET_FLAG_START_GUI`. Its other branch
  waits for ResourceManager initialization and checks the SDK server result.
  The examined headless initialization does not construct a `QCoreApplication`
  either. GUI plugin-manager initialization belongs to the GUI path.
- [main](https://github.com/CalcProgrammer1/OpenRGB/blob/8e16de27858cf2be6af54a33ce003df74cb9fd3e/startup/main_FreeBSD_Linux_MacOS.cpp#L81)
  registers headless signal handlers and waits in the server loop. Dock/UI
  transformations in `qt/macutils.mm` are called from the GUI path.

Both measured official Mac bundles contain only
`Contents/PlugIns/platforms/libqcocoa.dylib`; neither contains the offscreen or
minimal QPA plugin. The frozen `QT_QPA_PLATFORM=offscreen` environment therefore
does not prove headlessness or provide a GUI fallback. The supported candidate
depends on keeping the real non-GUI server path. All bundled Qt/framework code
remains included in the exact whole-bundle hash and native image ABI checks;
ambient plugin and dynamic-library search inputs are not admitted.

## Transport initialization before CLI parsing

The Intel main calls `InitMacUSPCIODriver` before CLI parsing under the build
macro `_MACOSX_X86_X64`. Despite the nearby upstream comment saying “Install”,
the [actual helper](https://github.com/CalcProgrammer1/OpenRGB/blob/8e16de27858cf2be6af54a33ce003df74cb9fd3e/dependencies/macUSPCIO/macUSPCIOAccess.h#L16)
only finds `macUSPCIO` and attempts `IOServiceOpen`; it contains no installer or
prompt. Failure returns false. The pinned qmake file includes that integration
for its Intel build-host branch. Read-only inspection of the measured Intel
payload finds the `macUSPCIO` literal; the measured ARM payload does not contain
it. This corroborates the source branch without claiming a native execution.

The actual-operator runtime still requires the separate maintained Darwin
transport policy to deny `macUSPCIOUserClient` opening and `macUSPCIO`
connections before vendor execution. Ordinary UID and a disabled device
detector do not establish this early-startup safety property. This narrow
transport facet must compose with the native child-process policy. Its actual
native enforcement and supported HID access remain qualification work.

## Next native proof

On each target architecture, explicitly prepare the immutable pinned bundle,
then launch the governed actual-operator backend from its native boot owner
without an interactive login. Verify the mapped image and environment, absence
of GUI/WindowServer dependency, denial of the macUSPCIO transport, configured HID
scope, SDK listener ownership, Off/preset behavior, stop/restart and recovery.
No such native launch or device access was performed by this source assessment.
