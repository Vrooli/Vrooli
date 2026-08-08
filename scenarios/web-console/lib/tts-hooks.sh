#!/usr/bin/env bash
# TTS Hook Registration for Web Console
#
# Registers/deregisters a Claude Code Stop hook so that AI responses are
# POSTed to the web-console backend for auto-TTS playback.
#
# Functions:
#   wc::register_tts_hook   - Register the Stop hook (idempotent)
#   wc::deregister_tts_hook - Deregister the Stop hook (idempotent)

# Source guard
[[ -n "${_WC_TTS_HOOKS_SOURCED:-}" ]] && return 0
_WC_TTS_HOOKS_SOURCED=1

# Resolve APP_ROOT (Vrooli project root)
_WC_SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
_WC_SCENARIO_DIR="$(builtin cd "${_WC_SCRIPT_DIR}/.." && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${_WC_SCRIPT_DIR}/../../.." && builtin pwd)}"

# Hook identity constants
_WC_HOOK_EVENT="Stop"
_WC_HOOK_ID="web-console-tts"
_WC_PROMPT_HOOK_EVENT="UserPromptSubmit"
_WC_PROMPT_HOOK_ID="web-console-prompt"
_WC_HOOK_SCOPE="project"

_wc::source_claude_code_hooks() {
    _wc::claude_code_cli >/dev/null 2>&1
}

_wc::claude_code_cli() {
    if command -v resource-claude-code >/dev/null 2>&1; then
        command -v resource-claude-code
        return 0
    fi
    if [[ -n "${VROOLI_BIN_DIR:-}" && -x "${VROOLI_BIN_DIR}/resource-claude-code" ]]; then
        echo "${VROOLI_BIN_DIR}/resource-claude-code"
        return 0
    fi
    return 1
}

_wc::hooks_reconcile() {
    local cli
    cli=$(_wc::claude_code_cli) || return 1
    "$cli" hooks reconcile \
        --event "$1" \
        --id "$2" \
        --hook-json "$3" \
        --scope "$4"
}

_wc::hooks_remove() {
    local cli
    cli=$(_wc::claude_code_cli) || return 1
    "$cli" hooks remove \
        --event "$1" \
        --id "$2" \
        --scope "$3"
}

#######################################
# Resolve the API port from the lifecycle process state.
# Outputs the port number on stdout; returns 1 if unavailable.
#######################################
_wc::resolve_api_port() {
    local state_file="$HOME/.vrooli/processes/scenarios/web-console/start-api.json"
    if [[ -f "$state_file" ]] && command -v jq &>/dev/null; then
        local port
        port=$(jq -r '.port // empty' "$state_file" 2>/dev/null)
        if [[ -n "$port" && "$port" =~ ^[0-9]+$ ]]; then
            echo "$port"
            return 0
        fi
    fi

    # Fallback: check API_PORT env var (set by lifecycle during develop phase)
    if [[ -n "${API_PORT:-}" && "${API_PORT}" =~ ^[0-9]+$ ]]; then
        echo "$API_PORT"
        return 0
    fi

    return 1
}

#######################################
# Resolve the hook auth token written by the Go backend.
# Reads the canonical api-core/storage state path on Linux.
# Outputs the token on stdout; returns 1 if unavailable.
#######################################
_wc::resolve_hook_token() {
    local token=""

    local xdg_state="${XDG_STATE_HOME:-$HOME/.local/state}"
    local primary_path="${xdg_state}/vrooli/web-console/hook-token.txt"
    if [[ -f "$primary_path" ]]; then
        token=$(<"$primary_path")
        token="${token%%[[:space:]]}"  # trim trailing whitespace/newline
        if [[ -n "$token" ]]; then
            echo "$token"
            return 0
        fi
    fi

    return 1
}

