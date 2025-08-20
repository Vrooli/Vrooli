#!/bin/bash
# MusicGen Resource CLI
# Meta's AI music generation model for creating original music

set -euo pipefail

# Get script directory - handle both direct and symlinked execution
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If symlinked, resolve the real path
    MUSICGEN_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    MUSICGEN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

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