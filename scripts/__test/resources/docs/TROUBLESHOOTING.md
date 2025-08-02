# 🔧 Troubleshooting Guide - Fixing Common Issues

**Solutions for the most frequent problems when testing Vrooli resources.**

## 🚨 Emergency Quick Fixes

### **"Nothing runs" or "All tests skipped"**
```bash
# 1. Check what's actually running
../index.sh --action discover

# 2. If nothing shows up, start basic services
../index.sh --action install --resources "ollama"

# 3. Verify the service started
curl http://localhost:11434/api/tags

# 4. Run test again
./run.sh --resource ollama
```

### **"Permission denied" errors**
```bash
# Make scripts executable
chmod +x run.sh
chmod +x single/**/*.test.sh
chmod +x scenarios/**/*.sh

# Fix Docker permissions
sudo usermod -aG docker $USER
newgrp docker

# Test Docker access
docker ps
```

### **"Command not found" errors**
```bash
# Install missing tools
sudo apt update
sudo apt install -y curl jq docker.io

# For macOS
brew install curl jq docker

# Verify installation
curl --version
jq --version
docker --version
```

---

## 🔍 Diagnostic Commands

### **Check System Status**
```bash
# See what's running
../index.sh --action discover

# Check Docker containers
docker ps

# Check ports in use
sudo netstat -tlnp | grep -E "(11434|5678|4113|8090)"

# Check resource configuration
cat ~/.vrooli/resources.local.json | jq '.services'

# Verify directory structure
ls -la single/*/
ls -la framework/
```

### **Test Individual Components**
```bash
# Test framework components
source framework/discovery.sh && echo "Discovery: OK"
source framework/runner.sh && echo "Runner: OK"
source framework/reporter.sh && echo "Reporter: OK"

# Test specific service manually
curl -I http://localhost:11434/api/tags    # Ollama
curl -I http://localhost:5678/            # n8n
curl -I http://localhost:4113/health      # Agent-S2
```

---

## ❌ Common Error Messages & Solutions

### **"Required resources unavailable"**
**Error**: `⏭️ Test skipped - required resources unavailable`

**Causes & Solutions**:
```bash
# Cause 1: Service not running
../ai/ollama/manage.sh --action start

# Cause 2: Service not enabled in config
# Edit ~/.vrooli/resources.local.json and set "enabled": true

# Cause 3: Service running on wrong port
# Check actual port: docker ps
# Update test with correct port

# Cause 4: Service unhealthy
curl http://localhost:11434/health
# Check logs: ../ai/ollama/manage.sh --action logs
```

### **"Test timeout" errors**
**Error**: `❌ Test timed out after 300s`

**Solutions**:
```bash
# Increase timeout for slow services
./run.sh --resource slow-service --timeout 900

# Check if service is responding
curl --max-time 5 http://localhost:YOUR_PORT/health

# Check system resources
top
df -h
free -h

# Restart the service
../your-category/your-service/manage.sh --action restart
```

### **"Connection refused" errors**
**Error**: `curl: (7) Failed to connect to localhost port XXXX: Connection refused`

**Solutions**:
```bash
# Check if service is actually running
docker ps | grep service-name

# Check if port is correct
docker port container-name

# Check firewall (Ubuntu/Debian)
sudo ufw status
sudo ufw allow YOUR_PORT

# Check service logs
../service-category/service-name/manage.sh --action logs

# Restart service
../service-category/service-name/manage.sh --action restart
```

### **"No such file or directory"**
**Error**: `bash: ./run.sh: No such file or directory`

**Solutions**:
```bash
# Make sure you're in the right directory
cd /home/matthalloran8/Vrooli/scripts/resources/tests

# Check if file exists
ls -la run.sh

# Make it executable
chmod +x run.sh

# Check path issues
pwd
echo $PATH
```

### **"Invalid JSON" errors**
**Error**: `parse error: Invalid numeric literal`

**Solutions**:
```bash
# Debug the actual response
curl -v http://localhost:YOUR_PORT/endpoint

# Check if service is returning HTML instead of JSON
curl http://localhost:YOUR_PORT/endpoint | head -10

# Verify service is fully started
# Wait longer or check service logs

# Test with manual JSON validation
echo "$response" | jq . || echo "Invalid JSON"
```

---

## 🐛 Service-Specific Issues

