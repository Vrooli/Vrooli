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
1. Check /etc/resolv.conf
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
3. Verify symlink: `ls -la /etc/resolv.conf`

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
**Platforms:** Linux (xrdp), Windows (TermService)

Monitors remote desktop service availability.

### Why It Matters
RDP access is used for:
- Remote administration
- GUI-based development
- Desktop automation

### Status Meanings
- **OK**: RDP service running
- **Warning**: Not installed
- **Critical**: Service failed

### Troubleshooting (Linux)
1. Check xrdp: `sudo systemctl status xrdp`
2. Check port 3389: `ss -tlnp | grep 3389`
3. Restart: `sudo systemctl restart xrdp`
