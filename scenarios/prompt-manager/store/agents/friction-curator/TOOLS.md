# TOOLS

## Tool Access
- `prompt-manager team member-context meta-optimization friction-curator`
- `prompt-manager team knowledge-list meta-optimization --topic-prefix=friction-inbox/...` — drain the inbox
- `prompt-manager team knowledge-update meta-optimization <id> ...` — used to reclassify (`route-to-another-topic` mirroring) by rewriting the topic from `friction-inbox/unknown/<slug>` to `friction-inbox/<real-scope>/<slug>` when evidence supports it
- `prompt-manager team knowledge-add meta-optimization --topic=friction-report/<scope>/<date>/<slug> ...` — write the routed entry to the destination scoped friction topic on the sub-member's behalf
- `prompt-manager team knowledge-add meta-optimization --topic=friction-triage-record/<YYYY-MM-DD> ...` — write/update the daily triage snapshot (supersedesPrevious=true within a day)
- `prompt-manager team knowledge-delete meta-optimization <id>` — used on `drop` outcomes (one-off severity) after the friction-triage entry records the drop
- `prompt-manager skill read report-friction` — for spot-checking the reporter-side schema
- `prompt-manager skill read conversation-friction-analysis` — for spot-checking the boundary; never invoke this skill directly, the post-hoc analysis layer is its lane
- `vrooli help`

