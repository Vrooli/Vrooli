# SearXNG Troubleshooting Guide

This guide covers common issues and their solutions when working with SearXNG.

## 🚨 Common Issues

### 1. Permission Denied Errors
**Symptoms**: Container fails to start with permission errors on config files

**Solution**:
```bash
# Fix permissions on config directory
docker run --rm -v ~/.searxng:/fix alpine chmod -R 777 /fix

# Alternative: Reset permissions
sudo chown -R $USER:$USER ~/.searxng
chmod -R 755 ~/.searxng
```

### 2. Container Keeps Restarting
**Symptoms**: SearXNG container continuously restarts

**Diagnosis**:
```bash
# Check container logs
docker logs searxng

# Check container status
docker ps -a | grep searxng
```

**Common Causes & Solutions**:
- **Invalid config syntax**: Verify YAML syntax in `~/.searxng/settings.yml`
- **Port conflicts**: Check if port 9200 is already in use with `lsof -i :9200`
- **Memory issues**: Ensure sufficient Docker memory allocation

### 3. 403 Forbidden on API Calls
**Symptoms**: API returns 403 Forbidden error

**Solution**:
```bash
# Ensure GET method is enabled in settings.yml
echo 'server:
  method: "GET"
  limiter: false' >> ~/.searxng/settings.yml

# Verify JSON format is enabled
echo 'search:
  formats:
    - html
    - json
    - csv' >> ~/.searxng/settings.yml

# Restart service
./manage.sh --action restart
```

### 4. No Results Returned
**Symptoms**: Search queries return empty results or errors

**Diagnosis Steps**:
```bash
# Test basic connectivity
curl -s "http://localhost:9200/search?q=test&format=json" | jq -r '.query'

# Check if SearXNG is accessible
curl -s http://localhost:9200/stats | jq .

# Test specific search engines
curl -s "http://localhost:9200/search?q=test&format=json&engines=google"
```

**Common Solutions**:
- **Network connectivity**: Ensure Docker can reach external search engines
- **Search engine blocking**: Some engines may temporarily block requests
- **Rate limiting**: Check if engines are rate-limiting requests

### 5. Slow Response Times
**Symptoms**: Search queries take unusually long to complete

**Optimization**:
```bash
# Check current timeout settings
cat ~/.searxng/settings.yml | grep timeout

# Reduce timeouts in settings.yml
outgoing:
  request_timeout: 3.0
  max_request_timeout: 8.0
```

### 6. High Memory Usage
**Symptoms**: SearXNG container uses excessive memory

**Solutions**:
```bash
# Restart container to clear cache
./manage.sh --action restart

# Limit Docker memory if needed
docker update --memory="512m" searxng

# Check for memory leaks in logs
docker logs searxng | grep -i memory
```

## 🔧 Diagnostic Commands

### Health Checks
```bash
# Basic health check
./manage.sh --action status

# Comprehensive diagnostics
./manage.sh --action diagnose

# API functionality test
./manage.sh --action api-test

# Performance benchmark
./manage.sh --action benchmark --count 5
```

### Log Analysis
```bash
# View recent logs
./manage.sh --action logs

# Follow logs in real-time
docker logs -f searxng

# Filter for specific issues
docker logs searxng 2>&1 | grep -i "error\|warning\|timeout"

# Check startup logs
docker logs searxng 2>&1 | head -50
```

### Configuration Validation
```bash
# Check YAML syntax
python3 -c "import yaml; yaml.safe_load(open('~/.searxng/settings.yml'))" 2>/dev/null && echo "Config OK" || echo "Config Error"

# View current configuration
./manage.sh --action config-show

# Compare with template
diff ~/.searxng/settings.yml config/settings.yml.template
```

## 🔄 Reset and Recovery

### Soft Reset
```bash
# Restart service
./manage.sh --action restart

# Clear Docker cache
docker system prune -f

# Reset to default config (backup current first)
cp ~/.searxng/settings.yml ~/.searxng/settings.yml.backup
./manage.sh --action reset-config
```

### Complete Reset
```bash
# Stop and remove container
./manage.sh --action stop
docker rm -f searxng

# Remove configuration (backup first!)
mv ~/.searxng ~/.searxng.backup.$(date +%Y%m%d_%H%M%S)

# Reinstall from scratch
./manage.sh --action install
```

### Data Recovery
```bash
# Restore from backup
cp ~/.searxng.backup.*/settings.yml ~/.searxng/

# Restore specific configuration
./manage.sh --action install --restore-config ~/.searxng.backup.*/
```

## 🐛 Debugging Mode

### Enable Debug Logging
Add to `~/.searxng/settings.yml`:
```yaml
general:
  debug: true
  
logging:
  level: DEBUG
```

### Debug API Calls
```bash
# Test with verbose output
curl -v "http://localhost:9200/search?q=test&format=json"

# Time API responses
time curl -s "http://localhost:9200/search?q=performance+test&format=json" > /dev/null

# Monitor network activity
docker exec searxng netstat -tuln
```

## 📞 Getting Help

### Check Resource Status
```bash
# Overall resource health
./scripts/resources/index.sh --action discover | grep searxng

# Specific SearXNG status
./manage.sh --action status --verbose
```

### Log Collection
```bash
# Collect diagnostic information
./manage.sh --action collect-logs > searxng-diagnostics.txt

# Include system information
echo "=== System Info ===" >> searxng-diagnostics.txt
docker --version >> searxng-diagnostics.txt
docker-compose --version >> searxng-diagnostics.txt
uname -a >> searxng-diagnostics.txt
```

### Common Solutions Summary

| Problem | Quick Fix |
|---------|-----------|
| Permission errors | `docker run --rm -v ~/.searxng:/fix alpine chmod -R 777 /fix` |
| 403 Forbidden | Check `method: "GET"` and `formats: [json]` in settings.yml |
| No results | Test engines individually, check network connectivity |
| Slow responses | Reduce timeouts in configuration |
| High memory | Restart container, limit Docker memory |
| Container won't start | Check logs, verify config syntax, check port conflicts |

For persistent issues, collect logs and configuration details before seeking support.