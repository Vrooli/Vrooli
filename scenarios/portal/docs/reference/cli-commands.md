# CLI Commands — Portal

Portal's CLI is a thin Go wrapper over generated Connect clients. The API owns
behavior; the CLI parses operator input, calls one RPC, and renders either a
human report or proto JSON with `--json`.

The command surface is declared in [`cli/manifest.json`](../../cli/manifest.json)
and loaded by `cliapp.LoadFromManifest`. `cli-health validate scenario portal`
checks the manifest schema, proto method coverage, runtime help surface, and
governance metadata.

## Global flags (provided by cli-core)

| Flag | Purpose |
|---|---|
| `--api-base <url>` | Override the API endpoint for this invocation |
| `--auto-start` | Start Portal through Vrooli lifecycle if the API is unreachable |
| `--json` | Emit machine-readable output |
| `--no-color` | Disable ANSI color |
| `--color` | Force-enable color |
| `--help`, `-h` | Show command help |
| `--version`, `-v` | Show version metadata |

## Built-in commands (auto-provided by `cli-core`)

### `portal status`

Calls the health probe and renders operational status.

```bash
portal status
portal status --json
```

### `portal configure <key> [value]`

Reads or writes CLI configuration such as `api_base` and `token`.

```bash
portal configure api_base http://localhost:17476
portal configure api_base
```

## Scenario commands — Portal

### Chats

| Command | RPC | Purpose |
|---|---|---|
| `portal chats list [--group-id <id>] [--query <q>]` | `ChatService.ListChats` | List chats and groups |
| `portal chats create --title <title> [--group-id <id>] [--model <model>] [--mode llm\|agent] [--harness claude-code\|codex\|opencode\|grok] [--web-search]` | `ChatService.CreateChat` | Create a chat |
| `portal chats show <id>` | `ChatService.GetChat` | Show one chat |
| `portal chats update <id> [--title <title>] [--group-id <id>] [--model <model>] [--active-leaf <message-id>] [--web-search\|--no-web-search]` | `ChatService.UpdateChat` | Update chat metadata |
| `portal chats delete <id>` | `ChatService.DeleteChat` | Delete a chat |
| `portal chats groups` | `ChatService.ListGroups` | List chat groups |
| `portal chats group-create --name <name> --color <color>` | `ChatService.CreateGroup` | Create a chat group |
| `portal chats group-update <id> [--name <name>] [--color <color>] [--collapsed\|--expanded] [--sort-order <n>]` | `ChatService.UpdateGroup` | Update a chat group |
| `portal chats group-delete <id>` | `ChatService.DeleteGroup` | Delete a chat group |

### Messages

| Command | RPC | Purpose |
|---|---|---|
| `portal messages tree <chat-id>` | `MessageService.GetTree` | Show a chat message tree |
| `portal messages send <chat-id> <text> [--parent <message-id>] [--model <model>] [--skill-ids <ids>] [--web-search]` | `MessageService.SendMessage` | Append a user message |
| `portal messages edit <message-id> <text>` | `MessageService.EditMessage` | Edit a message branch |
| `portal messages regenerate <message-id> [--model <model>]` | `MessageService.Regenerate` | Regenerate an assistant branch |
| `portal messages stream <chat-id> [--from <message-id>] [--model <model>] [--mode llm\|agent] [--harness claude-code\|codex\|opencode\|grok] [--skill-ids <ids>] [--web-search]` | `MessageService.StreamCompletion` | Stream completion events |

`messages stream --json` prints one proto JSON `CompletionEvent` per line.
Human output prints status lines, tokens, search attachment notices, agent
activity, errors, and done events.

### Integrations

| Command | RPC | Purpose |
|---|---|---|
| `portal integrations status` | `IntegrationsService.Status` | Show readiness registry state |
| `portal integrations override auto\|force-off\|force-passive` | `IntegrationsService.UpdateOverride` | Set behavior override |

### Search

| Command | RPC | Purpose |
|---|---|---|
| `portal search suggest <query> [--types <csv>] [--limit <n>] [--group <name>]` | `SearchService.Suggest` | Request bounded ecosystem suggestions |

## Output contracts

Proto-backed read commands use `cliapp.RenderProtoList`; mutation commands use
`cliapp.RenderProtoMutation`. Human output follows the fleet contracts:

| Contract | Used by | Structure |
|---|---|---|
| Operational | `status` and diagnostics | Status → Triage → Next Steps |
| Data retrieval | `list`, `show`, `tree`, `suggest` | Summary → Results → Retrieval Hints |
| Mutation | `create`, `update`, `delete`, `send`, `override` | Result → What Changed → Next Command |

## Adding a new command

1. Add or update the proto service method first.
2. Regenerate proto artifacts from `packages/proto/`.
3. Add the command to `cli/manifest.json` with binding and governance.
4. Implement the handler in `cli/domains/<domain>/handlers.go`.
5. Register the handler in `cli/domains/<domain>/register.go`.
6. Run `go test ./...` in `scenarios/portal/cli`.
7. Run `cli-health validate scenario portal --json`.

## Cross-references

- [`api-endpoints.md`](api-endpoints.md) — API endpoints these commands mirror
- [`configuration.md`](configuration.md) — env vars and config-file precedence
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — CLI/API role split
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — manifest-to-handler seam
