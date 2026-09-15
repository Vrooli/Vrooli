# Finite delivery wake

This effort is enabled and incomplete. Treat this wake as an advance/recovery pass, not a status-only review.

1. Read the owner projection first:
   `agent-manager effort board --effort-ref effort:3132d75d-c3b7-42fd-9441-ba0a0a9e2887 --json`
2. Read the accepted effort workspace at `/home/matthalloran8/.vrooli/plan-artifacts/efforts/rcl-prompt-manager-ux-design-campaign/`, especially `README.md`, `sources/intent.md`, `sources/campaign-contract.md`, `requirements.json`, and `handoffs/`.
3. Continue the exact retained task/run identity when one is active. Otherwise select the highest-priority unfinished action from the workspace. The current campaign is incomplete; do not use `no-action`.
4. The design contract and bounded implementation surface are already shaped in the accepted workspace. Dispatch the first bounded implementation child directly through Agent Manager now, naming the exact Prompt Manager or React Component Library surface, allowed paths, acceptance evidence, and handoff. No roster member is required for a child: an empty team org beyond the coordinator does not mean “no child”; create the durable child run with the coordinator as parent. Use Plan Manager only if the selected action is genuinely plan-backed, and invoke its autonomous owner route without human workshop or acceptance.
5. Claim/update the existing P1 task so the board reflects `in-progress`, and record the resulting handoff and exact next action in the Source Ledger. If a true owner grant is missing, report the exact grant and reopening condition as a typed blocker; do not claim completion.

The final handoff must state changed, verified, remaining, unverified, exact next action, task/run/plan references, and evidence references. The effort may be completed only with its explicit revision-checked completion receipt.
