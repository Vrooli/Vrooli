# Node-RED Troubleshooting Guide

This guide helps diagnose and resolve common issues with Node-RED. Issues are organized by symptom with step-by-step solutions.

## Quick Diagnostics

### Health Check Commands

```bash
# Check service status with comprehensive information (recommended)
./manage.sh --action status

# Test all functionality
./manage.sh --action test

# Check Docker container
docker ps | grep node-red

# View service logs
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
netstat -tlnp | grep :1880
```

## Service Issues

### Service Won't Start

**Symptoms**: Container fails to start, web interface not accessible, connection refused

**Diagnosis Steps**:
```bash
# Check comprehensive status (recommended)
./manage.sh --action status

# Check logs for startup errors
./manage.sh --action logs

# Check Docker status
docker ps -a | grep node-red

# Verify Docker daemon is running (if needed)
systemctl status docker
```

**Solutions**:
```bash
# Method 1: Restart the service
./manage.sh --action restart

# Method 2: Rebuild from scratch
./manage.sh --action install --force yes --build-image yes

# Method 3: Check for resource constraints
docker system df
docker system prune  # Free up space if needed

# Method 4: Use standard image if custom build fails
./manage.sh --action install --build-image no --force yes
```

### Service Starts But Interface Not Accessible

**Symptoms**: Container running but web interface shows connection refused or timeout

**Diagnosis Steps**:
```bash
# Test functionality (recommended)
./manage.sh --action test

# Check if service is responding
curl -f http://localhost:1880/

# Check container networking
docker inspect node-red | grep -A 10 NetworkSettings
```

**Solutions**:
```bash
# Check port mapping
docker port node-red

# Restart with explicit port mapping
./manage.sh --action install --port 1880 --force yes

# Check firewall rules
sudo ufw status
sudo ufw allow 1880
```

## Flow Issues

### Flows Not Loading or Executing

**Symptoms**: Editor shows no flows, flows don't execute, "Flow not found" errors

**Diagnosis Steps**:
```bash
# List all flows (recommended)
./manage.sh --action flow-list

# Check flow file status
docker exec node-red ls -la /data/flows.json

# Check flow validation
docker exec node-red node-red-admin flows validate
```

**Solutions**:
```bash
# Method 1: Import backup flows
./manage.sh --action flow-import --flow-file backup-flows.json

# Method 2: Reset flows to defaults
docker exec node-red cp /data/flows.json /data/flows.json.backup
./manage.sh --action install --force yes

# Method 3: Check flow file permissions
docker exec node-red chown node-red:node-red /data/flows.json
```

### Flow Execution Errors

**Symptoms**: Flows show errors, nodes have warning states, execution stops unexpectedly

**Common Flow Error Patterns**:
```bash
# Check for specific error patterns in logs
./manage.sh --action logs | grep -E "(Error|TypeError|ReferenceError)"

# Check flow validation
./manage.sh --action test
```

**Solutions**:
1. **Missing dependencies**:
   ```bash
   # Check installed nodes
   docker exec node-red npm list
   
   # Rebuild custom image to install missing nodes
   ./manage.sh --action install --build-image yes --force yes
   ```

2. **Function node errors**:
   ```bash
   # Check function node syntax
   # Look for common issues: missing semicolons, undefined variables
   # Use debug nodes to trace message flow
   ```

3. **Context storage issues**:
   ```bash
   # Reset context storage
   docker exec node-red rm -rf /data/context
   ./manage.sh --action restart
   ```

## Host Integration Issues

### Command Execution Failures

**Symptoms**: exec nodes fail, "command not found" errors, permission denied

**Diagnosis Steps**:
```bash
# Test host command access (recommended)
./manage.sh --action validate-host

# Check command availability
docker exec node-red which curl
docker exec node-red echo $PATH

# Test command execution manually
docker exec node-red curl --version
```

**Solutions**:
```bash
# Method 1: Ensure custom image is built
./manage.sh --action install --build-image yes --force yes

# Method 2: Check PATH configuration
docker exec node-red printenv PATH

# Method 3: Use absolute paths in flows
# Instead of: "curl http://example.com"
# Use: "/usr/bin/curl http://example.com"
```

### Docker Socket Access Issues

**Symptoms**: Docker commands fail from within flows, "docker: command not found"

**Diagnosis Steps**:
```bash
# Test Docker socket access (recommended)
./manage.sh --action validate-docker

# Check Docker socket mount
docker exec node-red ls -la /var/run/docker.sock

# Test Docker access manually
docker exec node-red docker ps
```

