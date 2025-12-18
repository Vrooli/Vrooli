#!/usr/bin/env bash
################################################################################
# Setup Condition: Files/Directories Check
# 
# Generic check for existence of files or directories
# Can check for multiple paths and different types
#
# Input: JSON object with "paths" array and optional "type"
# Returns: 0 if setup needed (files missing), 1 if all exist
################################################################################

set -euo pipefail

# Get check configuration from argument
# Note: Cannot use ${1:-{}} as bash has a parsing bug when arg ends with }
CHECK_CONFIG="${1:-}"
[[ -z "$CHECK_CONFIG" ]] && CHECK_CONFIG='{}'
APP_ROOT="${APP_ROOT:-$(pwd)}"

# Extract paths and type
PATHS=$(echo "$CHECK_CONFIG" | jq -r '.paths[]?' 2>/dev/null || echo "")
CHECK_TYPE=$(echo "$CHECK_CONFIG" | jq -r '.type // "any"' 2>/dev/null)

if [[ -z "$PATHS" ]]; then
    # No paths specified - consider setup not needed
    exit 1
fi

# Check each path
MISSING_COUNT=0
while IFS= read -r path; do
    if [[ -z "$path" ]]; then
        continue
    fi
    
    # Resolve path relative to APP_ROOT if not absolute
    if [[ ! "$path" =~ ^/ ]]; then
        path="${APP_ROOT}/$path"
    fi
    
    # Check based on type
    case "$CHECK_TYPE" in
        file)
            if [[ ! -f "$path" ]]; then
                echo "[DEBUG] File missing: $path" >&2
                ((++MISSING_COUNT))
            fi
            ;;
        directory|dir)
            if [[ ! -d "$path" ]]; then
                echo "[DEBUG] Directory missing: $path" >&2
                ((++MISSING_COUNT))
            fi
            ;;
        any|*)
            if [[ ! -e "$path" ]]; then
                echo "[DEBUG] Path missing: $path" >&2
                ((++MISSING_COUNT))
            fi
            ;;
    esac
done <<< "$PATHS"

# Return 0 if any paths are missing (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0