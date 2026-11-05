# Recovery Space — Healing Outcomes

## This Space

| | |
|---|---|
| Projection | recovery |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — five cells are declared; action outcomes are durable, remediation reach is registry-backed, delivery reach reports an unreadable projection until configured, and episode grouping is not yet persisted |
| Leg unit | heal episode |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| R1 | What is the success rate of recovery actions? | vrooli-autoheal | NOW | | Healing service exposes typed outcomes. |
| R2 | Which checks enter repeated heal loops? | vrooli-autoheal | IN-REACH | | Tracker state exists; durable episode grouping remains to be completed. |
| R3 | What is end-to-end heal episode duration? | vrooli-autoheal | IN-REACH | 2026-08-22 | Typed readiness evidence joins a failing probe to the first healthy probe after a recovery action; the operator setpoint has not yet ratified an episode-duration threshold. |
| R4 | What fraction of critical findings have a remediation path? | vrooli-autoheal | IN-REACH | 2026-08-25 | `coverage-remediation-reach` counts critical registry results with at least one registered action and reports missing check IDs. |
| R5 | What fraction of critical findings reach a human? | notification-hub | IN-REACH | 2026-08-25 | `coverage-delivery-reach` joins incident IDs to delivery-attempt receipts; the current host reports the cross-scenario projection as unreadable until configured. |
