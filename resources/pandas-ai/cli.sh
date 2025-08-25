#!/usr/bin/env bash
set -euo pipefail

# Get the real directory of this script (follows symlinks)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/pandas-ai"
PANDAS_AI_LIB_DIR="${SCRIPT_DIR}/lib"

# Source dependencies using absolute paths
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
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
