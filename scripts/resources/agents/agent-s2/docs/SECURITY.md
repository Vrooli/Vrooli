# Agent S2 Security Guide

Agent S2's dual-mode architecture provides different security levels for different use cases. This guide covers security considerations, best practices, and monitoring for both sandbox and host modes.

## Security Architecture Overview

Agent S2 operates with two distinct security models:

- **Sandbox Mode**: High security with full container isolation
- **Host Mode**: Medium security with controlled host access

## Sandbox Mode Security (High Security)

Sandbox mode provides maximum isolation and is recommended for most automation tasks.

### Container Isolation

- **User**: Runs as non-root user (`agents2`) inside container
- **Filesystem**: Isolated container filesystem with read-only host mounts
- **Display**: Virtual X11 display with no host display access
- **Network**: External HTTPS only, no localhost or private network access
- **Resources**: Strict CPU and memory limits enforced

### Restricted Access

- **Applications**: Container applications only (Firefox, LibreOffice, etc.)
- **File System**: Limited to `/home/agents2`, `/tmp`, `/opt/agent-s2`
- **Commands**: Whitelist-based command filtering
- **Network**: No access to host services or internal networks

### Sandbox Mode Security Controls

```bash
# Container runs with limited capabilities
docker run --cap-drop=ALL --cap-add=SYS_CHROOT

# No privileged access
--security-opt no-new-privileges

# Read-only root filesystem
--read-only

# Memory and CPU limits
--memory=1g --cpus=0.5
```

## Host Mode Security (Medium Security)

Host mode provides controlled access to host resources for advanced automation scenarios.

### Controlled Host Access

- **User**: Still runs as `agents2` user with controlled escalation
- **Display**: X11 forwarding with host display access (optional)
- **Network**: Controlled access to localhost and private networks
- **Resources**: Enhanced limits with host resource access

### Security Monitoring

- **AppArmor**: Mandatory security profile (`docker-agent-s2-host`)
- **Audit Logging**: All actions logged to `/var/log/agent-s2/audit/`
- **Threat Detection**: Real-time monitoring for suspicious activities
- **Input Validation**: Enhanced validation for all inputs and commands

### Host Mode Constraints

The following restrictions are enforced by AppArmor and runtime monitoring:

```bash
# Forbidden paths (enforced by AppArmor)
/etc/passwd, /etc/shadow, /root/*, /var/log/auth.log

# Blocked commands
sudo su -, passwd, usermod, chmod 4755, nc -l, bash -i >&

# Network restrictions
No access to: 127.0.0.1:22, sensitive internal services

# Application restrictions
Password managers, system settings still blocked
```

## Security Best Practices

### For Sandbox Mode

```bash
# Keep container updated
./scripts/resources/agents/agent-s2/manage.sh --action update

# Monitor resource usage
docker stats agent-s2

# Review logs regularly
./scripts/resources/agents/agent-s2/manage.sh --action logs

# Verify container integrity
docker inspect agent-s2 | jq '.Config.SecurityOpt'
```

### For Host Mode

```bash
# Verify AppArmor profile is active
sudo apparmor_status | grep docker-agent-s2-host

# Monitor audit logs
sudo tail -f /var/log/agent-s2/audit/$(date +%Y-%m-%d).log

# Review security events
curl http://localhost:4113/modes/security | jq '.recent_events'

# Check for security violations
./scripts/resources/agents/agent-s2/security/check-violations.sh

# Validate security profile
./scripts/resources/agents/agent-s2/security/validate-profile.sh
```

## Security Monitoring

Agent S2 includes comprehensive security monitoring to detect and respond to threats.

### Threat Detection

The system monitors for:

- **Suspicious file access**: `/etc/passwd`, `/root/*`, private keys
- **Privilege escalation**: `sudo`, `setuid`, `chmod 4755` attempts
- **Network anomalies**: Reverse shells, suspicious connections
- **Rapid actions**: Automated attack patterns or brute force attempts
- **Command injection**: Attempts to execute unauthorized commands

### Audit Logging

```bash
# View audit summary
curl http://localhost:4113/modes/security/audit

# Export security events for analysis
curl http://localhost:4113/modes/security/events > security-report.json

# Check threat indicators
curl http://localhost:4113/modes/security | jq '.threat_indicators'

# Review blocked actions
curl http://localhost:4113/modes/security/blocked | jq '.'
```

### Real-time Monitoring

```bash
# Monitor security events in real-time
curl -N http://localhost:4113/modes/security/stream

# Set up alerts for security violations
./scripts/resources/agents/agent-s2/security/setup-alerts.sh

# Configure threat response automation
./scripts/resources/agents/agent-s2/security/setup-response.sh
```

