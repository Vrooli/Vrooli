# Ollama Installation Guide

This guide covers the installation and setup of Ollama for local LLM inference.

## Prerequisites

- **Hardware Requirements**:
  - Minimum: 8GB RAM, 4GB free disk space
  - Recommended: 16GB+ RAM, 50GB+ disk space
  - GPU: NVIDIA with 8GB+ VRAM (optional but recommended)

- **Software Requirements**:
  - Docker and Docker Compose
  - NVIDIA Container Toolkit (for GPU support)

## Installation Methods

### 1. Quick Install (Default Models)

Install Ollama with recommended models:

```bash
./manage.sh --action install
```

This will:
- Install Ollama server
- Download default models: llama3.1:8b, deepseek-r1:8b, qwen2.5-coder:7b
- Verify GPU availability (if present)
- Start the service on port 11434

### 2. Custom Installation

#### Install Specific Models

```bash
# Install with custom model set
./manage.sh --action install --models "llama3.1:8b,gemma2:9b"

# Install without any models (manual selection later)
./manage.sh --action install --skip-models
```

#### Install with Custom Port

```bash
# Use custom port
OLLAMA_CUSTOM_PORT=9999 ./manage.sh --action install
```

#### GPU Configuration

```bash
# Force CPU-only mode
./manage.sh --action install --force-cpu

# Verify GPU support
./manage.sh --action gpu-info
```

## Model Installation

### Automatic Model Installation

During installation, these default models are downloaded:

- **llama3.1:8b** - General purpose model (4.7GB)
- **deepseek-r1:8b** - Reasoning and math (4.9GB)  
- **qwen2.5-coder:7b** - Code generation (4.2GB)

Total download: ~14GB

### Manual Model Installation

After installation, add more models:

```bash
# Install single model
./manage.sh --action pull --models "gemma2:9b"

# Install multiple models
./manage.sh --action pull --models "llama3.1:70b,claude3:haiku"

# List available models in catalog
./manage.sh --action models
```

## Post-Installation Verification

### 1. Check Service Status

```bash
./manage.sh --action status
```

Expected output:
```
[INFO] Ollama Status:
✅ Service: Running on port 11434
✅ API: Responding (200 OK)
✅ Models: 3 models installed
✅ GPU: NVIDIA RTX 4090 detected
✅ Memory: 15.2GB available
```

### 2. Test Model Inference

```bash
# Quick test
./manage.sh --action prompt --text "What is 2+2?" --model "llama3.1:8b"

# Test each type
./manage.sh --action prompt --text "Hello world" --type general
./manage.sh --action prompt --text "def fibonacci(n):" --type code
./manage.sh --action prompt --text "Solve: x^2 + 5x + 6 = 0" --type reasoning
```

### 3. Verify GPU Usage (if available)

```bash
# Check GPU utilization during inference
nvidia-smi

# View Ollama GPU status
./manage.sh --action gpu-info
```

## Configuration

### Environment Variables

```bash
# Server configuration
export OLLAMA_CUSTOM_PORT=11434              # API port
export OLLAMA_HOST=0.0.0.0                   # Bind address
export OLLAMA_ORIGINS="*"                    # CORS origins

# Performance tuning
export OLLAMA_NUM_PARALLEL=4                 # Concurrent requests
export OLLAMA_MAX_LOADED_MODELS=3            # Models in memory
export OLLAMA_FLASH_ATTENTION=1              # Enable flash attention

# GPU configuration  
export OLLAMA_GPU_LAYERS=35                  # GPU layers (auto-detected)
export CUDA_VISIBLE_DEVICES=0                # Specific GPU
```

### Model Storage

```bash
# Custom model storage location
export OLLAMA_MODELS=/opt/ollama/models

# Create directory with proper permissions
sudo mkdir -p /opt/ollama/models
sudo chown -R $(id -u):$(id -g) /opt/ollama/models
```

## Troubleshooting Installation

### Common Issues

#### Port Already in Use
```bash
# Check what's using port 11434
sudo lsof -i :11434

# Use different port
OLLAMA_CUSTOM_PORT=11435 ./manage.sh --action install
```

#### GPU Not Detected
```bash
# Check NVIDIA drivers
nvidia-smi

# Verify Docker GPU support
docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi

# Install NVIDIA Container Toolkit
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
```

#### Insufficient Memory
```bash
# Check available memory
free -h

# Reduce default models
./manage.sh --action install --models "llama3.1:8b"  # Just one model

# Use smaller models
./manage.sh --action install --models "llama3.2:3b,qwen2.5:3b"
```

#### Download Failures
```bash
# Check network connectivity
curl -I https://ollama.ai

# Resume interrupted download
./manage.sh --action pull --models "llama3.1:8b" --force

# Clear corrupted downloads
./manage.sh --action cleanup
```

### Verification Commands

```bash
# Test API directly
curl http://localhost:11434/api/tags

# Check container logs
docker logs ollama

# Monitor resource usage
./manage.sh --action monitor
```

## Next Steps

- [Configure Ollama](CONFIGURATION.md) for your use case
- [Explore the model catalog](MODELS.md) for other options
- [Learn the API](API.md) for integration
- [Optimize performance](PERFORMANCE.md) for your hardware