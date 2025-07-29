# n8n Troubleshooting Guide

This guide helps diagnose and resolve common issues with n8n. Issues are organized by symptom with step-by-step solutions.

## Quick Diagnostics

### Health Check Commands

```bash
# Check service status with comprehensive information (recommended)
./manage.sh --action status

# Test all functionality
./manage.sh --action test

# Check Docker container
docker ps | grep n8n

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
netstat -tlnp | grep :5678
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
docker ps -a | grep n8n

# Verify Docker daemon is running (if needed)
systemctl status docker
```

**Solutions**:
```bash
# Method 1: Restart the service
./manage.sh --action restart

# Method 2: Rebuild from scratch
./manage.sh --action install --build-image yes --force yes

# Method 3: Try standard image if custom build fails
./manage.sh --action install --build-image no --force yes

# Method 4: Check for resource constraints
docker system df
docker system prune  # Free up space if needed
```

### Authentication Issues

**Symptoms**: Cannot log in, "Authentication failed" errors, credential loops

**Diagnosis Steps**:
```bash
# Check authentication configuration
./manage.sh --action status | grep -i auth

# Check basic auth settings
docker exec n8n env | grep N8N_BASIC_AUTH
```

**Solutions**:
```bash
# Method 1: Reset authentication
./manage.sh --action install --basic-auth yes --username admin --password newpassword --force yes

# Method 2: Disable authentication temporarily
export N8N_BASIC_AUTH_ACTIVE=false
./manage.sh --action restart

# Method 3: Check credentials in environment
echo $N8N_BASIC_AUTH_USER
echo $N8N_BASIC_AUTH_PASSWORD
```

### Web Interface Not Accessible

**Symptoms**: Browser shows connection refused, timeout, or blank page

**Diagnosis Steps**:
```bash
# Test functionality (recommended)
./manage.sh --action test

# Check if service is responding
curl -f http://localhost:5678/

# Check container networking
docker inspect n8n | grep -A 10 NetworkSettings
```

**Solutions**:
```bash
# Check port mapping
docker port n8n

# Restart with explicit port mapping
./manage.sh --action install --force yes

# Check firewall rules
sudo ufw status
sudo ufw allow 5678
```

## Workflow Issues

### Workflows Not Executing

**Symptoms**: Workflows don't trigger, show "Waiting" status, or fail silently

**Diagnosis Steps**:
```bash
# List all workflows (recommended)
./manage.sh --action workflow-list

# Check workflow status
./manage.sh --action execute --workflow-id WORKFLOW_ID

# Check execution history via API
curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
  "http://localhost:5678/api/v1/executions?limit=10"
```

**Solutions**:
```bash
# Method 1: Activate workflows
./manage.sh --action activate-workflow --workflow-id WORKFLOW_ID

# Method 2: Check for CLI execution bug (known issue in v1.93.0+)
# Use manage.sh or API instead of direct CLI
./manage.sh --action execute --workflow-id WORKFLOW_ID

# Method 3: Restart service to clear stuck executions
./manage.sh --action restart
```

### Workflow Import/Export Issues

**Symptoms**: Cannot import workflows, export generates empty files, JSON errors

**Solutions**:
```bash
# Export workflows safely (recommended)
./manage.sh --action export-workflows --output "backup-$(date +%Y%m%d).json"

# Validate JSON before import
cat workflows-to-import.json | jq '.' > /dev/null && echo "Valid JSON" || echo "Invalid JSON"

# Import workflows step by step
./manage.sh --action import-workflows --input validated-workflows.json

# Check for workflow conflicts
./manage.sh --action workflow-list | grep -i duplicate
```

### Node Execution Errors

**Common Node Error Patterns**:

1. **Execute Command Node Failures**:
   ```bash
   # Test command availability
   docker exec n8n which curl
   docker exec n8n curl --version
   
   # Check PATH configuration
   docker exec n8n echo $PATH
   
   # Verify host access (if using custom image)
   ./manage.sh --action test
   ```

2. **HTTP Request Node Timeouts**:
   ```bash
   # Test network connectivity
   docker exec n8n ping google.com
   docker exec n8n curl -I https://httpbin.org/status/200
   
   # Check DNS resolution
   docker exec n8n nslookup google.com
   ```

