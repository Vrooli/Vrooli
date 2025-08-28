#!/bin/bash

# AutoGen Studio Core Functions
# Multi-agent conversation framework for complex task orchestration

set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AUTOGEN_LIB_DIR="${APP_ROOT}/resources/autogen-studio/lib"
AUTOGEN_DIR="${APP_ROOT}/resources/autogen-studio"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/scripts/lib/utils/docker-utils.sh"
source "${APP_ROOT}/scripts/lib/utils/wait-utils.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Helper function for formatting status
format_status() {
    local label="$1"
    local value="$2"
    if [[ "${value}" == "true" ]] || [[ "${value}" == "healthy" ]] || [[ "${value}" == "running" ]]; then
        log::success "✅ ${label}: ${value}"
    else
        log::error "❌ ${label}: ${value}"
    fi
}

# Configuration
AUTOGEN_NAME="autogen-studio"
AUTOGEN_PORT="${AUTOGEN_PORT:-8081}"
AUTOGEN_IMAGE="microsoft/autogen-studio:latest"
AUTOGEN_DATA_DIR="${HOME}/.autogen-studio"
AUTOGEN_WORKSPACE_DIR="${AUTOGEN_DATA_DIR}/workspace"
AUTOGEN_AGENTS_DIR="${AUTOGEN_DATA_DIR}/agents"
AUTOGEN_SKILLS_DIR="${AUTOGEN_DATA_DIR}/skills"
AUTOGEN_CONFIG_FILE="${AUTOGEN_DATA_DIR}/config.json"

# Initialize function
autogen_init() {
    log::info "Initializing AutoGen Studio..."
    
    # Create data directories
    mkdir -p "${AUTOGEN_WORKSPACE_DIR}"
    mkdir -p "${AUTOGEN_AGENTS_DIR}"
    mkdir -p "${AUTOGEN_SKILLS_DIR}"
    
    # Create default config if not exists
    if [[ ! -f "${AUTOGEN_CONFIG_FILE}" ]]; then
        cat > "${AUTOGEN_CONFIG_FILE}" << 'EOF'
{
    "llm_configs": [
        {
            "name": "ollama",
            "type": "ollama",
            "base_url": "http://host.docker.internal:11434",
            "model": "qwen2.5:3b"
        },
        {
            "name": "openrouter",
            "type": "openai",
            "base_url": "https://openrouter.ai/api/v1",
            "api_key": "${OPENROUTER_API_KEY}",
            "model": "openai/gpt-3.5-turbo"
        }
    ],
    "workspace_path": "/workspace",
    "agents_path": "/agents",
    "skills_path": "/skills"
}
EOF
        log::success "Created default configuration"
    fi
    
    return 0
}

# Install function
autogen_install() {
    log::header "Installing AutoGen Studio"
    
    # Initialize first
    autogen_init
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        log::error "Docker is required for AutoGen Studio"
        return 1
    fi
    
    # Pull the image
    log::info "Pulling AutoGen Studio Docker image..."
    if docker pull "${AUTOGEN_IMAGE}" 2>/dev/null; then
        log::success "AutoGen Studio image pulled successfully"
    else
        # If official image doesn't exist, use Python with autogen-studio
        log::info "Building custom AutoGen Studio image..."
        
        # Create Dockerfile
        cat > "${AUTOGEN_DATA_DIR}/Dockerfile" << 'EOF'
FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir autogenstudio

EXPOSE 8081

CMD ["autogenstudio", "ui", "--host", "0.0.0.0", "--port", "8081"]
EOF
        
        # Build the image
        if docker build -t "local/autogen-studio:latest" "${AUTOGEN_DATA_DIR}"; then
            AUTOGEN_IMAGE="local/autogen-studio:latest"
            log::success "Custom AutoGen Studio image built"
        else
            log::error "Failed to build AutoGen Studio image"
            return 1
        fi
    fi
    
    log::success "AutoGen Studio installed successfully"
    return 0
}

