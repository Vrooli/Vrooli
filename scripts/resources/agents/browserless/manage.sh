#!/usr/bin/env bash
set -euo pipefail

# Browserless Chrome Service Setup and Management
# This script handles installation, configuration, and management of Browserless.io using Docker

export DESCRIPTION="Install and manage Browserless headless Chrome service using Docker"

BROWSERLESS_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$(dirname "$(dirname "$(dirname "${BROWSERLESS_SCRIPT_DIR}")")")/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/config/messages.sh"

# Export configuration
browserless::export_config

# Source refactored library modules
# Core module contains most shared functionality
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/health.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/recovery.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/usage.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_SCRIPT_DIR}/lib/inject.sh"

#######################################
# Parse command line arguments
#######################################
browserless::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|version|test|usage|screenshot|pdf|scrape|pressure|inject|injection-status|url|create-backup|list-backups|backup-info|recover" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Browserless appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "lines" \
        --flag "n" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "50"
    
    args::register \
        --name "max-browsers" \
        --desc "Maximum concurrent browser instances" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "headless" \
        --desc "Run browsers in headless mode" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "timeout" \
        --desc "Browser timeout in milliseconds" \
        --type "value" \
        --default "30000"
    
    args::register \
        --name "usage-type" \
        --desc "Type of usage example to run" \
        --type "value" \
        --options "screenshot|pdf|scrape|pressure|function|all|help" \
        --default "help"
    
    args::register \
        --name "url" \
        --desc "URL to use for operations" \
        --type "value" \
        --default "http://httpbin.org/html"
    
    args::register \
        --name "output" \
        --desc "Output file for operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "selector" \
        --desc "CSS selector for scraping" \
        --type "value" \
        --default ""
    
    args::register \
        --name "data" \
        --desc "JSON data for API operations" \
        --type "value" \
        --default ""
    
    args::register \
        --name "label" \
        --desc "Label for backup (default: auto)" \
        --type "value" \
        --default "auto"
    
    
    if args::is_asking_for_help "$@"; then
        browserless::usage
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    FORCE=$(args::get "force")
    LINES=$(args::get "lines")
    YES=$(args::get "yes")
    MAX_BROWSERS=$(args::get "max-browsers")
    HEADLESS=$(args::get "headless")
    TIMEOUT=$(args::get "timeout")
    USAGE_TYPE=$(args::get "usage-type")
    URL=$(args::get "url")
    OUTPUT=$(args::get "output")
    SELECTOR=$(args::get "selector")
    DATA=$(args::get "data")
    LABEL=$(args::get "label")
    export ACTION FORCE LINES YES MAX_BROWSERS HEADLESS TIMEOUT USAGE_TYPE URL OUTPUT SELECTOR DATA LABEL
}

#######################################
# Display usage information
#######################################
browserless::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  # Installation"
    echo "  $0 --action install                             # Install with defaults"
    echo "  $0 --action install --max-browsers 10          # Install with 10 max browsers"
    echo "  $0 --action install --headless no              # Install with headed browsers"
    echo
    echo "  # Container Management"
    echo "  $0 --action status                              # Check status"
    echo "  $0 --action logs                                # View logs"
    echo "  $0 --action restart                             # Restart container"
    echo
    echo "  # API Operations"
    echo "  $0 --action screenshot --url https://google.com # Take screenshot"
    echo "  $0 --action pdf --url https://example.com      # Generate PDF"
    echo "  $0 --action scrape --url https://example.com   # Scrape content"
    echo "  $0 --action test                                # Test all APIs"
    echo
    echo "  # Data Management"
    echo "  $0 --action inject                              # Inject test data"
    echo "  $0 --action injection-status                    # Check injection status"
    echo "  $0 --action create-backup                       # Create backup"
    echo "  $0 --action recover                             # Recover from backup"
    echo
    echo "  # Usage Examples"
    echo "  $0 --action usage                               # Show usage examples"
    echo "  $0 --action usage --usage-type all             # Run all examples"
}

#######################################
# Main execution function
#######################################
browserless::main() {
    browserless::parse_arguments "$@"
    
    case "$ACTION" in
        install)
            browserless::install
            ;;
        uninstall)
            browserless::uninstall
            ;;
        start)
            browserless::start
            ;;
        stop)
            browserless::stop
            ;;
        restart)
            browserless::restart
            ;;
        status)
            browserless::status
            ;;
        logs)
            browserless::logs
            ;;
        info)
            browserless::info
            ;;
        version)
            browserless::version
            ;;
        test)
            browserless::test
            ;;
        usage)
            browserless::run_usage_example "$USAGE_TYPE"
            ;;
        screenshot)
            browserless::test_screenshot "$URL" "$OUTPUT"
            ;;
        pdf)
            browserless::test_pdf "$URL" "$OUTPUT"
            ;;
        scrape)
            browserless::test_scrape "$URL" "$OUTPUT"
            ;;
        pressure)
            browserless::test_pressure
            ;;
        inject)
            browserless::inject
            ;;
        injection-status)
            browserless::injection_status
            ;;
        url)
            browserless::get_urls
            ;;
        create-backup)
            browserless::create_backup "${LABEL:-auto}"
            ;;
        list-backups)
            backup::list "browserless"
            ;;
        backup-info)
            backup::info "browserless"
            ;;
        recover)
            browserless::recover
            ;;
        *)
            log::error "Unknown action: $ACTION"
            browserless::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    browserless::main "$@"
fi