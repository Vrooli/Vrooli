#!/usr/bin/env bash
################################################################################
# Setup Condition: Dependencies Check
# 
# Checks if dependencies are installed for various package managers
# Supports Go, Node.js, Python, and other ecosystems
#
# Input: JSON object with "paths" array of dependency files
# Returns: 0 if setup needed (deps missing), 1 if installed
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"
APP_ROOT="${APP_ROOT:-$(pwd)}"

# Extract paths to check
PATHS=$(echo "$CHECK_CONFIG" | jq -r '.paths[]?' 2>/dev/null || echo "")

if [[ -z "$PATHS" ]]; then
    # No paths specified - consider setup not needed
    exit 1
fi

# Check each dependency file
MISSING_COUNT=0
while IFS= read -r path; do
    if [[ -z "$path" ]]; then
        continue
    fi
    
    # Resolve path relative to APP_ROOT if not absolute
    if [[ ! "$path" =~ ^/ ]]; then
        path="${APP_ROOT}/$path"
    fi
    
    # Check based on file type
    case "$path" in
        */package.json)
            # Node.js - check for node_modules
            DIR=$(dirname "$path")
            if [[ ! -d "$DIR/node_modules" ]]; then
                echo "[DEBUG] Node modules not installed at $DIR" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        */go.mod)
            # Go - check for go.sum or vendor directory
            DIR=$(dirname "$path")
            if [[ ! -f "$DIR/go.sum" ]] && [[ ! -d "$DIR/vendor" ]]; then
                echo "[DEBUG] Go dependencies not downloaded at $DIR" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        */requirements.txt)
            # Python - check for virtual environment
            DIR=$(dirname "$path")
            if [[ ! -d "$DIR/venv" ]] && [[ ! -d "$DIR/.venv" ]]; then
                echo "[DEBUG] Python virtual environment not found at $DIR" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        */Cargo.toml)
            # Rust - check for target directory
            DIR=$(dirname "$path")
            if [[ ! -d "$DIR/target" ]]; then
                echo "[DEBUG] Rust dependencies not built at $DIR" >&2
                ((MISSING_COUNT++))
            fi
            ;;
        *)
            # Unknown dependency file - just check it exists
            if [[ ! -f "$path" ]]; then
                echo "[DEBUG] Dependency file not found: $path" >&2
                ((MISSING_COUNT++))
            fi
            ;;
    esac
done <<< "$PATHS"

# Return 0 if any dependencies are missing (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0