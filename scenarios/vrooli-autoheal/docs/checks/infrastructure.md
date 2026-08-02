# Infrastructure Health Checks

Infrastructure checks verify the foundational services and connectivity required for Vrooli to function.

---

## infra-network: Internet Connection

**Interval:** 30 seconds
**Platforms:** All

Tests TCP connectivity to Google DNS (8.8.8.8:53) to verify outbound internet access.

### Why It Matters
Network connectivity is required for:
- External API calls
- Package updates and downloads
- Cloudflare tunnel connectivity
- Container image pulls

### Status Meanings
- **OK**: TCP connection successful
- **Critical**: Cannot establish connection

### Troubleshooting
1. Check physical/WiFi connection
2. Verify firewall rules allow outbound port 53
3. Test with `ping 8.8.8.8`
4. Check router/gateway status

---

## infra-dns: DNS Resolution

**Interval:** 30 seconds
**Platforms:** All

Verifies domain name resolution by looking up google.com.

### Why It Matters
DNS resolution is required for:
- Resolving hostnames in API calls
- Service discovery
- Container networking

### Status Meanings
- **OK**: Domain resolved successfully
- **Critical**: Resolution failed

### Troubleshooting
1. Check resolver configuration (the `resolv.conf` file)
2. Test with `getent hosts google.com`
3. Verify systemd-resolved is running
4. Check if using local DNS server

---

## infra-ntp: Time Synchronization

**Interval:** 300 seconds (5 minutes)
**Platforms:** Linux

Verifies system clock is synchronized via NTP using timedatectl.

### Why It Matters
Accurate time is critical for:
- TLS certificate validation
- Log correlation across services
- Distributed system consensus
- Session/token expiration

### Status Meanings
- **OK**: NTP synchronized
- **Warning**: NTP disabled or not yet synchronized

### Recovery Actions
- **Enable NTP**: Run `sudo timedatectl set-ntp true`
- **Force Sync**: Restart systemd-timesyncd

### Troubleshooting
1. Check status with `timedatectl status`
2. View NTP servers: `timedatectl show-timesync`
3. Check network allows NTP (UDP 123)

---

## infra-resolved: DNS Resolver Service

**Interval:** 60 seconds
**Platforms:** Linux (systemd)

Monitors the systemd-resolved service which handles DNS resolution on modern Linux systems.

### Why It Matters
systemd-resolved provides:
- Local DNS caching
- DNSSEC validation
- DNS-over-TLS support
- Split DNS for VPNs

### Status Meanings
- **OK**: Service running
- **Warning**: Starting or unknown state
- **Critical**: Service stopped or failed

### Recovery Actions
- **Start Service**: `sudo systemctl start systemd-resolved`
- **Restart Service**: Full restart with cache clear
- **Flush Cache**: Clear DNS cache only
- **View Logs**: Check journalctl for errors

### Troubleshooting
1. Check status: `systemctl status systemd-resolved`
2. View logs: `journalctl -u systemd-resolved`
3. Verify resolver symlink: `ls -la <resolver-config-path>`

---

## infra-docker: Docker Daemon

**Interval:** 60 seconds
**Platforms:** Linux, macOS

Verifies Docker daemon is responsive via `docker info`.

### Why It Matters
Docker is required for:
- Running containerized resources
- Scenario development
- Browser automation

### Status Meanings
- **OK**: Docker responding
- **Critical**: Docker unresponsive or not installed

### Troubleshooting
1. Check service: `sudo systemctl status docker`
2. Test manually: `docker ps`
3. Check socket permissions
4. Restart: `sudo systemctl restart docker`

---

## infra-cloudflared: Cloudflare Tunnel

**Interval:** 60 seconds
**Platforms:** All (where installed)

Monitors the cloudflared service and checks for high error rates in logs.

### Why It Matters
Cloudflare Tunnel provides:
- External access to hosted scenarios
- Secure ingress without port forwarding
- DDoS protection

### Status Meanings
- **OK**: Service running, low error rate
- **Warning**: Not installed, or high error rate (>10 errors in 5 min)
- **Critical**: Service not running

### Recovery Actions
- **Start Service**: `sudo systemctl start cloudflared`
- **Restart Service**: Full restart (clears errors)
- **View Logs**: Recent service logs

### Troubleshooting
1. Check tunnel status: `cloudflared tunnel info`
2. View logs: `journalctl -u cloudflared -n 100`
3. Verify certificate: `ls ~/.cloudflared/`
4. Test tunnel connectivity

---

## infra-rdp: Remote Desktop Access

**Interval:** 60 seconds
**Platforms:** Linux (GNOME Remote Desktop, xrdp), Windows (TermService)

Monitors whether a remote client can connect **and authenticate** — remote
desktop serviceability, not just daemon liveness.

### Why It Matters
RDP access is used for:
- Remote administration
- GUI-based development
- Desktop automation

A daemon that is configured, running, and listening still denies every client
when its credentials are empty or unreadable. This check reports that state
rather than certifying the host healthy because a process exists.

### What It Checks
- Daemon liveness
- Credential state: `present`, `empty`, or `unreadable` (absence of credential
  output is never read as presence of credentials)
- Client denials in the daemon journal over a 15-minute window
- Host posture: autologin user, login keyring collection, user-session daemon,
  graphical session availability
- Credential model (`system` or `user-session`), which decides whether autoheal
  may repair the fault automatically

### Status Meanings
- **OK**: no RDP service installed, or running with credentials present and no denials
- **Warning**: configured but not running, or credential state `unreadable`
- **Critical**: credentials `empty`, clients actively denied, or no graphical session

### Boundary
`infra-rdp` owns the RDP service layer. `infra-display` owns the
graphical-session layer and makes no statement about RDP.

### Troubleshooting
1. Full detail: `vrooli-autoheal check get infra-rdp --json`
2. GNOME credentials: run `grdctl status` with `XDG_RUNTIME_DIR` and
   `DBUS_SESSION_BUS_ADDRESS` set for the session user
3. Denials: `journalctl --user-unit gnome-remote-desktop --since "15 minutes ago"`
4. xrdp hosts: `sudo systemctl status xrdp` and `ss -tlnp | grep 3389`
