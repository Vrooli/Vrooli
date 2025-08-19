#!/usr/bin/env bash
# Cline Status Functions

# Set script directory for sourcing
CLINE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/../../../../lib/utils/format.sh" 2>/dev/null || true

#######################################
# Check Cline status
# Arguments:
#   --format: Output format (text/json)
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cline::status() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status="stopped"
    local health="unhealthy"
    local version="not installed"
    local provider="${CLINE_DEFAULT_PROVIDER}"
    local api_configured="false"
    local vscode_available="false"
    local details=""
    local running="false"
    
    # Check status file first
    if [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
        local file_status=$(cat "$CLINE_CONFIG_DIR/.status" 2>/dev/null || echo "stopped")
        if [[ "$file_status" == "running" ]]; then
            status="running"
            running="true"
        fi
    fi
    
    # Check VS Code availability
    if cline::check_vscode; then
        vscode_available="true"
    else
        details="VS Code not found"
    fi
    
    # Check configuration regardless of VS Code presence
    if cline::is_configured; then
        api_configured="true"
    fi
    
    # Determine health based on configuration and VS Code availability
    if [[ "$vscode_available" == "true" ]]; then
        # VS Code is available
        if cline::is_installed; then
            version=$(cline::get_version)
            if [[ "$api_configured" == "true" ]]; then
                health="healthy"
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
    
    # Build output data array for format utility
    local -a output_data=(
        "name" "cline"
        "status" "$status"
        "running" "$running"
        "health" "$health"
        "version" "$version"
        "provider" "$provider"
        "api_configured" "$api_configured"
        "vscode_available" "$vscode_available"
    )
    
    # Add details if present
    if [[ -n "$details" ]]; then
        output_data+=("details" "$details")
    fi
    
    # Use standard formatter if available, fallback to legacy output
    if type -t format::key_value &>/dev/null; then
        format::key_value "$format" "${output_data[@]}"
    else
        # Fallback to legacy output if format utility not available
        if [[ "$format" == "json" ]]; then
            # Output JSON format
            cat <<EOF
{
  "name": "cline",
  "status": "$status",
  "running": $running,
  "health": "$health",
  "version": "$version",
  "provider": "$provider",
  "api_configured": $api_configured,
  "vscode_available": $vscode_available,
  "details": "$details"
}
EOF
        else
            # Output text format
            echo "Cline Status:"
            echo "  Status: $status"
            echo "  Running: $running"
            echo "  Health: $health"
            echo "  Version: $version"
            echo "  Provider: $provider"
            echo "  API Configured: $api_configured"
            echo "  VS Code Available: $vscode_available"
            [[ -n "$details" ]] && echo "  Details: $details"
        fi
    fi
    
    # Return appropriate exit code
    # Return 0 if healthy or partial (configured but waiting for VS Code)
    if [[ "$health" == "healthy" || ( "$health" == "partial" && "$api_configured" == "true" ) ]]; then
        return 0
    else
        return 1
    fi
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