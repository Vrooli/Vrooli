# Headroom Space — Storage Capacity Headroom

## This Space

| | |
|---|---|
| Projection | headroom |
| Owner | `storage-manager` |
| Denominator confidence | `FULL` for a complete storage-manager feed; `PARTIAL` when census attribution or freshness is degraded |
| Leg unit | device / ceiling |

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| H1 | Is every load-bearing device in the storage census? | storage-manager | NOW | | Read from storage-manager's device inventory. |
| H2 | What is the growth slope for each device? | storage-manager | IN-REACH | | Historical slope is available for selected stores. |
| H3 | Does each owner have a declared ceiling? | storage-manager | NOW | 2026-09-02 | Read from the typed declared-ceiling coverage feed. |
| H4 | Do recovery runs reach their free-space target? | storage-manager | NOW | 2026-09-02 | Reports recent terminal-run efficacy and stop reasons. |
| H5 | Do enforced budgets agree with declared bytes? | storage-manager | NOW | 2026-09-02 | Reports measured bytes covered by successful enforcement. |
| H6 | Are governed roots producing hot-writer pressure? | storage-manager | NOW | 2026-09-02 | Reports the count of hot roots in the latest writer window. |
