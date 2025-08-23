#!/bin/bash

# Resolve the actual script location (handles symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    FFMPEG_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    FFMPEG_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source required utilities first
source "${FFMPEG_CLI_DIR}/../../../lib/utils/log.sh"
source "${FFMPEG_CLI_DIR}/../../../lib/utils/format.sh"

# Source all lib functions directly
for lib in install status start stop uninstall inject; do
    source "${FFMPEG_CLI_DIR}/lib/${lib}.sh"
done

# Help function
show_help() {
    cat << HELP
Usage: resource-ffmpeg <command> [options]

FFmpeg universal media processing framework

Commands:
  help         Show this help message
  install      Install FFmpeg [modifies system]
  uninstall    Uninstall FFmpeg (requires --force) [modifies system]
  start        Verify FFmpeg is available
  stop         No-op for CLI tool
  status       Show installation status with JSON support
  test         Run integration tests
  inject       Process media files [modifies system]
  info         Get media file information
  transcode    Convert media to different format [modifies system]
  extract      Extract audio/video/frames [modifies system]

Examples:
  resource-ffmpeg install
  resource-ffmpeg status --format json
  resource-ffmpeg inject video.mp4 info
  resource-ffmpeg inject input.avi transcode
  resource-ffmpeg inject video.mp4 extract

Media Processing Examples:
  # Get media info
  resource-ffmpeg info video.mp4
  
  # Convert video format
  resource-ffmpeg transcode input.avi
  
  # Extract audio from video
  resource-ffmpeg extract video.mp4
  
  # Process with default settings
  resource-ffmpeg inject video.mp4 process

HELP
}

# Route commands
main() {
    local cmd="${1:-help}"
    shift
    
    case "$cmd" in
        help|--help|-h)
            show_help
            ;;
        install)
            ffmpeg_install "$@"
            ;;
        uninstall)
            ffmpeg_uninstall "$@"
            ;;
        start)
            ffmpeg_start "$@"
            ;;
        stop)
            ffmpeg_stop "$@"
            ;;
        status)
            ffmpeg_status "$@"
            ;;
        test)
            # Run integration tests
            if command -v bats &> /dev/null; then
                cd "${FFMPEG_CLI_DIR}" && bats test/integration.bats | tee test/.test_results
            else
                echo "Error: bats is not installed. Install with: sudo apt install bats"
                return 1
            fi
            ;;
        inject)
            ffmpeg_inject "$@"
            ;;
        info)
            # Shortcut for inject <file> info
            ffmpeg_inject "$1" "info"
            ;;
        transcode)
            # Shortcut for inject <file> transcode
            ffmpeg_inject "$1" "transcode"
            ;;
        extract)
            # Shortcut for inject <file> extract
            ffmpeg_inject "$1" "extract"
            ;;
        *)
            echo "Unknown command: $cmd"
            echo "Run 'resource-ffmpeg help' for usage"
            return 1
            ;;
    esac
}

main "$@"
