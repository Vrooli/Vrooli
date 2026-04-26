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
    payload="$(printf '%s' "$payload" | jq -c --arg sid "$session_id" '. + {web_console_session_id: $sid}')"
elif [[ "${WC_HOOK_WARN_UNATTRIBUTED:-}" == "1" ]]; then
    # Opt-in debug aid: surface claude invocations that fire outside any
    # web-console-managed shell (external terminal, pre-existing tmux
    # server, SSH session). The server will log conversation_target_missing
    # when this payload arrives, but that's only visible if someone is
    # watching `make logs`. Set WC_HOOK_WARN_UNATTRIBUTED=1 to see it here.
    echo "web-console: stop-hook firing without WC_WEB_CONSOLE_SESSION_ID — event will be dropped" >&2
fi

post_one() {
    curl -fsS \
        -H "Content-Type: application/json" \
        -H "X-Hook-Token: ${HOOK_TOKEN}" \
        --data "$1" \
        "$HOOK_URL" \
        >/dev/null
}

# Multi-segment turn extraction.
#
# Claude Code's Stop hook payload exposes only `last_assistant_message` —
# the text emitted after the model's *final* reply in the turn. But a
# single user-prompt cycle can produce text → tool_use → text, in which
# case the pre-tool text never reaches the conversation pipeline and is
# silently lost from the messages pane.
#
# When transcript_path is present we parse the JSONL ourselves, walk back
# to the previous *real* user prompt (skipping tool_result follow-ups),
# and POST one event per assistant text segment in chronological order.
# Server-side dedup (30s window) makes accidental re-fires harmless.
transcript_path="$(printf '%s' "$payload" | jq -r '.transcript_path // empty')"
if [[ -n "$transcript_path" && -r "$transcript_path" ]]; then
    segments_json="$(jq -cs '
        . as $arr
        | [
            $arr | to_entries[]
            | select(
                .value.type == "user"
                and (
                    (.value.message.content // []) | (
                        if type == "array"
                        then (map(.type // "") | index("tool_result")) == null
                        else true
                        end
                    )
                )
              )
            | .key
          ] as $idx
        | ($idx | last // -1) as $cut
        | $arr[$cut + 1:]
        | map(
            select(.type == "assistant")
            | (.message.content // [])
            | (if type == "array" then . else [] end)
            | map(select(.type == "text") | .text)
            | .[]
          )
        | map(select(. != null and . != ""))
    ' "$transcript_path" 2>/dev/null || echo '[]')"

    # Race guard: the Stop hook can fire before Claude Code flushes the
    # final assistant text to the transcript. The hook payload's
    # `last_assistant_message` is the in-memory source of truth for the
    # final segment, so always append it if the transcript-derived list
    # doesn't already end with it. This is idempotent — when the
    # transcript already contains the final segment the appended copy is
    # filtered out by exact equality with the last collected segment.
    lam="$(printf '%s' "$payload" | jq -r '.last_assistant_message // .assistantResponse // ""')"
    if [[ -n "$lam" ]]; then
        segments_json="$(printf '%s' "$segments_json" | jq -c --arg lam "$lam" '
            if (length == 0) or (.[-1] != $lam) then . + [$lam] else . end
        ')"
    fi

    count="$(printf '%s' "$segments_json" | jq 'length' 2>/dev/null || echo 0)"
    if [[ "$count" -gt 1 ]]; then
        for i in $(seq 0 $((count - 1))); do
            text="$(printf '%s' "$segments_json" | jq -r ".[$i]")"
            body="$(printf '%s' "$payload" | jq -c --arg t "$text" '. + {last_assistant_message: $t, assistantResponse: $t}')"
            post_one "$body"
        done
        exit 0
    fi
fi

post_one "$payload"
