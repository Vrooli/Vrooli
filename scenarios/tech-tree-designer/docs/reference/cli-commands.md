# CLI Commands

## Global flags (provided by cli-core)

The scenario CLI uses cli-core's standard application shell and installed command name `tech-tree-designer`.

## Built-in commands (auto-provided by `cli-core`)

```bash
tech-tree-designer status
```

## Scenario commands

Phase 1 has no product subcommands. Planned groups:

| Group | Purpose |
|---|---|
| graph | graph display/query/export |
| plan | planned scenario proto CRUD, validate, materialize |
| roadmap | sectors, tiers, milestones, progress |

## Output contracts

Commands should use cli-core report-shaped output and JSON mode where supported.

## Cross-references

- [`../../cli/manifest.json`](../../cli/manifest.json)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
