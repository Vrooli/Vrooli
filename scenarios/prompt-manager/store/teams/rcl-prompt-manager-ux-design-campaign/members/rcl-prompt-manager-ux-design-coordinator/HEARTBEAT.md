# Finite delivery wake

## Task Loop

Treat each wake as a bounded delivery/recovery pass against the accepted effort
workspace, never as a status-only heartbeat. Preserve the exact effort, team,
task, run, revision, owner, and evidence identities. Read the owner projection
and current campaign workspace before selecting work. Select the highest
unfinished action whose acceptance evidence is still missing; do not report
`no-action` while product requirements or readiness gates remain open.

For a selected action, state the exact surface, allowed paths, acceptance
evidence, and handoff before dispatch. Reuse an active task/run when its identity
is authoritative; do not invent a replacement after an uncertain dispatch. Keep
the team disabled unless operator activation is present. A disabled team may
write durable knowledge and handoffs, but it must not imply execution or claim
that the campaign is complete.

Reconcile implementation evidence with the campaign ledger. Focused tests,
builds, and screenshots support a claim but do not replace the required visual,
behavioral, owner, and completion gates. Record unavailable or contradictory
evidence explicitly and retain the exact reopening condition.

## Run Decision

Record one typed disposition for the wake in the owner system and Source Ledger:
`existing-action-reference` when an authoritative action is active,
`new-action-candidate` only when no existing action covers the selected gap,
`blocked` only for a missing authority/capability that cannot be repaired in
scope, or `complete` only after the revision-checked completion receipt exists.
The handoff must include changed, verified, remaining, unverified, exact next
action, task/run/plan references, and evidence references. Do not turn a paused
team, a passing focused test, or a startup heartbeat into a completion claim.

This effort is enabled and incomplete. Treat this wake as an advance/recovery pass, not a status-only review.

1. Read the owner projection first:
   `agent-manager effort board --effort-ref effort:3132d75d-c3b7-42fd-9441-ba0a0a9e2887 --json`
2. Read the accepted effort workspace at `/home/matthalloran8/.vrooli/plan-artifacts/efforts/rcl-prompt-manager-ux-design-campaign/`, especially `README.md`, `sources/intent.md`, `sources/campaign-contract.md`, `requirements.json`, and `handoffs/`.
3. Continue the exact retained task/run identity when one is active. Otherwise select the highest-priority unfinished action from the workspace. The current campaign is incomplete; do not use `no-action`.
4. The design contract and bounded implementation surface are already shaped in the accepted workspace. Dispatch the first bounded implementation child directly through Agent Manager now, naming the exact Prompt Manager or React Component Library surface, allowed paths, acceptance evidence, and handoff. No roster member is required for a child: an empty team org beyond the coordinator does not mean “no child”; create the durable child run with the coordinator as parent. Use Plan Manager only if the selected action is genuinely plan-backed, and invoke its autonomous owner route without human workshop or acceptance.
5. Claim/update the existing P1 task so the board reflects `in-progress`, and record the resulting handoff and exact next action in the Source Ledger. If a true owner grant is missing, report the exact grant and reopening condition as a typed blocker; do not claim completion.

The final handoff must state changed, verified, remaining, unverified, exact next action, task/run/plan references, and evidence references. The effort may be completed only with its explicit revision-checked completion receipt.
