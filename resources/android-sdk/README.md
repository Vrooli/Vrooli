# android-sdk

This resource owns the Android host toolchain consumed by the `android-adb`
device-control strategy. Install it through the governed resource lifecycle;
do not run a package manager or copy an SDK into the repository.

The resource exposes `adb`, `emulator`, `sdkmanager`, and `avdmanager` when the
operator's Android SDK installation is healthy. AVD create/start/stop/delete
are explicit lifecycle actions so emulator state remains attributable and
recoverable.