3. **Credential Access Issues**:
   ```bash
   # Check credential storage
   curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
     http://localhost:5678/api/v1/credentials
   
   # Verify encryption key
   docker exec n8n env | grep N8N_ENCRYPTION_KEY
   ```

## Host Integration Issues

### Command Execution Failures

**Symptoms**: Execute Command nodes fail, "command not found" errors

**Diagnosis Steps**:
```bash
# Test host command access (recommended)
./manage.sh --action test

# Check if custom image is being used
docker inspect n8n | grep -i image

# Test command execution manually
docker exec n8n which ls
docker exec n8n ls /workspace
```

**Solutions**:
```bash
# Method 1: Ensure custom image is built
./manage.sh --action install --build-image yes --force yes

# Method 2: Use absolute paths in Execute Command nodes
# Instead of: "curl http://example.com"
# Use: "/usr/bin/curl http://example.com"

# Method 3: Check PATH configuration
docker exec n8n printenv PATH
```

### Docker Socket Access Issues

**Symptoms**: Cannot control Docker containers from workflows

**Diagnosis Steps**:
```bash
# Check Docker socket mount
docker exec n8n ls -la /var/run/docker.sock

# Test Docker access
docker exec n8n docker ps
```

**Solutions**:
```bash
# Ensure Docker socket is mounted
docker inspect n8n | grep -A 5 Mounts

# Fix Docker socket permissions
sudo chmod 666 /var/run/docker.sock
./manage.sh --action restart

# Add user to docker group (if applicable)
sudo usermod -aG docker $(whoami)
newgrp docker
```

### File System Access Issues

**Symptoms**: Cannot read/write files, "ENOENT" errors, permission denied

**Solutions**:
```bash
# Check workspace mount
docker exec n8n ls -la /workspace

# Check host home access
docker exec n8n ls -la /host/home

# Fix permissions if needed
chmod -R 755 .
./manage.sh --action restart
```

## Database Issues

### SQLite Database Problems

**Symptoms**: Database locked, corruption errors, slow queries

**Solutions**:
```bash
# Check database file
docker exec n8n ls -la /home/node/.n8n/database.sqlite

# Backup database
./manage.sh --action backup-database --output sqlite-backup.db

# Enable WAL mode for better concurrency
export N8N_DB_SQLITE_ENABLE_WAL=true
./manage.sh --action restart

# Vacuum database to optimize
docker exec n8n sqlite3 /home/node/.n8n/database.sqlite "VACUUM;"
```

### PostgreSQL Connection Issues

**Symptoms**: Cannot connect to PostgreSQL, connection timeout, authentication failed

**Diagnosis Steps**:
```bash
# Test PostgreSQL connectivity
docker exec n8n pg_isready -h postgres-host -p 5432

# Check environment variables
docker exec n8n env | grep N8N_DB_POSTGRES
```

**Solutions**:
```bash
# Verify PostgreSQL settings
export N8N_DB_TYPE=postgres
export N8N_DB_POSTGRESDB_HOST=correct-host
export N8N_DB_POSTGRESDB_USER=correct-user
export N8N_DB_POSTGRESDB_PASSWORD=correct-password

# Restart with correct settings
./manage.sh --action install --database postgres --force yes

# Test connection manually
docker exec n8n psql -h postgres-host -U username -d n8n -c "SELECT 1;"
```

## Performance Issues

### Slow Workflow Execution

**Symptoms**: Workflows take long time to execute, timeouts, high resource usage

**Diagnosis Steps**:
```bash
# Check resource usage (recommended)
./manage.sh --action status

# Monitor container stats
docker stats n8n

# Check execution history
curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
  "http://localhost:5678/api/v1/executions?limit=5&includeData=false"
```

**Solutions**:
```bash
# Increase memory allocation
docker update n8n --memory 2g

# Increase CPU allocation
docker update n8n --cpus 1.5

# Optimize Node.js memory
export NODE_OPTIONS="--max-old-space-size=2048"
./manage.sh --action restart

# Enable execution pruning
export N8N_EXECUTIONS_DATA_PRUNE=true
export N8N_EXECUTIONS_DATA_MAX_AGE=168  # 7 days
```

