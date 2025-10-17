# MinIO Troubleshooting Guide

This guide covers common issues and their solutions when working with MinIO object storage.

## 🚨 Installation and Startup Issues

### Port Conflicts

**Symptoms**: Cannot bind to port errors, installation fails

**Diagnosis**:
```bash
# Check what's using the ports
sudo lsof -i :9000
sudo lsof -i :9001

# Check if other services are running
netstat -tlnp | grep -E "(9000|9001)"
```

**Solutions**:
```bash
# Use custom ports
MINIO_CUSTOM_PORT=9100 MINIO_CUSTOM_CONSOLE_PORT=9101 \
  resource-minio manage install

# Stop conflicting services if identified
sudo systemctl stop conflicting-service

# Verify ports are available
resource-minio status
```

### Container Won't Start

**Symptoms**: Docker container fails to start or exits immediately

**Diagnosis**:
```bash
# Check container status
docker ps -a | grep minio

# View startup logs
resource-minio logs --lines 100

# Check Docker daemon
sudo systemctl status docker
```

**Common Solutions**:

1. **Insufficient Disk Space**:
   ```bash
   # Check disk space
   df -h ~/.minio/
   
   # Clean up if needed
   docker system prune -f
   rm -rf ~/.minio/data/.minio.sys/tmp/*
   ```

2. **Permission Issues**:
   ```bash
   # Fix directory permissions
   chmod -R 755 ~/.minio/
   chmod 600 ~/.minio/config/credentials
   
   # Ensure Docker can access the directory
   ls -la ~/.minio/
   ```

3. **Corrupted Configuration**:
   ```bash
   # Reset configuration
   resource-minio manage stop
   rm -rf ~/.minio/config/
   resource-minio manage install
   ```

### Docker Issues

**Symptoms**: Docker-related errors during installation

**Solutions**:
```bash
# Ensure Docker is running
sudo systemctl start docker
sudo systemctl enable docker

# Check Docker access permissions
sudo usermod -aG docker $USER
newgrp docker

# Verify Docker installation
docker --version
docker run hello-world
```

## 🔐 Authentication and Access Issues

### Can't Access Console

**Symptoms**: Cannot load web console at http://localhost:9001

**Diagnosis**:
```bash
# Verify container is running
docker ps | grep minio

# Check console port
curl -I http://localhost:9001

# Check network connectivity
resource-minio status
```

**Solutions**:
```bash
# Verify correct credentials
resource-minio credentials

# Check if console is disabled
docker exec minio printenv | grep MINIO_BROWSER

# Restart container
resource-minio manage restart

# Access via different address
curl -I http://127.0.0.1:9001
```

### Invalid Credentials

**Symptoms**: Access denied errors, authentication failures

**Solutions**:
```bash
# Show current credentials
resource-minio credentials

# Reset credentials if needed
resource-minio content execute --name reset-credentials

# Verify credentials file
cat ~/.minio/config/credentials
ls -la ~/.minio/config/credentials  # Should be 600 permissions

# Test credentials with curl
curl -u username:password http://localhost:9000/
```

### S3 API Authentication Errors

**Symptoms**: AWS SDK or CLI authentication failures

**Diagnosis**:
```bash
# Test basic API access
curl -I http://localhost:9000/

# Check credentials format
resource-minio credentials

# Test with AWS CLI
aws s3 ls --endpoint-url http://localhost:9000 --no-verify-ssl
```

**Solutions**:
```bash
# Configure AWS CLI correctly
CREDS=$(resource-minio credentials)
ACCESS_KEY=$(echo "$CREDS" | grep Username | cut -d' ' -f2)
SECRET_KEY=$(echo "$CREDS" | grep Password | cut -d' ' -f2)

aws configure set aws_access_key_id "$ACCESS_KEY"
aws configure set aws_secret_access_key "$SECRET_KEY"
aws configure set default.region us-east-1

# Test connection
aws s3 ls --endpoint-url http://localhost:9000
```

## 📦 Storage and Data Issues

### Bucket Operations Fail

**Symptoms**: Cannot create, delete, or access buckets

**Diagnosis**:
```bash
# List existing buckets
resource-minio content list

# Check bucket permissions
docker exec minio mc ls local/

# Verify bucket policies
docker exec minio mc policy list local/bucket-name
```

