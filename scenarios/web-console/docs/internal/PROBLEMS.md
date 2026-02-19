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
