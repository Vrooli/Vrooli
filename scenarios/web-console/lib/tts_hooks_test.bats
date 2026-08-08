#!/usr/bin/env bats

setup() {
    TEST_TMP_DIR=$(mktemp -d)
    export HOME="$TEST_TMP_DIR/home"
    mkdir -p "$HOME/.vrooli/processes/scenarios/web-console"
    mkdir -p "$HOME/.local/state/vrooli/web-console"

    cat > "$HOME/.vrooli/processes/scenarios/web-console/start-api.json" <<'EOF'
{
  "port": 17086
}
EOF
    echo "secret-token" > "$HOME/.local/state/vrooli/web-console/hook-token.txt"

    export APP_ROOT="$TEST_TMP_DIR/app"
    mkdir -p "$APP_ROOT/bin"
    cat > "$APP_ROOT/bin/resource-claude-code" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "${TEST_CLAUDE_CLI_ARGS}"
printf '%s\n' "${TEST_CLAUDE_CLI_OUTPUT}"
EOF
    chmod +x "$APP_ROOT/bin/resource-claude-code"
    export PATH="$APP_ROOT/bin:$PATH"

    export TEST_CLAUDE_CLI_ARGS="$TEST_TMP_DIR/claude-cli-args.txt"
    export TEST_CLAUDE_CLI_OUTPUT='{"status":"applied","code":"hook_reconciled","reason":"Claude hook was written to settings","settingsPath":"/tmp/project/.claude/settings.json"}'

    unset _WC_TTS_HOOKS_SOURCED
}

teardown() {
    rm -rf "$TEST_TMP_DIR"
}

@test "register_tts_hook delegates reconciliation to claude-code resource hooks seam" {
    run bash -lc 'source lib/tts-hooks.sh && wc::register_tts_hook'
    [ "$status" -eq 0 ]
    [[ "$output" == *"registered Stop hook"* ]]

    args=$(cat "$TEST_CLAUDE_CLI_ARGS")
    [[ "$args" == *"hooks reconcile"* ]]
    [[ "$args" == *"--event Stop"* ]]
    [[ "$args" == *"--id web-console-tts"* ]]
    [[ "$args" == *"--scope project"* ]]
    [[ "$args" == *'"type": "command"'* ]]
    [[ "$args" == *'claude-stop-hook.sh --url http://localhost:17086/api/v1/hooks/stop --token secret-token'* ]]
}

@test "register_tts_hook returns non-zero when reconcile fails" {
    export TEST_CLAUDE_CLI_OUTPUT='{"status":"failed","code":"hook_write_failed","reason":"write failed"}'

    run bash -lc 'source lib/tts-hooks.sh && wc::register_tts_hook'
    [ "$status" -eq 1 ]
    [[ "$output" == *"unexpected reconcile result"* ]]
}
