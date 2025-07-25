# ComfyUI Resource for Vrooli

ComfyUI is a powerful and modular AI image generation workflow platform with a node-based interface. This resource integrates ComfyUI into Vrooli's local resource management system, providing AI-powered image generation capabilities through both a web UI and API.

## Features

- 🎨 **Node-based Workflow Editor**: Visual workflow creation with drag-and-drop interface
- 🤖 **AI Model Support**: Compatible with SD, SDXL, and custom models
- 🚀 **GPU Acceleration**: Automatic detection and configuration for NVIDIA/AMD GPUs
- 📡 **API Integration**: Execute workflows programmatically via REST API
- 🔄 **Workflow Automation**: Integrate with n8n and other automation tools
- 💾 **Model Management**: Download and organize AI models
- 🐳 **Docker-based**: Isolated, reproducible environment

## Requirements

- Docker installed and running
- At least 16GB RAM (32GB recommended)
- 50GB+ free disk space for models
- GPU recommended (NVIDIA/AMD) but CPU mode available

## Installation

### Basic Installation

```bash
# Install ComfyUI with auto GPU detection
./scripts/resources/automation/comfyui/manage.sh --action install

# Install with specific GPU type
./scripts/resources/automation/comfyui/manage.sh --action install --gpu nvidia
./scripts/resources/automation/comfyui/manage.sh --action install --gpu amd
./scripts/resources/automation/comfyui/manage.sh --action install --gpu cpu
```

### Via Vrooli Setup

```bash
# Include ComfyUI during Vrooli setup
./scripts/main/setup.sh --target native-linux --resources comfyui

# Install all automation resources
./scripts/main/setup.sh --target native-linux --resources automation-only
```

## Usage

### Service Management

```bash
# Start ComfyUI
./scripts/resources/automation/comfyui/manage.sh --action start

# Stop ComfyUI
./scripts/resources/automation/comfyui/manage.sh --action stop

# Restart ComfyUI
./scripts/resources/automation/comfyui/manage.sh --action restart

# Check status
./scripts/resources/automation/comfyui/manage.sh --action status

# View logs
./scripts/resources/automation/comfyui/manage.sh --action logs
```

### Model Management

```bash
# Download default models (SDXL base + VAE)
./scripts/resources/automation/comfyui/manage.sh --action download-models

# List installed models
./scripts/resources/automation/comfyui/manage.sh --action list-models

# Check GPU information
./scripts/resources/automation/comfyui/manage.sh --action gpu-info
```

### Workflow Management

```bash
# Import a workflow
./scripts/resources/automation/comfyui/manage.sh --action import-workflow --workflow my-workflow.json

# Execute a workflow
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow --workflow workflow.json

# Execute with custom output directory
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow \
  --workflow workflow.json \
  --output /path/to/output
```

## Directory Structure

ComfyUI data is stored in `~/.comfyui/`:

```
~/.comfyui/
├── models/
│   ├── checkpoints/    # Main model files (SD, SDXL, etc.)
│   ├── vae/           # VAE models
│   ├── loras/         # LoRA models
│   ├── controlnet/    # ControlNet models
│   └── ...           # Other model types
├── custom_nodes/      # Custom ComfyUI nodes
├── outputs/          # Generated images
├── workflows/        # Saved workflows
└── input/           # Input images for workflows
```

## API Integration

ComfyUI provides a REST API for workflow automation:

### Endpoints

- `POST http://localhost:5679/prompt` - Queue a workflow for execution
- `GET http://localhost:5679/history/{prompt_id}` - Get execution results
- `GET http://localhost:5679/view` - Retrieve generated images
- `POST http://localhost:5679/upload/image` - Upload input images
- `GET http://localhost:5679/system_stats` - System information
- `WebSocket ws://localhost:5679/ws` - Real-time updates

### Example: Execute Workflow via API

```bash
# Submit workflow
curl -X POST http://localhost:5679/prompt \
  -H "Content-Type: application/json" \
  -d @workflow.json

# Check system stats
curl http://localhost:5679/system_stats
```

### Example: Python Integration

```python
import requests
import json

# Load workflow
with open('workflow.json', 'r') as f:
    workflow = json.load(f)

# Submit to ComfyUI
payload = {
    "client_id": "python-client",
    "prompt": workflow
}

response = requests.post('http://localhost:5679/prompt', json=payload)
prompt_id = response.json()['prompt_id']

# Check status
history = requests.get(f'http://localhost:5679/history/{prompt_id}').json()
```

