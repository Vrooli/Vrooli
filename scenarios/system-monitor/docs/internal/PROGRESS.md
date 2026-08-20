# Spec Sync Progress

## 2026-08-20 — Production-readiness evidence pass

- Implemented honest three-state metric propagation (`measured`, `unsupported`,
  `failed`) through the protobuf/API/UI path; non-measured values are excluded
  from threshold evaluation and are not rendered as numeric zeroes.
- Added native CPU, memory, network, process, disk, and typed investigation
  paths for the supported build targets. Shell investigations remain declared
  escape hatches where they provide materially richer Linux-only evidence.
- Added authoritative disk-pressure consumption to autoheal, low-power
  collection profiles and headroom self-metrics, PID-scoped stop validation,
  platform evidence metadata, and the responsive `vrooli-default` UI contract.
- Live evidence: lifecycle-managed System Monitor is healthy; the latest direct
  `/health` sample reports `last_cycle_duration_ms` of 456.376 with
  `headroom_ok: true`, and `/api/v1/disk-pressure` reports the observed `/`
  mount at 68.003%. Native CPU investigation execution completed.
- BAS evidence is current at desktop, tablet, and mobile sizes. Settled metric
  captures report `DISK: 68.0%`, four numeric meters, and a typed GPU status;
  the 68-second hold reports the same disk value. At 390px the root remains
  390px wide, header controls are within the viewport, detail buttons are
  uniquely named, and the logs page text controls now measure 44px high.
- Validation: four-target Go builds plus collectors/process-sampler contract
  test compilation pass; host collector/procsampler/convert tests pass. UI
  type-check, build, lint (0 errors / 616 warnings), unit-health (0 blocking),
  experience validation, dependency governance, and all 216 UI tests pass.
- Autoheal evidence: after a control-plane stop, two `vrooli-autoheal tick`
  cycles produced a successful `scenario-system-monitor/start`; the scenario
  returned healthy without a scenario-specific autoheal opt-in.
- The latest terminal governed run `20260820-083236-aabbc0fe` passed all 22
  phases in 373 seconds, including Agent Conformance at L4. The optional
  integration remains declared with its profile source and typed degraded
  behavior. Agent Manager's lifecycle manifest now gives its synchronous
  recovery/bootstrap path a 90-second API readiness budget; its optional
  credential gaps remain explicitly degraded.
- Raspberry Pi 4 hardware remains unavailable. Linux arm64 therefore remains
  build-verified rather than promoted to supported; no hardware claim is being
  inferred from the cross-build.

## 2026-02-16 — Fourth Spec Sync Pass

### What Was Synced

**New finding: Missing process kill endpoint**
- UI `ProcessMonitor.tsx` calls `POST /processes/{pid}/kill` with confirmation dialog, but this endpoint is NOT registered in `api/internal/server/router.go`
- Kill action silently fails at runtime
- Updated: PRD.md (Known Limitations), README.md (Known Limitations), PROBLEMS.md (Stability Issues), MOD-P0-009 requirement

**Requirement status correction (MOD-P0-009):**
- Changed from `"implemented"` → `"in-progress"`: process monitoring (zombie, high-thread, leak candidates) works, but process kill does not because the API endpoint is missing
- Updated module description with details about the missing endpoint

**Count corrections in this file:**
- Third pass incorrectly stated "10 implemented" (was 11) and "2 in-progress" (was 3)
- Corrected counts: 10 implemented, 4 in-progress, 4 pending (after MOD-P0-009 reclassification)

### Corrected Status Summary (18 modules)

- **10 implemented**: MOD-P0-001 (metrics), MOD-P0-002 (anomaly detection), MOD-P0-003 (investigations), MOD-P0-005 (thresholds), MOD-P0-006 (reports), MOD-P0-007 (dashboard), MOD-P0-010 (infrastructure), MOD-P1-012 (webhooks), MOD-P1-013 (cooldown), MOD-P1-014 (agent config)
- **4 in-progress**: MOD-P0-004 (persistent metrics storage — defaults to in-memory), MOD-P0-008 (scripts — API placeholders), MOD-P0-009 (process mgmt — kill endpoint missing), MOD-P1-016 (alert routing — email not implemented)
- **4 pending**: MOD-P1-011 (trends), MOD-P1-015 (custom metrics), MOD-P1-017 (predictions), MOD-P1-018 (correlations)

