#!/usr/bin/env bash
################################################################################
# AudioCraft Core Library
# Core functionality for AudioCraft resource management
################################################################################
set -euo pipefail

# Resource directories
AUDIOCRAFT_DIR="${SCRIPT_DIR}"
AUDIOCRAFT_DATA_DIR="${AUDIOCRAFT_DATA_DIR:-${APP_ROOT}/data/resources/audiocraft}"
AUDIOCRAFT_CONFIG_DIR="${AUDIOCRAFT_CONFIG_DIR:-${AUDIOCRAFT_DATA_DIR}/config}"
AUDIOCRAFT_MODELS_DIR="${AUDIOCRAFT_MODELS_DIR:-${AUDIOCRAFT_DATA_DIR}/models}"
AUDIOCRAFT_OUTPUT_DIR="${AUDIOCRAFT_OUTPUT_DIR:-${AUDIOCRAFT_DATA_DIR}/output}"

# Container settings
AUDIOCRAFT_PORT="${AUDIOCRAFT_PORT:-7862}"
AUDIOCRAFT_CONTAINER_NAME="${AUDIOCRAFT_CONTAINER_NAME:-vrooli-audiocraft}"
AUDIOCRAFT_IMAGE_NAME="${AUDIOCRAFT_IMAGE_NAME:-vrooli/audiocraft:latest}"

################################################################################
# Management Functions
################################################################################

audiocraft::manage::help() {
    cat << EOF
⚙️  AudioCraft Management Commands

USAGE:
    resource-audiocraft manage <subcommand> [options]

SUBCOMMANDS:
    install              Install AudioCraft and dependencies
    uninstall            Remove AudioCraft completely
    start                Start AudioCraft service
    stop                 Stop AudioCraft service
    restart              Restart AudioCraft service

OPTIONS:
    --force              Skip confirmation prompts
    --wait               Wait for service to be ready
    --timeout <seconds>  Timeout for operations

EXAMPLES:
    resource-audiocraft manage install
    resource-audiocraft manage start --wait
    resource-audiocraft manage stop --force

EOF
}

audiocraft::manage::install() {
    echo "🎵 Installing AudioCraft resource..."
    
    # Create data directories
    mkdir -p "${AUDIOCRAFT_DATA_DIR}"
    mkdir -p "${AUDIOCRAFT_MODELS_DIR}"
    mkdir -p "${AUDIOCRAFT_OUTPUT_DIR}"
    mkdir -p "${AUDIOCRAFT_CONFIG_DIR}"
    
    # Build Docker image
    echo "Building AudioCraft Docker image..."
    if [[ -f "${AUDIOCRAFT_DIR}/docker/Dockerfile" ]]; then
        docker build -t "${AUDIOCRAFT_IMAGE_NAME}" "${AUDIOCRAFT_DIR}/docker"
    else
        # Create minimal Dockerfile if not exists
        mkdir -p "${AUDIOCRAFT_DIR}/docker"
        cat > "${AUDIOCRAFT_DIR}/docker/Dockerfile" << 'DOCKERFILE'
FROM python:3.9-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    torch==2.1.0 \
    torchaudio \
    audiocraft \
    flask \
    flask-cors \
    numpy \
    scipy

# Create app directory
WORKDIR /app

# Copy API server script
COPY api_server.py /app/

# Expose port
EXPOSE 7862

# Run API server
CMD ["python", "api_server.py"]
DOCKERFILE
        
        # Create API server script
        cat > "${AUDIOCRAFT_DIR}/docker/api_server.py" << 'PYTHON'
#!/usr/bin/env python3
import os
import json
from flask import Flask, request, jsonify, send_file
from flask_cors import CORS
import torch
from audiocraft.models import MusicGen, AudioGen
from audiocraft.data.audio import audio_write
import tempfile
import traceback

app = Flask(__name__)
CORS(app)

# Load models on startup
print("Loading AudioCraft models...")
try:
    musicgen_model = MusicGen.get_pretrained('facebook/musicgen-medium')
    print("MusicGen model loaded")
except Exception as e:
    print(f"Failed to load MusicGen: {e}")
    musicgen_model = None

try:
    audiogen_model = AudioGen.get_pretrained('facebook/audiogen-medium')
    print("AudioGen model loaded")
except Exception as e:
    print(f"Failed to load AudioGen: {e}")
    audiogen_model = None

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        'status': 'healthy',
        'musicgen': musicgen_model is not None,
        'audiogen': audiogen_model is not None
    })

