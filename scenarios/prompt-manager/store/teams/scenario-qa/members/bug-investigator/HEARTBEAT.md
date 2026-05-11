# Heartbeat: Bug Investigator

## Reasoning Framework
The Inbox Flow section above is generated from `topics.json`; it is authoritative for taxonomy, drain procedure, and write contract. This file adds only the per-heartbeat task loop and handoff shape.

Investigation method comes from `docs/scenario-qa/methods/investigation/<slug>.md` paired with the matching skill. Default for every signal type is `scientific-debugging` (load with `prompt-manager skill read scientific-debugging`); other techniques activate via `meta-self-improvement` decisions.

## Task Loop
1. List unrouted bug-inbox entries, severity-ordered (`prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/`).
2. Pick the highest-severity entry not already in flight. Cap: at most 1 new investigation started per heartbeat (in addition to continuing any in-flight investigation).
3. Validate the entry's signal-type assignment against `docs/scenario-qa/taxonomies/bug-report/README.md`. If mismatched, record the correction in the bug-investigation entry's body — do not silently fix.
4. Pick a registered investigation technique. Default: `scientific-debugging`. Load the paired skill and apply it.
5. Take the smallest useful action from the taxonomy's `actionSelection` set: `drop`, `observe`, `file-backlog`, `file-decision`, `route-to-another-topic`, `capability-gap`.
6. Write a `bug-investigation-report/<slug>` entry conforming to the schema (front-matter + body sections "Findings" and "Action taken"). The slug should match the bug's slug for traceability.
7. Close the original bug-inbox entry per the action taken (delete on `drop`, retag on `route-to-another-topic`, leave-with-pointer on `capability-gap`).

## Handoff Shape
### Inbox state
### Investigation in flight
### Investigation closed this heartbeat
### Technique applied
### Action taken
### Backlog item / decision created
### Capability-gap raised
### Surface for technique graduation