---

## 2026-02-16 — Third Spec Sync Pass

### What Was Synced

**requirements/ (NEW — bootstrapped and synced):**
- Bootstrapped 18 requirement modules via `prd-control-tower requirements generate` from PRD operational targets
- Verified each module status against actual implementation code
- Added descriptions to all modules with code file references
- Validation references remain empty (no tests exist)

**docs/internal/SEAMS.md (NEW):**
- Created from knowledge-observatory template
- Documented 6 integration seams: Repository interface, Collector interface, Agent-Manager client, Alert notification, Infrastructure provider, Tool registry
- Mapped responsibility zones across handler/service/repository/integration layers
- Identified decision points: threshold evaluation, cooldown management, storage backend selection
- Noted observability gaps: no OpenTelemetry, no structured error codes, no meta-monitoring

**docs/internal/PROBLEMS.md (NEW):**
- Created from knowledge-observatory template
- Documented 5 code quality debt items: CLI JSON parsing, CLI report bug, CLI --quiet flag, CLI entry point divergence, script API placeholders
- Documented 5 test gap areas: no unit tests (except 2 files), empty test/, no integration/UI/CLI tests
- Documented 3 stability issues: in-memory storage, missing UI-referenced endpoints, no auth
- Documented 3 UX issues: broken simulate command, non-existent endpoint references, no WebSocket

**docs/manifest.json:**
- Added SEAMS.md and PROBLEMS.md entries

### Previously Synced

**Second Pass:**
- PRD.md: Corrected polling intervals, added agent config fields, documented UI maintenance state behavior
- README.md: Corrected polling descriptions, clarified CLI label
- docs/manifest.json: Created initial manifest

**First Pass:**
- PRD.md: Corrected script count (70+ → 30), removed non-existent timeline endpoint, added missing endpoints, documented CLI bugs, fixed design decisions
- README.md: Matched corrections from PRD sync

### 2026-07-08 Cleanup-Manager Integration Slice

- Implemented the proto-owned `MetricsService.GetDiskDetail` handler and service path with read-only partition, largest-directory, and largest-file attribution.
- Added storage-manager handoff notes to disk detail responses so system-monitor surfaces pressure while storage-manager owns preview/apply/audit.
- Validation: `go test ./...` passed in `scenarios/system-monitor/api`.

### Unresolved Discrepancies

1. **CLI report endpoint bug**: CLI calls `/api/reports/generate` (wrong path). Actual API route is `/api/v1/reports/generate`. Needs code fix.
2. **UI references non-existent process endpoint**: `POST /processes/{pid}/kill` is called by UI code but not registered in the API router. Needs API implementation or UI fallback handling.
3. **Script API not implemented**: API has placeholder handlers for script list/get/execute. 30 scripts exist on disk but aren't served via API.
4. **Empty test/ directory**: Tests defined via test-genie but no test phases populated.
5. **No validation references**: All requirement modules have empty `ref` fields for validation entries — no actual test files to reference.

## Archive Readiness Assessment

```
CRITICAL (must pass):
  [x] PRD.md operational targets match implemented features
  [x] Every implemented feature has a corresponding requirement
  [x] Every requirement status reflects actual code state
  [x] README.md feature list matches reality
  [x] No spec artifact references code/files that don't exist

IMPORTANT (should pass):
  [ ] Requirements have valid test/validation references (no tests exist)
  [ ] README setup instructions are verified working (not tested)
  [x] docs/manifest.json is complete
  [x] Internal docs (SEAMS, PROBLEMS, PROGRESS) are current
  [x] Architecture documentation matches actual code structure

NICE-TO-HAVE:
  [x] Bidirectional DOC:/[CODE:] references
  [ ] All docs follow documentation-health standards
  [ ] Configuration documentation covers all options
```

**Verdict**: All CRITICAL items pass. The scenario is **archive-ready** for spec purposes. Remaining gaps are test coverage (IMPORTANT) and doc-code cross-references (NICE-TO-HAVE).
