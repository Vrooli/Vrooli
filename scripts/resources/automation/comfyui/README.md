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

## Interface Overview

### ComfyUI Workflow Editor (Port 8188)
The main ComfyUI interface provides a node-based workflow editor where you can:
- Drag and drop nodes to create image generation workflows
- Connect nodes to define the data flow
- Queue workflows for execution
- View generated images

### AI-Dock Service Portal (Port 5679)
The container management interface shows:
- Running services and their ports
- Direct links to each service
- Container information and logs
- Additional services like Jupyter Notebook (8888) and Syncthing (8384)

## API Integration

ComfyUI provides a REST API for workflow automation on port 8188:

### Port Configuration

- **Port 8188**: ComfyUI Web Interface and API (main interface)
- **Port 5679**: AI-Dock Service Portal (container management)

### API Endpoints

- `POST http://localhost:8188/prompt` - Queue a workflow for execution
- `GET http://localhost:8188/history/{prompt_id}` - Get execution results
- `GET http://localhost:8188/view` - Retrieve generated images
- `POST http://localhost:8188/upload/image` - Upload input images
- `GET http://localhost:8188/system_stats` - System information
- `WebSocket ws://localhost:8188/ws` - Real-time updates

### Example: Execute Workflow via API

```bash
# Submit workflow
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d @workflow.json

# Check system stats
curl http://localhost:8188/system_stats
```

### API Response Examples

#### System Stats Response
```json
{
  "system": {
    "os": "posix",
    "comfyui_version": "v0.2.2",
    "python_version": "3.10.12",
    "pytorch_version": "2.4.1+cu121",
    "embedded_python": false,
    "argv": ["main.py", "--disable-auto-launch", "--port", "8188"],
    "devices": [{
      "name": "cuda:0 NVIDIA GeForce RTX 4070 Ti SUPER",
      "type": "cuda",
      "index": 0,
      "vram_total": 16844397184,
      "vram_free": 8491021282,
      "torch_vram_total": 6979321856,
      "torch_vram_free": 31896546
    }]
  }
}
```

#### Prompt Submission Response
```json
{
  "prompt_id": "89f0e3f5-ae42-4c93-b4b7-0b3e52a0f8e5",
  "number": 1,
  "node_errors": {}
}
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

response = requests.post('http://localhost:8188/prompt', json=payload)
prompt_id = response.json()['prompt_id']

# Check status
history = requests.get(f'http://localhost:8188/history/{prompt_id}').json()
```

### WebSocket Connection Example

```javascript
const ws = new WebSocket('ws://localhost:8188/ws');

ws.onopen = () => {
  console.log('Connected to ComfyUI WebSocket');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case 'status':
      console.log('Queue status:', message.data.status);
      break;
    case 'progress':
      console.log(`Progress: ${message.data.value}/${message.data.max}`);
      break;
    case 'executing':
      console.log('Executing node:', message.data.node);
      break;
    case 'executed':
      console.log('Completed node:', message.data.node);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

### API Access from Other Containers

If accessing ComfyUI API from another Docker container:
- Use the host machine's IP address instead of localhost
- Example: `http://192.168.1.x:8188` (replace with your host IP)
- For Docker networks, ensure containers can communicate
- Alternative: Use Docker's host network mode or create a shared network

```bash
# Get your host IP
hostname -I | awk '{print $1}'

# Example API call from another container
curl http://192.168.1.173:8188/system_stats
```

## Additional Services in AI-Dock Container

The AI-Dock image includes additional services beyond ComfyUI:

- **ComfyUI (Port 8188)**: Main image generation interface and API
- **Jupyter Notebook (Port 8888)**: Interactive Python notebooks for experimentation
- **Syncthing (Port 8384)**: File synchronization for models and outputs
- **Service Portal (Port 1111/5679)**: Internal service management and monitoring

These services are managed through the AI-Dock portal accessible on port 5679.

## Integration with Vrooli

ComfyUI is automatically configured in Vrooli's resource system:

```json
{
  "services": {
    "automation": {
      "comfyui": {
        "enabled": true,
        "baseUrl": "http://localhost:8188",
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

#### Example n8n Workflow
```json
{
  "nodes": [
    {
      "name": "Submit to ComfyUI",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "POST",
        "url": "http://localhost:8188/prompt",
        "jsonParameters": true,
        "options": {},
        "bodyParametersJson": "{\"client_id\": \"n8n-workflow\", \"prompt\": {...}}"
      }
    }
  ]
}
```

## GPU Configuration

### NVIDIA GPU

Requirements:
- NVIDIA drivers installed
- NVIDIA Container Runtime (automatically installed if missing)

The script automatically detects NVIDIA GPUs and handles Container Runtime setup:

**Automatic Setup:**
- Detects if NVIDIA Container Runtime is missing
- Offers automatic installation for Ubuntu/Debian, CentOS/RHEL/Fedora, and Arch Linux
- Configures Docker daemon automatically
- Validates installation with runtime tests

**Manual Setup Option:**
If automatic installation isn't suitable, the script provides comprehensive manual installation instructions for your specific operating system.

**Fallback to CPU Mode:**
If NVIDIA setup fails or is declined, ComfyUI can automatically fall back to CPU mode.

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
   - Lower resolution for initial tests (512x512 instead of 1024x1024)

3. **Missing models**
   ```bash
   # Check which models are needed
   ./scripts/resources/automation/comfyui/manage.sh --action import-workflow --workflow workflow.json
   
   # Download default models
   ./scripts/resources/automation/comfyui/manage.sh --action download-models
   
   # Check model integrity
   ./scripts/resources/automation/comfyui/manage.sh --action status
   ```

4. **Docker permission issues**
   ```bash
   # Add user to docker group
   sudo usermod -aG docker $USER
   # Log out and back in
   ```

5. **API Connection Issues**
   ```bash
   # Test from host
   curl http://localhost:8188/system_stats
   
   # Test from another container (use host IP)
   curl http://$(hostname -I | awk '{print $1}'):8188/system_stats
   
   # Check if ComfyUI is listening
   docker exec comfyui netstat -tlnp | grep 8188
   ```

6. **Workflow Execution Errors**
   - Check `/history` endpoint for detailed error messages
   - Verify all required models are installed
   - Ensure node connections are valid
   - Check GPU memory availability

### NVIDIA Runtime Validation

```bash
# Validate NVIDIA setup manually
./scripts/resources/automation/comfyui/manage.sh --action validate-nvidia

# Test NVIDIA runtime directly
docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi
```

### Logs and Debugging

```bash
# View container logs
docker logs comfyui

# Check container status
docker ps -a | grep comfyui

# Access container shell
docker exec -it comfyui /bin/bash

# Validate NVIDIA runtime setup
./scripts/resources/automation/comfyui/manage.sh --action validate-nvidia
```

## Workflow Examples

### Basic Text-to-Image Workflow
```json
{
  "3": {
    "inputs": {"ckpt_name": "sd_xl_base_1.0.safetensors"},
    "class_type": "CheckpointLoaderSimple"
  },
  "5": {
    "inputs": {
      "text": "a beautiful sunset over mountains, photorealistic",
      "clip": ["3", 1]
    },
    "class_type": "CLIPTextEncode"
  },
  "6": {
    "inputs": {
      "text": "blurry, low quality",
      "clip": ["3", 1]
    },
    "class_type": "CLIPTextEncode"
  },
  "7": {
    "inputs": {
      "seed": 42,
      "steps": 20,
      "cfg": 7.5,
      "sampler_name": "euler_ancestral",
      "scheduler": "karras",
      "denoise": 1,
      "model": ["3", 0],
      "positive": ["5", 0],
      "negative": ["6", 0],
      "latent_image": ["10", 0]
    },
    "class_type": "KSampler"
  },
  "8": {
    "inputs": {
      "samples": ["7", 0],
      "vae": ["3", 2]
    },
    "class_type": "VAEDecode"
  },
  "9": {
    "inputs": {
      "filename_prefix": "output",
      "images": ["8", 0]
    },
    "class_type": "SaveImage"
  },
  "10": {
    "inputs": {
      "width": 1024,
      "height": 1024,
      "batch_size": 1
    },
    "class_type": "EmptyLatentImage"
  }
}
```

### Example Workflows

ComfyUI includes several example workflows in the `examples/` directory:

#### Basic Text-to-Image Example
```bash
# Simple text-to-image generation
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow \
  --workflow ./scripts/resources/automation/comfyui/examples/basic_text_to_image.json
```

#### Multi-Panel Comic Example
```bash
# Generate 4 individual comic panels with automatic composition
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "comic-example",
    "prompt": '$(cat ./scripts/resources/automation/comfyui/examples/pirate_rabbit_comic_composite_v2.json)'
  }'

# Automatically combine panels into a comic page
python3 ./scripts/resources/automation/comfyui/examples/composite_comic_panels.py
```

These examples demonstrate:
- Basic and advanced workflow patterns
- Character consistency across multiple panels
- Story progression and narrative techniques
- Multi-stage refinement and upscaling
- Automatic image composition
- Professional comic book styling

**View all examples:** See `./scripts/resources/automation/comfyui/examples/README.md` for detailed documentation.

### Workflow Best Practices
- Start with lower resolutions (512x512) for testing
- Use appropriate step counts (20-30 for most use cases)
- Set CFG scale between 6-8 for balanced results
- Always include negative prompts for better quality
- Save intermediate results for debugging
- Use consistent character descriptions across panels
- Vary seeds for panel diversity while maintaining coherence

## Model Resources

### Recommended Models

1. **SDXL Base 1.0** - High-quality general purpose model (6.5GB)
2. **SDXL VAE** - Improved color accuracy (320MB)
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

- `COMFYUI_CUSTOM_PORT` - Override AI-Dock portal port (default: 5679, maps to ComfyUI on 8188)
- `COMFYUI_GPU_TYPE` - Force GPU type (auto/nvidia/amd/cpu)
- `COMFYUI_VRAM_LIMIT` - Limit VRAM usage in GB
- `COMFYUI_CUSTOM_IMAGE` - Use custom Docker image
- `COMFYUI_NVIDIA_CHOICE` - Non-interactive NVIDIA runtime choice (1-4)
  - 1: Auto-install NVIDIA Container Runtime
  - 2: Show manual installation instructions  
  - 3: Continue with CPU mode instead
  - 4: Cancel installation

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