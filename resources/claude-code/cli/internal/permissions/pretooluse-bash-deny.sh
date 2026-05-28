#!/usr/bin/env bash
# Vrooli-managed PreToolUse hook for Claude Code.
#
# Backstop for the upstream `permissions.deny` Bash enforcement bug
# (anthropics/claude-code#18846, #29026). Pair every Vrooli-managed
# Bash deny rule with this hook so denial is reliable.
#
# Claude invokes this with the tool-call JSON on stdin. We extract the
# bash command, glob-match against any deny patterns passed as args,
# and exit 2 to refuse the call. Exit 0 lets Claude proceed.
#
# Every invocation appends a one-line audit record to
# ${HOME}/.claude/.vrooli-hooks/log so `permissions doctor` can confirm
# the hook is actually firing.

set -euo pipefail

LOG="${HOME}/.claude/.vrooli-hooks/log"
mkdir -p "$(dirname "$LOG")"

input="$(cat || true)"
ts="$(date -u +%FT%TZ)"

# Prefer jq when present; fall back to a permissive grep-extract so the
# hook still functions on minimal images where jq is missing.
cmd=""
if command -v jq >/dev/null 2>&1; then
    cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
else
    cmd="$(printf '%s' "$input" | sed -n 's/.*"command"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
fi

printf '%s tool=%s cmd=%q patterns=%s\n' "$ts" "Bash" "$cmd" "$*" >>"$LOG" 2>/dev/null || true

if [ -z "$cmd" ]; then
    exit 0
fi

for pat in "$@"; do
    # shellcheck disable=SC2053
    if [[ "$cmd" == $pat ]]; then
        printf '%s BLOCKED cmd=%q pat=%q\n' "$ts" "$cmd" "$pat" >>"$LOG" 2>/dev/null || true
        echo "vrooli-managed deny rule blocked this command: pattern=$pat" >&2
        exit 2
    fi
done

exit 0
