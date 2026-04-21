#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo "Usage: $0 --url <hook-url> --token <hook-token>" >&2
    exit 2
}

HOOK_URL=""
HOOK_TOKEN=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --url)
            HOOK_URL="${2:-}"
            shift 2
            ;;
        --token)
            HOOK_TOKEN="${2:-}"
            shift 2
            ;;
        *)
            usage
            ;;
    esac
done

if [[ -z "$HOOK_URL" || -z "$HOOK_TOKEN" ]]; then
    usage
fi

payload="$(cat)"
if [[ -z "$payload" ]]; then
    exit 0
fi

session_id="${WC_WEB_CONSOLE_SESSION_ID:-}"
if [[ -n "$session_id" ]]; then
    payload="$(printf '%s' "$payload" | jq --arg sid "$session_id" '. + {web_console_session_id: $sid}')"
elif [[ "${WC_HOOK_WARN_UNATTRIBUTED:-}" == "1" ]]; then
    # Opt-in debug aid: surface claude invocations that fire outside any
    # web-console-managed shell (external terminal, pre-existing tmux
    # server, SSH session). The server will log conversation_target_missing
    # when this payload arrives, but that's only visible if someone is
    # watching `make logs`. Set WC_HOOK_WARN_UNATTRIBUTED=1 to see it here.
    echo "web-console: stop-hook firing without WC_WEB_CONSOLE_SESSION_ID — event will be dropped" >&2
fi

curl -fsS \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: ${HOOK_TOKEN}" \
    --data "$payload" \
    "$HOOK_URL" \
    >/dev/null
