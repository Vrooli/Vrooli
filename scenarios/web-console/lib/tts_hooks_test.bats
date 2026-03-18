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
    mkdir -p "$APP_ROOT/resources/claude-code/lib" "$APP_ROOT/resources/claude-code/config"
    mkdir -p "$APP_ROOT/scripts/lib/utils" "$APP_ROOT/scripts/lib/system"
    cat > "$APP_ROOT/resources/claude-code/config/defaults.sh" <<'EOF'
#!/usr/bin/env bash
CLAUDE_CONFIG_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
EOF
    cat > "$APP_ROOT/resources/claude-code/lib/settings.sh" <<'EOF'
#!/usr/bin/env bash
claude_code::export_settings_context() { :; }
EOF
    cat > "$APP_ROOT/resources/claude-code/lib/hooks.sh" <<'EOF'
#!/usr/bin/env bash
claude_code::hooks_reconcile() {
    printf '%s\n' "$*" > "${TEST_CLAUDE_CLI_ARGS}"
    printf '%s\n' "${TEST_CLAUDE_CLI_OUTPUT}"
}
claude_code::hooks_remove() { :; }
EOF
    cat > "$APP_ROOT/scripts/lib/utils/var.sh" <<'EOF'
#!/usr/bin/env bash
:
EOF
    cat > "$APP_ROOT/scripts/lib/utils/log.sh" <<'EOF'
#!/usr/bin/env bash
:
EOF
    cat > "$APP_ROOT/scripts/lib/system/system_commands.sh" <<'EOF'
#!/usr/bin/env bash
:
EOF

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
    [[ "$args" == Stop\ web-console-tts*project ]]
    [[ "$args" == *'"type": "command"'* ]]
    [[ "$args" == *'claude-stop-hook.sh --url http://localhost:17086/api/v1/hooks/stop --token secret-token'* ]]
}

@test "register_tts_hook returns non-zero when reconcile fails" {
    export TEST_CLAUDE_CLI_OUTPUT='{"status":"failed","code":"hook_write_failed","reason":"write failed"}'

    run bash -lc 'source lib/tts-hooks.sh && wc::register_tts_hook'
    [ "$status" -eq 1 ]
    [[ "$output" == *"unexpected reconcile result"* ]]
}