**Solutions**:
```bash
# Create bucket with proper policy
resource-minio content execute --name create-bucket --bucket test-bucket --policy none

# Fix bucket permissions
docker exec minio mc policy set download local/public-bucket

# Remove problematic bucket
resource-minio content execute --name remove-bucket --bucket problem-bucket --force yes
```

### File Upload/Download Failures

**Symptoms**: Cannot upload or download files

**Diagnosis**:
```bash
# Test basic upload/download
resource-minio test smoke

# Check available space
df -h ~/.minio/data/

# Monitor real-time operations
docker logs -f minio
```

**Solutions**:
```bash
# Free up disk space
docker exec minio find /data -name "*.tmp" -delete
docker exec minio mc rm --recursive --force local/vrooli-temp-storage/

# Check file permissions
ls -la ~/.minio/data/bucket-name/

# Test with simple file
echo "test" | docker exec -i minio mc pipe local/test-bucket/test.txt
docker exec minio mc cat local/test-bucket/test.txt
```

### Data Corruption Issues

**Symptoms**: Files are corrupted or inaccessible

**Diagnosis**:
```bash
# Check file integrity
docker exec minio mc stat local/bucket-name/file.txt

# Verify checksums if available
docker exec minio mc cat local/bucket-name/file.txt | md5sum

# Check system logs for errors
dmesg | grep -i error
```

**Solutions**:
```bash
# Restore from backup
resource-minio manage stop
tar -xzf minio-backup-date.tar.gz -C ~/
resource-minio manage start

# Run file system check
sudo fsck ~/.minio/data/

# Re-upload corrupted files
# Check backup sources for original files
```

## 🌐 Network and Connectivity Issues

### API Endpoint Not Responding

**Symptoms**: Connection refused, timeout errors

**Diagnosis**:
```bash
# Test local connectivity
curl -v http://localhost:9000/

# Check container networking
docker network ls | grep minio
docker network inspect minio-network

# Verify port mapping
docker port minio
```

**Solutions**:
```bash
# Restart networking
resource-minio manage restart

# Check firewall rules
sudo ufw status
sudo iptables -L

# Test different addresses
curl http://127.0.0.1:9000/
curl http://$(hostname -I | cut -d' ' -f1):9000/
```

### Slow Performance

**Symptoms**: Slow upload/download speeds, timeouts

**Diagnosis**:
```bash
# Monitor resource usage
docker stats minio

# Check disk I/O
iostat -x 1

# Test network speed
time curl -o /dev/null http://localhost:9000/bucket/large-file
```

**Solutions**:
```bash
# Increase resource limits
# Edit lib/docker.sh to add:
# --memory="4g" --cpus="4.0"

# Optimize disk performance
# Move to SSD if using HDD
# Check mount options: noatime,nodiratime

# Increase API limits
MINIO_API_REQUESTS_MAX=2000 resource-minio manage restart

# Check for competing processes
top -p $(pgrep -f minio)
```

## 🛠️ Service Management Issues

### Service Won't Stop

**Symptoms**: resource-minio manage stop doesn't work

**Solutions**:
```bash
# Force stop container
docker stop minio
docker kill minio

# Check if process is still running
docker ps -a | grep minio

# Remove container if needed
docker rm -f minio

# Clean restart
resource-minio manage install
```

### Logs Not Available

**Symptoms**: Cannot view logs, empty log output

**Solutions**:
```bash
# Check container exists
docker ps -a | grep minio

# Try different log commands
docker logs minio
docker logs --details minio
docker logs --timestamps minio

# Check log file directly
docker exec minio ls -la /root/.minio/

# Enable debug logging
export MINIO_ROOT_LOG_LEVEL=DEBUG
resource-minio manage restart
```

### Health Check Failures

**Symptoms**: Service reports as unhealthy

**Diagnosis**:
```bash
# Check health endpoint
curl http://localhost:9000/minio/health/live
curl http://localhost:9000/minio/health/ready

# Run diagnostics
resource-minio content execute --name diagnose

# Check container health
docker inspect minio | grep -A 10 Health
```

**Solutions**:
```bash
# Restart service
resource-minio manage restart

# Check resource availability
free -h
df -h ~/.minio/

# Verify configuration
resource-minio status --verbose
```

## 📊 Performance and Resource Issues

### High Memory Usage

**Symptoms**: Excessive memory consumption, system slowdown

