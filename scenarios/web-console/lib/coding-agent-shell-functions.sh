#!/usr/bin/env bash

# Ambient coding-agent adoption for interactive operator shells.
#
# The command -v check is intentionally the only preflight. If the shared
# launcher is absent, the original binary is executed directly. If the
# launcher exists but fails, its exit status is preserved; silently starting a
# second agent would make attribution and operator intent ambiguous.

_vrooli_launch_coding_agent() {
    local agent="$1"
    shift
    if command -v vrooli-agent-launcher >/dev/null 2>&1; then
        command vrooli-agent-launcher --agent "$agent" -- "$@"
        return $?
    fi
    command "$agent" "$@"
}

claude() {
    _vrooli_launch_coding_agent claude "$@"
}

codex() {
    _vrooli_launch_coding_agent codex "$@"
}

grok() {
    _vrooli_launch_coding_agent grok "$@"
}
