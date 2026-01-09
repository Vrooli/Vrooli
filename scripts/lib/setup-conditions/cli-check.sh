#!/usr/bin/env bash
################################################################################
# Setup Condition: CLI Command Check
#
# Checks if specified CLI commands are available in PATH and up-to-date
# Used for verifying CLI tools are installed and current
#
# Checks performed:
# 1. Command exists in PATH
# 2. CLI binary is newer than source files in cli/ directory
# 3. CLI binary is newer than local replace directive dependencies
#
# Input: JSON object with "command" field
# Returns: 0 if setup needed (command missing/outdated), 1 if available and current
################################################################################

set -euo pipefail

# Get check configuration from argument
# Note: Cannot use ${1:-{}} as bash has a parsing bug when arg ends with }
CHECK_CONFIG="${1:-}"
[[ -z "$CHECK_CONFIG" ]] && CHECK_CONFIG='{}'

# Extract command to check
COMMAND=$(echo "$CHECK_CONFIG" | jq -r '.command // empty' 2>/dev/null)

if [[ -z "$COMMAND" ]]; then
    # No command specified - consider setup not needed
    exit 1
fi

# Check if command is available
CLI_PATH=$(command -v "$COMMAND" 2>/dev/null || true)

if [[ -z "$CLI_PATH" ]]; then
    echo "[DEBUG] CLI command not found: $COMMAND" >&2
    exit 0  # Setup needed
fi

# Command exists - now check if it's stale
# Look for cli/ directory relative to APP_ROOT
CLI_SOURCE_DIR="${APP_ROOT:-$(pwd)}/cli"

if [[ -d "$CLI_SOURCE_DIR" ]]; then
    # Check if any Go source files are newer than the CLI binary
    if find "$CLI_SOURCE_DIR" -name "*.go" -newer "$CLI_PATH" 2>/dev/null | head -1 | grep -q .; then
        echo "[DEBUG] CLI outdated (source files modified): $COMMAND" >&2
        exit 0  # Setup needed
    fi

    # Check go.mod for dependency updates
    if [[ -f "$CLI_SOURCE_DIR/go.mod" ]] && [[ "$CLI_SOURCE_DIR/go.mod" -nt "$CLI_PATH" ]]; then
        echo "[DEBUG] CLI outdated (go.mod modified): $COMMAND" >&2
        exit 0  # Setup needed
    fi

    # Check go.sum for dependency version updates
    if [[ -f "$CLI_SOURCE_DIR/go.sum" ]] && [[ "$CLI_SOURCE_DIR/go.sum" -nt "$CLI_PATH" ]]; then
        echo "[DEBUG] CLI outdated (go.sum modified): $COMMAND" >&2
        exit 0  # Setup needed
    fi

    # Check local replace directives for shared package changes
    if [[ -f "$CLI_SOURCE_DIR/go.mod" ]]; then
        while IFS= read -r replace_path; do
            [[ -z "$replace_path" ]] && continue

            # Resolve path relative to CLI_SOURCE_DIR
            resolved_path="$CLI_SOURCE_DIR/$replace_path"

            # Only check if it's a local directory
            if [[ -d "$resolved_path" ]]; then
                if find "$resolved_path" -name "*.go" -newer "$CLI_PATH" 2>/dev/null | head -1 | grep -q .; then
                    echo "[DEBUG] CLI outdated (replace dependency modified): $replace_path" >&2
                    exit 0  # Setup needed
                fi

                # Also check go.mod in the replaced package
                if [[ -f "$resolved_path/go.mod" ]] && [[ "$resolved_path/go.mod" -nt "$CLI_PATH" ]]; then
                    echo "[DEBUG] CLI outdated (replace dependency go.mod modified): $replace_path" >&2
                    exit 0  # Setup needed
                fi
            fi
        done < <(grep "^replace.*=>" "$CLI_SOURCE_DIR/go.mod" 2>/dev/null | awk '{print $NF}' | grep "^\.\./")
    fi
fi

# All checks passed - CLI is current
exit 1