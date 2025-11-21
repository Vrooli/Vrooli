# Requirement Sync Metadata Separation Plan

## Executive Summary

Move ephemeral test synchronization metadata from git-tracked requirement files to gitignored coverage directory, reducing git noise while preserving semantic status tracking.

**Impact**: Every test run currently dirties all requirement files with timestamp updates. Post-implementation, only actual status changes will trigger requirement file modifications.

**Status**: Planning (Ready for implementation - all issues addressed)

**Migration Strategy**: Clean break - no backward compatibility. All requirement files will be cleaned in a single migration pass.

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
# (Because validation.status changed from 'not_implemented' to 'implemented')
```

**Note**: Requirement files will still be modified when:
- Requirement `status` changes (`pending` → `in_progress` → `complete`)
- Validation `status` changes (`not_implemented` → `passing` → `failing`)
- `notes` fields are manually updated

This is correct behavior - these are semantic changes. The goal is eliminating **timestamp-only** modifications.

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
  - `captureFileSnapshot()` - Line 186 (captures sync metadata to snapshot)
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

### Critical Insight: The Root Cause

**Problem location**: `sync.js:318-355`

```javascript
// Current logic (BROKEN - writes on ANY metadata change)
if (!metadataEqual(previousRequirementMetadata, nextRequirementMetadata)) {
  parsedRequirement._sync_metadata = nextRequirementMetadata;
  metadataChanged = true;  // ← This triggers file write even for timestamp changes!
}

// Later...
if (updates.length > 0 || metadataChanged) {  // ← Writes file on every test run
  parsed._metadata.last_synced_at = new Date().toISOString();
  fs.writeFileSync(filePath, finalContent, 'utf8');
}
```

The `metadataChanged` flag tracks **any** metadata change (timestamps, test_names, etc.), not just **status** changes. This causes writes on every test run.

---

## Proposed Solution

### Design Principles

1. **Semantic vs. Ephemeral Separation**: Git-tracked files contain only semantic state; ephemeral metadata lives in gitignored coverage directory
2. **Status-Only Writes**: Only write to requirement files when `status` fields actually change
3. **Clean Break Migration**: No backward compatibility - full migration in single pass
4. **Maintain Snapshot APIs**: External consumers (drift-check, etc.) continue working without changes

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
  - Builds sync metadata (as before)
  - Tracks status changes separately from metadata changes
  - ONLY writes to requirements/*.json if status changed
  - ALWAYS writes ALL sync metadata to coverage/sync/*.json
  ↓
requirements/**/*.json (ONLY modified when status changes)
coverage/sync/*.json (NEW - contains all _sync_metadata)
```

### Sync Metadata Storage Format

**Location**: `coverage/sync/` directory (gitignored via existing `coverage` ignore rule)

**Structure**: One JSON file per requirement module, mirroring the requirements directory structure

