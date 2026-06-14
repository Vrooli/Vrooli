# Problems and Solutions - Scenario Dependency Analyzer

## Problems Discovered (2025-09-28)

### 1. CLI-API Graph Type Mismatch
**Problem**: The CLI expected graph types like "hierarchical", "network", "circular" but the API expected "resource", "scenario", "combined".
**Solution**: Fixed the CLI to use the correct API types and added mapping for user-friendly aliases.
**Status**: ✅ Fixed

### 2. Missing Optimize Command
**Problem**: The optimize command was specified in the PRD as P0 but was not implemented in the CLI.
**Solution**: Added the optimize command with basic functionality and a roadmap for full implementation.
**Status**: ✅ Implemented (basic version)

### 3. Database Initialization
**Problem**: The database schema exists but may not be properly initialized when the scenario starts.
**Solution**: The schema.sql is present but requires proper lifecycle integration for auto-initialization.
**Status**: ⚠️ Needs verification

### 4. AI Resource Integration
**Problem**: Qdrant-based semantic matching may fail silently if resources aren't running.
**Solution**: Added fallback heuristics when semantic resources are unavailable.
**Status**: ✅ Fallbacks implemented

### 5. Test Failures
**Problem**: The `make test` command fails at the API health check step.
**Solution**: The health endpoint works when tested directly, suggesting a timing or port issue in the test script.
**Status**: 🔍 Needs investigation

## Technical Debt

1. **Qdrant Integration**: Currently uses exec commands to call resource-qdrant, should use proper API client
2. **Optimization Engine**: Currently returns placeholder data, needs full implementation
3. **Circular Dependency Detection**: Graph algorithms exist in code but not fully integrated
4. **Historical Tracking**: Database tables exist but no automatic tracking implemented

## Recommendations for Next Iteration

1. **Priority**: Implement proper resource client libraries instead of exec calls
2. **Database**: Add automatic migration runner on startup
3. **Testing**: Fix the test suite to properly wait for API readiness
4. **UI**: Add WebSocket support for real-time updates during analysis
5. **Performance**: Add Redis caching layer for frequently accessed dependency data

## Actual Interface Graph Migration (2026-06-13)

### Declared Dependencies Are Not Actual Interface Usage
**Problem**: SDA historically reported declared dependencies from `service.json` plus local scanner heuristics. That cannot answer whether a scenario actually imports another scenario's protobuf or generated Go client, and it cannot reliably flag `service.json` drift.

**Planned solution**: Consume batch facts from `proto-health` and `code-facts`, attribute import paths to scenario slugs, and expose an on-demand evidence-tagged interface graph.

**Status**: Planned by `scenario-dependency-analyzer-actual-interface-graph-and-import-drift`.

### Scanner Ownership Boundary
**Problem**: SDA's scanner owns too much language-specific evidence logic. `detectPortCalls` follows obsolete `resolveScenarioPortViaCLI` usage, and `detectCLIReferences` is superseded by Connect/proto imports. Retaining this style would duplicate code-facts and keep dependency evidence brittle.

**Planned solution**: Delete the obsolete/superseded detectors during the graph migration. Keep only interim non-import signals with precise comments until upstream AST facts exist.

**Status**: Planned cleanup.

### Follow-Up: AST Facts for Runtime and CLI Usage
**Problem**: Runtime `ResolveScenarioURL*` calls and `vrooli scenario run` shell-outs are real cross-scenario usage signals, but they are not import-level evidence and should not be rebuilt as regex scanners inside SDA.

**Follow-up plan**: `scenario-dependency-analyzer-code-evidence-via-ast-facts`

**Scope**: Add AST analyzers in `go-code-graph` and `typescript-code-graph` for modern discovery calls and scenario shell-outs, surface those through `code-facts`, then delete SDA's retained regex detectors and delegate resource-usage detection to fact providers.