**Diagnosis**:
```bash
# Monitor memory usage
docker stats minio --no-stream

# Check memory limits
docker inspect minio | grep -i memory

# System memory status
free -h
```

**Solutions**:
```bash
# Set memory limits
# Edit lib/docker.sh to add: --memory="2g"

# Restart with limits
resource-minio manage stop
# Modify docker run command in lib/docker.sh
resource-minio manage start

# Clear caches
docker exec minio sync
docker exec minio sh -c 'echo 3 > /proc/sys/vm/drop_caches'
```

### Disk Space Issues

**Symptoms**: No space left on device, quota exceeded

**Solutions**:
```bash
# Check disk usage
df -h ~/.minio/data/
du -sh ~/.minio/data/*/

# Clean temporary files
docker exec minio find /data/.minio.sys/tmp -type f -mtime +1 -delete

# Remove old objects
docker exec minio mc rm --recursive --older-than 30d local/vrooli-temp-storage/

# Enable lifecycle policies
docker exec minio mc ilm add local/large-bucket --expire-days 30

# Move to larger disk
# 1. Stop MinIO
# 2. Move ~/.minio/data to new location
# 3. Create symlink or update mount point
# 4. Restart MinIO
```

## 🔄 Recovery and Maintenance

### Complete System Reset

**When all else fails**:

```bash
# 1. Backup important data first
docker exec minio mc mirror local/important-bucket /backup/important-bucket/

# 2. Complete uninstall
resource-minio manage uninstall --remove-data yes

# 3. Clean up any remaining files
docker system prune -f
rm -rf ~/.minio/

# 4. Fresh installation
resource-minio manage install

# 5. Restore critical data
docker exec minio mc mirror /backup/important-bucket local/important-bucket/
```

### Data Recovery from Backup

```bash
# Stop MinIO service
resource-minio manage stop

# Backup current state (just in case)
mv ~/.minio/data ~/.minio/data.backup.$(date +%s)

# Restore from backup
mkdir -p ~/.minio/data
tar -xzf minio-backup-20240115.tar.gz -C ~/.minio/

# Fix permissions
chmod -R 755 ~/.minio/data
chmod 600 ~/.minio/config/credentials

# Start service
resource-minio manage start

# Verify restoration
resource-minio content list
```

## 🔍 Diagnostic Commands

### Comprehensive Health Check

```bash
# Full diagnostic report
{
  echo "=== MinIO Status ==="
  resource-minio status
  echo "=== Container Info ==="
  docker inspect minio
  echo "=== Resource Usage ==="
  docker stats minio --no-stream
  echo "=== Disk Usage ==="
  df -h ~/.minio/
  du -sh ~/.minio/data/*/
  echo "=== Network ==="
  docker network inspect minio-network
  echo "=== Recent Logs ==="
  docker logs --tail 50 minio
} > minio-diagnostic-report.txt
```

### Log Analysis

```bash
# Search for specific errors
docker logs minio 2>&1 | grep -E "(ERROR|FATAL|panic)"

# API request logs
docker logs minio 2>&1 | grep "API:"

# Performance metrics
docker logs minio 2>&1 | grep -E "(slow|timeout|latency)"

# Authentication issues
docker logs minio 2>&1 | grep -E "(auth|credential|access denied)"
```

## 📞 Getting Help

### Information to Collect

Before seeking support, collect:

```bash
# System information
uname -a
docker --version
df -h ~/.minio/

# MinIO status
resource-minio status
resource-minio credentials

# Container details
docker ps | grep minio
docker logs --tail 100 minio

# Configuration
ls -la ~/.minio/config/
cat ~/.vrooli/service.json | jq '.services.storage.minio'
```

### Quick Fix Summary

| Problem | Quick Fix |
|---------|-----------|
| Port conflicts | Use `MINIO_CUSTOM_PORT=9100` |
| Container won't start | Check `resource-minio logs` |
| Can't access console | Verify `resource-minio credentials` |
| Authentication failed | Run `resource-minio content execute --name reset-credentials` |
| Bucket operations fail | Check `resource-minio content list` |
| Out of disk space | Clean temp files with `docker exec minio find /data/.minio.sys/tmp -delete` |
| Slow performance | Monitor with `docker stats minio` |
| Complete failure | Run `resource-minio content execute --name diagnose` |

Most issues can be resolved by checking logs with `resource-minio logs` and running diagnostics with `resource-minio content execute --name diagnose`.