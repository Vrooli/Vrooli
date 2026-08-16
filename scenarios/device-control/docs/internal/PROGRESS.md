# Progress — Device Control

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-16 | codex | partial | Investigated black bars in Android evidence. The raw stored PNGs and MP4 were affected before web-console display: producer redaction painted the entire top quarter whenever notification protection was enabled. Replaced the fixed quarter-screen mask with a bounded status-bar mask, added portrait image/native-video regression tests, and documented detector-supplied sensitive regions. Final managed physical screenshot flow `98fdca8f-13ce-4117-94f5-b95928e4edc2` passed after deployment and retained screenshot evidence `4f3ff631-e5a3-40bf-86bc-84daa1aa1caf` at 720x1600 with only the declared `status_bar_identifiers` rule; the remaining narrow black band is the intentional status-bar privacy mask, not a geometry bar. |
| 2026-08-15 | codex | partial | Hardened Android state restoration with bounded keyguard confirmation after asynchronous power transitions, and fixed the share verb's ADB argument ordering/quoting so multi-word SEND extras cannot become an accidental package token. Focused strategy/control tests pass. Physical flow `02a5358d-44c8-4031-baab-c8282e5321b2` passed on the Galaxy A03s, including active-profile unlock, Android resolver screenshot evidence `818edea8-760c-4a20-a649-0ffc99f4d9c8`, resolver log evidence `0a9b265c-794b-478b-a9b7-6aaf8d2f86ec`, uninstall cleanup, and locked-state restoration. |
| 2026-08-15 | codex | partial | Fixed the device-control CLI reconnect manifest binding (`id` → `device_id`) exposed by the server-owned suite, regenerated primitive evidence, and verified CLI tests plus CLI Health with zero errors. The remaining architecture warnings are pre-existing manifest maturity debt, not binding errors. |
| 2026-08-15 | codex | partial | Removed the resolved visual-understanding route and strategy-floor entries from the problem register; the live vision route and producer evidence verifier remain documented in the progress and architecture records, while black capture and operator-unlocked physical proof remain active constraints. |
| 2026-08-15 | codex | partial | Mounted the production target-resolution API, wired its resolver through the generated `ai-gateway` client, and verified a live `locate.visual` response from the running services (`rung=vision`, provider `ollama`, model `qwen3-vl:4b`). Added producer-owned absolute review-recording path reporting while keeping cross-scenario evidence references path-free. The Galaxy A03s physical journey remains open because the operator-unlocked surface is still unavailable. |
| 2026-08-13 | codex | partial | Completed the plan's traceability and documentation pass, including structural redaction verification, evidence classes, the bridge device-lease dispatch gate, and deleted resolved problem entries. A governed Device Control flow on Galaxy A03s serial `R9TT608Q6MH` now produces a native, redaction-verified recording: run `5773894d-000c-42d1-9e26-be783e96fa08`, 20,856 bytes, SHA-256 `fee5429cca21fc6fb78478b2743850349ae7011121e11afd37523515be784113`, effective 30.92 FPS against a 15 FPS animation minimum, with device-state restoration passed. Targeted Go/API/CLI tests and UI type-check pass; full Test Genie/baseline and browser experience gates remain infrastructure-degraded and are not claimed as passing. |
| 2026-08-13 | codex | partial | Follow-up live validation found the Galaxy A03s `screenrecord` path can produce a valid H.264 container whose display body is uniformly black even while ADB screenshot/state probes report an awake launcher. Added bounded decoded-content validation at the evidence boundary; flow `8f60f7cb-57c2-4739-ae88-eb3074035fb4` now fails at recording-stop with `video body is uniformly near black` and publishes no evidence. The underlying Android capture path remains open for repair; black media is no longer accepted as proof. |
| 2026-08-13 | codex | partial | Resumed secure-authentication validation: added safe unlock audit metadata and SQLite migration coverage, repaired the DVC-P0-013 to OT-P0-013 traceability link, refreshed stale architecture/error-handling documentation, and re-ran API/CLI/requirements checks. The governed Galaxy A03s reconnect still reports no USB device, so locked-to-unlocked authentication acceptance remains open. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