### Memory Issues

**Symptoms**: Out of memory errors, container restarts, slow performance

**Solutions**:
```bash
# Check current memory usage
docker stats n8n --no-stream

# Increase memory limit
docker update n8n --memory 4g

# Optimize binary data handling
export N8N_DEFAULT_BINARY_DATA_MODE=filesystem
export N8N_BINARY_DATA_TTL=24
./manage.sh --action restart

# Clean execution data
curl -X DELETE -H "X-N8N-API-KEY: $N8N_API_KEY" \
  "http://localhost:5678/api/v1/executions/prune"
```

### Database Performance Issues

**Solutions**:
```bash
# For SQLite - vacuum and analyze
docker exec n8n sqlite3 /home/node/.n8n/database.sqlite "VACUUM; ANALYZE;"

# For PostgreSQL - run maintenance
docker exec postgres psql -U username -d n8n -c "VACUUM ANALYZE;"

# Enable connection pooling
export N8N_DB_POSTGRESDB_POOL_SIZE=20
./manage.sh --action restart
```

## API Issues

### API Authentication Problems

**Symptoms**: API returns 401 errors, "Invalid API key" messages

**Solutions**:
```bash
# Create API key in web interface
# 1. Go to http://localhost:5678
# 2. Settings → n8n API
# 3. Create API key

# Test API key
export N8N_API_KEY="your-api-key-here"
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows

# Verify API key format
echo $N8N_API_KEY | grep -E '^n8n_[a-zA-Z0-9]+$'
```

### API Rate Limiting

**Symptoms**: API returns 429 errors, "Too many requests"

**Solutions**:
```bash
# Configure rate limiting
export N8N_API_RATE_LIMIT_ENABLED=true
export N8N_API_RATE_LIMIT_COUNT=200
export N8N_API_RATE_LIMIT_WINDOW=60
./manage.sh --action restart

# Implement client-side rate limiting
# Wait between API calls, use exponential backoff
```

## Webhook Issues

### Webhooks Not Triggering

**Symptoms**: Webhook endpoints don't respond, workflows don't trigger from webhooks

**Diagnosis Steps**:
```bash
# Test webhook endpoint
curl -X POST -H "Content-Type: application/json" \
  -d '{"test": "data"}' \
  http://localhost:5678/webhook-test/test-endpoint

# Check webhook configuration
curl -H "X-N8N-API-KEY: $N8N_API_KEY" \
  http://localhost:5678/api/v1/workflows | jq '.data[] | select(.nodes[].type == "n8n-nodes-base.webhook")'
```

**Solutions**:
```bash
# Configure webhook URL
export N8N_WEBHOOK_URL=https://your-domain.com
./manage.sh --action restart

# Use test webhooks for development
# Test endpoint: /webhook-test/your-path
# Production endpoint: /webhook/your-path

# Check CORS settings
export N8N_WEBHOOK_CORS_ORIGINS=*
./manage.sh --action restart
```

## Network and Connectivity Issues

### External Service Connection Problems

**Symptoms**: HTTP Request nodes fail, SSL errors, DNS resolution issues

**Solutions**:
```bash
# Test network connectivity from container
docker exec n8n ping google.com
docker exec n8n curl -I https://api.github.com

# Check DNS resolution
docker exec n8n nslookup api.github.com

# Test SSL/TLS connections
docker exec n8n curl -v https://api.github.com

# For self-signed certificates
docker exec n8n curl -k https://self-signed.example.com
```

### Vrooli Resource Integration

**Symptoms**: Cannot connect to other Vrooli resources

**Diagnosis Steps**:
```bash
# Test resource connectivity
docker exec n8n curl -f http://ollama:11434/api/tags
docker exec n8n curl -f http://node-red:1880/

# Check Docker network
docker network ls
docker network inspect bridge | grep n8n
```

**Solutions**:
```bash
# Ensure containers are on same network
docker network inspect bridge | grep -E "(n8n|ollama|node-red)"

# Use container names for connections
# Instead of: http://localhost:11434
# Use: http://ollama:11434

# Check Vrooli resource configuration
cat ~/.vrooli/resources.local.json | jq '.services.automation.n8n'
```

