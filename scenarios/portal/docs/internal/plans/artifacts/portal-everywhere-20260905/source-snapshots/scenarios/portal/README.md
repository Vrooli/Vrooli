# Portal

Portal is the chat-first front door to the Vrooli ecosystem. It gives
operators one workspace for OpenRouter LLM chat, coding-agent chat through
agent-manager, passive ecosystem search, and honest readiness status for
optional dependencies.

## What You Get

- A Go API, React/Vite UI, and Go CLI wired through generated
  `portal/v1` Connect contracts.
- Persistent local chat state in SQLite: chat groups, branchable message
  trees, active leaf tracking, search attachments, user settings, and usage
  records.
- LLM mode with streamed OpenRouter completions, model overrides, web-search
  toggles, and selected-skill system-prompt injection.
- Agent mode with normalized agent-manager activity and persisted final
  agent transcripts.
- A readiness registry for OpenRouter, search-hub, agent-manager, and
  prompt-manager. Portal boots when they are down and surfaces OFF/PASSIVE
  behavior with explicit reasons.
- Passive search: suggestions and send-time search attachments never gate the
  completion path.
- A manifest-driven CLI with chat, message, integration, and search commands.

## Running The Scenario

Use the Vrooli lifecycle. Do not run binaries directly.

```bash
make setup
make start
vrooli scenario status portal
```

The UI and API ports are allocated by `.vrooli/service.json`. Optional
dependencies are non-fatal. Set `OPENROUTER_API_KEY` only when you want live
OpenRouter completions; missing credentials produce typed integration errors
instead of boot failure.

Run the scenario suite through test-genie:

```bash
vrooli scenario test portal
```

To wait for a run, block once with:

```bash
test-genie runs wait --json portal 20260706-000000-example
```

## CLI Quick Reference

After `make setup`, the `portal` binary is installed through the scenario CLI
tooling. All scenario commands call generated Connect clients.

```bash
portal status
portal chats list
portal chats create --title "Scratch"
portal messages send <chat-id> "Summarize the current portal readiness"
portal messages stream <chat-id> --from <message-id>
portal integrations status
portal integrations override auto
portal search suggest "portal readiness"
```

Every command supports `--json` for proto JSON output where the API response is
proto-typed.

## Documentation Map

| Need | Start Here |
|---|---|
| First local run | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| System shape | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Data model | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| Integration posture | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| API contracts | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Configuration | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Testing | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Seams and fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Product requirements | [`PRD.md`](PRD.md) |

## Customize Safely

- Keep Portal greenfield. Do not import from agent-inbox or app-monitor, and do
  not add migration or compatibility layers for them.
- Keep Connect/proto as the default API surface. The health endpoint remains
  the lifecycle REST exception.
- Add dependencies only through Scenario Dependency Analyzer.
- Keep UI and CLI thin: business rules live in the API.
- Update `PRD.md`, `requirements/`, and the relevant docs when behavior
  changes.
- Preserve PASSIVE semantics: search may enrich the conversation, but it must
  not delay sends or completions.
