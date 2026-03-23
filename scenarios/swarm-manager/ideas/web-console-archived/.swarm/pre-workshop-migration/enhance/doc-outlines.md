# Documentation Outlines: Web Console

> Prepared for process step to incorporate into the revived web-console scenario.

## README.md Sections

The archive/README.md provides a solid foundation. The revived README should update these sections:

### Sections to Keep (with updates)
- **Product Goals** — update to reflect SQLite-only storage, single-user model
- **Target UX and Behavior** — keep pane layout, terminal fidelity, session continuity, AI input, launcher, mobile toolbar, drawer sections
- **Architecture Direction** — update API layer to specify SQLite persistence instead of "durable transcript/session persistence model"; update dependencies to remove Redis/Postgres
- **Validation and Testing Expectations** — keep as-is, good traceability approach

### Sections to Update
- **Dependencies > Required** — remove "Authenticated reverse proxy headers" (handled by api-base). Add `packages/api-base` explicitly.
- **Dependencies > Optional** — remove "Redis/Postgres adapters for future persistence extensions" entirely
- **Configuration Model** — add SQLite database path configuration; keep session policy, shortcut catalog, AI provider policy, routing sections

### Sections to Add
- **Storage Architecture** — new section describing SQLite setup, schema initialization, WAL mode, scenario isolation
- **Single-User Design** — brief note explaining the personal server assumption

## PROBLEMS.md Entries

Known risks and deferred issues to track:

1. **Interactive CLI Fidelity Edge Cases** — PTY handling for complex interactive CLIs (Claude Code, Codex) may have edge cases around resize during active output, cursor reporting conflicts, and reconnect during mid-escape-sequence. Requires dedicated e2e validation.

2. **Mobile Browser Variability** — Floating keyboard toolbar behavior varies across mobile browsers (Safari iOS, Chrome Android, Firefox Mobile). Virtual keyboard interaction with xterm.js focus management needs testing on real devices.

3. **SQLite Concurrency Under Load** — While single-user bounds concurrency, rapid terminal output from multiple panes could create write contention on the SQLite transcript table. WAL mode mitigates but may need monitoring.

4. **AI Provider Timeout Tuning** — Ollama timeout before OpenRouter fallback needs empirical tuning. Too short = unnecessary fallbacks; too long = poor UX when Ollama is down.

5. **Offline Output Buffer Growth** — Sessions with default never-expire policy could accumulate unbounded transcript data in SQLite. May need eventual transcript rotation or archival strategy (deferred — not MVP).

## PROGRESS.md Initial Entry

```markdown
# Web Console Progress

## 2026-02-18 — Revived from Archive

- Item revived as standalone scenario from archived web-console
- All 7 clarification questions answered
- Key decisions: SQLite-only storage, single-user, mobile P0, api-base for auth
- Enhanced plan produced and ready for processing
- Next: Process step to scaffold scenario and generate PRD
```
