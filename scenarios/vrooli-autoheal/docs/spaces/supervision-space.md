# Supervision Space — Reliability Supervision

## This Space

| | |
|---|---|
| Projection | supervision |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `FULL` — the live registry is reconciled from the canonical `vrooli supervision-set` closure plus explicit additive overrides |
| Leg unit | check |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| S1 | Does every computed supervision-set member have a registered check? | vrooli-autoheal | NOW | | Startup and a 30-second source check reconcile scenarios and resources; every canonical result carries intent and its complete attribution chain. |
| S2 | Are registered checks free of ghost targets? | vrooli-autoheal | NOW | | Reconcile returns explicit ghost check IDs. |
| S3 | Is the alarm channel free of saturation? | vrooli-autoheal | NOW | | Transition-backed saturation read. |
| S4 | Are deliberate stops represented by expiring shelves? | vrooli-autoheal | NOW | | Shelf state is durable and expiry is mandatory. |
