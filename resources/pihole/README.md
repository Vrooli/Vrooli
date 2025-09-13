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
- **DNS Port**: 53 (TCP/UDP) - DNS service
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

```bash
# Get status
curl -s "http://localhost:8087/admin/api.php?status&auth=${PIHOLE_API_TOKEN}"

# Get statistics
curl -s "http://localhost:8087/admin/api.php?summary&auth=${PIHOLE_API_TOKEN}"

# Disable for 5 minutes
curl -s "http://localhost:8087/admin/api.php?disable=300&auth=${PIHOLE_API_TOKEN}"
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