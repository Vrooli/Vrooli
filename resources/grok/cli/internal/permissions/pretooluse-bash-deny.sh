#!/usr/bin/env bash
# Vrooli-managed PreToolUse backstop hook for Grok Build.
#
# Grok evaluates PreToolUse hooks BEFORE the permission system and they
# apply even under --always-approve / bypassPermissions, so this hook is
# the hard enforcer paired with every Vrooli-managed Bash deny rule.
#
# Grok sends the tool-call JSON on stdin. We extract the shell command
# (`.toolInput.command`, camelCase — Grok's schema), glob-match it against
# the deny globs passed as args, and on a match emit an explicit deny
# decision on stdout (Grok honors a `deny` decision regardless of exit
# code) plus exit 2. On no match we stay silent and exit 0 so the call
# falls through to Grok's normal policy (we do NOT emit `allow`, which
# would short-circuit Grok's own approval prompts for everything else).
#
# Deliberately no `set -e`: a blocking hook must always run to completion
# and emit its decision. Grok fails OPEN on any hook error, so an early
# exit would silently disable the block.
#
# Every invocation appends a one-line audit record to
# ${GROK_HOOKS_DIR:-$HOME/.grok/hooks}/.vrooli-deny-log so
# `permissions doctor` can confirm the hook is actually firing.

set -u

HOOKS_DIR="${GROK_HOOKS_DIR:-${HOME}/.grok/hooks}"
LOG="${HOOKS_DIR}/.vrooli-deny-log"
mkdir -p "${HOOKS_DIR}" 2>/dev/null || true

input="$(cat 2>/dev/null || true)"
ts="$(date -u +%FT%TZ 2>/dev/null || echo unknown)"

# Prefer jq; fall back to a permissive sed-extract so the hook still
# functions on minimal images where jq is missing.
cmd=""
if command -v jq >/dev/null 2>&1; then
    cmd="$(printf '%s' "$input" | jq -r '.toolInput.command // empty' 2>/dev/null || true)"
fi
if [ -z "$cmd" ]; then
    cmd="$(printf '%s' "$input" | sed -n 's/.*"command"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
fi

printf '%s tool=Bash cmd=%q globs=%s\n' "$ts" "$cmd" "$*" >>"$LOG" 2>/dev/null || true

if [ -z "$cmd" ]; then
    exit 0
fi

for glob in "$@"; do
    # shellcheck disable=SC2053
    if [[ "$cmd" == $glob ]]; then
        printf '%s BLOCKED cmd=%q glob=%q\n' "$ts" "$cmd" "$glob" >>"$LOG" 2>/dev/null || true
        printf '{"decision":"deny","reason":"vrooli-managed deny rule blocked this command (pattern: Bash(%s))"}\n' "$glob"
        exit 2
    fi
done

# No match: stay neutral so Grok's normal policy/prompt still applies.
exit 0
