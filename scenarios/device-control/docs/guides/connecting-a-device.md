# Connecting a device

Device Control is designed so the owner can follow this guide without opening
source code. Run the commands through the installed CLI after starting the
scenario with `vrooli scenario start device-control`.

Once a device is listed, inspect live state with
`device-control device state <device-id> --json`. The report is sourced from
the phone at request time; an unavailable field names the exact probe command.
Screenshot-dependent flow steps refuse a locked or sleeping surface rather
than recording a black frame. Lease-owned orientation and radio changes are
restored when the session is released or killed.

## Android over USB

Run `device-control device connect --kind android`. The report identifies each
rung, its owner, and the next action. The scenario owns the probe and report;
the owner supplies the phone and trust decision.

For a live setup session, use `device-control device connect --kind android
--watch`. The command re-probes without restarting the scenario, records rung
status changes in the `transitions` output field, and returns as soon as every
rung is available or after the bounded 30-second window. Change the window
with `--watch-seconds`; an expired watch still returns the latest named
diagnosis.

1. Ensure the `android-sdk` resource is running so `adb` and platform-tools
   are on PATH. If it is missing, run `vrooli resource install android-sdk`.
   The resource must either make `adb version` succeed or return a non-zero
   diagnostic; a successful no-op is not a valid installation result.
2. On the phone, enable Developer Options and USB debugging.
3. Connect a data-capable USB cable and select **File Transfer** when Android
   asks for the USB mode. If an old ADB server is genuinely stale, run
   `adb kill-server` once before reconnecting; do not run it for every probe.
   Restarting ADB can make Android show the RSA authorization prompt again.
   A charge-only cable is diagnosed as no usable device; it is not reported as
   an unknown phone.
4. Accept the RSA authorization prompt only for the intended phone and confirm
   `adb devices` reports it as `device`, not `unauthorized`. Declining the
   prompt leaves the named `unauthorized` rung active. Normal device-control
   probes and flow runs do not bypass or suppress this Android trust boundary.
5. Run `device-control device connect --kind android` again. The probe is live:
   it distinguishes missing `adb`, no USB device, `unauthorized`, `offline`,
   and insufficient udev permissions. Run `device-control device list --json`
   to see the stable id, serial, model, OS version, host node, transport, and
   per-capability result. No code change or scenario restart is needed.

If the report says **insufficient permissions**, install the supplied udev rule
from [`51-android.rules`](51-android.rules) into `/etc/udev/rules.d/`, reload
udev, add your user to `plugdev`, and replug the phone:

```sh
sudo install -m 0644 51-android.rules /etc/udev/rules.d/51-android.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
sudo usermod -aG plugdev "$USER"
```

Log out and back in after changing group membership. The onboarding probe
distinguishes this host-permission failure from an unauthorized RSA prompt.

The onboarding report separates the physical `usb-bus` rung from the
`usb-debugging` rung. `usb-bus` unavailable means the host cannot see an
Android USB vendor on the bus; check the cable, USB mode, and host permissions.
When the bus is available but `usb-debugging` is unavailable, the phone is
present and the next action names the RSA, offline, or udev issue.

If an old retained identity is no longer yours, remove it deliberately with
`device-control device forget <device-id>`. Normal disconnects never remove a
row, so reconnecting the same serial restores the same stable id.

### Reading the `usb-bus` rung correctly

A locked phone still enumerates. It appears on the USB bus and reports
`unauthorized` to `adb` — present, but not yet trusted. So a locked screen never
causes an absent device, and unlocking the phone never repairs one.

Use this table to turn the two rungs into one physical action:

| `usb-bus` | `usb-debugging` | Meaning | Do this |
| --- | --- | --- | --- |
| unavailable | unavailable | No USB data link at all | Reseat or replace the cable; select **File Transfer** USB mode; try a direct port instead of a hub |
| available | unavailable | Phone is present, not yet trusted | Accept the RSA prompt, or apply the udev rule when the reason names permissions |
| available | available | Ready | Run `device-control device list --json` |

The first row is the one operators most often misread. A charge-only cable, a
damaged cable, or a cable that does not seat its data pins produces a phone that
charges normally and is invisible to both `lsusb` and `adb`. No Android setting
can produce that state, so do not change phone settings to chase it.

Confirm a repaired cable is stable before you rely on it. The onboarding
probe takes a bounded presence sample and reports `flap_count` on the
`usb-bus` rung. A non-zero count means the link was intermittent during the
sample; replace the cable before capturing evidence you intend to keep.

### Transport support today

USB is the onboarding and release-evidence transport. After a trusted USB
onboarding, Android devices may be promoted to wireless ADB with
`device-control device promote --transport wireless`. Promotion keeps the same
device id and verifies the recorded serial after connecting. Wireless is useful
for automation, but pairing is lost after a device reboot; reattach USB and
promote it again. Release-grade conformance defaults to USB. To explicitly run
a flow over the promoted wireless endpoint, use
`device-control flow run --transport wireless`; omitting the flag requests USB.
Promotion may also cause Android to re-ask for trust when ADB is re-established;
that is an OS authorization event, not a hardcoded device address or an
unattended approval.

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
