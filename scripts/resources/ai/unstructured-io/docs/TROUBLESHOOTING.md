# Unstructured.io Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the Unstructured.io resource.

## 🔍 Quick Diagnostics

Run the validation check first to identify issues:
```bash
./manage.sh --action validate-installation
```

## 🚨 Common Issues and Solutions

### 1. Service Won't Start

**Symptoms:**
- Container fails to start
- `docker ps` doesn't show the container

**Solutions:**

Check Docker status:
```bash
sudo systemctl status docker
```

Check for port conflicts:
```bash
sudo lsof -i :11450
# If occupied, stop the conflicting service or change the port
```

Review container logs:
```bash
./manage.sh --action logs --follow yes
```

Force reinstall:
```bash
./manage.sh --action uninstall
./manage.sh --action install --force yes
```

### 2. API Connection Errors

**Error codes and solutions:**

#### [ERROR:CONNECTION] Failed to connect
```bash
# Check if service is running
./manage.sh --action status

# Restart the service
./manage.sh --action restart

# Check firewall rules
sudo ufw status
```

#### [ERROR:TIMEOUT] Processing timeout
- Use `--strategy fast` for quicker processing
- Process smaller files or split large documents
- Increase timeout in configuration

### 3. Document Processing Failures

#### [ERROR:UNSUPPORTED_TYPE] Unsupported file format
```bash
# Check supported formats
./manage.sh --action info

# Verify file type
file your-document.ext

# Convert to supported format if needed
```

#### [ERROR:FILE_TOO_LARGE] File size exceeded
```bash
# Check current file size
ls -lh your-file.pdf

# Split large PDFs
pdftk large.pdf cat 1-50 output part1.pdf
pdftk large.pdf cat 51-100 output part2.pdf
```

#### [ERROR:INVALID_PARAMS] Invalid parameters
```bash
# Valid strategies: fast, hi_res, auto
./manage.sh --action process --file doc.pdf --strategy hi_res

# Valid output formats: json, markdown, text, elements
./manage.sh --action process --file doc.pdf --output markdown
```

### 4. Cache Issues

**Clear specific file cache:**
```bash
./manage.sh --action clear-cache --file /path/to/file.pdf
```

**Clear all cache:**
```bash
./manage.sh --action clear-all-cache
```

**Disable caching temporarily:**
```bash
export UNSTRUCTURED_IO_CACHE_ENABLED=no
./manage.sh --action process --file doc.pdf
```

### 5. Memory Issues

**Symptoms:**
- Container crashes during processing
- "Out of memory" errors

**Solutions:**

Check container memory usage:
```bash
docker stats vrooli-unstructured-io
```

Increase memory limits (edit docker-compose.yml):
```yaml
deploy:
  resources:
    limits:
      memory: 8G  # Increase from 4G
```

Use fast strategy for memory-intensive documents:
```bash
./manage.sh --action process --file large.pdf --strategy fast
```

### 6. Performance Issues

**Slow processing:**
- Use `--strategy fast` for simple documents
- Enable caching for repeated processing
- Process files in batch mode for efficiency

**Monitor performance:**
```bash
# Check processing time
time ./manage.sh --action process --file doc.pdf --quiet yes

# Monitor resource usage
docker stats vrooli-unstructured-io
```

### 7. Docker Issues

**Permission denied errors:**
```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```

**Container name conflicts:**
```bash
# Remove old container
docker rm -f vrooli-unstructured-io

# Reinstall
./manage.sh --action install --force yes
```

**Image pull failures:**
```bash
# Manual pull with retry
docker pull downloads.unstructured.io/unstructured-io/unstructured-api:0.0.78

# Check Docker Hub status
curl -s https://status.docker.com/
```

## 🛠️ Advanced Debugging

### Enable Debug Logging

Edit docker-compose.yml:
```yaml
environment:
  - UNSTRUCTURED_API_LOG_LEVEL=DEBUG
```

Then restart:
```bash
./manage.sh --action restart
./manage.sh --action logs --follow yes
```

### Test with Curl

Test API directly:
```bash
# Health check
curl -v http://localhost:11450/healthcheck

# Process document
curl -X POST http://localhost:11450/general/v0/general \
  -F "files=@test.pdf" \
  -F "strategy=fast"
```

### Container Shell Access

Access container for debugging:
```bash
docker exec -it vrooli-unstructured-io /bin/bash
```

### Network Diagnostics

Check network connectivity:
```bash
# Test from host
nc -zv localhost 11450

# Check Docker network
docker network inspect vrooli-resources

# Test DNS resolution
docker exec vrooli-unstructured-io nslookup google.com
```

## 📊 Performance Tuning

### Optimize for Speed
```bash
# Use fast strategy
./manage.sh --action process --file doc.pdf --strategy fast

# Disable page breaks
export INCLUDE_PAGE_BREAKS=no

# Reduce chunk size
export CHUNK_CHARS=1000
```

### Optimize for Quality
```bash
# Use hi-res strategy
./manage.sh --action process --file doc.pdf --strategy hi_res

# Enable all languages for OCR
./manage.sh --action process --file doc.pdf --languages "eng,fra,deu,spa"
```

### Batch Processing Tips
```bash
# Process similar documents together
./manage.sh --action process --file "*.pdf" --batch yes --strategy fast

# Use consistent settings for cached results
```

## 🆘 Getting More Help

1. **Check logs thoroughly:**
   ```bash
   ./manage.sh --action logs | grep -i error
   ```

2. **Run test suite:**
   ```bash
   ./test-suite.sh
   ```

3. **Validate installation:**
   ```bash
   ./manage.sh --action validate-installation
   ```

4. **Check resource status:**
   ```bash
   ./scripts/resources/index.sh --action discover
   ```

5. **Review configuration:**
   ```bash
   cat ~/.vrooli/resources.local.json | jq '.services.ai."unstructured-io"'
   ```

## 📝 Reporting Issues

When reporting issues, include:

1. Error messages and codes
2. Output of `./manage.sh --action validate-installation`
3. Docker version: `docker --version`
4. Container logs: `./manage.sh --action logs --tail 50`
5. File type and size causing issues
6. Steps to reproduce the problem

## 🔄 Recovery Procedures

### Complete Reset
```bash
# 1. Stop and remove everything
./manage.sh --action uninstall

# 2. Clear cache
rm -rf ~/.vrooli/cache/unstructured-io

# 3. Remove configuration
rm -f ~/.vrooli/resources.local.json

# 4. Reinstall
./manage.sh --action install

# 5. Validate
./manage.sh --action validate-installation
```

### Restore Default Configuration
```bash
# Reset to defaults
export UNSTRUCTURED_IO_DEFAULT_STRATEGY=hi_res
export UNSTRUCTURED_IO_DEFAULT_LANGUAGES=eng
export UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
export UNSTRUCTURED_IO_CACHE_ENABLED=yes
```

## 🎯 Quick Reference

| Issue | Command |
|-------|---------|
| Check status | `./manage.sh --action status` |
| View logs | `./manage.sh --action logs` |
| Restart service | `./manage.sh --action restart` |
| Clear cache | `./manage.sh --action clear-all-cache` |
| Validate install | `./manage.sh --action validate-installation` |
| Test processing | `./manage.sh --action test` |
| Force reinstall | `./manage.sh --action install --force yes` |

---

For additional support, check the [main documentation](README.md) or the [Unstructured.io official docs](https://docs.unstructured.io/).