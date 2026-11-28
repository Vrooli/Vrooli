# Progress — Research Desk

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-11-11 | design owner (agent) | done | Generated from react-vite; PRD and requirements written; first design. |
| 2026-11-11 | design owner (agent) | done | Review 1 (design): answers survive a desk outage through the Hub, states derived from evidence, the governed launcher, one packet renderer and one needs-you grammar, pause/resume/stop without chat, since-last-look, and a no-growth test. One design doc, one file contract. |
| 2026-11-11 | skills owner (agent) | done | Skills move: `research-coordination`, `adaptive-research` and `local-experimentation` moved into `skills/` with links at the old paths; `research-review` added; `packet.py`, templates and `limits.json` (ceilings) added. Review 2 (skills): the operator orders first workstreams, card notes, models validated, a failed size gate names what to cut. The links' removal is open (PROBLEMS.md). |
| 2026-11-12 | API owner (agent) | done | API and CLI built: three domains, courier, launcher, control. Review 3 (code): no swallowed read errors, single sources for limits and lenses, one error mapper, txtar fixtures and `make real-home-check`, controls reach only the commission's own thread, turning the courier on never sends an old item, only the live instance writes. |
| 2026-11-12 | UI owner (agent) | done | Phone UI built over the API: Now, Needs you, Topics, Commissions, Launch, Reviews, Desk, and a research-home file view. Review 4 (UI): blockers resolved in the launch request, suggested quotes, launch messages as a read, the API owns the words and the delivery status, a 24-hour default window, research writes shown apart from mission and review edits. |
| 2026-11-12 | authorization repair (agent) | done | Launch and Control take an agent with an approved `research-management` grant through the canonical evaluator, checked for every superseded commission before any effect; other operator actions stay operator-only. |
| 2026-11-12 | API owner (agent) | done | Review 5 (debt): `launch` and `control` not runnable by agents in the CLI manifest, with a test against the API; codes and limits on the wire instead of prose; one first-sightings record for the courier (old state migrated once); `launch/service.go` split; real-home check fails on an absent home; docs cut to the four-file reading set; no review citations left in code or docs. |
| 2026-11-12 | API owner (agent) | done | Validation (charter goals on the real home): suggested quotes only from desk-built packets or started commissions; owners' reading includes the topic mission with a tidy flag; any RFC 3339 heading time, unparsed headings reported with their line; an unknown end is the last write, open overlaps computed in the API; older needs-you lists are feed rows; without a look, ledger sections by dated headings and a plain "not shown"; the launch check shows the packet's own `STATE.md` start; steering has a dry run; one `DECISIONS.md`. |
| 2026-11-26 | design study (agent) | proposed | Three complete UX redesigns (`internal/proposals/UX-REDESIGN-20261126/`): The Bridge (recommended), The Desk and The Constellation. Captured current state at 390/1440 light and dark; 32-group feature parity map; path through EX1/EX3/EX7/EX8/EX9/EX11/EX12. Proposal only: no scenario code, API, supervisor file or commission changed. |
| 2026-11-26 | design study (agent) | proposed | Replaced the first study with a second built from the operator's three references (`internal/proposals/UX-REDESIGN-20261126-v2/`): light canvas, white cards, electric blue, lime, black pills, big numbers, soft tiles. Three structures (Daylight recommended, Index, Week). The first study is retained for its captures and 34-group inventory and is marked superseded. Proposal only: no scenario code, API, supervisor file or commission changed. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`../DECISIONS.md`](../DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/SYSTEM.md`](../concepts/SYSTEM.md) — the design
