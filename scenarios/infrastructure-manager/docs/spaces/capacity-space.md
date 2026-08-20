# Capacity Space — Control-Plane Capacity Claims

## This Space

| | |
|---|---|
| Projection | capacity |
| Owner | control plane (`vrooli capacity`) |
| Denominator confidence | `SKETCH` — instrument-held pending a control-plane space owner; revisit when capacity gains a typed space service |
| Leg unit | claim |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| C1 | Does every load-bearing process hold a capacity claim? | control plane | IN-REACH | | `vrooli capacity reconcile` is the current fenced read. |
| C2 | Are reserves honest against observed peaks? | control plane | IN-REACH | | `vrooli capacity recommend` provides comparison evidence. |
| C3 | Is enforcement posture visible for every claim? | control plane | MISSING | 2026-08-20 | Typed capacity space owner is not yet assigned. |
