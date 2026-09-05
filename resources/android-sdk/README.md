# android-sdk

This resource owns the Android host toolchain consumed by the `android-adb`
device-control strategy. Install it through the governed resource lifecycle;
do not run a package manager or copy an SDK into the repository.

`vrooli resource install android-sdk` downloads the official platform-tools and
command-line-tools archives, a pinned JDK 17 distribution, and Gradle 8.10.2;
accepts the Android SDK licenses non-interactively; and installs the API 36
platform, emulator, and host-architecture Google APIs system image. Archives
are staged before activation, and SHA-256 pins are applied to the default Linux
amd64 JDK and Gradle archives. Other hosts may override the official archive
URLs and pins with `ANDROID_JDK_URL`, `ANDROID_JDK_SHA256`,
`ANDROID_GRADLE_URL`, and `ANDROID_GRADLE_SHA256`. A download, checksum,
extraction, license, package, or validation failure is non-zero and names the
failed step.

The resource exposes `adb`, `emulator`, `sdkmanager`, `avdmanager`, `java`,
`javac`, and a Gradle wrapper through `~/.vrooli/bin` when the installation is
healthy. The wrapper selects the governed JDK automatically. The default API level is 36
(`ANDROID_API_LEVEL` may select a newer level but cannot select an older one).
AVD create/start/stop/delete are explicit lifecycle actions. `avd-start` checks
for a present and writable `/dev/kvm`, launches without snapshots, and returns
only after `sys.boot_completed=1`; an unaccelerated emulator is unavailable,
not a degraded evidence run.

For hermetic unit tests, set `ANDROID_SDK_SKIP_COMPONENTS=1` and provide a
platform-tools test archive through `ANDROID_PLATFORM_TOOLS_URL`. This bypass
also skips the large JDK/Gradle archives and is intentionally not used by the
normal resource lifecycle. The standalone `toolchain-install` lifecycle verb
installs only the pinned build toolchain when the SDK is already present.
