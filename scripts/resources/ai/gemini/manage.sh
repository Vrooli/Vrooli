#!/bin/bash
# Gemini resource management script

# Get script directory
GEMINI_MANAGE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the CLI which has all functionality
source "${GEMINI_MANAGE_DIR}/cli.sh"

# Main handler
main() {
    local command="${1:-help}"
    shift
    
    # Pass to CLI handler
    gemini::cli "$command" "$@"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi