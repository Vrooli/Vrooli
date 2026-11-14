# Suppression Markers

A **suppression marker** is a durable, in-repo source comment that sanctions a
specific architecture finding as intentional. Markers are version-controlled
and discoverable right next to the code they excuse — the deliberate
alternative to a database-only suppression that is invisible to code readers
and easily orphaned.

Markers are modeled on the `documentation-health` `// DOC:` / `// seam:`
conventions: a known token inside an ordinary comment, scanned out of the
source tree.

## Grammar

```
<comment-prefix> arch:allow <id> reason="<why>" [expires="<condition>"]
```

- **`<comment-prefix>`** — any comment style: `//` (Go/TS/JS), `#` (Python,
  shell), or a `/* … */` block. The scanner matches the `arch:allow` token
  anywhere in the line.
- **`<id>`** (required) — what to sanction. Matches a **detector name**
  (`cycle`, `mislocated_file`, `convergence_drift`, `coupling_smell`), a
  **conflict type**, or a **finding subtype** (`god_domain`,
  `unstable_dependency`, `missing_implementation`, …).
- **`reason="…"`** (required) — the human rationale. A marker with no reason
  is malformed and is surfaced as a diagnostic, never honored.
- **`expires="…"`** (optional) — an expiry condition. The only
  machine-evaluated form is `until:YYYY-MM-DD` (the marker stops suppressing
  after that date). Any other text (e.g. `until:#1234-merges`) is advisory and
  the marker stays active.

### Examples

```go
// arch:allow cycle reason="legacy alpha↔beta hub, tracked in #412" expires="until:2026-12-31"
```

```python
# arch:allow coupling_smell reason="orchestration root wires every domain by design"
```

## How a marker suppresses

A finding is reported as **suppressed-with-reason** (never silently dropped)
when an active, well-formed marker:

1. has an `<id>` equal to the finding's detector, type, or subtype, **and**
2. is **location-relevant** — the marker's file lies under one of the
   finding's locations, or the file's owning domain (resolved via the derived
   domain map) is one of the finding's domains.

Suppressed findings still appear in `arch-cart conflicts detect` output
(flagged suppressed with the reason) and no longer block
`arch-cart conflicts validate` cartographer-clean closure.

## Scope of the substrate

- **Discovery** is the `SuppressionScanner` seam
  (`internal/suppressions`); production walks the scenario's source tree
  (skipping `node_modules`, `gen`, `vendor`, `dist`, `.git`, `.vrooli`).
- **Writing** a marker is the safe, non-destructive apply path:
  `arch-cart apply suppress <scenario> <file> <id> --reason "…"`. It inserts a
  comment only — it never moves or rewrites code (destructive execution stays
  deferred to `RunApply`). Writes are idempotent per `(file, id)`.
- **Ephemeral** migration/workflow state lives in cartographer's SQLite, not
  in markers. Markers hold only durable "this is intentional" decisions.

## Cross-references

- [`domains-contract.md`](domains-contract.md) — the derived domain map markers are matched against
- [`configuration.md`](configuration.md) — the global heuristic levers markers excuse
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — the `SuppressionScanner` / `Writer` seams
