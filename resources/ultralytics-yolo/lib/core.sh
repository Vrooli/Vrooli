#!/bin/bash
# Core functionality for Ultralytics YOLO resource

set -euo pipefail

# Load configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Source port registry
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"

#######################################
# Show runtime configuration
#######################################
yolo::info() {
    cat << EOF
Ultralytics YOLO Configuration:
  Port: ${YOLO_PORT}
  Model: ${YOLO_MODEL_VARIANT}
  Task: ${YOLO_TASK}
  Device: ${YOLO_DEVICE}
  Confidence: ${YOLO_CONFIDENCE}
  IOU Threshold: ${YOLO_IOU_THRESHOLD}
  Image Size: ${YOLO_IMAGE_SIZE}
  
Integrations:
  Qdrant: ${YOLO_ENABLE_QDRANT}
  PostgreSQL: ${YOLO_ENABLE_POSTGRES}
  MinIO: ${YOLO_ENABLE_MINIO}
  Redis: ${YOLO_ENABLE_REDIS}
  
Docker:
  Image: ${YOLO_DOCKER_IMAGE}
  Container: ${YOLO_CONTAINER_NAME}
  GPU: ${YOLO_DOCKER_GPU}
EOF
}

#######################################
# Show service status
#######################################
yolo::status() {
    local verbose="${1:-}"
    
    echo "Ultralytics YOLO Status:"
    
    # Check if container is running
    if docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        echo "  Service: Running ✓"
        
        # Check health endpoint
        if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/health" &> /dev/null; then
            echo "  Health: Healthy ✓"
            
            if [[ "$verbose" == "--verbose" ]]; then
                echo "  Health Response:"
                timeout 5 curl -sf "http://localhost:${YOLO_PORT}/health" | jq '.' 2>/dev/null || echo "    (Unable to parse JSON)"
            fi
        else
            echo "  Health: Unhealthy ✗"
        fi
        
        # Check GPU availability
        if command -v nvidia-smi &> /dev/null; then
            if nvidia-smi &> /dev/null; then
                echo "  GPU: Available ✓"
                if [[ "$verbose" == "--verbose" ]]; then
                    echo "  GPU Info:"
                    nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader | sed 's/^/    /'
                fi
            else
                echo "  GPU: Not Available ✗"
            fi
        else
            echo "  GPU: Not Configured"
        fi
        
        # Show model status
        if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/models/status" &> /dev/null; then
            echo "  Models: Loaded ✓"
            if [[ "$verbose" == "--verbose" ]]; then
                echo "  Available Models:"
                timeout 5 curl -sf "http://localhost:${YOLO_PORT}/models" | jq -r '.models[]' 2>/dev/null | sed 's/^/    /' || echo "    (Unable to fetch)"
            fi
        else
            echo "  Models: Not Loaded ✗"
        fi
        
    else
        echo "  Service: Stopped ✗"
        echo "  Health: N/A"
        echo "  GPU: N/A"
        echo "  Models: N/A"
    fi
    
    # Show integration status if verbose
    if [[ "$verbose" == "--verbose" ]]; then
        echo ""
        echo "Integration Status:"
        
        if [[ "${YOLO_ENABLE_QDRANT}" == "true" ]]; then
            if timeout 2 curl -sf "http://${QDRANT_HOST}:${QDRANT_PORT}/collections" &> /dev/null; then
                echo "  Qdrant: Connected ✓"
            else
                echo "  Qdrant: Not Connected ✗"
            fi
        else
            echo "  Qdrant: Disabled"
        fi
        
        if [[ "${YOLO_ENABLE_POSTGRES}" == "true" ]]; then
            if timeout 2 pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" &> /dev/null; then
                echo "  PostgreSQL: Connected ✓"
            else
                echo "  PostgreSQL: Not Connected ✗"
            fi
        else
            echo "  PostgreSQL: Disabled"
        fi
        
        if [[ "${YOLO_ENABLE_MINIO}" == "true" ]]; then
            if timeout 2 curl -sf "http://${MINIO_ENDPOINT}/minio/health/live" &> /dev/null; then
                echo "  MinIO: Connected ✓"
            else
                echo "  MinIO: Not Connected ✗"
            fi
        else
            echo "  MinIO: Disabled"
        fi
        
        if [[ "${YOLO_ENABLE_REDIS}" == "true" ]]; then
            if timeout 2 redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping &> /dev/null; then
                echo "  Redis: Connected ✓"
            else
                echo "  Redis: Not Connected ✗"
            fi
        else
            echo "  Redis: Disabled"
        fi
    fi
    
    return 0
}

