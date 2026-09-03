# One Asset, One Verdict

## Purpose

The React Component Library uses one governed edit loop for an asset. The
loop builds the asset's derived projections, validates the asset and its
dependency closure, reuses or runs component evidence, checks package
resolution, and returns one aggregate verdict. The authoritative command is:

```bash
react-component-library asset check <asset-id>
```

This document records the architecture and the evidence for the
`react-component-library-one-asset-one-verdict-unwedgeable` plan. The central
workflow is implemented. The plan-wide completion gates remain open where the
record below says so.

## Why the setup has this shape

The defects addressed by the plan share one shape: a concept crosses a
boundary from authored source to generated projections, readers, package
outputs, and consumers, and each side had its own partial implementation.
The durable fix is to assign one owner at each boundary and make the other
side consume that owner.

The six main collapses are:

1. Heavy catalog work runs in cancellable jobs with separate database access,
   so a client disconnect cannot starve health or workbench requests.
2. Gate scope is resolved as an asset closure, and gates declare what they
   read and report.
3. `librarywalk` is the shared API tree reader.
4. `BuildRevisionIndex` is the shared authority for source and dependency
   staleness.
5. `libspec` owns the import grammar in Go and its JavaScript binding is
   generated.
6. Catalog build stages own derived files, while the CLI composes them into
   one asset verdict.

## Reads and reports

Scope has two independent contracts. A gate may read a closure or the whole
corpus when its rule requires corpus context, but its reported findings must
belong to the requested asset or its dependency closure. The runner filters
and records both dimensions so a passing result cannot hide an out-of-scope
finding or an unbounded scan.

## Revision authority and jobs

The revision index folds an asset's declaration, manifest, source, and live
dependency revisions. Gate evidence, component-test freshness and identity,
adoption preflight, and package staleness use that folded revision. Heavy
matrix work uses its own bounded job admission and database connection; the
serving pool remains available to health and workbench RPCs.

## Evidence snapshot

Evidence is retained under:

`/home/matthalloran8/.vrooli/plan-artifacts/react-component-library-one-asset-one-verdict/`

| Check | Result | Evidence |
| --- | --- | --- |
| Button asset verdict | Pass | `asset check controls.button --run-tests --json` resolved the latest catalog id to `react-component-library:Button@2.2.9` and returned `PUBLISHABLE`, zero findings, and fresh report `ctr_c7705ca85366a928` (`reused=false`). |
| Reused validation latency | Pass | Three cached invocations completed in approximately 0.22 seconds each. |
| Fresh Button component test | Pass | The current catalog-id invocation passed all Button stages and stories; the live asset check refreshed evidence through report `ctr_c7705ca85366a928`. |
| Graph reconciliation | Pass | `catalog gates graph-reconciled --asset-id controls.button` returned zero findings. |
| Corpus instrument | Pass with open measurements | The corpus report emits the invariant set and the tracked machinery ratio, but the latest governed suite still reports inherited maturity findings; the full corpus result is not a plan-completion oracle. |
| Targeted Go validation | Pass | `GOWORK=off go test ./internal/components ./internal/preview ./internal/catalogcoverage ./handlers/componenttests -count=1` passes; catalog check reports zero broken version imports. |
| Full Test Genie suite | Fail with recorded inherited debt | Latest run `20260903-093914-e97069ce`: 16 passed, 11 failed, 1 skipped. BAS/UI-health repairs are present; failures remain across dependency, docs, unit, storage, workflow, business, experience, tidiness, security, measures, and component-test phases. |
| Plan Manager validation | Unknown | Baseline generation `g4` is ready; operation `646bee70-3037-4bbe-b453-5924a6a6f203` is `UNKNOWN` because GCT classifies the readable member run as `not-comparable`. |

## Current DoD status

| Area | Status | Note |
| --- | --- | --- |
| One command and one verdict | Pass | Button acceptance is proven end to end. |
| Durable Test tab | Pass | Component detail now exposes the existing version-pinned `ComponentTestPanel` at `tab=tests`; deep-linked report URLs and the route contract are covered by focused tests. |
| API cancellation and health isolation | Pass | The 60-second proof recorded 60/60 HTTP 200 health samples under 1s and one `matrix canceled` log entry. |
| Scoped gates and graph ownership | Pass for Button | Scoped all-gate completed in 8.36s with zero findings and 159 inspected files; graph attribution is covered by targeted tests. |
| Revision-aware reuse | Pass for Button | Unchanged reports reuse; changed dependency revisions invalidate evidence. |
| Grammar, walker, and generated stages | Partial | Structural checks cover the single registry scope contract, no numbered gate files, and the single API walker; current catalog projections report 255 assets, 298 versioned exports, and zero broken imports. Revision hashing ignores only volatile lock timestamps while retaining semantic dependency choices. |
| Agent-facing surface | Partial | `api/cmd` currently contains two merged generator directories and the CLI has 28 manifest commands; broad cleanup targets remain. |
| Workbench | Partial | Runtime error/timeout handling, released Popover adoption, the component Test tab, compact data-driven vision filters, first-consumer package scanning, and a controller/view split for ComponentTestPanel landed; fresh captures now exist, but plan-wide line ceilings remain open. |
| Net-negative machinery | Open | Current shipped-source measurements remain above the plan ceilings: the retained count implementation reports I27 = 1.0092 against 0.92. Test-only mocks/fixtures and generated selector/string projections are excluded; the Go and ratio ceilings remain open. |
| Instrument honesty | Partial | Graph, fallback-parity, asset, and package paths are honest; corpus and full-suite evidence still contain inherited failures. |

