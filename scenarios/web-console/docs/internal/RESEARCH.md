# Web Console — Research Notes

## Related Scenarios

- **browser-automation-studio**: Reference for React + Vite template pattern, iframe bridge, api-base integration.
- **app-monitor**: Reference for pane-based workspace ergonomics (web-console layout should align).

## Key Technical References

- **xterm.js**: Terminal emulator for the browser. Core dependency for terminal fidelity.
- **creack/pty**: Go library for PTY allocation and management.
- **SQLite WAL mode**: Write-ahead logging for concurrent read/write on single-user workloads.
- **packages/api-base**: Shared HTTP/WebSocket routing for proxy-correct networking.

## Design Decisions from Clarification

| Decision | Rationale |
|----------|-----------|
| SQLite-only storage | Single-user design eliminates need for Postgres/Redis. Simplifies deployment. |
| Auth via api-base | No custom auth implementation. Parent scenario handles authentication. |
| Single-user model | Personal server assumption. No multi-user isolation needed. |
| Mobile as P0 | Operators actively use phones for terminal access. |
| AI context = prompt + last N lines | No environment variable injection. Keeps context simple and predictable. |
| No specific performance targets | "Feels responsive" is the bar. No ms-level SLA. |

## Archive Origin

This scenario was revived from an archived web-console scenario. The archive contained:
- PRD.md (v3.0.0) with 8 P0, 4 P1, 2 P2 operational targets
- README.md with architecture direction and UX specifications
- requirements/index.json with 12 requirement modules

All archive materials were preserved and incorporated into the revived scenario.
