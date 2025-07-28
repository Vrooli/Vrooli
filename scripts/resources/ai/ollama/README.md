# Ollama Resource

Ollama is a local LLM inference engine that allows you to run large language models on your own hardware. This resource provides automated installation, configuration, and management of Ollama for the Vrooli project.

## Overview

- **Category**: AI
- **Default Port**: 11434
- **Service Type**: Systemd Service
- **Container**: No (native installation)
- **Documentation**: [ollama.com](https://ollama.com)

## Features

- 🚀 **Local LLM Inference**: Run models entirely on your hardware
- 📦 **Multiple Model Support**: Easy model management and switching
- 🔄 **Streaming Responses**: Real-time text generation
- 🌐 **OpenAI-Compatible API**: Drop-in replacement for many applications
- ⚡ **GPU Acceleration**: Automatic NVIDIA/AMD GPU detection and usage
- 🔧 **Model Quantization**: Efficient model compression options
- 🎨 **Custom Model Creation**: Build your own models with Modelfiles
- 👁️ **Multi-modal Support**: Vision models for image understanding

## Prerequisites

- Linux system (Ubuntu/Debian recommended)
- 8GB+ RAM (16GB+ recommended for larger models)
- 10GB+ free disk space
- sudo privileges for installation
- (Optional) NVIDIA GPU with CUDA support for acceleration

## Installation

### Quick Install
```bash
# Install with default models (llama3.1:8b, deepseek-r1:8b, qwen2.5-coder:7b)
./manage.sh --action install

# Install without models
./manage.sh --action install --skip-models

# Install with specific models
./manage.sh --action install --models "llama3.1:8b,phi-4:14b"
```

### Installation Process
1. Downloads and installs Ollama binary
2. Creates system user and service
3. Configures systemd service
4. Starts Ollama service
5. Pulls default models (unless skipped)
6. Updates Vrooli configuration

## Usage

### Basic Commands
```bash
# Check status
./manage.sh --action status

# Start service
./manage.sh --action start

# Stop service
./manage.sh --action stop

# Restart service
./manage.sh --action restart

# View logs
journalctl -u ollama -f
```

### Model Management
```bash
# List installed models
./manage.sh --action models

# Show available models from catalog
./manage.sh --action available

# Pull a new model
ollama pull llama3.3:8b

# Remove a model
ollama rm llama3.3:8b

# Run interactive chat
ollama run llama3.1:8b
```

## Model Catalog

### Default Models (2025 Recommendations)

| Model | Size | Use Case | Description |
|-------|------|----------|-------------|
| **llama3.1:8b** | 4.9GB | General Purpose | Latest general-purpose model from Meta |
| **deepseek-r1:8b** | 4.7GB | Advanced Reasoning | Breakthrough model with explicit thinking process |
| **qwen2.5-coder:7b** | 4.1GB | Code Generation | Superior code model, replaces CodeLlama |

### Alternative Models

#### General Purpose
- `llama3.3:8b` (4.9GB) - Very latest from Meta (Dec 2024)
- `phi-4:14b` (8.2GB) - Microsoft's efficient multilingual model
- `qwen2.5:14b` (8.0GB) - Strong multilingual with excellent reasoning
- `mistral-small:22b` (13.2GB) - Excellent balanced performance

#### Reasoning & Math
- `deepseek-r1:14b` (8.1GB) - Larger reasoning model for complex problems
- `deepseek-r1:1.5b` (0.9GB) - Smallest reasoning model for resource-constrained environments

#### Code & Programming
- `qwen2.5-coder:32b` (19.1GB) - Large code model for complex projects
- `deepseek-coder:6.7b` (3.8GB) - Specialized programming model

#### Vision/Multimodal
- `llava:13b` (7.3GB) - Image understanding and visual reasoning
- `qwen2-vl:7b` (4.2GB) - Vision-language model for image analysis

## API Reference

### Base Endpoints
- Health Check: `http://localhost:11434/api/tags`
- List Models: `http://localhost:11434/api/tags`
- Chat: `http://localhost:11434/api/chat`
- Generate: `http://localhost:11434/api/generate`
- Pull Model: `http://localhost:11434/api/pull`
- Show Model: `http://localhost:11434/api/show`

### Example API Usage

#### Smart Prompt with Type-Based Model Selection (Recommended)
```bash
# Use default general model type
./manage.sh --action prompt --text "What is the capital of France?"

# Explicitly specify model type for specialized tasks
./manage.sh --action prompt --text "Write a Python function" --type code
./manage.sh --action prompt --text "Solve this equation" --type reasoning  
./manage.sh --action prompt --text "Describe this image" --type vision

# Override with specific model if needed
./manage.sh --action prompt --text "Hello world" --model qwen2.5-coder:7b
```

#### Direct API Usage (Advanced)
```bash
# Get list of available models first to avoid errors
curl -s http://localhost:11434/api/tags | jq -r '.models[].name'

# Generate text with a known available model
curl -X POST http://localhost:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Why is the sky blue?",
    "stream": false
  }'
```

#### Chat Completion (OpenAI-compatible)
```bash
curl -X POST http://localhost:11434/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1:8b",
    "messages": [
      {"role": "user", "content": "Hello! How are you?"}
    ]
  }'
```

#### Stream Response
```bash
curl -X POST http://localhost:11434/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1:8b",
    "prompt": "Write a haiku about programming",
    "stream": true
  }'
```

## Configuration

### Environment Variables
- `OLLAMA_CUSTOM_PORT`: Override default port (default: 11434)
- `OLLAMA_HOST`: Set host binding (default: 0.0.0.0)
- `OLLAMA_MODELS`: Model storage location (default: ~/.ollama/models)
- `OLLAMA_NUM_PARALLEL`: Concurrent request handling (default: auto)
- `OLLAMA_MAX_LOADED_MODELS`: Models kept in memory (default: 1)
- `OLLAMA_KEEP_ALIVE`: Model idle timeout (default: 5m)

### Vrooli Integration
Ollama is automatically configured in `~/.vrooli/resources.local.json`:
```json
{
  "services": {
    "ai": {
      "ollama": {
        "enabled": true,
        "baseUrl": "http://localhost:11434",
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 5000
        },
        "models": {
          "defaultModel": "llama3.1:8b",
          "supportsFunctionCalling": true
        },
        "api": {
          "version": "v1",
          "modelsEndpoint": "/api/tags",
          "chatEndpoint": "/api/chat",
          "generateEndpoint": "/api/generate"
        }
      }
    }
  }
}
```

## Smart Model Selection

### Explicit Type-Based Model Selection

The Ollama resource includes intelligent model selection using explicit model types, giving you control over which specialized model to use:

**Available Model Types:**
- **`--type general`**: General-purpose models like `llama3.1:8b` (default)
- **`--type code`**: Code-specialized models like `qwen2.5-coder:7b`
- **`--type reasoning`**: Advanced reasoning models like `deepseek-r1:8b`
- **`--type vision`**: Multimodal/vision models like `llava:13b`

**Fallback Strategy:**
- If the ideal model for a type isn't available, selects the next best option
- Gracefully handles missing models without API errors
- Always validates model availability before making requests

**Example Usage:**
```bash
# Explicit control over model selection
./manage.sh --action prompt --text "Debug this Python error" --type code
./manage.sh --action prompt --text "Calculate 15% of 200" --type reasoning
./manage.sh --action prompt --text "What's in this image?" --type vision
./manage.sh --action prompt --text "Tell me about quantum physics"  # defaults to general

# Override with specific model when needed
./manage.sh --action prompt --text "Any task" --model llama2:7b
```

**Benefits:**
- **Explicit control**: You decide which type of model to use
- **Predictable behavior**: No magic auto-detection, just clear type-to-model mapping
- **Optimal performance**: Use specialized models when you need them
- **Error prevention**: Eliminates API failures from non-existent models

## Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check service status
systemctl status ollama

# View detailed logs
journalctl -u ollama -n 50

# Check port availability
sudo lsof -i :11434
```

#### GPU Not Detected
```bash
# Check NVIDIA GPU
nvidia-smi

# Check Ollama GPU detection
ollama run llama3.1:8b --verbose
```

#### Model Download Fails
```bash
# Check disk space
df -h ~/.ollama

# Check network connectivity
curl -I https://ollama.com

# Retry with verbose output
ollama pull llama3.1:8b --verbose
```

### Performance Tuning

#### CPU-Only Systems
```bash
# Reduce context size for faster inference
OLLAMA_NUM_CTX=2048 ollama run llama3.1:8b

# Use smaller models
ollama pull deepseek-r1:1.5b
```

#### GPU Systems
```bash
# Check GPU memory usage
nvidia-smi

# Adjust GPU layers
OLLAMA_NUM_GPU=999 ollama run llama3.1:8b  # Use all layers on GPU
```

## Advanced Usage

### Custom Models
Create a Modelfile:
```dockerfile
FROM llama3.1:8b
PARAMETER temperature 0.7
PARAMETER top_p 0.9
SYSTEM "You are a helpful coding assistant."
```

Build and run:
```bash
ollama create my-assistant -f Modelfile
ollama run my-assistant
```

### Multi-Model Deployment
```bash
# Load multiple models
ollama pull llama3.1:8b
ollama pull deepseek-r1:8b
ollama pull qwen2.5-coder:7b

# Models are loaded on-demand and unloaded based on OLLAMA_KEEP_ALIVE
```

## Maintenance

### Regular Tasks
- Monitor disk usage: `du -sh ~/.ollama/models`
- Check for model updates: `ollama list`
- Review service logs: `journalctl -u ollama --since "1 week ago"`
- Clean unused models: `ollama rm <model-name>`

### Backup
```bash
# Backup models (large!)
tar -czf ollama-models-backup.tar.gz ~/.ollama/models

# Backup configuration
cp ~/.vrooli/resources.local.json ollama-config-backup.json
```

### Uninstall
```bash
# Complete removal
./manage.sh --action uninstall

# This will:
# - Stop and disable service
# - Remove binary
# - Optionally remove user
# - Remove from Vrooli config
# Note: Model data in ~/.ollama is preserved
```

## Security Considerations

- Ollama binds to all interfaces (0.0.0.0) by default
- No authentication built-in (rely on firewall/network security)
- Models are stored unencrypted in ~/.ollama
- Consider using reverse proxy with authentication for production

## Resource Requirements

### Minimum Requirements by Model Size
- **Small models (1-3B)**: 4GB RAM, 5GB disk
- **Medium models (7-8B)**: 8GB RAM, 10GB disk
- **Large models (13-14B)**: 16GB RAM, 20GB disk
- **Extra large models (30B+)**: 32GB+ RAM, 40GB+ disk

### GPU Recommendations
- **NVIDIA**: GTX 1660+ (6GB+ VRAM) for 7B models
- **AMD**: RX 6600+ with ROCm support
- **Apple Silicon**: M1/M2 with 16GB+ unified memory

## Related Resources

- [Ollama Documentation](https://github.com/jmorganca/ollama/blob/main/docs/README.md)
- [Model Library](https://ollama.com/library)
- [API Documentation](https://github.com/jmorganca/ollama/blob/main/docs/api.md)
- [Modelfile Reference](https://github.com/jmorganca/ollama/blob/main/docs/modelfile.md)