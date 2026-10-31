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

The package is intentionally not scenario-adoptable: resource runtimes and
internal platform commands are its only declared consumers.
