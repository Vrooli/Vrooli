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
_WC_HOOK_SCOPE="project"

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
# Checks api-core/storage XDG state path first, then the fallback path.
# Outputs the token on stdout; returns 1 if unavailable.
#######################################
_wc::resolve_hook_token() {
    local token=""

    # Primary: XDG state path used by api-core/storage on Linux
    # ~/.local/state/vrooli/web-console/hook-token.txt
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

    # Fallback: relative to scenario root (matches fallbackHookTokenPath in Go)
    local fallback_path="${_WC_SCENARIO_DIR}/store/hook-token.txt"
    if [[ -f "$fallback_path" ]]; then
        token=$(<"$fallback_path")
        token="${token%%[[:space:]]}"
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
    # Check if the hooks library is available
    local hooks_lib="${APP_ROOT}/resources/claude-code/lib/hooks.sh"
    if [[ ! -f "$hooks_lib" ]]; then
        # Claude Code resource not installed -- skip silently
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

    # Source required Vrooli utilities (hooks.sh depends on var.sh, log, system_commands)
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/system/system_commands.sh" 2>/dev/null || true

    # Source the hooks library
    # shellcheck disable=SC1091
    if ! source "$hooks_lib" 2>/dev/null; then
        echo "tts-hook: failed to source hooks library" >&2
        return 0
    fi

    # Build the hook JSON payload
    local hook_json
    hook_json=$(cat <<ENDJSON
{
  "type": "http",
  "url": "http://localhost:${api_port}/api/v1/hooks/stop",
  "headers": { "X-Hook-Token": "${hook_token}" },
  "timeout": 30
}
ENDJSON
)

    # Register (or update) the hook
    if claude_code::hooks_add "$_WC_HOOK_EVENT" "$_WC_HOOK_ID" "$hook_json" "$_WC_HOOK_SCOPE"; then
        echo "tts-hook: registered Stop hook -> localhost:${api_port}"
    else
        echo "tts-hook: registration failed (non-fatal)" >&2
    fi

    return 0
}

#######################################
# Deregister the Claude Code Stop hook for auto-TTS.
#
# Silently skips if the hooks library is not available (idempotent).
#######################################
wc::deregister_tts_hook() {
    # Check if the hooks library is available
    local hooks_lib="${APP_ROOT}/resources/claude-code/lib/hooks.sh"
    if [[ ! -f "$hooks_lib" ]]; then
        return 0
    fi

    # Source required Vrooli utilities
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/lib/system/system_commands.sh" 2>/dev/null || true

    # Source the hooks library
    # shellcheck disable=SC1091
    if ! source "$hooks_lib" 2>/dev/null; then
        echo "tts-hook: failed to source hooks library" >&2
        return 0
    fi

    # Remove the hook
    if claude_code::hooks_remove "$_WC_HOOK_EVENT" "$_WC_HOOK_ID" "$_WC_HOOK_SCOPE"; then
        echo "tts-hook: deregistered Stop hook"
    else
        echo "tts-hook: deregistration failed (non-fatal)" >&2
    fi

    return 0
}