## Log Analysis

### Accessing Logs

```bash
# View all logs (recommended)
./manage.sh --action logs

# Follow logs in real-time
./manage.sh --action logs --follow

# Filter logs by level
docker logs n8n 2>&1 | grep -E "(ERROR|WARN)"

# Export logs for analysis
./manage.sh --action logs > n8n-logs-$(date +%Y%m%d).txt
```

### Common Log Patterns

**Error Patterns to Look For**:
```bash
# Database errors
grep -E "(database|sqlite|postgres)" logs.txt

# Authentication errors
grep -E "(auth|login|credential)" logs.txt

# Workflow execution errors
grep -E "(workflow|execution|node)" logs.txt

# Network errors
grep -E "(ECONNREFUSED|ETIMEDOUT|ENOTFOUND)" logs.txt

# Memory errors
grep -E "(memory|heap|garbage)" logs.txt
```

## Recovery Procedures

### Complete Service Recovery

```bash
# Step 1: Stop service
./manage.sh --action stop

# Step 2: Backup current state
./manage.sh --action export-workflows --output recovery-backup.json
./manage.sh --action backup-database --output recovery-db.backup

# Step 3: Clean installation
./manage.sh --action install --build-image yes --force yes

# Step 4: Restore workflows
./manage.sh --action import-workflows --input recovery-backup.json

# Step 5: Test functionality
./manage.sh --action test
```

### Database Recovery

```bash
# For SQLite corruption
cp /home/node/.n8n/database.sqlite database.sqlite.backup
docker exec n8n sqlite3 database.sqlite.backup ".recover database.sqlite.recovered"

# For PostgreSQL issues
./manage.sh --action backup-database --output pg-backup.sql
./manage.sh --action restore-database --input pg-backup.sql
```

## Prevention and Maintenance

### Regular Maintenance

```bash
# Daily health check
./manage.sh --action status

# Weekly workflow backup
./manage.sh --action export-workflows --output "weekly-backup-$(date +%Y%m%d).json"

# Monthly database maintenance
docker exec n8n sqlite3 /home/node/.n8n/database.sqlite "VACUUM; ANALYZE;"

# Clean old executions
curl -X DELETE -H "X-N8N-API-KEY: $N8N_API_KEY" \
  "http://localhost:5678/api/v1/executions/prune"
```

### Monitoring Setup

```bash
# Set up health monitoring
crontab -e
# Add: */5 * * * * curl -f http://localhost:5678/ || echo "n8n unhealthy" | mail -s "Alert" admin@example.com

# Monitor API availability
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows > /dev/null
```

## Getting Help

### Diagnostic Information Collection

Before seeking help, collect this information:

```bash
# System information
./manage.sh --action status > diagnostics.txt
./manage.sh --action test >> diagnostics.txt
./manage.sh --action logs >> diagnostics.txt

# Include:
# - Docker version and configuration
# - Container status and resource usage
# - Workflow configuration (anonymized)
# - Error messages with timestamps
# - API key status (don't include actual key)
```

### Support Resources

1. **Check documentation**:
   - [Configuration Guide](CONFIGURATION.md)
   - [API Reference](API.md)
   - [n8n Official Docs](https://docs.n8n.io)

2. **Community support**:
   - n8n Community Forum: https://community.n8n.io/
   - GitHub Issues: https://github.com/n8n-io/n8n/issues

3. **Create detailed bug reports**:
   - Include diagnostic information
   - Provide reproduction steps
   - Attach relevant logs and workflow exports (anonymized)

## Known Issues

### CLI Execute Bug (v1.93.0+)

**Issue**: `n8n execute` command fails with authentication errors

**Workaround**: Use manage.sh or REST API instead:
```bash
# Instead of: docker exec n8n n8n execute --id=123
# Use: ./manage.sh --action execute --workflow-id 123
```

**GitHub Issue**: [#15567](https://github.com/n8n-io/n8n/issues/15567)

This troubleshooting guide covers the most common n8n issues. For specific problems not covered here, use the diagnostic commands to gather information and consult the n8n community resources.