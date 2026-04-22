# Audience Observations

Raw, pre-structured audience / competitor / trend observations that don't yet meet the threshold for a `shared/audience-scans.jsonl` entry OR don't yet have a clear persona key to attach to.

**Promotion target:**
- If an observation stabilizes and attaches to a persona → `audience-update` decision → `AUDIENCES.md` edit.
- If an observation is benchmark-adjacent for monetization → cross-team knowledge entry under `monetization-benchmark-adjacent/<topic>`.
- If an observation is a pattern about how to do research itself → `documentation-health` skill update or new skill proposal.

**Retirement signal:** observation has been promoted into a structured scan, an `AUDIENCES.md` revision, or a market-validator-consumed entry — AND ≥30 heartbeats have passed without the observation being cited again.

**Revisit marker (file-level):** revisit after 14 heartbeats.

## Entries

_No entries yet._

Append new entries in the shape below:

```markdown
### <YYYY-MM-DD> — <short description>

**Written by:** <member-id, typically researcher>
**Observation:** <concrete, citation-grounded>
**Source:** <URL, post ref, repo link>
**Interpretation flag:** observation | light-interpretation | heavy-interpretation
**Possible persona attachment:** <persona-key-or-null>
**Cross-team relevance:** <null or "monetization-benchmark-adjacent/<topic>">
**Revisit marker:** revisit after N heartbeats
```
