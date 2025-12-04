# Certificate Expiration Check (infra-certificate)

Monitors SSL/TLS certificate expiration dates to prevent service disruptions from expired certificates.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-certificate` |
| Category | Infrastructure |
| Interval | 1 hour |
| Platforms | All |

## What It Monitors

This check scans for SSL/TLS certificates (primarily cloudflared tunnel certificates) and monitors their expiration dates:

- Cloudflared tunnel certificates (`~/.cloudflared/cert.pem`)
- System-wide certificates in standard locations
- Custom certificate paths (configurable)

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | All certificates are valid with > 7 days until expiry |
| **Warning** | At least one certificate expires within 7 days |
| **Critical** | At least one certificate expires within 3 days or has expired |

## Why It Matters

Certificate expiration causes:
- Cloudflare tunnel disconnection
- HTTPS handshake failures
- API authentication errors
- Service-to-service communication breakdowns
- User-facing SSL warnings

Unlike many other issues, certificate expiration is **100% predictable** and should never catch you off guard.

## Certificate Locations Checked

### Linux
- `~/.cloudflared/cert.pem`
- `/etc/cloudflared/cert.pem`
- `/etc/ssl/certs/cloudflared.crt`

### macOS
- `~/.cloudflared/cert.pem`
- `/usr/local/etc/cloudflared/cert.pem`

### Windows
- `%USERPROFILE%\.cloudflared\cert.pem`

## Troubleshooting Steps

### 1. Check Certificate Details
```bash
# View certificate expiration
openssl x509 -in ~/.cloudflared/cert.pem -noout -dates

# Full certificate info
openssl x509 -in ~/.cloudflared/cert.pem -noout -text
```

### 2. Renew Cloudflared Certificate
```bash
# Re-authenticate with Cloudflare
cloudflared login

# This opens a browser for authentication and downloads a new cert
```

### 3. Verify After Renewal
```bash
# Check new certificate expiry
openssl x509 -in ~/.cloudflared/cert.pem -noout -enddate
```

## Configuration

Default thresholds can be customized:
- **Warning threshold**: 7 days (configurable)
- **Critical threshold**: 3 days (configurable)

## Related Checks

- **infra-cloudflared**: Monitors cloudflared service health
- **infra-network**: Network connectivity affects certificate validation (OCSP)

## Proactive Monitoring

Certificate expiration is one of the most easily preventable outages. Set up:
1. This autoheal check running hourly
2. External monitoring if possible
3. Calendar reminders for manual review

---

*Back to [Check Catalog](../check-catalog.md)*