#######################################
# View service logs
#######################################
yolo::logs() {
    local follow="${1:-}"
    
    if docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
            docker logs -f "${YOLO_CONTAINER_NAME}"
        else
            docker logs "${YOLO_CONTAINER_NAME}" --tail 100
        fi
    else
        echo "Service is not running"
        return 1
    fi
}

#######################################
# Install YOLO with dependencies
#######################################
yolo::install() {
    echo "Installing Ultralytics YOLO..."
    
    # Pull Docker image
    echo "  Pulling Docker image..."
    docker pull "${YOLO_DOCKER_IMAGE}"
    
    # Create model directory
    echo "  Creating model cache directory..."
    mkdir -p "${HOME}/.yolo/models"
    
    # Create API server script
    echo "  Creating API server..."
    cat > "${RESOURCE_DIR}/docker/api_server.py" << 'EOF'
#!/usr/bin/env python3
"""
Ultralytics YOLO API Server
Provides REST endpoints for object detection, segmentation, and classification
"""

import os
import json
import time
import torch
import numpy as np
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime

from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import uvicorn
from ultralytics import YOLO

# Configuration
PORT = int(os.getenv('YOLO_PORT', 11455))
MODEL_PATH = os.getenv('YOLO_MODEL_PATH', '/models')
DEFAULT_MODEL = os.getenv('YOLO_DEFAULT_MODEL', 'yolov8m.pt')
DEVICE = os.getenv('YOLO_DEVICE', 'auto')
CONFIDENCE = float(os.getenv('YOLO_CONFIDENCE', 0.25))
IOU_THRESHOLD = float(os.getenv('YOLO_IOU_THRESHOLD', 0.45))
IMAGE_SIZE = int(os.getenv('YOLO_IMAGE_SIZE', 640))

# Initialize FastAPI
app = FastAPI(
    title="Ultralytics YOLO API",
    description="Real-time object detection, segmentation, and classification",
    version="1.0.0"
)

# Model cache
models = {}
current_model = None

# Device selection
def get_device():
    if DEVICE == 'auto':
        return 'cuda' if torch.cuda.is_available() else 'cpu'
    return DEVICE

# Load model
def load_model(model_name: str = DEFAULT_MODEL):
    global current_model
    if model_name not in models:
        model_path = Path(MODEL_PATH) / model_name
        if not model_path.exists():
            # Download if not exists
            model = YOLO(model_name)
        else:
            model = YOLO(str(model_path))
        models[model_name] = model
    current_model = models[model_name]
    return current_model

# Health check endpoint
@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "device": get_device(),
        "model_loaded": current_model is not None,
        "models_cached": list(models.keys())
    }

# Model status endpoint
@app.get("/models/status")
async def model_status():
    return {
        "current_model": DEFAULT_MODEL if current_model else None,
        "available_models": list(models.keys()),
        "device": get_device(),
        "cuda_available": torch.cuda.is_available()
    }

# List models endpoint
@app.get("/models")
async def list_models():
    model_dir = Path(MODEL_PATH)
    model_files = list(model_dir.glob("*.pt"))
    return {
        "models": [f.name for f in model_files],
        "current": DEFAULT_MODEL if current_model else None
    }

# Detection endpoint
@app.post("/detect")
async def detect(
    file: UploadFile = File(...),
    model: str = DEFAULT_MODEL,
    confidence: float = CONFIDENCE,
    iou: float = IOU_THRESHOLD
):
    try:
        # Load model
        yolo = load_model(model)
        
        # Read image
        contents = await file.read()
        nparr = np.frombuffer(contents, np.uint8)
        import cv2
        img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
        
        # Run detection
        start_time = time.time()
        results = yolo(
            img, 
            conf=confidence, 
            iou=iou,
            device=get_device()
        )
        inference_time = (time.time() - start_time) * 1000
        
        # Process results
        detections = []
        for r in results:
            if r.boxes is not None:
                for box in r.boxes:
                    detections.append({
                        "class": r.names[int(box.cls)],
                        "confidence": float(box.conf),
                        "bbox": box.xyxy[0].tolist(),
                        "class_id": int(box.cls)
                    })
        
        return {
            "detections": detections,
            "metadata": {
                "model": model,
                "inference_time": inference_time,
                "image_size": [img.shape[1], img.shape[0]],
                "device": get_device()
            }
        }
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

