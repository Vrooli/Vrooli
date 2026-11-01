# system-host-pressure

Reports host CPU pressure, memory and swap state, process count, and fork rate
through the portable `internal/hostpressure` reader. Every field carries a
read/unread state and provenance. Unread means the platform could not answer;
it is never treated as zero.

The check is diagnostic and joins into substrate SB14–SB17. Fork rate is a
Linux signal by decision and is reported unread on macOS and Windows.


## Actions and automatic recovery

`report-findings` is a manual diagnostic. It returns current portable readings,
bounded parent attribution and up to 20 swap-heavy process holders in its action
output; the action API retains that output in the action log. Its receipt marks
`diagnostic_only=true` and `host_mutated=false`. A high swap/RSS ratio does not
establish that a process is idle, abandoned or disposable.

Diagnostic success does not repair pressure. Host-pressure diagnostics and resource
recycling are marked `manualOnly` and are excluded from every scheduled autoheal
selection path. Recycling retains the control-plane managed-resource, serving-state
and saturation checks. The unimplemented disposal-preview action is unavailable;
use `vrooli agent reap --orphaned --dry-run --json` for the owner-checked preview.

Safe periodic orphan reaping already belongs to the runtime supervisor. Its
operator-configured `AgentOrphanReapInterval` delegates to
`internal/agentsession.ReapOrphaned`, preserving the one-hour minimum age, owner
recheck, frozen-scope protections and audit. It is off by default; a separate
scenario-private disposal loop would duplicate ownership. Registered observational
checks such as `system-swap` return an empty actions list rather than a missing-check
404. Their raw autoheal preference does not create an available recovery action.
