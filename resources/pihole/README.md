# Pi-hole Resource

Network-level DNS sinkhole that blocks ads, tracking, and malware domains for all devices on your network.

## Overview

Pi-hole acts as a DNS server that filters out unwanted domains at the network level, providing:
- Network-wide ad blocking without client configuration
- Malware and tracking protection
- Reduced bandwidth usage (40-60% fewer requests)
- REST API for programmatic control
- Query logging and statistics

## Quick Start

```bash
# Install Pi-hole
vrooli resource pihole manage install

# Start the service
vrooli resource pihole manage start --wait

# Check status
vrooli resource pihole status

# View blocking statistics
vrooli resource pihole content stats

# Add domain to blacklist
vrooli resource pihole content blacklist add doubleclick.net

# Run tests
vrooli resource pihole test all
```

## Configuration

Pi-hole runs on:
- **DNS Port**: 53 (TCP/UDP) - DNS service (uses 5353 if 53 is occupied)
- **API Port**: 8087 - Web interface and REST API
- **DHCP Port**: 67 (UDP) - Optional DHCP server

Default credentials:
- **Web Password**: Auto-generated (view with `vrooli resource pihole credentials`)

## Features

### DNS Filtering
- Blocks ads, tracking, and malware domains
- Supports regex patterns for advanced filtering
- Custom whitelist/blacklist management
- Local DNS record definitions
- DHCP server with static reservations
- Common pattern library for quick setup

### API Access
- Full REST API for automation
- Query statistics and logs
- Manage blocklists programmatically
- Enable/disable blocking on demand

### Integration
- Works transparently with all network devices
- Enhances browserless and web scraping scenarios
- Provides clean testing environments
- Integrates with home automation systems

## Content Management

```bash
# View blocking statistics
vrooli resource pihole content stats

# Update blocklists
vrooli resource pihole content update

# Add to blacklist
vrooli resource pihole content blacklist add domain.com

# Add to whitelist  
vrooli resource pihole content whitelist add safe-domain.com

# Query the log
vrooli resource pihole content logs --tail 100

# Disable blocking temporarily
vrooli resource pihole content disable --duration 300

# Manage custom DNS records
vrooli resource pihole content dns add myserver.local 192.168.1.100
vrooli resource pihole content dns remove myserver.local
vrooli resource pihole content dns list
```

## DHCP Server Management

```bash
# Enable DHCP server
vrooli resource pihole content dhcp enable 192.168.1.100 192.168.1.200

# Check DHCP status
vrooli resource pihole content dhcp status

# View active leases
vrooli resource pihole content dhcp leases

# Add static reservation
vrooli resource pihole content dhcp reserve aa:bb:cc:dd:ee:ff 192.168.1.50 mydevice

# List reservations
vrooli resource pihole content dhcp reservations

# Disable DHCP
vrooli resource pihole content dhcp disable
```

## Web Interface Management

```bash
# Check web interface status
vrooli resource pihole web status

# Get web interface URL
vrooli resource pihole web url

# Get/reset web password
vrooli resource pihole web password
vrooli resource pihole web password MyNewPassword123

# Open in browser
vrooli resource pihole web open

# Enable/disable web interface
vrooli resource pihole web enable
vrooli resource pihole web disable
```

## Group Management

```bash
# List groups
vrooli resource pihole groups list

# Create a new group
vrooli resource pihole groups create "Kids" "Strict filtering for children's devices"
vrooli resource pihole groups create "Work" "Minimal filtering for work devices"

# Add clients to groups
vrooli resource pihole groups add-client 192.168.1.50 Kids "Tablet"
vrooli resource pihole groups add-client 192.168.1.51 Work "Laptop"

# List clients in a group
vrooli resource pihole groups list-clients Kids

# Add domains to group blocklist
vrooli resource pihole groups add-domain Kids youtube.com
vrooli resource pihole groups add-domain Kids tiktok.com

# Enable/disable groups
vrooli resource pihole groups disable Work
vrooli resource pihole groups enable Work

# Remove client from group
vrooli resource pihole groups remove-client 192.168.1.50

# Delete group
vrooli resource pihole groups delete Kids
```

