# Validation Cost Space — Validation Reliability and Cost

## This Space

| | |
|---|---|
| Projection | validation-cost |
| Owner | `test-genie` |
| Denominator confidence | `PARTIAL` — phase catalog is authoritative; cost calibration is moving |
| Leg unit | phase |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| VC1 | Does each declared phase produce a terminal result? | test-genie | NOW | | Phase catalog and run ledger provide the denominator. |
| VC2 | What is the reliable cost of a phase? | test-genie | IN-REACH | | Cost query exists; calibration freshness remains bounded. |
| VC3 | What is the cache-hit rate by phase? | test-genie | IN-REACH | | Run evidence exists for populated phases. |
