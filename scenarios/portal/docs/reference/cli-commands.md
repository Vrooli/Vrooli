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

### Context briefs

| Command | RPC or local operation | Purpose |
|---|---|---|
| `portal brief build <prompt> [--consumer portal-llm\|portal-agent\|external-harness] [--render agent\|system] [--json]` | `BriefService.Build` | Build, gate, render, and persist one current-turn brief |
| `portal brief get <id>` | `BriefService.Get` | Inspect a persisted verdict and its delivered items |
| `portal brief list [--consumer <consumer>] [--chat-id <id>] [--session-ref <ref>]` | `BriefService.List` | List retained brief records |
| `portal brief record-use <brief-id> <item-index> <kind>` | `BriefService.RecordUse` | Record an inspector use event |
| `portal brief stats [--window <days>] [--consumer <consumer>]` | `BriefService.Stats` | Report built/delivered/withheld briefs and item usage rates |
| `portal brief hook --runtime claude-code` | local hook adapter | Read one hook request and emit context only when the external lane is verified |
| `portal brief hooks status --runtime <runtime>` | local capability read | Show declared and canary-verified hook capability |
| `portal brief hooks install --runtime <runtime>` | local capability plus broker | Refuse unverified capability; otherwise reconcile the owned hook |
| `portal brief hooks remove --runtime <runtime>` | local broker operation | Remove only the Portal-owned hook |

### Legacy Assistant migration

These local commands inspect the old `vrooli-assistant` corpus without starting
its runtime or deleting source data. Source records must be regular files that
are not group/world-writable; `export` writes a separate mode-0600 review copy
and `reconcile` verifies that the source still matches its checksum manifest.

```bash
portal assistant-migration inventory --source /path/to/vrooli-assistant --json
portal assistant-migration export --source /path/to/vrooli-assistant --destination /private/assistant-review --json
portal assistant-migration reconcile --source /path/to/vrooli-assistant --manifest /private/assistant-review/manifest.json --json
portal assistant-migration review --source /path/to/vrooli-assistant --manifest /private/assistant-review/manifest.json --json
portal assistant-migration capture --source /path/to/vrooli-assistant --manifest /private/assistant-review/manifest.json --destination /private/portal-owner-handoffs --json
```

The manifest contains stable record IDs, relative paths, sizes, and SHA-256
digests only. Reconciliation requires the manifest's original source root and
rejects changed content. `review` parses the bounded task format, reconciles
task/context links, assigns a current owner (manual reports go to
`scenario-qa`), and emits stable capture keys for idempotent routing. It is a
metadata-only handoff: source records are retained until an operator reviews
the output and the owner confirms capture. `capture` verifies and copies linked
context records into a private owner evidence bundle, writes one atomic owner
handoff per stable key, and returns the same receipt on a retry; it refuses
conflicting reuse and never deletes the legacy source. External screenshot
paths remain references and are not read implicitly.

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

## Surface catalog

`portal surfaces list` returns owner-scoped references and source inventory
status. Failed providers leave healthy sources visible. The production adapter
currently consumes Web Console terminal inventory.

`portal surfaces resolve "Office PC"` returns every matching label; multiple
matches remain ambiguous. Exact selection uses all fields from a returned ref:

```bash
portal surfaces resolve --target-owner web-console --target-id local --surface-owner web-console --surface-id local --json
```

Include `--host-node-id` for references that carry a host relation. Discovery
never grants access; destination owners still admit sessions. The local terminal
is labeled "Web Console host", independently of the companion's machine.


## Retained context images

Use an operator access token in a private regular file (mode 0600):

```sh
portal context import --access-token-file /private/token --request /private/context.json --json
portal context cancel-import --access-token-file /private/token --request-id <request-id> --json
portal context reconcile-import --access-token-file /private/token --request-id <request-id> --json
portal context render --access-token-file /private/token --id <context-id> --output /private/new-crop.png --json
portal context read --access-token-file /private/token --id <context-id> --output /private/new-image.png --json
portal context delete --access-token-file /private/token --id <context-id> --json
```

The import file is a private protobuf JSON `ImportRequest` for
`ContextCaptureService`, including source provenance, region, strokes, base64 PNG,
explicit `retentionSeconds` (at most 86400), and a caller-generated UUID
`requestId`. Reuse that ID only for an identical import. Unknown fields are rejected.
The source capture must be recent; imported source claims are not desktop input
authority. Token and import files must be regular, private files, not symlinks.

Read saves the original image to a new mode-0600 file after verifying its ID,
expiry, byte bound, and digest. Existing output files are never replaced.
JSON output contains document metadata, not image bytes. The exported file is
owned by the caller; Portal deletion/expiry removes Portal's retained copy and
does not delete an exported file. Commands do not automatically retry imports. When `requestId` was supplied,
resubmitting the same request resolves a successful import to its original ID
and expiry without rewriting pixels. Changed content is rejected. Request
receipts remain for 24 hours after initial admission, including after deletion,
to prevent a delayed retry from recreating removed data. An expired or removed
artifact stays unavailable. Omitting `requestId` creates a new intent and does
not provide safe retry after a lost response.

`reconcile-import` recovers publication status using only the request UUID and the
same authenticated account. It returns `ABSENT`, `STAGING`, `READY`, or
`UNAVAILABLE`; only `READY` includes document metadata. It does not read image
bytes, replay the import, or extend retention. Use `read` to verify retained
bytes before use. `ABSENT` means no receipt exists in that account scope, not
proof that an import never happened: receipts have a bounded 24-hour lifetime.
A pending or unavailable import must not be treated as a usable attachment.

`cancel-import` cancels by request UUID even when its import reply was lost.
It removes an existing image under publication exclusion, or records a negative
receipt if the request has not arrived. Repeated cancellation does not extend
the receipt lifetime. Receipts are bounded to 24 hours. A failed cleanup is
reported as failure and remains tracked; retry cancellation to finish cleanup.
The UI preserves its pending marker until cancellation is acknowledged.

`render` exports only the selected region with red annotations clipped to that
region. Annotation width is two source-image pixels. It preserves the retained
original and returns separate `renderedSha256` metadata; `originalSha256` still
identifies the full original. Rendering checks ownership, integrity, deletion
and expiry, bounds raster work, and creates a new private output file.
