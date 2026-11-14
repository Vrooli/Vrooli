# CLI Domains

Start new command work here, not in `app.go`.
This domain-package layout is the default greenfield architecture for scenario CLIs.

Guidance:

- Use one package per domain.
- Register each domain from `domains/domains.go`.
- Prefer `SubcommandGroup` for command-rich domains; flat `CommandGroup`
  entries are best when a domain is a single verb.
- Keep API calling thin: argument parsing, request building, response formatting.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and
  `RenderMutationReport` for human-first output by default.
- If a command has a `--json` mode, emit the same report structure with
  `cliapp.PrintReportJSON(...)`.
