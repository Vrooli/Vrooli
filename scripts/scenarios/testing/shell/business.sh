#!/usr/bin/env bash
# Generic business validation helpers for .vrooli/endpoints.json
set -euo pipefail

# Check if a business check is enabled in testing.json
# Usage: testing::business::is_enabled "endpoints"
# Returns: 0 if enabled, 1 if disabled
testing::business::is_enabled() {
    local check_name="$1"
    local testing_config="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"

    # If no testing.json or no business section, default to enabled
    if [ ! -f "$testing_config" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0  # Default to enabled if can't check
    fi

    local enabled
    enabled=$(jq -r ".business.checks.${check_name}.enabled" "$testing_config" 2>/dev/null || echo "null")

    # Default to enabled if not specified or null
    if [ "$enabled" = "null" ] || [ "$enabled" = "true" ]; then
        return 0
    else
        return 1
    fi
}

# Validate API endpoints from endpoints.json
# Reads .vrooli/endpoints.json and checks that each endpoint path exists in the codebase
testing::business::validate_endpoints() {
    # Check if enabled
    if ! testing::business::is_enabled "endpoints"; then
        return 0
    fi

    local endpoints_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/endpoints.json"

    if [ ! -f "$endpoints_file" ]; then
        testing::phase::add_warning "No .vrooli/endpoints.json found; skipping endpoint validation"
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        testing::phase::add_warning "jq not available; skipping endpoint validation"
        return 0
    fi

    # Extract endpoints array from config
    local endpoints_json
    if ! endpoints_json=$(jq -r '.endpoints // []' "$endpoints_file" 2>/dev/null); then
        testing::phase::add_warning "Failed to parse endpoints.json; skipping endpoint validation"
        return 0
    fi

    if [ "$endpoints_json" = "[]" ]; then
        return 0  # No endpoints configured
    fi

    local endpoint_count
    endpoint_count=$(echo "$endpoints_json" | jq 'length')

    if [ "$endpoint_count" -eq 0 ]; then
        return 0
    fi

    echo "📋 Validating $endpoint_count API endpoints..."

    # Determine search tool
    local search_cmd="grep"
    if command -v rg >/dev/null 2>&1; then
        search_cmd="rg"
    fi

    local search_dir="api"
    if [ ! -d "$search_dir" ]; then
        search_dir="."
    fi

    local endpoints_passed=0
    local endpoints_failed=0
    local endpoints_skipped=0

    for i in $(seq 0 $((endpoint_count - 1))); do
        local endpoint
        endpoint=$(echo "$endpoints_json" | jq -r ".[$i]")

        # Extract fields
        local id path method category requirements_json deprecated experimental
        id=$(echo "$endpoint" | jq -r '.id // ""')
        path=$(echo "$endpoint" | jq -r '.path // ""')
        method=$(echo "$endpoint" | jq -r '.method // ""')
        category=$(echo "$endpoint" | jq -r '.category // ""')
        requirements_json=$(echo "$endpoint" | jq -r '.requirements // []')
        deprecated=$(echo "$endpoint" | jq -r '.deprecated // false')
        experimental=$(echo "$endpoint" | jq -r '.experimental // false')

        if [ -z "$path" ]; then
            continue
        fi

        # Build display label
        local label="$path"
        [ -n "$method" ] && label="$method $path"
        [ "$deprecated" = "true" ] && label="$label (deprecated)"
        [ "$experimental" = "true" ] && label="$label (experimental)"

        # Search for endpoint in codebase
        local found=false
        if [ "$search_cmd" = "rg" ]; then
            if rg --fixed-strings --quiet "$path" "$search_dir" 2>/dev/null; then
                found=true
            fi
        else
            if grep -r --fixed-strings --quiet "$path" "$search_dir" 2>/dev/null; then
                found=true
            fi
        fi

        if [ "$found" = "true" ]; then
            testing::phase::check "API endpoint: ${label}" true
            endpoints_passed=$((endpoints_passed + 1))

            # Record requirement evidence
            if [ "$requirements_json" != "[]" ] && [ "$requirements_json" != "null" ]; then
                local req_count
                req_count=$(echo "$requirements_json" | jq 'length')
                for j in $(seq 0 $((req_count - 1))); do
                    local req_id
                    req_id=$(echo "$requirements_json" | jq -r ".[$j]")
                    if [ -n "$req_id" ] && [ "$req_id" != "null" ]; then
                        testing::phase::add_requirement \
                            --id "$req_id" \
                            --status passed \
                            --evidence "Endpoint $label present in codebase"
                    fi
                done
            fi
        else
            if [ "$deprecated" = "true" ] || [ "$experimental" = "true" ]; then
                testing::phase::add_warning "Optional endpoint missing: $label"
                endpoints_skipped=$((endpoints_skipped + 1))
            else
                testing::phase::check "API endpoint: ${label}" false
                endpoints_failed=$((endpoints_failed + 1))

                # Record requirement failure
                if [ "$requirements_json" != "[]" ] && [ "$requirements_json" != "null" ]; then
                    local req_count
                    req_count=$(echo "$requirements_json" | jq 'length')
                    for j in $(seq 0 $((req_count - 1))); do
                        local req_id
                        req_id=$(echo "$requirements_json" | jq -r ".[$j]")
                        if [ -n "$req_id" ] && [ "$req_id" != "null" ]; then
                            testing::phase::add_requirement \
                                --id "$req_id" \
                                --status failed \
                                --evidence "Required endpoint $label missing from codebase"
                        fi
                    done
                fi
            fi
        fi
    done

    echo "  ✓ Endpoints: $endpoints_passed passed, $endpoints_failed failed, $endpoints_skipped skipped"
}

# Validate CLI commands from endpoints.json
# Checks that CLI commands exist in the CLI binary/script
testing::business::validate_cli_commands() {
    # Check if enabled
    if ! testing::business::is_enabled "cli_commands"; then
        return 0
    fi

    local endpoints_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/endpoints.json"

    if [ ! -f "$endpoints_file" ]; then
        return 0  # Already warned in validate_endpoints
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0  # Already warned
    fi

    # Find CLI binary/script
    local cli_path=""
    local scenario_name
    scenario_name=$(basename "$TESTING_PHASE_SCENARIO_DIR")

    # Try common CLI locations
    local cli_candidates=(
        "cli/$scenario_name"
        "cli/cli.sh"
        "cli/cli"
    )

    for candidate in "${cli_candidates[@]}"; do
        if [ -f "${TESTING_PHASE_SCENARIO_DIR}/${candidate}" ]; then
            cli_path="${TESTING_PHASE_SCENARIO_DIR}/${candidate}"
            break
        fi
    done

    if [ -z "$cli_path" ]; then
        testing::phase::add_warning "No CLI found; skipping CLI command validation"
        return 0
    fi

    # Extract CLI commands from endpoints.json
    local commands_json
    if ! commands_json=$(jq -r '.cli_commands // []' "$endpoints_file" 2>/dev/null); then
        return 0
    fi

    if [ "$commands_json" = "[]" ]; then
        return 0
    fi

    local command_count
    command_count=$(echo "$commands_json" | jq 'length')

    if [ "$command_count" -eq 0 ]; then
        return 0
    fi

    echo "📋 Validating $command_count CLI commands..."

    # Get CLI help output or read file directly
    local cli_content
    if [ -x "$cli_path" ]; then
        cli_content=$("$cli_path" help 2>&1 || cat "$cli_path")
    else
        cli_content=$(cat "$cli_path")
    fi

    local commands_passed=0
    local commands_failed=0

    for i in $(seq 0 $((command_count - 1))); do
        local command
        command=$(echo "$commands_json" | jq -r ".[$i]")

        local name subcommands_json requirements_json
        name=$(echo "$command" | jq -r '.name // ""')
        subcommands_json=$(echo "$command" | jq -r '.subcommands // []')
        requirements_json=$(echo "$command" | jq -r '.requirements // []')

        if [ -z "$name" ]; then
            continue
        fi

        # Build label
        local label="$name"
        if [ "$subcommands_json" != "[]" ] && [ "$subcommands_json" != "null" ]; then
            local subcommands
            subcommands=$(echo "$subcommands_json" | jq -r 'join(", ")')
            label="$name (subcommands: $subcommands)"
        fi

        # Check if command appears in CLI
        # For commands with spaces (e.g., "workflow create"), check for both patterns:
        # 1. The full command name in help output
        # 2. Just the last word as a function/case statement
        local found=false
        if echo "$cli_content" | grep -qi "\b$name\b"; then
            found=true
        elif echo "$cli_content" | grep -qi "$(echo "$name" | sed 's/.* //')"; then
            # Try searching for just the last word (e.g., "create" from "workflow create")
            found=true
        fi

        if [ "$found" = "true" ]; then
            testing::phase::check "CLI command: $label" true
            commands_passed=$((commands_passed + 1))

            # Record requirement evidence
            if [ "$requirements_json" != "[]" ] && [ "$requirements_json" != "null" ]; then
                local req_count
                req_count=$(echo "$requirements_json" | jq 'length')
                for j in $(seq 0 $((req_count - 1))); do
                    local req_id
                    req_id=$(echo "$requirements_json" | jq -r ".[$j]")
                    if [ -n "$req_id" ] && [ "$req_id" != "null" ]; then
                        testing::phase::add_requirement \
                            --id "$req_id" \
                            --status passed \
                            --evidence "CLI command '$name' present"
                    fi
                done
            fi
        else
            testing::phase::check "CLI command: $label" false
            commands_failed=$((commands_failed + 1))

            # Record requirement failure
            if [ "$requirements_json" != "[]" ] && [ "$requirements_json" != "null" ]; then
                local req_count
                req_count=$(echo "$requirements_json" | jq 'length')
                for j in $(seq 0 $((req_count - 1))); do
                    local req_id
                    req_id=$(echo "$requirements_json" | jq -r ".[$j]")
                    if [ -n "$req_id" ] && [ "$req_id" != "null" ]; then
                        testing::phase::add_requirement \
                            --id "$req_id" \
                            --status failed \
                            --evidence "Required CLI command '$name' missing"
                    fi
                done
            fi
        fi
    done

    echo "  ✓ CLI commands: $commands_passed passed, $commands_failed failed"
}

# Validate WebSocket endpoints from endpoints.json
testing::business::validate_websockets() {
    # Check if enabled
    if ! testing::business::is_enabled "websockets"; then
        return 0
    fi

    local endpoints_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/endpoints.json"

    if [ ! -f "$endpoints_file" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0
    fi

    local websockets_json
    if ! websockets_json=$(jq -r '.websockets // []' "$endpoints_file" 2>/dev/null); then
        return 0
    fi

    if [ "$websockets_json" = "[]" ]; then
        return 0
    fi

    local ws_count
    ws_count=$(echo "$websockets_json" | jq 'length')

    if [ "$ws_count" -eq 0 ]; then
        return 0
    fi

    echo "📋 Validating $ws_count WebSocket endpoints..."

    local search_dir="api"
    if [ ! -d "$search_dir" ]; then
        search_dir="."
    fi

    for i in $(seq 0 $((ws_count - 1))); do
        local ws
        ws=$(echo "$websockets_json" | jq -r ".[$i]")

        local path requirements_json
        path=$(echo "$ws" | jq -r '.path // ""')
        requirements_json=$(echo "$ws" | jq -r '.requirements // []')

        if [ -z "$path" ]; then
            continue
        fi

        # Search for WebSocket path in codebase
        local found=false
        if command -v rg >/dev/null 2>&1; then
            if rg --fixed-strings --quiet "$path" "$search_dir" 2>/dev/null; then
                found=true
            fi
        else
            if grep -r --fixed-strings --quiet "$path" "$search_dir" 2>/dev/null; then
                found=true
            fi
        fi

        if [ "$found" = "true" ]; then
            testing::phase::check "WebSocket: $path" true

            # Record requirements
            if [ "$requirements_json" != "[]" ] && [ "$requirements_json" != "null" ]; then
                local req_count
                req_count=$(echo "$requirements_json" | jq 'length')
                for j in $(seq 0 $((req_count - 1))); do
                    local req_id
                    req_id=$(echo "$requirements_json" | jq -r ".[$j]")
                    if [ -n "$req_id" ] && [ "$req_id" != "null" ]; then
                        testing::phase::add_requirement \
                            --id "$req_id" \
                            --status passed \
                            --evidence "WebSocket $path present"
                    fi
                done
            fi
        else
            testing::phase::check "WebSocket: $path" false
        fi
    done
}

export -f testing::business::is_enabled
export -f testing::business::validate_endpoints
export -f testing::business::validate_cli_commands
export -f testing::business::validate_websockets
