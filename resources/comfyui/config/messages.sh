#!/usr/bin/env bash
# ComfyUI User Messages and Help Text
# Contains all user-facing messages, guides, and documentation

#######################################
# Show comprehensive help documentation
#######################################
messages::show_extended_help() {
    cat << 'EOF'

ComfyUI Resource Management - Extended Help

OVERVIEW
--------
ComfyUI is a powerful node-based AI image generation platform that integrates
with Vrooli's automation system. It supports various Stable Diffusion models
and provides both a visual workflow editor and API access.

INSTALLATION
------------
Basic installation:
  ./manage.sh --action install

With specific GPU:
  ./manage.sh --action install --gpu nvidia
  ./manage.sh --action install --gpu amd
  ./manage.sh --action install --gpu cpu

Skip model downloads:
  ./manage.sh --action install --skip-models yes

WORKFLOW MANAGEMENT
------------------
Import a workflow:
  ./manage.sh --action import-workflow --workflow my-workflow.json

Execute a workflow:
  ./manage.sh --action execute-workflow --workflow workflow.json

Execute with custom output:
  ./manage.sh --action execute-workflow \
    --workflow workflow.json \
    --output /path/to/outputs

MODEL MANAGEMENT
---------------
Download default models (SDXL):
  ./manage.sh --action download-models

Download specific models:
  ./manage.sh --action download-models \
    --models "https://url/to/model1.safetensors,https://url/to/model2.safetensors"

List installed models:
  ./manage.sh --action list-models

GPU TROUBLESHOOTING
------------------
Check GPU status:
  ./manage.sh --action gpu-info

Validate NVIDIA setup:
  ./manage.sh --action validate-nvidia

For NVIDIA GPU issues:
1. Ensure drivers are visible: vrooli host inventory --field has_nvidia_gpu
2. Install Container Runtime if missing
3. Restart Docker after configuration
4. Test with: vrooli host inventory --field has_docker_addressable_nvidia_gpu

API USAGE
---------
The ComfyUI API is available at http://localhost:8188 (or custom port).

Submit workflow:
  curl -X POST http://localhost:8188/prompt \
    -H "Content-Type: application/json" \
    -d @workflow.json

Check status:
  curl http://localhost:8188/history/{prompt_id}

Python example:
  import requests
  
  with open('workflow.json') as f:
      workflow = json.load(f)
  
  response = requests.post('http://localhost:8188/prompt', 
                          json={'prompt': workflow, 'client_id': 'my-client'})
  prompt_id = response.json()['prompt_id']

INTEGRATION WITH VROOLI
----------------------
ComfyUI automatically registers with Vrooli's resource system. Use it in:
- n8n workflows via HTTP nodes
- Automation routines
- Agent-based image generation

The resource configuration is stored in ~/.vrooli/service.json

ENVIRONMENT VARIABLES
--------------------
COMFYUI_CUSTOM_PORT      - Override default port (8188)
COMFYUI_GPU_TYPE         - Force GPU type: auto|nvidia|amd|cpu
COMFYUI_CUSTOM_IMAGE     - Use custom Docker image
COMFYUI_VRAM_LIMIT       - Limit VRAM usage in GB
COMFYUI_NVIDIA_CHOICE    - Non-interactive NVIDIA choice (1-4)

COMMON ISSUES
------------
1. Port conflicts:
   Set COMFYUI_CUSTOM_PORT=5680 before installation

2. Out of memory:
   - Reduce batch size in workflows
   - Use smaller models
   - Set COMFYUI_VRAM_LIMIT environment variable

3. Missing models:
   Download required models to appropriate directories

4. Container won't start:
   Check logs: ./manage.sh --action logs
   Verify GPU setup: ./manage.sh --action gpu-info

UNINSTALLATION
-------------
Remove ComfyUI (keeps data):
  ./manage.sh --action uninstall

Force remove with all data:
  ./manage.sh --action uninstall --force yes

For manual cleanup instructions:
  ./manage.sh --action cleanup-help

SUPPORT
-------
- ComfyUI Documentation: https://docs.comfy.org/
- ComfyUI GitHub: https://github.com/comfyanonymous/ComfyUI
- Vrooli Documentation: https://github.com/Vrooli/Vrooli

EOF
}

