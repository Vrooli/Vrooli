# Web Console — Known Problems and Risks

## 1. Interactive CLI Fidelity Edge Cases

PTY handling for complex interactive CLIs (Claude Code, Codex) may have edge cases around resize during active output, cursor reporting conflicts, and reconnect during mid-escape-sequence. Requires dedicated e2e validation.

## 2. Mobile Browser Variability

Floating keyboard toolbar behavior varies across mobile browsers (Safari iOS, Chrome Android, Firefox Mobile). Virtual keyboard interaction with xterm.js focus management needs testing on real devices.

## 3. SQLite Concurrency Under Load

While single-user bounds concurrency, rapid terminal output from multiple panes could create write contention on the SQLite transcript table. WAL mode mitigates but may need monitoring.

## 4. AI Provider Timeout Tuning

Ollama timeout before OpenRouter fallback needs empirical tuning. Too short = unnecessary fallbacks; too long = poor UX when Ollama is down.

## 5. Standards: Setup Steps Configuration (MEDIUM)

**RESOLVED** (2026-02-19): Simplified setup steps in service.json (removed conditional wrappers). Auditor now reports 0 standards violations.

## 6. Go Lint: Remaining errcheck Warnings

**RESOLVED** (2026-02-19): Fixed the 2 remaining errcheck warnings in `session_test.go` by wrapping deferred Delete calls in `func() { _ = sm.Delete(id) }()`.

## 7. Lighthouse SEO: 82% (Below 90% Threshold)

**RESOLVED** (2026-02-19): Added meta description to index.html and robots.txt. Lighthouse SEO now 100%.

## 8. Offline Output Buffer Growth

Sessions with default never-expire policy could accumulate unbounded transcript data in SQLite. May need eventual transcript rotation or archival strategy (deferred — not MVP).

## 9. E2E Issues

**PARTIALLY RESOLVED** (2026-02-19): Added BAS workflows for terminal command execution, route-level session persistence, reconnect replay, and multi-pane independence:
- `bas/cases/01-foundation/01-terminal/launch-custom-command-executes.json`
- `bas/cases/01-foundation/01-terminal/session-metadata-persists-across-route-navigation.json`
- `bas/cases/01-foundation/01-terminal/reconnect-offline-buffer-replay.json`
- `bas/cases/01-foundation/01-terminal/multi-pane-independent-io.json`
- `bas/cases/01-foundation/01-terminal/interactive-stdin-roundtrip.json`
- `bas/cases/01-foundation/01-terminal/session-persists-across-full-reload.json`

**Remaining gaps:**
- No BAS mobile viewport workflow yet for floating toolbar key/chord behavior.
- Playbooks phase currently blocks on BAS startup when browser-automation-studio dependencies are unavailable (e2e workflows present but cannot execute).
