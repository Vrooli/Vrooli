#!/usr/bin/env bash
# Vrooli-managed PreToolUse hook for Claude Code.
#
# Claude supplies Bash permission patterns in the form Bash(pattern), while
# the command in tool_input.command is the raw shell text. Normalize the
# native wrapper and the operator-home aliases before applying Bash's glob
# matcher. This hook receives command text as data; it never evaluates it.

set -euo pipefail

LOG="${HOME}/.claude/.vrooli-hooks/log"
LOG_MAX_BYTES="${VROOLI_HOOK_LOG_MAX_BYTES:-8388608}"
LOG_CMD_MAX_CHARS="${VROOLI_HOOK_LOG_CMD_MAX_CHARS:-300}"

mkdir -p "$(dirname "$LOG")"

rotate_log() {
    [ -f "$LOG" ] || return 0
    local size
    size="$(stat -c %s "$LOG" 2>/dev/null || echo 0)"
    if [ "$size" -gt "$LOG_MAX_BYTES" ]; then
        mv -f "$LOG" "${LOG}.1" 2>/dev/null || true
    fi
}
rotate_log

input="$(cat || true)"
ts="$(date -u +%FT%TZ)"

cmd=""
if command -v jq >/dev/null 2>&1; then
    cmd="$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
else
    cmd="$(printf '%s' "$input" | sed -n 's/.*"command"[[:space:]]*:[[:space:]]*"\([^\"]*\)".*/\1/p' | head -n1)"
fi

log_excerpt="$(printf '%s' "$cmd" | tr '\n' ' ' | cut -c1-"$LOG_CMD_MAX_CHARS")"
printf '%s tool=%s cmd=%q patterns=%d\n' "$ts" "Bash" "$log_excerpt" "$#" >>"$LOG" 2>/dev/null || true

[ -n "$cmd" ] || exit 0

normalize_pattern() {
    local pat="$1"

    case "$pat" in
        Bash\(*\))
            pat="${pat#Bash(}"
            pat="${pat%)}"
            ;;
    esac

    # These are literal aliases in a Claude permission pattern, not shell
    # expansions. Resolve them as text so the command is still only matched.
    pat="${pat//\$HOME/$HOME}"
    pat="${pat//\~/$HOME}"
    printf '%s' "$pat"
}

for raw_pattern in "$@"; do
    pattern="$(normalize_pattern "$raw_pattern")"
    [ -n "$pattern" ] || continue

    # Deliberately use the normalized value as a Bash glob. The command is
    # quoted on the left and is never passed to eval, bash -c, or a shell.
    # shellcheck disable=SC2053
    if [[ "$cmd" == $pattern ]]; then
        printf '%s BLOCKED cmd=%q pat=%q normalized=%q\n' "$ts" "$log_excerpt" "$raw_pattern" "$pattern" >>"$LOG" 2>/dev/null || true
        echo "vrooli-managed deny rule blocked this command: pattern=$raw_pattern" >&2
        exit 2
    fi
done

exit 0
