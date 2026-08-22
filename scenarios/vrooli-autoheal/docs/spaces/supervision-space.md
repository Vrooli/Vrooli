# Supervision Space — Reliability Supervision

## This Space

| | |
|---|---|
| Projection | supervision |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — registry coverage is live, while the derived plant closure is external |
| Leg unit | check |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| S1 | Does every derived core-set scenario have a registered check? | vrooli-autoheal | NOW | | `check reconcile` computes both directions at request time. |
| S2 | Are registered checks free of ghost targets? | vrooli-autoheal | NOW | | Reconcile returns explicit ghost check IDs. |
| S3 | Is the alarm channel free of saturation? | vrooli-autoheal | NOW | | Transition-backed saturation read. |
| S4 | Are deliberate stops represented by expiring shelves? | vrooli-autoheal | NOW | | Shelf state is durable and expiry is mandatory. |
