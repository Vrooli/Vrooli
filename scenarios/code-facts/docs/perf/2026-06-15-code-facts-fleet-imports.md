# code-facts `DescribeFleetImports` — fleet import scan performance

**Date:** 2026-06-15
**Scope:** `code-facts` `DescribeFleetImports` over the full fleet (~111–116 scenarios).
**Plan:** `fleet-interface-graph-performance-ttd-sda-wiring-proto-drift-cleanup` (Phase 1 / §98 audit).

## Summary

| Metric | Before | After |
|---|---|---|
| Fleet import scan, **cold** | **≈ 27s** | **< deadline** (folds into SDA's 17.1s cold full-graph build, run in parallel with proto-health) |
| Fleet import scan, **warm** | **> 90s** (cache was an anti-optimization — warm *slower* than cold) | **sub-second** (SDA warm full-graph build = 0.21s, which includes this call) |

The warm-slower-than-cold anomaly is gone.

## Method

`DescribeFleetImports` is an internal RPC consumed by `scenario-dependency-analyzer` to build the interface graph; it is not exposed as a standalone CLI verb. After-numbers are measured **transitively** through SDA's `graph actual`, which fans out to `DescribeFleetImports` (all scenarios) in parallel with the proto-health surface call:

```
# cold full-graph build (includes the parallel DescribeFleetImports fan-out)
time scenario-dependency-analyzer graph actual --json   # real 0m17.114s
# warm (persistent cache hit on both upstreams)
time scenario-dependency-analyzer graph actual --json   # real 0m0.210s
```

Before this work SDA's graph never completed in 3 minutes — dominated by the sequential, uncached `DescribeFleetImports` cost (27s cold, >90s warm). The fact that the whole graph (including this call) is now 17.1s cold / 0.21s warm demonstrates the fix landed.

## Root causes fixed (per plan Phase 1)

1. **Sequential per-scenario scan** → `DescribeFleetImports` parallelized with `min(16, NumCPU)` workers (`service.go`).
2. **In-memory cache** → `SQLiteCacheRepository` (already built, previously unused) wired as the production default (`handlers/facts/module.go`, gated on `db != nil`).
3. **Expensive cache key** → `sourceFingerprint` changed from SHA-256-every-file to an **mtime+size signature** (cheap stat-walk) as the primary key, with content-hash fallback on signature change (`cache.go`). This is what makes warm O(stat) instead of O(hash-all-files).

## Tradeoffs

- **mtime+size primary key**: a same-mtime same-size content edit could be missed by the signature; mitigated by the content-hash fallback on signature change. Acceptable for a derived-fact cache. Documented in `cache.md`.