@app.route('/api/generate/music', methods=['POST'])
def generate_music():
    try:
        if not musicgen_model:
            return jsonify({'error': 'MusicGen model not loaded'}), 503
            
        data = request.json
        prompt = data.get('prompt', 'relaxing ambient music')
        duration = min(data.get('duration', 10), 120)
        
        # Set generation parameters
        musicgen_model.set_generation_params(duration=duration)
        
        # Generate music
        wav = musicgen_model.generate([prompt])
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as f:
            audio_write(f.name[:-4], wav[0].cpu(), musicgen_model.sample_rate)
            return send_file(f.name, mimetype='audio/wav')
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/generate/sound', methods=['POST'])
def generate_sound():
    try:
        if not audiogen_model:
            return jsonify({'error': 'AudioGen model not loaded'}), 503
            
        data = request.json
        prompt = data.get('prompt', 'rain sound')
        duration = min(data.get('duration', 5), 30)
        
        # Set generation parameters
        audiogen_model.set_generation_params(duration=duration)
        
        # Generate sound
        wav = audiogen_model.generate([prompt])
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as f:
            audio_write(f.name[:-4], wav[0].cpu(), audiogen_model.sample_rate)
            return send_file(f.name, mimetype='audio/wav')
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/models', methods=['GET'])
def list_models():
    return jsonify({
        'musicgen': {
            'loaded': musicgen_model is not None,
            'variants': ['small', 'medium', 'large', 'melody']
        },
        'audiogen': {
            'loaded': audiogen_model is not None,
            'variants': ['medium']
        }
    })

if __name__ == '__main__':
    port = int(os.environ.get('AUDIOCRAFT_PORT', 7862))
    app.run(host='0.0.0.0', port=port, debug=False)
PYTHON
        
        docker build -t "${AUDIOCRAFT_IMAGE_NAME}" "${AUDIOCRAFT_DIR}/docker"
    fi
    
    echo "✅ AudioCraft installed successfully"
}

audiocraft::manage::uninstall() {
    echo "🗑️  Uninstalling AudioCraft resource..."
    
    # Stop container if running
    audiocraft::manage::stop
    
    # Remove Docker image
    if docker images | grep -q "${AUDIOCRAFT_IMAGE_NAME}"; then
        docker rmi "${AUDIOCRAFT_IMAGE_NAME}" || true
    fi
    
    # Optionally remove data
    if [[ "${1:-}" != "--keep-data" ]]; then
        echo "Removing AudioCraft data..."
        rm -rf "${AUDIOCRAFT_DATA_DIR}"
    fi
    
    echo "✅ AudioCraft uninstalled"
}

audiocraft::manage::start() {
    echo "🚀 Starting AudioCraft service..."
    
    # Check if already running
    if docker ps | grep -q "${AUDIOCRAFT_CONTAINER_NAME}"; then
        echo "AudioCraft is already running"
        return 0
    fi
    
    # Start container
    docker run -d \
        --name "${AUDIOCRAFT_CONTAINER_NAME}" \
        -p "${AUDIOCRAFT_PORT}:${AUDIOCRAFT_PORT}" \
        -v "${AUDIOCRAFT_MODELS_DIR}:/models" \
        -v "${AUDIOCRAFT_OUTPUT_DIR}:/output" \
        -e AUDIOCRAFT_PORT="${AUDIOCRAFT_PORT}" \
        -e AUDIOCRAFT_MODEL_SIZE="${AUDIOCRAFT_MODEL_SIZE:-medium}" \
        -e AUDIOCRAFT_USE_GPU="${AUDIOCRAFT_USE_GPU:-false}" \
        --restart unless-stopped \
        "${AUDIOCRAFT_IMAGE_NAME}"
    
    # Wait for service to be ready if requested
    if [[ "${1:-}" == "--wait" ]]; then
        echo "Waiting for AudioCraft to be ready..."
        local timeout="${2:-60}"
        local elapsed=0
        
        while [[ $elapsed -lt $timeout ]]; do
            if curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1; then
                echo "✅ AudioCraft is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        
        echo "⚠️  AudioCraft did not become ready within ${timeout} seconds"
        return 1
    fi
    
    echo "✅ AudioCraft started"
}

audiocraft::manage::stop() {
    echo "🛑 Stopping AudioCraft service..."
    
    if docker ps | grep -q "${AUDIOCRAFT_CONTAINER_NAME}"; then
        docker stop "${AUDIOCRAFT_CONTAINER_NAME}" || true
        docker rm "${AUDIOCRAFT_CONTAINER_NAME}" || true
        echo "✅ AudioCraft stopped"
    else
        echo "AudioCraft is not running"
    fi
}

audiocraft::manage::restart() {
    echo "🔄 Restarting AudioCraft service..."
    audiocraft::manage::stop
    audiocraft::manage::start "$@"
}

################################################################################
# Test Functions
################################################################################

audiocraft::test::help() {
    cat << EOF
🧪 AudioCraft Test Commands

USAGE:
    resource-audiocraft test <subcommand>

SUBCOMMANDS:
    all                  Run all test suites
    smoke                Quick health validation
    integration          Integration tests
    unit                 Unit tests

EXAMPLES:
    resource-audiocraft test smoke
    resource-audiocraft test all

EOF
}

audiocraft::test::all() {
    echo "🧪 Running all AudioCraft tests..."
    
    if [[ -f "${AUDIOCRAFT_DIR}/test/run-tests.sh" ]]; then
        bash "${AUDIOCRAFT_DIR}/test/run-tests.sh" all
    else
        audiocraft::test::smoke
        audiocraft::test::integration
    fi
}