### **Ollama Issues**
```bash
# Problem: No models available
# Solution: Install a model
docker exec -it ollama ollama pull llama3.1:8b

# Problem: Slow inference
# Solution: Check GPU availability
nvidia-smi
# Or use smaller model for testing
docker exec -it ollama ollama pull llama3.2:1b

# Problem: Out of memory
# Solution: Check Docker memory limits
docker stats
# Increase Docker memory allocation or use smaller model
```

### **n8n Issues**  
```bash
# Problem: n8n UI not loading
# Solution: Check if service started completely
curl http://localhost:5678/healthz

# Problem: Workflow execution fails
# Solution: Check n8n logs
../automation/n8n/manage.sh --action logs

# Problem: Database connection issues
# Solution: Check PostgreSQL if using it
docker logs vrooli_postgres
```

### **Agent-S2 Issues**
```bash
# Problem: Screenshot fails
# Solution: Check if display is available
echo $DISPLAY
xhost +local:

# Problem: Mouse/keyboard automation fails  
# Solution: Check if running in container with proper permissions
# Agent-S2 needs access to X11 or Wayland

# Problem: High CPU usage
# Solution: Check if stuck in automation loop
# Kill and restart: ../agents/agent-s2/manage.sh --action restart
```

### **MinIO Issues**
```bash
# Problem: Access denied errors
# Solution: Check credentials
curl -u minioadmin:minioadmin http://localhost:9000/

# Problem: Bucket not found
# Solution: Create default bucket
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/test-bucket

# Problem: Disk space issues
# Solution: Check available space
df -h
docker system prune
```

---

## 🔧 Framework Issues

### **Test Discovery Problems**
```bash
# Problem: Tests not found
# Check test file naming: *.test.sh
ls single/**/*.test.sh

# Check test file permissions
chmod +x single/**/*.test.sh

# Verify directory structure
ls -la single/ai/
ls -la single/automation/
```

### **Import/Source Errors**
```bash
# Problem: Framework helpers not found
# Solution: Check SCRIPT_DIR variable
echo $SCRIPT_DIR

# Fix relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
```

### **Assertion Failures**
```bash
# Problem: Tests fail with assertion errors
# Solution: Debug step by step

# Check what the service actually returns
curl -v http://localhost:PORT/endpoint

# Add debug output to test
echo "Response: $response"
echo "Expected: $expected"

# Use verbose mode
./run.sh --resource yours --verbose --debug
```

---

## 💻 Environment Issues

### **Docker Problems**
```bash
# Docker daemon not running
sudo systemctl start docker
sudo systemctl enable docker

# Docker permission issues
sudo usermod -aG docker $USER
newgrp docker

# Docker out of space
docker system prune -f
docker volume prune -f

# Docker network issues
docker network ls
docker network prune
```

### **Port Conflicts**
```bash
# Find what's using a port
sudo lsof -i :11434
sudo netstat -tlnp | grep 11434

# Kill process using port
sudo kill -9 PID

# Change service port
# Edit service configuration or docker-compose.yml

# Check port registry
./scripts/resources/port-registry.sh --action list
```

### **Memory Issues**
```bash
# Check available memory
free -h

# Check swap
swapon --show

# Monitor memory usage during tests
watch -n 1 'free -h'

# Increase Docker memory limit
# Docker Desktop: Settings > Resources > Advanced
```

### **Network Issues**
```bash
# DNS problems
nslookup localhost
ping localhost

# Firewall blocking
sudo ufw status
sudo iptables -L

# Proxy issues
echo $http_proxy
echo $https_proxy
unset http_proxy https_proxy
```

---

## 🎯 Test-Specific Debugging

### **Single Resource Test Debugging**
```bash
# Run with maximum debugging
./run.sh --resource service --debug --verbose

# Test just the basics first
curl http://localhost:PORT/health

# Check test file syntax
bash -n single/category/service.test.sh

# Run test directly (bypass framework)
cd single/category/
bash service.test.sh
```

### **Scenario Test Debugging**
```bash
# Check scenario dependencies
head -20 scenarios/scenario-name/test.sh | grep "REQUIRED_RESOURCES"

# Verify all required services are running
for service in ollama whisper comfyui; do
    echo "Testing $service..."
    curl -I http://localhost:PORT/
done

# Run scenario with extended timeout
./run.sh --scenarios "scenario-name" --timeout 1800
```

