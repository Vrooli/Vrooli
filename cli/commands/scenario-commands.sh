#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Scenario Management Commands
# 
# Thin wrapper around the Vrooli Scenario HTTP API
#
# Usage:
#   vrooli scenario <subcommand> [options]
#
################################################################################

set -euo pipefail

# API configuration
API_PORT="${VROOLI_API_PORT:-8090}"
API_BASE="http://localhost:${API_PORT}"

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Check if API is running
check_api() {
    if ! curl -s "${API_BASE}/health" >/dev/null 2>&1; then
        log::error "Scenario API is not running"
        echo "Start it with: cd api && go run main.go"
        return 1
    fi
}

# Show help for scenario commands
show_scenario_help() {
    cat << EOF
🚀 Vrooli Scenario Commands

USAGE:
    vrooli scenario <subcommand> [options]

SUBCOMMANDS:
    list                    List all available scenarios
    info <name>             Show detailed information about a scenario
    validate <name>         Validate scenario configuration
    convert <name>          Convert scenario to standalone app
    convert-all             Convert all enabled scenarios
    enable <name>           Enable scenario in catalog
    disable <name>          Disable scenario in catalog

OPTIONS:
    --help, -h              Show this help message
    --verbose, -v           Show detailed output

EXAMPLES:
    vrooli scenario list                      # List all scenarios
    vrooli scenario info research-assistant   # Show scenario details
    vrooli scenario validate my-scenario      # Validate configuration
    vrooli scenario convert my-scenario       # Generate app from scenario
    vrooli scenario convert-all --force       # Regenerate all apps

For more information: https://docs.vrooli.com/cli/scenarios
EOF
}

# List all scenarios
scenario_list() {
    check_api || return 1
    
    log::header "Available Scenarios"
    
    local response
    response=$(curl -s "${API_BASE}/scenarios")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "Failed to list scenarios"
        return 1
    fi
    
    echo ""
    printf "%-35s %-10s %-15s %s\n" "SCENARIO NAME" "ENABLED" "CATEGORY" "DESCRIPTION"
    printf "%-35s %-10s %-15s %s\n" "-------------" "-------" "--------" "-----------"
    
    local enabled_count=0
    local total_count=0
    
    echo "$response" | jq -c '.data[]' | while IFS= read -r scenario; do
        local name enabled category description
        name=$(echo "$scenario" | jq -r '.name')
        enabled=$(echo "$scenario" | jq -r '.enabled')
        category=$(echo "$scenario" | jq -r '.category // "general"')
        description=$(echo "$scenario" | jq -r '.description // "No description"' | cut -c1-40)
        
        local status_icon="❌"
        if [[ "$enabled" == "true" ]]; then
            status_icon="✅"
        fi
        
        printf "%-35s %-10s %-15s %s...\n" "$name" "$status_icon" "$category" "$description"
    done
    
    # Get counts from API response
    total_count=$(echo "$response" | jq '.data | length')
    enabled_count=$(echo "$response" | jq '[.data[] | select(.enabled == true)] | length')
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Total: $total_count scenarios ($enabled_count enabled)"
}

# Show detailed info about a scenario
scenario_info() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario info <scenario-name>"
        return 1
    fi
    
    check_api || return 1
    
    local response
    response=$(curl -s "${API_BASE}/scenarios/${scenario_name}")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "$(echo "$response" | jq -r '.error // "Failed to get scenario"')"
        return 1
    fi
    
    local scenario_data
    scenario_data=$(echo "$response" | jq -r '.data')
    
    local name enabled description category location has_service path
    name=$(echo "$scenario_data" | jq -r '.scenario.name')
    enabled=$(echo "$scenario_data" | jq -r '.scenario.enabled')
    description=$(echo "$scenario_data" | jq -r '.scenario.description // "No description"')
    category=$(echo "$scenario_data" | jq -r '.scenario.category // "general"')
    location=$(echo "$scenario_data" | jq -r '.scenario.location')
    has_service=$(echo "$scenario_data" | jq -r '.has_service')
    path=$(echo "$scenario_data" | jq -r '.path')
    
    log::header "Scenario: $scenario_name"
    echo ""
    echo "📍 Location: $path"
    echo "📁 Category: $category"
    echo "🔧 Status: $([ "$enabled" == "true" ] && echo "✅ Enabled" || echo "❌ Disabled")"
    echo "📝 Description: $description"
    
    if [[ "$has_service" == "true" ]]; then
        echo ""
        echo "✅ Has service.json configuration"
    else
        echo ""
        log::warning "No service.json found"
    fi
    
    # Check for generated app
    local app_path="$HOME/generated-apps/$scenario_name"
    echo ""
    echo "Generated App:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if [[ -d "$app_path" ]]; then
        echo "✅ App exists: $app_path"
    else
        echo "❌ Not generated yet"
        echo "   Run: vrooli scenario convert $scenario_name"
    fi
}

