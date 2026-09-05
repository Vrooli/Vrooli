# Durability Space — Backup and Restore Durability

## This Space

| | |
|---|---|
| Projection | durability |
| Owner | `data-backup-manager` |
| Denominator confidence | `PARTIAL` — plans and targets are durable; restore evidence is uneven |
| Leg unit | plan / target |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| D1 | Is every declared backup target covered by a plan? | data-backup-manager | NOW | | Plan and target registries are queryable. |
| D2 | Is backup freshness within target? | data-backup-manager | NOW | | Run history supplies freshness evidence. |
| D3 | Has each target passed a restore drill? | data-backup-manager | IN-REACH | | Drill history exists where an operator has run it. |
