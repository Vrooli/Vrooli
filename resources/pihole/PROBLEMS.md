# Pi-hole Known Issues and Solutions

## Port 53 Conflict with systemd-resolved

### Problem
On Ubuntu/Debian systems, systemd-resolved uses port 53 by default, preventing Pi-hole from binding to the standard DNS port.

### Solutions

#### Option 1: Use Alternative DNS Port (Recommended for Development)
The resource automatically uses port 5353 as an alternative when port 53 is unavailable:
```bash
# Pi-hole will bind to 5353 if 53 is in use
vrooli resource pihole manage start --wait

# Test DNS resolution on alternative port
dig @localhost -p 5353 google.com
```

#### Option 2: Disable systemd-resolved (Production Only)
```bash
# Disable systemd-resolved
sudo systemctl disable systemd-resolved
sudo systemctl stop systemd-resolved

# Remove symlink and create new resolv.conf
sudo rm /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee /etc/resolv.conf
```

#### Option 3: Configure systemd-resolved as Forwarder
```bash
# Edit systemd-resolved configuration
sudo nano /etc/systemd/resolved.conf
# Set: DNSStubListener=no

# Restart systemd-resolved
sudo systemctl restart systemd-resolved
```

## Docker Network DNS Resolution

### Problem
Containers may not use Pi-hole for DNS resolution by default.

### Solution
Start containers with explicit DNS configuration:
```bash
docker run --dns 172.17.0.1 --dns-search local myapp
```

Or configure Docker daemon defaults:
```json
{
  "dns": ["172.17.0.1"],
  "dns-search": ["local"]
}
```

## High Memory Usage with Large Blocklists

### Problem
Pi-hole uses significant memory (>512MB) with millions of blocked domains.

### Solution
- Use optimized blocklists instead of comprehensive ones
- Monitor memory: `vrooli resource pihole status --verbose`
- Reduce query log retention period
- Consider using group management for selective blocking

## API Authentication Failures

### Problem
API calls fail with authentication errors.

### Solution
Always retrieve current API token:
```bash
# Get API credentials
vrooli resource pihole credentials

# Use in API calls
PIHOLE_TOKEN=$(vrooli resource pihole credentials | grep "API Token" | cut -d: -f2 | xargs)
curl "http://localhost:8087/admin/api.php?status&auth=${PIHOLE_TOKEN}"
```

## DHCP Service Conflicts

### Problem
DHCP service (port 67) may conflict with existing DHCP servers.

### Solution
Pi-hole DHCP is disabled by default. Only enable if you want Pi-hole to manage network DHCP:
```bash
# Enable DHCP only after disabling other DHCP servers
vrooli resource pihole content enable-dhcp --range "192.168.1.100-192.168.1.200"
```

## Slow DNS Resolution

### Problem
Initial DNS queries are slow after startup.

### Solution
- Allow 30-60 seconds for blocklist loading
- Pre-warm DNS cache with common domains
- Use persistent volumes to maintain cache between restarts
- Configure multiple upstream DNS servers for redundancy

## False Positive Blocking

### Problem
Legitimate sites get blocked incorrectly.

### Solution
```bash
# Check if domain is blocked
vrooli resource pihole content logs | grep "domain.com"

# Add to whitelist
vrooli resource pihole content whitelist add domain.com

# Whitelist regex pattern
vrooli resource pihole content whitelist add-regex "^.*\.example\.com$"
```

## Docker Container Auto-Restart Loop

### Problem
Pi-hole container keeps restarting due to configuration errors.

### Solution
```bash
# Check logs for errors
docker logs vrooli-pihole --tail 50

# Stop container
vrooli resource pihole manage stop

# Reset configuration
rm -rf ~/.vrooli/pihole/etc-pihole/pihole-FTL.conf

# Reinstall
vrooli resource pihole manage uninstall
vrooli resource pihole manage install
```