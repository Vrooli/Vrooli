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

Implemented planning commands:

```bash
tech-tree-designer plan create <slug> [--display-name "..."] [--sector engineering] [--tier foundation] [--stability experimental]
tech-tree-designer plan list [--sector engineering] [--tier foundation]
tech-tree-designer plan tree <slug> [path]
tech-tree-designer plan add <slug> <path> [--from-file -|<file>]
tech-tree-designer plan rm <slug> <path>
tech-tree-designer plan validate <slug> [--json]
tech-tree-designer plan materialize <slug>
```

`plan add` stores or replaces file text. `plan tree <slug> <path>` prints one stored file; without `path` it lists the stored tree.

Implemented ontology commands:

```bash
tech-tree-designer ontology capabilities [--parent <id>] [--kind sector|capability|component|capstone|simulation]
tech-tree-designer ontology capability <slug>
tech-tree-designer ontology capability-upsert <slug> [--name "..."] [--description "..."] [--kind capability] [--parent <id>] [--importance <number>]
tech-tree-designer ontology capability-rm <slug>
tech-tree-designer ontology edge-add <from> <to> [--type progression|decomposes|requires]
tech-tree-designer ontology edge-rm <from> <to> [--type progression|decomposes|requires]
tech-tree-designer ontology import --from-file data/seed/macro_topology.json
tech-tree-designer ontology fulfill <capability-id> <scenario-slug> [--note "..."]
tech-tree-designer ontology unfulfill <capability-id> <scenario-slug>
tech-tree-designer ontology fulfillments [--capability <id>] [--scenario <slug>]
tech-tree-designer ontology coverage [--subtree true|false]
tech-tree-designer ontology focus [--limit 10]
tech-tree-designer ontology capability-scenarios <slug> [--descendants true|false]
tech-tree-designer ontology scenario <slug>
tech-tree-designer ontology overlay [--implementation true|false] [--ontology true|false] [--fulfillment true|false]
```

## Output contracts

Commands should use cli-core report-shaped output and JSON mode where supported.

## Cross-references

- [`../../cli/manifest.json`](../../cli/manifest.json)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
