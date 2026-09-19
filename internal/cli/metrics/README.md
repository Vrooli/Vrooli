# internal/cli/metrics

Passive timing telemetry for the `vrooli` CLI.

A `Recorder` appends one `Event` per invocation to
`$HOME/.vrooli/metrics/timings.jsonl`. The recorder is wired into the single
dispatch site in `internal/cli/rootcli/rootcli.go` so every top-level command
is measured.

The active JSONL file is bounded by a 64 MiB size limit and a 30-day age
limit. Up to three rotated files are retained; these defaults are injectable
through `NewWithPolicy` for focused tests.

See the user-facing README written to `~/.vrooli/metrics/README.md` on first
record for the schema and opt-out instructions (source: `recorder.go`).
