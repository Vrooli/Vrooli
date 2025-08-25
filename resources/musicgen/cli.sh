#!/bin/bash
# MusicGen Resource CLI
# Meta's AI music generation model for creating original music

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    MUSICGEN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${MUSICGEN_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi

MUSICGEN_DIR="${APP_ROOT}/resources/musicgen"

# Source dependencies
source "${MUSICGEN_DIR}/lib/common.sh"
source "${MUSICGEN_DIR}/lib/core.sh"
source "${MUSICGEN_DIR}/lib/status.sh"

# Main CLI logic
main() {
    local action="${1:-}"
    shift || true
    
    case "${action}" in
        status)
            musicgen::status "$@"
            ;;
        install)
            musicgen::install "$@"
            ;;
        uninstall)
            musicgen::uninstall "$@"
            ;;
        start)
            musicgen::start "$@"
            ;;
        stop)
            musicgen::stop "$@"
            ;;
        generate)
            musicgen::generate "$@"
            ;;
        list-models)
            musicgen::list_models "$@"
            ;;
        inject)
            musicgen::inject "$@"
            ;;
        help|--help|-h)
            musicgen::help
            ;;
        *)
            log::error "Unknown action: ${action}"
            musicgen::help
            exit 1
            ;;
    esac
}

# Execute main if not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