#######################################
# Show quick start guide
#######################################
messages::show_quickstart() {
    cat << 'EOF'

🚀 ComfyUI Quick Start Guide

1. Install ComfyUI:
   ./manage.sh --action install

2. Access the Web UI:
   http://localhost:8188

3. Download models (if not done during install):
   ./manage.sh --action download-models

4. Import a workflow:
   ./manage.sh --action import-workflow --workflow example.json

5. Execute workflow via API:
   ./manage.sh --action execute-workflow --workflow example.json

For more help: ./manage.sh --help

EOF
}

#######################################
# Show model download sources
#######################################
messages::show_model_sources() {
    cat << 'EOF'

📚 Model Download Sources

CHECKPOINT MODELS
----------------
Stable Diffusion XL:
  https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0

Stable Diffusion 1.5:
  https://huggingface.co/runwayml/stable-diffusion-v1-5

Custom Models:
  https://civitai.com/models

VAE MODELS
----------
SDXL VAE:
  https://huggingface.co/stabilityai/sdxl-vae

SD 1.5 VAE:
  https://huggingface.co/stabilityai/sd-vae-ft-mse

LORA MODELS
-----------
Browse LoRAs:
  https://civitai.com/models?types=LORA

CONTROLNET MODELS
----------------
ControlNet v1.1:
  https://huggingface.co/lllyasviel/ControlNet-v1-1

Place downloaded models in:
  ${COMFYUI_MODELS_DIR:-${XDG_DATA_HOME:-~/.local/share}/vrooli/resources/comfyui/models}/<type>/

Model types:
  - checkpoints/  → Main SD models
  - vae/         → VAE models
  - loras/       → LoRA models
  - controlnet/  → ControlNet models

EOF
}

#######################################
# Show workflow examples
#######################################
messages::show_workflow_examples() {
    cat << 'EOF'

🎨 ComfyUI Workflow Examples

BASIC TEXT-TO-IMAGE
------------------
{
  "1": {
    "class_type": "CheckpointLoaderSimple",
    "inputs": {
      "ckpt_name": "sd_xl_base_1.0.safetensors"
    }
  },
  "2": {
    "class_type": "CLIPTextEncode",
    "inputs": {
      "text": "a beautiful landscape",
      "clip": ["1", 1]
    }
  },
  "3": {
    "class_type": "KSampler",
    "inputs": {
      "model": ["1", 0],
      "positive": ["2", 0],
      "latent_image": ["4", 0],
      "steps": 20
    }
  },
  "4": {
    "class_type": "EmptyLatentImage",
    "inputs": {
      "width": 1024,
      "height": 1024
    }
  },
  "5": {
    "class_type": "VAEDecode",
    "inputs": {
      "samples": ["3", 0],
      "vae": ["1", 2]
    }
  },
  "6": {
    "class_type": "SaveImage",
    "inputs": {
      "images": ["5", 0]
    }
  }
}

Save this as workflow.json and execute:
  ./manage.sh --action execute-workflow --workflow workflow.json

WORKFLOW RESOURCES
-----------------
- Example workflows: https://comfyanonymous.github.io/ComfyUI_examples/
- Community workflows: https://openart.ai/workflows/home
- Workflow sharing: https://github.com/comfyanonymous/ComfyUI_examples

EOF
}

#######################################
# Show API integration examples
#######################################
messages::show_api_examples() {
    cat << 'EOF'

🔌 ComfyUI API Integration Examples

PYTHON INTEGRATION
-----------------
import requests
import json
import time

# Load workflow
with open('workflow.json', 'r') as f:
    workflow = json.load(f)

# Submit to ComfyUI
api_url = "http://localhost:8188"
payload = {
    "prompt": workflow,
    "client_id": "python-client"
}

response = requests.post(f"{api_url}/prompt", json=payload)
prompt_id = response.json()['prompt_id']

# Monitor execution
while True:
    history = requests.get(f"{api_url}/history/{prompt_id}").json()
    if prompt_id in history:
        if history[prompt_id]['status']['status_str'] == 'success':
            print("Workflow completed!")
            break
    time.sleep(2)

# Get outputs
outputs = history[prompt_id]['outputs']

NODE.JS INTEGRATION
------------------
const axios = require('axios');
const fs = require('fs');

async function executeWorkflow() {
    const workflow = JSON.parse(fs.readFileSync('workflow.json'));
    
    const response = await axios.post('http://localhost:8188/prompt', {
        prompt: workflow,
        client_id: 'nodejs-client'
    });
    
    const promptId = response.data.prompt_id;
    console.log(`Workflow submitted: ${promptId}`);
    
    // Poll for completion
    let completed = false;
    while (!completed) {
        const history = await axios.get(`http://localhost:8188/history/${promptId}`);
        if (history.data[promptId]?.status?.status_str === 'success') {
            completed = true;
            console.log('Workflow completed!');
        }
        await new Promise(resolve => setTimeout(resolve, 2000));
    }
}

N8N WORKFLOW
-----------
1. HTTP Request Node:
   - Method: POST
   - URL: http://localhost:8188/prompt
   - Body: { "prompt": {workflow}, "client_id": "n8n" }

2. Wait Node:
   - 2 seconds

3. HTTP Request Node:
   - Method: GET  
   - URL: http://localhost:8188/history/{{prompt_id}}

4. IF Node:
   - Check if status is "success"

WEBSOCKET MONITORING
-------------------
const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:8188/ws');

ws.on('message', (data) => {
    const message = JSON.parse(data);
    console.log('Progress:', message);
});

EOF
}