**Naming Convention**:
- `requirements/01-template-management/module.json` → `coverage/sync/01-template-management.json`
- `requirements/index.json` → `coverage/sync/index.json`

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
      "test_coverage_count": 3,
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
          "test_names": [
            "[REQ:TMPL-AVAILABILITY] CLI command 'template list' executes successfully"
          ],
          "phase_result": "coverage/phase-results/integration.json",
          "source_phase": "integration",
          "tests_run": ["test/run-tests.sh"]
        }
      }
    }
  }
}
```

### What Stays in Requirements Files

**KEEP** (semantic state - belongs in git):
- `requirements[n].id`
- `requirements[n].status` - **ONLY updated when actual status changes**
- `requirements[n].title`, `description`, `criticality`, `prd_ref`, `dependencies`, `notes`
- `requirements[n].validation[m].type`, `ref`, `phase`, `workflow_id`
- `requirements[n].validation[m].status` - **ONLY updated when actual status changes**
- `requirements[n].validation[m].notes`
- `_metadata.module`, `description`, `auto_sync_enabled`, `schema_version`

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

### Phase 1: Core Sync Writer Logic

**File**: `scripts/requirements/lib/sync.js`

**Changes**:

1. **Add new function** `writeSyncMetadataFile(moduleFilePath, syncData, scenarioRoot)`:
   ```javascript
   /**
    * Write sync metadata to coverage/sync/ directory
    * @param {string} moduleFilePath - Absolute path to requirement module file
    * @param {object} syncData - Sync metadata object
    * @param {string} scenarioRoot - Scenario root directory
    */
   function writeSyncMetadataFile(moduleFilePath, syncData, scenarioRoot) {
     const relativePath = path.relative(
       path.join(scenarioRoot, 'requirements'),
       moduleFilePath
     );

     // Transform path: requirements/01-foo/module.json → 01-foo.json
     //                requirements/index.json → index.json
     let syncFileName = relativePath
       .replace(/^requirements\//, '')  // Remove leading 'requirements/' if present
       .replace(/\/module\.json$/, '.json');  // Replace '/module.json' with '.json'

     // Handle index.json case
     if (syncFileName === 'index.json') {
       syncFileName = 'index.json';
     }

     const syncDir = path.join(scenarioRoot, 'coverage', 'sync');
     const syncFilePath = path.join(syncDir, syncFileName);

     fs.mkdirSync(path.dirname(syncFilePath), { recursive: true });

     try {
       fs.writeFileSync(syncFilePath, JSON.stringify(syncData, null, 2) + '\n', 'utf8');
     } catch (error) {
       console.error(`Failed to write sync metadata to ${syncFilePath}: ${error.message}`);
       throw error;
     }
   }
   ```

2. **Add helper** `removeAllSyncMetadata(parsed)`:
   ```javascript
   /**
    * Remove all _sync_metadata blocks from requirement structure
    * @param {object} parsed - Parsed requirement file
    */
   function removeAllSyncMetadata(parsed) {
     if (!parsed.requirements) return;

     parsed.requirements.forEach(req => {
       delete req._sync_metadata;

       // NOTE: Property name is 'validation' (singular) in JSON files
       if (Array.isArray(req.validation)) {
         req.validation.forEach(val => {
           delete val._sync_metadata;
         });
       }
     });
   }
   ```

3. **Completely rewrite** `syncRequirementFile()` (line 236):

   **Key changes**:
   - Separate `statusChanged` from `metadataChanged` tracking
   - Build sync metadata into separate structure
   - Write sync file on every run
   - Write requirement file ONLY when status changes

   ```javascript
   function syncRequirementFile(filePath, requirements, context = {}) {
     if (!requirements || requirements.length === 0) {
       return [];
     }

     const originalContent = fs.readFileSync(filePath, 'utf8');
     const parsed = JSON.parse(originalContent);
     const updates = [];
     let statusChanged = false;  // Track status changes separately
     const testsRunCommands = normalizeTestCommands(context.testCommands);

     const requirementMap = new Map();
     (parsed.requirements || []).forEach((req, idx) => {
       requirementMap.set(req.id, idx);
     });

     // Build sync data structure (for coverage/sync/)
     const syncData = {
       module_id: parsed.module_id || (parsed._metadata && parsed._metadata.module) || null,
       module_file: normalizeRelativePath(context.scenarioRoot, filePath),
       last_synced_at: new Date().toISOString(),
       requirements: {}
     };

     requirements.forEach((requirement) => {
       const requirementMeta = requirement.__meta;
       const validationStatuses = [];
       const originalStatus = requirementMeta && typeof requirementMeta.originalStatus === 'string'
         ? requirementMeta.originalStatus
         : requirement.status;

       // Process validations and check for status changes
       if (Array.isArray(requirement.validations)) {
         requirement.validations.forEach((validation, index) => {
           const derivedStatus = deriveValidationStatus(validation);
           validationStatuses.push(derivedStatus);

           if (derivedStatus && derivedStatus !== validation.status) {
             validation.status = derivedStatus;
             statusChanged = true;  // Validation status changed
             updates.push({
               type: 'validation',
               requirement: requirement.id,
               index,
               status: derivedStatus,
               file: filePath,
             });
           }
         });
       }

       // Check requirement status
       const nextStatus = deriveRequirementStatus(requirement.status, validationStatuses);
       if (nextStatus && nextStatus !== originalStatus) {
         requirement.status = nextStatus;
         if (requirementMeta) {
           requirementMeta.originalStatus = nextStatus;
         }
         statusChanged = true;  // Requirement status changed
         updates.push({
           type: 'requirement',
           requirement: requirement.id,
           status: nextStatus,
           file: filePath,
         });
       }

       const reqIndex = requirementMap.get(requirement.id);
       if (reqIndex === undefined || !parsed.requirements[reqIndex]) {
         return;
       }

       const parsedRequirement = parsed.requirements[reqIndex];

       // Update statuses in parsed structure (for potential write)
       parsedRequirement.status = requirement.status;

       if (Array.isArray(requirement.validations) && Array.isArray(parsedRequirement.validation)) {
         requirement.validations.forEach((validation, idx) => {
           if (parsedRequirement.validation[idx]) {
             parsedRequirement.validation[idx].status = validation.status;
           }
         });
       }

       // Build sync metadata (ALWAYS, regardless of status changes)
       const evidenceLookup = buildEvidenceLookup(requirement.liveEvidence || []);
       const previousRequirementMetadata = (requirementMeta && requirementMeta.originalSyncMetadata)
         || parsedRequirement._sync_metadata
         || null;

       const nextRequirementMetadata = buildRequirementSyncMetadata(
         requirement,
         previousRequirementMetadata,
         testsRunCommands,
       );

       // Store in syncData structure
       syncData.requirements[requirement.id] = {
         ...nextRequirementMetadata,
         validations: {}
       };

       // Build validation sync metadata
       if (Array.isArray(requirement.validations) && Array.isArray(parsedRequirement.validation)) {
         requirement.validations.forEach((validation, idx) => {
           const parsedValidation = parsedRequirement.validation[idx];
           if (!parsedValidation) {
             return;
           }

           const validationMeta = validation && validation.__meta ? validation.__meta.originalSyncMetadata : null;
           const previousValidationMetadata = validationMeta || parsedValidation._sync_metadata || null;
           const manualEntry = context.manualEntries ? context.manualEntries.get(requirement.id) : null;

           const nextValidationMetadata = buildValidationSyncMetadata(
             validation,
             previousValidationMetadata,
             testsRunCommands,
             evidenceLookup,
             manualEntry,
             context.manualManifestRelative,
           );

           // Store in syncData structure
           syncData.requirements[requirement.id].validations[idx] = nextValidationMetadata;
         });
       }
     });

     // ALWAYS write sync file (contains timestamps, test_names, etc.)
     writeSyncMetadataFile(filePath, syncData, context.scenarioRoot);

     // ONLY write requirement file if status changed
     if (statusChanged) {
       // Remove ALL sync metadata before writing
       removeAllSyncMetadata(parsed);

       // Remove module-level last_synced_at
       if (parsed._metadata) {
         delete parsed._metadata.last_synced_at;
       }

       const finalContent = `${JSON.stringify(parsed, null, 2)}\n`;
       fs.writeFileSync(filePath, finalContent, 'utf8');
     }

     // Capture snapshot (reads from requirement file + sync file)
     captureFileSnapshot(filePath, parsed, originalContent, context);

     return updates;
   }
   ```

4. **Update** `captureFileSnapshot()` (line 186):

   Remove sync metadata capture since it won't exist in requirement files anymore:

   ```javascript
   function captureFileSnapshot(filePath, parsed, serializedContent, context) {
     if (!context.snapshotFiles) {
       return;
     }

     const hash = crypto.createHash('sha256').update(serializedContent).digest('hex');
     let mtime = null;
     try {
       const stats = fs.statSync(filePath);
       mtime = stats.mtime.toISOString();
     } catch (error) {
       mtime = null;
     }

     const relativePath = normalizeRelativePath(context.scenarioRoot, filePath);
     const fileRequirements = Array.isArray(parsed.requirements) ? parsed.requirements : [];
     const moduleName = parsed._metadata && parsed._metadata.module ? parsed._metadata.module : null;

     // Build sync file path for reference
     const syncRelativePath = relativePath
       .replace(/^requirements\//, '')
       .replace(/\/module\.json$/, '.json');
     const syncFilePath = `coverage/sync/${syncRelativePath}`;

     const requirementSnapshots = fileRequirements.map((req) => ({
       id: req.id,
       status: req.status,
       criticality: req.criticality || null,
       prd_ref: req.prd_ref || null,
       module: moduleName,
       // NOTE: sync_metadata will be loaded separately in snapshot.js
       sync_metadata: null,  // Placeholder - filled by normalizeRequirementSnapshots
       validations: Array.isArray(req.validation)
         ? req.validation.map((validation) => ({
           type: validation.type || null,
           ref: validation.ref || null,
           status: validation.status || null,
           phase: validation.phase || null,
           workflow_id: validation.workflow_id || null,
           // NOTE: sync_metadata will be loaded separately in snapshot.js
           sync_metadata: null,  // Placeholder - filled by normalizeRequirementSnapshots
         }))
         : [],
     }));

     context.snapshotFiles.push({
       relative_path: relativePath,
       absolute_path: filePath,  // NEW: Needed by loadSyncMetadataForModule
       sync_file: syncFilePath,  // NEW: Track corresponding sync file
       hash,
       mtime,
       module: moduleName,
       requirement_count: requirementSnapshots.length,
       requirements: requirementSnapshots,
     });
   }
   ```

### Phase 2: Update Snapshot Generation

**File**: `scripts/requirements/lib/snapshot.js`

**Changes**:

1. **Add new function** `loadSyncMetadataForModule()`:
   ```javascript
   /**
    * Load sync metadata from coverage/sync/ directory
    * @param {string} moduleFilePath - Absolute path to requirement module file
    * @param {string} scenarioRoot - Scenario root directory
    * @returns {object} Sync metadata object with requirements map
    */
   function loadSyncMetadataForModule(moduleFilePath, scenarioRoot) {
     const relativePath = path.relative(
       path.join(scenarioRoot, 'requirements'),
       moduleFilePath
     );

     // Transform path to match sync file naming
     let syncFileName = relativePath
       .replace(/^requirements\//, '')
       .replace(/\/module\.json$/, '.json');

     const syncFilePath = path.join(scenarioRoot, 'coverage', 'sync', syncFileName);

     if (!fs.existsSync(syncFilePath)) {
       // No sync file yet (first run or deleted) - return empty structure
       return { requirements: {} };
     }

     try {
       const content = fs.readFileSync(syncFilePath, 'utf8');
       const parsed = JSON.parse(content);

       // Validate structure
       if (!parsed || typeof parsed !== 'object') {
         console.warn(`Invalid sync file structure: ${syncFilePath}`);
         return { requirements: {} };
       }

       if (!parsed.requirements || typeof parsed.requirements !== 'object') {
         console.warn(`Sync file missing requirements object: ${syncFilePath}`);
         return { requirements: {} };
       }

       return parsed;
     } catch (error) {
       console.error(`Unable to load sync metadata from ${syncFilePath}: ${error.message}`);
       return { requirements: {} };
     }
   }
   ```

2. **Update** `normalizeRequirementSnapshots()` (around line 52):
   ```javascript
   function normalizeRequirementSnapshots(fileSnapshots, scenarioRoot) {
     const requirementMap = {};
     if (!Array.isArray(fileSnapshots)) {
       return requirementMap;
     }

     fileSnapshots.forEach((file) => {
       // Load sync metadata from coverage/sync/ (NEW)
       const syncData = loadSyncMetadataForModule(file.absolute_path, scenarioRoot);

       const requirements = Array.isArray(file.requirements) ? file.requirements : [];
       requirements.forEach((req) => {
         if (!req || !req.id) return;

         // Get sync metadata from loaded sync file (not from requirement object)
         const reqSyncMeta = syncData.requirements && syncData.requirements[req.id]
           ? syncData.requirements[req.id]
           : null;

         requirementMap[req.id] = {
           id: req.id,
           status: req.status || 'pending',
           criticality: req.criticality || null,
           prd_ref: req.prd_ref || null,
           module: req.module || file.module || null,
           file: file.relative_path,
           sync_metadata: reqSyncMeta,  // Now from coverage/sync/
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
                   sync_metadata: valSyncMeta,  // Now from coverage/sync/
                 };
               })
             : [],
         };
       });
     });
     return requirementMap;
   }
   ```

### Phase 3: Update Schema (Remove Legacy References)

**File**: `scripts/requirements/schema.json`

**Change**: Remove `_sync_metadata` from schema entirely (clean break - no deprecation):

```json
{
  "requirement": {
    "type": "object",
    "properties": {
      "id": { "type": "string" },
      "status": { "type": "string" },
      "title": { "type": "string" },
      // ... other properties ...
      // REMOVED: "_sync_metadata" property
    }
  },
  "validation": {
    "type": "object",
    "properties": {
      "type": { "type": "string" },
      "status": { "type": "string" },
      // ... other properties ...
      // REMOVED: "_sync_metadata" property
    }
  }
}
```

**File**: `docs/testing/reference/requirement-schema.md`

Update documentation to reflect new architecture:

```markdown
## Sync Metadata Storage

