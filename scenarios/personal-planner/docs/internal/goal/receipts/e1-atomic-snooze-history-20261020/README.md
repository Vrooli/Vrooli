# E1 atomic snooze history receipt

The exact admitted source bytes were copied to `baseline-original/` before editing and verified against the supplied SHA-256 identities. The snapshot manifest records both identities and notes that the precise wall-clock capture time was not retained.

| Path | Baseline SHA-256 | Baseline lines | Current SHA-256 | Current lines | Delta |
|---|---|---:|---|---:|---:|
| `api/internal/work/sqlite.go` | `00bfbcb301d842f4d70954bb1351453426f5d451cd46e8ca2d3c8400f1ac0e35` | 189 | `653c04a6a8153e071180bc4cbd5d6a24e321b0300182a1ca1e364f78e3928bff` | 199 | +10 runtime |
| `api/internal/work/sqlite_test.go` | `282d80dacbff22b2d75180e45910d6abc7ba302b79ef4d3e7919f0703675c9e6` | 246 | `869ce6ecdfaec523f741fae580e8d5e90f26bc1493fc2b1707128bfb835b432f` | 318 | +72 test |

The test delta exceeds the +30 estimate; it includes explicit failure atomicity, successful exact-one matching row, and completed-work rejection assertions. E1 totals after the slice are +1,418 runtime and +1,813 test lines; runtime remains below the +2,100 ceiling.

`go-test.log` retains the exact output of `cd scenarios/personal-planner/api && go test -count=1 ./internal/work` (PASS). `diff-check.log` retains the exact output of scoped `git diff --check` (PASS). `epoch-check.txt` retains the native `agent-manager effort epoch-check` result (exit 0, no step-back). The change uses deterministic in-memory SQLite. No schema, UI, provider, product data, or other source files were edited for this slice.

This receipt does not accept E1 or the slice and does not change any review disposition or acceptance gate. The historical fake-clock mismatch remains only a qualified fixture observation, not a production failure claim.

The retained epoch-check also includes the inherited wall-clock review prompt and legacy untyped-gate warning. It was run without `--acceptance`; it does not accept E1.
