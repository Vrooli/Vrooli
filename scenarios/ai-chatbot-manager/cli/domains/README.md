# CLI Domains

Each file here wires one domain of the AI Chatbot Manager API into the
scenario CLI. Domains are thin wrappers: parse flags, call `core.Get` /
`core.Request`, render a `ListReport` / `MutationReport` / `OperationalReport`,
and support `--json` on the same report struct.

Layout:

```text
cli/
├── app.go
├── domains/
│   ├── domains.go
│   ├── chat/           # POST /api/v1/chat/{id} (single-shot; not interactive)
│   ├── analytics/      # GET  /api/v1/analytics/{id}
│   ├── chatbots/       # CRUD + widget embed for /api/v1/chatbots
│   ├── escalations/    # list/update via /api/v1/chatbots/{id}/escalations
│   ├── tenants/        # multi-tenant create/get
│   ├── abtests/        # A/B test create/start/results
│   └── crm/            # CRM integration create + lead sync
└── internal/support/
```

Guidance:

- Use one package per domain; register each from `domains/domains.go`.
- `SubcommandGroup` for multi-verb domains; `CommandGroup` for single-verb.
- Complex/variable JSON payloads accept `--body-file PATH` rather than
  hand-assembling nested JSON from flags.
- The built-in `status` command (from cli-core) covers root `/health`; do not
  re-implement it here.
