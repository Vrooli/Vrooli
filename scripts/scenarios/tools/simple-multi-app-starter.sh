#!/usr/bin/env bash
################################################################################
# Simple Multi-App Starter for Vrooli Develop
# 
# Discovers enabled scenarios and starts their generated apps using the
# process manager. Replaces the orchestrator-based multi-app-runner.sh
# with a simpler approach that uses our new process management architecture.
################################################################################

set -euo pipefail

# Script directory and paths
SCENARIO_TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_TOOLS_DIR/../../.." && pwd)"

# shellcheck disable=SC1091
source "${SCENARIO_TOOLS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Default configuration
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"
PORT_START="${VROOLI_DEV_PORT_START:-3001}"
MAX_APPS="${VROOLI_DEV_MAX_APPS:-10}"
VERBOSE=false
# Fast mode is always enabled when running within Vrooli context
# The main Vrooli setup handles network diagnostics, firewall, etc.
# Generated apps include these scripts for standalone deployment but skip them when orchestrated
FAST_MODE="${FAST_MODE:-true}"

# Get list of resources already managed by main Vrooli
# These will be skipped by generated apps to avoid redundant checks
MANAGED_RESOURCES=""
if command -v vrooli &>/dev/null; then
    MANAGED_RESOURCES=$(vrooli status --verbose --json 2>/dev/null | \
        jq -r '.resources_list // ""' | \
        tr ',' '\n' | \
        grep -E ':healthy|:running' | \
        cut -d: -f1 | \
        tr '\n' ',' | \
        sed 's/,$//')
    
    if [[ -n "$MANAGED_RESOURCES" ]]; then
        [[ "$VERBOSE" == "true" ]] && echo -e "${BLUE}[INFO]${NC} Main Vrooli managing resources: ${MANAGED_RESOURCES//,/, }"
        export VROOLI_MANAGED_RESOURCES="$MANAGED_RESOURCES"
    fi
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --verbose) VERBOSE=true ;;
        --fast) FAST_MODE=true ;;
        --port-start) PORT_START="$2"; shift ;;
        --max-apps) MAX_APPS="$2"; shift ;;
        *) break ;;
    esac
    shift
done

# Logging
log_info() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Get enabled scenarios from catalog
get_enabled_scenarios() {
    local catalog_file="$VROOLI_ROOT/scripts/scenarios/catalog.json"
    
    if [[ ! -f "$catalog_file" ]]; then
        log_error "Catalog file not found: $catalog_file"
        return 1
    fi
    
    log_info "Reading catalog: $catalog_file"
    
    # Extract enabled scenarios
    jq -r '.scenarios[] | select(.enabled == true) | .name' "$catalog_file" 2>/dev/null || {
        log_error "Failed to parse catalog file"
        return 1
    }
}

# Check if app exists and is valid
validate_app() {
    local app_name="$1"
    local app_path="$GENERATED_APPS_DIR/$app_name"
    
    log_info "Validating app: $app_name"
    
    # Check if directory exists
    if [[ ! -d "$app_path" ]]; then
        log_warning "App directory not found: $app_path"
        return 1
    fi
    
    # Check for manage script
    if [[ ! -f "$app_path/scripts/manage.sh" ]]; then
        log_warning "App missing manage.sh: $app_name"
        return 1
    fi
    
    # Check if app has service.json
    if [[ ! -f "$app_path/.vrooli/service.json" ]]; then
        log_warning "App missing service.json: $app_name"
        return 1
    fi
    
    return 0
}

# Verify app is actually running after start
verify_app_running() {
    local app_name="$1"
    local max_attempts=10
    
    # Source process manager if not already available
    if ! type -t pm::is_running >/dev/null 2>&1; then
        local process_manager="${VROOLI_ROOT}/scripts/lib/process-manager.sh"
        if [[ -f "$process_manager" ]]; then
            # shellcheck disable=SC1090
            source "$process_manager" 2>/dev/null || return 1
        else
            return 1
        fi
    fi
    
    for ((i=1; i<=max_attempts; i++)); do
        # Check for exact match first
        if pm::is_running "vrooli.develop.$app_name" 2>/dev/null; then
            return 0
        fi
        
        # Also check for any processes related to this app
        if type -t pm::list >/dev/null 2>&1; then
            while IFS= read -r process; do
                # Check various naming patterns that might be used
                if [[ "$process" == "vrooli.develop.$app_name" ]] || \
                   [[ "$process" == "vrooli.develop."*".$app_name" ]] || \
                   [[ "$process" == "vrooli.$app_name."* ]] || \
                   [[ "$process" == "vrooli.develop.start-"* ]]; then
                    # For generic process names, do additional verification
                    if [[ "$process" == "vrooli.develop.start-"* ]]; then
                        # Check if this process belongs to the app we're verifying
                        local app_path="${GENERATED_APPS_DIR:-$HOME/generated-apps}/$app_name"
                        local process_info
                        process_info=$(pm::status "$process" 2>/dev/null || echo "")
                        if [[ "$process_info" == *"$app_path"* ]] || [[ "$process_info" == *"$app_name"* ]]; then
                            return 0
                        fi
                    else
                        return 0
                    fi
                fi
            done < <(pm::list 2>/dev/null)
        fi
        
        sleep 1
    done
    return 1
}

