# DNS Resolution Check (infra-dns)

Verifies that DNS name resolution is working correctly by resolving a well-known domain.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-dns` |
| Category | Infrastructure |
| Interval | 30 seconds |
| Platforms | All |

## What It Monitors

This check attempts to resolve `google.com` using the system's configured DNS resolver. Success confirms:

- DNS client is configured correctly
- DNS server is reachable
- DNS resolution is functioning

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | DNS resolution succeeded |
| **Critical** | DNS resolution failed |

## Why It Matters

DNS is essential for:
- All hostname-based connections
- API calls to external services
- Package managers (apt, npm, pip)
- Container image pulls
- Certificate validation (OCSP, CRL)

Without working DNS, most network-dependent services will fail even if raw IP connectivity works.

## Common Failure Causes

### 1. DNS Server Unreachable
```bash
# Check configured DNS servers
cat /etc/resolv.conf

# Test DNS server directly
nslookup google.com 8.8.8.8
dig @8.8.8.8 google.com
```

### 2. resolv.conf Misconfigured
```bash
# View current config
cat /etc/resolv.conf

# Common fix: ensure valid nameserver
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

### 3. systemd-resolved Issues
```bash
# Check service status
systemctl status systemd-resolved

# Restart if needed
sudo systemctl restart systemd-resolved

# Check resolved config
resolvectl status
```

### 4. Network Connectivity
DNS requires network connectivity. If this check fails alongside `infra-network`, fix network first.

## Troubleshooting Steps

1. **Test manual resolution**
   ```bash
   nslookup google.com
   dig google.com
   host google.com
   ```

2. **Check DNS configuration**
   ```bash
   cat /etc/resolv.conf
   resolvectl status    # If using systemd-resolved
   ```

3. **Test with specific DNS server**
   ```bash
   nslookup google.com 8.8.8.8
   nslookup google.com 1.1.1.1
   ```

4. **Check for DNS cache issues**
   ```bash
   # Flush DNS cache (systemd-resolved)
   sudo systemd-resolve --flush-caches

   # Or restart resolved
   sudo systemctl restart systemd-resolved
   ```

5. **Check firewall rules for DNS**
   ```bash
   # DNS uses UDP port 53
   sudo iptables -L -n | grep 53
   ```

## Configuration

The check resolves `google.com` by default. This domain was chosen for its extremely high availability.

## Related Checks

- **infra-network**: Network must work before DNS can work
- **infra-cloudflared**: Uses DNS for tunnel coordination
- **resource-***: May use DNS for external connections

## Auto-Heal Actions

When this check fails, autoheal may attempt:
1. Restart systemd-resolved service
2. Reset resolv.conf to known-good values
3. Flush DNS cache

---

*Back to [Check Catalog](../check-catalog.md)*
