# ComfyUI API Reference

ComfyUI provides comprehensive REST API access and WebSocket capabilities for AI image generation workflow automation. This guide covers the manage.sh integration and direct API usage.

> **💡 Recommended Usage**: Use the `manage.sh` script for most operations. This guide shows direct API usage for advanced integration scenarios.

## Base Access

- **Web Interface**: `http://localhost:8188` (ComfyUI Editor)
- **Service Portal**: `http://localhost:5679` (AI-Dock Management)
- **Recommended Management**: `./manage.sh --action [action]`
- **REST API Base**: `http://localhost:8188/`
- **WebSocket**: `ws://localhost:8188/ws`

## Management API (via manage.sh)

### Service Management

**Recommended Method:**
```bash
# Check service status with comprehensive information
./manage.sh --action status

# Start/stop/restart service
./manage.sh --action start
./manage.sh --action stop  
./manage.sh --action restart

# View service logs
./manage.sh --action logs

# Get GPU information
./manage.sh --action gpu-info
```

### Model Management

**Recommended Method:**
```bash
# Download default models (SDXL base + VAE)
./manage.sh --action download-models

# List installed models
./manage.sh --action list-models

# Check model status and integrity
./manage.sh --action status
```

### Workflow Management

**Recommended Method:**
```bash
# Import a workflow from file
./manage.sh --action import-workflow --workflow my-workflow.json

# Execute a workflow
./manage.sh --action execute-workflow --workflow workflow.json

# Execute with custom output directory
./manage.sh --action execute-workflow --workflow workflow.json --output /path/to/output

# Validate workflow requirements
./manage.sh --action import-workflow --workflow workflow.json
```

## ComfyUI REST API

### Port Configuration

- **Port 8188**: ComfyUI Web Interface and API (main interface)
- **Port 5679**: AI-Dock Service Portal (container management)

### Core API Endpoints

#### System Information
```http
GET /system_stats
```

**Recommended Method:**
```bash
./manage.sh --action status
```

**Direct API:**
```bash
curl http://localhost:8188/system_stats
```

**Response:**
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

#### Workflow Execution
```http
POST /prompt
```

**Recommended Method:**
```bash
./manage.sh --action execute-workflow --workflow workflow.json
```

**Direct API:**
```bash
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

**Payload Format:**
```json
{
  "client_id": "your-client-id",
  "prompt": {
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
    }
  }
}
```

**Response:**
```json
{
  "prompt_id": "89f0e3f5-ae42-4c93-b4b7-0b3e52a0f8e5",
  "number": 1,
  "node_errors": {}
}
```

#### Execution History
```http
GET /history/{prompt_id}
```

```bash
curl http://localhost:8188/history/89f0e3f5-ae42-4c93-b4b7-0b3e52a0f8e5
```

#### Image Retrieval
```http
GET /view?filename={filename}&subfolder={subfolder}&type={type}
```

```bash
# Get generated image
curl http://localhost:8188/view?filename=output_00001_.png&type=output
```

#### Image Upload
```http
POST /upload/image
```

```bash
curl -X POST http://localhost:8188/upload/image \
  -F "image=@input.png" \
  -F "subfolder=inputs"
```

### Advanced API Usage

#### Queue Management
```bash
# Get queue status
curl http://localhost:8188/queue

# Clear queue
curl -X POST http://localhost:8188/queue \
  -H "Content-Type: application/json" \
  -d '{"clear": true}'

# Cancel specific execution
curl -X POST http://localhost:8188/interrupt
```

#### Node Information
```bash
# Get available nodes
curl http://localhost:8188/object_info

# Get specific node info
curl http://localhost:8188/object_info/CheckpointLoaderSimple
```

## WebSocket API

ComfyUI provides real-time updates through WebSocket connections:

### WebSocket Connection
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

### Message Types

#### Status Updates
```json
{
  "type": "status",
  "data": {
    "status": {
      "exec_info": {
        "queue_remaining": 0
      }
    }
  }
}
```

#### Progress Updates
```json
{
  "type": "progress", 
  "data": {
    "value": 5,
    "max": 20,
    "prompt_id": "prompt-id-here"
  }
}
```

#### Execution Events
```json
{
  "type": "executing",
  "data": {
    "node": "3",
    "prompt_id": "prompt-id-here"
  }
}
```

## Programming Language Integration

### Python Integration
```python
import requests
import json
import time

class ComfyUIClient:
    def __init__(self, server_address="http://localhost:8188"):
        self.server_address = server_address
    
    def submit_workflow(self, workflow, client_id="python-client"):
        """Submit workflow for execution"""
        payload = {
            "client_id": client_id,
            "prompt": workflow
        }
        
        response = requests.post(f'{self.server_address}/prompt', json=payload)
        return response.json()['prompt_id']
    
    def get_history(self, prompt_id):
        """Get execution history"""
        response = requests.get(f'{self.server_address}/history/{prompt_id}')
        return response.json()
    
    def wait_for_completion(self, prompt_id, timeout=300):
        """Wait for workflow completion"""
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            history = self.get_history(prompt_id)
            if prompt_id in history:
                return history[prompt_id]
            time.sleep(1)
        
        raise TimeoutError(f"Workflow {prompt_id} did not complete within {timeout}s")

# Usage example
client = ComfyUIClient()

# Load workflow
with open('workflow.json', 'r') as f:
    workflow = json.load(f)