# Start an app using individual vrooli app start command
start_app() {
    local app_name="$1"
    local log_file="/tmp/vrooli-start-${app_name}.log"
    
    log_info "Starting app: $app_name"
    
    # Use the vrooli app start command with increased timeout and capture logs
    # Build command with fast mode flag if enabled
    local start_cmd=("$VROOLI_ROOT/cli/vrooli" app start "$app_name")
    [[ "$FAST_MODE" == "true" ]] && start_cmd+=(--fast)
    
    # First attempt with 30 second timeout
    if timeout 30s "${start_cmd[@]}" >"$log_file" 2>&1; then
        # Verify it's actually running
        if verify_app_running "$app_name"; then
            log_success "Started $app_name"
            PORT_START=$((PORT_START + 1))
            rm -f "$log_file"  # Clean up log on success
            return 0
        else
            log_warning "$app_name started but verification failed"
        fi
    fi
    
    # Retry once if failed
    log_info "Retrying start for $app_name..."
    sleep 2
    # Build retry command with fast mode and skip-setup flags
    local retry_cmd=("$VROOLI_ROOT/cli/vrooli" app start "$app_name" --skip-setup)
    [[ "$FAST_MODE" == "true" ]] && retry_cmd+=(--fast)
    
    if timeout 30s "${retry_cmd[@]}" >>"$log_file" 2>&1; then
        if verify_app_running "$app_name"; then
            log_success "Started $app_name (on retry)"
            PORT_START=$((PORT_START + 1))
            rm -f "$log_file"  # Clean up log on success
            return 0
        fi
    fi
    
    # Failed to start - provide error details
    log_error "Failed to start $app_name after retry"
    if [[ -f "$log_file" ]]; then
        log_warning "Error details saved to: $log_file"
        # Show last few lines of error
        tail -5 "$log_file" | while IFS= read -r line; do
            echo "  > $line" >&2
        done
    fi
    echo "  Use 'vrooli app start $app_name' to debug manually" >&2
    PORT_START=$((PORT_START + 1))  # Still increment port to avoid conflicts
    return 1
}

# Main function
main() {
    echo -e "${CYAN}🚀 Starting Generated Apps${NC}"
    echo ""
    
    # Get enabled scenarios
    log_info "Discovering enabled scenarios..."
    local enabled_scenarios
    if ! enabled_scenarios=$(get_enabled_scenarios); then
        log_error "Failed to get enabled scenarios"
        return 1
    fi
    
    if [[ -z "$enabled_scenarios" ]]; then
        log_warning "No enabled scenarios found"
        echo ""
        echo "To enable scenarios:"
        echo "  1. Check scripts/scenarios/catalog.json"
        echo "  2. Set 'enabled': true for desired scenarios"
        echo "  3. Run 'vrooli setup' to generate apps"
        return 0
    fi
    
    # Count and validate scenarios
    local scenario_list=()
    local valid_apps=()
    
    while IFS= read -r scenario_name; do
        [[ -z "$scenario_name" ]] && continue
        
        scenario_list+=("$scenario_name")
        
        if validate_app "$scenario_name" >/dev/null 2>&1; then
            valid_apps+=("$scenario_name")
        fi
    done <<< "$enabled_scenarios"
    
    local valid_count=${#valid_apps[@]}
    echo "Found ${#scenario_list[@]} enabled scenarios, $valid_count valid for startup"
    
    if [[ $valid_count -eq 0 ]]; then
        log_warning "No valid apps found to start"
        echo ""
        echo "To generate apps:"
        echo "  vrooli setup  # This will convert enabled scenarios to apps"
        return 0
    fi
    
    # Check limits
    if [[ $valid_count -gt $MAX_APPS ]]; then
        log_warning "Limiting to $MAX_APPS apps (found $valid_count valid apps)"
        valid_count=$MAX_APPS
    fi
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Starting $valid_count apps..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Start valid apps
    local started_count=0
    local failed_count=0
    local started_apps=()
    
    for scenario_name in "${valid_apps[@]}"; do
        # Stop if we've reached the limit
        if [[ $started_count -ge $MAX_APPS ]]; then
            break
        fi
        
        if start_app "$scenario_name"; then
            started_apps+=("$scenario_name:$PORT_START")
            started_count=$((started_count + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        
        # Small delay between starts
        sleep 1
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Startup Summary:"
    echo "  Started: $started_count apps"
    echo "  Failed:  $failed_count apps"
    echo "  Ports:   $((PORT_START - started_count)) - $((PORT_START - 1))"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $started_count -gt 0 ]]; then
        echo ""
        log_success "✅ Multi-app environment started successfully!"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${CYAN}Running Applications:${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Display URLs for each started app
        local port=$((PORT_START - started_count))
        for scenario_name in "${started_apps[@]%:*}"; do
            printf "  %-30s → " "$scenario_name"
            echo -e "${BLUE}http://localhost:$port${NC}"
            port=$((port + 1))
        done
        
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${CYAN}Management Commands:${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  vrooli app list                # Show all app status"
        echo "  vrooli app stop <name>         # Stop specific app"
        echo "  vrooli app logs <name>         # Show app logs"
        echo "  vrooli app restart <name>      # Restart specific app"
    fi
    
    return 0
}

# Execute main function
main "$@"