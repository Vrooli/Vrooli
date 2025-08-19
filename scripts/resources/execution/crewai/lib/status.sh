#!/bin/bash
set -euo pipefail

# CrewAI Status Script
STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source core first
source "${STATUS_LIB_DIR}/core.sh"

# Source utilities
source "/home/matthalloran8/Vrooli/scripts/lib/utils/format.sh"
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh"

# Check installation
check_installation() {
    # In mock mode, just check if server file exists
    if [[ -f "${CREWAI_SERVER_FILE}" ]]; then
        return 0
    fi
    return 1
}

# Get CrewAI version
get_version() {
    if check_installation; then
        echo "1.0.0-mock"
    else
        echo "not installed"
    fi
}

# Count crews
count_crews() {
    if [[ -d "${CREWAI_CREWS_DIR}" ]]; then
        find "${CREWAI_CREWS_DIR}" -name "*.py" -type f 2>/dev/null | wc -l
    else
        echo "0"
    fi
}

# Count agents
count_agents() {
    if [[ -d "${CREWAI_AGENTS_DIR}" ]]; then
        find "${CREWAI_AGENTS_DIR}" -name "*.py" -type f 2>/dev/null | wc -l
    else
        echo "0"
    fi
}

# Check LLM configuration
check_llm_config() {
    # Check if Ollama is available
    if vrooli resource ollama status --format json 2>/dev/null | grep -q '"healthy":true'; then
        echo "true"
    elif [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
        echo "true"
    else
        echo "false"
    fi
}

# Main status check
main() {
    local format_type="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                if [[ $# -gt 1 ]]; then
                    format_type="$2"
                    shift 2
                else
                    shift
                fi
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Gather status information
    local installed="false"
    local running="false"
    local health="not_installed"
    local health_msg="Not installed - Use 'resource-crewai install'"
    local version="not installed"
    
    if check_installation; then
        installed="true"
        version=$(get_version)
        
        if is_running; then
            running="true"
            health="healthy"
            health_msg="Healthy - CrewAI service is operational"
        else
            health="stopped"
            health_msg="Installed but not running"
        fi
    fi
    
    local crews=$(count_crews)
    local agents=$(count_agents)
    local llm_configured=$(check_llm_config)
    
    # Build status data
    local status_data=(
        "name" "crewai"
        "category" "execution"
        "description" "Framework for orchestrating role-playing autonomous AI agents"
        "installed" "${installed}"
        "running" "${running}"
        "healthy" "$([ "$health" == "healthy" ] && echo "true" || echo "false")"
        "health_message" "${health_msg}"
        "version" "${version}"
        "port" "${CREWAI_PORT}"
        "crews" "${crews}"
        "agents" "${agents}"
        "llm_configured" "${llm_configured}"
    )
    
    if [[ "${running}" == "true" ]]; then
        status_data+=(
            "api_url" "http://localhost:${CREWAI_PORT}"
            "health_url" "http://localhost:${CREWAI_PORT}/health"
        )
    fi
    
    status_data+=(
        "crews_dir" "${CREWAI_CREWS_DIR}"
        "agents_dir" "${CREWAI_AGENTS_DIR}"
        "workspace_dir" "${CREWAI_WORKSPACE_DIR}"
    )
    
    # Output using format library
    if [[ "${format_type}" == "json" ]]; then
        format::output json kv "${status_data[@]}"
    else
        # Human-readable output
        echo "[HEADER]  🤖 CrewAI Status"
        echo "[INFO]    📝 Description: Framework for orchestrating role-playing autonomous AI agents"
        echo "[INFO]    📂 Category: execution"
        echo ""
        echo "[INFO]    📊 Basic Status:"
        if [[ "${installed}" == "true" ]]; then
            echo "[SUCCESS]    ✅ Installed: Yes"
        else
            echo "[ERROR]      ❌ Installed: No"
        fi
        
        if [[ "${running}" == "true" ]]; then
            echo "[SUCCESS]    ✅ Running: Yes"
        else
            echo "[ERROR]      ❌ Running: No"
        fi
        
        if [[ "${health}" == "healthy" ]]; then
            echo "[SUCCESS]    ✅ Health: ${health_msg}"
        else
            echo "[WARNING]    ⚠️  Health: ${health_msg}"
        fi
        
        echo ""
        echo "[INFO]    📦 Version: ${version}"
        echo "[INFO]    🔌 Port: ${CREWAI_PORT}"
        echo "[INFO]    🤖 Crews: ${crews}"
        echo "[INFO]    👥 Agents: ${agents}"
        echo "[INFO]    🧠 LLM Configured: $([ "${llm_configured}" == "true" ] && echo "Yes" || echo "No")"
        
        if [[ "${running}" == "true" ]]; then
            echo ""
            echo "[INFO]    🌐 Service Endpoints:"
            echo "[INFO]       🔌 API: http://localhost:${CREWAI_PORT}"
            echo "[INFO]       🏥 Health: http://localhost:${CREWAI_PORT}/health"
        fi
        
        echo ""
        echo "[INFO]    📁 Directories:"
        echo "[INFO]       🤖 Crews: ${CREWAI_CREWS_DIR}"
        echo "[INFO]       👥 Agents: ${CREWAI_AGENTS_DIR}"
        echo "[INFO]       📂 Workspace: ${CREWAI_WORKSPACE_DIR}"
    fi
}