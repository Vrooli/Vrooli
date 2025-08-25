#!/usr/bin/env bash
# Cline Status Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

#######################################
# Collect Cline status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
cline::status::collect_data() {
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
    
    # Basic status checks
    local status="stopped"
    local health="unhealthy"
    local version="not installed"
    local provider="${CLINE_DEFAULT_PROVIDER}"
    local api_configured="false"
    local vscode_available="false"
    local details=""
    local running="false"
    local healthy="false"
    
    # Check status file first
    if [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
        local file_status=$(cat "$CLINE_CONFIG_DIR/.status" 2>/dev/null || echo "stopped")
        if [[ "$file_status" == "running" ]]; then
            status="running"
            running="true"
        fi
    fi
    
    # Check VS Code availability (skip if fast mode)
    if [[ "$fast_mode" == "false" ]]; then
        if cline::check_vscode; then
            vscode_available="true"
        else
            details="VS Code not found"
        fi
        
        # Check configuration regardless of VS Code presence
        if cline::is_configured; then
            api_configured="true"
        fi
    else
        # Fast mode - skip expensive checks
        vscode_available="N/A"
        api_configured="N/A"
        details="Fast mode - skipped checks"
    fi
    
    # Determine health based on configuration and VS Code availability
    if [[ "$fast_mode" == "true" ]]; then
        # Fast mode - assume healthy if status file exists
        if [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
            health="healthy"
            healthy="true"
            version="N/A"
        else
            health="partial"
            version="N/A"
        fi
    elif [[ "$vscode_available" == "true" ]]; then
        # VS Code is available
        if cline::is_installed; then
            version=$(cline::get_version)
            if [[ "$api_configured" == "true" ]]; then
                health="healthy"
                healthy="true"
                details="Cline is ready for use in VS Code"
            else
                health="partial"
                details="Cline installed but not configured"
            fi
        else
            health="partial"
            details="Cline extension not installed"
        fi
    else
        # VS Code not available but check if config is ready
        if [[ -f "$CLINE_CONFIG_DIR/.status" && "$api_configured" == "true" ]]; then
            health="partial"
            details="Cline configured, awaiting VS Code installation"
        elif [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
            health="partial"
            details="Cline prepared, VS Code not found"
        else
            details="VS Code not found, Cline not configured"
        fi
    fi
    
    # Basic resource information
    status_data+=("name" "cline")
    status_data+=("category" "agents")
    status_data+=("description" "AI coding assistant for VS Code")
    status_data+=("installed" "$([[ "$version" != "not installed" ]] && echo "true" || echo "false")")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$details")
    status_data+=("status" "$status")
    status_data+=("health" "$health")
    status_data+=("version" "$version")
    status_data+=("provider" "$provider")
    status_data+=("api_configured" "$api_configured")
    status_data+=("vscode_available" "$vscode_available")
    
    # Add details if present
    if [[ -n "$details" ]]; then
        status_data+=("details" "$details")
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display status in text format
#######################################
cline::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    echo "Cline Status:"
    echo "  Status: ${data[status]:-unknown}"
    echo "  Running: ${data[running]:-false}"
    echo "  Health: ${data[health]:-unknown}"
    echo "  Version: ${data[version]:-unknown}"
    echo "  Provider: ${data[provider]:-unknown}"
    echo "  API Configured: ${data[api_configured]:-false}"
    echo "  VS Code Available: ${data[vscode_available]:-false}"
    [[ -n "${data[details]:-}" ]] && echo "  Details: ${data[details]}"
}

#######################################
# Check Cline status - Main status function
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cline::status() {
    status::run_standard "cline" "cline::status::collect_data" "cline::status::display_text" "$@"
}

#######################################
# Detailed health check for Cline
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cline::health_check() {
    local verbose="${1:-false}"
    
    if [[ "$verbose" == "true" ]]; then
        log::header "🔍 Cline Health Check"
    fi
    
    local checks_passed=0
    local checks_total=4
    
    # Check 1: VS Code installed
    if cline::check_vscode; then
        ((checks_passed++))
        [[ "$verbose" == "true" ]] && log::success "✓ VS Code is installed"
    else
        [[ "$verbose" == "true" ]] && log::error "✗ VS Code not found"
    fi
    
    # Check 2: Cline extension installed
    if cline::is_installed; then
        ((checks_passed++))
        local version
        version=$(cline::get_version)
        [[ "$verbose" == "true" ]] && log::success "✓ Cline extension installed (v$version)"
    else
        [[ "$verbose" == "true" ]] && log::error "✗ Cline extension not installed"
    fi
    
    # Check 3: Configuration exists
    if [[ -f "${CLINE_SETTINGS_FILE}" ]]; then
        ((checks_passed++))
        [[ "$verbose" == "true" ]] && log::success "✓ Configuration file exists"
    else
        [[ "$verbose" == "true" ]] && log::warn "✗ Configuration file missing"
    fi
    
    # Check 4: API key configured
    local api_key
    api_key=$(cline::get_api_key "${CLINE_DEFAULT_PROVIDER}")
    if [[ -n "$api_key" ]]; then
        ((checks_passed++))
        [[ "$verbose" == "true" ]] && log::success "✓ API key configured for ${CLINE_DEFAULT_PROVIDER}"
    else
        [[ "$verbose" == "true" ]] && log::warn "✗ No API key for ${CLINE_DEFAULT_PROVIDER}"
    fi
    
    if [[ "$verbose" == "true" ]]; then
        log::info "Health check: $checks_passed/$checks_total checks passed"
    fi
    
    # Consider healthy if at least 3 out of 4 checks pass
    if [[ $checks_passed -ge 3 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Display Cline information
#######################################
cline::info() {
    log::header "ℹ️ Cline Information"
    
    echo "Installation Status:"
    if cline::is_installed; then
        echo "  ✓ Installed (version: $(cline::get_version))"
    else
        echo "  ✗ Not installed"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Provider: ${CLINE_DEFAULT_PROVIDER}"
    echo "  Model: ${CLINE_DEFAULT_MODEL}"
    echo "  Config Dir: ${CLINE_CONFIG_DIR}"
    
    if cline::is_configured; then
        echo "  ✓ API configured"
    else
        echo "  ✗ API not configured"
    fi
    
    echo ""
    echo "Integration:"
    if [[ "${CLINE_USE_OLLAMA}" == "true" ]]; then
        echo "  ✓ Ollama integration enabled"
        echo "    URL: ${CLINE_OLLAMA_BASE_URL}"
    else
        echo "  ✗ Ollama integration disabled"
    fi
    
    if [[ "${CLINE_USE_OPENROUTER}" == "true" ]]; then
        echo "  ✓ OpenRouter integration enabled"
    else
        echo "  ✗ OpenRouter integration disabled"
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::status "$@"
fi