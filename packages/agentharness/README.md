# agentharness

`agentharness` is the resource-facing coding-agent policy surface. It owns
offline policy evaluation, permission-document projection, model discovery
commands, and coding-role commands. Catalog contracts consumed by the control
plane live in `cli-core/agentcatalog`.

It also owns filesystem-removal policy. `removal_lexer.go` and
`removal_parse.go` read command text (POSIX shells, PowerShell, cmd.exe) into
the paths it would delete, without executing it. `removal.go` decides those
paths against the floor and provider `path_rules`. See
`docs/architecture/agent-policy-runtime.md` §"Filesystem removal".

The runtime also enforces expiring per-check `log_only` modes and hard-stop
restoration. Their stable string-based store is `cli-core/guardmodes`, shared
by hooks and the root `vrooli guard-mode` command; the standalone policy runner
keeps list/audit read-only. The Steward may call
`GuardModeStore.SetLogOnlyFromAnswer(check, expiresAt, reason, actor, answerID)`
only after reading and validating the exact answered structural ask from
Notification Hub; the answer ID is persisted as authorization evidence. See
`docs/architecture/agent-policy-runtime.md` §"Agent guards" for the four
guards, their policy file (`hard_stops.json`), grants, log-only modes and the
guard event log the steward reads.

The package is intentionally not scenario-adoptable: resource runtimes and
internal platform commands are its only declared consumers.
