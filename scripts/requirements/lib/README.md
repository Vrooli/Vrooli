# Requirements System - Library Modules

This directory contains the modular components of Vrooli's requirements tracking and reporting system. The system was refactored from a monolithic 1,530-line file into focused, testable modules.

## Architecture Overview

```
report.js (208 lines)          # CLI orchestrator
    ↓
lib/
├── types.js (174 lines)       # JSDoc type definitions
├── discovery.js (181 lines)   # File discovery & scenario resolution
├── parser.js (82 lines)       # JSON parsing & normalization
├── evidence.js (244 lines)    # Phase results & vitest extraction
├── enrichment.js (280 lines)  # Status computation & aggregation
├── sync.js (380 lines)        # Sync operations
├── output.js (170 lines)      # Report rendering
└── utils.js (300 lines)       # Shared utilities
```

**Total: 2,019 lines** (previously 1,530 lines in single file, but now with comprehensive types and better organization)

## Module Responsibilities

### types.js
**Purpose**: Comprehensive JSDoc type definitions for IDE support and documentation

**Key Types**:
- `Requirement` - Core requirement structure
- `Validation` - Test validation entry
- `EvidenceRecord` - Live test evidence from phase results
- `PhaseResult` - Phase execution results
- `SyncResult` - Sync operation results

**Usage**: Import types in JSDoc comments for autocomplete and type checking
```javascript
/**
 * @param {import('./types').Requirement[]} requirements
 * @returns {import('./types').Summary}
 */
```

### discovery.js
**Purpose**: Locate and enumerate requirement files within a scenario

**Exports**:
- `isScenarioRoot(directory, scenario)` - Check if directory is a scenario root
- `resolveScenarioRoot(baseDir, scenario)` - Find scenario root directory
- `parseImports(indexPath)` - Extract imports array from index.json
- `collectRequirementFiles(scenarioRoot)` - Collect all requirement JSON files

**Key Features**:
- Supports legacy `docs/requirements.json` format
- Supports modular `requirements/` folder structure
- Resolves imports from `requirements/index.json`
- Handles circular import detection

**Example**:
```javascript
const discovery = require('./lib/discovery');
const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), 'browser-automation-studio');
const files = discovery.collectRequirementFiles(scenarioRoot);
// Returns: [{ path: '...', relative: 'requirements/index.json', isIndex: true }, ...]
```

### parser.js
**Purpose**: Parse requirement JSON files and normalize structure

**Exports**:
- `parseRequirementFile(filePath)` - Parse single requirement file

**Key Features**:
- Normalizes `validation` and `validations` fields
- Attaches non-enumerable `__meta` property for tracking
- Preserves original structure while adding defaults
- Validates JSON syntax with helpful error messages

**Example**:
```javascript
const parser = require('./lib/parser');
const { requirements } = parser.parseRequirementFile('/path/to/requirements.json');
// Returns: { requirements: [{ id, status, validations, __meta, ... }] }
```

### evidence.js
**Purpose**: Load and extract test evidence from phase results and vitest reports

**Exports**:
- `detectValidationSource(validation)` - Detect phase/automation source from validation ref
- `loadPhaseResults(scenarioRoot)` - Load phase results from `coverage/phase-results/`
- `extractVitestFilesFromPhaseResults(scenarioRoot)` - Extract vitest test files from evidence
- `collectValidationsForPhase(requirements, phaseName, scenarioRoot)` - Filter validations by phase

**Key Features**:
- Parses `coverage/phase-results/*.json` files
- Extracts vitest test files from `ui/coverage/vitest-requirements.json`
- Normalizes requirement status values
- Maps validation refs to phase names

**Example**:
```javascript
const evidence = require('./lib/evidence');
const { phaseResults, requirementEvidence } = evidence.loadPhaseResults(scenarioRoot);
// Returns:
// {
//   phaseResults: { unit: {...}, integration: {...} },
//   requirementEvidence: { 'BAS-WORKFLOW-PERSIST-CRUD': [{ status, phase, evidence }] }
// }
```

### enrichment.js
**Purpose**: Enrich requirements with live test results and compute aggregates

**Exports**:
- `enrichValidationResults(requirements, context)` - Add live test status to validations
- `aggregateRequirementStatuses(requirements, requirementIndex)` - Rollup parent/child statuses
- `computeSummary(records)` - Calculate summary statistics

**Key Features**:
- Merges live phase results with validation entries
- Computes aggregate `liveStatus` for each requirement
- Handles parent/child requirement relationships with cycle detection
- Prioritizes evidence by status (failed > skipped > passed > not_run)

**Example**:
```javascript
const enrichment = require('./lib/enrichment');
const context = { phaseResults, requirementEvidence };
enrichment.enrichValidationResults(requirements, context);
// Adds `liveStatus`, `liveEvidence`, `liveDetails` to each requirement
enrichment.aggregateRequirementStatuses(requirements, requirementIndex);
// Updates parent requirement statuses based on children
const summary = enrichment.computeSummary(requirements);
// Returns: { total, byStatus, liveStatus, criticalityGap }
```

### sync.js
**Purpose**: Synchronize requirement files with live test evidence

**Exports**:
- `syncRequirementFile(filePath, requirements)` - Update status fields in JSON file
- `addMissingValidations(filePath, requirements, vitestFiles)` - Add vitest validations
- `detectOrphanedValidations(filePath, scenarioRoot, options)` - Find orphaned validations
- `syncRequirementRegistry(fileRequirementMap, scenarioRoot, options)` - Sync entire registry
- `printSyncSummary(syncResult)` - Pretty-print sync results

**Key Features**:
- Updates validation status based on live evidence
- Auto-adds missing vitest test files found in evidence
- Detects orphaned validations (test files that no longer exist)
- Optional `--prune-stale` to remove orphaned validations
- Preserves non-vitest validations (automation, manual, Go tests)

