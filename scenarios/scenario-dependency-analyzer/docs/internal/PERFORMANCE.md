# Performance

## Purpose Of This Document

Track performance budgets and known constraints.

## Budgets

UI pages should meet configured Lighthouse thresholds and keep graph interactions responsive for normal local scenario counts.

Dependency freshness targets are a warm `freshness --touched` run below 1 s,
a cold-cache run below 6 s, and `deps reconcile --all --json` below 10 s.
The repeatable measurement command is:

```bash
bash scenarios/scenario-dependency-analyzer/test/perf/freshness-bench.sh \
  --label after scenarios/scenario-dependency-analyzer/coverage/perf
```

## Current Measurements

Scenario test artifacts under `coverage/logs` hold the latest Lighthouse measurements.

## Known Constraints

D3 force layouts can become expensive for large graphs.

Bundle-manifest requests use a request-local graph that expands each scenario
owner once. Shared dependency edges retain their declarations; embedding can
follow a shared owner first encountered through a discovery-only branch. The
manifest route reads current inputs and does not build, load, clone, or persist
the full deployment report. Request cancellation prevents returning a partial
manifest. Full deployment reports and scenario target-DAG exports also use a
request-local owner expansion cache. They retain shared dependency declarations
without copying every root-to-leaf path; tier totals count an active shared owner
once. Stored deployment-report reads are capped at 32 MiB, and oversized legacy
reports require recomputation before use.

The focused regression `TestFreshBundleManifestBoundsRepeatedFanIn` covers an
18-level graph with two shared owners per level. It retains 36 owners and 70
edges, rather than expanding every root-to-leaf path. Companion tests cover
discovery-before-embedding, UI inclusion, version/feature fidelity, cycles, and
request cancellation.

`TestBuildReportBoundsRepeatedFanIn` covers 16 owners with 30 retained edges and
about 12 KiB of report output. `BuildReportWithOptions` initializes the shared
cache for report and legacy scenario-DAG callers; `BuildTargetDAGWithOptions`
initializes it for target-aware scenario exports. These bounds address the
deployment-report expansion that previously caused multi-gigabyte retained
reports and API memory growth.

These graph and report limits do not impose a process memory ceiling or bound
the number of concurrent HTTP requests. The synchronous report/DAG HTTP adapters
currently call reporter methods without a request context; cancellation support
in `BuildReportContext` applies to its server-owned analysis callers. Host
memory ceilings and recovery reservations belong to the control plane, and
require separate kernel-facing inspection and pressure qualification.

## Regression Procedure

Run the full scenario test and inspect performance artifacts when thresholds fail.

## Cross-References

- `../operations/OBSERVABILITY.md`
- `../guides/troubleshooting.md`
