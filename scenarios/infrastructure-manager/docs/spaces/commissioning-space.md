# Commissioning Space — Host Bring-Up

## This Space

| | |
|---|---|
| Projection | commissioning |
| Owner | `control-plane` — `vrooli setup`; no scenario holds this space |
| Denominator confidence | `SKETCH` — the control plane owns setup; revisit when `vrooli setup` exposes a first-class space service |
| Leg unit | host bring-up |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| CM1 | Is setup end-to-end time measured for a clean host? | control plane | MISSING | 2026-08-20 | No durable setup sensor exists. |
| CM2 | Are required host capabilities verified after setup? | control plane | NOW | 2026-08-22 | `vrooli host safeguard list --json` is consumed as a typed control-plane observation by infrastructure-manager condition and portability reads. The value is a live state distribution, not an assertion that every safeguard is satisfied. |
