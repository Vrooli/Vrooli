# Contract Impact Guide

Use contract impact when a proto edit might affect another scenario. Validation
answers "is this surface honest today"; impact answers "what changes relative to
a baseline and who still imports the changed file."

## Scopes

Run the adaptive default first:

```bash
proto-health impact scenario <scenario>
```

The default asks git-control-tower for the newest baseline on the current branch.
If git-control-tower is unavailable or no baseline exists, proto-health falls
back to `merge-base` and labels the fallback in the report.

Use drill-down scopes when you need a specific comparison:

```bash
proto-health impact scenario <scenario> --against HEAD
proto-health impact scenario <scenario> --against merge-base
proto-health impact scenario <scenario> --against master
proto-health impact scenario <scenario> --against baseline:<name>
proto-health impact scenario <scenario> --against <sha>
```

## Reading The Report

Each change includes the proto file, Buf's compatibility message, wire and JSON
breaking flags, and the file's `@stability` tier. Renames are JSON-breaking even
when the wire number is unchanged.

The consumer list is v1 file-level blast radius. A consumer is unreconciled when
it still imports the changed proto file and the changed file is not
`@stability experimental`. This intentionally does not prove symbol-level use;
that v2 proof belongs with code-facts or code-graph providers.

## Promotion Gate

`git-control-tower baseline promote` calls this impact report before mutating
live. It blocks only when stable proto breaking changes still have unreconciled
consumers. Beta and experimental changes remain advisory, and `--force` can
bypass the gate when the operator records an intentional override.

## Cross-References

- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- Proto style guide: path:packages/proto/STYLE_GUIDE.md
