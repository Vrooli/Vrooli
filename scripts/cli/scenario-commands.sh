#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Scenario Management Commands
# 
# Handles scenario operations including listing, validation, and conversion.
#
# Usage:
#   vrooli scenario <subcommand> [options]
#
################################################################################

set -euo pipefail

# Get CLI directory
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Scenario paths
SCENARIOS_DIR="${var_SCRIPTS_SCENARIOS_DIR}"
CATALOG_FILE="${SCENARIOS_DIR}/catalog.json"
SCENARIO_TO_APP="${SCENARIOS_DIR}/tools/scenario-to-app.sh"
AUTO_CONVERTER="${SCENARIOS_DIR}/tools/auto-converter.sh"

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
    log::header "Available Scenarios"
    
    if [[ ! -f "$CATALOG_FILE" ]]; then
        log::error "Scenario catalog not found: $CATALOG_FILE"
        return 1
    fi
    
    echo ""
    echo "Catalog: $CATALOG_FILE"
    echo ""
    printf "%-35s %-10s %-15s %s\n" "SCENARIO NAME" "ENABLED" "CATEGORY" "DESCRIPTION"
    printf "%-35s %-10s %-15s %s\n" "-------------" "-------" "--------" "-----------"
    
    # Parse catalog.json
    local enabled_count=0
    local total_count=0
    
    while IFS= read -r scenario; do
        local name enabled location description category
        name=$(echo "$scenario" | jq -r '.name')
        enabled=$(echo "$scenario" | jq -r '.enabled')
        location=$(echo "$scenario" | jq -r '.location')
        description=$(echo "$scenario" | jq -r '.description // "No description"' | cut -c1-40)
        category=$(echo "$scenario" | jq -r '.category // "general"')
        
        local status_icon="❌"
        if [[ "$enabled" == "true" ]]; then
            status_icon="✅"
            ((enabled_count++))
        fi
        
        printf "%-35s %-10s %-15s %s\n" "$name" "$status_icon" "$category" "$description..."
        ((total_count++))
    done < <(jq -c '.scenarios[]' "$CATALOG_FILE" 2>/dev/null)
    
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
    
    # Find scenario in catalog
    local scenario_json
    scenario_json=$(jq -c ".scenarios[] | select(.name == \"$scenario_name\")" "$CATALOG_FILE" 2>/dev/null)
    
    if [[ -z "$scenario_json" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    local location enabled description category
    location=$(echo "$scenario_json" | jq -r '.location')
    enabled=$(echo "$scenario_json" | jq -r '.enabled')
    description=$(echo "$scenario_json" | jq -r '.description // "No description"')
    category=$(echo "$scenario_json" | jq -r '.category // "general"')
    
    local scenario_path="${SCENARIOS_DIR}/${location}"
    
    log::header "Scenario: $scenario_name"
    echo ""
    echo "📍 Location: $scenario_path"
    echo "📁 Category: $category"
    echo "🔧 Status: $([ "$enabled" == "true" ] && echo "✅ Enabled" || echo "❌ Disabled")"
    echo "📝 Description: $description"
    
    # Check service.json
    local service_json_path="${scenario_path}/.vrooli/service.json"
    if [[ ! -f "$service_json_path" ]]; then
        service_json_path="${scenario_path}/service.json"
    fi
    
    if [[ -f "$service_json_path" ]]; then
        echo ""
        echo "Service Configuration:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        local display_name version type
        display_name=$(jq -r '.service.displayName // .service.name' "$service_json_path" 2>/dev/null)
        version=$(jq -r '.service.version // "Unknown"' "$service_json_path" 2>/dev/null)
        type=$(jq -r '.service.type // "Unknown"' "$service_json_path" 2>/dev/null)
        
        echo "📋 Display Name: $display_name"
        echo "🏷️  Version: $version"
        echo "🎯 Type: $type"
        
        # Count enabled resources
        local resource_count
        resource_count=$(jq -r '
            .resources | to_entries[] as $category | 
            $category.value | to_entries[] | 
            select(.value.enabled == true) | .key
        ' "$service_json_path" 2>/dev/null | wc -l)
        
        echo "🔌 Resources: $resource_count enabled"
        
        # List resources if verbose
        if [[ "${VERBOSE:-false}" == "true" ]] && [[ $resource_count -gt 0 ]]; then
            echo ""
            echo "   Enabled Resources:"
            jq -r '
                .resources | to_entries[] as $category | 
                $category.value | to_entries[] | 
                select(.value.enabled == true) | 
                "   • \(.key) (\($category.key))"
            ' "$service_json_path" 2>/dev/null
        fi
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
        
        # Quick status check
        if [[ -d "$app_path/.git" ]]; then
            cd "$app_path" 2>/dev/null || return 1
            local is_modified=false
            if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
                is_modified=true
            fi
            cd - >/dev/null || true
            
            if [[ "$is_modified" == "true" ]]; then
                echo "   🔧 Status: Modified"
            else
                echo "   ✅ Status: Clean"
            fi
        fi
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
    
    log::info "Validating scenario: $scenario_name"
    
    # Use the validation script if available
    local validate_script="${SCENARIOS_DIR}/tools/validate-scenario.sh"
    if [[ -f "$validate_script" ]]; then
        "$validate_script" "$scenario_name"
    else
        # Fallback to basic validation using scenario-to-app in dry-run mode
        "$SCENARIO_TO_APP" "$scenario_name" --dry-run
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
    
    # Pass through to scenario-to-app.sh
    "$SCENARIO_TO_APP" "$scenario_name" "$@"
}

# Convert all enabled scenarios
scenario_convert_all() {
    # Pass through to auto-converter.sh
    "$AUTO_CONVERTER" "$@"
}

# Enable scenario in catalog
scenario_enable() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario enable <scenario-name>"
        return 1
    fi
    
    if [[ ! -f "$CATALOG_FILE" ]]; then
        log::error "Catalog file not found: $CATALOG_FILE"
        return 1
    fi
    
    # Check if scenario exists
    if ! jq -e ".scenarios[] | select(.name == \"$scenario_name\")" "$CATALOG_FILE" >/dev/null 2>&1; then
        log::error "Scenario not found in catalog: $scenario_name"
        return 1
    fi
    
    # Update catalog
    local temp_file="${CATALOG_FILE}.tmp"
    if jq "(.scenarios[] | select(.name == \"$scenario_name\") | .enabled) = true" "$CATALOG_FILE" > "$temp_file"; then
        mv "$temp_file" "$CATALOG_FILE"
        log::success "✅ Scenario enabled: $scenario_name"
        echo "Run 'vrooli scenario convert $scenario_name' to generate the app"
    else
        log::error "Failed to update catalog"
        rm -f "$temp_file"
        return 1
    fi
}

# Disable scenario in catalog
scenario_disable() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario disable <scenario-name>"
        return 1
    fi
    
    if [[ ! -f "$CATALOG_FILE" ]]; then
        log::error "Catalog file not found: $CATALOG_FILE"
        return 1
    fi
    
    # Check if scenario exists
    if ! jq -e ".scenarios[] | select(.name == \"$scenario_name\")" "$CATALOG_FILE" >/dev/null 2>&1; then
        log::error "Scenario not found in catalog: $scenario_name"
        return 1
    fi
    
    # Update catalog
    local temp_file="${CATALOG_FILE}.tmp"
    if jq "(.scenarios[] | select(.name == \"$scenario_name\") | .enabled) = false" "$CATALOG_FILE" > "$temp_file"; then
        mv "$temp_file" "$CATALOG_FILE"
        log::success "✅ Scenario disabled: $scenario_name"
        echo "This scenario will not be auto-converted during setup"
    else
        log::error "Failed to update catalog"
        rm -f "$temp_file"
        return 1
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