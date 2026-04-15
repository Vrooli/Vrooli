# ComfyUI - AI Image Generation Workflows

ComfyUI is a powerful, node-based AI image generation workflow platform that enables visual creation of complex image generation pipelines. This resource integrates ComfyUI into Vrooli's local resource management system with enhanced GPU support and automation capabilities.

## 🎯 Quick Reference

- **Category**: Automation (AI Image Generation)
- **Ports**: 8188 (ComfyUI Web UI & API), 8889 (Jupyter Notebook)
- **Container**: comfyui
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 16GB+ RAM (32GB recommended)
- 50GB+ free disk space for models
- GPU recommended (NVIDIA/AMD) but CPU mode available

### Installation
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

### Basic Usage
```bash
# Check service status with comprehensive information
./manage.sh --action status

# Get GPU information and capabilities
./manage.sh --action gpu-info

# Download default models (SDXL base + VAE)
./manage.sh --action download-models

# List installed models
./manage.sh --action list-models

# View service logs
./manage.sh --action logs
```

### Verify Installation
```bash
# Check service health and functionality
./manage.sh --action status

# Test API connectivity
curl -f http://localhost:8188/

# Access interfaces:
# ComfyUI Web UI: http://localhost:8188
# Jupyter Notebook: http://localhost:8889
```

## 🔧 Core Features

- **🎨 Node-based Workflow Editor**: Visual workflow creation with drag-and-drop interface
- **🤖 AI Model Support**: Compatible with SDXL, SD 1.5, and custom models
- **🚀 GPU Acceleration**: Automatic detection and configuration for NVIDIA/AMD GPUs
- **📡 API Integration**: Execute workflows programmatically via REST API and WebSocket
- **🔄 Workflow Automation**: Integrate with n8n and other automation tools
- **💾 Model Management**: Download and organize AI models automatically
- **🐳 Docker-based**: Isolated, reproducible environment

## 📖 Documentation

- **[API Reference](docs/API.md)** - REST API, WebSocket, and workflow management
- **[Configuration Guide](docs/CONFIGURATION.md)** - GPU setup, models, and advanced options
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues, diagnostics, and solutions

## 🎯 When to Use ComfyUI

### Use ComfyUI When:
- Creating AI-generated images and artwork
- Building complex multi-stage image generation workflows
- Need fine-grained control over generation parameters
- Developing automated image processing pipelines
- Experimenting with different AI models and techniques
- Creating batch image generation systems

### Consider Alternatives When:
- Need general business workflow automation → [n8n](../n8n/)
- Want real-time system monitoring → [Node-RED](../node-red/)
- Building simple REST APIs → [Node-RED](../node-red/)
- Require text-based AI interactions → [Ollama](../../ai/ollama/)

## 🔗 Integration Examples

### Workflow Management
```bash
# Import a workflow from file (recommended)
./manage.sh --action import-workflow --workflow my-workflow.json

# Execute a workflow
./manage.sh --action execute-workflow --workflow workflow.json

# Execute with custom output directory
./manage.sh --action execute-workflow --workflow workflow.json --output /path/to/output

# Test with included examples
./manage.sh --action execute-workflow --workflow examples/basic_text_to_image.json
```

### Model Management
```bash
# Download default models (SDXL base + VAE)
./manage.sh --action download-models

# List all installed models
./manage.sh --action list-models

# Check model status and integrity
./manage.sh --action status
```

### API Integration
```bash
# Test API connectivity
curl -f http://localhost:8188/

# Submit workflow via ComfyUI API (port 8188 required for API)
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d @workflow.json

# Get execution history
curl http://localhost:8188/history
```

**Note**: ComfyUI runs on port 8188 for both web UI and API access. Jupyter Notebook runs separately on port 8889.

### With Other Vrooli Resources
```javascript
// n8n HTTP Request node to submit ComfyUI workflow
{
  "method": "POST",
  "url": "http://localhost:8188/prompt",
  "headers": {"Content-Type": "application/json"},
  "body": {
    "client_id": "n8n-workflow",
    "prompt": workflowObject
  }
}

// Node-RED function to check ComfyUI status
msg.url = "http://comfyui:8188/system_stats";
return msg;
```

## ⚡ Key Architecture

### Container Architecture
ComfyUI runs in a vanilla container with optional Jupyter support:

```
Vanilla Setup → Clean Architecture
├── ComfyUI (Port 8188) - Web UI & API for image generation
└── Jupyter (Port 8889) - Optional notebook for custom development
```

### GPU Support Matrix
| GPU Type | Auto-Detection | Container Runtime | Performance |
|----------|----------------|-------------------|-------------|
| **NVIDIA** | ✅ Automatic | Auto-installs NVIDIA Container Runtime | Excellent |
| **AMD** | ✅ Automatic | Manual ROCm setup required | Good |
| **CPU** | ✅ Fallback | Always available | Slow |

### Model Storage Structure
```bash
${COMFYUI_MODELS_DIR:-${XDG_DATA_HOME:-~/.local/share}/vrooli/resources/comfyui/models}/
├── checkpoints/     # Main model files (SDXL, SD 1.5)
├── vae/            # VAE models for improved quality
├── loras/          # LoRA fine-tuning models
├── controlnet/     # ControlNet guidance models
└── [other types]/  # Additional specialized models
```

## 🆘 Getting Help

- Check [Troubleshooting Guide](docs/TROUBLESHOOTING.md) for common issues
- Run `./manage.sh --action status` for detailed diagnostics
- View logs: `./manage.sh --action logs`
- Test GPU setup: `./manage.sh --action gpu-info`

## 📦 What's Included

```
comfyui/
├── manage.sh                    # Management script with GPU auto-detection
├── README.md                    # This overview
├── docs/                        # Detailed documentation
│   ├── API.md                  # Complete API reference
│   ├── CONFIGURATION.md        # Setup and configuration
│   └── TROUBLESHOOTING.md      # Issue resolution
├── lib/                        # Helper scripts and functions
├── config/                     # Configuration and defaults
├── examples/                   # Pre-built workflow examples
│   ├── README.md               # Example documentation
│   ├── basic_text_to_image.json
│   ├── pirate_rabbit_comic_composite_v2.json
│   └── composite_comic_panels.py
└── test/                       # Automated tests
```

## 🔧 Advanced Features

### NVIDIA Container Runtime Auto-Setup
ComfyUI automatically handles NVIDIA GPU setup:
- Detects NVIDIA GPUs with `nvidia-smi`
- Auto-installs NVIDIA Container Runtime for supported OS
- Validates setup with runtime tests
- Falls back to CPU mode if setup fails

### Included Workflow Examples
- **Basic Text-to-Image**: Simple SDXL image generation
- **Comic Creation**: Multi-panel comic generation with character consistency
- **Batch Processing**: Automated generation of multiple images

### Programming Integration
```python
# Python example
import requests
payload = {
    "client_id": "python-client", 
    "prompt": workflow_dict
}
response = requests.post('http://localhost:8188/prompt', json=payload)
```

```javascript
// WebSocket real-time updates
const ws = new WebSocket('ws://localhost:8188/ws');
ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    if (message.type === 'progress') {
        console.log(`Progress: ${message.data.value}/${message.data.max}`);
    }
};
```

---

**🎨 ComfyUI excels at AI image generation workflows, making it perfect for creative automation, batch image processing, and building sophisticated visual content generation pipelines with full programmatic control.**
