# Linux desktop companion

The companion binds an explicitly selected local X11 display and verifies its
logind session through the Vrooli host inventory before observing or applying
input. Its user service is socket activated and accepts local calls at a
mode-0600 Unix socket. Bridge remains the remote authorization boundary.

Build with `GOOS=linux make -C scenarios/device-control companion-build`.
Run `install-linux` from the desktop account with the resulting executable,
the numeric display, the existing private Xauthority path, and either the
exact logind session ID or `active-user`. The latter resolves the current
graphical login for the same UID and still checks that its X server is the
approved display. It supports a replacement logind ID after logout or reboot.
If the display number or authority path changes, re-run installation with the
new values. A locked or inactive session refuses access.

When the Bridge agent runs in presence-only posture, opt in to the exact
desktop protocol with `VROOLI_BRIDGE_DESKTOP_COMPANION=1` in its user service
environment. This leaves general scenario requests, shell jobs and
provisioning restricted. The agent must have a verified control-plane pin.

Use **Web Console → Settings → Devices → swarminator → Details → Desktop**.
The Linux companion offers same-origin HTTPS frames for phone or tunnel
connections and WebRTC for networks with a viable peer route. Device Control
owns the session, input lease, and frame lifecycle in both cases. The
on-screen typing field sends printable ASCII to the focused desktop field and
clears after submit. Linux clipboard transfer is currently unavailable.

Inspect with `systemctl --user status vrooli-desktop-companion.socket
vrooli-desktop-companion.service`. To withdraw access, stop the service and
disable its socket. A fresh connection after restart requires a new lease.
