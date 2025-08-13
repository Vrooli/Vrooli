#!/usr/bin/env bash
################################################################################
# Multi-App Runner for Vrooli Develop
# 
# Orchestrates multiple generated apps during the develop phase using the
# Vrooli Orchestrator. This script automatically discovers enabled scenarios,
# registers their generated apps with the orchestrator, and starts them all
# with proper port management and resource coordination.
#
# Usage:
#   multi-app-runner.sh [options]
#
# Options:
#   --dry-run          Show what would be started without starting
#   --port-start N     Start port assignment from N (default: 3001)
#   --max-apps N       Maximum apps to start (default: 10)
#   --verbose          Show detailed output
################################################################################

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_CLIENT="$SCRIPT_DIR/orchestrator-client.sh"
ORCHESTRATOR_CTL="$SCRIPT_DIR/orchestrator-ctl.sh"

# Default configuration
PORT_START="${VROOLI_DEV_PORT_START:-3001}"
MAX_APPS="${VROOLI_DEV_MAX_APPS:-100}"
DRY_RUN=false
VERBOSE=false
GENERATED_APPS_DIR="$HOME/generated-apps"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Source utilities if available
if [[ -f "$SCRIPT_DIR/../../lib/utils/var.sh" ]]; then
    # shellcheck disable=SC1090
    source "$SCRIPT_DIR/../../lib/utils/var.sh"
    # shellcheck disable=SC1090
    source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true
fi

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

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --port-start)
                PORT_START="$2"
                shift 2
                ;;
            --max-apps)
                MAX_APPS="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
}

# Show help
show_help() {
    cat << EOF
${CYAN}🚀 Multi-App Runner for Vrooli Develop${NC}

${YELLOW}USAGE:${NC}
    multi-app-runner.sh [options]

${YELLOW}OPTIONS:${NC}
    --dry-run          Show what would be started without starting
    --port-start N     Start port assignment from N (default: $PORT_START)
    --max-apps N       Maximum apps to start (default: 100)
    --verbose          Show detailed output
    --help, -h         Show this help message

${YELLOW}ENVIRONMENT VARIABLES:${NC}
    VROOLI_DEV_PORT_START    Starting port for app assignment
    VROOLI_DEV_MAX_APPS      Maximum number of apps to start

${YELLOW}EXAMPLES:${NC}
    multi-app-runner.sh                    # Start all enabled apps
    multi-app-runner.sh --dry-run          # Show what would be started
    multi-app-runner.sh --port-start 4000  # Start port assignment from 4000
    multi-app-runner.sh --max-apps 5       # Limit to 5 apps

EOF
}

# Get enabled scenarios from catalog
get_enabled_scenarios() {
    local catalog_file
    
    # Try to find catalog file
    if [[ -n "${var_SCRIPTS_SCENARIOS_DIR:-}" ]]; then
        catalog_file="${var_SCRIPTS_SCENARIOS_DIR}/catalog.json"
    elif [[ -f "scripts/scenarios/catalog.json" ]]; then
        catalog_file="scripts/scenarios/catalog.json"
    elif [[ -f "$(cd "$SCRIPT_DIR/../.." && pwd)/catalog.json" ]]; then
        catalog_file="$(cd "$SCRIPT_DIR/../.." && pwd)/catalog.json"
    else
        log_error "Cannot find scenarios catalog.json"
        return 1
    fi
    
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
    
    # Check if app is already running (basic check)
    if pgrep -f "manage.sh.*develop.*$app_name" >/dev/null 2>&1; then
        log_warning "App may already be running: $app_name"
        return 1
    fi
    
    return 0
}

# Assign port to app
assign_port() {
    local app_name="$1"
    local base_port="$2"
    
    # Check if port is already in use
    while netstat -tuln 2>/dev/null | grep -q ":$base_port "; do
        log_info "Port $base_port in use, trying next..."
        base_port=$((base_port + 1))
    done
    
    echo "$base_port"
}

# Start orchestrator if not running
ensure_orchestrator() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would ensure orchestrator is running"
        return 0
    fi
    
    log_info "Ensuring orchestrator is running..."
    
    # Source the client library
    if [[ -f "$ORCHESTRATOR_CLIENT" ]]; then
        # shellcheck disable=SC1090
        source "$ORCHESTRATOR_CLIENT" >/dev/null 2>&1
    else
        log_error "Orchestrator client not found: $ORCHESTRATOR_CLIENT"
        return 1
    fi
    
    # The client library will automatically start the daemon if needed
    return 0
}

