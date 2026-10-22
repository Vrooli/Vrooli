# Worker brief — actor-owned Goals and Milestones

## Authority and objective

Continue the approved Personal Planner I18 E1 outcome in `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`, under exact parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`. Implement one bounded identity outcome: newly created Goals and their Milestones are private to the verified actor across every Goals repository read/write method, while ownerless historical Goal and Milestone rows remain untouched and inaccessible. This proceeds independently of the pending human legacy-record mapping decision. Do not answer that decision or assign existing rows.

## Measured source boundary

The current source was read directly. The E1 measured net growth before this slice is +1,827 runtime lines and +2,488 test lines, against +2,100 runtime hard ceiling and +2,031 stop boundary. The source anchor is `gcta-8128c491b7bb`, HEAD `34515f0f501d511f65aa06c8e323620bf6d05891`, scope `scenarios/personal-planner/api/internal/goals/**`.

The anchor captured these current identities and sizes, including existing accepted-but-uncommitted measurable-goal and target-history work:

| Path | Lines | SHA-256 |
|---|---:|---|
| `api/internal/goals/sqlite.go` | 329 | `c02b7d0b7e69258080fcb0b91fcd2441f732a081d011316a534f30240074d56e` |
| `api/internal/goals/schema.sql` | 63 | `3b3276d92269cc89eed6dd43c1858f43beaddf235f6187af48a08eceaabce460` |
| `api/internal/goals/sqlite_test.go` | 177 | `941b1db91fd0cfeb2cf37b203781df3ce9cc0ec64725ce9080ac76daf4000eb2` |
| `api/internal/goals/service.go` | 198 | `f739e8ffac486d7c8bc50aff3775cc31232e47ae254bb0f6913888370a2c0d9d` |
| `api/internal/goals/service_test.go` | 134 | `5de324c1c332e554da02ff2d08a3ff902b482ae28f9001e3d65601ccda687a4d` |
| `api/internal/goals/repository.go` | 16 | `df3b04b266ec0a34f59eee00056be2f7aa12cb776ff0454985b038208d354ba2` |
| `api/internal/goals/types.go` | 74 | `5fac2360d09e8005124e90cb884187f2782b2f6445e848b33a74a0a251e1c285` |

The current Goals repository has List, Create, target-date history/update, progress, milestone list/create/status, prerequisite and work-link methods. It has no verified-principal lookup. Workspace ownership already uses `identity.PrincipalFromContext`, requires `Verified`, and scopes owner-specific rows; calendar events use an explicit owner join table. E1 preserves ownerless legacy rows without assignment.

## Exact allowed paths and changes

Only these seven anchor-scoped files may change:

- `scenarios/personal-planner/api/internal/goals/{sqlite.go,schema.sql,sqlite_test.go,service.go,service_test.go,repository.go,types.go}`

Use the existing verified identity context and owner subject. Persist ownership for new goals using a compatible additive owner representation that requires no owner guess or backfill. Scope all Goal and Milestone reads and mutations, including histories, progress, prerequisites and child lookup, to that owner. Missing/unverified principals fail closed. Preserve current measured-goal data, target-date history, optimistic revision behavior and milestone semantics. Do not alter work items or map/link existing ownerless records. If an owner check would require the pending legacy mapping decision or a broader Work ownership contract, retain the exact boundary and stop before widening scope.

Do not modify handlers, Connect/Proto, CLI/UI, other domains, dependencies, unrelated generated outputs, live runtime data or product database. Tests may use only a fresh temporary SQLite database. No product-data migration, import, assignment, restoration or broad Test Genie run.

## Acceptance criteria

1. A missing or unverified principal cannot create, list, read, update or mutate Goal/Milestone data.
2. Actor A can create and read/update their new goal and milestones; Actor B cannot list them or access them by guessed Goal/Milestone ID, and B's own new records remain separate.
3. Ownerless preexisting goal/milestone rows remain byte/row-preserved, absent from each actor's list and inaccessible by ID; no owner or inferred assignment is added.
4. Target-date history, measures, progress restrictions, revisions, prerequisite checks and existing unrelated work-link semantics continue to pass focused package tests.
5. Retain exact before/after file hashes, focused test command/output, measured net line delta and a concise handoff. No claim of whole UI/CLI/Connect actor parity or E1/Q1 acceptance.

Run only focused `go test ./internal/goals -count=1` from the Planner API module if its owner-managed environment is ready. If a generated-artifact freshness error appears, follow the repository generated-artifact owner rule before diagnosing or rerunning. Preserve every failed command/result.

## Growth, gate and stop

- Estimate: +95 runtime / +130 test lines.
- Expected maximum for this brief: +130 runtime lines; stop before E1 cumulative runtime reaches +2,031. The hard ceiling is +2,100; do not cross it.
- Slice gate: acceptance criteria above plus focused package test PASS and exact source attribution.
- E1 gates remain whole: this bounded slice cannot satisfy the original calendar journey, visual comparison, seven-phase suite, original independent criterion judgment, PP-UX/domain acceptance, Connect native-event proof or per-case LeaseStats.
- If the existing owner/mapping state, current source identities, fixture isolation or budget differs from this brief, do not write; retain evidence and return the mismatch.

## Required method

Read `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`. Preserve the exact source anchor, accepted prior changes and all unrelated user/agent data. Report actual work and evidence; task completion alone is not acceptance.