# Initialize on startup
@app.on_event("startup")
async def startup_event():
    load_model(DEFAULT_MODEL)
    print(f"YOLO API Server running on port {PORT}")
    print(f"Device: {get_device()}")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
EOF
    
    # Create requirements file
    cat > "${RESOURCE_DIR}/docker/requirements.txt" << 'EOF'
ultralytics>=8.0.0
fastapi
uvicorn[standard]
python-multipart
opencv-python-headless
torch
torchvision
numpy
pillow
EOF
    
    # Create Dockerfile
    cat > "${RESOURCE_DIR}/docker/Dockerfile" << 'EOF'
FROM ultralytics/ultralytics:latest

WORKDIR /app

# Install additional dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy API server
COPY api_server.py .

# Create model directory
RUN mkdir -p /models

EXPOSE 11455

CMD ["python", "api_server.py"]
EOF
    
    echo "✓ Ultralytics YOLO installed successfully"
    return 0
}

#######################################
# Start YOLO service
#######################################
yolo::start() {
    local wait_flag="${1:-}"
    
    echo "Starting Ultralytics YOLO service..."
    
    # Check if already running
    if docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        echo "Service is already running"
        return 0
    fi
    
    # Build Docker image if needed
    if ! docker images | grep -q "vrooli/ultralytics-yolo"; then
        echo "  Building Docker image..."
        docker build -t vrooli/ultralytics-yolo "${RESOURCE_DIR}/docker"
    fi
    
    # Prepare GPU flags
    local gpu_flags=""
    if [[ "${YOLO_DEVICE}" != "cpu" ]] && command -v nvidia-smi &> /dev/null; then
        if nvidia-smi &> /dev/null; then
            gpu_flags="--gpus ${YOLO_DOCKER_GPU}"
        fi
    fi
    
    # Start container
    echo "  Starting container..."
    docker run -d \
        --name "${YOLO_CONTAINER_NAME}" \
        -p "${YOLO_PORT}:${YOLO_PORT}" \
        -e YOLO_PORT="${YOLO_PORT}" \
        -e YOLO_DEVICE="${YOLO_DEVICE}" \
        -e YOLO_CONFIDENCE="${YOLO_CONFIDENCE}" \
        -e YOLO_IOU_THRESHOLD="${YOLO_IOU_THRESHOLD}" \
        -e YOLO_IMAGE_SIZE="${YOLO_IMAGE_SIZE}" \
        -v "${HOME}/.yolo/models:/models" \
        $gpu_flags \
        --restart unless-stopped \
        vrooli/ultralytics-yolo
    
    # Wait for service to be ready
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "  Waiting for service to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/health" &> /dev/null; then
                echo "✓ Ultralytics YOLO service started successfully"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "✗ Service failed to start within timeout"
        return 1
    fi
    
    echo "✓ Ultralytics YOLO service started"
    return 0
}

#######################################
# Stop YOLO service
#######################################
yolo::stop() {
    echo "Stopping Ultralytics YOLO service..."
    
    if docker ps --format "table {{.Names}}" | grep -q "^${YOLO_CONTAINER_NAME}$"; then
        docker stop "${YOLO_CONTAINER_NAME}"
        docker rm "${YOLO_CONTAINER_NAME}"
        echo "✓ Service stopped successfully"
    else
        echo "Service is not running"
    fi
    
    return 0
}

#######################################
# Restart YOLO service
#######################################
yolo::restart() {
    echo "Restarting Ultralytics YOLO service..."
    yolo::stop
    sleep 2
    yolo::start --wait
}

