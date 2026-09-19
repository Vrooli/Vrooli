# Fixture: member-undeclared-topic

A member's RESPONSIBILITIES.md instructs the agent to write to a topic
prefix that is **not declared** in that member's `topics.json::output[]`.

This is the canonical Class-II prose drift the scanner exists to catch:
the static graph (`topics.json`) and the prose the agent reads on every
heartbeat disagree, and the agent will follow the prose.

Setup:
- `marketing-crew/researcher` declares output `audience-scan/*` only.
- The same member's `RESPONSIBILITIES.md` cites
  `prompt-manager team knowledge-add ... --topic="campaign-draft/<slug>"`.
- `campaign-draft/*` is declared by **no one** on the team — neither as
  output, intake, required_read, nor evidence_consumed.

Expected: a single `prose_topic_leak` warning rooted at
`team:marketing-crew/researcher` with prefix `campaign-draft/<slug>`.

The detail string (in the live Finding) names the file path, line number,
and the matched pattern (`cli-knowledge-add-topic`). The golden checks
only the deterministic fields — Detail is intentionally excluded so the
golden does not embed absolute paths.
