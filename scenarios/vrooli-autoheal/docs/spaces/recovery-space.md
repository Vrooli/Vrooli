# Recovery Space — Healing Outcomes

## This Space

| | |
|---|---|
| Projection | recovery |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — action outcomes are durable; episode grouping is not yet persisted |
| Leg unit | heal episode |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| R1 | What is the success rate of recovery actions? | vrooli-autoheal | NOW | | Healing service exposes typed outcomes. |
| R2 | Which checks enter repeated heal loops? | vrooli-autoheal | IN-REACH | | Tracker state exists; durable episode grouping remains to be completed. |
| R3 | What is end-to-end heal episode duration? | vrooli-autoheal | MISSING | 2026-08-20 | Action duration is not episode duration. |
