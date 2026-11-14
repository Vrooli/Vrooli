# CLI Commands — Research Desk

## Sources Of Truth

- Planned verbs: [`../concepts/SYSTEM.md`](../concepts/SYSTEM.md) §2.
- Registered commands and their bindings: [`cli/manifest.json`](../../cli/manifest.json);
  global flags and built-ins (`status`, `configure`) come from `cli-core`.
- `research-desk --help` on the installed CLI.

Every domain group is flat: commands dispatch at the top level (`research-desk now`). This
file does not restate them.

## Cross-References

- [`api-endpoints.md`](api-endpoints.md)
- Template mechanics for adding a command: `/scenarios/template-manager/docs/reference/cli-commands.md`
