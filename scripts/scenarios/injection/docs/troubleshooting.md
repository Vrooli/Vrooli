# Resource Injection Troubleshooting Guide

**Diagnose and resolve common issues with the Resource Data Injection System.**

## 🎯 Overview

This guide covers common problems, diagnostic steps, and solutions for the Resource Data Injection System. Use this when scenarios fail to inject, configurations are invalid, or resources aren't behaving as expected.

## 🚨 Quick Diagnostics

### System Health Check

```bash
# Check if injection system is properly installed
ls -la scripts/scenarios/injection/
ls -la scripts/scenarios/injection/engine.sh

# Verify permissions
test -x scripts/scenarios/injection/engine.sh && echo "✅ Engine executable" || echo "❌ Engine not executable"

# Check dependencies
command -v jq >/dev/null && echo "✅ jq available" || echo "❌ jq missing"
command -v curl >/dev/null && echo "✅ curl available" || echo "❌ curl missing"
```

### Configuration Validation

```bash
# Validate scenarios configuration
./scripts/scenarios/injection/schema-validator.sh --action validate

# Check specific scenario
./scripts/scenarios/injection/engine.sh --action validate --scenario YOUR_SCENARIO

# List available scenarios
./scripts/scenarios/injection/engine.sh --action list-scenarios
```

### Resource Status

```bash
# Check which resources are running
./scripts/resources/index.sh --action discover

# Check specific resource
./scripts/resources/category/resource/manage.sh --action status
```

## ❌ Common Error Categories

### 1. Configuration Errors

#### **Error**: `Scenario 'xyz' not found in scenarios.json`

**Cause**: Scenario name doesn't exist in configuration file.

**Solution**:
```bash
# List available scenarios
./scripts/scenarios/injection/engine.sh --action list-scenarios

# Check configuration file location
ls -la ~/.vrooli/scenarios.json

# Validate configuration syntax
./scripts/scenarios/injection/schema-validator.sh --action validate
```

#### **Error**: `Invalid JSON in scenarios configuration`

**Cause**: Malformed JSON in scenarios.json.

**Solution**:
```bash
# Check JSON syntax
jq . ~/.vrooli/scenarios.json

# Validate against schema
./scripts/scenarios/injection/schema-validator.sh --action validate

# Fix common JSON issues
# - Missing commas between array elements
# - Unescaped quotes in strings
# - Trailing commas (not allowed in JSON)
```

#### **Error**: `Workflow file not found: /path/to/file.json`

**Cause**: Referenced file doesn't exist or path is incorrect.

**Solution**:
```bash
# Check if file exists (paths are relative to Vrooli root)
ls -la /home/matthalloran8/Vrooli/path/to/file.json

# Verify current working directory
pwd

# Check file permissions
ls -la path/to/file.json
```

### 2. Resource Connectivity Errors

#### **Error**: `Resource is not accessible at http://localhost:PORT`

**Cause**: Resource is not running or not accessible.

**Solution**:
```bash
# Check if resource is running
./scripts/resources/category/resource/manage.sh --action status

# Start resource if needed
./scripts/resources/category/resource/manage.sh --action start

# Check port availability
netstat -tlnp | grep :PORT
# or
ss -tlnp | grep :PORT

# Test connectivity manually
curl -v http://localhost:PORT/health
```

#### **Error**: `curl: command not found`

**Cause**: curl is not installed.

**Solution**:
```bash
# Install curl
sudo apt-get update && sudo apt-get install curl  # Ubuntu/Debian
brew install curl                                  # macOS
```

### 3. Permission Errors

#### **Error**: `Permission denied` when running injection scripts

**Cause**: Scripts don't have execute permissions.

**Solution**:
```bash
# Make scripts executable
chmod +x scripts/scenarios/injection/engine.sh
chmod +x scripts/scenarios/injection/schema-validator.sh
chmod +x scripts/resources/*/inject.sh

# Check permissions
ls -la scripts/scenarios/injection/engine.sh
```

#### **Error**: `Permission denied` when accessing files

**Cause**: Insufficient file permissions.

**Solution**:
```bash
# Check file ownership and permissions
ls -la path/to/file

# Fix permissions if needed
chmod 644 path/to/file.json

# For directories
chmod 755 path/to/directory
```

### 4. Dependency Errors

#### **Error**: `jq: command not found`

**Cause**: jq JSON processor is not installed.

**Solution**:
```bash
# Install jq
sudo apt-get update && sudo apt-get install jq    # Ubuntu/Debian
brew install jq                                    # macOS
```

#### **Error**: `Circular dependency detected involving scenario: X`

**Cause**: Scenarios have circular dependencies.