As of November 2025, sync metadata (test run timestamps, test names, durations, etc.) is stored separately from requirement files in `coverage/sync/*.json` files.

### Rationale

Requirement files (`requirements/**/*.json`) are git-tracked and should only contain semantic state (status, descriptions, etc.). Ephemeral test metadata belongs in the gitignored `coverage/` directory.

### Storage Location

- **Requirement files**: `requirements/01-module-name/module.json`
- **Sync metadata**: `coverage/sync/01-module-name.json`

### What Lives Where

**In requirement files** (git-tracked):
- Requirement IDs, titles, descriptions
- Status fields (updated only when tests pass/fail)
- PRD references, criticality, dependencies
- Validation definitions (type, ref, phase)

**In sync metadata files** (gitignored):
- Test run timestamps
- Test execution durations
- Test names that cover each requirement
- Phase result file references

See [Sync Metadata Schema](./sync-metadata-schema.md) for the complete sync file format.
```

### Phase 4: Migration Script

**File**: `scripts/requirements/migrate-remove-sync-metadata.js` (NEW)

```javascript
#!/usr/bin/env node
'use strict';

/**
 * One-time migration script to remove _sync_metadata from all requirement files
 * Run this BEFORE deploying the new sync logic
 */

const fs = require('node:fs');
const path = require('node:path');
const discovery = require('./lib/discovery');

