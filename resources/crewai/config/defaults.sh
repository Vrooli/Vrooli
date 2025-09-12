#!/usr/bin/env bash
# CrewAI Resource Configuration

# Get port from registry
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREWAI_PORT=$("${APP_ROOT}/scripts/resources/port_registry.sh" crewai | grep -E "crewai\s+:" | awk '{print $3}')

# CrewAI resource defaults
export CREWAI_NAME="crewai"
export CREWAI_DISPLAY_NAME="CrewAI"
export CREWAI_DATA_DIR="${HOME}/.crewai"
export CREWAI_PORT="${CREWAI_PORT:-8084}"
export CREWAI_MOCK_MODE="true"

# Directories
export CREWAI_WORKSPACE_DIR="${CREWAI_DATA_DIR}/workspace"
export CREWAI_CREWS_DIR="${CREWAI_DATA_DIR}/crews"
export CREWAI_AGENTS_DIR="${CREWAI_DATA_DIR}/agents"
export CREWAI_PID_FILE="${CREWAI_DATA_DIR}/crewai.pid"
export CREWAI_LOG_FILE="${CREWAI_DATA_DIR}/crewai.log"
export CREWAI_SERVER_FILE="${CREWAI_DATA_DIR}/server.py"