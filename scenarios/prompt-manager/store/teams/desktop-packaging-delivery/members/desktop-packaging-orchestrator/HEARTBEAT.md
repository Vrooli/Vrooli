## Task Loop

You are a long-lived run; the team heartbeat only relaunches you when your run is
absent or terminal. Follow `large-effort-orchestration` §2 with these values:

- **First on every wake:** process new or open `FEEDBACK.md` entries before
  anything else. Turn each into directives or queue changes, and move it to
  Resolved with a one-line resolution. Feedback that sits unprocessed is a
  defect.
- **Root causes, not workarounds (operator, DPF-FB-006).**
  - When an epoch is blocked by broken shared tooling, admit a repair slice
    that fixes the cause in its owning place. Do not park it and admit the
    next feature slice.
  - At most one feature epoch may be parked unaccepted at a time.
  - Add one line to `LESSONS.md` at each acceptance or park.
- Goal home: `scenarios/scenario-to-desktop/docs/internal/goal/`. Design of
  record: `docs/architecture/packaging-strategies.md` (amend it when a worker's
  `DESIGN:` slice-log line shows it is wrong; keep §9 Decisions current).
- Task scope: `agent-manager task create --title "Desktop packaging E<n>" --scope-path scenarios/scenario-to-desktop --project-root <repo root> --json`.
  Epochs also touch `scenarios/performance-health`, `scenarios/code-facts`,
  `packages/delivery-ramp-go` and `packages/proto`; name them in the brief's
  Modules section.
- Worker profile: the ID of `prompt-manager/delivery-epoch-worker` from
  `agent-manager profile list`; `--execution-mode interactive`; workload key
  `desktop-packaging/E<n>`; model: the Luna model that profile resolves to,
  effort medium. Pass the epoch file path, the worker card path
  (`scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`),
  your run ID, and "mark your native goal complete when the exit gate passes"
  in `--prompt`.
- E1 is already admitted (`epochs/E1.md`): set `Started`, spawn, record
  `Workers`. Admit E2–E6 from QUEUE.md in order; E4 adds its Needs operator item
  (`sudo vrooli setup` for WebKitGTK headers) at admission.
- Shared services, BAS fallback and daily qualification: `TARGETS.md`.
- Supervisor wake: `large-effort-orchestration` §2 "Waking the supervisor".

## Run Decision

End each check-in as `parked`, `accepted`, `step-back` or `closed`
(`large-effort-orchestration` §2 and §6).

## Stop Conditions

If no slice is admissible, plan one (`large-effort-orchestration` §2.1). Only
when every remaining slice waits on an operator decision or an outside repair,
follow §2's `## Needs operator` steps and park for 72h. Accept only through §4.4.
When E6 is accepted and the GOAL.md destination is met (or every remaining gap
is parked on an operator decision), close per §6: publish the synthesis report
path in QUEUE.md Handoff, list M1 (mobile) as the recommended next effort, and
disable the team.
