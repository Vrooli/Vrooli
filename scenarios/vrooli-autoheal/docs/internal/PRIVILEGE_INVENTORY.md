# Autoheal privilege inventory

This inventory is the phase-1 migration ledger. It records the 42 non-test
scenario elevation sites found before the typed elevation migration on
2026-08-13. Each site has exactly one disposition; the later implementation
may remove the source call, but the original site remains represented here so
the migration is auditable.

Disposition meanings:

- `grant`: fixed service recovery argv covered by the setup-managed
  `autoheal_recovery_privileges` sudoers grant.
- `delete`: an automatic host mutation that is not defensible as recovery.
- `operator-action`: useful only as an explicit operator recommendation.
- `setup-only`: installation/lifecycle ownership that belongs to setup or the
  platform service seam, not a long-lived scenario check.

| # | Original source site | Original argv/action | Disposition | Reason |
|---:|---|---|---|---|
| 1 | `checks/infra/docker.go` | `systemctl start docker` | grant | Docker start is a fixed service recovery operation. |
| 2 | `checks/infra/docker.go` | `systemctl restart docker` | grant | Docker restart is a fixed service recovery operation. |
| 3 | `checks/infra/dns.go` | `systemctl restart systemd-resolved` | grant | Resolver recovery is a fixed service operation. |
| 4 | `checks/infra/dns.go` | `resolvectl flush-caches` | operator-action | Cache flushing is host-wide and should require an explicit operator choice. |
| 5 | `checks/infra/rdp.go` | `systemctl start gnome-remote-desktop` | grant | The supported desktop service has a fixed unit name. |
| 6 | `checks/infra/rdp.go` | `systemctl restart gnome-remote-desktop` | grant | The supported desktop service has a fixed unit name. |
| 7 | `checks/infra/rdp.go` | `systemctl start xrdp` | grant | The supported xrdp service has a fixed unit name. |
| 8 | `checks/infra/rdp.go` | `systemctl restart xrdp` | grant | The supported xrdp service has a fixed unit name. |
| 9 | `checks/infra/certificate.go` | `systemctl restart cloudflared` | grant | Cloudflared restart is a fixed service recovery operation. |
| 10 | `checks/infra/resolved.go` | `systemctl start systemd-resolved` | grant | Resolver start is a fixed service recovery operation. |
| 11 | `checks/infra/network.go` | `systemctl restart NetworkManager` | grant | NetworkManager restart is a fixed service recovery operation. |
| 12 | `checks/infra/network.go` | `systemctl restart systemd-networkd` | grant | systemd-networkd restart is a fixed service recovery operation. |
| 13 | `checks/infra/cloudflared.go` | `systemctl start cloudflared` | grant | Cloudflared start is a fixed service recovery operation. |
| 14 | `checks/infra/cloudflared.go` | `systemctl restart cloudflared` | grant | Cloudflared restart is a fixed service recovery operation. |
| 15 | `checks/infra/display.go` | `systemctl restart <resolved display-manager unit>` | grant | The unit is selected from the host's declared display manager, then mapped to the fixed grant set. |
| 16 | `checks/infra/ntp.go` | `timedatectl set-ntp true` | operator-action | Changing clock synchronization is an explicit host policy choice. |
| 17 | `checks/system/memory.go` | `sh -c "echo 3 > /proc/sys/vm/drop_caches"` | delete | Dropping the host page cache is destructive, non-diagnostic, and not a safe automatic repair. |
| 18 | `checks/system/disk.go` | `apt-get clean` | delete | Mutating the package cache is outside autoheal's recovery responsibility. |
| 19 | `checks/system/gpu.go` | `nvidia-smi --gpu-reset` | operator-action | Resetting a GPU can interrupt unrelated workloads and requires operator consent. |
| 20 | `watchdog/installer.go` | `loginctl enable-linger <user>` | setup-only | User lingering is installation state and is now reported as setup guidance. |
| 21 | `watchdog/installer.go` | `tee etc/systemd/system/vrooli-autoheal.service` | setup-only | Service-file installation belongs to the platform service seam. |
| 22 | `watchdog/installer.go` | `systemctl daemon-reload` (install) | setup-only | Reloading the service manager is lifecycle work, not a health-check heal. |
| 23 | `watchdog/installer.go` | `systemctl enable vrooli-autoheal` | setup-only | Enabling the watchdog is installation state. |
| 24 | `watchdog/installer.go` | `systemctl restart vrooli-autoheal` | setup-only | Restarting the installed watchdog belongs to the platform seam. |
| 25 | `watchdog/installer.go` | `tee Library/LaunchDaemons/com.vrooli.autoheal.plist` | setup-only | launchd installation belongs to the platform service seam. |
| 26 | `watchdog/installer.go` | `launchctl bootstrap` | setup-only | Launchd loading is owned by the platform-go bootstrap backend. |
| 27 | `watchdog/installer.go` | `systemctl stop vrooli-autoheal` | setup-only | Watchdog uninstall belongs to the platform service seam. |
| 28 | `watchdog/installer.go` | `systemctl disable vrooli-autoheal` | setup-only | Watchdog uninstall belongs to the platform service seam. |
| 29 | `watchdog/installer.go` | `rm -f etc/systemd/system/vrooli-autoheal.service` | setup-only | Removing installed service state belongs to setup/lifecycle ownership. |
| 30 | `watchdog/installer.go` | `systemctl daemon-reload` (uninstall) | setup-only | Service-manager reload is lifecycle work. |
| 31 | `watchdog/installer.go` | `launchctl unload` | setup-only | Legacy launchd unloading is replaced by platform-owned bootout. |
| 32 | `watchdog/installer.go` | `rm -f Library/LaunchDaemons/com.vrooli.autoheal.plist` | setup-only | Removing launchd state belongs to setup/lifecycle ownership. |
| 33 | `watchdog/installer.go` | `schtasks /Create /XML` | setup-only | Windows watchdog installation belongs to the native task/service seam. |
| 34 | `watchdog/installer.go` | `schtasks /Delete` | setup-only | Windows watchdog removal belongs to the native task/service seam. |
| 35 | `healing/strategies/systemd.go` | `systemctl restart <service>` | grant | Runtime recovery is expressed through the typed closed action set; callers cannot supply arbitrary privileged argv. |
| 36 | `healing/strategies/systemd.go` | `systemctl start <service>` | grant | Runtime recovery is expressed through the typed closed action set. |
| 37 | `healing/strategies/systemd.go` | `systemctl stop <service>` | operator-action | Stopping a service is disruptive and is not an automatic recovery grant. |
| 38 | `healing/strategies/systemd.go` | `systemctl status <service>` | setup-only | Status inspection is delegated to the platform service seam. |
| 39 | `healing/strategies/systemd.go` | `systemctl is-active <service>` | setup-only | Service-state inspection is delegated to the platform service seam. |
| 40 | `checks/infra/display.go` | `systemctl restart <display-manager>` (fallback) | grant | The fallback is constrained to the declared display-manager unit set. |
| 41 | `checks/infra/ntp.go` | `systemctl restart systemd-timesyncd` | grant | Timesyncd restart is a fixed service recovery operation. |
| 42 | `checks/infra/network.go` | `systemctl restart systemd-timesyncd` (network-time fallback) | grant | The fallback uses the same fixed timesyncd unit grant. |

The grant content is intentionally narrower than the historical inventory:
it contains only absolute `/usr/bin/systemctl` `start` and `restart` entries
for the fixed units. Dynamic service names, host-wide mutations, watchdog
installation, and operator actions do not become sudoers permissions.
