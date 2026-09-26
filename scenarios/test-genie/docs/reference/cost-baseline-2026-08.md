# Test Genie cost and storage: before and after

Closing evidence for `test-genie-cost-recovery-and-sqlite-ownership-consolidation`,
2026-08-20. Every "after" value below was measured on this host. Where a value
cannot yet be measured, this document says so rather than reporting the code
change as though it were an outcome — several of the defects this plan fixed
were surfaces that reported success over data that was empty or stale, and
repeating that here would be self-defeating.

## Target Outcome table

| Measure | Baseline | Target | Measured after |
| --- | --- | --- | --- |
| Test Genie rows written outside its own database | 146 runs and growing | 0 | **6,354 rows recovered**; new writes stop at restart (see *Pending a restart*) |
| Scenarios sharing `autoheal.sqlite` | 12 observed | 0 | **0** — five scenarios verified by open file descriptor under a fully inherited supervisor environment |
| Duplicated SQLite bootstrap lines | 2,473 across 62 scenarios | 0 | **0 duplicated**; 48 lines remain across 4 files, each a deliberate remainder (below) |
| Distinct SQLite pragma implementations | 4 | 1 | **1** — a fifth was found in `api-core/databasetest` and folded in |
| `workflow` records carrying CPU and RSS | 0 of 1,733 | every completed record | **code fixed and tested; 0 of 2,394 durable rows yet** — requires a restart |
| Records with `metrics_present = 1` and no reliability enum | 1,733 | 0 | **code fixed; 1,500 historical rows remain** — the column's meaning changed, not the history |
| Phase time reproducing a byte-identical prior failure | 44.5 h (36%) | measurably reduced | **cache extended to failed verdicts**; `runs cost --fleet` now reports repeat-failure cost directly |
| Phase-hours structurally ineligible for caching | 71.1 h (57%) | `unit`/`component-tests` eligible; others named | **`unit` is file-determined; the other 9 name a concrete observation** (`component-tests` stayed observational — reason below) |
| Providers using `BoilerplateDeterminismReason` | 10 | 0 | **0** |
| CPU capacity `used` reported to `Decide` | always 0 | derived from measured host load | **derived from `NormalizedLoad1`**, with swap pressure as a tunable denial |
| RAM claim for `unit` | 48 MB | reflects child process peak | **child peak now included**; the defect measured at **27.9 MB reported vs 274.7 MB actual (9.8×)** |
| Durable record of submit-to-verdict latency | none | `requested_at` on every run | **`requested_at` recorded**, stamped once at admission and never re-stamped |
| Fleet-wide cost projection | hand-written SQL | one command | **`test-genie runs cost --fleet`** with provider attribution, queue latency, repeat-failure cost |
| `runs fleet-health` coverage denominator | 23 of 23 against 121 | true roster | **121 of 121** — see *Premises that were wrong* |

### The four remaining bootstrap functions

None is duplication. `react-component-library` resolves through routed roots so
Test Genie can lease it an isolated data root; `git-control-tower` keeps a
two-line wrapper carrying its typed tuning; `storage-manager`'s analyzer test
contains the pattern deliberately, as its fixture; `browser-automation-studio`
is deny-listed and genuinely unmigrated.

## Verified on a running process

The defect lived in process environment inheritance, so the evidence had to come
from a process, not from a resolved path.
`storage-manager/internal/migration/reconcile.TestScenarioOpensItsOwnDatabaseUnderInheritedEnvironment`
starts a child carrying a supervisor's full environment — `SQLITE_PATH`,
`SQLITE_DB`, `SCENARIO_DATA_DIR`, `VROOLI_SCENARIO`, `SCENARIO_NAME` all naming
the supervisor — has it OPEN the database, and reads `/proc/<pid>/fd`. Run
against the real scenario set it reports:

