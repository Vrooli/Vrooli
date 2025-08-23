# Agent S2 Transparent Proxy Configuration

## Overview

Agent S2 includes an optional transparent proxy feature that can intercept and monitor HTTP/HTTPS traffic for security analysis. **This feature is now DISABLED by default** to prevent unintended system-wide traffic interception.

## The Issue

When enabled, the transparent proxy:
- Creates iptables NAT rules that redirect ALL system HTTP/HTTPS traffic (ports 80, 443, 8080, 8443) to mitmproxy on port 8085
- Requires the mitmproxy CA certificate to be trusted system-wide
- Can cause SSL/TLS errors for applications that don't trust the mitmproxy certificate
- Affects ALL network requests on the host system, not just Agent S2 traffic

## Default Behavior (Safe Mode)

As of this update, the proxy is **DISABLED by default**. Agent S2 will:
- Run without creating any iptables rules
- Not intercept any system traffic
- Still provide browser automation capabilities
- Still perform AI-based security monitoring at the application level

## Enabling the Proxy (Advanced Users Only)

If you need the transparent proxy for security research or monitoring, you can enable it:

### Method 1: Environment Variable
```bash
export AGENT_S2_ENABLE_PROXY=true
vrooli resource start agent-s2
```

### Method 2: Configuration File
Add to `.vrooli/service.json`:
```json
{
  "services": {
    "agents": {
      "agent-s2": {
        "environment": {
          "AGENT_S2_ENABLE_PROXY": "true"
        }
      }
    }
  }
}
```

### Method 3: Docker Compose
Set in docker-compose.yml:
```yaml
environment:
  - AGENT_S2_ENABLE_PROXY=true
```

## Prerequisites for Proxy Mode

When enabling the proxy, ensure:

1. **Root/sudo access**: Required for iptables manipulation
2. **Trust the CA certificate**: Install mitmproxy CA certificate system-wide
3. **Understand the impact**: ALL HTTP/HTTPS traffic will be intercepted
4. **Container capabilities**: NET_ADMIN, NET_RAW, SYS_ADMIN are required

## Installing the mitmproxy CA Certificate

If you enable the proxy, you need to trust the certificate:

```bash
# Get the certificate from the container
docker exec agent-s2 cat /home/agents2/.mitmproxy/mitmproxy-ca-cert.pem > /tmp/mitmproxy-ca.pem

# Install system-wide (Ubuntu/Debian)
sudo cp /tmp/mitmproxy-ca.pem /usr/local/share/ca-certificates/mitmproxy-ca.crt
sudo update-ca-certificates

# For applications using custom certificate stores (e.g., Firefox, Chrome):
# Import the certificate manually through their settings
```

## Troubleshooting

### SSL/TLS Errors
If you see SSL errors after enabling the proxy:
1. Check if the mitmproxy CA certificate is properly installed
2. Some applications may need manual certificate configuration
3. Consider using proxy bypass for specific applications

### Detecting Active Proxy
Check if iptables rules are active:
```bash
sudo iptables -t nat -L OUTPUT -n --line-numbers | grep 8085
```

### Removing Proxy Rules
If proxy rules were created and you need to remove them:
```bash
# Remove all Agent S2 proxy rules
sudo iptables -t nat -D OUTPUT -p tcp --dport 80 -j REDIRECT --to-port 8085 2>/dev/null
sudo iptables -t nat -D OUTPUT -p tcp --dport 443 -j REDIRECT --to-port 8085 2>/dev/null
sudo iptables -t nat -D OUTPUT -p tcp --dport 8080 -j REDIRECT --to-port 8085 2>/dev/null
sudo iptables -t nat -D OUTPUT -p tcp --dport 8443 -j REDIRECT --to-port 8085 2>/dev/null
```

## Security Considerations

- **Privacy**: When enabled, the proxy can see ALL HTTP/HTTPS traffic including passwords, tokens, and sensitive data
- **Performance**: Proxying all traffic may impact network performance
- **Compatibility**: Some applications may not work correctly with the proxy
- **Scope**: Consider using application-specific proxy settings instead of system-wide interception

## Recommended Usage

For most use cases, keep the proxy **DISABLED** and rely on:
- Agent S2's AI-based security monitoring at the application level
- Browser-specific proxy settings when needed
- Targeted security analysis tools for specific applications

Only enable the transparent proxy when:
- Performing security research
- Analyzing specific malware or suspicious applications
- You fully understand the implications and have proper authorization

## Network Diagnostics Integration

The Vrooli network diagnostics script now includes automatic detection of Agent S2 proxy interference. Run diagnostics to check:
```bash
vrooli lib network diagnostics
```

The script will detect if Agent S2's proxy is causing issues and provide specific remediation steps.