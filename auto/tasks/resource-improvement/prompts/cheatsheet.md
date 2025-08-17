# Resource CLI Cheatsheet

- List resources (JSON):
  - `vrooli resource list --format json`
- Show status (enabled resources):
  - `vrooli resource status`
- Specific resource status:
  - `vrooli resource <name> status`
  - `resource-<name> status`
- Start/Stop resource:
  - `vrooli resource start <name>`
  - `vrooli resource stop <name>`
- Install resource:
  - `vrooli resource install <name>`
- Ollama models:
  - `resource-ollama list-models`
  - `resource-ollama pull-model llama3.2:3b`

Notes
- Prefer `resource-<name>` when available for richer capabilities.
- Use `timeout 30` around commands that may hang.
- Redact secrets and avoid echoing config files directly. 