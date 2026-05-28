#!/usr/bin/env bash
# Canonical coding-agent permission policy for Vrooli.
#
# Applies the same intent (deny / ask) to every installed coding-agent
# resource (resource-claude-code, resource-codex, resource-opencode) using
# each agent's native pattern syntax. Re-run idempotently on any fresh
# machine.
#
# This script is itself an agent-gated caller, so it passes
# --i-was-explicitly-authorized to each per-resource invocation. That flag
# means "a human has authorized this specific change"; do not invoke this
# script from an unattended agent run without an explicit OK.
#
# To extend: add the pattern to the matching DENY_* / ASK_* arrays below
# (one array per agent because their pattern syntaxes differ) and re-run.

set -euo pipefail

# shellcheck disable=SC2034  # *_PATTERNS_* arrays are used via nameref `local -n`
# shellcheck disable=SC2016  # `$HOME` is intentionally literal in deny patterns

OVERRIDE="--i-was-explicitly-authorized"

# Whether each agent CLI is on PATH. Missing agents are skipped, not failed.
have() { command -v "$1" >/dev/null 2>&1; }

# --- DENY ---------------------------------------------------------------
# "Never, no matter what." Git-history mutations, wholesale filesystem
# destruction, host power. Drawn from feedback_no_git_mutations and
# feedback_no_git_stash memory plus first-principles host-safety.

# Claude / OpenCode: glob with trailing *. Codex: stores verbatim, no native enforcement.
DENY_PATTERNS_CLAUDE=(
  'Bash(git commit*)'
  'Bash(git push*)'
  'Bash(git reset --hard*)'
  'Bash(git reset --merge*)'
  'Bash(git revert*)'
  'Bash(git stash*)'
  'Bash(git rebase*)'
  'Bash(git branch -D*)'
  'Bash(git branch --delete --force*)'
  'Bash(git checkout .*)'
  'Bash(git restore .*)'
  'Bash(git clean -f*)'
  'Bash(git worktree add*)'
  'Bash(rm -rf /*)'
  'Bash(rm -rf ~*)'
  'Bash(rm -rf $HOME*)'
  'Bash(sudo rm -rf*)'
  'Bash(dd *of=/dev/sd*)'
  'Bash(mkfs*)'
  'Bash(shutdown*)'
  'Bash(reboot*)'
  'Bash(systemctl poweroff*)'
  'Bash(systemctl reboot*)'
)

# OpenCode uses bare glob (no Bash() wrapper); last-match-wins, alphabetized.
DENY_PATTERNS_OPENCODE=(
  'git commit*'
  'git push*'
  'git reset --hard*'
  'git reset --merge*'
  'git revert*'
  'git stash*'
  'git rebase*'
  'git branch -D*'
  'git branch --delete --force*'
  'git checkout .*'
  'git restore .*'
  'git clean -f*'
  'git worktree add*'
  'rm -rf /*'
  'rm -rf ~*'
  'rm -rf $HOME*'
  'sudo rm -rf*'
  'dd *of=/dev/sd*'
  'mkfs*'
  'shutdown*'
  'reboot*'
  'systemctl poweroff*'
  'systemctl reboot*'
)

# Codex: same strings as opencode (intent only — Codex does not enforce per-pattern).
DENY_PATTERNS_CODEX=("${DENY_PATTERNS_OPENCODE[@]}")

# --- ASK ----------------------------------------------------------------
# "Stop and ask the human." Package installs (memory: never install w/o
# permission), sudo escalation, piped remote scripts, direct scenario-binary
# execution (CLAUDE.md rule 6).

ASK_PATTERNS_CLAUDE=(
  'Bash(npm install*)'
  'Bash(npm i *)'
  'Bash(pnpm add*)'
  'Bash(pnpm install --save*)'
  'Bash(yarn add*)'
  'Bash(go get*)'
  'Bash(pip install*)'
  'Bash(uv add*)'
  'Bash(cargo add*)'
  'Bash(brew install*)'
  'Bash(apt-get install*)'
  'Bash(snap install*)'
  'Bash(sudo *)'
  'Bash(iptables*)'
  'Bash(ufw disable*)'
  'Bash(./scenarios/*/api/scenario-api*)'
  'Bash(nohup ./*)'
  'Bash(curl * | sh*)'
  'Bash(curl * | bash*)'
  'Bash(wget * | sh*)'
)

ASK_PATTERNS_OPENCODE=(
  'npm install*'
  'npm i *'
  'pnpm add*'
  'pnpm install --save*'
  'yarn add*'
  'go get*'
  'pip install*'
  'uv add*'
  'cargo add*'
  'brew install*'
  'apt-get install*'
  'snap install*'
  'sudo *'
  'iptables*'
  'ufw disable*'
  './scenarios/*/api/scenario-api*'
  'nohup ./*'
  'curl * | sh*'
  'curl * | bash*'
  'wget * | sh*'
)

ASK_PATTERNS_CODEX=("${ASK_PATTERNS_OPENCODE[@]}")

# --- Apply --------------------------------------------------------------

apply_one() {
  local agent="$1" verb="$2" pattern="$3"
  if ! "$agent" permissions "$verb" "$OVERRIDE" "$pattern" >/dev/null 2>&1; then
    echo "    -> FAILED: $agent $verb $pattern" >&2
    return 1
  fi
}

apply_agent() {
  local label="$1" agent="$2" deny_var="$3" ask_var="$4"
  if ! have "$agent"; then
    echo "==> $label: not installed (skipped)"
    return 0
  fi
  echo "==> $label: applying policy"
  local -n DENY="$deny_var"
  local -n ASK="$ask_var"
  local fails=0
  for p in "${DENY[@]}"; do apply_one "$agent" deny "$p" || fails=$((fails+1)); done
  for p in "${ASK[@]}"; do apply_one "$agent" ask "$p" || fails=$((fails+1)); done
  if (( fails > 0 )); then
    echo "    -> $fails failures on $label" >&2
    return 1
  fi
  echo "    -> ${#DENY[@]} deny + ${#ASK[@]} ask applied"
}

total_fails=0
apply_agent "resource-claude-code" "resource-claude-code" DENY_PATTERNS_CLAUDE ASK_PATTERNS_CLAUDE || total_fails=$((total_fails+1))
apply_agent "resource-opencode"    "resource-opencode"    DENY_PATTERNS_OPENCODE ASK_PATTERNS_OPENCODE || total_fails=$((total_fails+1))
apply_agent "resource-codex"       "resource-codex"       DENY_PATTERNS_CODEX ASK_PATTERNS_CODEX || total_fails=$((total_fails+1))

echo
if (( total_fails > 0 )); then
  echo "FAILED: $total_fails agent(s) had errors" >&2
  exit 1
fi
echo "OK: policy applied to all installed coding-agent resources"
