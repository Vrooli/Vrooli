#!/bin/bash
set -euo pipefail

# CrewAI Status Script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
STATUS_LIB_DIR="${APP_ROOT}/resources/crewai/lib"

# Source core first
source "${STATUS_LIB_DIR}/core.sh"

# Source utilities
source "/home/matthalloran8/Vrooli/scripts/lib/utils/format.sh"
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh"

# Source status-args library
source "${STATUS_LIB_DIR}/../../../lib/status-args.sh"

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
    local fast_mode="${1:-false}"
    if [[ -d "${CREWAI_CREWS_DIR}" ]] && [[ "$fast_mode" == "false" ]]; then
        timeout 3s find "${CREWAI_CREWS_DIR}" -name "*.py" -type f 2>/dev/null | wc -l
    elif [[ -d "${CREWAI_CREWS_DIR}" ]]; then
        echo "N/A"
    else
        echo "0"
    fi
}

# Count agents
count_agents() {
    local fast_mode="${1:-false}"
    if [[ -d "${CREWAI_AGENTS_DIR}" ]] && [[ "$fast_mode" == "false" ]]; then
        timeout 3s find "${CREWAI_AGENTS_DIR}" -name "*.py" -type f 2>/dev/null | wc -l
    elif [[ -d "${CREWAI_AGENTS_DIR}" ]]; then
        echo "N/A"
    else
        echo "0"
    fi
}

# Check LLM configuration
check_llm_config() {
    local fast_mode="${1:-false}"
    if [[ "$fast_mode" == "true" ]]; then
        # Skip expensive external resource check in fast mode
        if [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
            echo "true"
        else
            echo "N/A"
        fi
    elif vrooli resource ollama status --format json 2>/dev/null | grep -q '"healthy":true'; then
        echo "true"
    elif [[ -n "${OPENROUTER_API_KEY:-}" ]]; then
        echo "true"
    else
        echo "false"
    fi
}

# Collect CrewAI status data in format-agnostic structure
crewai::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
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
    
    local crews=$(count_crews "$fast_mode")
    local agents=$(count_agents "$fast_mode")
    local llm_configured=$(check_llm_config "$fast_mode")
    
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
        "crews_dir" "${CREWAI_CREWS_DIR}"
        "agents_dir" "${CREWAI_AGENTS_DIR}"
        "workspace_dir" "${CREWAI_WORKSPACE_DIR}"
    )
    
    if [[ "${running}" == "true" ]]; then
        status_data+=(
            "api_url" "http://localhost:${CREWAI_PORT}"
            "health_url" "http://localhost:${CREWAI_PORT}/health"
        )
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
crewai::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🤖 CrewAI Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install CrewAI, run: resource-crewai install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: ${data[health_message]:-Healthy}"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📦 Version: ${data[version]:-unknown}"
    log::info "   🔌 Port: ${data[port]:-unknown}"
    log::info "   🤖 Crews: ${data[crews]:-0}"
    log::info "   👥 Agents: ${data[agents]:-0}"
    log::info "   🧠 LLM Configured: $([ "${data[llm_configured]:-false}" == "true" ] && echo "Yes" || echo "No")"
    echo
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🌐 Service Endpoints:"
        log::info "   🔌 API: ${data[api_url]:-unknown}"
        log::info "   🏥 Health: ${data[health_url]:-unknown}"
        echo
    fi
    
    log::info "📁 Directories:"
    log::info "   🤖 Crews: ${data[crews_dir]:-unknown}"
    log::info "   👥 Agents: ${data[agents_dir]:-unknown}"
    log::info "   📂 Workspace: ${data[workspace_dir]:-unknown}"
    echo
}

# Main status function using standard wrapper
crewai::status() {
    status::run_standard "crewai" "crewai::status::collect_data" "crewai::status::display_text" "$@"
}

# Legacy main function for backward compatibility
main() {
    crewai::status "$@"
}