#!/usr/bin/env bash
# Godot Engine Resource - Default Configuration

# API Configuration
export GODOT_API_PORT="${GODOT_API_PORT:-11457}"
export GODOT_LSP_PORT="${GODOT_LSP_PORT:-6005}"
export GODOT_DEBUG_PORT="${GODOT_DEBUG_PORT:-6007}"

# Godot Version
export GODOT_VERSION="${GODOT_VERSION:-4.3}"
export GODOT_HEADLESS="${GODOT_HEADLESS:-true}"

# Directory Configuration
export GODOT_BASE_DIR="${GODOT_BASE_DIR:-${HOME}/.vrooli/godot}"
export GODOT_PROJECTS_DIR="${GODOT_PROJECTS_DIR:-${GODOT_BASE_DIR}/projects}"
export GODOT_EXPORTS_DIR="${GODOT_EXPORTS_DIR:-${GODOT_BASE_DIR}/exports}"
export GODOT_TEMPLATES_DIR="${GODOT_TEMPLATES_DIR:-${GODOT_BASE_DIR}/templates}"
export GODOT_ASSETS_DIR="${GODOT_ASSETS_DIR:-${GODOT_BASE_DIR}/assets}"
export GODOT_LOG_DIR="${GODOT_LOG_DIR:-${GODOT_BASE_DIR}/logs}"

# Container Configuration
export GODOT_CONTAINER_NAME="${GODOT_CONTAINER_NAME:-vrooli-godot}"
export GODOT_IMAGE="${GODOT_IMAGE:-barichello/godot-ci:${GODOT_VERSION}}"
export GODOT_MEMORY_LIMIT="${GODOT_MEMORY_LIMIT:-2G}"
export GODOT_CPU_LIMIT="${GODOT_CPU_LIMIT:-2}"

# Performance Configuration
export GODOT_MAX_CONCURRENT_BUILDS="${GODOT_MAX_CONCURRENT_BUILDS:-2}"
export GODOT_BUILD_TIMEOUT="${GODOT_BUILD_TIMEOUT:-300}"
export GODOT_EXPORT_TIMEOUT="${GODOT_EXPORT_TIMEOUT:-600}"
export GODOT_STARTUP_TIMEOUT="${GODOT_STARTUP_TIMEOUT:-60}"

# AI Integration (optional)
export GODOT_AI_ENABLED="${GODOT_AI_ENABLED:-false}"
export GODOT_AI_MODEL="${GODOT_AI_MODEL:-codellama}"
export GODOT_OLLAMA_URL="${GODOT_OLLAMA_URL:-http://localhost:11434}"

# Health Check Configuration
export GODOT_HEALTH_CHECK_INTERVAL="${GODOT_HEALTH_CHECK_INTERVAL:-30}"
export GODOT_HEALTH_CHECK_TIMEOUT="${GODOT_HEALTH_CHECK_TIMEOUT:-5}"
export GODOT_HEALTH_CHECK_RETRIES="${GODOT_HEALTH_CHECK_RETRIES:-3}"

# Process Management
export GODOT_PROCESS_NAME="${GODOT_PROCESS_NAME:-godot-server}"
export GODOT_PID_FILE="${GODOT_PID_FILE:-${GODOT_BASE_DIR}/godot.pid}"

# Export Templates
export GODOT_EXPORT_TEMPLATES=(
    "HTML5"
    "Linux"
    "Windows Desktop"
    "Android"
    "iOS"
)

# Project Templates
export GODOT_PROJECT_TEMPLATES=(
    "empty:Blank project"
    "2d-platformer:2D platform game"
    "3d-fps:3D first-person shooter"
    "puzzle:Puzzle game template"
    "rpg:RPG game framework"
)

# Validation patterns
export GODOT_PROJECT_NAME_PATTERN="^[a-zA-Z0-9_-]+$"
export GODOT_MAX_PROJECT_NAME_LENGTH=64

# Security Configuration
export GODOT_SANDBOX_ENABLED="${GODOT_SANDBOX_ENABLED:-true}"
export GODOT_MAX_FILE_SIZE="${GODOT_MAX_FILE_SIZE:-100M}"
export GODOT_ALLOWED_EXTENSIONS="${GODOT_ALLOWED_EXTENSIONS:-gd,tscn,tres,png,jpg,ogg,wav,glb,gltf}"

# Logging Configuration
export GODOT_LOG_LEVEL="${GODOT_LOG_LEVEL:-INFO}"
export GODOT_LOG_FILE="${GODOT_LOG_FILE:-${GODOT_LOG_DIR}/godot.log}"
export GODOT_LOG_MAX_SIZE="${GODOT_LOG_MAX_SIZE:-10M}"
export GODOT_LOG_MAX_FILES="${GODOT_LOG_MAX_FILES:-5}"