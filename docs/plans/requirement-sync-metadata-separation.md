# Requirement Sync Metadata Separation Plan

## Executive Summary

Move ephemeral test synchronization metadata from git-tracked requirement files to gitignored coverage directory, reducing git noise while preserving semantic status tracking.

**Impact**: Every test run currently dirties all requirement files with timestamp updates. Post-implementation, only actual progress/regression will trigger requirement file changes.

**Status**: Planning (No code changes yet)

---

## Problem Statement

### Current Behavior

When tests run, the sync system (`scripts/requirements/report.js --mode sync`) updates ALL requirement module files with `_sync_metadata` blocks containing:

**At requirement level** (`requirements[n]._sync_metadata`):
- `last_updated` - ISO timestamp
- `updated_by` - Always "auto-sync"
- `test_coverage_count` - Number of implemented validations
- `all_tests_passing` - Boolean
- `phase_results` - Array of phase result file paths
- `tests_run` - Array of test commands executed

**At validation level** (`requirements[n].validation[m]._sync_metadata`):
- `last_test_run` - ISO timestamp
- `test_duration_ms` - Execution time
- `auto_updated` - Boolean flag
- `test_names` - Array of test names
- `phase_result` - Phase result file path
- `source_phase` - Phase name
- `tests_run` - Array of test commands
- `manual` - Manual validation metadata (if type=manual)

**At module level** (`_metadata.last_synced_at`):
- ISO timestamp of last sync

### Why This Is Problematic

1. **Git Noise**: Running `git status` after tests shows dozens of modified requirement files, even when no functional changes occurred
2. **Unnecessary Diffs**: Pull requests and commits become cluttered with timestamp-only changes
3. **Agent Confusion**: AI agents see massive diffs and struggle to identify actual progress vs. routine test runs
4. **Poor Signal-to-Noise**: The real signal (status changes from `pending` → `in_progress` → `complete`) gets buried in timestamp noise

### Example of Current Git Noise

```bash
$ npm test
# ... tests run successfully, no code changes ...

$ git status
modified:   scenarios/landing-manager/requirements/01-template-management/module.json
modified:   scenarios/landing-manager/requirements/02-admin-portal/module.json
modified:   scenarios/landing-manager/requirements/03-customization-ux/module.json
# ... 7 more files with only timestamp changes ...
```

### What Should Happen Instead

```bash
$ npm test
# ... tests run successfully, no code changes ...

$ git status
# Nothing to commit, working tree clean

# Only if a test goes from failing → passing:
modified:   scenarios/landing-manager/requirements/01-template-management/module.json
# (Because status changed from 'failing' to 'implemented')
```

---

## Current Architecture

### Sync Flow (Simplified)

```
test/run-tests.sh
  ↓
scripts/scenarios/testing/shell/suite.sh
  ↓
[Runs test phases: unit, integration, business, etc.]
  ↓
coverage/phase-results/*.json (ephemeral test results)
  ↓
scripts/requirements/report.js --mode sync
  ↓
scripts/requirements/lib/sync.js
  - syncRequirementFile()
    - Reads requirement file
    - Computes new _sync_metadata
    - Writes back to requirement file (ALWAYS if metadata changed)
  ↓
requirements/**/*.json (git-tracked, MODIFIED ON EVERY TEST RUN)
```

### Key Files Involved

**Writers** (code that creates/updates `_sync_metadata`):
- `scripts/requirements/lib/sync.js` - Main sync logic
  - `buildRequirementSyncMetadata()` - Line 53
  - `buildValidationSyncMetadata()` - Line 96
  - `syncRequirementFile()` - Line 236 (writes to disk)
  - `addMissingValidations()` - Line 370
  - `detectOrphanedValidations()` - Line 447

**Readers** (code that reads `_sync_metadata`):
- `scripts/requirements/lib/parser.js` - Line 54, 65 (preserves in `__meta`)
- `scripts/requirements/lib/snapshot.js` - Line 64, 72 (copies to snapshot)
- `scripts/requirements/drift-check.js` - Line 236 (checks manual validations)
- `scripts/requirements/schema.json` - Lines 137-166, 212-258 (schema definitions)

**Consumers** (code that uses snapshot data):
- `scripts/requirements/drift-check.js` - Compares current state vs. snapshot
- Test phases consume `coverage/phase-results/*.json` directly (not requirement files)

### Critical Insight: Snapshot Already Exists

