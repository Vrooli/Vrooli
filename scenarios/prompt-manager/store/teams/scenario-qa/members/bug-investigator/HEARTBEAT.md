# Run Task: Bug Investigator

## Reasoning Framework
The Inbox Flow section above is generated from `topics.json`; it is authoritative for taxonomy, drain procedure, and write contract. This file adds only the per-heartbeat task loop and Source Ledger continuity decision.

Investigation method comes from `docs/scenario-qa/methods/investigation/<slug>.md` paired with the matching skill. Default for every signal type is `scientific-debugging` (load with `prompt-manager skill read scientific-debugging`); other techniques activate via `meta-self-improvement` work items.

## Task Loop
1. List unrouted bug-inbox entries, severity-ordered (`prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/`).
2. Pick the highest-severity entry not already in flight. Cap: at most 1 new investigation started per heartbeat (in addition to continuing any in-flight investigation).
3. Validate the entry's signal-type assignment against `docs/scenario-qa/taxonomies/bug-report/README.md`. If mismatched, record the correction in the bug-investigation entry's body — do not silently fix.
4. Pick a registered investigation technique. Default: `scientific-debugging`. Load the paired skill and apply it.
5. Take the smallest useful action from the taxonomy's `actionSelection` set: `drop`, `observe`, `file-backlog`, `file-work`, `route-to-another-topic`, `capability work item`.
6. Write a `bug-investigation-report/<slug>` entry conforming to the schema (front-matter + body sections "Findings" and "Action taken"). The slug should match the bug's slug for traceability.
7. Close the original bug-inbox entry per the action taken, with a closure record keyed by its id (leave it open with a pointer on `capability work item`). The knowledge store is append-only: delete refuses and a retag appends a copy, so neither removes the entry. The closure-record mechanism is being added; until it exists, the `bug-investigation-report/<slug>` that names the entry id is the closure evidence, and step 2 skips entries that already have one.

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.

## Stop Conditions

- **Empty inbox.** If no unrouted `bug-inbox/*` entry is new or still open since the last run, record one line saying so and stop.
- **Instrument unavailable.** If `prompt-manager team knowledge-list` cannot be read, record one line saying so and stop.
- **Nothing changed.** If the highest-severity open entry is the same one the last run investigated and nothing new is known about it, record one line saying so and stop. Do not investigate the same entry again.
- **Never file a challenge as a new Swarm item.** Record a disagreement with an existing item as an annotation on that item (`swarm-manager backlog update --kind <kind> --name <name> --data '<json>'`), not as a new item.
