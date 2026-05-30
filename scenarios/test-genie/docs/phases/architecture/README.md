# Architecture Phase

The `architecture` phase audits a scenario's **structural cohesion** — does the
code's shape scream the product's capabilities, and do the domains hold together
without import cycles, runaway coupling, or misplaced files? It delegates to the
**architecture-cartographer** scenario's read-only `AuditService.Run` Connect-RPC
and normalizes the findings into the shared `ArchitectureFinding` contract that
every test-genie phase now emits.

This is the **cohesion axis** of the audit battery. The per-surface phases
(`contracts`, `ui-health`, `docs`, `standards`) ask "is each surface built
right?"; the `architecture` phase asks "does the whole scenario cohere?".

## How It Runs

Test Genie resolves the architecture-cartographer base URL via service discovery
and calls `AuditService.Run` for the target scenario. The response's findings
(detector type + subtype, severity, locations, domains) become Observations and
normalized `ArchitectureFinding`s (`source = ARCHITECTURE`).

Equivalent operator flow:

```bash
architecture-cartographer audit run <scenario>
```

## Advisory, Not Gating

The phase is **Optional** and **advisory**: it never hard-fails the suite on
findings. This preserves the cartographer's graded semantics — coupling and
convergence are signals, not pass/fail invariants. The phase only fails on:

- a transport error (cartographer unreachable → `missing_dependency`), or
- the cartographer's own `TOOL_ERROR` outcome (the audit could not run).

Blocker-severity findings (import cycles) still surface prominently in the
report and, together with the overall finding count, drive the **campaign
nudge**: when findings exceed the single-pass threshold, the suite output steers
you to open a tracked improvement campaign in architecture-cartographer rather
than fixing ad-hoc.

## Preset

The phase is included in the `architecture-audit` preset, the single command the
screaming-architecture audit skill points at:

```bash
vrooli scenario test <scenario> --preset architecture-audit
# runs: structure, contracts, ui-health, docs, standards, architecture
```

## Summary Metrics

The phase pointer records an `ArchitectureSummary`:

| Field | Meaning |
|---|---|
| `outcome` | cartographer audit outcome (`clean` / `findings` / `tool_error`) |
| `total` | total findings after filters |
| `blockers` / `errors` / `warnings` / `infos` | counts by normalized severity |
| `suppressed` | findings excused by in-repo `// arch:allow` markers (reported, not dropped) |
| `authority` | domain-derivation authority confidence (high/low) |

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip architecture
```

Disable for the process via environment:

```bash
TEST_GENIE_SKIP_ARCHITECTURE=1
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 120s):

```json
{
  "phases": {
    "architecture": { "timeout": "180s" }
  }
}
```