# Submit and wait for completion
prompt_id = client.submit_workflow(workflow)
result = client.wait_for_completion(prompt_id)
print(f"Workflow completed: {result}")
```

### Node.js Integration
```javascript
const axios = require('axios');
const WebSocket = require('ws');

class ComfyUIClient {
    constructor(serverAddress = 'http://localhost:8188') {
        this.serverAddress = serverAddress;
        this.wsAddress = serverAddress.replace('http', 'ws') + '/ws';
    }
    
    async submitWorkflow(workflow, clientId = 'nodejs-client') {
        const payload = {
            client_id: clientId,
            prompt: workflow
        };
        
        const response = await axios.post(`${this.serverAddress}/prompt`, payload);
        return response.data.prompt_id;
    }
    
    async getHistory(promptId) {
        const response = await axios.get(`${this.serverAddress}/history/${promptId}`);
        return response.data;
    }
    
    connectWebSocket() {
        const ws = new WebSocket(this.wsAddress);
        
        ws.on('open', () => {
            console.log('Connected to ComfyUI WebSocket');
        });
        
        ws.on('message', (data) => {
            const message = JSON.parse(data);
            this.handleMessage(message);
        });
        
        return ws;
    }
    
    handleMessage(message) {
        switch(message.type) {
            case 'progress':
                console.log(`Progress: ${message.data.value}/${message.data.max}`);
                break;
            case 'executing':
                console.log(`Executing node: ${message.data.node}`);
                break;
        }
    }
}

// Usage
const client = new ComfyUIClient();
const ws = client.connectWebSocket();
```

## Complete Workflow Examples

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

### Batch Processing Example
```bash
# Execute multiple workflows
./manage.sh --action execute-workflow --workflow examples/basic_text_to_image.json
./manage.sh --action execute-workflow --workflow examples/pirate_rabbit_comic_composite_v2.json

# Custom batch processing
for workflow in examples/*.json; do
    echo "Processing $workflow"
    ./manage.sh --action execute-workflow --workflow "$workflow" --output "output/$(basename "$workflow" .json)"
done
```

## Container Network Access

When accessing ComfyUI API from other Docker containers:

### From Host Machine
```bash
# Standard localhost access
curl http://localhost:8188/system_stats
```

### From Another Container
```bash
# Use host machine's IP address
HOST_IP=$(hostname -I | awk '{print $1}')
curl http://$HOST_IP:8188/system_stats

# Example: 192.168.1.173:8188
curl http://192.168.1.173:8188/system_stats
```

### Docker Network Configuration
```bash
# Create shared network (if needed)
docker network create comfyui-network

# Connect containers to shared network
docker network connect comfyui-network comfyui
docker network connect comfyui-network your-other-container

# Use container name for communication
curl http://comfyui:8188/system_stats
```

## Error Handling

### Standard Error Responses
```json
{
  "error": {
    "type": "node_error", 
    "message": "Model file not found",
    "details": {
      "node_id": "3",
      "node_type": "CheckpointLoaderSimple"
    }
  }
}
```

### Common Error Types
- `missing_model`: Required model file not found
- `out_of_memory`: Insufficient VRAM/RAM for operation
- `invalid_input`: Workflow contains invalid node connections
- `timeout`: Operation exceeded time limit
- `gpu_error`: GPU-related processing error

## Integration Examples

### With n8n Workflows
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
        "bodyParametersJson": "{\"client_id\": \"n8n-workflow\", \"prompt\": {...}}"
      }
    },
    {
      "name": "Check Completion",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "GET", 
        "url": "http://localhost:8188/history/{{$json.prompt_id}}"
      }
    }
  ]
}
```

### With Shell Scripts
```bash
#!/bin/bash

# Submit workflow and capture prompt ID
PROMPT_ID=$(curl -s -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d @workflow.json | jq -r '.prompt_id')

echo "Submitted workflow: $PROMPT_ID"

# Wait for completion
while true; do
  STATUS=$(curl -s http://localhost:8188/history/$PROMPT_ID | jq -r 'keys | length')
  if [ "$STATUS" -gt 0 ]; then
    echo "Workflow completed!"
    break
  fi
  sleep 2
done

# Download generated images
curl -s http://localhost:8188/view?filename=output_00001_.png&type=output -o result.png
```

## Best Practices

### API Usage
1. **Use manage.sh for operations** - More reliable than direct API calls
2. **Implement proper error handling** - Always check response status
3. **Monitor memory usage** - Large models can consume significant VRAM
4. **Use WebSocket for real-time updates** - More efficient than polling
5. **Batch process when possible** - Reduces API overhead

### Workflow Design
1. **Validate workflows before submission** - Use import-workflow action
2. **Include error handling nodes** - Handle edge cases gracefully
3. **Optimize for available hardware** - Adjust batch sizes for your GPU
4. **Use appropriate model sizes** - Balance quality vs. performance
5. **Save intermediate results** - Helps with debugging complex workflows

### Performance Optimization
1. **Pre-load models** - Avoid model switching overhead
2. **Use appropriate resolutions** - Start with 512x512 for testing
3. **Monitor VRAM usage** - Use system_stats endpoint
4. **Implement queuing** - Prevent overloading the system
5. **Cache frequently used workflows** - Store templates locally

See the [examples directory](../examples/) for complete workflow definitions and the [configuration guide](CONFIGURATION.md) for advanced setup options.