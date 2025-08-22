#!/usr/bin/env bash
#
# Browserless CLI - Headless Chrome automation service
#
# Commands:
#   start      - Start browserless container
#   stop       - Stop browserless container
#   status     - Show browserless status
#   install    - Install browserless
#   uninstall  - Uninstall browserless
#   test       - Run integration tests
#   inject     - Inject data/scripts

set -euo pipefail

# Get script directory (resolve symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BROWSERLESS_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    BROWSERLESS_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source shared utilities
source "$BROWSERLESS_CLI_DIR/../../../lib/utils/format.sh"
source "$BROWSERLESS_CLI_DIR/lib/common.sh"

# Main command handler
case "${1:-}" in
    start)
        source "$BROWSERLESS_CLI_DIR/lib/start.sh"
        start_browserless
        ;;
    stop)
        source "$BROWSERLESS_CLI_DIR/lib/stop.sh"
        stop_browserless
        ;;
    status)
        source "$BROWSERLESS_CLI_DIR/lib/status.sh"
        status "${@:2}"
        ;;
    install)
        source "$BROWSERLESS_CLI_DIR/lib/install.sh"
        install_browserless
        ;;
    uninstall)
        source "$BROWSERLESS_CLI_DIR/lib/uninstall.sh"
        uninstall_browserless
        ;;
    test)
        source "$BROWSERLESS_CLI_DIR/lib/test.sh"
        run_tests
        ;;
    inject)
        source "$BROWSERLESS_CLI_DIR/lib/inject.sh"
        browserless::inject "${@:2}"
        ;;
    for)
        # Load adapter framework
        source "$BROWSERLESS_CLI_DIR/adapters/common.sh"
        
        adapter_name="${2:-}"
        if [[ -z "$adapter_name" ]]; then
            echo "📚 Browserless Adapters"
            echo "Usage: $0 for <adapter> <command> [args]"
            echo ""
            adapter::list
            exit 1
        fi
        
        # Load and execute the specified adapter
        if adapter::load "$adapter_name"; then
            # Get the adapter's dispatch function name
            dispatch_function="${adapter_name}::dispatch"
            
            # Check if the dispatch function exists
            if declare -f "$dispatch_function" >/dev/null 2>&1; then
                # Call the adapter's dispatch function with remaining arguments
                "$dispatch_function" "${@:3}"
            else
                echo "Error: Adapter '$adapter_name' missing dispatch function: $dispatch_function" >&2
                exit 1
            fi
        else
            exit 1
        fi
        ;;
    # Atomic Actions - Single-purpose browser operations
    screenshot|navigate|health-check|element-exists|extract-text|extract|extract-forms|interact|console|performance)
        source "$BROWSERLESS_CLI_DIR/lib/actions.sh"
        actions::dispatch "$@"
        ;;
    *)
        echo "📚 Browserless CLI"
        echo "Usage: $0 {start|stop|status|install|uninstall|test|inject|for|workflow|<action>}"
        echo ""
        echo "Service Commands:"
        echo "  start      - Start browserless container"
        echo "  stop       - Stop browserless container  "
        echo "  status     - Show browserless status"
        echo "  install    - Install browserless"
        echo "  uninstall  - Uninstall browserless"
        echo "  test       - Run integration tests"
        echo "  inject     - Inject data/scripts"
        echo "  for        - Use adapters for other resources (e.g., for n8n execute <id>)"
        echo "  workflow   - Workflow compilation with flow control"
        echo ""
        echo "Atomic Actions (for quick agent tasks):"
        echo "  screenshot     - Take screenshots of URLs"
        echo "  navigate       - Navigate to URLs and get basic info"
        echo "  health-check   - Check if URLs load successfully"
        echo "  element-exists - Check if elements exist on pages"
        echo "  extract-text   - Extract text content from elements"
        echo "  extract        - Extract structured data with custom scripts"
        echo "  extract-forms  - Extract form data and input fields"
        echo "  interact       - Perform form fills, clicks, and interactions"
        echo "  console        - Capture console logs from pages"
        echo "  performance    - Measure page performance metrics"
        echo ""
        echo "Universal Options (for atomic actions):"
        echo "  --output <path>    - Save result to file"
        echo "  --timeout <ms>     - Max wait time (default: 30000)"
        echo "  --wait-ms <ms>     - Initial wait before action (default: 2000)"
        echo "  --session <name>   - Use persistent session"
        echo "  --fullpage         - Full page screenshots"
        echo "  --mobile           - Use mobile viewport"
        echo ""
        echo "Examples:"
        echo "  # Service management"
        echo "  $0 for n8n execute <workflow-id>    # Execute N8n workflow via browserless"
        echo "  $0 for n8n list                     # List N8n workflows"
        echo ""
        echo "  # Atomic actions"
        echo "  $0 screenshot http://localhost:3000 --output ui-check.png --fullpage"
        echo "  $0 health-check http://localhost:8080/health \"OK\""
        echo "  $0 element-exists http://localhost:3000 --selector \".main-navigation\""
        echo "  $0 extract-text http://localhost:3000 --selector \"h1\" --output title.txt"
        echo "  $0 extract http://localhost:3000 --script \"return {links: [...document.querySelectorAll('a')].length}\" --output data.json"
        echo "  $0 interact http://localhost:3000 --fill \"input[name=email]:test@example.com\" --click \"button[type=submit]\" --screenshot result.png"
        echo "  $0 console http://localhost:3000 --filter error --output errors.json"
        echo "  $0 performance http://localhost:3000 --output metrics.json"
        exit 1
        ;;
esac
