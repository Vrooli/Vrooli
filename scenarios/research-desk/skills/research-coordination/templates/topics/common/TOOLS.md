# Tools — {{title}}

Capabilities whose failure changes this topic's work, each with its fallback. Observations go to
runs, not here. The supervisor's `scripts/topic_health.py` reads this file and names each tool
that is down or `unknown` with the fallback below.

## Search

- **Used for:** finding prior work, code and topic knowledge.
- **Fallback:** `rg` over the topic folder and the repository.
- **Monitoring scenario:** `search-hub`

## Skills and programs

- **Used for:** reading shared and topic skills, running supervisor and topic programs.
- **Fallback:** read `SKILL.md` by path; run a program's underlying commands.
- **Monitoring scenario:** `prompt-manager`, `program-runtime`

## Decisions

- **Used for:** delivering needs-you items with their pages.
- **Fallback:** the item stays in `STATE.md` and `ATTENTION.md`; the supervisor delivers it later.
- **Monitoring scenario:** `notification-hub`, `research-desk`
{{mode_tools}}
