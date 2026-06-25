# Swarm Manager Perf Audits

Index of headless performance audits run against the swarm-manager UI. Each
audit is a `<YYYY-MM-DD>-<slug>.md` file with required YAML frontmatter
(validated by `knowledge-observatory docs audit swarm-manager`) and a
per-component aggregation table copied from the
`scenario-performance-audit` skill's Phase 5 analyser.

| Date | Slug | Status | Subject |
|---|---|---|---|
| 2026-05-03 | [sidebar-resize-and-backlog-scroll](2026-05-03-sidebar-resize-and-backlog-scroll.md) | fixed | Sidebar drag lag and BacklogTab scroll cost |

## Process

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