## Gravity Sync (Multi-Instance)

```bash
# Initialize sync configuration
vrooli resource pihole gravity-sync init

# Add remote Pi-hole instances
vrooli resource pihole gravity-sync add-remote backup 192.168.1.200 22
vrooli resource pihole gravity-sync add-remote secondary 10.0.0.5 22 /path/to/ssh/key

# List configured remotes
vrooli resource pihole gravity-sync list-remotes

# Export gravity database
vrooli resource pihole gravity-sync export /backup/gravity.db

# Import gravity database
vrooli resource pihole gravity-sync import /backup/gravity.db

# Sync with remote (pull/push/bidirectional)
vrooli resource pihole gravity-sync sync backup pull
vrooli resource pihole gravity-sync sync backup push

# Schedule automatic sync
vrooli resource pihole gravity-sync schedule daily backup
vrooli resource pihole gravity-sync schedule hourly all

# Remove remote
vrooli resource pihole gravity-sync remove-remote backup
```

## Regex Filtering

```bash
# Add regex patterns
vrooli resource pihole content regex add '^ad[0-9]*[-.]'
vrooli resource pihole content regex add-white '^(www\.)?mysite\.com$'

# List patterns
vrooli resource pihole content regex list
vrooli resource pihole content regex list black
vrooli resource pihole content regex list white

# Test a pattern
vrooli resource pihole content regex test '^ad[0-9]*[-.]' 'ad123.example.com'

# Add common pattern sets
vrooli resource pihole content regex common basic      # Basic ad blocking
vrooli resource pihole content regex common aggressive # More aggressive
vrooli resource pihole content regex common social     # Social media tracking
vrooli resource pihole content regex common malware    # Malware patterns

# Import/export patterns
vrooli resource pihole content regex export patterns.txt
vrooli resource pihole content regex import patterns.txt
```

## Network Configuration

To use Pi-hole, configure your devices or router to use it as the DNS server:

1. **Single Device**: Set DNS to your Pi-hole server IP
2. **Network-wide**: Configure router's DHCP to distribute Pi-hole as DNS
3. **Docker Networks**: Use `--dns` flag when running containers

## Testing

```bash
# Quick health check
vrooli resource pihole test smoke

# Full integration tests
vrooli resource pihole test integration

# Test DNS resolution
dig @localhost google.com

# Test ad blocking
dig @localhost doubleclick.net
```

## Troubleshooting

### DNS Not Resolving
- Check if Pi-hole is running: `vrooli resource pihole status`
- Verify port 53 is not in use: `sudo lsof -i :53`
- Check upstream DNS settings

### Websites Not Loading
- Domain may be incorrectly blocked
- Add to whitelist: `vrooli resource pihole content whitelist add domain.com`
- Check query log for blocked requests

### High Memory Usage
- Normal with large blocklists (2M+ domains)
- Reduce blocklist size if needed
- Monitor with `vrooli resource pihole status --verbose`

## API Examples

### Using the New API v2
```bash
# Get statistics (using pihole CLI which handles auth)
docker exec vrooli-pihole pihole api stats/summary

# Get query log
docker exec vrooli-pihole pihole api queries

# Get top domains
docker exec vrooli-pihole pihole api stats/top_domains
```

### Direct API Access (Legacy)
```bash
# Note: The API has moved from /admin/api.php to /api in newer versions
# Authentication is now handled internally by the pihole CLI

# Get status
curl -s "http://localhost:8087/api/stats/summary"

# View blocked domains count
curl -s "http://localhost:8087/api/stats/summary" | jq '.gravity.domains_being_blocked'
```

## Security Notes

- API requires authentication token
- Web interface password is auto-generated
- Regular updates via Docker image
- DNSSEC validation supported
- No sensitive data logged

## Resources

- [Official Pi-hole Documentation](https://docs.pi-hole.net/)
- [Pi-hole API Documentation](https://discourse.pi-hole.net/t/pi-hole-api/1863)
- [Blocklist Collection](https://firebog.net/)
- [Pi-hole Community](https://discourse.pi-hole.net/)