# Availability Space — Check Availability

## This Space

| | |
|---|---|
| Projection | availability |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — per-check history is durable; cold-start latency remains open |
| Leg unit | check |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| A1 | What is the per-check uptime trend? | vrooli-autoheal | NOW | | Actions service exposes distinct per-check trends. |
| A2 | How often does each check transition? | vrooli-autoheal | NOW | | Transition history is queryable by window. |
| A3 | What is cold-start latency for each element? | vrooli-autoheal | IN-REACH | 2026-08-22 | Typed readiness evidence now reports the first healthy probe after autoheal process start; the operator setpoint has not yet ratified a latency threshold. |
