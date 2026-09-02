# Run Task: Vision Walk Prep

## Downstream Consumer

Write the prep deliverable to the declared Source Ledger topic for this member. The later `morning-vision-walk` agent reads that topic and uses it to guide a conversational work item and ideation session with the operator.

Optimize for phase-aligned briefing notes: concise enough to skim, specific enough to support work items, and clearly separated into durable fields in the declared topic. Do not answer the operator's work items. Prepare the context so the walk agent can ask clearly.

## Task Loop
1. Find the reusable morning-prep program by running `program-runtime library search "prepare the morning vision walk briefing" --json`; do not use a hard-coded filesystem path or assume a program name. Select the top promoted result and record its name and version.
2. Read that program with `program-runtime library get <name> --version <version> --json`. Confirm its contract covers the morning walk and inspect every returned drift row. If the search, read, contract, or drift check fails, fail this heartbeat with the evidence; never emit a placeholder handoff.
3. Run the selected program in a new governed session with `program-runtime sessions create` and `program-runtime programs submit --provenance agent`. Treat the bounded projection as RuntimeData and use its real values in the handoff. Reject the run if it contains `example.invalid` or is shorter than 200 characters.
4. Gather retrospective portfolio changes and recent Source Ledger entries. Include the latest completed Swarm Manager execution and the current `prompt-manager team heartbeat-fleet-health` aggregate. Record the aggregate's `generatedAt`, `windowHours`, `enabledMembers`, `succeededMembers`, `membersWithTwoFailures`, `successPercent`, `thresholdPercent`, and `meetsThreshold` fields in a `### Fleet-health observation` section of today's `vision-walk-record/YYYY-MM-DD/morning-briefing`; treat the CLI's exact threshold verdict as authoritative rather than rounding it in prose. Derive schedule context from current heartbeat state rather than assuming every enabled member runs daily. When `membersWithTwoFailures` is non-zero or `meetsThreshold` is false, inspect the affected members with `prompt-manager team heartbeat-list <team>`, name each failure streak, and record a bounded explanation or explicitly mark it unexplained. If the aggregate command is unavailable, record that evidence gap instead of inventing a health result. Read the latest `goal-portfolio-record/YYYY-MM-DD` snapshot and copy its bounded staleness-verdict section into the briefing with every typed reference, verdict, and evidence statement intact. If the section is absent, say so explicitly instead of inferring verdicts.
5. Gather pending portfolio and cross-team work items.
3. Pull monetization, marketing, meta-optimization, and infra-health work type.
4. Run `prompt-manager team handoff-latest director-swarm vision-walk-prep` before synthesis. Preserve any existing `## Walk Checkpoint` section verbatim. Treat the CLI result as RuntimeData-class state. Do not check for a handoff in the Config-class team store.
5. Identify life-audit prompts and big-picture context. Frame the big-picture context with the ecosystem lens (`path:docs/concepts/ECOSYSTEM.md`): for active/stalled scenarios and candidate ideas, note which interface(s) they serve or enable, their functional role, and compound-value/bundle fit — so the walk's ideation starts from how things fit the whole, not just the app in isolation.
6. State the self-improvement loop ladder (see RESPONSIBILITIES) with the evidence behind each rung claim.
7. Record the Source Ledger continuity without mutating queues or choosing answers. Confirm the dated `vision-walk-record/YYYY-MM-DD/morning-briefing` write receipt so the fleet-health observation is durable evidence rather than only final-response prose.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Handoff Output

End the final response with a `## Handoff` section. Put the complete phase-aligned morning briefing in that section. This section is the RuntimeData-class snapshot that `prompt-manager team handoff-latest director-swarm vision-walk-prep` returns. Do not write heartbeat output into the Config-class team store.

The handoff must name the discovered program and version, include its bounded projection and drift result, and contain no `example.invalid` URL. A missing program, failed run, stale binding, or undersized projection is a failed heartbeat, not a successful prose fallback.