**Solutions**:
```bash
# Method 1: Ensure Docker socket is mounted
# Check that manage.sh includes Docker socket mount
./manage.sh --action install --force yes

# Method 2: Fix Docker socket permissions
sudo chmod 666 /var/run/docker.sock
./manage.sh --action restart

# Method 3: Add user to docker group (if applicable)
sudo usermod -aG docker $(whoami)
newgrp docker
```

### File System Access Issues

**Symptoms**: Cannot read/write files, "ENOENT" errors, permission denied on /workspace

**Solutions**:
```bash
# Check workspace mount
docker exec node-red ls -la /workspace

# Check file permissions
ls -la $(pwd)

# Fix permissions if needed
chmod -R 755 .
./manage.sh --action restart
```

## Performance Issues

### Slow Response Times

**Symptoms**: Web interface loads slowly, flows execute slowly, timeouts

**Diagnosis Steps**:
```bash
# Check resource usage (recommended)
./manage.sh --action metrics

# Monitor system resources
htop

# Check container stats
docker stats node-red
```

**Solutions**:
```bash
# Increase memory allocation
docker update node-red --memory 2g

# Increase CPU allocation
docker update node-red --cpus 1.5

# Optimize flows
# - Reduce debug node output
# - Limit concurrent executions
# - Use delays between operations
```

### Memory Issues

**Symptoms**: Out of memory errors, container restarts, performance degradation

**Solutions**:
```bash
# Check current memory usage
docker stats node-red --no-stream

# Increase memory limit
docker update node-red --memory 2g

# Optimize Node.js memory
export NODE_OPTIONS="--max-old-space-size=1024"
./manage.sh --action restart

# Clear context storage
docker exec node-red rm -rf /data/context
```

### Dashboard Not Loading

**Symptoms**: Dashboard UI not accessible, blank dashboard, CSS/JS not loading

**Diagnosis Steps**:
```bash
# Check dashboard access
curl -f http://localhost:1880/ui

# Check for dashboard nodes in flows
./manage.sh --action flow-list | grep -i dashboard
```

**Solutions**:
```bash
# Reinstall with dashboard support
./manage.sh --action install --build-image yes --force yes

# Clear browser cache
# Try accessing dashboard in incognito mode

# Check dashboard configuration in settings.js
docker exec node-red cat /data/settings.js | grep -A 10 ui
```

## Flow Development Issues

### Function Node Errors

**Common JavaScript Errors in Function Nodes**:

1. **Undefined variables**:
   ```javascript
   // Wrong
   let data = msg.undefinedProperty.value;
   
   // Correct
   let data = msg.payload && msg.payload.value ? msg.payload.value : 'default';
   ```

2. **Async operations**:
   ```javascript
   // Wrong (doesn't wait for async)
   const result = exec('ls -la');
   msg.payload = result;
   
   // Correct
   const { exec } = require('child_process');
   exec('ls -la', (error, stdout, stderr) => {
       if (error) {
           node.error(error.message, msg);
           return;
       }
       msg.payload = stdout;
       node.send(msg);
   });
   return; // Don't send message synchronously
   ```

3. **Context usage**:
   ```javascript
   // Wrong
   let data = context.myData;
   
   // Correct
   let data = context.get('myData') || {};
   ```

### Import/Export Issues

**Symptoms**: Cannot import flows, export generates empty files, flow corruption

**Solutions**:
```bash
# Export flows safely
./manage.sh --action flow-export --output "backup-$(date +%Y%m%d).json"

# Validate JSON before import
cat flows-to-import.json | jq '.' > /dev/null && echo "Valid JSON" || echo "Invalid JSON"

# Import flows step by step
# 1. Backup current flows first
# 2. Import new flows
# 3. Validate and test
./manage.sh --action flow-export --output current-backup.json
./manage.sh --action flow-import --flow-file new-flows.json
./manage.sh --action test
```

## Network and Connectivity Issues

### HTTP Request Node Failures

**Symptoms**: HTTP requests timeout, connection refused, SSL errors

**Solutions**:
```bash
# Test network connectivity from container
docker exec node-red ping google.com
docker exec node-red curl -I https://httpbin.org/status/200

# Check DNS resolution
docker exec node-red nslookup google.com

# For SSL issues, check certificates
docker exec node-red curl -k https://self-signed.example.com
```

### MQTT Connection Issues

**Symptoms**: MQTT nodes show disconnected, broker unreachable

**Solutions**:
```bash
# Test MQTT connectivity
docker exec node-red ping mqtt-broker-host

# Check MQTT broker configuration
# Verify: host, port, username, password, SSL settings

# Test with mosquitto client
docker exec node-red mosquitto_pub -h broker-host -t test -m "hello"
```

## Authentication and Security Issues

### Admin Interface Access

