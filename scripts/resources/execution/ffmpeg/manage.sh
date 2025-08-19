#!/bin/bash

FFMPEG_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
for lib in install status start stop uninstall inject; do
    source "${FFMPEG_SCRIPT_DIR}/lib/${lib}.sh"
done

# Main function
main() {
    local action="${1:-status}"
    shift
    
    case "${action}" in
        install|--install)
            ffmpeg_install "$@"
            ;;
        uninstall|--uninstall)
            ffmpeg_uninstall "$@"
            ;;
        start|--start)
            ffmpeg_start "$@"
            ;;
        stop|--stop)
            ffmpeg_stop "$@"
            ;;
        status|--status)
            ffmpeg_status "$@"
            ;;
        inject|--inject)
            ffmpeg_inject "$@"
            ;;
        *)
            echo "Usage: $0 {install|uninstall|start|stop|status|inject} [options]"
            return 1
            ;;
    esac
}

main "$@"
