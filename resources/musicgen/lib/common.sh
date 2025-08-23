#!/bin/bash
# MusicGen Common Functions

set -euo pipefail

# Get directories
MUSICGEN_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MUSICGEN_BASE_DIR="$(dirname "${MUSICGEN_LIB_DIR}")"
VROOLI_DIR="$(cd "${MUSICGEN_BASE_DIR}/../../../../" && pwd)"

# Source shared utilities
source "${VROOLI_DIR}/scripts/lib/utils/log.sh"
source "${VROOLI_DIR}/scripts/lib/utils/format.sh"
source "${VROOLI_DIR}/scripts/lib/utils/var.sh"
source "${VROOLI_DIR}/scripts/lib/runtimes/docker.sh"

# Constants
export MUSICGEN_CONTAINER_NAME="musicgen"
export MUSICGEN_IMAGE="vrooli/musicgen:latest"
export MUSICGEN_PORT=8765
export MUSICGEN_DATA_DIR="${var_DATA_DIR}/musicgen"
export MUSICGEN_MODELS_DIR="${MUSICGEN_DATA_DIR}/models"
export MUSICGEN_OUTPUT_DIR="${MUSICGEN_DATA_DIR}/outputs"
export MUSICGEN_CONFIG_DIR="${MUSICGEN_DATA_DIR}/config"
export MUSICGEN_INJECT_DIR="${MUSICGEN_DATA_DIR}/inject"

# Help text
musicgen::help() {
    cat << EOF
MusicGen Resource Manager

Usage: resource-musicgen <action> [options]

Actions:
  status          Check MusicGen status
  install         Install MusicGen
  uninstall       Uninstall MusicGen
  start           Start MusicGen service
  stop            Stop MusicGen service
  generate        Generate music from prompt
  list-models     List available models
  inject          Inject music generation tasks
  help            Show this help message

Examples:
  resource-musicgen status
  resource-musicgen generate --prompt "upbeat electronic dance music"
  resource-musicgen list-models

Documentation:
  See scripts/resources/execution/musicgen/README.md for detailed usage
EOF
}