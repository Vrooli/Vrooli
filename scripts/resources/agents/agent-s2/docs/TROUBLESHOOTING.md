# Agent S2 Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Agent S2. Issues are organized by symptom with step-by-step solutions.

## Quick Diagnostics

### Health Check Commands

```bash
# Check service status (recommended)
./manage.sh --action status

# Test all functionality
./manage.sh --action usage --usage-type all

# Check Docker container
docker ps | grep agent-s2

# View recent logs
./manage.sh --action logs
```

### System Requirements Verification

```bash
# Check Docker version
docker --version

# Check available memory
free -h

# Check available disk space
df -h

# Check port availability (if status shows issues)
netstat -tlnp | grep :4113
netstat -tlnp | grep :5900
```

## Service Issues

### Service Won't Start

**Symptoms**: Container fails to start, API not accessible, no response from health check

**Diagnosis Steps**:
```bash
# Check comprehensive status (recommended)
./manage.sh --action status

# Check logs for startup errors
./manage.sh --action logs

# Check Docker status
docker ps -a | grep agent-s2

# Verify Docker daemon is running (if needed)
systemctl status docker
```

**Solutions**:
```bash
# Method 1: Restart the service
./manage.sh --action restart

# Method 2: Rebuild from scratch
./manage.sh --action uninstall
./manage.sh --action install --force yes

# Method 3: Check for resource constraints
docker system df
docker system prune  # Free up space if needed

# Method 4: Increase resource limits
docker update agent-s2 --memory 4g --cpus 2.0
```

### Service Starts But Becomes Unresponsive

**Symptoms**: Service starts successfully but stops responding to API calls

**Diagnosis Steps**:
```bash
# Check container resource usage
docker stats agent-s2

# Check system resource availability
htop

# Monitor log output in real-time
./manage.sh --action logs --follow
```

**Solutions**:
```bash
# Increase memory limit
docker update agent-s2 --memory 4g

# Increase shared memory for browser operations
docker update agent-s2 --shm-size 2g

# Restart with clean state
./manage.sh --action restart --clean-state
```

## Display and VNC Issues

### VNC Connection Issues

**Symptoms**: Cannot connect to VNC, VNC client shows connection refused, blank screen

**Diagnosis Steps**:
```bash
# Check VNC port accessibility
telnet localhost 5900

# Verify VNC service is running
./manage.sh --action status --verbose

# Check VNC logs
docker exec agent-s2 tail -f /var/log/Xvfb.log
```

**Solutions**:
1. **Port is blocked**:
   ```bash
   # Check firewall rules
   sudo ufw status
   sudo ufw allow 5900
   ```

2. **Wrong VNC password**:
   ```bash
   # Reset VNC password
   ./manage.sh --action install --vnc-password newpassword
   ```

3. **Try alternative VNC clients**:
   - RealVNC Viewer
   - TightVNC
   - TigerVNC
   - Remote Desktop (Windows)

### Display Issues

**Symptoms**: Applications don't display correctly, screen appears corrupted, wrong resolution

**Common Display Problems**:
- Virtual display runs at 1920x1080 by default
- Some applications require specific display settings
- Color depth issues with certain applications

**Solutions**:
```bash
# Change display resolution
export AGENT_S2_DISPLAY_WIDTH=1280
export AGENT_S2_DISPLAY_HEIGHT=720
./manage.sh --action restart

# Increase color depth
export AGENT_S2_DISPLAY_DEPTH=32
./manage.sh --action restart

# Reset display to defaults
./manage.sh --action install --reset-display
```

## API Issues

### API Not Responding

**Symptoms**: HTTP requests to API fail, timeout errors, connection refused

**Diagnosis Steps**:
```bash
# Test functionality (recommended)
./manage.sh --action usage --usage-type capabilities

# Check comprehensive status
./manage.sh --action status

# Verify container networking (if needed)
docker inspect agent-s2 | grep -A 10 NetworkSettings
```

**Solutions**:
1. **Service not running**:
   ```bash
   ./manage.sh --action start
   ```

2. **Port not accessible**:
   ```bash
   # Check Docker port mapping
   docker port agent-s2
   
   # Restart with explicit port mapping
   ./manage.sh --action install --api-port 4113
   ```

