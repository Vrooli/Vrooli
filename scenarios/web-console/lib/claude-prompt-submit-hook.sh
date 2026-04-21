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

prompt="$(printf '%s' "$payload" | jq -r '.prompt // empty')"
if [[ -z "$prompt" ]]; then
    exit 0
fi

session_id="${WC_WEB_CONSOLE_SESSION_ID:-}"
if [[ -z "$session_id" && "${WC_HOOK_WARN_UNATTRIBUTED:-}" == "1" ]]; then
    echo "web-console: prompt-submit-hook firing without WC_WEB_CONSOLE_SESSION_ID — event will be dropped" >&2
fi

body="$(jq -nc --arg p "$prompt" --arg sid "$session_id" '{userPrompt: $p, webConsoleSessionId: $sid}')"

curl -fsS \
    -H "Content-Type: application/json" \
    -H "X-Hook-Token: ${HOOK_TOKEN}" \
    --data "$body" \
    "$HOOK_URL" \
    >/dev/null
