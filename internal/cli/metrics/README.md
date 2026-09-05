# internal/cli/metrics

Passive timing telemetry for the `vrooli` CLI.

A `Recorder` appends one `Event` per invocation to
`$HOME/.vrooli/metrics/timings.jsonl`. The recorder is wired into the single
dispatch site in `internal/cli/rootcli/rootcli.go` so every top-level command
is measured.

See the user-facing README written to `~/.vrooli/metrics/README.md` on first
record for the schema and opt-out instructions (source: `recorder.go`).