# Start function
autogen_start() {
    log::header "Starting AutoGen Studio"
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        log::info "AutoGen Studio is already running"
        return 0
    fi
    
    # Ensure initialized
    autogen_init
    
    # Get API keys if available
    local env_flags=""
    if [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
        local api_key=$(jq -r '.data.apiKey // empty' "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" 2>/dev/null)
        if [[ -n "${api_key}" ]]; then
            env_flags="-e OPENROUTER_API_KEY=${api_key}"
        fi
    fi
    
    # Start container
    log::info "Starting AutoGen Studio container..."
    if docker run -d \
        --name "${AUTOGEN_NAME}" \
        --restart unless-stopped \
        -p "${AUTOGEN_PORT}:8081" \
        -v "${AUTOGEN_WORKSPACE_DIR}:/workspace" \
        -v "${AUTOGEN_AGENTS_DIR}:/agents" \
        -v "${AUTOGEN_SKILLS_DIR}:/skills" \
        -v "${AUTOGEN_CONFIG_FILE}:/app/config.json" \
        ${env_flags} \
        --network host \
        "${AUTOGEN_IMAGE}" > /dev/null 2>&1; then
        
        log::info "Waiting for AutoGen Studio to be ready..."
        if wait_for_http "localhost:${AUTOGEN_PORT}" 60; then
            log::success "AutoGen Studio started successfully"
            log::info "AutoGen Studio UI available at: http://localhost:${AUTOGEN_PORT}"
            return 0
        else
            log::error "AutoGen Studio failed to become ready"
            docker logs "${AUTOGEN_NAME}" 2>&1 | tail -20
            return 1
        fi
    else
        log::error "Failed to start AutoGen Studio container"
        return 1
    fi
}

# Stop function
autogen_stop() {
    log::header "Stopping AutoGen Studio"
    
    if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        if docker stop "${AUTOGEN_NAME}" > /dev/null 2>&1; then
            docker rm "${AUTOGEN_NAME}" > /dev/null 2>&1
            log::success "AutoGen Studio stopped"
        else
            log::error "Failed to stop AutoGen Studio"
            return 1
        fi
    else
        log::info "AutoGen Studio is not running"
    fi
    
    return 0
}

# Status function
autogen_status() {
    local format="${1:-text}"
    local verbose="${2:-false}"
    
    # Collect status information
    local installed="false"
    local running="false"
    local health="unhealthy"
    local version="unknown"
    local agents_count=0
    local skills_count=0
    local llm_configured="false"
    
    # Check if installed
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -qE "(autogen-studio|microsoft/autogen)"; then
        installed="true"
    fi
    
    # Check if running
    if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        running="true"
        
        # Check health
        if curl -s -f "http://localhost:${AUTOGEN_PORT}" > /dev/null 2>&1; then
            health="healthy"
        fi
        
        # Get version
        version=$(docker exec "${AUTOGEN_NAME}" python -c "import autogenstudio; print(autogenstudio.__version__)" 2>/dev/null || echo "0.1.0")
    fi
    
    # Count agents and skills
    if [[ -d "${AUTOGEN_AGENTS_DIR}" ]]; then
        agents_count=$(find "${AUTOGEN_AGENTS_DIR}" -name "*.json" 2>/dev/null | wc -l)
    fi
    if [[ -d "${AUTOGEN_SKILLS_DIR}" ]]; then
        skills_count=$(find "${AUTOGEN_SKILLS_DIR}" -name "*.py" 2>/dev/null | wc -l)
    fi
    
    # Check LLM configuration
    if [[ -f "${AUTOGEN_CONFIG_FILE}" ]] && [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
        llm_configured="true"
    elif docker ps --format "{{.Names}}" | grep -q "^ollama$"; then
        llm_configured="true"
    fi
    
    # Use format utility for output
    if [[ "${format}" == "json" ]]; then
        format::output json kv \
            name "${AUTOGEN_NAME}" \
            installed "${installed}" \
            running "${running}" \
            health "${health}" \
            version "${version}" \
            port "${AUTOGEN_PORT}" \
            agents "${agents_count}" \
            skills "${skills_count}" \
            llm_configured "${llm_configured}" \
            ui_url "http://localhost:${AUTOGEN_PORT}"
    else
        log::header "AutoGen Studio Status"
        format_status "Installed" "${installed}"
        format_status "Running" "${running}"
        format_status "Health" "${health}"
        
        if [[ "${verbose}" == "true" ]] || [[ "${running}" == "true" ]]; then
            log::info "Details:"
            echo "  Version: ${version}"
            echo "  Port: ${AUTOGEN_PORT}"
            echo "  Agents: ${agents_count}"
            echo "  Skills: ${skills_count}"
            echo "  LLM Configured: ${llm_configured}"
            if [[ "${running}" == "true" ]]; then
                echo "  UI URL: http://localhost:${AUTOGEN_PORT}"
            fi
        fi
    fi
    
    return 0
}

# Create agent function
autogen_create_agent() {
    local name="${1}"
    local type="${2:-assistant}"
    local system_message="${3:-You are a helpful AI assistant.}"
    
    if [[ -z "${name}" ]]; then
        log::error "Agent name is required"
        return 1
    fi
    
    local agent_file="${AUTOGEN_AGENTS_DIR}/${name}.json"
    
    cat > "${agent_file}" << EOF
{
    "name": "${name}",
    "type": "${type}",
    "system_message": "${system_message}",
    "llm_config": "ollama",
    "max_consecutive_auto_reply": 10,
    "human_input_mode": "NEVER"
}
EOF
    
    log::success "Created agent: ${name}"
    return 0
}

# Create skill function
autogen_create_skill() {
    local name="${1}"
    local code="${2}"
    
    if [[ -z "${name}" ]]; then
        log::error "Skill name is required"
        return 1
    fi
    
    local skill_file="${AUTOGEN_SKILLS_DIR}/${name}.py"
    
    if [[ -n "${code}" ]]; then
        echo "${code}" > "${skill_file}"
    else
        # Create a template skill
        cat > "${skill_file}" << 'EOF'
def main(input_data):
    """
    AutoGen skill template
    """
    result = f"Processed: {input_data}"
    return result
EOF
    fi
    
    log::success "Created skill: ${name}"
    return 0
}

# List agents function
autogen_list_agents() {
    log::header "Available Agents"
    
    if [[ -d "${AUTOGEN_AGENTS_DIR}" ]]; then
        find "${AUTOGEN_AGENTS_DIR}" -name "*.json" -type f | while read -r agent_file; do
            local agent_name=$(basename "${agent_file}" .json)
            local agent_type=$(jq -r '.type // "unknown"' "${agent_file}" 2>/dev/null)
            echo "  • ${agent_name} (${agent_type})"
        done
    else
        log::info "No agents found"
    fi
    
    return 0
}

# List skills function
autogen_list_skills() {
    log::header "Available Skills"
    
    if [[ -d "${AUTOGEN_SKILLS_DIR}" ]]; then
        find "${AUTOGEN_SKILLS_DIR}" -name "*.py" -type f | while read -r skill_file; do
            local skill_name=$(basename "${skill_file}" .py)
            echo "  • ${skill_name}"
        done
    else
        log::info "No skills found"
    fi
    
    return 0
}

# Export functions
export -f autogen_init
export -f autogen_install
export -f autogen_start
export -f autogen_stop
export -f autogen_status
export -f autogen_create_agent
export -f autogen_create_skill
export -f autogen_list_agents
export -f autogen_list_skills

# Inject function - install agent or skill from file
autogen_inject() {
    local file="${1:-}"
    
    # Handle help request
    if [[ "${file}" == "--help" ]] || [[ "${file}" == "-h" ]]; then
        cat << EOF
Usage: inject FILE

Inject an agent or skill definition from a file into AutoGen Studio.

Arguments:
  FILE    Path to JSON file containing agent definition or Python skill file

The command automatically detects whether the file is an agent or skill based on:
  - Filename containing 'agent' or 'skill'
  - JSON structure (agents have 'system_message' field)
  - File extension (.py files are treated as skills)

Examples:
  inject agents/researcher.json
  inject skills/data_processor.py
EOF
        return 0
    fi
    
    if [[ -z "${file}" ]]; then
        log::error "File path is required"
        return 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        log::error "File not found: ${file}"
        return 1
    fi
    
    local filename=$(basename "${file}")
    local type=""
    local dest_dir=""
    
    # Determine type based on content or extension
    if [[ "${filename}" == *"agent"* ]] || grep -q '"type".*"assistant"' "${file}" 2>/dev/null; then
        type="agent"
        dest_dir="${AUTOGEN_AGENTS_DIR}"
    elif [[ "${filename}" == *"skill"* ]] || [[ "${filename}" == *.py ]]; then
        type="skill"
        dest_dir="${AUTOGEN_SKILLS_DIR}"
    else
        # Try to detect from JSON structure
        if grep -q '"system_message"' "${file}" 2>/dev/null; then
            type="agent"
            dest_dir="${AUTOGEN_AGENTS_DIR}"
        else
            type="skill"
            dest_dir="${AUTOGEN_SKILLS_DIR}"
        fi
    fi
    
    # Ensure directory exists
    mkdir -p "${dest_dir}"
    
    # Copy file to appropriate directory
    local dest_file="${dest_dir}/${filename}"
    if cp "${file}" "${dest_file}"; then
        log::success "Injected ${type}: ${filename}"
        
        # If container is running, reload it
        if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
            log::info "Reloading AutoGen Studio to apply changes..."
            # Try to reload using Python signal or just restart the service
            docker exec "${AUTOGEN_NAME}" python -c "import os, signal; os.kill(1, signal.SIGHUP)" 2>/dev/null || \
                docker restart "${AUTOGEN_NAME}" 2>/dev/null || true
        fi
        
        return 0
    else
        log::error "Failed to inject ${type}"
        return 1
    fi
}

export -f autogen_inject

# ==============================================================================
# v2.0 Universal Contract Wrapper Functions
# ==============================================================================

# Install/Uninstall handlers
autogen::install::execute() {
    autogen_install "$@"
}

autogen::install::uninstall() {
    log::header "Uninstalling AutoGen Studio"
    
    # Stop if running
    if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        autogen_stop
    fi
    
    # Remove Docker images
    docker rmi "${AUTOGEN_IMAGE}" "${AUTOGEN_FALLBACK_IMAGE}" 2>/dev/null || true
    
    # Remove data directory (with confirmation)
    if [[ -d "${AUTOGEN_DATA_DIR}" ]]; then
        log::warning "Remove data directory ${AUTOGEN_DATA_DIR}? [y/N]"
        read -r response
        if [[ "${response,,}" == "y" ]]; then
            rm -rf "${AUTOGEN_DATA_DIR}"
            log::success "Data directory removed"
        fi
    fi
    
    log::success "AutoGen Studio uninstalled"
}

# Docker lifecycle handlers  
autogen::docker::start() {
    autogen_start "$@"
}

autogen::docker::stop() {
    autogen_stop "$@"
}

autogen::docker::restart() {
    autogen_stop "$@" && autogen_start "$@"
}

autogen::docker::logs() {
    if docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        docker logs "${AUTOGEN_NAME}" "$@"
    else
        log::error "AutoGen Studio container is not running"
        return 1
    fi
}

# Status and test handlers
autogen::status() {
    autogen_status "$@"
}

autogen::status::check() {
    # Smoke test - basic health check
    log::header "AutoGen Studio Smoke Test"
    
    # Check if installed
    if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -qE "(autogen-studio|microsoft/autogen)"; then
        log::error "AutoGen Studio is not installed"
        return 1
    fi
    
    # Check if running
    if ! docker ps --format "{{.Names}}" | grep -q "^${AUTOGEN_NAME}$"; then
        log::warning "AutoGen Studio is not running"
        return 0  # Not an error for smoke test
    fi
    
    # Check if responding
    if curl -s -f "http://localhost:${AUTOGEN_PORT}" > /dev/null 2>&1; then
        log::success "AutoGen Studio is healthy"
        return 0
    else
        log::error "AutoGen Studio is not responding"
        return 1
    fi
}

# Content handlers for business functionality
autogen::content::add() {
    local type="${1:-agent}"
    local name="${2:-}"
    
    case "${type}" in
        agent)
            if [[ -z "${name}" ]]; then
                log::error "Agent name is required. Usage: content add agent <name> [type] [message]"
                return 1
            fi
            autogen_create_agent "${name}" "${3:-assistant}" "${4:-You are a helpful AI assistant.}"
            ;;
        skill)
            if [[ -z "${name}" ]]; then
                log::error "Skill name is required. Usage: content add skill <name> [code]"
                return 1
            fi
            autogen_create_skill "${name}" "${3:-}"
            ;;
        *)
            log::error "Invalid type. Use 'agent' or 'skill'"
            return 1
            ;;
    esac
}