## Component Security

### VNC Security

- **Authentication**: Password-protected VNC access
- **Binding**: Local-only by default (`127.0.0.1:5900`)
- **Encryption**: Optional SSL/TLS for remote access
- **Access Control**: VNC only accessible to authorized users

#### Secure VNC Configuration

```bash
# Enable VNC with SSL
export AGENT_S2_VNC_SSL=true
export AGENT_S2_VNC_PASSWORD="secure_password_here"

# Restrict VNC access to specific IPs
export AGENT_S2_VNC_ALLOWED_IPS="192.168.1.100,10.0.0.50"

# Custom VNC port
export AGENT_S2_VNC_PORT=5901
```

### API Security

- **Authentication**: API key support for production environments
- **Rate Limiting**: Configurable request rate limits
- **Input Validation**: All inputs validated and sanitized
- **CORS**: Configurable cross-origin request policies

#### API Security Configuration

```bash
# Enable API authentication
export AGENT_S2_API_AUTH=true
export AGENT_S2_API_KEY="your-secure-api-key"

# Configure rate limiting
export AGENT_S2_API_RATE_LIMIT=100  # requests per minute
export AGENT_S2_API_BURST_LIMIT=20  # burst allowance

# CORS settings
export AGENT_S2_CORS_ORIGINS="https://trusted-domain.com"
```

## Production Deployment Security

### Recommended Production Configuration

```bash
# Enable all security features
export AGENT_S2_HOST_AUDIT_LOGGING=true
export AGENT_S2_HOST_SECURITY_PROFILE=agent-s2-host
export AGENT_S2_API_RATE_LIMIT=100
export AGENT_S2_VNC_SSL=true
export AGENT_S2_API_AUTH=true

# Install with security hardening
./scripts/resources/agents/agent-s2/manage.sh --action install \
  --mode sandbox \
  --host-mode-enabled yes \
  --audit-logging yes \
  --security-hardening yes \
  --vnc-ssl yes \
  --api-auth yes
```

### Security Hardening Options

```bash
# Network isolation
--network-isolation yes

# Filesystem protection
--filesystem-protection strict

# Process monitoring
--process-monitoring yes

# Memory protection
--memory-protection yes
```

## Security Checklist

### Pre-deployment Security Checklist

- [ ] AppArmor profile installed and active (host mode)
- [ ] Audit logging enabled and monitored
- [ ] VNC password changed from default
- [ ] API rate limiting configured
- [ ] Security events monitoring set up
- [ ] Regular security log reviews scheduled
- [ ] Container images kept updated
- [ ] Network access properly restricted
- [ ] File system permissions verified
- [ ] Backup and recovery procedures tested

### Runtime Security Monitoring

- [ ] Daily audit log reviews
- [ ] Weekly security scan reports
- [ ] Monthly security configuration audits
- [ ] Incident response procedures documented
- [ ] Security update notifications configured

## Incident Response

### Security Event Response

```bash
# Immediate threat response
./scripts/resources/agents/agent-s2/security/emergency-stop.sh

# Isolate compromised container
docker pause agent-s2

# Collect forensic data
./scripts/resources/agents/agent-s2/security/collect-forensics.sh

# Generate incident report
./scripts/resources/agents/agent-s2/security/incident-report.sh
```

### Recovery Procedures

```bash
# Clean shutdown and forensic backup
./scripts/resources/agents/agent-s2/manage.sh --action stop --forensic-backup

# Reinstall with clean state
./scripts/resources/agents/agent-s2/manage.sh --action install --clean-install

# Restore verified data only
./scripts/resources/agents/agent-s2/security/restore-clean-data.sh
```

## Security Updates

### Regular Maintenance

```bash
# Update security profiles
./scripts/resources/agents/agent-s2/security/update-profiles.sh

# Update container base images
./scripts/resources/agents/agent-s2/manage.sh --action update --security-patches

# Verify security configuration
./scripts/resources/agents/agent-s2/security/audit-config.sh
```

## Reporting Security Issues

If you discover security vulnerabilities in Agent S2:

1. **DO NOT** create public GitHub issues for security problems
2. Email security concerns to the maintainers
3. Include detailed reproduction steps
4. Allow time for security patches before disclosure

## Security Resources

- [AppArmor Documentation](https://wiki.ubuntu.com/AppArmor)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Container Security Guidelines](https://security/agent-s2/container-security.md)
- [Audit Log Analysis Guide](https://security/agent-s2/audit-analysis.md)