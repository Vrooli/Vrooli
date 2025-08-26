#!/bin/bash

# Wiki.js Logs Functions

set -euo pipefail

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"
source "$WIKIJS_LIB_DIR/common.sh"

# Show Wiki.js logs
show_logs() {
    local lines=50
    local follow=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --lines|-n)
                lines="$2"
                shift 2
                ;;
            --follow|-f)
                follow=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if ! is_running; then
        echo "[ERROR] Wiki.js is not running"
        return 1
    fi
    
    if [[ "$follow" == true ]]; then
        docker logs -f "$WIKIJS_CONTAINER"
    else
        docker logs --tail "$lines" "$WIKIJS_CONTAINER"
    fi
}