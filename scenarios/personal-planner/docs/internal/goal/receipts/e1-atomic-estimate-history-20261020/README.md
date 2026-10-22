# E1 atomic estimate history receipt

Captured 2026-10-20T15:29:59Z by worker run `06e51651-bd47-4f82-b8ad-8740be0020b7` (Agent Manager identity verified; task `f287240d-0ba8-483f-9744-854dac8c3d1d`). The original admitted files were copied byte-for-byte before the first write into `baseline-original/`.

| Admitted path | Baseline SHA-256 | Baseline lines | Current SHA-256 | Current lines | Net from saved baseline |
|---|---|---:|---|---:|---:|
| `scenarios/personal-planner/api/internal/work/sqlite.go` | `dbb4550df54d10d7845ee394991f8fc08ff4ed740282e7b89c123779eb9f3cf0` | 180 | see `current-identities.txt` | 189 | +9 |
| `scenarios/personal-planner/api/internal/work/sqlite_test.go` | `2b4c0895b903b6d0645c8282211202abb451365e02ab14a6cc533bb819097360` | 185 | see `current-identities.txt` | 246 | +61 |

Baseline capture attribution/time are above. Current hashes and line counts are retained separately. `go-test.txt` and `diff-check.txt` contain the exact required focused validation outputs. The focused tests use only deterministic in-memory SQLite, including an injected history insert trigger failure.
