# Attribution Space — Saturation Attribution

## This Space

| | |
|---|---|
| Projection | attribution |
| Owner | `system-monitor` |
| Denominator confidence | `SKETCH` — process sampling is live but investigation closure is not complete |
| Leg unit | saturation window |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| AT1 | Can a sustained saturation window be attributed to a process owner? | system-monitor | NOW | | Process timeline is the named sensor. |
| AT2 | Can the owning scenario or run be identified? | system-monitor | IN-REACH | | Attribution is available for some process classes. |
| AT3 | Is every investigation closed with evidence? | system-monitor | MISSING | 2026-08-20 | Closure measure is not yet universal. |
| AT4 | Which governed root is filling the disk fastest right now? | system-monitor | IN-REACH | 2026-09-02 | Sensor exists: writer samples carry root, rate, and partial-measure state. No setpoint bar is ratified, so nothing grades the readings yet. |
| AT5 | Is a runaway writer distinguishable from slow growth within one sensor window? | system-monitor | IN-REACH | 2026-09-02 | Sensor exists: the bounded fill-rate estimator supplies the comparison. No setpoint bar is ratified, so nothing grades the readings yet. |