The system ALREADY creates `coverage/requirements-sync/latest.json` containing all sync metadata! This snapshot includes:
- Complete requirement state (line 239 in snapshot.js)
- Operational targets rollup (line 240)
- Manual validations (line 254)
- Test commands executed (line 234)

**The snapshot is gitignored but requirement files are tracked** - this is backwards!

---

## Proposed Solution

### Design Principles

1. **Semantic vs. Ephemeral Separation**: Git-tracked files contain only semantic state; ephemeral metadata lives in gitignored coverage directory
2. **Idempotent Status Updates**: Only write to requirement files when `status` fields actually change
3. **No Backwards Compatibility**: Clean break - accept that old snapshots become invalid (acceptable since coverage/ is gitignored)
4. **Maintain Existing APIs**: External consumers should continue working without changes

### New Architecture

```
test/run-tests.sh
  ↓
[Test execution - unchanged]
  ↓
coverage/phase-results/*.json
  ↓
scripts/requirements/report.js --mode sync
  ↓
scripts/requirements/lib/sync.js (MODIFIED)
  - Derives status changes
  - ONLY writes to requirements/*.json if status changed
  - Writes ALL sync metadata to coverage/sync/*.json
  ↓
requirements/**/*.json (ONLY modified when status changes)
coverage/sync/*.json (NEW - contains all _sync_metadata)
```

### Sync Metadata Storage Format

**Location**: `coverage/sync/` directory (gitignored)

**Structure**: One JSON file per requirement module, mirroring the requirements directory structure

**Example**: For `requirements/01-template-management/module.json`, create:
```
coverage/sync/01-template-management.json
```

**Schema**:
```json
{
  "module_id": "template-management",
  "module_file": "requirements/01-template-management/module.json",
  "last_synced_at": "2025-11-21T22:45:00.000Z",
  "requirements": {
    "TMPL-AVAILABILITY": {
      "last_updated": "2025-11-21T22:45:00.000Z",
      "updated_by": "auto-sync",
      "test_coverage_count": 0,
      "all_tests_passing": true,
      "phase_results": [
        "coverage/phase-results/integration.json"
      ],
      "tests_run": [
        "test/run-tests.sh"
      ],
      "validations": {
        "0": {
          "last_test_run": "2025-11-21T22:45:00.000Z",
          "test_duration_ms": 1000,
          "auto_updated": true,
          "test_names": [],
          "phase_result": "coverage/phase-results/integration.json",
          "source_phase": "integration",
          "tests_run": ["test/run-tests.sh"]
        }
      }
    },
    "TMPL-METADATA": {
      "last_updated": "...",
      "validations": { ... }
    }
  }
}
```

### What Stays in Requirements Files

**KEEP** (semantic state - belongs in git):
- `requirements[n].id`
- `requirements[n].status` - **ONLY updated when actual status changes**
- `requirements[n].title`, `description`, `criticality`, `prd_ref`
- `requirements[n].validation[m].type`, `ref`, `phase`, `workflow_id`
- `requirements[n].validation[m].status` - **ONLY updated when actual status changes**
- `requirements[n].validation[m].notes`
- `_metadata.module`, `description` (if present)

**REMOVE** (ephemeral metadata - moves to coverage/sync/):
- `requirements[n]._sync_metadata` (entire block)
- `requirements[n].validation[m]._sync_metadata` (entire block)
- `_metadata.last_synced_at`

### Stale Sync Data Cleanup

When a requirement is deleted/renamed from a module file:
- **Approach**: On write, only include sync data for requirements that exist in current requirement file
- **Effect**: Orphaned requirement sync data automatically drops on next test run
- **No explicit cleanup command needed**: The write operation naturally prunes stale entries

---

## Implementation Plan

### Phase 1: Create New Sync Data Writer

**File**: `scripts/requirements/lib/sync.js`

**Changes**:

1. **Add new function** `writeSyncMetadataFile(moduleFilePath, syncData, scenarioRoot)`:
   ```javascript
   function writeSyncMetadataFile(moduleFilePath, syncData, scenarioRoot) {
     // Derive sync file path from module file path
     // e.g., requirements/01-foo/module.json → coverage/sync/01-foo.json
     const relativePath = path.relative(
       path.join(scenarioRoot, 'requirements'),
       moduleFilePath
     );
     const syncFileName = relativePath
       .replace(/\/module\.json$/, '.json')
       .replace(/\.json$/, '.json'); // Keep .json extension

     const syncDir = path.join(scenarioRoot, 'coverage', 'sync');
     const syncFilePath = path.join(syncDir, syncFileName);

     fs.mkdirSync(path.dirname(syncFilePath), { recursive: true });
     fs.writeFileSync(syncFilePath, JSON.stringify(syncData, null, 2) + '\n', 'utf8');
   }
   ```

