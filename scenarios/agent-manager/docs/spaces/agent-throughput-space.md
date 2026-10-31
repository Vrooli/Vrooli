# Agent Throughput Space — Agent Run Throughput

## This Space

| | |
|---|---|
| Projection | agent-throughput |
| Owner | `agent-manager` |
| Denominator confidence | `PARTIAL` — terminal run ledger is durable; run-class completeness is still being audited |
| Leg unit | run class |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| AG1 | Do agent runs spawn successfully? | agent-manager | NOW | | Run ledger and measures expose terminal outcomes. |
| AG2 | Which error patterns recur by run class? | agent-manager | NOW | | Error-patterns measure is the named sensor. |
| AG3 | Is every run class represented in throughput measures? | agent-manager | IN-REACH | | Denominator completeness remains under audit. |
