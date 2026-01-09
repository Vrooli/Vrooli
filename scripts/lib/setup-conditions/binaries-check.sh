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
# 3. Binary is newer than dependency files (go.mod, go.sum, Cargo.toml, etc.)
# 4. Binary is newer than local replace directive dependencies (shared packages)
#
# Input: JSON object with "targets" array
# Returns: 0 if setup needed (binaries missing/outdated), 1 if all current
################################################################################

set -euo pipefail

# Get check configuration from argument
# Note: Cannot use ${1:-{}} as bash has a parsing bug when arg ends with }
CHECK_CONFIG="${1:-}"
[[ -z "$CHECK_CONFIG" ]] && CHECK_CONFIG='{}'

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

        # Check go.sum for dependency version updates
        if [[ -f "$binary_dir/go.sum" ]] && [[ "$binary_dir/go.sum" -nt "$target" ]]; then
            echo "[DEBUG] Binary outdated (go.sum modified): $target" >&2
            MISSING_COUNT=$((MISSING_COUNT + 1))
        fi

        # Check local replace directives for shared package changes
        # Parse go.mod for lines like: replace github.com/foo => ../../../packages/foo
        if [[ -f "$binary_dir/go.mod" ]]; then
            while IFS= read -r replace_path; do
                [[ -z "$replace_path" ]] && continue

                # Resolve path relative to binary_dir
                resolved_path="$binary_dir/$replace_path"

                # Only check if it's a local directory (not a version replacement)
                if [[ -d "$resolved_path" ]]; then
                    if find "$resolved_path" -name "*.go" -newer "$target" 2>/dev/null | head -1 | grep -q .; then
                        echo "[DEBUG] Binary outdated (replace dependency modified): $replace_path" >&2
                        MISSING_COUNT=$((MISSING_COUNT + 1))
                        break  # One stale dependency is enough to trigger rebuild
                    fi

                    # Also check go.mod in the replaced package
                    if [[ -f "$resolved_path/go.mod" ]] && [[ "$resolved_path/go.mod" -nt "$target" ]]; then
                        echo "[DEBUG] Binary outdated (replace dependency go.mod modified): $replace_path" >&2
                        MISSING_COUNT=$((MISSING_COUNT + 1))
                        break
                    fi
                fi
            done < <(grep "^replace.*=>" "$binary_dir/go.mod" 2>/dev/null | awk '{print $NF}' | grep "^\.\./")
        fi
    fi
done <<< "$TARGETS"

# Return 0 if any binaries are missing (setup needed)
[[ $MISSING_COUNT -eq 0 ]] && exit 1 || exit 0