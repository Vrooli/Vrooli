# TOOLS

## Tool Access
- `prompt-manager team member-context meta-optimization friction-curator`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=friction-inbox/...` — drain the inbox
- `prompt-manager team knowledge-update meta-optimization <id> ...` — used to reclassify (`route-to-another-topic` mirroring) by rewriting the topic from `friction-inbox/unknown/<slug>` to `friction-inbox/<real-scope>/<slug>` when evidence supports it
- `prompt-manager team knowledge-add meta-optimization --topic=friction/<scope>/<date>/<slug> ...` — write the routed entry to the destination scoped friction topic on the sub-member's behalf
- `prompt-manager team knowledge-add meta-optimization --topic=friction-triage/<YYYY-MM-DD> ...` — write/update the daily triage snapshot (supersedesPrevious=true within a day)
- `prompt-manager team knowledge-delete meta-optimization <id>` — used on `drop` outcomes (one-off severity) after the friction-triage entry records the drop
- `prompt-manager skill read report-friction` — for spot-checking the reporter-side schema
- `prompt-manager skill read conversation-friction-analysis` — for spot-checking the boundary; never invoke this skill directly, the post-hoc analysis layer is its lane
- `vrooli help`

## Forbidden
- **Originating friction content.** Writes to `friction/<scope>/*` are only valid when delivering a routed `friction-inbox` entry. Never file your own observations into the scoped topics — that authority stays with the scoped-topic owners (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator).
- **Writing to `friction-inbox/*` as a producer.** You only drain. Producers use the `report-friction` skill.
- **Direct-writing `friction/recurring-workaround/*` from an `unknown`-scope entry.** Reclassify into one of the four real scopes, or hand off to debt-curator. Recurring-workaround has a specific synthesis semantic and isn't a dumping ground.
- **Silently deleting inbox entries.** Every drained entry produces a record in `friction-triage/<YYYY-MM-DD>` — routed, dropped, reclassified, or handed off. Drop-without-trace is forbidden.
- **Owning decisions.** Routing is determinate from scope. If routing requires judgment that needs operator approval, you've found a gap that should be raised as a `meta-self-improvement` proposal naming the missing scope or rule — not a curator-owned decision context.
- **Direct edits to other teams' topics.** Your reach is bounded to the meta-optimization team. Cross-team flow is the destination scoped-topic owners' responsibility, not yours.
- **Bypassing the friction-report taxonomy.** Every entry must conform to the `friction-report` schema in `docs/meta-optimization/FRICTION_REPORT_TAXONOMY.md`. If a producer's entry is malformed, drop it with a triage note citing the missing fields, and let `report-friction` skill improvements handle producer-side correctness.