autogen::content::list() {
    local type="${1:-all}"
    
    case "${type}" in
        agent|agents)
            autogen_list_agents
            ;;
        skill|skills)
            autogen_list_skills
            ;;
        all|*)
            autogen_list_agents
            echo ""
            autogen_list_skills
            ;;
    esac
}

autogen::content::get() {
    local type="${1}"
    local name="${2}"
    
    if [[ -z "${type}" || -z "${name}" ]]; then
        log::error "Usage: content get <agent|skill> <name>"
        return 1
    fi
    
    case "${type}" in
        agent)
            local agent_file="${AUTOGEN_AGENTS_DIR}/${name}.json"
            if [[ -f "${agent_file}" ]]; then
                cat "${agent_file}"
            else
                log::error "Agent '${name}' not found"
                return 1
            fi
            ;;
        skill)
            local skill_file="${AUTOGEN_SKILLS_DIR}/${name}.py"
            if [[ -f "${skill_file}" ]]; then
                cat "${skill_file}"
            else
                log::error "Skill '${name}' not found"
                return 1
            fi
            ;;
        *)
            log::error "Invalid type. Use 'agent' or 'skill'"
            return 1
            ;;
    esac
}

autogen::content::remove() {
    local type="${1}"
    local name="${2}"
    
    if [[ -z "${type}" || -z "${name}" ]]; then
        log::error "Usage: content remove <agent|skill> <name>"
        return 1
    fi
    
    case "${type}" in
        agent)
            local agent_file="${AUTOGEN_AGENTS_DIR}/${name}.json"
            if [[ -f "${agent_file}" ]]; then
                rm "${agent_file}"
                log::success "Agent '${name}' removed"
            else
                log::error "Agent '${name}' not found"
                return 1
            fi
            ;;
        skill)
            local skill_file="${AUTOGEN_SKILLS_DIR}/${name}.py"
            if [[ -f "${skill_file}" ]]; then
                rm "${skill_file}"
                log::success "Skill '${name}' removed"
            else
                log::error "Skill '${name}' not found"
                return 1
            fi
            ;;
        *)
            log::error "Invalid type. Use 'agent' or 'skill'"
            return 1
            ;;
    esac
}

autogen::content::execute() {
    # Business functionality - execute agents/workflows
    log::error "Content execution not yet implemented for AutoGen Studio"
    log::info "Use the web UI at http://localhost:${AUTOGEN_PORT} to run agent conversations"
    return 1
}

# Export all v2.0 wrapper functions
export -f autogen::install::execute
export -f autogen::install::uninstall
export -f autogen::docker::start
export -f autogen::docker::stop
export -f autogen::docker::restart
export -f autogen::docker::logs
export -f autogen::status
export -f autogen::status::check
export -f autogen::content::add
export -f autogen::content::list
export -f autogen::content::get
export -f autogen::content::remove
export -f autogen::content::execute
EOF < /dev/null