### **Integration Test Debugging**
```bash
# Test each service individually first
./run.sh --resource service1
./run.sh --resource service2

# Check service communication
curl http://service1:port/api/data
curl -X POST http://service2:port/process -d @testdata.json

# Monitor network traffic
sudo tcpdump -i lo port 11434
```

---

## 📊 Performance Issues

### **Slow Test Execution**
```bash
# Profile test execution
time ./run.sh --resource service

# Check service response times
time curl http://localhost:PORT/health

# Monitor system resources
htop
iotop
```

### **Memory Leaks**
```bash
# Monitor memory usage over time
while true; do
    date
    free -h
    sleep 10
done

# Check for memory leaks in containers
docker stats

# Clean up test artifacts
./run.sh --cleanup
```

### **High CPU Usage**
```bash
# Find CPU-intensive processes
top -o %CPU

# Check if services are stuck
ps aux | grep -E "(ollama|n8n|agent)"

# Restart problematic services
../category/service/manage.sh --action restart
```

---

## 🎛️ Configuration Issues

### **Wrong Service URLs**
```bash
# Check actual service URLs
docker ps
docker port container-name

# Update test configuration
# Edit test file BASE_URL or PORT variables

# Verify URL accessibility
curl -I http://localhost:ACTUAL_PORT/
```

### **Missing Environment Variables**
```bash
# Check required environment variables
env | grep -E "(OLLAMA|WHISPER|N8N)"

# Set missing variables
export OLLAMA_BASE_URL="http://localhost:11434"
export WHISPER_BASE_URL="http://localhost:8090"

# Add to shell profile for persistence
echo 'export OLLAMA_BASE_URL="http://localhost:11434"' >> ~/.bashrc
```

### **Incorrect Resource Configuration**
```bash
# Validate configuration file
cat ~/.vrooli/resources.local.json | jq .

# Fix JSON syntax errors
jq . ~/.vrooli/resources.local.json || echo "Invalid JSON"

# Reset to default configuration
cp .vrooli/resources.example.json ~/.vrooli/resources.local.json
```

---

## 🆘 Last Resort Solutions

### **Complete Reset**
```bash
# Stop all services
../index.sh --action stop --resources all

# Clean up everything
docker system prune -af
docker volume prune -f

# Remove configuration
rm ~/.vrooli/resources.local.json

# Start fresh
../index.sh --action install --resources "essential"
```

### **Reinstall Framework**
```bash
# Backup any custom tests
cp -r single/custom/ /tmp/backup/

# Reset framework
git checkout scripts/resources/tests/

# Restore custom tests
cp -r /tmp/backup/ single/custom/
```

### **Getting Help**
```bash
# Generate comprehensive debug report
./run.sh --debug --verbose > debug-report.txt 2>&1

# Include system information
echo "=== SYSTEM INFO ===" >> debug-report.txt
uname -a >> debug-report.txt
docker --version >> debug-report.txt
free -h >> debug-report.txt
df -h >> debug-report.txt

# Include service status
echo "=== SERVICE STATUS ===" >> debug-report.txt
../index.sh --action discover >> debug-report.txt 2>&1

# Share debug-report.txt for support
```

---

## 💡 Prevention Tips

### **Regular Maintenance**
```bash
# Weekly cleanup
docker system prune -f
docker volume prune -f

# Check disk space
df -h

# Update services
../index.sh --action install --resources enabled
```

### **Monitoring**
```bash
# Monitor service health
watch -n 30 '../index.sh --action discover'

# Log rotation
sudo logrotate -f /etc/logrotate.conf

# Backup test configurations
cp ~/.vrooli/resources.local.json ~/.vrooli/backup-$(date +%Y%m%d).json
```

### **Best Practices**
1. **Always test in verbose mode first**
2. **Start with simple services (ollama) before complex scenarios**
3. **Check service health before running tests**
4. **Keep Docker images updated**
5. **Monitor disk space and memory usage**
6. **Use appropriate timeouts for your hardware**

---

**🎉 Still Having Issues?** 

1. Check [GETTING_STARTED.md](GETTING_STARTED.md) for basics
2. Review [COMMON_PATTERNS.md](COMMON_PATTERNS.md) for examples  
3. Generate a debug report and seek help from the community

Most issues are solved by checking service health first: `../index.sh --action discover`