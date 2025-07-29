# ComfyUI Configuration Guide

This guide covers all configuration options for ComfyUI, including installation parameters, GPU setup, model management, and advanced customization.

## Installation Configuration

### Basic Installation Options

```bash
# Install with auto GPU detection (recommended)
./manage.sh --action install

# Install with specific GPU type
./manage.sh --action install --gpu nvidia
./manage.sh --action install --gpu amd  
./manage.sh --action install --gpu cpu

# Force reinstall if already exists
./manage.sh --action install --force yes
```

### Installation Parameters

| Parameter | Description | Default | Options |
|-----------|-------------|---------|---------|
| `--gpu` | GPU type to use | `auto` | `auto`, `nvidia`, `amd`, `cpu` |
| `--force` | Force reinstall | `no` | `yes`, `no` |

### Complete Installation Example

```bash
./manage.sh --action install --gpu nvidia --force yes
```

## GPU Configuration

### NVIDIA GPU Setup

**Requirements:**
- NVIDIA drivers installed
- NVIDIA Container Runtime (automatically installed if missing)

**Automatic Setup Process:**
The script automatically detects NVIDIA GPUs and handles Container Runtime setup:

1. **Detection**: Checks for NVIDIA GPU presence with `nvidia-smi`
2. **Runtime Check**: Verifies NVIDIA Container Runtime installation
3. **Auto-Installation**: Offers automatic installation for supported operating systems:
   - Ubuntu/Debian
   - CentOS/RHEL/Fedora
   - Arch Linux
4. **Docker Configuration**: Automatically configures Docker daemon
5. **Runtime Validation**: Tests installation with runtime verification

**Installation Options:**
```bash
# Non-interactive NVIDIA setup (environment variable)
export COMFYUI_NVIDIA_CHOICE=1  # 1: Auto-install, 2: Manual, 3: CPU mode, 4: Cancel
./manage.sh --action install --gpu nvidia
```

**Choice Options:**
- `1`: Auto-install NVIDIA Container Runtime
- `2`: Show manual installation instructions
- `3`: Continue with CPU mode instead  
- `4`: Cancel installation

**Manual Validation:**
```bash
# Validate NVIDIA setup manually
./manage.sh --action validate-nvidia

# Test NVIDIA runtime directly
docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi
```

### AMD GPU Setup

**Requirements:**
- ROCm drivers installed
- Docker configured for AMD GPU passthrough

```bash
# Install with AMD GPU support
./manage.sh --action install --gpu amd
```

**Note**: AMD GPU support may require additional system-specific configuration.

### CPU Mode Configuration

```bash
# Force CPU mode (no GPU acceleration)
./manage.sh --action install --gpu cpu
```

**Considerations:**
- Generation will be significantly slower
- Memory usage will be higher
- Some models may not work efficiently
- Recommended for testing or low-resource environments

## Environment Variables

### Core Configuration

```bash
# Port Configuration
export COMFYUI_CUSTOM_PORT=5680              # Override AI-Dock portal port (default: 5679)

# GPU Configuration  
export COMFYUI_GPU_TYPE=nvidia               # Force GPU type: auto/nvidia/amd/cpu
export COMFYUI_VRAM_LIMIT=8                  # Limit VRAM usage in GB

# Docker Configuration
export COMFYUI_CUSTOM_IMAGE=my-comfyui:latest # Use custom Docker image

# NVIDIA Configuration
export COMFYUI_NVIDIA_CHOICE=1               # Non-interactive NVIDIA runtime choice
```

### Advanced Configuration

```bash
# Performance Settings
export COMFYUI_LOW_MEMORY=true               # Enable low memory mode
export COMFYUI_CPU_THREADS=8                # CPU thread count for processing

# Network Settings
export COMFYUI_LISTEN_HOST=0.0.0.0          # Listen on all interfaces
export COMFYUI_CORS_ORIGINS=*               # CORS configuration

# Storage Settings
export COMFYUI_OUTPUT_DIR=/custom/output     # Custom output directory
export COMFYUI_MODEL_DIR=/custom/models     # Custom model directory
```

## Docker Configuration

### AI-Dock Image Features

ComfyUI uses the AI-Dock image which includes:

