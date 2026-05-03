# Tool & Safeguard Migration Candidates

Remaining bash reference files from the pre-Go-migration era. Everything
fully ported to Go manifests/handlers has been deleted. These files are
kept only as implementation reference for capabilities not yet ported.

## tools/

| File | Purpose | Notes |
|---|---|---|
| `ast-grep.sh` | Install `ast-grep` | Manifest exists (`internal/tools/ast-grep/tool.json`) but brew-only — no apt install path yet. CLAUDE.md references ast-grep as preferred dev tool |
| `lychee.sh` | Install `lychee` (link checker) | Manifest exists (`internal/tools/lychee/tool.json`) but brew-only — no apt install path yet. Relevant for doc-link validation |

## safeguards/

| File | Purpose | Notes |
|---|---|---|
| `common_deps.sh` | Install common OS packages (build-essential, pkg-config, etc.) | UNPORTED. Design decision pending: one safeguard vs many individual tools |
| `ssh_authorize_key.sh` | Add public key to `authorized_keys` | UNPORTED. Manual bootstrap — may belong as `vrooli bootstrap ssh-*` CLI command |
| `ssh_enable.sh` | Enable sshd, configure auth | UNPORTED. Manual bootstrap |
| `ssh_keyless.sh` | Passwordless SSH between nodes | UNPORTED. Manual bootstrap |
