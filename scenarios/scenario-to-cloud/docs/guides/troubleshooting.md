# Troubleshooting Guide

Common issues and how to resolve them.

## Preflight Failures

### SSH Connection Failed

**Symptoms**: Preflight shows "SSH connectivity" as failed.

**Causes**:
- Incorrect hostname or IP
- SSH key not authorized
- Firewall blocking port 22
- Wrong SSH user

**Solutions**:
1. Verify you can connect manually: `ssh user@host`
2. Check SSH key is in `~/.ssh/authorized_keys` on VPS
3. Verify port 22 is open: `nc -zv host 22`
4. Try with verbose output: `ssh -v user@host`

### Insufficient Disk Space

**Symptoms**: Preflight shows disk space warning or failure.

**Solutions**:
1. Check current usage: `df -h`
2. Clear package cache: `apt clean`
3. Remove old logs: `journalctl --vacuum-time=7d`
4. Remove unused Docker images: `docker system prune -a`

### Domain Not Resolving

**Symptoms**: Preflight shows domain reachability failure.

**Solutions**:
1. Verify DNS record exists: `dig +short yourdomain.com`
2. Wait for DNS propagation (up to 48 hours)
3. Ensure domain points to VPS IP, not a CDN
4. Check for typos in domain name

## Deployment Failures

### Bundle Build Failed

**Symptoms**: Error during "Build Bundle" step.

**Common Causes**:
- Scenario doesn't exist
- Missing dependencies
- File permission issues

**Solutions**:
1. Verify scenario exists: `ls scenarios/your-scenario`
2. Check scenario has `service.json`
3. Ensure you have read access to all files

### Setup Failed

**Symptoms**: Error during VPS setup phase.

**Solutions**:
1. Check VPS has internet access
2. Verify sudo/root permissions
3. Check for conflicting services
4. Review setup logs in Deployment Details

### Deploy Failed

**Symptoms**: Deployment starts but fails to complete.

**Solutions**:
1. Check resource dependencies are met
2. Verify ports aren't in use: `ss -tlnp`
3. Check for sufficient memory
4. Review scenario logs via Inspect

## HTTPS Issues

### Certificate Not Issued

**Symptoms**: Site accessible via HTTP but HTTPS fails.

**Causes**:
- DNS not propagated
- Port 80 blocked
- Domain verification failed

**Solutions**:
1. Verify ports 80/443 are open
2. Check Caddy logs: `journalctl -u caddy`
3. Ensure domain resolves to VPS IP

### Mixed Content Warnings

**Symptoms**: HTTPS works but browser shows warnings.

**Solutions**:
1. Update scenario to use relative URLs
2. Ensure all resources use HTTPS
3. Check for hardcoded HTTP URLs in code

## Runtime Issues

### Scenario Not Responding

**Symptoms**: Deployed but site shows error.

**Solutions**:
1. Use **Inspect** to check scenario status
2. Check if process is running: `vrooli scenario status`
3. Review logs for errors
4. Verify port mappings are correct

### High Resource Usage

**Symptoms**: Slow performance or OOM errors.

**Solutions**:
1. Monitor with: `htop` or `top`
2. Check memory: `free -h`
3. Consider upgrading VPS
4. Optimize scenario resource usage

## Log Locations

| Component | Location |
|-----------|----------|
| Vrooli | `~/.vrooli/logs/` |
| Scenario | `~/Vrooli/scenarios/{name}/logs/` |
| Caddy | `journalctl -u caddy` |
| System | `journalctl -xe` |

## Getting Help

If you're still stuck:

1. Check the logs for specific error messages
2. Review the API response in browser dev tools
3. Search existing issues in the repository
4. Open a new issue with:
   - Manifest (redact sensitive info)
   - Error message
   - VPS details (OS, specs)
   - Steps to reproduce