**Solution**:
```bash
# Review dependency chain in scenarios.json
# Example problem:
# scenario-a depends on scenario-b
# scenario-b depends on scenario-a

# Fix by removing circular dependency or restructuring scenarios
```

### 5. Resource-Specific Errors

#### **n8n Errors**

**Error**: `Failed to import workflow: workflow-name`

**Diagnostics**:
```bash
# Check n8n is accessible
curl -v http://localhost:5678/healthz

# Validate workflow JSON
jq . path/to/workflow.json

# Check n8n logs
./scripts/resources/automation/n8n/manage.sh --action logs
```

**Solutions**:
- Ensure n8n is running and healthy
- Validate workflow JSON format
- Check n8n API version compatibility
- Verify workflow doesn't have name conflicts

#### **Windmill Errors**

**Error**: `Failed to create script/app in Windmill`

**Diagnostics**:
```bash
# Check Windmill status
./scripts/resources/automation/windmill/manage.sh --action status

# Test Windmill API
curl -v http://localhost:5681/api/health
```

#### **PostgreSQL Errors**

**Error**: `psql: connection refused`

**Diagnostics**:
```bash
# Check PostgreSQL status
./scripts/resources/storage/postgres/manage.sh --action status

# Check if PostgreSQL is listening
netstat -tlnp | grep :5432

# Test connection
psql -h localhost -p 5432 -U postgres -c "SELECT version();"
```

## 🔧 Diagnostic Commands

### Configuration Debugging

```bash
# Validate entire configuration
./scripts/scenarios/injection/schema-validator.sh --action validate --verbose yes

# Check specific scenario structure
jq '.scenarios.YOUR_SCENARIO' ~/.vrooli/scenarios.json

# Validate JSON schema itself
./scripts/scenarios/injection/schema-validator.sh --action check-schema

# Test with minimal configuration
echo '{
  "version": "1.0.0",
  "scenarios": {
    "test": {
      "description": "Test scenario",
      "version": "1.0.0",
      "resources": {}
    }
  },
  "active": []
}' | jq . > test-config.json

./scripts/scenarios/injection/schema-validator.sh --action validate --config-file test-config.json
```

### Resource Debugging

```bash
# Check all resource adapters
find scripts/resources -name "inject.sh" -executable

# Test specific adapter
./scripts/resources/automation/n8n/inject.sh --help

# Check resource configuration
jq '.services.ai.ollama' ~/.vrooli/service.json

# Test resource APIs manually
curl -v http://localhost:5678/healthz  # n8n
curl -v http://localhost:11434/api/tags  # Ollama
curl -v http://localhost:5681/api/health  # Windmill
```

### Network Debugging

```bash
# Check listening ports
netstat -tlnp | grep -E ':(5678|11434|5681|1880)'

# Test connectivity
for port in 5678 11434 5681 1880; do
  echo -n "Port $port: "
  nc -z localhost $port && echo "✅ Open" || echo "❌ Closed"
done

# Check for port conflicts
sudo lsof -i :5678  # Check what's using port 5678
```

## 🛠️ Advanced Troubleshooting

### Debug Mode

Enable verbose logging for detailed diagnostics:

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run injection with debug output
./scripts/scenarios/injection/engine.sh --action inject --scenario test-scenario --dry-run yes

# Check specific function
bash -x scripts/scenarios/injection/engine.sh --action validate --scenario test-scenario
```

### Manual Step-by-Step Testing

```bash
# 1. Test schema validation
./scripts/scenarios/injection/schema-validator.sh --action validate

# 2. Test scenario parsing
./scripts/scenarios/injection/engine.sh --action list-scenarios

# 3. Test specific scenario validation
./scripts/scenarios/injection/engine.sh --action validate --scenario YOUR_SCENARIO

# 4. Test dry run
./scripts/scenarios/injection/engine.sh --action inject --scenario YOUR_SCENARIO --dry-run yes

# 5. Test actual injection
./scripts/scenarios/injection/engine.sh --action inject --scenario YOUR_SCENARIO
```

### Configuration Recovery

```bash
# Backup current configuration
cp ~/.vrooli/scenarios.json ~/.vrooli/scenarios.json.backup

# Reset to defaults
./scripts/scenarios/injection/schema-validator.sh --action init

