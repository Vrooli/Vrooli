# Goal — Native-feel desktop packaging

**Destination.** Generated desktop apps start and idle like native apps, and
the desktop ramp picks the fastest proven packaging per scenario on its own:
- several packaging strategies are selectable;
- performance-health measures each one with one metric contract;
- an experiment report compares them on the reference scenarios;
- auto-selection from code-facts stack facts plus evidence is live.
Targets and bands are in TARGETS.md. Design (the source of truth):
`docs/architecture/packaging-strategies.md`.

**Stakes.** Users now expect native speed and treat Electron apps as bloated.
Today cold start is about 7 s, and about 6.2 s of that is our runtime, not
Electron.

**Constraints.**
- Desktop only. Mobile is the next effort on the same contracts: shared types
  in `packages/delivery-ramp-go` must not be desktop-specific (design §7).
- Keep every working strategy selectable. Keep fixes that help any strategy,
  even if it loses.
- Selection reads facts. Go-only strategies must leave non-Go scenarios
  eligible for others.
- One web UI source per scenario. No per-target native UI.
- Orchestrated epochs per `large-effort-orchestration`. No Swarm Manager, no
  Plan Manager, no git (the operator commits). Privileged installs only through
  `vrooli setup`. Dependencies only through Scenario Dependency Analyzer.
- BAS has its own delivery team. Request BAS changes through its FEEDBACK.md.
- Operator pre-approved Rust (Tauri) and Wails v3 beta (FEEDBACK DPF-FB-001).

**Gates.** Acceptance per `large-effort-orchestration` §4.4 and the worker
card. Feature epochs follow its growth rule.

**Links.** [QUEUE.md](QUEUE.md) · [TARGETS.md](TARGETS.md) ·
[FEEDBACK.md](FEEDBACK.md) · [WORKAROUNDS.md](WORKAROUNDS.md) ·
[INSTALLS.md](INSTALLS.md) · [epochs/](epochs/) · Ramp docs:
[ARCHITECTURE](../../concepts/ARCHITECTURE.md),
[live-desktop-api](../../reference/live-desktop-api.md),
[PROBLEMS](../PROBLEMS.md) · Measurement owner:
`scenarios/performance-health/PRD.md`.
