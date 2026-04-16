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

## Current state

The LLM Evaluator API exposes only `/health` today, which cli-core's built-in
`status` command already covers. No custom domains are registered yet. When
new endpoints land (evaluations, benchmarks, feedback, etc.), add a package
per domain under `cli/domains/<name>/` and wire it up in `domains.go`.

Shared helpers for new domains live in `cli/internal/support/`.
