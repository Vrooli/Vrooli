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
| A3 | What is cold-start latency for each element? | vrooli-autoheal | MISSING | 2026-08-20 | No durable start-latency sensor yet. |