| Fresh-agent edit-loop | Pass with environment note | A fresh Agent Manager run, given only the README/task framing, completed the governed Button draft, focused closure test, publish to `Button@2.2.9`, and immutable validation in 15 shell attempts and about 4m28s, with no files outside the Button workflow. The host's protected sandbox still fails `bwrap: loopback: Failed RTM_NEWADDR`, so the successful proof used explicit in-place execution; two inherited Pressable evidence failures remain recorded. The post-publish asset check was refreshed independently with report `ctr_c7705ca85366a928`. See `phase-17/fresh-agent-after.md`. |

## Follow-ups

The remaining follow-ups are deliberately visible rather than hidden behind
allowlists or a green summary: repair the remaining UI and maturity findings;
complete the plan-wide machinery reductions; repeat the fresh-agent edit-loop
proof in a functioning edit sandbox; finish fresh workbench captures; and obtain a comparable Plan Manager
validation result. Proto alias cleanup, support-layout normalisation, the
control-plane live probe decision, and the remaining adoption work belong to
their owning follow-up workflows.

The corpus report now runs through the promoted CLI verb and the in-process
catalog coverage implementation; the obsolete `api/cmd/corpus-report`
directory is gone. The temporary complexity ledger was retired after the final
measurement checkpoint: `counts.go` remains as the I27 measurement owner, while
the final machinery ceilings remain open and are recorded honestly above.

Post-retirement checkpoint (2026-09-03): `catalog corpus-report --json`
completed in the live CLI with exit 0, emitted all 25 invariants, and reported
I27 = 1.0091544493692466, I22 = 7, and I8 = 50 seconds. The API's complete Go
package suite passes after the retirement, and `git diff --check` is clean.

Fresh-release checkpoint (2026-09-03): the fresh-agent-published
`Button@2.2.9` initially caused the expected stale-evidence verdict. Running
the prescribed `asset check controls.button --run-tests --json` recovery path
returned `PUBLISHABLE` with zero findings and report
`ctr_c7705ca85366a928`; catalog build/check reported zero broken version
imports.

Retention checkpoint (2026-09-03): the governed `versions reap --json` preview
found zero additional eligible releases (255 latest, 47 referenced, 19 already
retired), so the remaining I5 gap is not removable through currently safe
retention candidates.

The refreshed normal corpus report (2026-09-03) exited 0 in 49.423 seconds and
measured I8 = 21 seconds and I9 = 8 seconds. I18 = 12, I22 = 7, and I27 =
1.0103 remain open.

The revision resolver also reduced the live all-gates matrix to 20.320 seconds
(15,970 inspected files, 801 findings). This clears the plan's 30-second I8
timing target, although inherited runner errors and findings remain open.

Evidence rerun checkpoint (2026-09-03): the governed pinned rerun for
`IntegrationCard@0.1.4` passed as report `ctr_338bfcfc89a520ba`. The matching
`AuthClient@1.0.0` rerun failed during closure revision computation because a
persisted closure references non-materialized `IntegrationCard@1.0.0`, while
AuthClient's checked-in dependency lock is empty. This remains an explicit
materialization defect, not a freshness bypass.

The post-rerun live all-gates matrix completed in approximately 20.8 seconds
with 15,953 inspected files and 801 findings. It reports five runner errors:
one AuthClient freshness error, three missing historical IntegrationCard
sources, and the current IntegrationCard token-ramp finding.

The supplemental boundary correction was then applied to evidence freshness,
token-ramp, and released-version immutability. Focused gate tests passed, and
the rebuilt live all-gates matrix completed with 15,939 inspected files, 801
findings, and zero runner errors. The remaining findings are ordinary catalog
observations, not runner failures.

The in-process corpus matrix probe now opens the same read-only live database
used by DB-backed gates. After rebuilding and reinstalling the CLI through the
governed setup path, `catalog corpus-report --json` exited 0 with I7 = 0,
I8 = 22 seconds, and I9 = 8 seconds. I18 = 12, I22 = 7, and I27 = 1.0114
remain above target.

Final Phase 13 measurement checkpoint (2026-09-03): the corpus report exited 0
with no `failed_measurement` invariants. I7 = 0, I8 = 21 seconds, I9 = 8
seconds, I18 = 1 (machine 51, manual 50), and I22 = 1. The detailed shape
census still reports historical, supplemental, and support layouts separately;
I22 measures distinct shapes among latest non-supplemental catalog versions.
I27 = 1.0047 remains an honest non-blocking machinery-ceiling follow-up.

## Cross-references

- [Asset update flow](../guides/asset-update-flow.md)
- [Asset derivation](ASSET-DERIVATION.md)
- [Architecture](ARCHITECTURE.md)
- Plan artifacts: `/home/matthalloran8/.vrooli/plan-artifacts/react-component-library-one-asset-one-verdict/`
