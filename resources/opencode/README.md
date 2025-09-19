# OpenCode AI CLI Resource

This resource installs the official [OpenCode](https://opencode.ai) CLI so agents can drive the terminal-first coding workflow inside Vrooli. The binary is downloaded from the upstream GitHub releases and isolated under `data/opencode/` alongside configuration, cache, and auth state.

## Quick start
```bash
# Download the latest OpenCode binary and scaffold a config
resource-opencode manage install

# Inspect installation details and config
resource-opencode status

# Run a one-off command (non-interactive)
resource-opencode run run "Summarise the repo"

# Explore the interactive TUI
resource-opencode run
```

## Configuration & Secrets
- The active config lives at `data/opencode/config/opencode.json`. Alternate profiles are stored as `config-*.json` in the same directory and can be activated via `resource-opencode content execute <name>`.
- Environment secrets such as `OPENROUTER_API_KEY` or `CLOUDFLARE_API_TOKEN` are loaded automatically from Vault / `~/.vrooli/secrets.json` and exposed to the CLI. They can also be written into the CLI's auth store with `resource-opencode run auth login`.
- Secrets remain declared in `config/secrets.yaml`, allowing the Secrets Manager scenario to prompt for and provision credentials.

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
Standard output is streamed directly to the caller. The resource also sets `OPENCODE_LOG_DIR` to `data/opencode/logs/` so you can enable structured logging via the CLI's `--print-logs` or config options if desired.