# Register and start an app
register_and_start_app() {
    local app_name="$1"
    local app_port="$2"
    local app_path="$GENERATED_APPS_DIR/$app_name"
    
    log_info "Registering and starting app: $app_name (port: $app_port)"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [DRY RUN] Would register: $app_name"
        echo "  [DRY RUN] Command: cd $app_path && ./scripts/manage.sh develop"
        echo "  [DRY RUN] Port: $app_port"
        echo "  [DRY RUN] Working dir: $app_path"
        return 0
    fi
    
    # Source orchestrator client if not already loaded
    if [[ -z "${VROOLI_ORCHESTRATOR_CLIENT_LOADED:-}" ]]; then
        # shellcheck disable=SC1090
        source "$ORCHESTRATOR_CLIENT" >/dev/null
    fi
    
    # Change to app directory for proper context detection
    local original_pwd="$PWD"
    cd "$app_path"
    
    # Set explicit parent context to prevent infinite nesting
    # All apps started by multi-app-runner should be direct children of "vrooli"
    export VROOLI_ORCHESTRATOR_PARENT="vrooli"
    
    # Register the app with orchestrator using app name as the process name
    if orchestrator::register \
        "$app_name.develop" \
        "./scripts/manage.sh develop" \
        --working-dir "$app_path" \
        --env "PORT" "$app_port" \
        --env "VROOLI_APP_NAME" "$app_name" \
        --metadata "app_type" "generated" \
        --metadata "port" "$app_port" \
        --auto-restart; then
        
        log_info "Registered $app_name successfully"
        
        # Start the app using exact registered name to avoid parent detection
        # Use direct orchestrator command to bypass parent auto-detection  
        if orchestrator::send_command "start" "vrooli.$app_name.develop"; then
            log_success "Started $app_name on port $app_port"
        else
            log_error "Failed to start $app_name"
            cd "$original_pwd"
            return 1
        fi
    else
        log_error "Failed to register $app_name"
        cd "$original_pwd"
        return 1
    fi
    
    cd "$original_pwd"
    return 0
}

# Main function
main() {
    parse_args "$@"
    
    echo -e "${CYAN}🚀 Multi-App Runner for Vrooli${NC}"
    echo ""
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}🔍 DRY RUN MODE - Nothing will be started${NC}"
        echo ""
    fi
    
    # Get enabled scenarios
    log_info "Discovering enabled scenarios..."
    local enabled_scenarios
    if ! enabled_scenarios=$(get_enabled_scenarios); then
        log_error "Failed to get enabled scenarios"
        exit 1
    fi
    
    if [[ -z "$enabled_scenarios" ]]; then
        log_warning "No enabled scenarios found"
        echo ""
        echo "To enable scenarios:"
        echo "  1. Check scripts/scenarios/catalog.json"
        echo "  2. Set 'enabled': true for desired scenarios"
        echo "  3. Run 'vrooli setup' to generate apps"
        exit 0
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
        exit 0
    fi
    
    # Check limits
    if [[ $valid_count -gt $MAX_APPS ]]; then
        log_warning "Limiting to $MAX_APPS apps (found $valid_count valid apps)"
        valid_count=$MAX_APPS
    fi
    
    # Start orchestrator
    if ! ensure_orchestrator; then
        log_error "Failed to start orchestrator"
        exit 1
    fi
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Starting $valid_count apps..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Start valid apps
    local current_port=$PORT_START
    local started_count=0
    local failed_count=0
    local started_apps=()
    
    for scenario_name in "${valid_apps[@]}"; do
        # Stop if we've reached the limit
        if [[ $started_count -ge $MAX_APPS ]]; then
            break
        fi
        
        # Assign port
        local app_port
        app_port=$(assign_port "$scenario_name" "$current_port")
        
        # Register and start the app
        if register_and_start_app "$scenario_name" "$app_port"; then
            started_apps+=("$scenario_name:$app_port")
            started_count=$((started_count + 1))
            current_port=$((app_port + 1))
        else
            failed_count=$((failed_count + 1))
        fi
        
        # Small delay between starts
        [[ "$DRY_RUN" != "true" ]] && sleep 2
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Startup Summary:"
    echo "  Started: $started_count apps"
    echo "  Failed:  $failed_count apps"
    echo "  Ports:   $PORT_START - $((current_port - 1))"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ "$DRY_RUN" != "true" ]] && [[ $started_count -gt 0 ]]; then
        echo ""
        log_success "✅ Multi-app environment started successfully!"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${CYAN}Running Applications:${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Display clickable URLs for each started app
        for app_info in "${started_apps[@]}"; do
            local app_name="${app_info%%:*}"
            local app_port="${app_info##*:}"
            printf "  %-30s → " "$app_name"
            echo -e "${BLUE}http://localhost:$app_port${NC}"
        done
        
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${CYAN}Management Commands:${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  vrooli orchestrator status     # Show all app status"
        echo "  vrooli orchestrator tree       # Show app hierarchy"
        echo "  vrooli orchestrator logs <app> # Show app logs"
        echo "  vrooli orchestrator monitor    # Real-time dashboard"
        echo "  vrooli orchestrator stop       # Stop all apps"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo -e "${CYAN}View Logs:${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Display log commands for each started app
        for app_info in "${started_apps[@]}"; do
            local app_name="${app_info%%:*}"
            echo "  vrooli orchestrator logs $app_name    # View $app_name logs"
        done
    fi
    
    # Show immediate status if apps were started
    if [[ "$DRY_RUN" != "true" ]] && [[ $started_count -gt 0 ]]; then
        echo ""
        echo "Initial Status:"
        sleep 3  # Give apps time to start
        "$ORCHESTRATOR_CTL" status 2>/dev/null || true
    fi
}

# Execute main function
main "$@"