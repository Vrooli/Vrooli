# ComfyUI Troubleshooting Guide

This guide helps diagnose and resolve common issues with ComfyUI. Issues are organized by symptom with step-by-step solutions.

## Quick Diagnostics

### Health Check Commands

```bash
# Check service status with comprehensive information (recommended)
./manage.sh --action status

# Get GPU information and capabilities
./manage.sh --action gpu-info

# List installed models
./manage.sh --action list-models

# View service logs
./manage.sh --action logs

# Test basic functionality
curl -f http://localhost:8188/system_stats
```

### System Requirements Verification

```bash
# Check Docker version
docker --version

# Check available memory (16GB+ recommended)
free -h

# Check available disk space (50GB+ recommended)
df -h

# Check port availability
netstat -tlnp | grep :8188
netstat -tlnp | grep :5679
```

## Installation Issues

### NVIDIA Container Runtime Issues

**Symptoms**: NVIDIA GPU not detected, Docker can't access GPU, "nvidia-smi" fails in container

**Diagnosis Steps:**
```bash
# Test NVIDIA driver installation (recommended)
nvidia-smi

# Validate NVIDIA runtime setup
./manage.sh --action validate-nvidia

# Check Docker NVIDIA runtime
docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi
```

**Solutions:**
```bash
# Method 1: Auto-install NVIDIA Container Runtime
export COMFYUI_NVIDIA_CHOICE=1
./manage.sh --action install --gpu nvidia

# Method 2: Manual runtime installation (Ubuntu/Debian)
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker

# Method 3: Fall back to CPU mode
./manage.sh --action install --gpu cpu
```

### Container Won't Start

**Symptoms**: Container fails to start, ComfyUI web interface not accessible, connection refused

**Diagnosis Steps:**
```bash
# Check service status (recommended)
./manage.sh --action status

# Check container status
docker ps -a | grep comfyui

# Check container logs
docker logs comfyui

# Check port conflicts
netstat -tlnp | grep :8188
```

**Solutions:**
```bash
# Method 1: Restart the service
./manage.sh --action restart

# Method 2: Use custom port if conflict
export COMFYUI_CUSTOM_PORT=5680
./manage.sh --action install --force yes

# Method 3: Check for resource constraints
docker system df
docker system prune  # Free up space if needed

# Method 4: Reinstall with different GPU mode
./manage.sh --action install --gpu cpu --force yes
```

### Model Download Issues

**Symptoms**: Models not downloading, download stuck, insufficient space errors

**Diagnosis Steps:**
```bash
# Check available disk space (recommended)
df -h ~/.comfyui

# Check model directory structure
ls -la ~/.comfyui/models/

# Test internet connectivity
curl -I https://huggingface.co
```

**Solutions:**
```bash
# Method 1: Download models manually
./manage.sh --action download-models

# Method 2: Check space and clean up
docker system prune -a
rm -rf ~/.comfyui/temp/*

# Method 3: Use external model directory
export COMFYUI_MODEL_DIR=/external/models
./manage.sh --action install --force yes

# Method 4: Download specific models manually
cd ~/.comfyui/models/checkpoints
wget https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0/resolve/main/sd_xl_base_1.0.safetensors
```

## Service Issues

### Web Interface Not Accessible

**Symptoms**: Browser shows connection refused, timeout, or blank page

**Diagnosis Steps:**
```bash
# Test service connectivity (recommended)
curl -f http://localhost:8188/system_stats

# Check if ComfyUI is listening
docker exec comfyui netstat -tlnp | grep 8188

# Check AI-Dock portal
curl -f http://localhost:5679/

# Check container networking
docker inspect comfyui | grep -A 10 NetworkSettings
```

**Solutions:**
```bash
# Method 1: Check port mapping
docker port comfyui

# Method 2: Restart with explicit port mapping
./manage.sh --action restart

# Method 3: Check firewall rules
sudo ufw status
sudo ufw allow 8188
sudo ufw allow 5679

# Method 4: Access via AI-Dock portal
# Open http://localhost:5679 and click ComfyUI link
```

### Slow Startup Times

**Symptoms**: Container takes long time to start, ComfyUI loading screen persists

**Diagnosis Steps:**
```bash
# Monitor container startup (recommended)
./manage.sh --action logs --follow

# Check system resources during startup
docker stats comfyui

# Check model loading
ls -la ~/.comfyui/models/checkpoints/
```

**Solutions:**
```bash
# Method 1: Pre-download models
./manage.sh --action download-models

# Method 2: Use smaller models for testing
rm ~/.comfyui/models/checkpoints/large_model.safetensors

# Method 3: Increase system resources
docker update comfyui --memory 16g

# Method 4: Enable low memory mode
export COMFYUI_LOW_MEMORY=true
./manage.sh --action restart
```

