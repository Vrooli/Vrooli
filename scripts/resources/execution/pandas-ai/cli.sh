#!/usr/bin/env bash
set -euo pipefail

# Get the real directory of this script (follows symlinks)
SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
PANDAS_AI_LIB_DIR="${SCRIPT_DIR}/lib"

# Source dependencies using absolute paths
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
source "${SCRIPT_DIR}/../../../lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${PANDAS_AI_LIB_DIR}/lifecycle.sh"
source "${PANDAS_AI_LIB_DIR}/status.sh"
source "${PANDAS_AI_LIB_DIR}/inject.sh"
source "${PANDAS_AI_LIB_DIR}/install.sh"

# Main function
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        start)
            pandas_ai::start "$@"
            ;;
        stop)
            pandas_ai::stop "$@"
            ;;
        status)
            pandas_ai::status "$@"
            ;;
        install)
            pandas_ai::install "$@"
            ;;
        inject)
            pandas_ai::inject "$@"
            ;;
        analyze)
            pandas_ai::analyze "$@"
            ;;
        help|--help|-h)
            pandas_ai::help
            ;;
        *)
            log::error "Unknown command: ${command}"
            pandas_ai::help
            exit 1
            ;;
    esac
}

# Execute main if not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi