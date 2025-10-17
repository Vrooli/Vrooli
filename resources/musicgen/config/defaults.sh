#!/usr/bin/env bash
################################################################################
# MusicGen Configuration Defaults
################################################################################

# MusicGen directories
export MUSICGEN_DATA_DIR="${MUSICGEN_DATA_DIR:-${APP_ROOT}/data/resources/musicgen}"
export MUSICGEN_MODELS_DIR="${MUSICGEN_MODELS_DIR:-${MUSICGEN_DATA_DIR}/models}"
export MUSICGEN_OUTPUT_DIR="${MUSICGEN_OUTPUT_DIR:-${MUSICGEN_DATA_DIR}/output}"
export MUSICGEN_CONFIG_DIR="${MUSICGEN_CONFIG_DIR:-${MUSICGEN_DATA_DIR}/config}"
export MUSICGEN_INJECT_DIR="${MUSICGEN_INJECT_DIR:-${MUSICGEN_DATA_DIR}/inject}"

# MusicGen network settings
export MUSICGEN_PORT="${MUSICGEN_PORT:-7860}"
export MUSICGEN_HOST="${MUSICGEN_HOST:-0.0.0.0}"

# Container settings
export MUSICGEN_CONTAINER_NAME="${MUSICGEN_CONTAINER_NAME:-musicgen}"
export MUSICGEN_IMAGE_NAME="${MUSICGEN_IMAGE_NAME:-musicgen:latest}"