```dockerfile
# Based on ghcr.io/ai-dock/comfyui
# Includes:
# - ComfyUI (Port 8188)
# - Jupyter Notebook (Port 8888)  
# - Syncthing (Port 8384)
# - Service Portal (Port 1111/5679)
```

### Volume Mounts

The ComfyUI container uses several volume mounts:

```bash
# Data persistence
-v ~/.comfyui:/data                          # Main data directory

# Model storage
-v ~/.comfyui/models:/data/models            # AI models
-v ~/.comfyui/outputs:/data/outputs          # Generated images
-v ~/.comfyui/workflows:/data/workflows      # Saved workflows

# Optional custom mounts
-v /custom/models:/data/models               # Custom model directory
-v /custom/output:/data/outputs              # Custom output directory
```

### Port Mapping

```bash
# Default port mapping
-p 5679:1111                                 # AI-Dock service portal
-p 8188:8188                                 # ComfyUI web interface and API
-p 8888:8888                                 # Jupyter Notebook (optional)
-p 8384:8384                                 # Syncthing (optional)

# Custom port configuration
export COMFYUI_CUSTOM_PORT=5680
# Maps to: -p 5680:1111
```

### Resource Limits

```bash
# Memory and CPU limits
docker update comfyui --memory 16g --cpus 8.0

# GPU memory limit (via environment)
export COMFYUI_VRAM_LIMIT=12                 # Limit to 12GB VRAM

# Restart policy
docker update comfyui --restart unless-stopped
```

## Model Management

### Default Model Configuration

```bash
# Download default models (SDXL base + VAE)
./manage.sh --action download-models
```

**Default Models Installed:**
- **SDXL Base 1.0** (6.5GB) - High-quality general purpose model
- **SDXL VAE** (320MB) - Improved color accuracy for SDXL

### Custom Model Installation

**Model Directory Structure:**
```
~/.comfyui/models/
├── checkpoints/          # Main model files (SD, SDXL, etc.)
│   ├── sd_xl_base_1.0.safetensors
│   └── custom_model.safetensors
├── vae/                 # VAE models
│   └── sdxl_vae.safetensors
├── loras/              # LoRA models
├── controlnet/         # ControlNet models
├── clip_vision/        # CLIP vision models
├── diffusers/          # Diffusers format models
├── embeddings/         # Text embeddings
├── hypernetworks/      # Hypernetwork models
├── style_models/       # Style transfer models
└── upscale_models/     # Upscaling models
```

**Manual Model Installation:**
```bash
# Download models to appropriate directories
cd ~/.comfyui/models/checkpoints
wget https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0/resolve/main/sd_xl_base_1.0.safetensors

# Verify model installation
./manage.sh --action list-models
```

### Model Sources and Recommendations

