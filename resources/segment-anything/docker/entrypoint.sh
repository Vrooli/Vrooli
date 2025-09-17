#!/bin/bash
set -e

# Download models on first run if not present
MODEL_DIR="${SAM_MODEL_PATH:-/app/models}"
DEFAULT_MODEL="${DEFAULT_MODEL:-sam2_hiera_small}"

# Check if we need to download models
if [ ! -f "$MODEL_DIR/${DEFAULT_MODEL}.pt" ]; then
    echo "Models not found. Downloading default model: ${DEFAULT_MODEL}"
    python /app/download_models.py "${DEFAULT_MODEL}.pt"
fi

# Start the API server
exec python -m uvicorn api_server:app --host 0.0.0.0 --port 11454 --workers 1