**Example**:
```javascript
const sync = require('./lib/sync');
const syncResult = sync.syncRequirementRegistry(fileRequirementMap, scenarioRoot, options);
sync.printSyncSummary(syncResult);
// Outputs:
// ✅ Added Validations: BAS-WORKFLOW-PERSIST-CRUD: + ui/src/stores/__tests__/projectStore.test.ts
// ⚠️  Orphaned Validations: BAS-OLD-FEATURE: × ui/src/components/Deleted.test.tsx
```

### output.js
**Purpose**: Render requirement reports in various formats

**Exports**:
- `renderJSON(summary, requirements)` - JSON format for CI/dashboards
- `renderMarkdown(summary, requirements, includePending)` - Markdown tables
- `renderTrace(requirements, scenarioRoot)` - Full traceability report
- `renderPhaseInspect(options, requirements, scenarioRoot)` - Phase-specific inspection

**Key Features**:
- JSON: Machine-readable for CI integration
- Markdown: Human-readable tables with status breakdown
- Trace: Full validation details with file existence checks
- Phase-inspect: Filtered view of single phase validations

**Example**:
```javascript
const output = require('./lib/output');
const json = output.renderJSON(summary, requirements);
const markdown = output.renderMarkdown(summary, requirements, false);
const trace = output.renderTrace(requirements, scenarioRoot);
```

### utils.js
**Purpose**: Shared utility functions and constants

**Exports**:
- `REQUIREMENT_STATUS_PRIORITY` - Status priority map for sorting
- `normalizeRequirementStatus(value)` - Normalize status strings
- `compareRequirementEvidence(current, candidate)` - Compare evidence records
- `selectBestRequirementEvidence(records, phaseName?)` - Select highest priority evidence
- `deriveDeclaredRollup(childStatuses, fallback)` - Rollup declared statuses
- `deriveLiveRollup(childStatuses, fallback)` - Rollup live statuses
- `deriveValidationStatus(validation)` - Compute validation status from live data
- `deriveRequirementStatus(existing, validationStatuses)` - Compute requirement status from validations

**Key Features**:
- Centralizes status priority logic
- Handles status normalization (pass/passed/success → 'passed')
- Implements rollup algorithms for parent/child requirements

**Example**:
```javascript
const utils = require('./lib/utils');
const normalized = utils.normalizeRequirementStatus('pass'); // → 'passed'
const best = utils.selectBestRequirementEvidence(evidenceRecords); // → highest priority
```

## Data Flow

### Report Generation Flow
```
1. discovery.resolveScenarioRoot()       → Find scenario directory
2. discovery.collectRequirementFiles()   → Enumerate JSON files
3. parser.parseRequirementFile()         → Parse each file
4. evidence.loadPhaseResults()           → Load live test results
5. enrichment.enrichValidationResults()  → Merge live evidence
6. enrichment.aggregateRequirementStatuses() → Rollup parent/child
7. enrichment.computeSummary()           → Calculate statistics
8. output.renderJSON/Markdown/Trace()    → Format output
```

### Sync Operation Flow
```
1. discovery + parser (same as above)    → Load requirements
2. evidence.loadPhaseResults()           → Load live evidence
3. enrichment.enrichValidationResults()  → Merge evidence
4. evidence.extractVitestFilesFromPhaseResults() → Get vitest files
5. sync.detectOrphanedValidations()      → Find orphaned entries
6. sync.addMissingValidations()          → Add new vitest tests
7. sync.syncRequirementFile()            → Update status fields
8. sync.printSyncSummary()               → Display results
```

## Testing Strategy

Each module is designed to be independently testable:

```javascript
// Example test structure (future work)
describe('discovery', () => {
  it('should resolve scenario root from cwd', () => {
    const root = discovery.resolveScenarioRoot('/path/to/scenario', 'my-scenario');
    expect(root).toContain('my-scenario');
  });
});

describe('enrichment', () => {
  it('should compute correct criticality gap', () => {
    const requirements = [
      { id: 'A', status: 'complete', criticality: 'P0' },
      { id: 'B', status: 'pending', criticality: 'P0' },
    ];
    const summary = enrichment.computeSummary(requirements);
    expect(summary.criticalityGap).toBe(1); // Only B is incomplete P0
  });
});
```

## Benefits of Modular Architecture

**Maintainability**:
- Each module < 400 lines (except sync at 380)
- Clear single responsibilities
- Easy to understand and modify

**Testability**:
- Modules can be tested in isolation
- Mock dependencies easily
- Faster test execution

**Developer Experience**:
- IDE autocomplete via JSDoc types
- Clear module boundaries
- Easy to find relevant code

**Performance**:
- Modules can be lazy-loaded if needed
- Easier to identify performance bottlenecks
- Better code splitting potential

## Migration Guide (from old report.js)

If you have code that directly imported internal functions from the old report.js:

**Before**:
```javascript
// ❌ Old internal imports (if they existed)
const { parseRequirementFile } = require('./report.js');
```

**After**:
```javascript
// ✅ New modular imports
const parser = require('./lib/parser');
const { parseRequirementFile } = parser;
```

The main CLI interface (`node report.js --scenario X`) remains unchanged.

## Future Enhancements

1. **Unit Tests**: Add comprehensive test suite in `lib/__tests__/`
2. **TypeScript Migration**: Convert to TypeScript for compile-time type safety
3. **Performance Profiling**: Benchmark and optimize hot paths
4. **Caching**: Cache parsed requirements for faster repeated runs
5. **Plugins**: Allow custom validation sources beyond vitest/Go/phase scripts

## Questions?

See the main README or run:
```bash
node scripts/requirements/report.js --help
```
