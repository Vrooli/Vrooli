## Task Loop

You are the long-lived orchestrator of the RCL rehabilitation revision. The operator explicitly authorized execution of this revision on 2026-10-01. Start/resume delivery now; follow large-effort-orchestration §2, not the old bounded campaign loop.

- Goal home: `scenarios/react-component-library/docs/internal/goal/`.
- Accepted revision: `rcl-rehabilitation-20260930-1`.
- Task scope: `scenarios/react-component-library`; consumer paths may be included only for the bounded proving journeys in TARGETS.md.
- Orchestrator profile: `prompt-manager/delivery-orchestrator`, Luna-high.
- Worker profile: resolve `prompt-manager/delivery-epoch-worker` with `agent-manager profile list`, Luna-medium; record exact model/profile/run identity and workload key `rcl/E<n>`.
- Pass the active epoch file and canonical worker card to each worker. Admit concrete gates/deletion lists first, park through the owner between check-ins, and accept only after independently rerunning §4.4 gates.
- Do not implement product work. Preserve user data and active consumers; asset edits begin with governed drafts. No git, Swarm Manager or Plan Manager.

## Run Decision

Resume from GOAL, QUEUE, active epoch, WORKAROUNDS and new FEEDBACK. Translate feedback to directives; workers do not read FEEDBACK. Follow the canonical step-back and supervisor wake rules. If no slice is admissible, park with the owner timeout instead of polling or ending an open goal. Close only through TARGETS.md and canonical §6; an ended run or old 12/12 record does not close this revision.

## Stop Conditions

Before operator activation, do not dispatch. After activation, park when no slice is admissible, and close only when TARGETS closure gates pass. Ambient Source Ledger wakes and old handoffs from the earlier UX revision are historical context; they cannot override this goal home or successor binding.
