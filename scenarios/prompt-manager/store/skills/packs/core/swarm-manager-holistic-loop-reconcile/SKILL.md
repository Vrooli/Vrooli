# Swarm Manager Holistic Loop Reconcile

You are running the `holistic-loop` reconcile phase for initiative `{{INITIATIVE_NAME}}`. The round refresher auto-started this phase the moment the prior `review` phase completed.

## Why this phase exists

The holistic-loop is operator-gated: the operator decides whether to keep iterating after each round. Across iterations the *initiative backlog* drifts from what the *code* actually does — items that describe work the round already completed, items the round outgrew, missing follow-ups the round surfaced. Without this reconciliation step that drift compounds and the backlog stops being a faithful map of the initiative.

Your job is to read everything the round produced and propose backlog mutations that bring the initiative spec back in line with reality. **You read; the operator decides.** Mutations are surfaced as a checklist; the operator accepts or rejects them individually.

## Context

- Initiative: `{{INITIATIVE_NAME}}`
- Title: `{{INITIATIVE_TITLE}}`
- Mode: `{{OPERATING_MODE}}`
- Phase: `{{PHASE}}`
- Round: `{{ROUND_NUMBER}}`
- Agent profile: `{{AGENT_PROFILE_KEY}}`

Description:

{{INITIATIVE_DESCRIPTION}}

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Member items:

```json
{{MEMBER_ITEMS_JSON}}
```

Mode artifacts (findings, plan, handoffs from this round and prior ones):

```json
{{MODE_ARTIFACTS_JSON}}
```

Prior rounds:

```json
{{PRIOR_ROUNDS_JSON}}
```

## Your task

1. **Read what the round produced.** Walk `modes/holistic-loop/findings.md`, the bound plan-manager plan (see the plan context in prior rounds), the latest execute and review handoffs, and any code paths the round touched.
2. **Compare against the initiative spec.** Each member item has a `title`, `description`, `acceptance_allow/deny`, `priority`, `effort`, and `depends_on`. Look for items where the real work diverges: completed scope no longer reflected, new follow-ups surfaced, items that should be split or merged because the round revealed coupling, items that drifted into another initiative's domain.
3. **Honor the BacklogSyncPolicy.** This mode allows `propose_mutations`, `mark_complete`, `create_followups`, and `update_scope`. Do not propose mutations outside that capability set.
4. **Be conservative.** Propose only mutations the round produced *evidence* for. The operator-gated apply means a noisy proposal is annoying but not harmful — but a misleading rationale hides the actual signal. If you have low confidence in an item's drift, leave it for the next round.

{{BACKLOG_SYNC_PROPOSAL_SNIPPET}}

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result`. The `backlog_sync.proposal` field carries the envelope above; the surrounding fields are summary metadata the round payload records:

```json
{
  "operating_mode_result": {
    "backlog_sync": {
      "completed_items": [],
      "created_items": [],
      "updated_items": [],
      "proposal": {
        "form": "mutation_list",
        "rationale": "...",
        "mutations": []
      },
      "rationale": "Why this proposal addresses what the round produced."
    },
    "handoff": {
      "summary": "...",
      "next_step": "operator review of backlog-sync proposal"
    }
  }
}
```

If the round didn't surface anything that warrants a backlog change, emit `proposal.mutations: []` with a one-sentence envelope rationale explaining what you considered and ruled out. Empty proposals are valid and expected.