# Validate a scenario
scenario_validate() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario validate <scenario-name>"
        return 1
    fi
    
    check_api || return 1
    
    log::info "Validating scenario: $scenario_name"
    
    local response
    response=$(curl -s -X POST "${API_BASE}/scenarios/${scenario_name}/validate")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "Validation failed"
        return 1
    fi
    
    local result
    result=$(echo "$response" | jq -r '.data')
    
    if [[ $(echo "$result" | jq -r '.valid') == "true" ]]; then
        log::success "✅ Scenario is valid"
    else
        log::error "❌ Scenario has validation issues:"
        echo "$result" | jq -r '.issues[]' | while IFS= read -r issue; do
            echo "   • $issue"
        done
    fi
}

# Convert scenario to app
scenario_convert() {
    local scenario_name="${1:-}"
    shift
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario convert <scenario-name> [options]"
        return 1
    fi
    
    check_api || return 1
    
    log::info "Starting conversion: $scenario_name"
    
    local response
    response=$(curl -s -X POST "${API_BASE}/scenarios/${scenario_name}/convert")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "$(echo "$response" | jq -r '.error // "Conversion failed"')"
        return 1
    fi
    
    log::success "✅ Conversion started"
    echo "Checking status..."
    
    # Poll for completion
    local max_wait=300  # 5 minutes
    local elapsed=0
    while [[ $elapsed -lt $max_wait ]]; do
        sleep 2
        response=$(curl -s "${API_BASE}/scenarios/${scenario_name}/status")
        local status
        status=$(echo "$response" | jq -r '.data')
        
        if [[ $(echo "$status" | jq -r '.in_progress') == "false" ]]; then
            local message
            message=$(echo "$status" | jq -r '.message')
            
            if echo "$message" | grep -q "successfully"; then
                log::success "✅ $message"
                echo "App location: $HOME/generated-apps/$scenario_name"
            else
                log::error "$message"
                return 1
            fi
            return 0
        fi
        
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo ""
    log::warning "Conversion is taking longer than expected"
    echo "Check status with: vrooli scenario status $scenario_name"
}

# Convert all enabled scenarios
scenario_convert_all() {
    check_api || return 1
    
    log::info "Converting all enabled scenarios..."
    
    local response
    response=$(curl -s -X POST "${API_BASE}/scenarios/convert-all")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "Failed to start conversion"
        return 1
    fi
    
    log::success "✅ $(echo "$response" | jq -r '.data')"
    echo "This will run in the background. Check individual app status."
}

# Enable scenario in catalog
scenario_enable() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario enable <scenario-name>"
        return 1
    fi
    
    check_api || return 1
    
    local response
    response=$(curl -s -X PUT "${API_BASE}/scenarios/${scenario_name}/enable")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "$(echo "$response" | jq -r '.error // "Failed to enable scenario"')"
        return 1
    fi
    
    log::success "✅ Scenario enabled: $scenario_name"
    echo "Run 'vrooli scenario convert $scenario_name' to generate the app"
}

# Disable scenario in catalog
scenario_disable() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario disable <scenario-name>"
        return 1
    fi
    
    check_api || return 1
    
    local response
    response=$(curl -s -X DELETE "${API_BASE}/scenarios/${scenario_name}/enable")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "$(echo "$response" | jq -r '.error // "Failed to disable scenario"')"
        return 1
    fi
    
    log::success "✅ Scenario disabled: $scenario_name"
    echo "This scenario will not be auto-converted during setup"
}

# Check conversion status
scenario_status() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario status <scenario-name>"
        return 1
    fi
    
    check_api || return 1
    
    local response
    response=$(curl -s "${API_BASE}/scenarios/${scenario_name}/status")
    
    if ! echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log::error "Failed to get status"
        return 1
    fi
    
    local status
    status=$(echo "$response" | jq -r '.data')
    
    if [[ $(echo "$status" | jq -r '.in_progress') == "true" ]]; then
        log::info "🔄 Conversion in progress"
        echo "Message: $(echo "$status" | jq -r '.message')"
        echo "Started: $(echo "$status" | jq -r '.start_time')"
    else
        echo "Status: $(echo "$status" | jq -r '.message')"
    fi
}

# Main command handler
main() {
    # Check for verbose flag
    VERBOSE=false
    for arg in "$@"; do
        if [[ "$arg" == "--verbose" ]] || [[ "$arg" == "-v" ]]; then
            VERBOSE=true
        fi
    done
    
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_scenario_help
        return 0
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        list)
            scenario_list "$@"
            ;;
        info)
            scenario_info "$@"
            ;;
        validate)
            scenario_validate "$@"
            ;;
        convert)
            scenario_convert "$@"
            ;;
        convert-all)
            scenario_convert_all "$@"
            ;;
        status)
            scenario_status "$@"
            ;;
        enable)
            scenario_enable "$@"
            ;;
        disable)
            scenario_disable "$@"
            ;;
        *)
            log::error "Unknown scenario command: $subcommand"
            echo ""
            show_scenario_help
            return 1
            ;;
    esac
}

# Execute main function
main "$@"