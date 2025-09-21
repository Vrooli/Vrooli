# OpenCode AI CLI Resource

This resource installs the official [OpenCode](https://opencode.ai) CLI so agents can drive the terminal-first coding workflow inside Vrooli. The binary is downloaded from the upstream GitHub releases and isolated under `data/opencode/` alongside configuration, cache, and auth state.

## Quick start
```bash
# Download the latest OpenCode binary and scaffold a config
resource-opencode manage install

# Inspect installation details and config
resource-opencode status

# Start the OpenCode HTTP server so agents can make edits
resource-opencode manage start

# Send a prompt through the agent-aware workflow (creates a server session automatically)
resource-opencode agents run --prompt "Fix the failing test" --model openrouter/qwen3-coder

# Inspect active sessions or running agents
resource-opencode agents session list
resource-opencode agents list --json

# Apply safety rails and execution limits (mirrors resource-claude-code/codex flags)
resource-opencode agents run --prompt "Refactor the handler" --allowed-tools "edit,write" --max-turns 8
resource-opencode agents run --prompt "Generate release notes" --skip-permissions --task-timeout 180

# Run a one-off command (non-interactive)
resource-opencode run run "Summarise the repo"

# Explore the interactive TUI
resource-opencode run
```

## Configuration & Secrets
- The active config lives at `data/opencode/config/opencode.json`. Alternate profiles are stored as `config-*.json` in the same directory and can be activated via `resource-opencode content execute <name>`.
- Environment secrets such as `OPENROUTER_API_KEY` or `CLOUDFLARE_API_TOKEN` are loaded automatically from Vault / `~/.vrooli/secrets.json` and exposed to the CLI. They can also be written into the CLI's auth store with `resource-opencode run auth login`.
- Secrets remain declared in `config/secrets.yaml`, allowing the Secrets Manager scenario to prompt for and provision credentials.
- Default model selection now targets OpenRouter's `qwen3-coder`; set `OPENROUTER_API_KEY` (or edit `opencode.json`) before running agents.
- Permission and safety overrides mirror other automation agents:
  - `--allowed-tools` (or `OPENCODE_ALLOWED_TOOLS` / `ALLOWED_TOOLS`) to constrain tool usage.
  - `--skip-permissions` (or `OPENCODE_SKIP_PERMISSIONS` / `SKIP_PERMISSIONS`) to auto-approve every request.
  - `--max-turns` to abort after N completed tool calls (`OPENCODE_MAX_TURNS`).
  - `--task-timeout` / `--timeout` to stop a run after N seconds (`OPENCODE_TASK_TIMEOUT`, falls back to `TIMEOUT`).

## AGENTS.md & custom instructions
- The official CLI recursively scans for `AGENTS.md` from the working directory upward (and respects any paths listed in `opencode.json`).
- The resource's default config pins `OPENCODE_CONFIG` to `data/opencode/config/opencode.json` so you can manage instructions without touching global user state.

## Models
Use the upstream CLI to enumerate available models once you've configured credentials:

```bash
resource-opencode run models
```

Model discovery pulls from local Ollama installs as well as any provider keys configured via `opencode auth login` or environment variables.

## Programmatic usage
`resource-opencode run …` is a thin wrapper around the official `opencode` binary. Pass arguments exactly as you would to the upstream CLI:

```bash
# Non-interactive run with explicit model override
resource-opencode run run --model openrouter/gpt-4o-mini "Generate release notes for the last commit"

# Manage credentials
resource-opencode run auth login
```

## Logs
Standard output is streamed directly to the caller. The resource also sets `OPENCODE_LOG_DIR` to `data/opencode/logs/` so you can enable structured logging via the CLI's `--print-logs` or config options if desired. When the HTTP server is running its detailed logs are stored at `data/opencode/logs/server.log` and can be inspected with `resource-opencode logs`.