**Recommended Model Sources:**
- [HuggingFace Models](https://huggingface.co/models?pipeline_tag=text-to-image)
- [CivitAI](https://civitai.com/) (community models)
- [Official Stability AI Models](https://huggingface.co/stabilityai)

**Popular Models:**
```bash
# SDXL Models (Recommended)
sd_xl_base_1.0.safetensors          # Base SDXL model (6.5GB)
sd_xl_refiner_1.0.safetensors       # SDXL refiner model (6.2GB)

# Stable Diffusion 1.5 Models
v1-5-pruned-emaonly.safetensors     # Classic SD1.5 (4.3GB)

# VAE Models
sdxl_vae.safetensors                # SDXL VAE (320MB)
vae-ft-mse-840000-ema-pruned.safetensors # SD1.5 VAE (320MB)
```

## Directory Structure Configuration

### Default Directory Layout

```
~/.comfyui/                          # Main data directory
├── models/                         # All AI models
│   ├── checkpoints/               # Main model files
│   ├── vae/                      # VAE models
│   ├── loras/                    # LoRA models
│   ├── controlnet/               # ControlNet models
│   └── [other model types]/     # Additional model categories
├── custom_nodes/                  # Custom ComfyUI nodes
├── outputs/                      # Generated images
├── workflows/                    # Saved workflows
├── input/                       # Input images for workflows
└── temp/                        # Temporary files
```

### Custom Directory Configuration

```bash
# Use custom base directory
export COMFYUI_DATA_DIR=/custom/comfyui
./manage.sh --action install

# Use custom model directory
export COMFYUI_MODEL_DIR=/shared/models
./manage.sh --action install

# Use custom output directory
export COMFYUI_OUTPUT_DIR=/shared/outputs
./manage.sh --action install
```

## Network Configuration

### Service Ports

**Primary Services:**
- **Port 8188**: ComfyUI Web Interface and API
- **Port 5679**: AI-Dock Service Portal (container management)

**Additional Services (AI-Dock):**
- **Port 8888**: Jupyter Notebook (interactive development)
- **Port 8384**: Syncthing (file synchronization)

### External Access Configuration

```bash
# Listen on all interfaces (for external access)
export COMFYUI_LISTEN_HOST=0.0.0.0
./manage.sh --action install

# Configure firewall (if needed)
sudo ufw allow 8188
sudo ufw allow 5679
```

### API Access from Other Containers

**From Host Machine:**
```bash
curl http://localhost:8188/system_stats
```

**From Another Container:**
```bash
# Use host machine's IP address
HOST_IP=$(hostname -I | awk '{print $1}')
curl http://$HOST_IP:8188/system_stats

# Example with specific IP
curl http://192.168.1.173:8188/system_stats
```

**Docker Network Setup:**
```bash
# Create shared network
docker network create comfyui-network

# Connect containers
docker network connect comfyui-network comfyui
docker network connect comfyui-network your-app

# Use container name for communication
curl http://comfyui:8188/system_stats
```

## Performance Configuration

### Memory Management

```bash
# Enable low memory mode
export COMFYUI_LOW_MEMORY=true
./manage.sh --action install

# Set VRAM limits
export COMFYUI_VRAM_LIMIT=8           # Limit to 8GB VRAM
./manage.sh --action install

# CPU memory optimization
export COMFYUI_CPU_MEMORY_LIMIT=16    # Limit CPU RAM to 16GB
```

### CPU Configuration

```bash
# Set CPU thread count
export COMFYUI_CPU_THREADS=8
./manage.sh --action install

# Enable CPU optimizations
export COMFYUI_CPU_OPTIMIZATION=true
```

### GPU Memory Optimization

```bash
# Enable GPU memory management
export COMFYUI_GPU_MEMORY_FRACTION=0.8  # Use 80% of available VRAM
export COMFYUI_ENABLE_GPU_MEMORY_GROWTH=true

# NVIDIA-specific optimizations
export CUDA_MEMORY_FRACTION=0.8
export PYTORCH_CUDA_ALLOC_CONF="max_split_size_mb:512"
```

## Workflow Configuration

### Default Workflow Settings

```bash
# Set default workflow parameters
export COMFYUI_DEFAULT_STEPS=20         # Default sampling steps
export COMFYUI_DEFAULT_CFG=7.5          # Default CFG scale
export COMFYUI_DEFAULT_SAMPLER=euler_ancestral
export COMFYUI_DEFAULT_SCHEDULER=karras
export COMFYUI_DEFAULT_WIDTH=1024       # Default image width
export COMFYUI_DEFAULT_HEIGHT=1024      # Default image height
```

### Output Configuration

```bash
# Image output settings
export COMFYUI_OUTPUT_FORMAT=png        # Output format: png, jpg, webp
export COMFYUI_OUTPUT_QUALITY=95        # JPEG quality (1-100)
export COMFYUI_SAVE_METADATA=true       # Include generation metadata

# Filename patterns
export COMFYUI_FILENAME_PREFIX=comfyui  # Default filename prefix
export COMFYUI_INCLUDE_TIMESTAMP=true   # Include timestamp in filenames
```

## Security Configuration

### Access Control

```bash
# Enable basic authentication (if supported)
export COMFYUI_AUTH_ENABLED=true
export COMFYUI_AUTH_USERNAME=admin
export COMFYUI_AUTH_PASSWORD=secure_password

# API key authentication (if supported)
export COMFYUI_API_KEY=your-secure-api-key
```

### Network Security

```bash
# Restrict access to localhost only
export COMFYUI_LISTEN_HOST=127.0.0.1

# CORS configuration
export COMFYUI_CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# Disable external services (if not needed)
export COMFYUI_DISABLE_JUPYTER=true
export COMFYUI_DISABLE_SYNCTHING=true
```

## Integration Configuration

### Vrooli Resource Integration

ComfyUI is automatically configured in `~/.vrooli/resources.local.json`:

```json
{
  "services": {
    "automation": {
      "comfyui": {
        "enabled": true,
        "baseUrl": "http://localhost:8188",
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 10000,
          "endpoint": "/system_stats"
        },
        "features": {
          "workflows": true,
          "api": true,
          "gpu": true,
          "models": ["sdxl", "sd15"],
          "formats": ["png", "jpg", "webp"]
        },
        "limits": {
          "maxResolution": "1024x1024",
          "maxBatchSize": 4,
          "timeoutMs": 300000
        }
      }
    }
  }
}
```

### External Integration

```bash
# Webhook configuration (if supported)
export COMFYUI_WEBHOOK_URL=https://your-app.com/comfyui-webhook
export COMFYUI_WEBHOOK_SECRET=your-webhook-secret

# Queue configuration
export COMFYUI_MAX_QUEUE_SIZE=10        # Maximum queued workflows
export COMFYUI_QUEUE_TIMEOUT=300        # Queue timeout in seconds
```

## Custom Node Configuration

### Installing Custom Nodes

```bash
# Install custom nodes manually
cd ~/.comfyui/custom_nodes
git clone https://github.com/author/custom-node-repo

# Restart ComfyUI to load new nodes
./manage.sh --action restart
```

### Custom Node Directory

```
~/.comfyui/custom_nodes/
├── ComfyUI-Custom-Scripts/
├── ComfyUI-Manager/
├── ComfyUI-Advanced-ControlNet/
└── your-custom-nodes/
```

## Backup and Recovery Configuration

### Automated Backup Setup

```bash
# Enable automatic workflow backup
export COMFYUI_AUTO_BACKUP=true
export COMFYUI_BACKUP_INTERVAL=daily
export COMFYUI_BACKUP_RETENTION=7       # Keep 7 days of backups

# Backup directory
export COMFYUI_BACKUP_DIR=~/.comfyui/backups
```

### Manual Backup Commands

```bash
# Backup workflows
./manage.sh --action export-workflows --output workflows-backup.json

# Backup complete configuration
tar -czf comfyui-backup-$(date +%Y%m%d).tar.gz ~/.comfyui/

# Backup only models (separate due to size)
tar -czf models-backup-$(date +%Y%m%d).tar.gz ~/.comfyui/models/
```

## Monitoring Configuration

### Health Check Setup

```bash
# Enable health monitoring
export COMFYUI_HEALTH_CHECK=true
export COMFYUI_HEALTH_CHECK_INTERVAL=60  # Check every 60 seconds

# Health check endpoints
export COMFYUI_HEALTH_CHECK_URL=http://localhost:8188/system_stats
```

### Logging Configuration

```bash
# Log level configuration
export COMFYUI_LOG_LEVEL=INFO           # DEBUG, INFO, WARNING, ERROR

# Log file configuration
export COMFYUI_LOG_FILE=~/.comfyui/logs/comfyui.log
export COMFYUI_LOG_MAX_SIZE=100M        # Max log file size
export COMFYUI_LOG_RETENTION=7          # Keep 7 days of logs
```

## Troubleshooting Configuration

### Debug Mode

```bash
# Enable debug mode
export COMFYUI_DEBUG=true
export COMFYUI_VERBOSE=true

# Enable performance profiling
export COMFYUI_PROFILE=true
export COMFYUI_PROFILE_OUTPUT=~/.comfyui/profiles/
```

### Resource Monitoring

```bash
# Enable resource monitoring
export COMFYUI_MONITOR_RESOURCES=true
export COMFYUI_RESOURCE_LOG=~/.comfyui/resource-usage.log

# Memory monitoring
export COMFYUI_MEMORY_WARNING_THRESHOLD=80    # Warn at 80% memory usage
export COMFYUI_MEMORY_KILL_THRESHOLD=95       # Kill at 95% memory usage
```

This configuration guide provides comprehensive coverage of all ComfyUI configuration options. For specific use cases, refer to the [API documentation](API.md) and [troubleshooting guide](TROUBLESHOOTING.md).