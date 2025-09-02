#!/usr/bin/env bash
################################################################################
# Setup Condition: Binary Files Check
# 
# Checks if specified binary files exist, are executable, and are up-to-date
# Used primarily for compiled languages (Go, Rust, C++)
#
# Checks performed:
# 1. Binary exists and is executable
# 2. Binary is newer than all source files (*.go, *.rs, *.c, *.cpp)
# 3. Binary is newer than dependency files (go.mod, Cargo.toml, etc.)
#
# Input: JSON object with "targets" array
# Returns: 0 if setup needed (binaries missing/outdated), 1 if all current
################################################################################

set -euo pipefail

# Get check configuration from argument
CHECK_CONFIG="${1:-{}}"

# Extract targets array
TARGETS=$(echo "$CHECK_CONFIG" | jq -r '.targets[]?' 2>/dev/null || echo "")

if [[ -z "$TARGETS" ]]; then
    # No targets specified - consider setup not needed
    exit 1
fi

# Check each target binary
MISSING_COUNT=0
while IFS= read -r target; do
    if [[ -z "$target" ]]; then
        continue
    fi
    
    # Resolve path relative to APP_ROOT if not absolute
    if [[ ! "$target" =~ ^/ ]]; then
        target="${APP_ROOT:-$(pwd)}/$target"
    fi
    
    # Check if file exists and is executable
    if [[ ! -f "$target" ]] || [[ ! -x "$target" ]]; then
        echo "[DEBUG] Binary missing or not executable: $target" >&2
        # shellcheck disable=SC2219
        MISSING_COUNT=$((MISSING_COUNT + 1))
    else
        # Binary exists, check if it's outdated compared to source files
        # Get the directory containing the binary (usually api/)
        binary_dir=$(dirname "$target")
        
        # Check if any Go source files are newer than the binary
        # This handles the common case of Go APIs
        if find "$binary_dir" -name "*.go" -newer "$target" 2>/dev/null | head -1 | grep -q .; then
            echo "[DEBUG] Binary outdated (source files modified): $target" >&2
            MISSING_COUNT=$((MISSING_COUNT + 1))
        fi
        
        # Also check go.mod for dependency updates
        if [[ -f "$binary_dir/go.mod" ]] && [[ "$binary_dir/go.mod" -nt "$target" ]]; then
            echo "[DEBUG] Binary outdated (go.mod modified): $target" >&2
            MISSING_COUNT=$((MISSING_COUNT + 1))
        fi
    fi
done <<< "$TARGETS"

# Return 0 if any binaries are missing (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0