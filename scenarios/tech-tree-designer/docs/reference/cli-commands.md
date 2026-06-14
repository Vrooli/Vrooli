# CLI Commands

## Global flags (provided by cli-core)

The scenario CLI uses cli-core's standard application shell and installed command name `tech-tree-designer`.

## Built-in commands (auto-provided by `cli-core`)

```bash
tech-tree-designer status
```

## Scenario commands

Implemented graph commands:

```bash
tech-tree-designer graph describe [--scenarios a,b] [--stability stable]
tech-tree-designer graph neighbors <scenario> [--depth 2] [--scenarios a,b]
tech-tree-designer graph path <from> <to> [--scenarios a,b]
tech-tree-designer graph ancestors <scenario> [--scenarios a,b]
tech-tree-designer graph export [--format text|dot|json] [--scenarios a,b] [--stability stable]
```

Planned groups:

| Group | Purpose |
|---|---|
| plan | planned scenario proto CRUD, validate, materialize |
| roadmap | sectors, tiers, milestones, progress |

## Output contracts

Commands should use cli-core report-shaped output and JSON mode where supported.

## Cross-references

- [`../../cli/manifest.json`](../../cli/manifest.json)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
