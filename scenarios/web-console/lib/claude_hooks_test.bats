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
    cat > "$TEST_BIN/curl" <<'SCRIPT'
#!/usr/bin/env bash
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

@test "prompt-submit-hook is silent when session id is set" {
    export WC_HOOK_WARN_UNATTRIBUTED=1
    export WC_WEB_CONSOLE_SESSION_ID=abc
    run bash -c "echo '{\"prompt\":\"hello\"}' | bash \"\$PROMPT_HOOK\" --url http://127.0.0.1:1/x --token t"
    [ "$status" -eq 0 ]
    [[ "$output" != *"prompt-submit-hook firing without"* ]]
}