3. **Firewall blocking access**:
   ```bash
   sudo ufw allow 4113
   ```

### API Returns Errors

**Symptoms**: API responds but returns error messages, 500 status codes

**Common Error Codes**:
- `SERVICE_UNAVAILABLE`: Display server not ready
- `AI_UNAVAILABLE`: AI features requested but no API key configured
- `INVALID_COORDINATES`: Mouse coordinates out of display bounds
- `SCREENSHOT_FAILED**: Display not accessible

**Solutions**:
```bash
# For SERVICE_UNAVAILABLE
docker exec agent-s2 ps aux | grep Xvfb

# For AI_UNAVAILABLE
export ANTHROPIC_API_KEY="your_key_here"
./manage.sh --action restart

# For coordinate issues
curl http://localhost:4113/capabilities  # Check display bounds

# For screenshot failures
docker exec agent-s2 xset -display :99 q  # Test X display
```

## AI and LLM Issues

### AI Features Not Working

**Symptoms**: AI endpoints return errors, natural language commands fail

**Diagnosis Steps**:
```bash
# Test AI capabilities (recommended)
./manage.sh --action usage --usage-type planning

# Check service status for AI availability
./manage.sh --action status

# Check API key configuration
env | grep -E "(ANTHROPIC|OPENAI|AGENTS2)_API_KEY"
```

**Solutions**:
1. **No API key configured**:
   ```bash
   export ANTHROPIC_API_KEY="your_key_here"
   ./manage.sh --action restart
   ```

2. **Invalid API key**:
   ```bash
   # Test key manually
   curl -H "x-api-key: $ANTHROPIC_API_KEY" \
     https://api.anthropic.com/v1/messages \
     -d '{"model": "claude-3-sonnet-20240229", "max_tokens": 10, "messages": [{"role": "user", "content": "Hi"}]}' \
     -H "Content-Type: application/json"
   ```

3. **Rate limiting**:
   ```bash
   # Test if rate limiting is the issue
   ./manage.sh --action usage --usage-type planning
   
   # Wait and retry, or upgrade API plan
   ```

### AI Tasks Timeout

**Symptoms**: AI operations start but never complete, timeout errors

**Solutions**:
```bash
# Increase task timeout
export AGENT_S2_AI_TIMEOUT=300  # 5 minutes
./manage.sh --action restart

# Use manage.sh for task management (recommended)
# Check logs for stuck tasks
./manage.sh --action logs

# Restart service if needed
./manage.sh --action restart
```

## Mode-Specific Issues

### Host Mode Issues

**Symptoms**: Host mode features don't work, permission denied errors

**Diagnosis Steps**:
```bash
# Check mode status (recommended)
./manage.sh --action mode

# Check comprehensive status
./manage.sh --action status

# Verify AppArmor profile (if needed)
sudo apparmor_status | grep docker-agent-s2-host
```

**Solutions**:
```bash
# Install security prerequisites
sudo ./security/install-security.sh install

# Enable host mode
export AGENT_S2_HOST_MODE_ENABLED=true
./manage.sh --action install --host-mode-enabled yes

# Verify X11 forwarding (if needed)
./lib/modes.sh setup_x11_forwarding
```

### Sandbox Mode Restrictions

**Symptoms**: Operations fail due to sandbox restrictions

**Solutions**:
1. **File access denied**:
   ```bash
   # Check mode status and restrictions
   ./manage.sh --action mode
   
   # Use allowed paths only: /home/agents2, /tmp, /opt/agent-s2
   ```

2. **Network access blocked**:
   ```bash
   # Sandbox mode only allows HTTPS external connections
   # No localhost or private network access
   ```

## Performance Issues

### Slow Response Times

**Symptoms**: API calls take too long, timeouts occur frequently

**Diagnosis Steps**:
```bash
# Check resource usage
docker stats agent-s2

# Monitor system resources
htop

# Check disk I/O
iotop
```

**Solutions**:
```bash
# Increase memory allocation
docker update agent-s2 --memory 4g

# Increase CPU allocation
docker update agent-s2 --cpus 2.0

