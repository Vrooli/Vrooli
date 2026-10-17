# Web Console review sources

Current input, acknowledgement, reconnect, and presence semantics live in
[Terminal Input Protocol](../TERMINAL-INPUT-PROTOCOL.md). Implementation boundaries
belong in [Seams](../SEAMS.md), and unresolved work in [Problems](../PROBLEMS.md).
In particular, input offsets are connection-scoped and presence is independent
of grid dimensions; use the protocol contract when changing these behaviors.

The six historical reviews are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/web-console/docs/internal/reviews/`.

| Preserved source | Historical subject |
|---|---|
| `01-terminal-signal-path.html` | Terminal signal path |
| `02-scroll-zoom-redraw.html` | Scrolling, zoom, and redraw |
| `03-runs-anywhere-proven-nowhere.html` | Portability declarations and qualification |
| `04-four-decisions-three-designs.html` | Design alternatives and decisions |
| `05-terminal-io-teardown.html` | Teardown and terminal I/O reliability |
| `06-terminal-second-pass.html` | Transport, storage, portability, and recovery |

Read their owner plans with:

```text
plan-manager plans get web-console-terminal-reliability-and-portable-seams-one
plan-manager plans get web-console-terminal-predictive-input-scroll-fidelity-and
```

Review findings and measurements describe their source revisions. Plan decisions
and current protocol docs own accepted behavior; the archive preserves rejected
alternatives and evidence without turning a past observation into current status.