#######################################
# Register the Claude Code Stop hook for auto-TTS.
#
# Prerequisites:
#   - The web-console API must be running (hook token file must exist)
#   - The claude-code resource hooks library must be available
#
# Silently skips if prerequisites are not met (idempotent, safe to call anytime).
#######################################
wc::register_tts_hook() {
    if ! _wc::source_claude_code_hooks; then
        return 0
    fi

    # Resolve the API port
    local api_port
    if ! api_port=$(_wc::resolve_api_port); then
        echo "tts-hook: skipping registration -- API port not available" >&2
        return 0
    fi

    # Resolve the hook auth token (retry up to 5 times to handle startup race
    # where the API hasn't created hook-token.txt yet)
    local hook_token=""
    local max_attempts=5
    local attempt=0
    while (( attempt < max_attempts )); do
        if hook_token=$(_wc::resolve_hook_token); then
            break
        fi
        (( attempt++ ))
        if (( attempt < max_attempts )); then
            echo "tts-hook: hook token not found, retrying in 2s (${attempt}/${max_attempts})..." >&2
            sleep 2
        fi
    done
    if [[ -z "$hook_token" ]]; then
        echo "tts-hook: WARNING -- hook token not available after ${max_attempts} attempts. Auto-TTS will not work until hook is registered." >&2
        echo "tts-hook: Run 'source lib/tts-hooks.sh && wc::register_tts_hook' manually after API starts." >&2
        return 2
    fi

    # Build the hook JSON payload
    local hook_json
    hook_json=$(cat <<ENDJSON
{
  "type": "command",
  "command": "bash ${_WC_SCENARIO_DIR}/lib/claude-stop-hook.sh --url http://localhost:${api_port}/api/v1/hooks/stop --token ${hook_token}",
  "timeout": 30
}
ENDJSON
)

    local result=""
    if ! result=$(_wc::hooks_reconcile "$_WC_HOOK_EVENT" "$_WC_HOOK_ID" "$hook_json" "$_WC_HOOK_SCOPE"); then
        echo "tts-hook: reconciliation failed" >&2
        if [[ -n "$result" ]]; then
            echo "$result" >&2
        fi
        return 1
    fi

    local status=""
    local reason=""
    if command -v jq &>/dev/null; then
        status=$(echo "$result" | jq -r '.status // empty' 2>/dev/null || true)
        reason=$(echo "$result" | jq -r '.reason // empty' 2>/dev/null || true)
    fi

    case "$status" in
        applied)
            echo "tts-hook: registered Stop hook -> localhost:${api_port}"
            ;;
        unchanged)
            echo "tts-hook: hook already healthy -> localhost:${api_port}"
            ;;
        *)
            echo "tts-hook: unexpected reconcile result${reason:+ -- ${reason}}" >&2
            echo "$result" >&2
            return 1
            ;;
    esac

    # Register the UserPromptSubmit hook
    local prompt_hook_json
    prompt_hook_json=$(cat <<ENDJSON
{
  "type": "command",
  "command": "bash ${_WC_SCENARIO_DIR}/lib/claude-prompt-submit-hook.sh --url http://localhost:${api_port}/api/v1/hooks/prompt-submit --token ${hook_token}",
  "timeout": 10
}
ENDJSON
)

    local prompt_result=""
    if ! prompt_result=$(_wc::hooks_reconcile "$_WC_PROMPT_HOOK_EVENT" "$_WC_PROMPT_HOOK_ID" "$prompt_hook_json" "$_WC_HOOK_SCOPE"); then
        echo "tts-hook: prompt-submit reconciliation failed" >&2
        if [[ -n "$prompt_result" ]]; then
            echo "$prompt_result" >&2
        fi
        return 1
    fi

    local prompt_status=""
    local prompt_reason=""
    if command -v jq &>/dev/null; then
        prompt_status=$(echo "$prompt_result" | jq -r '.status // empty' 2>/dev/null || true)
        prompt_reason=$(echo "$prompt_result" | jq -r '.reason // empty' 2>/dev/null || true)
    fi

    case "$prompt_status" in
        applied)
            echo "tts-hook: registered UserPromptSubmit hook -> localhost:${api_port}"
            ;;
        unchanged)
            echo "tts-hook: prompt hook already healthy -> localhost:${api_port}"
            ;;
        *)
            echo "tts-hook: unexpected prompt reconcile result${prompt_reason:+ -- ${prompt_reason}}" >&2
            echo "$prompt_result" >&2
            return 1
            ;;
    esac

    return 0
}

#######################################
# Deregister the Claude Code Stop hook for auto-TTS.
#
# Silently skips if the hooks library is not available (idempotent).
#######################################
wc::deregister_tts_hook() {
    if ! _wc::source_claude_code_hooks; then
        return 0
    fi

    # Remove the hooks
    if _wc::hooks_remove "$_WC_HOOK_EVENT" "$_WC_HOOK_ID" "$_WC_HOOK_SCOPE" >/dev/null; then
        echo "tts-hook: deregistered Stop hook"
    else
        echo "tts-hook: deregistration failed" >&2
        return 1
    fi

    _wc::hooks_remove "$_WC_PROMPT_HOOK_EVENT" "$_WC_PROMPT_HOOK_ID" "$_WC_HOOK_SCOPE" >/dev/null
    echo "tts-hook: deregistered UserPromptSubmit hook"

    return 0
}