# Increase shared memory for browsers
docker update agent-s2 --shm-size 2g

# Optimize Docker storage driver
docker info | grep "Storage Driver"
```

### Memory Issues

**Symptoms**: Out of memory errors, container killed by OOM killer

**Solutions**:
```bash
# Check current memory usage
docker stats agent-s2 --no-stream

# Increase memory limit
docker update agent-s2 --memory 8g

# Add swap space (if needed)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### High CPU Usage

**Symptoms**: System becomes unresponsive, high CPU usage by agent-s2

**Solutions**:
```bash
# Limit CPU usage
docker update agent-s2 --cpus 1.5

# Check for runaway processes
docker exec agent-s2 top

# Restart service to clear processes
./manage.sh --action restart
```

## Network Issues

### Cannot Access External Services

**Symptoms**: HTTP requests fail, DNS resolution issues

**Diagnosis Steps**:
```bash
# Test network connectivity from container
docker exec agent-s2 ping google.com
docker exec agent-s2 nslookup google.com

# Check Docker network configuration
docker network inspect bridge
```

**Solutions**:
```bash
# Restart Docker networking
sudo systemctl restart docker

# Use custom DNS servers
./manage.sh --action install --dns "8.8.8.8,8.8.4.4"

# Check firewall rules
sudo ufw status verbose
```

## Docker-Specific Issues

### Container Crashes

**Symptoms**: Container exits unexpectedly, cannot restart

**Diagnosis Steps**:
```bash
# Check container exit code
docker ps -a | grep agent-s2

# View crash logs
docker logs agent-s2

# Check Docker daemon logs
journalctl -u docker.service
```

**Solutions**:
```bash
# Restart Docker daemon
sudo systemctl restart docker

# Remove and recreate container
./manage.sh --action uninstall
./manage.sh --action install --force yes

# Check for Docker corruption
docker system prune --all
```

### Volume Mount Issues

**Symptoms**: Files not accessible, permission denied on mounted volumes

**Solutions**:
```bash
# Check volume permissions
ls -la /host/mounted/path

# Fix permissions
sudo chown -R 1000:1000 /host/mounted/path

# Remount with correct permissions
./manage.sh --action install --host-mounts "/host/path:/container/path:rw"
```

## Log Analysis

### Accessing Logs

```bash
# View all logs
./manage.sh --action logs

# Follow logs in real-time
./manage.sh --action logs --follow

# Filter logs by severity
./manage.sh --action logs --filter error

# Export logs for analysis
./manage.sh --action logs --output logs.txt
```

### Common Log Patterns

**Error Patterns to Look For**:
```bash
# API errors
grep "HTTP 500" logs.txt

# Display errors
grep "X11\|display\|DISPLAY" logs.txt

# Permission errors
grep "permission denied\|access denied" logs.txt

# Memory errors
grep "out of memory\|OOM\|killed" logs.txt

# Network errors
grep "connection refused\|timeout\|DNS" logs.txt
```

## Getting Help

### Diagnostic Information Collection

Before seeking help, collect this information:

```bash
# System information
./manage.sh --action diagnostics > diagnostics.txt

# Include:
# - Docker version and configuration
# - System resources and usage
# - Container status and logs
# - Configuration files
# - Recent error messages
```

### Support Channels

1. **Check existing documentation**:
   - [Configuration Guide](CONFIGURATION.md)
   - [Security Guide](SECURITY.md)
   - [API Reference](API.md)

2. **Search existing issues**:
   - GitHub Issues
   - Community discussions

3. **Create detailed bug reports**:
   - Include diagnostic information
   - Provide reproduction steps
   - Attach relevant logs

## Prevention and Maintenance

### Regular Maintenance

```bash
# Update container regularly
./manage.sh --action update

# Clean up Docker system
docker system prune

# Monitor resource usage
./manage.sh --action metrics

# Review logs periodically
./manage.sh --action logs --filter warning
```

### Health Monitoring

```bash
# Set up automated health checks
crontab -e
# Add: */5 * * * * curl -f http://localhost:4113/health || echo "Agent S2 unhealthy" | mail -s "Alert" admin@example.com

# Monitor resource usage
./manage.sh --action monitor --alert-threshold 80
```