2. **Modify** `syncRequirementFile()` (line 236):
   - Build sync metadata as before
   - **Separate** sync metadata from status updates
   - **Only write requirement file if status changed**
   - **Always write sync file** with latest metadata

   ```javascript
   function syncRequirementFile(filePath, requirements, context = {}) {
     // ... existing code to derive status changes ...

     const syncData = {
       module_id: parsed.module_id || null,
       module_file: normalizeRelativePath(context.scenarioRoot, filePath),
       last_synced_at: new Date().toISOString(),
       requirements: {}
     };

     requirements.forEach((requirement) => {
       // Build sync metadata (as before)
       const reqSyncMeta = buildRequirementSyncMetadata(...);
       const valSyncMetas = requirement.validations.map(v =>
         buildValidationSyncMetadata(...)
       );

       // Store in syncData (for coverage/sync/)
       syncData.requirements[requirement.id] = {
         ...reqSyncMeta,
         validations: {}
       };
       valSyncMetas.forEach((meta, idx) => {
         syncData.requirements[requirement.id].validations[idx] = meta;
       });

       // Determine if status changed
       const statusChanged = requirement.status !== requirement.__meta.originalStatus;
       if (statusChanged) {
         // Update requirement file
         parsedRequirement.status = requirement.status;
         metadataChanged = true;
       }

       // DO NOT write _sync_metadata to requirement file anymore
     });

     // Write sync file (always)
     writeSyncMetadataFile(filePath, syncData, context.scenarioRoot);

     // Write requirement file (only if status changed)
     if (metadataChanged) {
       // Remove _sync_metadata before writing
       removeAllSyncMetadata(parsed);
       // Remove _metadata.last_synced_at
       if (parsed._metadata) {
         delete parsed._metadata.last_synced_at;
       }
       fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
     }

     return updates;
   }
   ```

3. **Add helper** `removeAllSyncMetadata(parsed)`:
   ```javascript
   function removeAllSyncMetadata(parsed) {
     if (!parsed.requirements) return;
     parsed.requirements.forEach(req => {
       delete req._sync_metadata;
       if (req.validation) {
         req.validation.forEach(val => {
           delete val._sync_metadata;
         });
       }
     });
   }
   ```

### Phase 2: Update Sync Metadata Readers

**File**: `scripts/requirements/lib/parser.js`

Currently preserves `_sync_metadata` in `__meta.originalSyncMetadata` (lines 54, 65). This is fine - it becomes a no-op when files don't have sync metadata.

**No changes needed** - the parser can handle missing `_sync_metadata` fields gracefully.

---

**File**: `scripts/requirements/lib/snapshot.js`

Currently copies `sync_metadata` to snapshot (lines 64, 72).

**Change**: Read sync metadata from coverage/sync/ instead of requirement files:

```javascript
function loadSyncMetadataForModule(moduleFilePath, scenarioRoot) {
  const relativePath = path.relative(
    path.join(scenarioRoot, 'requirements'),
    moduleFilePath
  );
  const syncFileName = relativePath
    .replace(/\/module\.json$/, '.json');
  const syncFilePath = path.join(scenarioRoot, 'coverage', 'sync', syncFileName);

  if (!fs.existsSync(syncFilePath)) {
    return { requirements: {} };
  }

  try {
    const content = fs.readFileSync(syncFilePath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    console.warn(`Unable to load sync metadata from ${syncFilePath}: ${error.message}`);
    return { requirements: {} };
  }
}

function normalizeRequirementSnapshots(fileSnapshots, scenarioRoot) {
  const requirementMap = {};
  if (!Array.isArray(fileSnapshots)) {
    return requirementMap;
  }

  fileSnapshots.forEach((file) => {
    // Load sync metadata from coverage/sync/
    const syncData = loadSyncMetadataForModule(file.absolute_path, scenarioRoot);

    const requirements = Array.isArray(file.requirements) ? file.requirements : [];
    requirements.forEach((req) => {
      if (!req || !req.id) return;

      // Get sync metadata from loaded file, not from requirement object
      const reqSyncMeta = syncData.requirements[req.id] || null;

      requirementMap[req.id] = {
        id: req.id,
        status: req.status || 'pending',
        criticality: req.criticality || null,
        prd_ref: req.prd_ref || null,
        module: req.module || file.module || null,
        file: file.relative_path,
        sync_metadata: reqSyncMeta, // Now from coverage/sync/
        validations: Array.isArray(req.validations)
          ? req.validations.map((validation, idx) => {
              const valSyncMeta = reqSyncMeta && reqSyncMeta.validations
                ? reqSyncMeta.validations[idx]
                : null;
              return {
                type: validation.type || null,
                ref: validation.ref || null,
                status: validation.status || null,
                phase: validation.phase || null,
                workflow_id: validation.workflow_id || null,
                sync_metadata: valSyncMeta, // Now from coverage/sync/
              };
            })
          : [],
      };
    });
  });
  return requirementMap;
}
```

