#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"

# Function to match target strings to their canonical form
# Usage: match_target "target_string"
# Returns the canonical target name or exits with ERROR_USAGE if invalid
match_target() {
    # Convert input to lowercase for case-insensitive matching
    local target
    target=$(echo "$1" | tr '[:upper:]' '[:lower:]')
    
    case "$target" in
        l|nl|linux|unix|ubuntu|native-linux)
            echo "native-linux"
            return 0
            ;;
        m|nm|mac|macos|mac-os|native-mac|native-macos)
            echo "native-mac"
            return 0
            ;;
        w|nw|win|windows|windows-os|native-win)
            echo "native-win"
            return 0
            ;;
        d|dc|docker|docker-only|docker-compose)
            echo "docker-only"
            return 0
            ;;
        k|kc|k8s|cluster|k8s-cluster|kubernetes)
            echo "k8s-cluster"
            return 0
            ;;
        *)
            log::error "Bad --target: $1" >&2
            return 1
            ;;
    esac
}
