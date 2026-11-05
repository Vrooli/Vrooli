# CLI Domains

Start new command work here, not in `app.go`.
This domain-package layout is the default greenfield architecture for scenario CLIs.

Guidance:

- Use one package per domain.
- Register each domain from `domains/domains.go`.
- Prefer `SubcommandGroup` for command-rich domains.
- Keep API calling thin: argument parsing, request building, response formatting.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and `RenderMutationReport` for human-first output by default.
- If a command has a `--json` mode, emit the same report structure with `cliapp.PrintReportJSON(...)`.

## Layout

```text
cli/
├── app.go
├── domains/
│   ├── domains.go
│   ├── health/      # flat `file-tools health` (root /health)
│   ├── docs/        # flat `file-tools docs` (root /docs)
│   ├── archive/     # file-tools archive compress|extract|split|merge
│   ├── files/       # file-tools files metadata|metadata-extract|checksum|operation|search|organize|duplicates
│   └── analyze/     # file-tools analyze relationships|storage|access|integrity
└── internal/
    └── support/
```
