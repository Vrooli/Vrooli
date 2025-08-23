#!/bin/bash

# BTCPay Server Resource CLI

set -euo pipefail

# Get script directory
BTCPAY_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Determine if we're running from the installed location or the source location
if [[ "${BTCPAY_CLI_DIR}" == *"/.local/bin"* ]]; then
    # Running from installed location, use absolute path to source
    BTCPAY_RESOURCE_DIR="/home/matthalloran8/Vrooli/scripts/resources/execution/btcpay"
else
    # Running from source location
    BTCPAY_RESOURCE_DIR="${BTCPAY_CLI_DIR}"
fi

# Source library functions
source "${BTCPAY_RESOURCE_DIR}/lib/common.sh"
source "${BTCPAY_RESOURCE_DIR}/lib/install.sh"
source "${BTCPAY_RESOURCE_DIR}/lib/start.sh"
source "${BTCPAY_RESOURCE_DIR}/lib/stop.sh"
source "${BTCPAY_RESOURCE_DIR}/lib/status.sh"
source "${BTCPAY_RESOURCE_DIR}/lib/inject.sh"

# Help function
show_help() {
    cat <<EOF
BTCPay Server Resource Management CLI

Usage: $(basename "$0") <command> [options]

Commands:
    install    Install BTCPay Server and dependencies
    uninstall  Uninstall BTCPay Server
    start      Start BTCPay Server
    stop       Stop BTCPay Server
    restart    Restart BTCPay Server
    status     Show BTCPay Server status
    inject     Inject configuration or data
    help       Show this help message

Options:
    --format json    Output in JSON format (for status)
    --type <type>    Injection type (store|webhook|invoice|api_key)

Examples:
    $(basename "$0") install
    $(basename "$0") status --format json
    $(basename "$0") inject store.json --type store
    $(basename "$0") restart

BTCPay Server is a self-hosted, open-source cryptocurrency payment processor.
It's secure, private, censorship-resistant and free.

For more information, visit: https://btcpayserver.org
EOF
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        install)
            btcpay::install "$@"
            ;;
        uninstall)
            btcpay::uninstall "$@"
            ;;
        start)
            btcpay::start "$@"
            ;;
        stop)
            btcpay::stop "$@"
            ;;
        restart)
            btcpay::stop
            btcpay::start "$@"
            ;;
        status)
            local format="text"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --format)
                        format="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            btcpay::status "${format}"
            ;;
        inject)
            local file="${1:-}"
            local type="store"
            shift || true
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --type)
                        type="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            btcpay::inject "${file}" "${type}"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log::error "Unknown command: ${command}"
            show_help
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"