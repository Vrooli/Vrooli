# Swarm Manager Perf Audits

Index of headless performance audits run against the swarm-manager UI. Each
audit is a `<YYYY-MM-DD>-<slug>.md` file with required YAML frontmatter
(validated by `knowledge-observatory docs audit swarm-manager`) and a
per-component aggregation table copied from performance-health analysis.

| Date | Slug | Status | Subject |
|---|---|---|---|
| 2026-07-06 | [graph-large-topology-baseline](2026-07-06-graph-large-topology-baseline.md) | measured | Graph large-topology baseline and grouped-layout candidate |
| 2026-05-03 | [sidebar-resize-and-backlog-scroll](2026-05-03-sidebar-resize-and-backlog-scroll.md) | fixed | Sidebar drag lag and BacklogTab scroll cost |

## Process

Graph audits use separate workload names so load/React commit evidence cannot
be mistaken for interaction usability:

- `graph-load` is load-only and may use `load_only` budgets.
- `graph-sustained-pan` must produce a marked `graph-sustained-pan` gesture
  window and enough `EventDispatch` evidence to gate panning.
- `graph-wheel-zoom` must produce marked wheel zoom windows for zoom in/out.
- `graph-pinch-zoom` is the current pinch-style fallback; BAS implements it
  with driver-level wheel input, so it is not true multi-touch parity evidence.

Run targeted captures through performance-health:

```bash
performance-health audit run swarm-manager --workflow graph-sustained-pan --json
performance-health analysis analyze swarm-manager --trace <trace-path> --json
performance-health budget check swarm-manager --flow graph-sustained-pan
```

To add a new audit:

```bash
SLUG=<short-kebab-slug>
DEST="docs/perf/$(date -I)-${SLUG}.md"
knowledge-observatory docs template perf-audit > "${DEST}"
${EDITOR:-vi} "${DEST}"
knowledge-observatory docs audit swarm-manager   # confirms frontmatter + table shape
```

Then add a row to the table above and register the file in
`docs/manifest.json` under the `perf` section.

## What lives here vs `/tmp/`

The persisted markdown lives in this folder. Raw trace JSON files (40+MB)
stay in `/tmp/swarm-manager/perf/` and are referenced by absolute path
from each audit doc. Traces may be GC'd; the markdown is the authoritative
record.
