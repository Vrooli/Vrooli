# CLI Domains

Start new command work here, not in `app.go`.
This domain-package layout is the default greenfield architecture for scenario CLIs.
Do not plan to grow a new scenario around flat `cmd_<domain>.go` files.

Recommended layout as the CLI grows:

```text
cli/
├── app.go
├── domains/
│   ├── domains.go
│   ├── tasks/
│   │   ├── register.go
│   │   ├── list.go
│   │   ├── get.go
│   │   ├── create.go
│   │   ├── update.go
│   │   ├── delete.go
│   │   ├── output.go
│   │   └── types.go
│   └── projects/
│       ├── register.go
│       └── ...
└── internal/
    └── output/
```

Guidance:

- Use one package per domain.
- Register each domain from `domains/domains.go`.
- Prefer `SubcommandGroup` for command-rich domains.
- Keep API calling thin: argument schema, request building, response formatting.
- Declare flags and positionals with `cliapp.ArgSchema`; implement handlers with `RunCtx`.
- Use generated Connect clients for proto-typed operations.
- Use `cliapp.RenderProtoList` and `RenderProtoMutation` when a command returns one proto response. Use the `RunContext` report helpers directly for aggregate/non-proto output.
