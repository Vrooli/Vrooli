# Connecting a device

Device Control is designed so the owner can follow this guide without opening
source code. Run the commands through the installed CLI after starting the
scenario with `vrooli scenario start device-control`.

## Android over USB

Run `device-control device connect --kind android`. The report identifies each
rung, its owner, and the next action. The scenario owns the probe and report;
the owner supplies the phone and trust decision.

1. Ensure the `android-sdk` resource is running so `adb` and platform-tools
   are on PATH.
2. On the phone, enable Developer Options and USB debugging.
3. Connect USB, accept the RSA authorization prompt, and confirm `adb devices`
   reports the phone as `device`, not `unauthorized`.
4. Run `device-control device list --json`. The Android rung changes to
   `available` from the probe; no code change or scenario restart is needed.

## iOS Simulator

The simulator path is owned by a macOS bridge host. Install Xcode and the
requested iOS Simulator runtime, boot a simulator, then run
`device-control strategy verify ios-simctl`. Missing Xcode components are
reported as `unavailable` with the exact install action.

## Physical iPhone

For semantic XCUITest control, enroll in the Apple Developer Program, configure
WebDriverAgent signing, attach the iPhone to a trusted macOS node, and accept
the trust prompt. For the floor-only mirroring path, pair iPhone Mirroring and
grant both Accessibility and Screen Recording permissions on that macOS node.
The conformance report remains honest until those probes succeed; mirroring
evidence is advisory OCR and cannot be promoted to release evidence.