## GPU and Memory Issues

### Out of Memory Errors

**Symptoms**: "CUDA out of memory", "RuntimeError: out of memory", workflows fail

**Diagnosis Steps:**
```bash
# Check GPU memory usage (recommended)
./manage.sh --action gpu-info

# Monitor VRAM during workflow execution
docker exec comfyui nvidia-smi -l 2

# Check system memory
free -h
```

**Solutions:**
```bash
# Method 1: Reduce batch size in workflows
# Edit workflow: Change batch_size from 4 to 1

# Method 2: Use smaller models
cd ~/.comfyui/models/checkpoints
# Remove large models, keep only essential ones

# Method 3: Enable low memory mode
export COMFYUI_LOW_MEMORY=true
./manage.sh --action restart

# Method 4: Limit VRAM usage
export COMFYUI_VRAM_LIMIT=8  # Limit to 8GB
./manage.sh --action restart

# Method 5: Lower image resolution
# In workflows: Change width/height from 1024 to 512
```

### GPU Not Detected

**Symptoms**: ComfyUI running on CPU, no GPU listed in system stats, slow generation

**Diagnosis Steps:**
```bash
# Check GPU detection (recommended)
./manage.sh --action gpu-info

# Test NVIDIA runtime
docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi

# Check ComfyUI system stats
curl http://localhost:8188/system_stats | jq '.system.devices'
```

**Solutions:**
```bash
# Method 1: Validate and reinstall NVIDIA runtime
./manage.sh --action validate-nvidia
export COMFYUI_NVIDIA_CHOICE=1
./manage.sh --action install --gpu nvidia --force yes

# Method 2: Check GPU passthrough
docker exec comfyui nvidia-smi

# Method 3: Reinstall with explicit GPU type
./manage.sh --action install --gpu nvidia --force yes

# Method 4: Check for conflicting containers
docker ps | grep nvidia
# Stop any conflicting containers using GPU
```

### Memory Leaks

**Symptoms**: Memory usage continuously increases, system becomes unresponsive

**Diagnosis Steps:**
```bash
# Monitor memory usage over time (recommended)
docker stats comfyui --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Check for memory leaks in logs
./manage.sh --action logs | grep -i "memory\|leak\|oom"

# Monitor system memory
watch -n 2 free -h
```

**Solutions:**
```bash
# Method 1: Restart service periodically
./manage.sh --action restart

# Method 2: Limit container memory
docker update comfyui --memory 16g

# Method 3: Enable garbage collection
export PYTORCH_CUDA_ALLOC_CONF="max_split_size_mb:128"
./manage.sh --action restart

# Method 4: Use CPU offloading
export COMFYUI_CPU_OFFLOAD=true
./manage.sh --action restart
```

## Workflow Issues

### Workflow Execution Failures

**Symptoms**: Workflows fail to execute, node errors, "missing model" errors

**Diagnosis Steps:**
```bash
# Validate workflow requirements (recommended)
./manage.sh --action import-workflow --workflow workflow.json

# Check for missing models
./manage.sh --action list-models

# Test with basic workflow
./manage.sh --action execute-workflow --workflow examples/basic_text_to_image.json

# Check execution logs
curl http://localhost:8188/history | jq
```

**Solutions:**
```bash
# Method 1: Download required models
./manage.sh --action download-models

# Method 2: Check workflow format
cat workflow.json | jq '.' > /dev/null && echo "Valid JSON" || echo "Invalid JSON"

# Method 3: Test with known working workflow
./manage.sh --action execute-workflow --workflow examples/basic_text_to_image.json

# Method 4: Check node connections
# Open workflow in ComfyUI web interface to verify connections
```

### Missing Models

**Symptoms**: "Model not found" errors, workflows fail with model loading errors

**Diagnosis Steps:**
```bash
# List installed models (recommended)
./manage.sh --action list-models

# Check model directory structure  
ls -la ~/.comfyui/models/checkpoints/
ls -la ~/.comfyui/models/vae/

# Check model file integrity
file ~/.comfyui/models/checkpoints/*.safetensors
```

**Solutions:**
```bash
# Method 1: Download default models
./manage.sh --action download-models

# Method 2: Download specific model
cd ~/.comfyui/models/checkpoints
wget https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0/resolve/main/sd_xl_base_1.0.safetensors

# Method 3: Verify model integrity
# Re-download corrupted models
rm ~/.comfyui/models/checkpoints/corrupted_model.safetensors
./manage.sh --action download-models

# Method 4: Use alternative model names
# Edit workflow to use available model names from list-models output
```

### Slow Workflow Execution

**Symptoms**: Workflows take extremely long time, appear stuck, no progress updates