audiocraft::test::smoke() {
    echo "🔥 Running smoke tests..."
    
    if [[ -f "${AUDIOCRAFT_DIR}/test/phases/test-smoke.sh" ]]; then
        bash "${AUDIOCRAFT_DIR}/test/phases/test-smoke.sh"
    else
        # Basic health check
        if curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1; then
            echo "✅ Health check passed"
            return 0
        else
            echo "❌ Health check failed"
            return 1
        fi
    fi
}

audiocraft::test::integration() {
    echo "🔗 Running integration tests..."
    
    if [[ -f "${AUDIOCRAFT_DIR}/test/phases/test-integration.sh" ]]; then
        bash "${AUDIOCRAFT_DIR}/test/phases/test-integration.sh"
    else
        echo "No integration tests defined"
        return 2
    fi
}

audiocraft::test::unit() {
    echo "📦 Running unit tests..."
    
    if [[ -f "${AUDIOCRAFT_DIR}/test/phases/test-unit.sh" ]]; then
        bash "${AUDIOCRAFT_DIR}/test/phases/test-unit.sh"
    else
        echo "No unit tests defined"
        return 2
    fi
}

################################################################################
# Content Functions
################################################################################

audiocraft::content::help() {
    cat << EOF
📄 AudioCraft Content Commands

USAGE:
    resource-audiocraft content <subcommand> [options]

SUBCOMMANDS:
    list                 List generated audio files
    add                  Add audio file or prompt
    get                  Retrieve specific audio
    remove               Remove audio file
    execute              Generate audio from prompt

OPTIONS:
    --format <type>      Output format (text/json)
    --filter <pattern>   Filter results

EXAMPLES:
    resource-audiocraft content list
    resource-audiocraft content execute "relaxing piano music"

EOF
}

audiocraft::content::list() {
    echo "📋 Listing AudioCraft content..."
    
    if [[ -d "${AUDIOCRAFT_OUTPUT_DIR}" ]]; then
        find "${AUDIOCRAFT_OUTPUT_DIR}" -type f -name "*.wav" -o -name "*.mp3" | sort
    else
        echo "No content found"
    fi
}

audiocraft::content::add() {
    echo "➕ Adding content to AudioCraft..."
    echo "Use API endpoints to generate content"
}

audiocraft::content::get() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        echo "Error: Content name required"
        return 1
    fi
    
    local file="${AUDIOCRAFT_OUTPUT_DIR}/${name}"
    if [[ -f "$file" ]]; then
        echo "$file"
    else
        echo "Content not found: $name"
        return 1
    fi
}

audiocraft::content::remove() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        echo "Error: Content name required"
        return 1
    fi
    
    local file="${AUDIOCRAFT_OUTPUT_DIR}/${name}"
    if [[ -f "$file" ]]; then
        rm "$file"
        echo "Removed: $name"
    else
        echo "Content not found: $name"
        return 1
    fi
}

audiocraft::content::execute() {
    local prompt="${1:-relaxing ambient music}"
    echo "🎵 Generating audio: $prompt"
    
    curl -X POST "http://localhost:${AUDIOCRAFT_PORT}/api/generate/music" \
        -H "Content-Type: application/json" \
        -d "{\"prompt\": \"$prompt\", \"duration\": 10}" \
        -o "${AUDIOCRAFT_OUTPUT_DIR}/generated_$(date +%s).wav"
}

################################################################################
# Status Functions
################################################################################

audiocraft::status() {
    echo "📊 AudioCraft Status"
    echo "=================="
    
    # Check container status
    if docker ps | grep -q "${AUDIOCRAFT_CONTAINER_NAME}"; then
        echo "Container: ✅ Running"
        
        # Check health endpoint
        if curl -sf "http://localhost:${AUDIOCRAFT_PORT}/health" > /dev/null 2>&1; then
            echo "Health: ✅ Healthy"
            
            # Show model status
            curl -s "http://localhost:${AUDIOCRAFT_PORT}/api/models" 2>/dev/null | jq . || true
        else
            echo "Health: ❌ Unhealthy"
        fi
    else
        echo "Container: ❌ Not running"
    fi
    
    # Show configuration
    echo ""
    echo "Configuration:"
    echo "  Port: ${AUDIOCRAFT_PORT}"
    echo "  Data: ${AUDIOCRAFT_DATA_DIR}"
    echo "  Models: ${AUDIOCRAFT_MODELS_DIR}"
}

audiocraft::logs() {
    if docker ps -a | grep -q "${AUDIOCRAFT_CONTAINER_NAME}"; then
        docker logs "${AUDIOCRAFT_CONTAINER_NAME}" "$@"
    else
        echo "AudioCraft container not found"
        return 1
    fi
}

audiocraft::credentials() {
    cat << EOF
🔑 AudioCraft Credentials

API Endpoint: http://localhost:${AUDIOCRAFT_PORT}
Health Check: http://localhost:${AUDIOCRAFT_PORT}/health

No authentication required for local access.

Example usage:
  curl http://localhost:${AUDIOCRAFT_PORT}/api/models

EOF
}