# CLI Domains

Start new command work here, not in `app.go`.

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
- Keep API calling thin: argument parsing, request building, response formatting.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and `RenderMutationReport` for human-first output by default.
