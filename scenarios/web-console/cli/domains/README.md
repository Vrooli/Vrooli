# CLI Domains

Start new command work here, not in `app.go`.
This domain-package layout is the default greenfield architecture for scenario CLIs.

Guidance:

- Use one package per domain.
- Register each domain from `domains/domains.go`:
  - `CommandGroup` (registered via `CommandGroups`) for single-verb domains so the invocation is `web-console <domain>` (e.g. `events`, `metrics`, `capabilities`).
  - `SubcommandGroup` (registered via `SubcommandGroups`) for domains with multiple verbs so the invocation is `web-console <domain> <verb>` (e.g. `session get`, `workspace pane-update`).
  - A domain starts as `CommandGroup` and is promoted to `SubcommandGroup` the moment it grows a second verb — don't fake a `list` subcommand to justify the hierarchy.
- Keep API calling thin: argument parsing, request building, response formatting.
- Use `cliapp.RenderOperationalReport`, `RenderListReport`, and `RenderMutationReport` for human-first output by default.
- If a command has a `--json` mode, emit the same report structure with `cliapp.PrintReportJSON(...)`.
