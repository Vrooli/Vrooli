# android-sdk

This resource owns the Android host toolchain consumed by the `android-adb`
device-control strategy. Install it through the governed resource lifecycle;
do not run a package manager or copy an SDK into the repository.

`vrooli resource install android-sdk` downloads the official platform-tools
archive, verifies its SHA-256 when `ANDROID_PLATFORM_TOOLS_SHA256` is supplied,
extracts it under `~/.vrooli/resources/android-sdk`, and exposes `adb` through
`~/.vrooli/bin`. It validates `adb version` before returning success; a
download, checksum, extraction, or validation failure is non-zero and names
the failed step.

The resource exposes `adb`, `emulator`, `sdkmanager`, and `avdmanager` when the
operator's Android SDK installation is healthy. AVD create/start/stop/delete
are explicit lifecycle actions so emulator state remains attributable and
recoverable.