```
test-genie          OK  opened ~/.vrooli/data/vrooli/test-genie/test-genie.db
plan-manager        OK  opened ~/.vrooli/data/vrooli/plan-manager/plan-manager.db
git-control-tower   OK  opened ~/.vrooli/data/vrooli/git-control-tower/git-control-tower.db
code-facts          OK  opened ~/.vrooli/data/vrooli/code-facts/code-facts.db
storage-manager     OK  opened ~/.vrooli/data/vrooli/storage-manager/storage-manager.db
```

The check runs in CI, and was verified to FAIL when the defect is reintroduced. With `SQLITE_PATH`
honoured again, all three children opened
`supervisor-data/autoheal.sqlite` — the original production failure, reproduced
and caught at the process level.

Note that `storage-manager validate prove-isolation` is a STATIC source scan and
answers a different question: whether the seams are wired. It cannot see this
defect, which is why a running-process check exists beside it.

Each resolves under the class roots rather than a scenario data directory,
because the inherited environment names the *supervisor* — so the identity guard
correctly rejects the assigned data directory and falls through. That is the
intended behaviour: the child gets its own private database rather than a
sibling's.

## Premises that were wrong

Three plan premises described stale observations rather than defects. Recording
them so they are not "fixed" later.

**Run retention is enforced.** `KeepMostRecent` is applied as written. The reason
121 scenarios hold 16 GB across up to 97 run directories each is that nearly
every run is **pinned by a baseline lease**: test-genie holds 63 leases over 78
runs, browser-automation-studio 85 over 97, prompt-manager 75 over 87. Leases are
owner-named with a 30-day expiry, so the footprint is bounded by the lease TTL,
not unbounded. The operating reality worth knowing: **the count cap never
engages, because the effective policy is the lease TTL.**

**The fleet roster is correct.** It names 121 of 121 scenario directories. The
"23 of 23" came from a stale running binary.

**`performance` exclusivity is correct.** It averages 27 s against a 300 s
timeout and its descriptor names a concrete reason — concurrent load changes
Lighthouse measurements. An exclusive measurement window inside the phase would
add machinery for a phase that is already cheap. Decision: keep it.

## Scheduling decisions

**`workflow` joins batches.** It is `provider-serial` with a 23 s average across
2,394 samples, so the deadline guard does not exclude it and provider-serial only
prevents two phases from the same provider batching together.

**`component-tests` timeout raised 600 s → 1200 s.** Its p90 measured
**600.003 s — three milliseconds past its own timeout**, so the distribution was
censored by the ceiling and 11 of 61 samples (18%) were being killed by it. With
`contentionAllowance = 1.5`, `600 × 1.5 ≥ 600` is always true, so the phase was
excluded from *every* batch. This is a measurement decision bounded to one round;
the full reasoning and the re-measurement it obliges are in
`scenarios/react-component-library/docs/reference/component-tests-timeout-decision.md`.

## Provider uptime

Filed to scenario-qa as `knw-1787256564353226589`. Measured over the retained
window: template-manager 2,371 checks / 119 not ready / **max 2,460 s (41
minutes)**; storage-manager 2,060 / 196 (9.5%) / max 240 s; knowledge-observatory
2,423 / 100 / max 347 s. Test Genie now emits a `SLOW provider check` warning
above two minutes so a case like the 41-minute start cannot pass unnoticed, but
the alarm reports the symptom; the cause belongs to those providers.

## Structure

`suite_execution.go` 2,980 → 2,684 lines, with three concerns extracted into
packages carrying their own tests: `phasebatch` (pure), `phaseadmission`
(injected broker and estimator), `phasecacheidentity` (path passed in).

## Pending a restart

Running scenarios still hold pre-change binaries, so Test Genie kept writing into
`autoheal.sqlite` throughout this work — 147 orphaned runs became 152. Three
measures above cannot become durable until the affected scenarios restart:
`workflow` CPU/RSS, the `metrics_present` count, and the end of cross-scenario
writes. After a restart, re-run the reconciliation — it is idempotent — and only
then consider deleting from the source. Deleting first lets the rows return.
