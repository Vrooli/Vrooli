[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Browserless.

## Table of Contents

- [Common Issues](#common-issues)
  - [Container Won't Start](#container-wont-start)
  - [Chrome Crashes / Out of Memory](#chrome-crashes--out-of-memory)
  - [Timeout Errors](#timeout-errors)
  - [Screenshot Validation Failures](#screenshot-validation-failures)
  - [Connection Refused](#connection-refused)
  - [Performance Issues](#performance-issues)
- [Debugging Procedures](#debugging-procedures)
- [Performance Optimization](#performance-optimization)
- [Maintenance Tasks](#maintenance-tasks)
- [Security Considerations](#security-considerations)
- [FAQ](#faq)

## Common Issues

### API Parameter Errors

**Symptoms:**
- "POST Body validation failed" errors
- Parameters rejected with "is not allowed" messages
- Requests fail despite correct-looking syntax

**Example errors:**
```
POST Body validation failed: "options.width" is not allowed
"options.height" is not allowed
"waitFor" is not allowed
```

**Diagnosis:**
```bash
# Test which parameters are accepted
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}' \
  -o test.png

# Check API version and capabilities
curl http://localhost:4110/config
```

**Solutions:**

1. **Use correct parameter structure:**
   ```bash
   # WRONG - these parameters are not accepted in options
   curl -X POST http://localhost:4110/chrome/screenshot \
     -d '{
       "url": "https://example.com",
       "options": {
         "width": 1920,
         "height": 1080
       }
     }'
   
   # CORRECT - use viewport in gotoOptions
   curl -X POST http://localhost:4110/chrome/screenshot \
     -d '{
       "url": "https://example.com",
       "gotoOptions": {
         "viewport": {
           "width": 1920,
           "height": 1080
         }
       }
     }'
   ```

2. **Common parameter mistakes:**
   - `waitFor` → Use `gotoOptions.waitUntil` or add custom wait logic
   - `options.width/height` → Use `gotoOptions.viewport`
   - `timeout` at root level → Use `gotoOptions.timeout`

### Wrong Endpoint Errors

**Symptoms:**
- "Not Found" responses
- "No route or file found for resource" errors
- 404 errors when endpoint seems correct

**Example errors:**
```
Not Found
No route or file found for resource GET: /
```

**Solutions:**

1. **Use correct endpoint paths:**
   ```bash
   # WRONG - missing /chrome prefix
   curl -X POST http://localhost:4110/screenshot
   
   # CORRECT - includes /chrome prefix
   curl -X POST http://localhost:4110/chrome/screenshot
   ```

2. **Available endpoints:**
   - `/chrome/screenshot` - Take screenshots
   - `/chrome/pdf` - Generate PDFs
   - `/chrome/content` - Get page content
   - `/chrome/scrape` - Extract specific elements
   - `/chrome/function` - Run custom code
   - `/pressure` - Check service health
   - `/metrics` - Get performance metrics

### Container Won't Start

**Symptoms:**
- `docker ps` doesn't show browserless container
- Management script shows "not running"
- Installation appears to fail

**Diagnosis:**
```bash
# Check container logs
docker logs browserless

# Verify shared memory
df -h /dev/shm

# Check if port is already in use
sudo lsof -i :4110

# Check Docker daemon status
sudo systemctl status docker
```

**Solutions:**

1. **Port conflict:**
   ```bash
   # Install on different port
   ./manage.sh --action install --port 4111
   ```

2. **Insufficient shared memory:**
   ```bash
   # Increase shared memory
   ./manage.sh --action install --shared-memory 4gb
   ```

3. **Docker permissions:**
   ```bash
   # Add user to docker group
   sudo usermod -aG docker $USER
   # Log out and back in or run:
   newgrp docker
   ```

### Chrome Crashes / Out of Memory

**Symptoms:**
- "Chrome crashed" errors
- Container restarts frequently
- High memory usage warnings

**Diagnosis:**
```bash
# Check container resource usage
docker stats browserless

# Check system memory
free -h

# View crash logs
docker logs browserless | grep -i crash
```

**Solutions:**

1. **Increase shared memory:**
   ```bash
   docker stop browserless
   docker rm browserless
   docker run -d \
     --name browserless \
     --shm-size=4gb \
     -p 4110:3000 \
     ghcr.io/browserless/chrome:latest
   ```

2. **Reduce concurrent browsers:**
   ```bash
   ./manage.sh --action install --max-browsers 3
   ```

3. **Add swap space (if needed):**
   ```bash
   sudo fallocate -l 4G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   ```

### Timeout Errors

**Symptoms:**
- "Timeout exceeded" errors
- Requests hang and eventually fail
- Slow page loads cause failures

**Diagnosis:**
```bash
# Test with extended timeout
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://slow-site.com",
    "timeout": 60000
  }'

# Check pressure
curl http://localhost:4110/pressure
```

**Solutions:**

1. **Increase default timeout:**
   ```bash
   ./manage.sh --action install --timeout 60000
   ```

2. **Per-request timeout:**
   ```json
   {
     "url": "https://slow-site.com",
     "waitUntil": "networkidle0",
     "timeout": 60000
   }
   ```

3. **Optimize wait conditions:**
   ```json
   {
     "url": "https://example.com",
     "waitUntil": "domcontentloaded",
     "waitFor": 1000
   }
   ```

### Screenshot Validation Failures

**Symptoms:**
- Screenshot commands fail with validation errors
- "HTTP status: 404" or "HTTP status: 500" errors
- "File too small" or "Not an image" errors
- Binary data returned as text

**Common validation errors:**
```bash
# File size too small (usually contains error text)
$ ls -la screenshot.png
-rw-r--r-- 1 user user 10 Jan 1 12:00 screenshot.png

$ cat screenshot.png
Not Found
```

**Diagnosis:**
```bash
# Test the URL directly
curl -I https://your-site.com

# Check with verbose output
./manage.sh --action usage --usage-type screenshot \
  --url https://your-site.com \
  --output test.png

# For local services, check Docker networking
docker network ls
docker inspect browserless | jq '.[0].NetworkSettings'
```

**Solutions:**

1. **For 404/500 errors - verify the URL is correct:**
   ```bash
   # Test URL accessibility
   curl -I https://your-site.com
   ```

2. **For local services (localhost URLs):**
   ```bash
   # Use Docker bridge IP instead of localhost
   ip route | grep docker0  # Usually 172.17.0.1
   
   # Use the bridge IP
   ./manage.sh --action usage --usage-type screenshot \
     --url http://172.17.0.1:8080 \
     --output screenshot.png
   ```

3. **For network isolation issues:**
   ```bash
   # Use host network mode (less secure)
   docker run -d \
     --name browserless \
     --network host \
     ghcr.io/browserless/chrome:latest
   ```

### Connection Refused

**Symptoms:**
- "Connection refused" errors
- Cannot reach http://localhost:4110
- curl commands fail immediately

**Diagnosis:**
```bash
# Check if container is running
docker ps | grep browserless

# Check port binding
docker port browserless

# Test connectivity
nc -zv localhost 4110
```

**Solutions:**

1. **Start the service:**
   ```bash
   ./manage.sh --action start
   ```

2. **Check firewall:**
   ```bash
   # Ubuntu/Debian
   sudo ufw status
   sudo ufw allow 4110

   # RHEL/CentOS
   sudo firewall-cmd --add-port=4110/tcp --permanent
   sudo firewall-cmd --reload
   ```

3. **Verify Docker networking:**
   ```bash
   docker network inspect bridge
   ```

### Performance Issues

**Symptoms:**
- Slow response times
- Queue backing up
- High CPU/memory usage

**Diagnosis:**
```bash
# Check system pressure
curl http://localhost:4110/pressure

# Monitor resource usage
docker stats browserless

# Check queue status
curl http://localhost:4110/metrics | grep queue
```

**Solutions:**

1. **Optimize browser settings:**
   ```bash
   ./manage.sh --action install \
     --max-browsers 10 \
     --timeout 20000
   ```

2. **Disable unnecessary features:**
   ```json
   {
     "args": [
       "--disable-gpu",
       "--disable-dev-shm-usage",
       "--disable-web-security",
       "--no-sandbox"
     ]
   }
   ```

3. **Use connection pooling in your application**

## Debugging Procedures

### Enable Verbose Logging

```bash
# Run with debug output
docker stop browserless
docker rm browserless
docker run -d \
  --name browserless \
  -e DEBUG=browserless* \
  -e LOG_LEVEL=debug \
  -p 4110:3000 \
  ghcr.io/browserless/chrome:latest

# View logs
docker logs -f browserless
```

### Capture Browser Console

```bash
# Get console output from a page
curl -X POST http://localhost:4110/function \
  -H "Content-Type: application/json" \
  -d '{
    "code": "async ({ page }) => {
      page.on(\"console\", msg => console.log(msg.text()));
      await page.goto(\"https://problem-site.com\");
      return { success: true };
    }"
  }'
```

### Test Basic Functionality

```bash
# 1. Test health endpoint
curl http://localhost:4110/pressure

# 2. Test simple screenshot
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}' \
  --output test.png

# 3. Verify output
file test.png  # Should show: PNG image data
```

## Performance Optimization

### Monitor Resource Usage

```bash
# Real-time stats
docker stats browserless

# Pressure monitoring script
while true; do
  curl -s http://localhost:4110/pressure | jq .
  sleep 5
done
```

### Optimization Strategies

1. **Pre-warm browsers:**
   ```bash
   docker run -d \
     -e PREBOOT_CHROME=true \
     -e CONCURRENT=5 \
     ghcr.io/browserless/chrome:latest
   ```

2. **Adjust concurrency based on load:**
   - Light load: 5-10 concurrent
   - Medium load: 10-20 concurrent
   - Heavy load: 20+ (with monitoring)

3. **Memory optimization:**
   ```bash
   # Limit memory per browser
   docker run -d \
     --memory="4g" \
     --memory-swap="4g" \
     ghcr.io/browserless/chrome:latest
   ```

## Maintenance Tasks

### Regular Health Checks

```bash
# Add to crontab
*/5 * * * * curl -f http://localhost:4110/pressure || docker restart browserless
```

### Log Rotation

```bash
# Configure Docker log rotation
cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
EOF
sudo systemctl restart docker
```

### Update Procedure

```bash
# 1. Pull latest image
docker pull ghcr.io/browserless/chrome:latest

# 2. Stop current instance
./manage.sh --action stop

# 3. Remove old container
docker rm browserless

# 4. Start with new image
./manage.sh --action start
```

## Security Considerations

### Running in Production

1. **Enable authentication:**
   ```bash
   docker run -d \
     -e TOKEN=your-secure-token \
     ghcr.io/browserless/chrome:latest
   ```

2. **Restrict network access:**
   ```bash
   # Only allow specific IPs
   iptables -A INPUT -p tcp --dport 4110 -s 10.0.0.0/24 -j ACCEPT
   iptables -A INPUT -p tcp --dport 4110 -j DROP
   ```

3. **Disable dangerous features:**
   ```bash
   docker run -d \
     -e FUNCTION_ENABLE=false \
     -e DOWNLOAD_DIR=/dev/null \
     ghcr.io/browserless/chrome:latest
   ```

### Security Best Practices

- Run with minimal privileges
- Use network isolation
- Enable authentication for external access
- Monitor for unusual activity
- Keep container updated
- Limit resource usage
- Disable unnecessary features

## FAQ

**Q: Can I run multiple Browserless instances?**
A: Yes, use different ports for each instance:
```bash
./manage.sh --action install --port 4110
./manage.sh --action install --port 4111
```

**Q: How do I access Browserless from other containers?**
A: Use the container name as hostname:
```bash
curl http://browserless:3000/pressure
```

**Q: Can I use custom Chrome extensions?**
A: No, Browserless doesn't support Chrome extensions for security reasons.

**Q: How do I handle SSL certificate errors?**
A: Add Chrome arguments:
```json
{
  "args": ["--ignore-certificate-errors"]
}
```

**Q: Why do screenshots of localhost fail?**
A: Docker containers can't access host's localhost. Use:
- Docker bridge IP (172.17.0.1)
- Host network mode
- Container names for other services

**Q: How do I access other Docker services from Browserless?**
A: When Browserless needs to access other containers (like SearXNG):
```bash
# WRONG - localhost refers to Browserless container
curl -X POST http://localhost:4110/chrome/screenshot \
  -d '{"url": "http://localhost:9200"}'

# CORRECT - use container name if on same network
curl -X POST http://localhost:4110/chrome/screenshot \
  -d '{"url": "http://searxng:8080"}'

# CORRECT - use host IP if containers on different networks
curl -X POST http://localhost:4110/chrome/screenshot \
  -d '{"url": "http://172.17.0.1:9200"}'
```

**Q: What are the most common parameter errors?**
A: These parameters often cause confusion:
- `waitFor` - Not a valid parameter, use `gotoOptions.waitUntil` instead
- `options.width/height` - Use `gotoOptions.viewport.width/height`
- `timeout` - Must be inside `gotoOptions`, not at root level
- Direct viewport settings - Must be nested under `gotoOptions`

**Q: How much memory does each browser use?**
A: Typically 100-500MB per browser instance, depending on page complexity.

---
**See also:** [Installation](./INSTALLATION.md) | [Configuration](./CONFIGURATION.md) | [API Reference](./API.md)