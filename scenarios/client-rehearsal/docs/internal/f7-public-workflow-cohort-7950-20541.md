# F7 public workflow cohort qualification: 7950 and 20541

Date: 2026-11-12. Scope: two new public-pool candidates for the `client-rehearsal` F7 160–280h build. No commission-folder material was read. The freelance `MISSION.md`, `CATEGORIES.md`, and F7 checkpoint/matrix/evidence files were not present in this checkout; source terms/rate guidance therefore comes only from the repository attribution note, not the missing F7-specific notes.

## Outcomes

| Workflow | Candidate | Sanitized full graph | Import | Generic simulator | Matrix credit |
|---|---|---:|---|---|---:|
| 7950 | Discord/GitHub/Gemini blog assistant, by Praneel S | 10 nodes, 9 edges | Rejected: `n8n-nodes-discord-trigger.discordTrigger` is not an allowlisted executable type | Not reached; import must succeed before an in-process simulator can consume the graph | 0 |
| 20541 | Idempotent lead intake, by Oneclick AI Squad | 15 executable nodes, 15 edges; five sticky notes removed | Rejected: Postgres `CREATE TABLE` / `CREATE INDEX` is DDL beyond the simulator's bounded DML contract | Not reached; import must succeed before an in-process simulator can consume the graph | 0 |

No graph was rewritten to fit the importer and no per-template adapter was added. The full source exports remain transient; only sanitized fixtures are stored. Author, official page/API URL, raw response byte count/hash, sanitized hash, and sanitization notes are recorded in `api/internal/rehearsals/testdata/public-workflows/ATTRIBUTION.md`.

## Terms and acquisition bounds

The attribution source note records n8n self-serve terms read 2026-11-10. It says public Community Materials may be used and adapted in a user's project, and explicitly calls applicability to the public workflow directory an inference. F7-specific current rate rules were unavailable here. Each workflow had one successful official API GET capped at 512 KiB (25,325 bytes and 59,409 bytes); source bodies stayed in process memory and were discarded after sanitization. No request retries followed failures. No account, credential, messaging, contacts, live service, n8n runtime, model, container, GPU, or download was used.

## Verification

Command: `go test ./internal/rehearsals -run 'TestPublic(7950|20541)' -count=1 -v` from `scenarios/client-rehearsal/api`.

Result: PASS as qualification-attempt tests. Both tests retain exact importer failures and explicitly log no run credit. No generic simulator profile/run was applicable because both complete sanitized graphs fail import.

Measured focused test process: 0.49 s user CPU + 0.20 s system CPU (0.69 CPU s total); 0.46 s elapsed; 216,920 KiB peak RSS; 0 swaps. GPU/model/container usage: none. Added fixture/test bytes: 35,905 bytes after the synthetic-host rewrite. Codex token/use accounting is unavailable to this task and is not claimed.

## Central gaps / remaining

- Add safe generic support for the public Discord community trigger and Discord/GitHub/Gemini node shapes only through the shared interpreter owner; no local changes were made to `simulator.go` or `simulator_test.go`.
- Decide whether a reusable, bounded Postgres DDL representation can be supported without executing SQL, or retain this candidate as excluded. Do not permit live or synthetic database DDL execution to claim a run.
- After generic import support exists, add synthetic profiles and require a full in-process run before any matrix credit.
- Reconcile these two zero-credit outcomes into the authoritative F7 matrix/checkpoint when that source becomes available.
