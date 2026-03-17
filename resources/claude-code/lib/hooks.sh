#!/usr/bin/env bash
# Claude Code Hook Management Functions
# Handles adding and removing hooks from Claude settings files

# Source guard
[[ -n "${_CLAUDE_CODE_HOOKS_SOURCED:-}" ]] && return 0
_CLAUDE_CODE_HOOKS_SOURCED=1

# Source var.sh for directory variables if not already sourced
# shellcheck disable=SC1091
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

#######################################
# Add or update a hook in Claude settings
# Arguments:
#   $1 - event name (e.g., "Stop")
#   $2 - identifier (dedup key, e.g., "web-console-tts")
#   $3 - hook JSON (e.g., '{"type":"http","url":"http://..."}')
#   $4 - scope ("project" or "global")
# Returns: 0 on success, 1 on failure
#######################################
claude_code::hooks_add() {
    local event="${1:-}"
    local identifier="${2:-}"
    local hook_json="${3:-}"
    local scope="${4:-project}"

    if [[ -z "$event" || -z "$identifier" || -z "$hook_json" ]]; then
        log::error "Event, identifier, and hook_json are required"
        return 1
    fi

    if ! system::is_command jq; then
        log::error "jq is required for hook management"
        return 1
    fi

    # Determine settings file from scope
    local settings_file=""
    case "$scope" in
        project)
            settings_file="$CLAUDE_PROJECT_SETTINGS"
            ;;
        global)
            settings_file="$CLAUDE_SETTINGS_FILE"
            ;;
        *)
            log::error "Invalid scope: $scope (use 'project' or 'global')"
            return 1
            ;;
    esac

    # Ensure parent directory exists
    mkdir -p "${settings_file%/*}"

    # Read existing settings or default to empty object
    local settings="{}"
    if [[ -f "$settings_file" ]]; then
        settings=$(jq '.' "$settings_file" 2>/dev/null) || settings="{}"
    fi

    # Add _id to the hook JSON
    local hook_with_id
    hook_with_id=$(echo "$hook_json" | jq --arg id "$identifier" '. + {"_id": $id}' 2>/dev/null)
    if [[ $? -ne 0 || -z "$hook_with_id" ]]; then
        log::error "Invalid hook JSON"
        return 1
    fi

    # Build the updated settings using jq
    local updated
    updated=$(echo "$settings" | jq \
        --arg event "$event" \
        --arg id "$identifier" \
        --argjson hook "$hook_with_id" '
        # Ensure .hooks exists
        .hooks //= {} |
        # Ensure .hooks[event] is an array
        .hooks[$event] //= [] |
        # Check if a matcher group with this _id already exists
        if (.hooks[$event] | map(select(.hooks[]? | ._id == $id)) | length) > 0 then
            # Update existing: replace the hook entry in-place
            .hooks[$event] = [
                .hooks[$event][] |
                if (.hooks // [] | map(select(._id == $id)) | length) > 0 then
                    .hooks = [
                        .hooks[]? |
                        if ._id == $id then $hook else . end
                    ]
                else
                    .
                end
            ]
        else
            # Append new matcher group
            .hooks[$event] += [{"matcher": "*", "hooks": [$hook]}]
        end
    ' 2>/dev/null)

    if [[ $? -ne 0 || -z "$updated" ]]; then
        log::error "Failed to update hooks"
        return 1
    fi

    # Atomic write: temp file + mv
    local temp_file
    temp_file=$(mktemp "${settings_file}.XXXXXX")
    if echo "$updated" | jq '.' > "$temp_file" 2>/dev/null; then
        mv "$temp_file" "$settings_file"
    else
        rm -f "$temp_file"
        log::error "Failed to write settings file"
        return 1
    fi

    return 0
}

#######################################
# Remove a hook from Claude settings
# Arguments:
#   $1 - event name (e.g., "Stop")
#   $2 - identifier (dedup key, e.g., "web-console-tts")
#   $3 - scope ("project" or "global")
# Returns: 0 on success (idempotent)
#######################################
claude_code::hooks_remove() {
    local event="${1:-}"
    local identifier="${2:-}"
    local scope="${3:-project}"

    if [[ -z "$event" || -z "$identifier" ]]; then
        log::error "Event and identifier are required"
        return 1
    fi

    if ! system::is_command jq; then
        log::error "jq is required for hook management"
        return 1
    fi

    # Determine settings file from scope
    local settings_file=""
    case "$scope" in
        project)
            settings_file="$CLAUDE_PROJECT_SETTINGS"
            ;;
        global)
            settings_file="$CLAUDE_SETTINGS_FILE"
            ;;
        *)
            log::error "Invalid scope: $scope (use 'project' or 'global')"
            return 1
            ;;
    esac

    # If file doesn't exist, nothing to remove (idempotent)
    if [[ ! -f "$settings_file" ]]; then
        return 0
    fi

    # Read existing settings
    local settings
    settings=$(jq '.' "$settings_file" 2>/dev/null) || settings="{}"

    # Filter out matcher groups containing hooks with this _id,
    # then clean up empty event arrays and empty hooks object
    local updated
    updated=$(echo "$settings" | jq \
        --arg event "$event" \
        --arg id "$identifier" '
        if .hooks[$event] then
            # Filter out matcher groups where any hook has the matching _id
            .hooks[$event] = [
                .hooks[$event][] |
                select((.hooks // [] | map(select(._id == $id)) | length) == 0)
            ] |
            # Remove event key if array is now empty
            if (.hooks[$event] | length) == 0 then
                .hooks |= del(.[$event])
            else
                .
            end |
            # Remove hooks key if object is now empty
            if (.hooks | length) == 0 then
                del(.hooks)
            else
                .
            end
        else
            .
        end
    ' 2>/dev/null)

    if [[ $? -ne 0 || -z "$updated" ]]; then
        log::error "Failed to update hooks"
        return 1
    fi

    # Atomic write: temp file + mv
    local temp_file
    temp_file=$(mktemp "${settings_file}.XXXXXX")
    if echo "$updated" | jq '.' > "$temp_file" 2>/dev/null; then
        mv "$temp_file" "$settings_file"
    else
        rm -f "$temp_file"
        log::error "Failed to write settings file"
        return 1
    fi

    return 0
}