## Integration with Vrooli

ComfyUI is automatically configured in Vrooli's resource system:

```json
{
  "services": {
    "automation": {
      "comfyui": {
        "enabled": true,
        "baseUrl": "http://localhost:5679",
        "features": {
          "workflows": true,
          "api": true,
          "gpu": true
        }
      }
    }
  }
}
```

### Integration with n8n

Create automated image generation workflows in n8n:

1. Use HTTP Request node to submit workflows to ComfyUI
2. Monitor execution with webhook or polling
3. Process generated images with other n8n nodes

## GPU Configuration

### NVIDIA GPU

Requirements:
- NVIDIA drivers installed
- Docker NVIDIA Container Toolkit

The script automatically detects NVIDIA GPUs and configures Docker accordingly.

### AMD GPU

Requirements:
- ROCm drivers installed
- Docker configured for AMD GPU passthrough

Note: AMD GPU support may require additional configuration depending on your system.

### CPU Mode

If no GPU is detected or you explicitly choose CPU mode:
- Generation will be significantly slower
- Memory usage will be higher
- Some models may not work efficiently

## Troubleshooting

### Common Issues

1. **Port conflicts**
   ```bash
   # Use custom port
   export COMFYUI_CUSTOM_PORT=5680
   ./scripts/resources/automation/comfyui/manage.sh --action install
   ```

2. **Out of memory**
   - Reduce batch size in workflows
   - Use smaller models
   - Enable CPU offloading in ComfyUI settings

3. **Missing models**
   ```bash
   # Check which models are needed
   ./scripts/resources/automation/comfyui/manage.sh --action import-workflow --workflow workflow.json
   
   # Download default models
   ./scripts/resources/automation/comfyui/manage.sh --action download-models
   ```

4. **Docker permission issues**
   ```bash
   # Add user to docker group
   sudo usermod -aG docker $USER
   # Log out and back in
   ```

### Logs and Debugging

```bash
# View container logs
docker logs comfyui

# Check container status
docker ps -a | grep comfyui

# Access container shell
docker exec -it comfyui /bin/bash
```

## Model Resources

### Recommended Models

1. **SDXL Base 1.0** - High-quality general purpose model
2. **SDXL VAE** - Improved color accuracy
3. **ControlNet Models** - For pose/depth/edge control
4. **LoRA Models** - Style and concept modifications

### Model Sources

- [HuggingFace](https://huggingface.co/models?pipeline_tag=text-to-image)
- [CivitAI](https://civitai.com/) (community models)
- [Official Stability AI Models](https://huggingface.co/stabilityai)

## Advanced Configuration

### Custom Docker Image

To use a custom ComfyUI image:

```bash
export COMFYUI_CUSTOM_IMAGE=my-comfyui:latest
./scripts/resources/automation/comfyui/manage.sh --action install
```

### Environment Variables

- `COMFYUI_CUSTOM_PORT` - Override default port (5679)
- `COMFYUI_GPU_TYPE` - Force GPU type (auto/nvidia/amd/cpu)
- `COMFYUI_VRAM_LIMIT` - Limit VRAM usage in GB
- `COMFYUI_CUSTOM_IMAGE` - Use custom Docker image

## Security Considerations

- ComfyUI runs in an isolated Docker container
- By default, only accessible on localhost
- No authentication enabled (add reverse proxy for external access)
- Be cautious with custom nodes from untrusted sources

## Uninstallation

```bash
# Remove ComfyUI (preserves models by default)
./scripts/resources/automation/comfyui/manage.sh --action uninstall

# During uninstall, you'll be prompted to keep or remove data
```

## Support and Resources

- [ComfyUI GitHub](https://github.com/comfyanonymous/ComfyUI)
- [ComfyUI Documentation](https://docs.comfy.org/)
- [ComfyUI Examples](https://comfyanonymous.github.io/ComfyUI_examples/)
- [Vrooli Documentation](https://github.com/Vrooli/Vrooli)

## Contributing

To contribute to the ComfyUI resource integration:

1. Follow Vrooli's contribution guidelines
2. Test changes with different GPU configurations
3. Update documentation for new features
4. Submit PR with clear description

## License

This integration follows Vrooli's license. ComfyUI itself is licensed under GPL-3.0.