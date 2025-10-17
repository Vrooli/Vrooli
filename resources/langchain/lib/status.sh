#!/usr/bin/env bash
# LangChain Status Monitoring
# Functions for checking LangChain health and status

# Source format utilities and required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LANGCHAIN_STATUS_DIR="${APP_ROOT}/resources/langchain/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/langchain/config/defaults.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/core.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Collect LangChain status data
# Args: [--fast] - Skip expensive operations for faster response
#######################################
langchain::status::collect_data() {
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
    
    local status_data=()
    
    # Basic information
    status_data+=("name:$LANGCHAIN_RESOURCE_NAME")
    status_data+=("category:$LANGCHAIN_RESOURCE_CATEGORY")
    status_data+=("description:$LANGCHAIN_RESOURCE_DESCRIPTION")
    
    # Installation status
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="Unknown"
    
    if langchain::is_installed; then
        installed="true"
    fi
    
    if langchain::is_running; then
        running="true"
    fi
    
    local status
    status=$(langchain::get_status)
    
    case "$status" in
        "healthy")
            healthy="true"
            health_message="Healthy - Framework operational"
            ;;
        "unhealthy")
            health_message="Unhealthy - Check packages and services"
            ;;
        "stopped")
            health_message="Stopped - No active processes"
            ;;
        "not_installed")
            health_message="Not installed - Run install command"
            ;;
    esac
    
    status_data+=("installed:$installed")
    status_data+=("running:$running")
    status_data+=("healthy:$healthy")
    status_data+=("status:$status")
    status_data+=("health_message:$health_message")
    
    # Environment information
    status_data+=("venv_dir:$LANGCHAIN_VENV_DIR")
    status_data+=("workspace_dir:$LANGCHAIN_WORKSPACE_DIR")
    status_data+=("chains_dir:$LANGCHAIN_CHAINS_DIR")
    status_data+=("agents_dir:$LANGCHAIN_AGENTS_DIR")
    
    # Version information
    if [[ "$installed" == "true" ]]; then
        local version
        version=$(langchain::get_version 2>/dev/null || echo "unknown")
        status_data+=("version:$version")
        
        # Package counts (skip expensive operations in fast mode)
        local chain_count agent_count
        
        if [[ "$fast_mode" == "true" ]]; then
            chain_count="N/A"
            agent_count="N/A"
        else
            chain_count=0
            agent_count=0
            
            if [[ -d "$LANGCHAIN_CHAINS_DIR" ]]; then
                chain_count=$(find "$LANGCHAIN_CHAINS_DIR" -name "*.py" 2>/dev/null | wc -l)
            fi
            
            if [[ -d "$LANGCHAIN_AGENTS_DIR" ]]; then
                agent_count=$(find "$LANGCHAIN_AGENTS_DIR" -name "*.py" 2>/dev/null | wc -l)
            fi
        fi
        
        status_data+=("chain_count:$chain_count")
        status_data+=("agent_count:$agent_count")
    fi
    
    # Integration status
    status_data+=("ollama_enabled:$LANGCHAIN_ENABLE_OLLAMA")
    status_data+=("openrouter_enabled:$LANGCHAIN_ENABLE_OPENROUTER")
    status_data+=("vectordb_enabled:$LANGCHAIN_ENABLE_VECTORDB")
    
    # API endpoints
    status_data+=("api_url:$LANGCHAIN_API_URL")
    status_data+=("base_url:$LANGCHAIN_BASE_URL")
    
    # Convert to standard format (key-value pairs)
    local output_data=()
    for item in "${status_data[@]}"; do
        if [[ "$item" =~ ^([^:]+):(.*)$ ]]; then
            output_data+=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}")
        fi
    done
    
    # Return the collected data
    printf '%s\n' "${output_data[@]}"
}

#######################################
# Display status in text format
#######################################
langchain::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    log::header "🤖 LangChain Status"
    echo
    
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install LangChain, run: resource-langchain install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    log::info "📦 Framework Info:"
    log::info "   📚 Version: ${data[version]:-unknown}"
    log::info "   🔗 Chains: ${data[chain_count]:-0}"
    log::info "   🤖 Agents: ${data[agent_count]:-0}"
    echo
    
    log::info "🔌 Integration Status:"
    [[ "${data[ollama_enabled]:-false}" == "true" ]] && log::info "   ✅ Ollama: Enabled" || log::info "   ❌ Ollama: Disabled"
    [[ "${data[openrouter_enabled]:-false}" == "true" ]] && log::info "   ✅ OpenRouter: Enabled" || log::info "   ❌ OpenRouter: Disabled"
    [[ "${data[vectordb_enabled]:-false}" == "true" ]] && log::info "   ✅ VectorDB: Enabled" || log::info "   ❌ VectorDB: Disabled"
    echo
    
    log::info "📁 Directories:"
    log::info "   📂 Workspace: ${data[workspace_dir]:-unknown}"
    log::info "   🔗 Chains: ${data[chains_dir]:-unknown}"
    log::info "   🤖 Agents: ${data[agents_dir]:-unknown}"
    echo
}

#######################################
# Main status function using standard wrapper
#######################################
langchain::status() {
    status::run_standard "langchain" "langchain::status::collect_data" "langchain::status::display_text" "$@"
}

# Make status available as default
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    langchain::status "$@"
fi