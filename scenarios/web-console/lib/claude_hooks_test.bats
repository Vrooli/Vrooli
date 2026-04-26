#!/usr/bin/env bats

# Guards the attribution-visibility contract for the Claude hook scripts:
# when a hook fires outside a web-console-managed shell (no
# WC_WEB_CONSOLE_SESSION_ID in env), the server will drop the event as
# conversation_target_missing. These tests lock in the opt-in debug warning
# that surfaces this silent-failure case — gated on WC_HOOK_WARN_UNATTRIBUTED=1
# so normal cwd-wide hook installations stay quiet.

setup() {
    TEST_TMP_DIR=$(mktemp -d)
    TEST_BIN="$TEST_TMP_DIR/bin"
    mkdir -p "$TEST_BIN"
    # Recording curl mock: appends each --data payload to $CURL_LOG, one
    # JSON body per line. Tests that don't care about bodies just ignore it.
    export CURL_LOG="$TEST_TMP_DIR/curl.log"
    cat > "$TEST_BIN/curl" <<'SCRIPT'
#!/usr/bin/env bash
data=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --data) data="$2"; shift 2 ;;
        -H|-X) shift 2 ;;
        -fsS|--fail|--silent|--show-error) shift ;;
        *) shift ;;
    esac
done
if [[ -n "${CURL_LOG:-}" && -n "$data" ]]; then
    printf '%s\n' "$data" >> "$CURL_LOG"
fi
exit 0
SCRIPT
    chmod +x "$TEST_BIN/curl"
    export PATH="$TEST_BIN:$PATH"

    export STOP_HOOK="$BATS_TEST_DIRNAME/claude-stop-hook.sh"
    export PROMPT_HOOK="$BATS_TEST_DIRNAME/claude-prompt-submit-hook.sh"

    unset WC_WEB_CONSOLE_SESSION_ID
    unset WC_HOOK_WARN_UNATTRIBUTED
}

teardown() {
    rm -rf "$TEST_TMP_DIR"
}

@test "stop-hook stays silent when session id is unset and warn flag is off" {
    run bash -c "echo '{\"last_assistant_message\":\"hi\"}' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" != *"stop-hook firing without"* ]]
}

@test "stop-hook emits stderr warning when session id is unset and warn flag is on" {
    export WC_HOOK_WARN_UNATTRIBUTED=1
    run bash -c "echo '{\"last_assistant_message\":\"hi\"}' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" == *"stop-hook firing without WC_WEB_CONSOLE_SESSION_ID"* ]]
}

@test "stop-hook is silent when session id is set even with warn flag" {
    export WC_HOOK_WARN_UNATTRIBUTED=1
    export WC_WEB_CONSOLE_SESSION_ID=abc
    run bash -c "echo '{\"last_assistant_message\":\"hi\"}' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" != *"stop-hook firing without"* ]]
}

@test "prompt-submit-hook emits stderr warning when session id is unset and warn flag is on" {
    export WC_HOOK_WARN_UNATTRIBUTED=1
    run bash -c "echo '{\"prompt\":\"hello\"}' | bash \"\$PROMPT_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" == *"prompt-submit-hook firing without WC_WEB_CONSOLE_SESSION_ID"* ]]
}

@test "stop-hook without transcript_path falls through to single POST (legacy behavior)" {
    run bash -c "echo '{\"last_assistant_message\":\"hi\"}' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [ -f "$CURL_LOG" ]
    # Exactly one POST containing the original message.
    [ "$(wc -l <"$CURL_LOG")" -eq 1 ]
    grep -q '"last_assistant_message":"hi"' "$CURL_LOG"
}

@test "stop-hook splits a multi-segment turn into one POST per assistant text segment" {
    # Build a synthetic transcript covering the failure mode we hit in
    # production: a turn that emits TEXT → tool_use → TEXT, where Claude
    # Code's last_assistant_message would only carry the trailing TEXT.
    transcript="$TEST_TMP_DIR/transcript.jsonl"
    {
        printf '%s\n' '{"type":"system"}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"the user prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"x"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"first segment"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"second segment"}]}}'
    } > "$transcript"

    payload=$(jq -nc --arg p "$transcript" '{transcript_path:$p, last_assistant_message:"second segment"}')
    run bash -c "printf '%s' '$payload' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]

    # Two POSTs in chronological order: "first segment" then "second segment".
    [ "$(wc -l <"$CURL_LOG")" -eq 2 ]
    line1=$(sed -n '1p' "$CURL_LOG")
    line2=$(sed -n '2p' "$CURL_LOG")
    [ "$(printf '%s' "$line1" | jq -r '.last_assistant_message')" = "first segment" ]
    [ "$(printf '%s' "$line2" | jq -r '.last_assistant_message')" = "second segment" ]
    # assistantResponse must mirror last_assistant_message so the server's
    # fallback path sees the same text.
    [ "$(printf '%s' "$line1" | jq -r '.assistantResponse')" = "first segment" ]
}

