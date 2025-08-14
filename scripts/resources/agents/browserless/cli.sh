#!/usr/bin/env bash
################################################################################
# Browserless Resource CLI
# 
# Lightweight CLI interface for Browserless that delegates to existing lib functions.
#
# Usage:
#   resource-browserless <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    BROWSERLESS_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
BROWSERLESS_CLI_DIR="$(cd "$(dirname "$BROWSERLESS_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$BROWSERLESS_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$BROWSERLESS_CLI_DIR"
export BROWSERLESS_SCRIPT_DIR="$BROWSERLESS_CLI_DIR"  # For compatibility with existing libs

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

# Source browserless configuration
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
browserless::export_config 2>/dev/null || true

# Source browserless libraries
for lib in core docker health status api inject usage recovery; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "browserless"

################################################################################
# Delegate to existing browserless functions
################################################################################

# Inject configuration into browserless
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-browserless inject <file.json>"
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
    if command -v browserless::inject &>/dev/null; then
        browserless::inject "$file"
    else
        log::error "Browserless injection function not available"
        return 1
    fi
}

# Validate browserless configuration
resource_cli::validate() {
    if command -v browserless::health &>/dev/null; then
        browserless::health
    elif command -v browserless::check_basic_health &>/dev/null; then
        browserless::check_basic_health
    else
        # Basic validation
        log::header "Validating Browserless"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "browserless" || {
            log::error "Browserless container not running"
            return 1
        }
        log::success "Browserless is running"
    fi
}

# Show browserless status
resource_cli::status() {
    if command -v browserless::status &>/dev/null; then
        browserless::status
    else
        # Basic status
        log::header "Browserless Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "browserless"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=browserless" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start browserless
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Browserless"
        return 0
    fi
    
    if command -v browserless::start &>/dev/null; then
        browserless::start
    else
        docker start browserless || log::error "Failed to start Browserless"
    fi
}

# Stop browserless
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Browserless"
        return 0
    fi
    
    if command -v browserless::stop &>/dev/null; then
        browserless::stop
    else
        docker stop browserless || log::error "Failed to stop Browserless"
    fi
}

# Install browserless
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Browserless"
        return 0
    fi
    
    if command -v browserless::install &>/dev/null; then
        browserless::install
    else
        log::error "browserless::install not available"
        return 1
    fi
}

# Uninstall browserless
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Browserless and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Browserless"
        return 0
    fi
    
    if command -v browserless::uninstall &>/dev/null; then
        browserless::uninstall
    else
        docker stop browserless 2>/dev/null || true
        docker rm browserless 2>/dev/null || true
        log::success "Browserless uninstalled"
    fi
}

################################################################################
# Browserless-specific commands
################################################################################

# Take screenshot
browserless_screenshot() {
    local url="${1:-}"
    local output="${2:-screenshot.png}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for screenshot"
        echo "Usage: resource-browserless screenshot <url> [output-file]"
        return 1
    fi
    
    if command -v browserless::screenshot &>/dev/null; then
        browserless::screenshot "$url" "$output"
    else
        log::error "Screenshot function not available"
        return 1
    fi
}

# Generate PDF
browserless_pdf() {
    local url="${1:-}"
    local output="${2:-output.pdf}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for PDF generation"
        echo "Usage: resource-browserless pdf <url> [output-file]"
        return 1
    fi
    
    if command -v browserless::pdf &>/dev/null; then
        browserless::pdf "$url" "$output"
    else
        log::error "PDF generation function not available"
        return 1
    fi
}

# Test all APIs
browserless_test_apis() {
    if command -v browserless::test_all_apis &>/dev/null; then
        browserless::test_all_apis
    else
        log::error "API testing not available"
        return 1
    fi
}

# Show browser pressure/metrics
browserless_metrics() {
    if command -v browserless::test_pressure &>/dev/null; then
        browserless::test_pressure
    else
        # Basic metrics via API
        local url="http://localhost:${BROWSERLESS_PORT:-3000}/pressure"
        if curl -s "$url" 2>/dev/null; then
            curl -s "$url" | jq '.' 2>/dev/null || curl -s "$url"
        else
            log::error "Could not fetch metrics"
            return 1
        fi
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Browserless Resource CLI

USAGE:
    resource-browserless <command> [options]

CORE COMMANDS:
    inject <file>           Inject configuration into Browserless
    validate                Validate Browserless health
    status                  Show Browserless status
    start                   Start Browserless container
    stop                    Stop Browserless container
    install                 Install Browserless
    uninstall               Uninstall Browserless (requires --force)
    
BROWSERLESS COMMANDS:
    screenshot <url> [file] Take screenshot of URL
    pdf <url> [file]        Generate PDF from URL
    test-apis               Test all Browserless APIs
    metrics                 Show browser pressure/metrics

OPTIONS:
    --verbose, -v           Show detailed output
    --dry-run               Preview actions without executing
    --force                 Force operation (skip confirmations)

EXAMPLES:
    resource-browserless status
    resource-browserless screenshot https://example.com screenshot.png
    resource-browserless pdf https://example.com document.pdf
    resource-browserless test-apis
    resource-browserless metrics

For more information: https://docs.vrooli.com/resources/browserless
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
            
        # Browserless-specific commands
        screenshot)
            browserless_screenshot "$@"
            ;;
        pdf)
            browserless_pdf "$@"
            ;;
        test-apis)
            browserless_test_apis "$@"
            ;;
        metrics)
            browserless_metrics "$@"
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