**Diagnosis Steps:**
```bash
# Monitor workflow progress (recommended)
./manage.sh --action status

# Check system resources during execution
docker stats comfyui

# Monitor GPU utilization
docker exec comfyui nvidia-smi -l 2

# Check for CPU vs GPU execution
curl http://localhost:8188/system_stats | jq '.system.devices'
```

**Solutions:**
```bash
# Method 1: Verify GPU is being used
# Check system_stats for GPU devices
curl http://localhost:8188/system_stats

# Method 2: Reduce complexity
# Use lower resolution (512x512 instead of 1024x1024)
# Reduce sampling steps (10-15 instead of 20-30)

# Method 3: Optimize workflow parameters
# CFG scale: 6-8 (not higher)
# Batch size: 1 for testing

# Method 4: Check for blocking operations
# Remove complex nodes for testing
# Use basic text-to-image workflow
```

## API and Integration Issues

### API Connection Issues

**Symptoms**: API calls fail, connection refused, timeout errors

**Diagnosis Steps:**
```bash
# Test API connectivity (recommended)
curl -f http://localhost:8188/system_stats

# Test from different network locations
curl -f http://$(hostname -I | awk '{print $1}'):8188/system_stats

# Check container networking
docker network ls
docker network inspect bridge | grep comfyui
```

**Solutions:**
```bash
# Method 1: Verify service is running
./manage.sh --action status

# Method 2: Test from host IP (container access)
HOST_IP=$(hostname -I | awk '{print $1}')
curl http://$HOST_IP:8188/system_stats

# Method 3: Create shared Docker network
docker network create comfyui-network
docker network connect comfyui-network comfyui
docker network connect comfyui-network your-client-container

# Method 4: Check CORS settings
export COMFYUI_CORS_ORIGINS=*
./manage.sh --action restart
```

### WebSocket Connection Issues

**Symptoms**: No real-time updates, connection drops, WebSocket errors

**Diagnosis Steps:**
```bash
# Test WebSocket connectivity
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: test" http://localhost:8188/ws

# Check browser console for WebSocket errors
# Check network tab in developer tools
```

**Solutions:**
```bash
# Method 1: Check firewall and proxy settings
sudo ufw status
# Ensure no proxy is blocking WebSocket connections

# Method 2: Use polling instead of WebSocket
# Implement status polling in your client code

# Method 3: Verify WebSocket support
# Test with ComfyUI web interface first

# Method 4: Check for network restrictions
# Ensure WebSocket traffic is not blocked
```

### Integration with Other Services

**Symptoms**: n8n workflows fail, external apps can't connect, timeout errors

**Solutions:**
```bash
# Method 1: Use host network access
# In n8n HTTP Request node, use host IP instead of localhost
HOST_IP=$(hostname -I | awk '{print $1}')
# URL: http://$HOST_IP:8188/prompt

# Method 2: Configure shared Docker network
docker network create vrooli-network
docker network connect vrooli-network comfyui
docker network connect vrooli-network n8n
# Use container names: http://comfyui:8188/prompt

# Method 3: Test connectivity between containers
docker exec n8n curl -f http://comfyui:8188/system_stats

# Method 4: Check timeout settings
# Increase timeout values in integration clients
# ComfyUI workflows can take 60+ seconds
```

## Performance Issues

### Poor Image Quality

**Symptoms**: Blurry images, artifacts, poor composition, wrong style

**Solutions:**
```bash
# Method 1: Check model quality
# Use high-quality models like SDXL
./manage.sh --action list-models | grep -i sdxl

# Method 2: Optimize workflow parameters
# CFG Scale: 7-8 for balanced results
# Steps: 20-30 for good quality
# Negative prompts: Include quality-negative terms

# Method 3: Use appropriate VAE
# Ensure SDXL VAE is installed for SDXL models
ls ~/.comfyui/models/vae/

# Method 4: Check sampler settings
# Use euler_ancestral or dpm_2m_karras
# Enable karras scheduler
```

### Resource Usage Issues

**Symptoms**: High CPU usage, excessive memory consumption, system lag

**Diagnosis Steps:**
```bash
# Monitor resource usage (recommended)
docker stats comfyui --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"

# Check disk usage
du -sh ~/.comfyui/
du -sh ~/.comfyui/models/

# Monitor during workflow execution
top -p $(docker inspect comfyui --format '{{.State.Pid}}')
```

**Solutions:**
```bash
# Method 1: Optimize resource allocation
docker update comfyui --memory 16g --cpus 8.0

# Method 2: Enable resource limits
export COMFYUI_VRAM_LIMIT=8
export COMFYUI_CPU_THREADS=6
./manage.sh --action restart

# Method 3: Clean up disk space
docker system prune -a
rm -rf ~/.comfyui/temp/*
rm -rf ~/.comfyui/outputs/old-images/

# Method 4: Optimize model storage
# Keep only essential models
# Use model compression if available
```

## Network and Connectivity Issues

### Port Conflicts

**Symptoms**: "Port already in use", service won't start, connection refused

**Diagnosis Steps:**
```bash
# Check port usage (recommended)
netstat -tlnp | grep :8188
netstat -tlnp | grep :5679

# Find processes using ports
lsof -i :8188
lsof -i :5679
```

**Solutions:**
```bash
# Method 1: Use custom port
export COMFYUI_CUSTOM_PORT=5680
./manage.sh --action install --force yes

# Method 2: Stop conflicting services
sudo systemctl stop service-using-port-8188

# Method 3: Kill processes using ports
sudo kill $(lsof -t -i:8188)
sudo kill $(lsof -t -i:5679)

# Method 4: Use alternative ports
export COMFYUI_WEB_PORT=8189
export COMFYUI_PORTAL_PORT=5680
```

### External Access Issues

**Symptoms**: Can't access from other machines, network timeout

**Solutions:**
```bash
# Method 1: Configure external access
export COMFYUI_LISTEN_HOST=0.0.0.0
./manage.sh --action restart

# Method 2: Configure firewall
sudo ufw allow 8188
sudo ufw allow 5679

# Method 3: Check network configuration
ip route show
ifconfig

# Method 4: Use reverse proxy (recommended for production)
# Set up nginx or Apache reverse proxy
# Configure SSL/TLS for secure access
```

## Recovery Procedures

### Complete Service Recovery

```bash
# Step 1: Stop service
./manage.sh --action stop

# Step 2: Backup important data
tar -czf comfyui-backup-$(date +%Y%m%d).tar.gz ~/.comfyui/workflows/
tar -czf models-backup-$(date +%Y%m%d).tar.gz ~/.comfyui/models/

# Step 3: Clean installation
docker rm -f comfyui
docker system prune -a
./manage.sh --action install --force yes

# Step 4: Restore workflows
tar -xzf comfyui-backup-*.tar.gz -C ~/

# Step 5: Download models
./manage.sh --action download-models

# Step 6: Test functionality
./manage.sh --action status
curl -f http://localhost:8188/system_stats
```

### Model Recovery

```bash
# Verify model integrity
find ~/.comfyui/models -name "*.safetensors" -exec file {} \;

# Re-download corrupted models
rm ~/.comfyui/models/checkpoints/corrupted_model.safetensors
./manage.sh --action download-models

# Restore from backup
tar -xzf models-backup-*.tar.gz -C ~/.comfyui/
```

### Configuration Recovery

```bash
# Reset to default configuration
rm -f ~/.comfyui/config/*
./manage.sh --action restart

# Restore from backup configuration
# (if you have backed up custom settings)
```

## Prevention and Maintenance

### Regular Maintenance Tasks

```bash
# Daily health check
./manage.sh --action status

# Weekly cleanup
docker system prune
rm -rf ~/.comfyui/temp/*

# Monthly model management
./manage.sh --action list-models
# Remove unused models to free space

# Quarterly backup
tar -czf comfyui-full-backup-$(date +%Y%m%d).tar.gz ~/.comfyui/
```

### Monitoring Setup

```bash
# Set up health monitoring
crontab -e
# Add: */5 * * * * curl -f http://localhost:8188/system_stats || echo "ComfyUI unhealthy" | mail -s "Alert" admin@example.com

# Monitor disk space
df -h ~/.comfyui/ | awk 'NR==2{print $5}' | sed 's/%//' > /tmp/comfyui_disk_usage
# Alert if > 90%
```

## Getting Help

### Diagnostic Information Collection

Before seeking help, collect this information:

```bash
# System information
./manage.sh --action status > diagnostics.txt
./manage.sh --action gpu-info >> diagnostics.txt
./manage.sh --action list-models >> diagnostics.txt
./manage.sh --action logs >> diagnostics.txt

# Include:
# - Docker version and GPU setup
# - Container status and resource usage  
# - Model list and workflow files (anonymized)
# - Error messages with timestamps
# - Hardware specifications (CPU, RAM, GPU)
```

### Support Resources

1. **Check documentation**:
   - [Configuration Guide](CONFIGURATION.md)
   - [API Reference](API.md)
   - [ComfyUI Official Docs](https://docs.comfy.org/)

2. **Community support**:
   - ComfyUI GitHub: https://github.com/comfyanonymous/ComfyUI
   - ComfyUI Reddit: https://reddit.com/r/comfyui/
   - Docker Image: https://hub.docker.com/r/zhangp365/comfyui

3. **Create detailed bug reports**:
   - Include diagnostic information
   - Provide reproduction steps
   - Attach relevant workflow files (anonymized)
   - Specify hardware configuration

This troubleshooting guide covers the most common ComfyUI issues. For specific problems not covered here, use the diagnostic commands to gather information and consult the ComfyUI community resources.