@test "stop-hook ignores tool_result user entries when finding the previous prompt" {
    # The cut-point should be the most recent REAL user prompt, not the
    # tool_result user-frames that Claude Code emits between assistant turns.
    transcript="$TEST_TMP_DIR/transcript.jsonl"
    {
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"older real prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"stale segment"}]}}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"current prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"alpha"}]}}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"beta"}]}}'
    } > "$transcript"

    payload=$(jq -nc --arg p "$transcript" '{transcript_path:$p, last_assistant_message:"beta"}')
    run bash -c "printf '%s' '$payload' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]

    [ "$(wc -l <"$CURL_LOG")" -eq 2 ]
    [ "$(sed -n '1p' "$CURL_LOG" | jq -r '.last_assistant_message')" = "alpha" ]
    [ "$(sed -n '2p' "$CURL_LOG" | jq -r '.last_assistant_message')" = "beta" ]
    # Importantly, the stale segment from the previous prompt must NOT appear.
    ! grep -q "stale segment" "$CURL_LOG"
}

@test "stop-hook race guard: last_assistant_message is appended when transcript lags behind" {
    # Simulates the production race where the Stop hook fires before
    # Claude Code has flushed the final assistant text segment to the
    # transcript JSONL. The hook payload's last_assistant_message is the
    # in-memory source of truth; the script must include it even when the
    # transcript-derived list doesn't yet contain it.
    transcript="$TEST_TMP_DIR/transcript.jsonl"
    {
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"first"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"second"}]}}'
        # NB: transcript intentionally OMITS the third "third (final)" segment
        # to model the disk-flush race. The hook payload still carries it.
    } > "$transcript"

    payload=$(jq -nc --arg p "$transcript" '{transcript_path:$p, last_assistant_message:"third (final)"}')
    run bash -c "printf '%s' '$payload' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]

    [ "$(wc -l <"$CURL_LOG")" -eq 3 ]
    [ "$(sed -n '1p' "$CURL_LOG" | jq -r '.last_assistant_message')" = "first" ]
    [ "$(sed -n '2p' "$CURL_LOG" | jq -r '.last_assistant_message')" = "second" ]
    [ "$(sed -n '3p' "$CURL_LOG" | jq -r '.last_assistant_message')" = "third (final)" ]
}

@test "stop-hook race guard: no duplicate when transcript already contains final segment" {
    transcript="$TEST_TMP_DIR/transcript.jsonl"
    {
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"alpha"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}'
        printf '%s\n' '{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"omega"}]}}'
    } > "$transcript"

    payload=$(jq -nc --arg p "$transcript" '{transcript_path:$p, last_assistant_message:"omega"}')
    run bash -c "printf '%s' '$payload' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]

    # Exactly 2 POSTs, omega appears once (no duplicate from race guard).
    [ "$(wc -l <"$CURL_LOG")" -eq 2 ]
    [ "$(grep -c '"last_assistant_message":"omega"' "$CURL_LOG")" -eq 1 ]
}

@test "stop-hook with single-segment transcript falls through to single POST" {
    transcript="$TEST_TMP_DIR/transcript.jsonl"
    {
        printf '%s\n' '{"type":"user","message":{"role":"user","content":"prompt"}}'
        printf '%s\n' '{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"only segment"}]}}'
    } > "$transcript"

    payload=$(jq -nc --arg p "$transcript" '{transcript_path:$p, last_assistant_message:"only segment"}')
    run bash -c "printf '%s' '$payload' | bash \"\$STOP_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]

    # Single segment → fall through to default single POST (avoid double work).
    [ "$(wc -l <"$CURL_LOG")" -eq 1 ]
    [ "$(jq -r '.last_assistant_message' "$CURL_LOG")" = "only segment" ]
}

@test "prompt-submit-hook is silent when session id is set" {
    export WC_HOOK_WARN_UNATTRIBUTED=1
    export WC_WEB_CONSOLE_SESSION_ID=abc
    run bash -c "echo '{\"prompt\":\"hello\"}' | bash \"\$PROMPT_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" != *"prompt-submit-hook firing without"* ]]
}
