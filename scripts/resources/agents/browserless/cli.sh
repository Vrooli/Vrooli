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
    *)
        echo "📚 Browserless CLI"
        echo "Usage: $0 {start|stop|status|install|uninstall|test|inject|for|workflow}"
        echo ""
        echo "Commands:"
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
        echo "Examples:"
        echo "  $0 for n8n execute <workflow-id>    # Execute N8n workflow via browserless"
        echo "  $0 for n8n list                     # List N8n workflows"
        echo "  $0 for                               # Show available adapters"
        exit 1
        ;;
esac