**Symptoms**: Cannot access admin interface, authentication loops, credential errors

**Solutions**:
```bash
# Check authentication configuration
docker exec node-red cat /data/settings.js | grep -A 10 adminAuth

# Reset authentication (disables auth)
docker exec node-red sed -i 's/adminAuth:/\/\/adminAuth:/' /data/settings.js
./manage.sh --action restart

# Generate new password hash
docker exec node-red node -e "console.log(require('bcryptjs').hashSync('newpassword', 8))"
```

### HTTP Authentication Issues

**Solutions**:
```bash
# Check HTTP auth configuration
docker exec node-red cat /data/settings.js | grep -A 5 httpNodeAuth

# Test with credentials
curl -u username:password http://localhost:1880/api/endpoint
```

## Integration Issues

### Vrooli Resource Integration

**Symptoms**: Cannot connect to other Vrooli resources, network errors

**Diagnosis Steps**:
```bash
# Test resource connectivity
docker exec node-red curl -f http://ollama:11434/api/tags
docker exec node-red curl -f http://n8n:5678/healthz

# Check Docker network
docker network ls
```

**Solutions**:
```bash
# Ensure containers are on same network
docker network inspect bridge | grep node-red
docker network inspect bridge | grep ollama

# Use container names for connections
# Instead of: http://localhost:11434
# Use: http://ollama:11434
```

## Log Analysis

### Accessing Logs

```bash
# View all logs (recommended)
./manage.sh --action logs

# Follow logs in real-time
./manage.sh --action logs --tail 100 --follow

# Filter logs by level
docker logs node-red 2>&1 | grep -E "(ERROR|WARN)"

# Export logs for analysis
./manage.sh --action logs > node-red-logs-$(date +%Y%m%d).txt
```

### Common Log Patterns

**Error Patterns to Look For**:
```bash
# Flow errors
grep "Flow error" logs.txt

# Node errors
grep "node.error" logs.txt

# HTTP errors
grep "HTTP 5[0-9][0-9]" logs.txt

# Memory errors
grep -E "(out of memory|heap|garbage)" logs.txt

# Network errors
grep -E "(ECONNREFUSED|ETIMEDOUT|ENOTFOUND)" logs.txt
```

## Recovery Procedures

### Complete Service Recovery

```bash
# Step 1: Stop service
./manage.sh --action stop

# Step 2: Backup current state
docker cp node-red:/data ./node-red-recovery-backup

# Step 3: Clean installation
./manage.sh --action install --force yes --build-image yes

# Step 4: Restore flows if needed
./manage.sh --action flow-import --flow-file backup-flows.json

# Step 5: Test functionality
./manage.sh --action test
```

### Data Recovery

```bash
# Recover from volume backup
docker volume create node-red-data-recovery
docker run --rm -v node-red-data-recovery:/recovery -v $(pwd)/backup:/backup alpine cp -r /backup/* /recovery/

# Restore specific files
docker cp ./flows-backup.json node-red:/data/flows.json
./manage.sh --action restart
```

## Prevention and Maintenance

### Regular Maintenance

```bash
# Daily health check
./manage.sh --action status

# Weekly flow backup
./manage.sh --action flow-export --output "weekly-backup-$(date +%Y%m%d).json"

# Monthly resource cleanup
docker system prune
./manage.sh --action metrics

# Update custom image
./manage.sh --action install --build-image yes --force yes
```

### Monitoring Setup

```bash
# Set up health monitoring
crontab -e
# Add: */5 * * * * curl -f http://localhost:1880/ || echo "Node-RED unhealthy" | mail -s "Alert" admin@example.com

# Monitor resource usage
./manage.sh --action metrics --alert-threshold 80
```

## Getting Help

### Diagnostic Information Collection

Before seeking help, collect this information:

```bash
# System information
./manage.sh --action status > diagnostics.txt
./manage.sh --action test >> diagnostics.txt
./manage.sh --action logs --tail 100 >> diagnostics.txt

# Include:
# - Docker version and configuration
# - Container status and resource usage
# - Flow configuration (anonymized)
# - Error messages with timestamps
```

### Support Resources

1. **Check documentation**:
   - [Configuration Guide](CONFIGURATION.md)
   - [API Reference](API.md)
   - [Flow examples](../flows/README.md)

2. **Community support**:
   - Node-RED Forum: https://discourse.nodered.org/
   - Node-RED Slack: https://nodered.org/slack/

3. **Create detailed bug reports**:
   - Include diagnostic information
   - Provide reproduction steps
   - Attach relevant logs and flow exports

This troubleshooting guide covers the most common Node-RED issues. For specific problems not covered here, use the diagnostic commands to gather information and consult the Node-RED community resources.