# First-run handoff evidence

This evidence set belongs to the execution of `first-run-handoff-completion-truth-and-structural-cleanup`.
The immutable source baseline was captured by Git Control Tower as
`first-run-handoff-completion-truth-and-structural-cleanup-baseline`, run
`20260824-221341-9741ddda`, and synchronized before implementation began.

The current development session is an SSH remote shell on Linux. A local
graphical session and an isolated headless container were not available during
this run; those files say so explicitly. No claim of hardware evidence is
made for macOS or Windows.

Files:

- `session-remote-shell.txt`: literal session probes from the invoking SSH session.
- `session-local-graphical.txt`: unavailable session shape and reason.
- `session-headless.txt`: unavailable isolated-container capture and reason.
- `markers-before.txt`: resolved marker paths and state before the completion-authority change.
- `detect-remote-shell.json`: bounded presentation classifier output for this session.
- `configuration-complete-sample.json`: marker written by the projectstate test authority.
- `census-before.json`: structural values captured from the checked-in source.
- `setup-dry-run.txt` and `result-dry-run.json`: dry-run capture metadata.
- `setup-baseline.txt` and `result-baseline.json`: explicit note that a pre-change raw setup transcript was not captured; the immutable baseline run is the authoritative before-state artifact.
- `go-baseline.txt` and `scenario-baseline.json`: baseline references and current validation notes.
- `census-after.json`: current structural measurements after implementation.
- `known-red.md`: historical note retained for the resolved setup fixture mismatch.
