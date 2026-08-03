# Storage writer reachability exceptions

The fleet declaration check requires an attributable Vrooli-owned writer for
storage that a Vrooli component creates or routes. A small set of declarations
describe paths whose writers are external runtimes or host facilities. They
remain declared so the census, placement, and retention reports do not lose
those bytes; they are not treated as Vrooli-owned writer paths.

These are the complete `STORAGE_ENTRY_NO_WRITER` findings from the current
fleet validation baseline. The list is intentionally explicit: adding a new
exception requires identifying the external writer and updating this table.

| Owner | Entry | Declared path | Why no Vrooli-owned writer exists |
| --- | --- | --- | --- |
| antigravity | home | `$USER_HOME/.gemini` | Google Antigravity owns its Gemini client home; Vrooli only provisions the resource. |
| claude-code | home | `$USER_HOME/.claude` | Claude Code owns its user home; Vrooli does not write the client’s state. |
| codex | home | `$USER_HOME/.codex` | Codex owns its user home; Vrooli does not write the client’s state. |
| grok | home | `$USER_HOME/.grok` | The external Grok client owns this home; Vrooli only declares its placement. |
| k6 | content | `$USER_HOME/.k6` | The k6 runtime and user scripts own this directory; no Vrooli writer routes it. |
| kopia | config | `$USER_HOME/.config/vrooli/resources/kopia` | Kopia owns its configuration files through its external runtime. |
| kopia | state | `$USER_HOME/.local/state/vrooli/resources/kopia` | Kopia owns its runtime state through its external runtime. |
| opencode | config | `$USER_HOME/.config/opencode` | OpenCode owns its client configuration. |
| opencode | storage | `$USER_HOME/.local/share/opencode/storage` | OpenCode owns its client storage and cache. |
| postgres | config | `$USER_CONFIG_DIR/vrooli/resources/postgres` | The PostgreSQL resource driver and server own this configuration path. |
| twilio | config | `$USER_CONFIG_DIR/vrooli/resources/twilio` | The Twilio resource integration owns its external configuration. |
| kdump-tools | crash_dumps | `/var/crash` | The host kernel crash-dump facility writes crash artifacts here. |
| mcelog | error_log | `/var/log/mcelog` | The host mcelog daemon writes machine-check error logs here. |
| rasdaemon | event_database | `/var/lib/rasdaemon/ras-mc_event.db` | The host rasdaemon service writes its event database here. |
| vault | token_cache | `$USER_HOME/.vault-token` | The Vault CLI owns this user token file; Vrooli must not write credentials. |

The exceptions are findings, not suppressions: they remain visible in fleet
output and continue to participate in census accounting and budget review.