#######################################
# Message Variables for Status/Logging
#######################################

# Installation messages
export MSG_COMFYUI_INSTALLING="🎨 Installing ComfyUI..."
export MSG_COMFYUI_ALREADY_INSTALLED="✅ ComfyUI is already installed"
export MSG_COMFYUI_NOT_FOUND="❌ ComfyUI not found or not running"
export MSG_COMFYUI_NOT_RUNNING="⏹️  ComfyUI is not running"

# Status messages
export MSG_STATUS_CHECKING="🔍 Checking ComfyUI status..."
export MSG_STATUS_CONTAINER_OK="✅ Container is healthy"
export MSG_STATUS_CONTAINER_RUNNING="🟢 Container is running"
export MSG_STATUS_CONTAINER_STOPPED="⏹️  Container is stopped"
export MSG_STATUS_CONTAINER_MISSING="❌ Container not found"

# Model messages
export MSG_MODEL_DOWNLOADING="⬇️  Downloading model..."
export MSG_MODEL_INSTALLED="✅ Model installed successfully"
export MSG_MODEL_FAILED="❌ Model download failed"

# Workflow messages
export MSG_WORKFLOW_EXECUTING="🔄 Executing workflow..."
export MSG_WORKFLOW_COMPLETE="✅ Workflow completed successfully"
export MSG_WORKFLOW_COMPLETED="✅ Workflow completed successfully"
export MSG_WORKFLOW_FAILED="❌ Workflow execution failed"

# GPU messages
export MSG_GPU_DETECTED="🎮 GPU detected"
export MSG_GPU_NOT_DETECTED="⚠️  No GPU detected, using CPU"
export MSG_GPU_CUDA_OK="✅ CUDA is available"
export MSG_GPU_ROCM_OK="✅ ROCm is available"

# Docker messages
export MSG_DOCKER_STARTING="🚀 Starting ComfyUI container..."
export MSG_DOCKER_STOPPING="⏹️  Stopping ComfyUI container..."
export MSG_DOCKER_REMOVING="🗑️  Removing ComfyUI container..."

# Processing messages
export MSG_PROCESSING_COMPLETE="✅ Processing completed"
export MSG_PROCESSING_FAILED="❌ Processing failed"
export MSG_PROCESSING_QUEUE="📋 Added to processing queue"

# Export function for consistency
messages::export_messages() {
    export MSG_COMFYUI_INSTALLING MSG_COMFYUI_ALREADY_INSTALLED MSG_COMFYUI_NOT_FOUND MSG_COMFYUI_NOT_RUNNING
    export MSG_STATUS_CHECKING MSG_STATUS_CONTAINER_OK MSG_STATUS_CONTAINER_RUNNING MSG_STATUS_CONTAINER_STOPPED MSG_STATUS_CONTAINER_MISSING
    export MSG_MODEL_DOWNLOADING MSG_MODEL_INSTALLED MSG_MODEL_FAILED
    export MSG_WORKFLOW_EXECUTING MSG_WORKFLOW_COMPLETE MSG_WORKFLOW_COMPLETED MSG_WORKFLOW_FAILED
    export MSG_GPU_DETECTED MSG_GPU_NOT_DETECTED MSG_GPU_CUDA_OK MSG_GPU_ROCM_OK
    export MSG_DOCKER_STARTING MSG_DOCKER_STOPPING MSG_DOCKER_REMOVING
    export MSG_PROCESSING_COMPLETE MSG_PROCESSING_FAILED MSG_PROCESSING_QUEUE
}

# Auto-export when sourced
messages::export_messages
