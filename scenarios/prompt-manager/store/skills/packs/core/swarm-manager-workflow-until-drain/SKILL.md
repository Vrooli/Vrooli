# Swarm Manager workflow: until drain

## Contract

Work the accepted Plan Manager execution as one warm, resumable engagement.
The workflow's `until` value is the completion test; it is not a replacement
for these operating instructions and must never be treated as agent narration.

Re-read the authoritative plan execution frontier before deciding whether the
engagement is complete. Preserve the accepted scope, write only within the
declared scope, and record evidence for each iteration. Continue the same
conversation when more work is required.

## Outcomes

| Observable state | Outcome |
|---|---|
| Every accepted phase is complete and evidence is applied | `complete` |
| A human decision or external approval is required | `needs_review` |
| The plan or repository is not actionable | `needs_attention` |
| The operator declines to continue | `abstained` |
| A governed budget ends the engagement | `budget_exhausted` |
| The agent cannot continue safely | `blocked` |
| An unexpected execution error occurs | `failed` |

Never claim completion from a handoff alone: Plan Manager execution state is
the completion authority.