**Also update**: `captureFileSnapshot()` in sync.js to NOT capture sync_metadata from parsed file (since it won't be there).

### Phase 3: Update Drift Detection

**File**: `scripts/requirements/drift-check.js`

Currently reads `validation.sync_metadata` at line 236 from snapshot. This continues to work because snapshot.js will populate it from coverage/sync/.

**No changes needed** - drift detection reads from snapshot, not directly from requirement files.

### Phase 4: Update Schema Documentation

**File**: `scripts/requirements/schema.json`

**Change**: Update description for `_sync_metadata` properties:

```json
{
  "_sync_metadata": {
    "type": "object",
    "description": "DEPRECATED: Sync metadata has moved to coverage/sync/. This field is no longer written to requirement files. For historical files only.",
    "properties": { ... }
  }
}
```

**File**: `docs/testing/reference/requirement-schema.md`

Add note explaining the change:

```markdown
## Sync Metadata (Deprecated in Requirement Files)

As of [DATE], `_sync_metadata` blocks are no longer written to requirement files.
All ephemeral test synchronization data is stored in `coverage/sync/*.json` files.

Requirement files now only track semantic state (status fields).
See [Architecture Doc] for details on the new sync metadata storage.
```

### Phase 5: Clean Existing Requirements Files

**One-time migration** (can be done manually or via script):

1. Read all requirement files
2. Remove all `_sync_metadata` blocks
3. Remove `_metadata.last_synced_at`
4. Write back to disk

**Script**: `scripts/requirements/migrate-remove-sync-metadata.js`

```javascript
#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const discovery = require('./lib/discovery');

function parseArgs(argv) {
  return {
    scenario: argv.find((arg, i) => argv[i - 1] === '--scenario') || '',
    dryRun: argv.includes('--dry-run'),
  };
}

function removeSyncMetadata(parsed) {
  let changed = false;

  if (parsed._metadata && parsed._metadata.last_synced_at) {
    delete parsed._metadata.last_synced_at;
    changed = true;
  }

  if (Array.isArray(parsed.requirements)) {
    parsed.requirements.forEach(req => {
      if (req._sync_metadata) {
        delete req._sync_metadata;
        changed = true;
      }
      if (Array.isArray(req.validation)) {
        req.validation.forEach(val => {
          if (val._sync_metadata) {
            delete val._sync_metadata;
            changed = true;
          }
        });
      }
    });
  }

  return changed;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  if (!options.scenario) {
    throw new Error('Missing --scenario argument');
  }

  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const files = discovery.collectRequirementFiles(scenarioRoot);

  let modifiedCount = 0;

  files.forEach(file => {
    const content = fs.readFileSync(file.path, 'utf8');
    const parsed = JSON.parse(content);

    const changed = removeSyncMetadata(parsed);

    if (changed) {
      modifiedCount++;
      console.log(`${options.dryRun ? '[DRY RUN] Would clean' : 'Cleaning'}: ${file.relative}`);

      if (!options.dryRun) {
        fs.writeFileSync(file.path, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
      }
    }
  });

  console.log(`\nTotal files ${options.dryRun ? 'that would be modified' : 'modified'}: ${modifiedCount}`);
}

try {
  main();
} catch (error) {
  console.error(`Migration failed: ${error.message}`);
  process.exitCode = 1;
}
```

**Usage**:
```bash
# Preview changes
node scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager --dry-run

# Apply migration
node scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager
```

### Phase 6: Testing & Validation

**Test scenarios**:

1. **Baseline**: Run tests on clean scenario, verify sync files created
2. **No-op run**: Run tests twice, verify requirement files unchanged on second run
3. **Status change**: Make a test pass, verify ONLY that requirement file updated
4. **Snapshot integrity**: Verify `coverage/requirements-sync/latest.json` still contains all data
5. **Drift detection**: Verify drift-check.js still works correctly

**Commands**:
```bash
# Test landing-manager (has comprehensive requirements)
cd scenarios/landing-manager

# Clean slate
rm -rf coverage/

# First test run
make test

# Verify sync files created
ls -la coverage/sync/
# Should see: 01-template-management.json, 02-admin-portal.json, etc.

# Verify requirement files have NO _sync_metadata
grep -r "_sync_metadata" requirements/
# Should return nothing

# Second test run (no changes)
git status requirements/
# Should show: nothing to commit, working tree clean

make test

# Verify still clean
git status requirements/
# Should show: nothing to commit, working tree clean

# Verify snapshot still works
cat coverage/requirements-sync/latest.json | jq '.requirements | keys | length'
# Should show requirement count

# Verify drift detection
node ../../scripts/requirements/drift-check.js --scenario landing-manager
# Should show: "status": "ok"
```

---

## Migration Timeline

1. **Implement Phase 1-2** (sync writer + reader updates) - 2-3 hours
2. **Test on single scenario** (landing-manager) - 1 hour
3. **Implement Phase 3-4** (drift detection + schema docs) - 1 hour
4. **Create migration script** (Phase 5) - 30 minutes
5. **Run migration on all scenarios** - 15 minutes
6. **Comprehensive testing** (Phase 6) - 1 hour
7. **Document changes** - 30 minutes

**Total estimate**: 6-7 hours

---

## Rollback Plan

If issues arise:

1. **Before migration**: Git tracks all requirement files, easy to revert
2. **After migration**: Re-run old sync to regenerate `_sync_metadata` in files

**Rollback script**:
```bash
# Revert requirement file changes
git checkout -- scenarios/*/requirements/

# Re-run tests to regenerate _sync_metadata (old way)
cd scenarios/landing-manager && make test
```

**Risk**: Low - coverage/ directory is already gitignored, worst case is losing test run metadata (easily regenerated)

---

## Benefits Summary

### For Developers
- ✅ Clean `git status` after test runs
- ✅ PRs show only meaningful requirement changes
- ✅ Easier to see when actual progress is made

### For AI Agents
- ✅ Clear signal when requirements advance (status changes)
- ✅ No confusion from timestamp-only diffs
- ✅ Better understanding of test impact

### For System Architecture
- ✅ Proper separation: semantic (git) vs ephemeral (gitignored)
- ✅ Smaller git history (no more timestamp commits)
- ✅ Faster requirement file parsing (less data)

---

## Open Questions

1. **Should we keep `_metadata.last_synced_at` in snapshot?**
   - **Answer**: Yes - snapshot needs it for drift detection

2. **What about index.json files?**
   - **Answer**: Same treatment - they can have requirements too

3. **Performance impact of reading separate sync files?**
   - **Answer**: Minimal - only read during snapshot generation, not during normal reporting

4. **Should coverage/sync/ files be human-readable or can we compress them?**
   - **Answer**: Keep JSON pretty-printed for debugging - storage is cheap

---

## Success Criteria

✅ **Primary Goal**: Running tests twice with no code changes results in zero git-tracked file modifications

✅ **Secondary Goals**:
- All existing test commands continue working
- Snapshot integrity maintained
- Drift detection continues working
- Status changes still update requirement files
- Documentation updated

---

## References

**Modified Files** (implementation):
- `scripts/requirements/lib/sync.js` - Core sync logic
- `scripts/requirements/lib/snapshot.js` - Snapshot generation
- `scripts/requirements/schema.json` - Schema documentation

**Documentation Updates**:
- `docs/testing/reference/requirement-schema.md` - Schema reference
- `docs/testing/architecture/REQUIREMENT_FLOW.md` - Flow diagram update

**New Files**:
- `scripts/requirements/migrate-remove-sync-metadata.js` - One-time migration
- `coverage/sync/*.json` - New sync metadata storage (gitignored)

**Related Docs**:
- [Requirement Flow Architecture](../testing/architecture/REQUIREMENT_FLOW.md)
- [Requirement Tracking Guide](../testing/guides/requirement-tracking.md)
- [Testing Glossary](../testing/GLOSSARY.md)
