#!/bin/bash
# MusicGen Core Functions

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
MUSICGEN_CORE_DIR="${APP_ROOT}/resources/musicgen/lib"
source "${MUSICGEN_CORE_DIR}/common.sh"

# Install MusicGen
musicgen::install() {
    log::header "Installing MusicGen"
    
    # Create directories
    mkdir -p "${MUSICGEN_DATA_DIR}" "${MUSICGEN_MODELS_DIR}" "${MUSICGEN_OUTPUT_DIR}" "${MUSICGEN_CONFIG_DIR}" "${MUSICGEN_INJECT_DIR}"
    
    # Build Docker image
    log::info "Building MusicGen Docker image..."
    
    # Create Dockerfile
    cat > "${MUSICGEN_DATA_DIR}/Dockerfile" << 'EOF'
FROM python:3.10-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages with compatible numpy
RUN pip install --no-cache-dir \
    "numpy<2" \
    torch \
    torchaudio \
    audiocraft \
    flask \
    flask-cors \
    gunicorn

# Create app directory
WORKDIR /app

# Create API server
RUN cat > /app/server.py << 'PYTHON'
import os
import json
import base64
import tempfile
from flask import Flask, request, jsonify, send_file
from flask_cors import CORS
from audiocraft.models import MusicGen
from audiocraft.data.audio import audio_write
import torch

app = Flask(__name__)
CORS(app)

# Load model once at startup
model = None

def get_model(model_name="facebook/musicgen-melody"):
    global model
    if model is None:
        model = MusicGen.get_pretrained(model_name)
    return model

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "healthy", "service": "musicgen"})

@app.route('/generate', methods=['POST'])
def generate():
    try:
        data = request.json
        prompt = data.get('prompt', 'upbeat electronic music')
        duration = data.get('duration', 10)
        model_name = data.get('model', 'facebook/musicgen-melody')
        
        # Get model
        model = get_model(model_name)
        model.set_generation_params(duration=duration)
        
        # Generate audio
        wav = model.generate([prompt])
        
        # Save to temp file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as tmp:
            audio_write(tmp.name[:-4], wav[0].cpu(), model.sample_rate, strategy="loudness")
            
            # Read and encode
            with open(tmp.name, 'rb') as f:
                audio_data = base64.b64encode(f.read()).decode('utf-8')
            
            os.unlink(tmp.name)
            
        return jsonify({
            "status": "success",
            "audio": audio_data,
            "format": "wav",
            "sample_rate": model.sample_rate
        })
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 500

@app.route('/models', methods=['GET'])
def list_models():
    models = [
        "facebook/musicgen-melody",
        "facebook/musicgen-small",
        "facebook/musicgen-medium",
        "facebook/musicgen-large"
    ]
    return jsonify({"models": models})

if __name__ == '__main__':
    # Preload default model
    get_model()
    app.run(host='0.0.0.0', port=8765)
PYTHON

EXPOSE 8765

CMD ["python", "/app/server.py"]
EOF
    
    # Build image
    docker build -t "${MUSICGEN_IMAGE}" "${MUSICGEN_DATA_DIR}"
    
    # Register CLI
    "${VROOLI_DIR}/scripts/lib/resources/install-resource-cli.sh" \
        --name "musicgen" \
        --path "${MUSICGEN_BASE_DIR}/cli.sh"
    
    log::success "MusicGen installed successfully"
}

# Uninstall MusicGen
musicgen::uninstall() {
    log::header "Uninstalling MusicGen"
    
    # Stop container
    musicgen::stop
    
    # Remove image
    docker rmi -f "${MUSICGEN_IMAGE}" 2>/dev/null || true
    
    log::success "MusicGen uninstalled"
}

# Start MusicGen
musicgen::start() {
    log::header "Starting MusicGen"
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${MUSICGEN_CONTAINER_NAME}$"; then
        log::info "MusicGen is already running"
        return 0
    fi
    
    # Start container
    docker run -d \
        --name "${MUSICGEN_CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${MUSICGEN_PORT}:8765" \
        -v "${MUSICGEN_MODELS_DIR}:/models" \
        -v "${MUSICGEN_OUTPUT_DIR}:/outputs" \
        -v "${MUSICGEN_INJECT_DIR}:/inject" \
        "${MUSICGEN_IMAGE}"
    
    # Wait for health
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://localhost:${MUSICGEN_PORT}/health" > /dev/null 2>&1; then
            log::success "MusicGen started successfully"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    log::error "MusicGen failed to start"
    return 1
}

# Stop MusicGen
musicgen::stop() {
    log::header "Stopping MusicGen"
    
    if docker ps --format "{{.Names}}" | grep -q "^${MUSICGEN_CONTAINER_NAME}$"; then
        docker stop "${MUSICGEN_CONTAINER_NAME}" > /dev/null 2>&1
        docker rm "${MUSICGEN_CONTAINER_NAME}" > /dev/null 2>&1
        log::success "MusicGen stopped"
    else
        log::info "MusicGen is not running"
    fi
}

# List available models
musicgen::list_models() {
    log::info "Available MusicGen models:"
    echo "  • facebook/musicgen-small (300M parameters)"
    echo "  • facebook/musicgen-medium (1.5B parameters)" 
    echo "  • facebook/musicgen-large (3.3B parameters)"
    echo "  • facebook/musicgen-melody (1.5B parameters, melody conditioning)"
}

# Inject generation tasks
musicgen::inject() {
    local inject_file="${1:-}"
    
    if [[ -z "$inject_file" ]]; then
        log::error "Usage: resource-musicgen inject <file.json>"
        return 1
    fi
    
    if [[ ! -f "$inject_file" ]]; then
        log::error "Inject file not found: $inject_file"
        return 1
    fi
    
    # Copy to inject directory
    local basename=$(basename "$inject_file")
    cp "$inject_file" "${MUSICGEN_INJECT_DIR}/${basename}"
    log::success "Injected generation task: $basename"
}

# Generate music
musicgen::generate() {
    local prompt="${1:-upbeat electronic music}"
    local duration="${2:-10}"
    
    log::header "Generating Music"
    log::info "Prompt: ${prompt}"
    log::info "Duration: ${duration}s"
    
    # Make API call
    local response=$(curl -s -X POST "http://localhost:${MUSICGEN_PORT}/generate" \
        -H "Content-Type: application/json" \
        -d "{\"prompt\": \"${prompt}\", \"duration\": ${duration}}")
    
    if echo "${response}" | jq -e '.status == "success"' > /dev/null 2>&1; then
        # Save audio
        local timestamp=$(date +%Y%m%d_%H%M%S)
        local output_file="${MUSICGEN_OUTPUT_DIR}/music_${timestamp}.wav"
        echo "${response}" | jq -r '.audio' | base64 -d > "${output_file}"
        log::success "Music generated: ${output_file}"
    else
        log::error "Generation failed"
        echo "${response}" | jq -r '.message'
        return 1
    fi
}

# List models
musicgen::list_models() {
    curl -s "http://localhost:${MUSICGEN_PORT}/models" | jq -r '.models[]'
}

# Inject music tasks
musicgen::inject() {
    local inject_file="${1:-}"
    
    if [ -z "${inject_file}" ]; then
        log::error "Inject file required"
        return 1
    fi
    
    if [ ! -f "${inject_file}" ]; then
        log::error "Inject file not found: ${inject_file}"
        return 1
    fi
    
    # Copy to inject directory
    local filename=$(basename "${inject_file}")
    cp "${inject_file}" "${MUSICGEN_INJECT_DIR}/${filename}"
    
    log::success "Injected: ${filename}"
}