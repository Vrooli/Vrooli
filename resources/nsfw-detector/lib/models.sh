#!/bin/bash

# NSFW Detector Model Management
# Handles downloading and caching of AI models

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Model storage directory
MODEL_DIR="${RESOURCE_DIR}/models"
NSFWJS_MODEL_DIR="${MODEL_DIR}/nsfwjs"

# Model URLs (from nsfwjs library)
NSFWJS_BASE_URL="https://d1zv2aa70wpiur.cloudfront.net/tfjs_quant_nsfw_mobilenet"

# Function to download model files
download_nsfwjs_model() {
    local model_json="${NSFWJS_MODEL_DIR}/model.json"
    
    # Check if model already exists
    if [[ -f "$model_json" ]]; then
        echo "NSFW.js model already downloaded"
        return 0
    fi
    
    echo "Downloading NSFW.js model files..."
    mkdir -p "$NSFWJS_MODEL_DIR"
    
    # Download model.json
    if command -v wget &> /dev/null; then
        wget -q -O "${model_json}" "${NSFWJS_BASE_URL}/model.json" || {
            echo "Failed to download model.json"
            return 1
        }
    elif command -v curl &> /dev/null; then
        curl -sL -o "${model_json}" "${NSFWJS_BASE_URL}/model.json" || {
            echo "Failed to download model.json"
            return 1
        }
    else
        echo "Neither wget nor curl is available"
        return 1
    fi
    
    # Parse model.json to find weight files
    if command -v jq &> /dev/null; then
        # Extract weight files from model.json
        local weight_files=$(jq -r '.weightsManifest[].paths[]' "$model_json" 2>/dev/null || echo "")
        
        if [[ -n "$weight_files" ]]; then
            for weight_file in $weight_files; do
                echo "Downloading weight file: $weight_file"
                local weight_path="${NSFWJS_MODEL_DIR}/${weight_file}"
                
                if command -v wget &> /dev/null; then
                    wget -q -O "$weight_path" "${NSFWJS_BASE_URL}/${weight_file}" || {
                        echo "Failed to download $weight_file"
                        return 1
                    }
                else
                    curl -sL -o "$weight_path" "${NSFWJS_BASE_URL}/${weight_file}" || {
                        echo "Failed to download $weight_file"
                        return 1
                    }
                fi
            done
        else
            # Fallback: download common weight file names
            echo "Downloading weight shards (fallback method)..."
            for i in {1..4}; do
                local shard="group${i}-shard1of1.bin"
                if command -v wget &> /dev/null; then
                    wget -q -O "${NSFWJS_MODEL_DIR}/${shard}" "${NSFWJS_BASE_URL}/${shard}" 2>/dev/null || true
                else
                    curl -sL -o "${NSFWJS_MODEL_DIR}/${shard}" "${NSFWJS_BASE_URL}/${shard}" 2>/dev/null || true
                fi
            done
        fi
    else
        # Without jq, try common file patterns
        echo "Downloading weight files (no jq available)..."
        for shard in "group1-shard1of1.bin" "group2-shard1of1.bin" "group3-shard1of1.bin" "group4-shard1of1.bin"; do
            if command -v wget &> /dev/null; then
                wget -q -O "${NSFWJS_MODEL_DIR}/${shard}" "${NSFWJS_BASE_URL}/${shard}" 2>/dev/null || true
            else
                curl -sL -o "${NSFWJS_MODEL_DIR}/${shard}" "${NSFWJS_BASE_URL}/${shard}" 2>/dev/null || true
            fi
        done
    fi
    
    echo "NSFW.js model downloaded successfully"
    return 0
}

# Function to verify model integrity
verify_model() {
    local model_json="${NSFWJS_MODEL_DIR}/model.json"
    
    if [[ ! -f "$model_json" ]]; then
        echo "Model not found"
        return 1
    fi
    
    # Check if model.json is valid JSON
    if command -v jq &> /dev/null; then
        jq empty "$model_json" 2>/dev/null || {
            echo "Invalid model.json file"
            return 1
        }
    fi
    
    echo "Model verified"
    return 0
}

# Function to clean model cache
clean_models() {
    echo "Cleaning model cache..."
    rm -rf "$MODEL_DIR"
    echo "Model cache cleaned"
}

# Function to list available models
list_models() {
    echo "Available models:"
    if [[ -f "${NSFWJS_MODEL_DIR}/model.json" ]]; then
        echo "  - nsfwjs (downloaded)"
    else
        echo "  - nsfwjs (not downloaded)"
    fi
}

# Main command handling
case "${1:-}" in
    download)
        download_nsfwjs_model
        ;;
    verify)
        verify_model
        ;;
    clean)
        clean_models
        ;;
    list)
        list_models
        ;;
    *)
        echo "Usage: $0 {download|verify|clean|list}"
        echo "  download - Download NSFW.js model files"
        echo "  verify   - Verify model integrity"
        echo "  clean    - Remove cached models"
        echo "  list     - List available models"
        exit 1
        ;;
esac