function parseArgs(argv) {
  return {
    scenario: argv.find((arg, i) => argv[i - 1] === '--scenario') || '',
    dryRun: argv.includes('--dry-run'),
    verify: argv.includes('--verify'),
  };
}

function removeSyncMetadata(parsed) {
  let changed = false;

  // Remove module-level last_synced_at
  if (parsed._metadata && parsed._metadata.last_synced_at) {
    delete parsed._metadata.last_synced_at;
    changed = true;
  }

  // Remove requirement-level and validation-level _sync_metadata
  if (Array.isArray(parsed.requirements)) {
    parsed.requirements.forEach(req => {
      if (req._sync_metadata) {
        delete req._sync_metadata;
        changed = true;
      }

      // NOTE: Property name is 'validation' (singular) in JSON
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

function verifyCleanFile(parsed, filePath) {
  const issues = [];

  if (parsed._metadata && parsed._metadata.last_synced_at) {
    issues.push('Module-level _metadata.last_synced_at still present');
  }

  if (Array.isArray(parsed.requirements)) {
    parsed.requirements.forEach((req, reqIdx) => {
      if (req._sync_metadata) {
        issues.push(`Requirement[${reqIdx}] (${req.id}) has _sync_metadata`);
      }

      if (Array.isArray(req.validation)) {
        req.validation.forEach((val, valIdx) => {
          if (val._sync_metadata) {
            issues.push(`Requirement[${reqIdx}].validation[${valIdx}] has _sync_metadata`);
          }
        });
      }
    });
  }

  if (issues.length > 0) {
    console.error(`\n❌ Verification failed for ${filePath}:`);
    issues.forEach(issue => console.error(`   - ${issue}`));
    return false;
  }

  return true;
}

function main() {
  const options = parseArgs(process.argv.slice(2));

  if (!options.scenario) {
    throw new Error('Missing --scenario argument. Usage: node migrate-remove-sync-metadata.js --scenario <name> [--dry-run] [--verify]');
  }

  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const files = discovery.collectRequirementFiles(scenarioRoot);

  console.log(`\n📋 Found ${files.length} requirement files in ${options.scenario}\n`);

  let modifiedCount = 0;
  let verificationFailed = false;
  let errorCount = 0;

  files.forEach(file => {
    let content, parsed;

    try {
      content = fs.readFileSync(file.path, 'utf8');
    } catch (error) {
      console.error(`❌ Failed to read ${file.relative}: ${error.message}`);
      errorCount++;
      return;
    }

    try {
      parsed = JSON.parse(content);
    } catch (error) {
      console.error(`❌ Failed to parse ${file.relative}: ${error.message}`);
      errorCount++;
      return;
    }

    const changed = removeSyncMetadata(parsed);

    if (changed) {
      modifiedCount++;

      if (options.dryRun) {
        console.log(`[DRY RUN] Would clean: ${file.relative}`);
      } else {
        console.log(`Cleaning: ${file.relative}`);
        try {
          fs.writeFileSync(file.path, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
        } catch (error) {
          console.error(`❌ Failed to write ${file.relative}: ${error.message}`);
          errorCount++;
          return;
        }
      }
    }

    // Verify if requested
    if (options.verify && !options.dryRun) {
      if (!verifyCleanFile(parsed, file.relative)) {
        verificationFailed = true;
      }
    }
  });

  console.log(`\n✅ Total files ${options.dryRun ? 'that would be modified' : 'modified'}: ${modifiedCount}/${files.length}\n`);

  if (errorCount > 0) {
    throw new Error(`Encountered ${errorCount} error(s) during migration`);
  }

  if (verificationFailed) {
    throw new Error('Verification failed - some files still contain sync metadata');
  }

  if (options.dryRun) {
    console.log('💡 Run without --dry-run to apply changes\n');
  } else {
    console.log('✨ Migration complete! Run tests to regenerate sync metadata in coverage/sync/\n');
  }
}

try {
  main();
} catch (error) {
  console.error(`\n❌ Migration failed: ${error.message}\n`);
  process.exitCode = 1;
}
```

**Usage**:
```bash
# Preview changes
node scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager --dry-run

# Apply migration
node scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager

# Apply and verify
node scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager --verify

# Migrate all scenarios
for scenario in scenarios/*/; do
  name=$(basename "$scenario")
  if [ -d "$scenario/requirements" ]; then
    echo "Migrating $name..."
    node scripts/requirements/migrate-remove-sync-metadata.js --scenario "$name"
  fi
done
```

### Phase 5: Testing & Validation

**Pre-migration verification**:
```bash
# Verify current behavior (baseline)
cd scenarios/landing-manager

# Run tests and capture git status
make test
git status requirements/ > /tmp/before-migration.txt

# Should show many modified files
wc -l /tmp/before-migration.txt
# Expected: 10+ modified files
```

**Migration execution**:
```bash
# Run migration script
node ../../scripts/requirements/migrate-remove-sync-metadata.js --scenario landing-manager --verify

# Verify all sync metadata removed
grep -r "_sync_metadata" requirements/
# Should return nothing

grep -r "last_synced_at" requirements/
# Should return nothing (except in descriptions/notes as text)

# Commit cleaned files
git add requirements/
git commit -m "Remove sync metadata from requirement files

Preparation for sync metadata separation. All ephemeral test data
will now live in coverage/sync/*.json (gitignored).
"
```

**Post-migration testing**:
```bash
# Deploy new sync logic (Phases 1-3 implemented)

# Clean coverage to start fresh
rm -rf coverage/

# First test run
make test

# Verify sync files created
ls -la coverage/sync/
# Should see: 01-template-management.json, 02-admin-portal.json, etc.

# Verify requirement files have NO _sync_metadata
grep -r "_sync_metadata" requirements/
# Should return nothing

# Check git status
git status requirements/
# Should show: nothing to commit, working tree clean

# Second test run (no changes) - THE CRITICAL TEST
make test

# Verify STILL clean (this is the goal!)
git status requirements/
# Should show: nothing to commit, working tree clean

# Verify sync files were updated (timestamps changed)
ls -lt coverage/sync/ | head -5
# Should show recent modification times

# Verify snapshot contains sync metadata
cat coverage/requirements-sync/latest.json | jq '.requirements | to_entries | .[0].value.sync_metadata'
# Should show sync metadata loaded from coverage/sync/

# Verify drift detection still works
node ../../scripts/requirements/drift-check.js --scenario landing-manager
# Should show: "status": "ok"

# Test drift-check with missing sync files (edge case)
rm -rf coverage/sync/
node ../../scripts/requirements/drift-check.js --scenario landing-manager
# Should handle gracefully - no crashes (sync_metadata will be null)
# Regenerate sync files
make test
```

**Status change testing**:
```bash
# Modify a test to fail
# Edit test/cli/template-management.bats - make one test fail

# Run tests
make test

# Verify requirement file WAS modified (status changed to 'failing')
git diff requirements/01-template-management/module.json
# Should show status change from 'implemented' to 'failing'

# Fix the test
# Edit test/cli/template-management.bats - restore test

# Run tests
make test

# Verify requirement file WAS modified again (status back to 'implemented')
git diff requirements/01-template-management/module.json
# Should show status change from 'failing' to 'implemented'
```

**Comprehensive validation checklist**:
- [ ] Sync files created in `coverage/sync/` directory
- [ ] Sync files contain all expected metadata (timestamps, test_names, etc.)
- [ ] Requirement files have ZERO `_sync_metadata` blocks
- [ ] Requirement files have NO `_metadata.last_synced_at`
- [ ] Running tests twice with no changes leaves git clean
- [ ] Changing test status DOES modify requirement files
- [ ] Snapshot (`coverage/requirements-sync/latest.json`) contains sync metadata
- [ ] Drift-check command works correctly
- [ ] Drift-check handles missing sync files gracefully (edge case)
- [ ] Missing sync files handled gracefully (first run)
- [ ] Corrupted sync files don't crash sync process
- [ ] `index.json` files handled correctly (if present in scenario)
- [ ] Module-level sync files properly created for all requirement modules

---

## Migration Timeline

**Estimated effort**: 8-9 hours total (includes additional fixes and testing)

1. **Implement Phase 1** (sync writer + helpers) - 3 hours
   - `writeSyncMetadataFile()` - 30 min
   - `removeAllSyncMetadata()` - 15 min
   - Rewrite `syncRequirementFile()` - 1.5 hours
   - Update `captureFileSnapshot()` - 30 min
   - Testing/debugging - 30 min

2. **Implement Phase 2** (snapshot reader) - 1.5 hours
   - `loadSyncMetadataForModule()` - 30 min
   - Update `normalizeRequirementSnapshots()` - 45 min
   - Testing/debugging - 15 min

3. **Implement Phase 3** (schema updates) - 30 minutes
   - Update schema.json - 15 min
   - Update requirement-schema.md - 15 min

4. **Create Phase 4** (migration script) - 1 hour
   - Write script - 30 min
   - Test on single scenario - 15 min
   - Test on all scenarios - 15 min

5. **Execute Phase 5** (testing & validation) - 2 hours
   - Pre-migration verification - 15 min
   - Run migration on all scenarios - 15 min
   - Post-migration testing - 1 hour
   - Status change testing - 30 min

6. **Documentation** - 30 minutes
   - Update REQUIREMENT_FLOW.md
   - Add sync-metadata-schema.md

---

## Success Criteria

### Primary Goal
✅ Running tests twice with no code changes results in **zero** git-tracked requirement file modifications

### Secondary Goals
- ✅ All existing test commands continue working without changes
- ✅ Snapshot integrity maintained (sync metadata accessible via snapshot)
- ✅ Drift detection continues working
- ✅ Status changes still update requirement files (and ONLY status changes)
- ✅ Schema updated to reflect new architecture
- ✅ Migration script successfully cleans all scenarios
- ✅ Documentation updated and accurate

### Validation Metrics
- **Before**: ~10 requirement files modified per test run (all scenarios)
- **After**: 0 requirement files modified per test run (unless status changes)
- **Git noise reduction**: ~95% (only status changes remain)

---

## Rollback Plan

### If Issues Arise During Implementation

**Option 1: Revert code changes**
```bash
# Revert all code changes
git checkout HEAD -- scripts/requirements/

# Requirement files already committed (cleaned), so just re-run tests
cd scenarios/landing-manager && make test
# Old sync logic will regenerate _sync_metadata in files
```

**Option 2: Git-based rollback (if migration committed)**
```bash
# Find commit before migration
git log --oneline requirements/ | head -5

# Revert to previous state
git revert <commit-hash>

# Or hard reset (if not pushed)
git reset --hard <commit-hash>
```

**Risk Assessment**: **Low**
- `coverage/` directory already gitignored - worst case is losing test run metadata (easily regenerated)
- Requirement files tracked in git - can always revert
- No database schema changes or external dependencies
- Changes isolated to requirement sync system

---

## Benefits Summary

### For Developers
- ✅ Clean `git status` after test runs (no noise)
- ✅ PRs show only meaningful requirement changes
- ✅ Easier to identify actual progress vs. routine test runs
- ✅ Smaller git diffs = faster code reviews

### For AI Agents
- ✅ Clear signal when requirements advance (status changes only)
- ✅ No confusion from timestamp-only diffs
- ✅ Better understanding of test impact
- ✅ Can accurately track implementation progress

### For System Architecture
- ✅ Proper separation: semantic (git) vs. ephemeral (gitignored)
- ✅ Smaller git history (no more timestamp-only commits)
- ✅ Faster requirement file parsing (less data in files)
- ✅ More maintainable codebase (clear boundaries)

---

## Open Questions & Answers

1. **Should we keep `_metadata.last_synced_at` in snapshot?**
   - **Answer**: Yes - move it to sync files, load into snapshot from there

2. **What about index.json files?**
   - **Answer**: Same treatment - they can have requirements too

3. **Performance impact of reading separate sync files?**
   - **Answer**: Snapshot generation reads all files anyway. Adding sync file reads has minimal impact. Consider caching sync files during a single sync operation if performance becomes an issue.

4. **Should coverage/sync/ files be human-readable or compressed?**
   - **Answer**: Keep JSON pretty-printed for debugging - storage is cheap

5. **What if sync file is deleted but requirement file exists?**
   - **Answer**: System gracefully handles missing sync files (returns empty metadata). Next test run recreates them.

6. **What about backward compatibility with old requirement files?**
   - **Answer**: None needed - clean migration removes all `_sync_metadata` before deploying new code

---

## References

### Modified Files (Implementation)
- `scripts/requirements/lib/sync.js` - Core sync logic (major rewrite)
- `scripts/requirements/lib/snapshot.js` - Snapshot generation (updated)
- `scripts/requirements/schema.json` - Schema (cleaned)

### New Files
- `scripts/requirements/migrate-remove-sync-metadata.js` - One-time migration script
- `coverage/sync/*.json` - New sync metadata storage (gitignored via existing rule)

### Documentation Updates
- `docs/testing/reference/requirement-schema.md` - Schema reference (updated)
- `docs/testing/architecture/REQUIREMENT_FLOW.md` - Flow diagram (update needed)
- `docs/testing/guides/requirement-tracking.md` - Tracking guide (update needed)

### Related Documentation
- [Requirement Flow Architecture](../testing/architecture/REQUIREMENT_FLOW.md)
- [Requirement Tracking Guide](../testing/guides/requirement-tracking.md)
- [Testing Glossary](../testing/GLOSSARY.md)
- [Phased Testing Architecture](../testing/architecture/PHASED_TESTING.md)

---

## Implementation Notes

### Critical Implementation Details

1. **Property Naming**: JSON files use `validation` (singular) not `validations` (plural)
2. **Status Tracking**: Separate `statusChanged` from `metadataChanged` - this is the core fix
3. **Sync File Writes**: ALWAYS write sync files, regardless of status changes
4. **Requirement File Writes**: ONLY write when status changes
5. **Path Transformations**: Handle both `module.json` and `index.json` cases correctly

### Testing Strategy

1. Test on single scenario first (landing-manager - has comprehensive requirements)
2. Verify git cleanliness with multiple test runs
3. Test status change detection (make tests fail/pass)
4. Test on scenario with no requirements (edge case)
5. Test on scenario with manual validations
6. Migrate all scenarios only after thorough testing

### Deployment Strategy

1. **Commit cleaned requirement files FIRST** (migration script output)
2. **Deploy new sync logic SECOND** (Phases 1-3 code changes)
3. **Run tests to generate sync files THIRD** (initial sync data population)
4. **Verify git cleanliness FOURTH** (run tests again, check git status)

This order ensures no intermediate broken state.
