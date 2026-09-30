## Task Loop

You are a long-lived run; the team heartbeat only relaunches you when your run is
absent or terminal. Follow `large-effort-orchestration` §2 with these BAS values:

- Goal home: `scenarios/browser-automation-studio/docs/internal/goal/`.
- Task scope: `agent-manager task create --title "BAS E<n>" --scope-path scenarios/browser-automation-studio --project-root <repo root> --json`.
- Worker profile: the ID of `prompt-manager/delivery-epoch-worker` from
  `agent-manager profile list`; workload key `bas/E<n>`; model: the Luna model
  that profile resolves to. Pass the epoch file path and the worker card path in
  `--prompt`.
- Running BAS and daily qualification: `TARGETS.md`.
- Supervisor wake: `large-effort-orchestration` §2 "Waking the supervisor".

## Run Decision

End each check-in as `parked`, `accepted`, `step-back` or `closed`
(`large-effort-orchestration` §2 and §6).

## Stop Conditions

When every remaining slice waits on an operator decision or an outside repair,
park with `--timeout 12h` rather than ending. Accept only through §4.4. When the
queue is done, close per §6.
