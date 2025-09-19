# OpenCode AI CLI Resource

This resource packages the terminal-first [OpenCode](https://opencode.ai) workflow so agents can drive AI-assisted coding without launching VS Code. It wraps a lightweight Python entry point that talks to Ollama, OpenRouter, or Cloudflare AI Gateway and stores configuration under `data/opencode/`.

## Quick start
```bash
# Install the CLI and create a default (Ollama) config
resource-opencode manage install

# Inspect status / configured providers
resource-opencode status

# Send a chat completion
resource-opencode run chat --prompt "Summarise the repo" --provider openrouter --model openrouter/gpt-4o-mini
```

## Configuration & Secrets
- Configurations live at `data/opencode/config-*.json`. Activate a saved config with `resource-opencode content execute <name>`.
- The active config (`config.json`) can include optional `openrouter_api_key`, `cloudflare_api_token`, and `cloudflare_gateway_url`. These are pulled automatically from Vault / `~/.vrooli/secrets.json` when present.
- Secrets are declared in `config/secrets.yaml` so the Secrets Manager scenario can guide provisioning. Keys from the `openrouter` or `cloudflare-ai-gateway` resources can be reused.

## Models
`resource-opencode run models list --json` queries:
- Local Ollama models (`ollama list`)
- OpenRouter remote models (needs `OPENROUTER_API_KEY`)
- Cloudflare AI Gateway models (`cloudflare_gateway_url` + `CLOUDFLARE_API_TOKEN`)
- Custom references stored in `available-models.json`

## Programmatic usage
Every resource command ultimately shells out to `lib/opencode_cli.py`, which exposes:
- `info` – machine-readable config summary
- `models list` – structured model inventory
- `chat` – single-turn chat completion across providers

You can call it directly with `resource-opencode run …` or import the Python script into your own automation.

## Logs
Outputs are tracked in `data/opencode/logs/opencode.log` (JSONL). Each entry records timestamp, provider, and approximate response length for traceability.
