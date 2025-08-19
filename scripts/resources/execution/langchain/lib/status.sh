#!/usr/bin/env bash
# LangChain Status Monitoring
# Functions for checking LangChain health and status

# Source format utilities and required libraries
LANGCHAIN_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/../../../../lib/utils/log.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${LANGCHAIN_STATUS_DIR}/core.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Collect LangChain status data
#######################################
langchain::status::collect_data() {
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
        
        # Package counts
        local chain_count=0
        local agent_count=0
        
        if [[ -d "$LANGCHAIN_CHAINS_DIR" ]]; then
            chain_count=$(find "$LANGCHAIN_CHAINS_DIR" -name "*.py" 2>/dev/null | wc -l)
        fi
        
        if [[ -d "$LANGCHAIN_AGENTS_DIR" ]]; then
            agent_count=$(find "$LANGCHAIN_AGENTS_DIR" -name "*.py" 2>/dev/null | wc -l)
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
    
    # Output the data
    printf "%s\n" "${status_data[@]}"
}

#######################################
# Display status in text format
#######################################
langchain::status::display_text() {
    local data=("$@")
    
    # Parse data into associative array
    declare -A status_map
    for item in "${data[@]}"; do
        if [[ "$item" =~ ^([^:]+):(.*)$ ]]; then
            status_map["${BASH_REMATCH[1]}"]="${BASH_REMATCH[2]}"
        fi
    done
    
    log::header "LangChain Status"
    
    log::info "📊 Basic Status:"
    
    if [[ "${status_map[installed]}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
    fi
    
    if [[ "${status_map[running]}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${status_map[healthy]}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${status_map[health_message]}"
    fi
    
    if [[ "${status_map[installed]}" == "true" ]]; then
        echo ""
        log::info "📦 Framework Info:"
        log::info "   📚 Version: ${status_map[version]:-unknown}"
        log::info "   🔗 Chains: ${status_map[chain_count]:-0}"
        log::info "   🤖 Agents: ${status_map[agent_count]:-0}"
        
        echo ""
        log::info "🔌 Integration Status:"
        [[ "${status_map[ollama_enabled]}" == "true" ]] && log::info "   ✅ Ollama: Enabled" || log::info "   ❌ Ollama: Disabled"
        [[ "${status_map[openrouter_enabled]}" == "true" ]] && log::info "   ✅ OpenRouter: Enabled" || log::info "   ❌ OpenRouter: Disabled"
        [[ "${status_map[vectordb_enabled]}" == "true" ]] && log::info "   ✅ VectorDB: Enabled" || log::info "   ❌ VectorDB: Disabled"
    fi
    
    echo ""
    log::info "📁 Directories:"
    log::info "   📂 Workspace: ${status_map[workspace_dir]}"
    log::info "   🔗 Chains: ${status_map[chains_dir]}"
    log::info "   🤖 Agents: ${status_map[agents_dir]}"
}

#######################################
# Main status function with format support
#######################################
langchain::status() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data
    local data_string
    data_string=$(langchain::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect LangChain status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        langchain::status::display_text "${data_array[@]}"
    fi
}

# Make status available as default
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    langchain::status "$@"
fi