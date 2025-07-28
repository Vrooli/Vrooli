#!/usr/bin/env bash
# ComfyUI Configuration Defaults
# This file contains all configuration constants and default values

# Resource identification
readonly COMFYUI_RESOURCE_NAME="comfyui"
readonly COMFYUI_CATEGORY="automation"

# Container configuration
readonly COMFYUI_CONTAINER_NAME="comfyui"
readonly COMFYUI_DEFAULT_IMAGE="ghcr.io/ai-dock/comfyui:latest"
readonly COMFYUI_DATA_DIR="${HOME}/.comfyui"

# Port configuration
readonly COMFYUI_DEFAULT_PORT="5679"
readonly COMFYUI_PROXY_PORT="1111"  # ai-dock proxy port
readonly COMFYUI_DIRECT_PORT="8188"  # ComfyUI direct port

# API configuration
readonly COMFYUI_API_BASE="http://localhost:${COMFYUI_DIRECT_PORT}"
readonly COMFYUI_HEALTH_ENDPOINT="/system_stats"
readonly COMFYUI_HEALTH_TIMEOUT=30

# Model configuration
readonly COMFYUI_DEFAULT_MODELS=(
    "https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0/resolve/main/sd_xl_base_1.0.safetensors"
    "https://huggingface.co/stabilityai/sdxl-vae/resolve/main/sdxl_vae.safetensors"
)
readonly COMFYUI_MODEL_NAMES=(
    "sd_xl_base_1.0.safetensors"
    "sdxl_vae.safetensors"
)
# Model metadata for verification
readonly COMFYUI_MODEL_SIZES=(
    "6938078334"  # sd_xl_base_1.0.safetensors - 6.5GB
    "334641164"   # sdxl_vae.safetensors - 319MB (actual HuggingFace file size)
)
readonly COMFYUI_MODEL_SHA256=(
    "31e35c80fc4829d14f90153f4c74cd59c90b779f6afe05a74cd6120b893f7e5b"  # sd_xl_base_1.0.safetensors
    "63aeecb90ff7bc1c115395962d3e803571385b61938377bc7089b36e81e92e2e"  # sdxl_vae.safetensors (actual)
)

# GPU configuration
readonly COMFYUI_GPU_TYPES=("nvidia" "amd" "cpu" "auto")
readonly COMFYUI_NVIDIA_MIN_DRIVER="450.80.02"
readonly COMFYUI_AMD_MIN_ROCM="5.0"

# Resource requirements
readonly COMFYUI_MIN_RAM_GB=16
readonly COMFYUI_MIN_DISK_GB=50
readonly COMFYUI_RECOMMENDED_RAM_GB=32
readonly COMFYUI_RECOMMENDED_DISK_GB=100

# Timeout configuration (in seconds)
readonly COMFYUI_INSTALL_TIMEOUT=600    # 10 minutes
readonly COMFYUI_START_TIMEOUT=120      # 2 minutes
readonly COMFYUI_STOP_TIMEOUT=30        # 30 seconds
readonly COMFYUI_MODEL_DOWNLOAD_TIMEOUT=1800  # 30 minutes

# Directory structure
readonly COMFYUI_DIRS=(
    "${COMFYUI_DATA_DIR}/models/checkpoints"
    "${COMFYUI_DATA_DIR}/models/vae"
    "${COMFYUI_DATA_DIR}/models/loras"
    "${COMFYUI_DATA_DIR}/models/controlnet"
    "${COMFYUI_DATA_DIR}/models/clip"
    "${COMFYUI_DATA_DIR}/models/clip_vision"
    "${COMFYUI_DATA_DIR}/models/diffusers"
    "${COMFYUI_DATA_DIR}/models/embeddings"
    "${COMFYUI_DATA_DIR}/models/gligen"
    "${COMFYUI_DATA_DIR}/models/hypernetworks"
    "${COMFYUI_DATA_DIR}/models/style_models"
    "${COMFYUI_DATA_DIR}/models/upscale_models"
    "${COMFYUI_DATA_DIR}/custom_nodes"
    "${COMFYUI_DATA_DIR}/outputs"
    "${COMFYUI_DATA_DIR}/input"
    "${COMFYUI_DATA_DIR}/workflows"
    "${COMFYUI_DATA_DIR}/user"
)

# Volume mappings for Docker
readonly COMFYUI_VOLUMES=(
    "${COMFYUI_DATA_DIR}/models:/workspace/ComfyUI/models"
    "${COMFYUI_DATA_DIR}/custom_nodes:/workspace/ComfyUI/custom_nodes"
    "${COMFYUI_DATA_DIR}/outputs:/workspace/ComfyUI/outputs"
    "${COMFYUI_DATA_DIR}/input:/workspace/ComfyUI/input"
    "${COMFYUI_DATA_DIR}/workflows:/workspace/ComfyUI/workflows"
    "${COMFYUI_DATA_DIR}/user:/workspace/ComfyUI/user"
)

# Environment variables for container
readonly COMFYUI_ENV_VARS=(
    "WORKSPACE=/workspace/ComfyUI"
    "CF_QUICK_TUNNELS=false"
    "WEB_ENABLE_AUTH=false"
    "SERVERLESS=false"
    "PUBLIC_KEY=''"
)

# Export variables that might be needed by other scripts
export COMFYUI_CUSTOM_PORT="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
export COMFYUI_CUSTOM_IMAGE="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
export COMFYUI_GPU_TYPE="${COMFYUI_GPU_TYPE:-auto}"
export COMFYUI_VRAM_LIMIT="${COMFYUI_VRAM_LIMIT:-}"
export COMFYUI_NVIDIA_CHOICE="${COMFYUI_NVIDIA_CHOICE:-}"