# Restore from backup
cp ~/.vrooli/scenarios.json.backup ~/.vrooli/scenarios.json
```

## 🔍 Specific Problem Solutions

### Problem: "No resources to inject for scenario"

**Cause**: Scenario has empty resources section.

**Solution**:
```json
{
  "scenarios": {
    "my-scenario": {
      "description": "My scenario",
      "version": "1.0.0",
      "resources": {
        "n8n": {
          "workflows": [
            {
              "name": "test-workflow",
              "file": "test-workflow.json",
              "enabled": true
            }
          ]
        }
      }
    }
  }
}
```

### Problem: "Resource doesn't support injection"

**Cause**: Resource doesn't have injection adapter.

**Solution**:
```bash
# Check if inject.sh exists
ls -la scripts/resources/category/resource/inject.sh

# If missing, resource doesn't support injection yet
# Either:
# 1. Remove resource from scenario
# 2. Create injection adapter (see adapter development guide)
# 3. Use different resource that supports injection
```

### Problem: Injection succeeds but resource doesn't show changes

**Cause**: Resource may need restart or refresh.

**Solution**:
```bash
# Restart resource
./scripts/resources/category/resource/manage.sh --action restart

# Check resource logs
./scripts/resources/category/resource/manage.sh --action logs

# Refresh resource UI/interface
# For n8n: Refresh browser tab
# For Windmill: Check workspace sync
```

### Problem: Rollback fails

**Cause**: Rollback actions may not be comprehensive.

**Solution**:
```bash
# Manual cleanup may be required
# Check resource state manually and clean up:

# For n8n: Delete workflows via UI
# For databases: Run DROP statements manually
# For file storage: Remove files/buckets manually

# Then retry injection
./scripts/scenarios/injection/engine.sh --action inject --scenario YOUR_SCENARIO
```

## 📋 Environment-Specific Issues

### Docker Issues

```bash
# Check Docker is running
systemctl status docker

# Check resource containers
docker ps | grep -E '(n8n|windmill|ollama)'

# Check container logs
docker logs CONTAINER_NAME

# Restart container
docker restart CONTAINER_NAME
```

### File System Issues

```bash
# Check disk space
df -h

# Check file system permissions
ls -la ~/.vrooli/

# Fix ownership if needed
sudo chown -R $USER:$USER ~/.vrooli/
```

### Network Issues

```bash
# Check firewall
sudo ufw status

# Check iptables
sudo iptables -L

# Reset network if needed
sudo systemctl restart networking
```

## 🚀 Performance Issues

### Slow Injection

**Symptoms**: Injection takes very long time.

**Diagnostics**:
```bash
# Check system resources
top
htop
iostat

# Check network latency
ping localhost

# Monitor injection progress
./scripts/scenarios/injection/engine.sh --action inject --scenario large-scenario --verbose yes
```

**Solutions**:
- Break large scenarios into smaller pieces
- Inject scenarios in parallel (if independent)
- Optimize resource configurations
- Check for network bottlenecks

### Memory Issues

**Symptoms**: Out of memory errors during injection.

**Solutions**:
```bash
# Check available memory
free -h

# Monitor memory usage during injection
top -p $(pgrep -f injection)

# Increase swap if needed
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

## 📞 Getting Help

### Log Collection

When reporting issues, collect these logs:

```bash
# System information
uname -a
cat /etc/os-release

# Vrooli version and status
cd /path/to/vrooli && git log --oneline -5
./scripts/resources/index.sh --action discover

# Configuration
cat ~/.vrooli/scenarios.json
cat ~/.vrooli/service.json

# Error output
./scripts/scenarios/injection/engine.sh --action inject --scenario PROBLEM_SCENARIO 2>&1 | tee injection-error.log
```

### Minimal Reproduction

Create a minimal test case:

```bash
# Create minimal scenario
cat > minimal-test.json << 'EOF'
{
  "version": "1.0.0",
  "scenarios": {
    "minimal-test": {
      "description": "Minimal test case",
      "version": "1.0.0",
      "resources": {
        "n8n": {
          "workflows": [
            {
              "name": "simple-test",
              "file": "simple-workflow.json",
              "enabled": false
            }
          ]
        }
      }
    }
  },
  "active": ["minimal-test"]
}
EOF

# Create minimal workflow
cat > simple-workflow.json << 'EOF'
{
  "name": "simple-test",
  "active": false,
  "nodes": [
    {
      "parameters": {},
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "typeVersion": 1,
      "position": [250, 300]
    }
  ],
  "connections": {}
}
EOF

# Test minimal case
./scripts/scenarios/injection/engine.sh --action inject --scenario minimal-test --config-file minimal-test.json
```

## 📚 Reference

- **[Main README](../README.md)**: System overview
- **[API Reference](api-reference.md)**: Complete interface documentation
- **[Adapter Development Guide](adapter-development.md)**: Creating new adapters
- **[GitHub Issues](https://github.com/your-repo/issues)**: Report bugs and get help

---

**When in doubt, start with dry-run mode and work your way up to full injection.**