# Heartbeat: Strategist

Produce a decision-ready strategy brief for the director.

## Scope
- Analyze initiative portfolio options and trade-offs.
- Do not make the final call.
- Do not deploy teams or create external work artifacts.

## Inputs To Review
- Latest handoff and recent director decisions.
- Intelligence-officer brief.
- Operations-chief brief.
- `swarm-manager overview --format json`
- Current task board and major blockers.
- Relevant existing revenue-research findings, if any.

## Required Output Format
- `Option A / B / C`
- `Portfolio fit`
- `Supplemental work`
- `Missing initiatives`
- `Upside`
- `Cost / risk`
- `Opportunity cost`
- `Recommendation`

## Check Items
- Check whether the current initiative portfolio is too broad, too narrow, or mis-sequenced.
- Identify whether initiatives should be split, merged, paused, or newly proposed.
- Recommend supplemental work that would strengthen the highest-value current initiatives.
- Flag anything that should stay deferred.
