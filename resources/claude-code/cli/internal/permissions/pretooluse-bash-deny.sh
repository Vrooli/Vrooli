#!/usr/bin/env bash
# Vrooli-managed PreToolUse hook for Claude Code. Command text is data: this
# hook never evaluates it. Filesystem deletion is checked by resolved paths,
# while native Bash patterns continue to cover non-filesystem command families.

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
log_excerpt=""
block() {
    local reason="$1"
    printf '%s BLOCKED cmd=%q reason=%q\n' "$ts" "$log_excerpt" "$reason" >>"$LOG" 2>/dev/null || true
    echo "vrooli-managed deny rule blocked this command: $reason" >&2
    exit 2
}

if ! command -v python3 >/dev/null 2>&1; then
    block "python3 is required for path-aware destructive-command checks"
fi
if ! cmd="$(printf '%s' "$input" | python3 -c 'import json,sys; value=json.load(sys.stdin); command=value.get("tool_input",{}).get("command"); assert isinstance(command,str) and command; print(command, end="")')"; then
    block "malformed hook input or missing tool_input.command"
fi

log_excerpt="$(printf '%s' "$cmd" | tr '\n' ' ' | cut -c1-"$LOG_CMD_MAX_CHARS")"
printf '%s tool=%s cmd=%q patterns=%d\n' "$ts" "Bash" "$log_excerpt" "$#" >>"$LOG" 2>/dev/null || true

if ! VROOLI_HOOK_COMMAND="$cmd" VROOLI_HOOK_REPO_ROOT="${VROOLI_REPO_ROOT:-}" python3 - "$HOME" <<'PY'
import os
import re
import shlex
import sys

command = os.environ["VROOLI_HOOK_COMMAND"]
home = os.path.realpath(sys.argv[1])
repo_raw = os.environ.get("VROOLI_HOOK_REPO_ROOT", "")
repo = os.path.realpath(repo_raw) if repo_raw else ""
roots = ["/tmp", "/var/tmp"]
roots += [x for x in os.environ.get("VROOLI_EPHEMERAL_ROOTS", "").split(":") if x]
roots = [os.path.realpath(x) for x in roots]

def deny(reason):
    print(reason, file=sys.stderr)
    raise SystemExit(2)

def resolve(raw):
    value = os.path.expanduser(os.path.expandvars(raw))
    if re.search(r"\$(?:\{)?[A-Za-z_][A-Za-z0-9_]*(?:\})?", value):
        deny("unresolved environment variable in destructive path")
    if not os.path.isabs(value):
        deny("destructive path is not absolute")
    return os.path.realpath(value)

def check(raw):
    path = resolve(raw)
    if path == "/" or path == home or path.startswith(home + os.sep) or (repo and (path == repo or path.startswith(repo + os.sep))):
        deny("destructive target is a protected root")
    if path.startswith("/") and path.count("/") == 1:
        deny("destructive target is a depth-one system directory")
    for root in roots:
        try:
            if path != root and os.path.commonpath([path, root]) == root:
                return
        except ValueError:
            pass
    deny("destructive target is outside an approved ephemeral root")

try:
    lexer = shlex.shlex(command, posix=True, punctuation_chars=";&|<>()")
    lexer.whitespace_split = True
    tokens = list(lexer)
except ValueError:
    deny("malformed shell command")

destructive = re.search(r"(?:^|\s)(?:sudo\s+)?(?:rm|find|truncate)(?:\s|$)", command)
if any(x in {";", "&&", "||", "|", "<", ">", "(", ")"} for x in tokens):
    if destructive:
        deny("compound destructive shell command requires review")
    raise SystemExit(0)
if re.search(r"(?:^|\s)(?:bash|sh|zsh)\s+-c(?:\s|$)", command) and destructive:
    deny("destructive command through a shell interpreter requires review")

def index(name):
    for i, token in enumerate(tokens):
        if token == name or token.endswith("/" + name):
            return i
    return -1

i = index("rm")
if i >= 0:
    targets = []
    end_options = False
    for token in tokens[i + 1:]:
        if not end_options and token == "--":
            end_options = True
            continue
        if not end_options and token.startswith("-"):
            continue
        targets.append(token)
    if not targets:
        deny("rm has no explicit target")
    for target in targets:
        if any(c in target for c in "*?["):
            deny("destructive path globs require review")
        check(target)
    raise SystemExit(0)

i = index("find")
if i >= 0 and "-delete" in tokens[i + 1:]:
    targets = []
    for token in tokens[i + 1:]:
        if token.startswith("-"):
            break
        targets.append(token)
    if not targets:
        deny("find -delete has no explicit root")
    for target in targets:
        check(target)
    raise SystemExit(0)

i = index("truncate")
if i >= 0:
    targets = []
    skip = False
    for token in tokens[i + 1:]:
        if skip:
            skip = False
        elif token in {"-s", "--size"}:
            skip = True
        elif not token.startswith("-"):
            targets.append(token)
    if not targets:
        deny("truncate has no explicit target")
    for target in targets:
        check(target)
    raise SystemExit(0)
raise SystemExit(0)
PY
then
    block "path-aware destructive filesystem policy"
fi

normalize_pattern() {
    local pat="$1"
    case "$pat" in
        Bash\(*\))
            pat="${pat#Bash(}"
            pat="${pat%)}"
            ;;
    esac
    pat="${pat//\$HOME/$HOME}"
    pat="${pat//\~/$HOME}"
    printf '%s' "$pat"
}

for raw_pattern in "$@"; do
    pattern="$(normalize_pattern "$raw_pattern")"
    [ -n "$pattern" ] || continue
    case "$pattern" in
        rm\ *|sudo\ rm\ *|find\ *|sudo\ find\ *|truncate\ *|sudo\ truncate\ *)
            continue
            ;;
    esac
    # shellcheck disable=SC2053
    if [[ "$cmd" == $pattern ]]; then
        block "native deny pattern=$raw_pattern"
    fi
done
exit 0
