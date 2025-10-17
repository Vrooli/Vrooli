#!/usr/bin/env bash
#
# AutoGen Studio - Default Configuration
# Multi-agent conversation framework for complex task orchestration
#

# Resource identification
AUTOGEN_RESOURCE_NAME="autogen-studio"
AUTOGEN_DESCRIPTION="AutoGen Studio multi-agent conversation framework"

# Network configuration
AUTOGEN_PORT="${AUTOGEN_PORT:-8081}"

# Docker configuration
AUTOGEN_NAME="autogen-studio"
AUTOGEN_IMAGE="microsoft/autogen-studio:latest"
AUTOGEN_FALLBACK_IMAGE="local/autogen-studio:latest"

# Storage configuration
AUTOGEN_DATA_DIR="${HOME}/.autogen-studio"
AUTOGEN_WORKSPACE_DIR="${AUTOGEN_DATA_DIR}/workspace"
AUTOGEN_AGENTS_DIR="${AUTOGEN_DATA_DIR}/agents"
AUTOGEN_SKILLS_DIR="${AUTOGEN_DATA_DIR}/skills"
AUTOGEN_CONFIG_FILE="${AUTOGEN_DATA_DIR}/config.json"

# Export all variables
export AUTOGEN_RESOURCE_NAME AUTOGEN_DESCRIPTION
export AUTOGEN_PORT AUTOGEN_NAME AUTOGEN_IMAGE AUTOGEN_FALLBACK_IMAGE
export AUTOGEN_DATA_DIR AUTOGEN_WORKSPACE_DIR AUTOGEN_AGENTS_DIR AUTOGEN_SKILLS_DIR AUTOGEN_CONFIG_FILE