#######################################
# Uninstall YOLO
#######################################
yolo::uninstall() {
    echo "Uninstalling Ultralytics YOLO..."
    
    # Stop service if running
    yolo::stop
    
    # Remove Docker image
    if docker images | grep -q "vrooli/ultralytics-yolo"; then
        echo "  Removing Docker image..."
        docker rmi vrooli/ultralytics-yolo
    fi
    
    # Optionally remove model cache
    read -p "Remove model cache? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${HOME}/.yolo/models"
    fi
    
    echo "✓ Ultralytics YOLO uninstalled"
    return 0
}

#######################################
# Run tests
#######################################
yolo::test() {
    local phase="${1:-all}"
    
    # Source test library
    source "${RESOURCE_DIR}/lib/test.sh"
    
    case "$phase" in
        smoke)
            yolo::test_smoke
            ;;
        integration)
            yolo::test_integration
            ;;
        unit)
            yolo::test_unit
            ;;
        all)
            yolo::test_smoke
            yolo::test_integration
            yolo::test_unit
            ;;
        *)
            echo "Unknown test phase: $phase"
            exit 1
            ;;
    esac
}

#######################################
# Content management functions
#######################################
yolo::content_list() {
    echo "Available YOLO models:"
    echo "  yolov8n - Nano (fastest, least accurate)"
    echo "  yolov8s - Small"
    echo "  yolov8m - Medium (balanced)"
    echo "  yolov8l - Large"
    echo "  yolov8x - Extra Large (slowest, most accurate)"
    echo ""
    echo "Cached models:"
    if [[ -d "${HOME}/.yolo/models" ]]; then
        ls -la "${HOME}/.yolo/models/*.pt" 2>/dev/null | awk '{print "  " $NF}' || echo "  (none)"
    else
        echo "  (none)"
    fi
}

yolo::content_add() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        echo "Usage: resource-ultralytics-yolo content add <model>"
        echo "Models: yolov8n, yolov8s, yolov8m, yolov8l, yolov8x"
        return 1
    fi
    
    echo "Downloading model: $model"
    # This would trigger model download via the API
    if timeout 5 curl -sf "http://localhost:${YOLO_PORT}/models/pull" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "{\"model\": \"$model\"}" &> /dev/null; then
        echo "✓ Model downloaded successfully"
    else
        echo "✗ Failed to download model (is service running?)"
        return 1
    fi
}

yolo::content_get() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        echo "Usage: resource-ultralytics-yolo content get <model>"
        return 1
    fi
    
    echo "Model information: $model"
    if [[ -f "${HOME}/.yolo/models/${model}.pt" ]]; then
        ls -lh "${HOME}/.yolo/models/${model}.pt"
    else
        echo "Model not found in cache"
        return 1
    fi
}

yolo::content_remove() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        echo "Usage: resource-ultralytics-yolo content remove <model>"
        return 1
    fi
    
    if [[ -f "${HOME}/.yolo/models/${model}.pt" ]]; then
        rm -f "${HOME}/.yolo/models/${model}.pt"
        echo "✓ Model removed"
    else
        echo "Model not found in cache"
        return 1
    fi
}

#######################################
# Detection functions
#######################################
yolo::detect() {
    local image=""
    local video=""
    local model="yolov8m"
    local output=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --image)
                image="$2"
                shift 2
                ;;
            --video)
                video="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$image" && -z "$video" ]]; then
        echo "Usage: resource-ultralytics-yolo detect --image <path> [--model <name>] [--output <path>]"
        return 1
    fi
    
    # Check service is running
    if ! timeout 5 curl -sf "http://localhost:${YOLO_PORT}/health" &> /dev/null; then
        echo "Service is not running. Start with: resource-ultralytics-yolo manage start"
        return 1
    fi
    
    # Run detection
    if [[ -n "$image" ]]; then
        echo "Running detection on image: $image"
        local result=$(curl -sf -X POST \
            "http://localhost:${YOLO_PORT}/detect" \
            -F "file=@$image" \
            -F "model=$model")
        
        if [[ -n "$output" ]]; then
            echo "$result" > "$output"
            echo "✓ Results saved to: $output"
        else
            echo "$result" | jq '.'
        fi
    fi
}

yolo::segment() {
    echo "Segmentation endpoint (similar to detect)"
    yolo::detect "$@"
}

yolo::classify() {
    echo "Classification endpoint (similar to detect)"
    yolo::detect "$@"
}