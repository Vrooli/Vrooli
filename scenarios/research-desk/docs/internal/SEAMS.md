# Seams

Boundaries production wires once and tests substitute. Wire shapes live in proto, not here.
Exact-value tests expand `api/internal/testutil/home.txtar` (small excerpts of real
research-home layouts, home path replaced by a placeholder) into a temp folder with
`testutil.NewHome`; `make real-home-check` walks the real home read-only and asserts
invariants (one evidence run in `docs/internal/evidence/real-home-check.txt`).

| Seam | Production | Test double | Used by |
|---|---|---|---|
| `home.Config` (`ResearchDir`, `CodexDir`, `OtherCodexHomes`) | `~/.vrooli/research`, Web Console's Codex sessions root; `RESEARCH_DESK_*` overrides | `testutil.NewHome` temp folders | all domains |
| `Clock` | wall clock | fixed clock | state rules, courier, looks |
| `Procs` (process table) | `/proc` scan for `codex` with `CODEX_HOME` | static list | commission state, stop |
| `Runner` (subprocess) | `python3 -I packet.py …` with timeout | the real script over the archive's missions; a runner printing a crash, no JSON or an old schema | launch, the read-set ceiling test |
| `launch.Terminal` | Web Console Connect `SessionsService.Create` and `TerminalService` `GetScreen`, `SendInput`, `WaitIdle` (the desk's service credential when present, otherwise Web Console's local availability policy; the access used is shown; `internal/peer`) | a fake Codex that draws its input row, replies "ready", writes a goal row and keeps an app-server | launch, control, launch test |
| `launch.Sessions` | the home reader: the session's `goals_1.sqlite` and its Codex processes by `CODEX_HOME` | the same fake Codex | launch verification, stop |
| `courier.Hub` | Vrooli Events (decision and notification events) to send; Notification Hub `ListAsks` and notification `List` to read, no credential | in-memory asks and notices | courier |
| `writer.Writer` | `flock` on `~/.vrooli/state/vrooli/research-desk/instance.lock`; appends and atomic JSON | the real writer over a temp state folder; a second instance for read-only | operator, courier, looks, launch |
| `capabilities.Registry` | projected from `.vrooli/service.json`; `/health` per scenario via discovery; `capabilities.Reach` gives launch and the courier a base URL only when it says available | fixed port resolver over `httptest` | `/api/v1/capabilities/describe`, the overview, launch, courier |

Not seams: the state rules, needs-you and operator-file parsers, and gates. Tests call
them directly. Launch is tested as a table over its linear steps through `Terminal`; there
is no flow model.

## Kept from the template

The Connect router, `cliapp` manifest bindings, `module.Module`, endpoint codegen, the
`testutil` import guards, and the UI transport, client modules, ErrorBoundary and i18n
singleton stay as the template defines them. The database, storage, `Doer`, `httpx` and
example domain are gone.
