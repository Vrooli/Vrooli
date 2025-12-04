# Internet Connection Check (infra-network)

Verifies basic TCP connectivity to external networks by attempting to connect to Google's public DNS server.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-network` |
| Category | Infrastructure |
| Interval | 30 seconds |
| Platforms | All |

## What It Monitors

This check attempts a TCP connection to `8.8.8.8:53` (Google DNS) with a 5-second timeout. A successful connection confirms:

- The network interface is up
- Outbound TCP connections are possible
- At least one route to the internet exists

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | TCP connection succeeded - network is functional |
| **Critical** | TCP connection failed - no internet connectivity |

## Why It Matters

Internet connectivity is required for:
- External API calls (AI providers, payment gateways, etc.)
- Package updates and dependency downloads
- Cloudflare Tunnel operation
- Container image pulls
- Database backups to cloud storage

A network failure affects almost every scenario that depends on external services.

## Common Failure Causes

### 1. Network Interface Down
```bash
# Check network interfaces
ip addr show

# Bring interface up (example)
sudo ip link set eth0 up
```

### 2. DNS/Firewall Issues
Even though this check uses an IP address directly, firewall rules may block outbound connections:
```bash
# Check if port 53 is blocked
sudo iptables -L -n | grep 53

# Temporary test - try another port
nc -zv 8.8.8.8 443
```

### 3. Router/Gateway Problems
```bash
# Check default route
ip route show default

# Test gateway reachability
ping -c 3 $(ip route | grep default | awk '{print $3}')
```

### 4. ISP Outage
If local network is fine but internet is unreachable:
- Check ISP status page
- Test with mobile hotspot to isolate the issue

## Troubleshooting Steps

1. **Verify local network**
   ```bash
   ip addr show
   ip route show
   ```

2. **Test connectivity manually**
   ```bash
   # Same test the check performs
   nc -zv 8.8.8.8 53

   # Alternative targets
   nc -zv 1.1.1.1 53    # Cloudflare DNS
   nc -zv google.com 443
   ```

3. **Check for firewall blocks**
   ```bash
   sudo iptables -L -n
   sudo ufw status
   ```

4. **Review system logs**
   ```bash
   journalctl -u NetworkManager --since "10 minutes ago"
   dmesg | tail -20
   ```

## Configuration

The target address (`8.8.8.8:53`) is set at check construction time. This is typically configured in the autoheal bootstrap code.

## Related Checks

- **infra-dns**: If network is OK but DNS fails, you have connectivity but name resolution issues
- **infra-cloudflared**: Depends on network connectivity
- **resource-***: All resources that make external calls depend on network

## Auto-Heal Actions

When this check fails, autoheal may attempt:
1. Restart networking service
2. Renew DHCP lease
3. Alert administrators (network issues often require manual intervention)

---

*Back to [Check Catalog](../check-catalog.md)*
