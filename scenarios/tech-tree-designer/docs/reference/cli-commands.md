# CLI Commands

Commands below describe existing behavior. General proposal create/revise/review/apply/recover operations are targets, not command names available today. Discover the installed owner surface before invoking it; do not use proto materialize as a generic artifact apply shortcut. See [target flows](../concepts/FLOWS.md) and [owner boundaries](../concepts/INTEGRATIONS.md).

## Global flags (provided by cli-core)

The scenario CLI uses cli-core's standard application shell and installed command name `tech-tree-designer`.

## Built-in commands (auto-provided by `cli-core`)

```bash
tech-tree-designer status
```

## Scenario commands

The live CLI owns argument syntax. Read the selected group before choosing a
mutation; the examples below are read-only and do not publish a proposal.

```bash
tech-tree-designer graph help
tech-tree-designer graph neighbors tech-tree-designer --depth 1
```

| Group | Existing operations | Boundary |
| --- | --- | --- |
| graph | describe, neighbors, path, ancestors, export | Observed interface projection, not authored fulfillment |
| plan | create, list, tree, add, rm, validate, materialize | Existing proto-only planning; mutation requires authority |
| ontology | capabilities, capability, capability-upsert, capability-rm, edge-add, edge-rm, import, fulfill, unfulfill, fulfillments, coverage, focus, capability-scenarios, scenario, overlay | Authored ontology and mappings; no automatic verified fulfillment |

Inspect planned proto inventory:

```bash
tech-tree-designer plan help
tech-tree-designer plan list
```

`plan add` stores or replaces file text. `plan tree` prints a selected stored file
or lists the plan's tree. `plan materialize` writes canonical proto files and runs
generation. It is not a dry-run or the proposed general artifact apply operation.

Inspect authored capability coverage:

```bash
tech-tree-designer ontology help
tech-tree-designer ontology coverage
tech-tree-designer ontology focus --limit 10
```

## Output contracts

Commands should use cli-core report-shaped output and JSON mode where supported.

## Cross-references

- [`../../cli/manifest.json`](../../cli/manifest.json)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
