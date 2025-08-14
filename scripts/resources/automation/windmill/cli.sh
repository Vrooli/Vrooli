#!/usr/bin/env bash
################################################################################
# Windmill Resource CLI
# 
# Lightweight CLI interface for Windmill that delegates to existing lib functions.
#
# Usage:
#   resource-windmill <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
WINDMILL_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$WINDMILL_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$WINDMILL_CLI_DIR"
export WINDMILL_SCRIPT_DIR="$WINDMILL_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source windmill configuration
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
windmill::export_config 2>/dev/null || true

# Source windmill libraries
for lib in common docker status install apps api workers; do
    lib_file="${WINDMILL_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "windmill"

################################################################################
# Delegate to existing windmill functions
################################################################################

# Inject workflows or apps into windmill
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-windmill inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function
    if command -v windmill::inject &>/dev/null; then
        windmill::inject
    else
        "${WINDMILL_CLI_DIR}/inject.sh" --inject "$(cat "$file")"
    fi
}

# Validate windmill configuration
resource_cli::validate() {
    if command -v windmill::validate &>/dev/null; then
        windmill::validate
    elif command -v windmill::status &>/dev/null; then
        windmill::status
    else
        # Basic validation
        log::header "Validating Windmill"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "windmill" || {
            log::error "Windmill containers not running"
            return 1
        }
        log::success "Windmill is running"
    fi
}

# Show windmill status
resource_cli::status() {
    if command -v windmill::status &>/dev/null; then
        windmill::status
    else
        # Basic status
        log::header "Windmill Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "windmill"; then
            echo "Containers: ✅ Running"
            docker ps --filter "name=windmill" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | tail -n +2
        else
            echo "Containers: ❌ Not running"
        fi
    fi
}

# Start windmill
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Windmill"
        return 0
    fi
    
    if command -v windmill::start &>/dev/null; then
        windmill::start
    else
        log::error "windmill::start not available"
        return 1
    fi
}

# Stop windmill
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Windmill"
        return 0
    fi
    
    if command -v windmill::stop &>/dev/null; then
        windmill::stop
    else
        log::error "windmill::stop not available"
        return 1
    fi
}

# Install windmill
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Windmill"
        return 0
    fi
    
    if command -v windmill::install &>/dev/null; then
        windmill::install
    else
        log::error "windmill::install not available"
        return 1
    fi
}

# Uninstall windmill
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Windmill and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Windmill"
        return 0
    fi
    
    if command -v windmill::uninstall &>/dev/null; then
        windmill::uninstall
    else
        log::error "windmill::uninstall not available"
        return 1
    fi
}

################################################################################
# Windmill-specific commands (if functions exist)
################################################################################

# List available apps
windmill_list_apps() {
    if command -v windmill::list_apps &>/dev/null; then
        windmill::list_apps
    else
        log::error "App listing not available"
        return 1
    fi
}

# Prepare app for deployment
windmill_prepare_app() {
    local app_name="${1:-}"
    local output_dir="${2:-.}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill prepare-app <app-name> [output-dir]"
        return 1
    fi
    
    if command -v windmill::prepare_app &>/dev/null; then
        windmill::prepare_app "$app_name" "$output_dir"
    else
        log::error "App preparation not available"
        return 1
    fi
}

# Deploy app to workspace
windmill_deploy_app() {
    local app_name="${1:-}"
    local workspace="${2:-demo}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill deploy-app <app-name> [workspace]"
        return 1
    fi
    
    if command -v windmill::deploy_app &>/dev/null; then
        windmill::deploy_app "$app_name" "$workspace"
    else
        log::error "App deployment not available"
        return 1
    fi
}

# Scale workers
windmill_scale_workers() {
    local count="${1:-2}"
    
    if command -v windmill::scale_workers &>/dev/null; then
        windmill::scale_workers "$count"
    else
        log::error "Worker scaling not available"
        return 1
    fi
}

# Backup windmill data
windmill_backup() {
    local backup_path="${1:-./windmill-backup}"
    
    if command -v windmill::backup &>/dev/null; then
        windmill::backup "$backup_path"
    else
        log::error "Backup not available"
        return 1
    fi
}

# Restore windmill data
windmill_restore() {
    local backup_path="${1:-}"
    
    if [[ -z "$backup_path" ]]; then
        log::error "Backup path required"
        echo "Usage: resource-windmill restore <backup-path>"
        return 1
    fi
    
    if command -v windmill::restore &>/dev/null; then
        windmill::restore "$backup_path"
    else
        log::error "Restore not available"
        return 1
    fi
}

# Show API setup instructions
windmill_api_setup() {
    if command -v windmill::show_api_setup_instructions &>/dev/null; then
        windmill::show_api_setup_instructions
    else
        log::error "API setup instructions not available"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🌪️  Windmill Resource CLI

USAGE:
    resource-windmill <command> [options]

CORE COMMANDS:
    inject <file>            Inject workflow/app into Windmill
    validate                 Validate Windmill configuration
    status                   Show Windmill status
    start                    Start Windmill containers
    stop                     Stop Windmill containers
    install                  Install Windmill
    uninstall                Uninstall Windmill (requires --force)
    
WINDMILL COMMANDS:
    list-apps                List available app examples
    prepare-app <name> [dir] Prepare app for deployment
    deploy-app <name> [ws]   Deploy app to workspace
    scale-workers <count>    Scale worker containers
    backup [path]            Backup Windmill data
    restore <path>           Restore from backup
    api-setup                Show API setup instructions

OPTIONS:
    --verbose, -v            Show detailed output
    --dry-run                Preview actions without executing
    --force                  Force operation (skip confirmations)

EXAMPLES:
    resource-windmill status
    resource-windmill inject shared:initialization/automation/windmill/workflow.json
    resource-windmill list-apps
    resource-windmill prepare-app admin-dashboard ./apps/
    resource-windmill scale-workers 4

For more information: https://docs.vrooli.com/resources/windmill
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall)
            resource_cli::$command "$@"
            ;;
            
        # Windmill-specific commands
        list-apps)
            windmill_list_apps "$@"
            ;;
        prepare-app)
            windmill_prepare_app "$@"
            ;;
        deploy-app)
            windmill_deploy_app "$@"
            ;;
        scale-workers)
            windmill_scale_workers "$@"
            ;;
        backup)
            windmill_backup "$@"
            ;;
        restore)
            windmill_restore "$@"
            ;;
        api-setup)
            windmill_api_setup "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi