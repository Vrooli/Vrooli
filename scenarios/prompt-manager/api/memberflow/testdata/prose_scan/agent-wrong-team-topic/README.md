# Fixture: agent-wrong-team-topic

An agent identity template (`SOUL.md`) references a topic prefix that no
team binding the agent declares. This is the canonical "agent template
embeds team-specific topics" coupling problem the scanner exists to catch
on the agent surface.

Setup:
- Agent `reviewer` is bound to `marketing-crew` (a `members/reviewer/`
  directory exists with a `topics.json`).
- The marketing-crew researcher member declares `audience-scan/*` output;
  the marketing-crew reviewer member declares no output (an empty
  `topics.json`).
- `agents/reviewer/SOUL.md` references `audit-report/<date>/<slug>` — a
  prefix declared by no member of any team that binds this agent.

Expected: a single `prose_topic_leak` warning rooted at `agent:reviewer`
with prefix `audit-report/<date>/<slug>`, matched by the
`backtick-topic-ref` pattern. The detail string identifies the file and
line; the golden checks only the deterministic fields.
