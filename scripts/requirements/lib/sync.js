#!/usr/bin/env node
'use strict';

/**
 * Requirement registry synchronization operations
 * @module requirements/lib/sync
 */

const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { deriveValidationStatus, deriveRequirementStatus } = require('./utils');
const { extractVitestFilesFromPhaseResults } = require('./evidence');

function normalizeTestCommands(commands) {
  if (!Array.isArray(commands)) {
    return [];
  }
  return commands
    .map((cmd) => (typeof cmd === 'string' ? cmd.trim() : ''))
    .filter((cmd) => cmd.length > 0);
}

function metadataEqual(previous, next) {
  const prevString = JSON.stringify(previous || null);
  const nextString = JSON.stringify(next || null);
  return prevString === nextString;
}

function buildEvidenceLookup(records) {
  const lookup = new Map();
  if (!Array.isArray(records)) {
    return lookup;
  }
  records.forEach((record) => {
    if (!record || !record.phase) {
      return;
    }
    const existing = lookup.get(record.phase);
    if (!existing) {
      lookup.set(record.phase, record);
      return;
    }
    const existingTime = existing.updated_at ? Date.parse(existing.updated_at) : 0;
    const recordTime = record.updated_at ? Date.parse(record.updated_at) : 0;
    if (recordTime >= existingTime) {
      lookup.set(record.phase, record);
    }
  });
  return lookup;
}

function buildRequirementSyncMetadata(requirement, existingMetadata, testsRunCommands) {
  const evidenceRecords = Array.isArray(requirement.liveEvidence)
    ? requirement.liveEvidence.filter(Boolean)
    : [];
  let newestTimestamp = existingMetadata && existingMetadata.last_updated ? existingMetadata.last_updated : null;
  const phaseResults = new Set();

  evidenceRecords.forEach((record) => {
    if (record.source_path) {
      phaseResults.add(record.source_path);
    }
    if (record.updated_at) {
      if (!newestTimestamp || Date.parse(record.updated_at) > Date.parse(newestTimestamp)) {
        newestTimestamp = record.updated_at;
      }
    }
  });

  const normalizedCommands = testsRunCommands.length
    ? [...testsRunCommands]
    : (existingMetadata && Array.isArray(existingMetadata.tests_run) ? [...existingMetadata.tests_run] : []);

  const implementedCount = Array.isArray(requirement.validations)
    ? requirement.validations.filter((validation) => validation && validation.status === 'implemented').length
    : 0;
  const hasFailingValidation = Array.isArray(requirement.validations)
    ? requirement.validations.some((validation) => validation && validation.status === 'failing')
    : false;

  const metadata = {
    last_updated: newestTimestamp || (existingMetadata && existingMetadata.last_updated) || new Date().toISOString(),
    updated_by: 'auto-sync',
    test_coverage_count: implementedCount,
    all_tests_passing: !hasFailingValidation,
    phase_results: phaseResults.size
      ? Array.from(phaseResults).sort()
      : (existingMetadata && Array.isArray(existingMetadata.phase_results) ? existingMetadata.phase_results : []),
    tests_run: normalizedCommands,
  };

  return metadata;
}

function buildValidationSyncMetadata(validation, existingMetadata, testsRunCommands, evidenceLookup, manualEntry, manualManifestRelative) {
  const prevMeta = existingMetadata || {};
  const normalizedCommands = testsRunCommands.length
    ? [...testsRunCommands]
    : (Array.isArray(prevMeta.tests_run) ? [...prevMeta.tests_run] : []);

  const metadata = {
    last_test_run: prevMeta.last_test_run || null,
    test_duration_ms: typeof prevMeta.test_duration_ms === 'number' ? prevMeta.test_duration_ms : null,
    auto_updated: prevMeta.auto_updated || false,
    test_names: Array.isArray(prevMeta.test_names) ? prevMeta.test_names : [],
    phase_result: prevMeta.phase_result || null,
    source_phase: prevMeta.source_phase || null,
    tests_run: normalizedCommands,
  };

  const liveDetails = validation && validation.liveDetails ? validation.liveDetails : null;
  if (liveDetails && liveDetails.updated_at) {
    metadata.last_test_run = liveDetails.updated_at;
  }
  if (liveDetails && typeof liveDetails.duration_seconds === 'number') {
    metadata.test_duration_ms = Math.round(liveDetails.duration_seconds * 1000);
  }

  let sourcePhase = metadata.source_phase;
  if (validation && validation.liveSource && validation.liveSource.kind === 'phase' && validation.liveSource.name) {
    sourcePhase = validation.liveSource.name;
  } else if (validation && validation.phase) {
    sourcePhase = validation.phase;
  } else if (liveDetails && liveDetails.requirement && liveDetails.requirement.phase) {
    sourcePhase = liveDetails.requirement.phase;
  }

  if (sourcePhase) {
    metadata.source_phase = sourcePhase;
    const evidenceRecord = evidenceLookup.get(sourcePhase);
    if (evidenceRecord && evidenceRecord.source_path) {
      metadata.phase_result = evidenceRecord.source_path;
    }
  }

  if (liveDetails || sourcePhase) {
    metadata.auto_updated = true;
  }

  if (validation && validation.type === 'manual') {
    if (manualEntry) {
      metadata.manual = {
        status: manualEntry.status || 'unknown',
        validated_at: manualEntry.validated_at || manualEntry.recorded_at || null,
        validated_by: manualEntry.validated_by || null,
        expires_at: manualEntry.expires_at || null,
        artifact_path: manualEntry.artifact_path || null,
        manifest_path: manualManifestRelative || manualEntry.manifest_path || null,
        notes: manualEntry.notes || null,
      };
      if (!metadata.last_test_run && metadata.manual.validated_at) {
        metadata.last_test_run = metadata.manual.validated_at;
      }
      metadata.auto_updated = true;
    } else if (metadata.manual) {
      metadata.manual = null;
    }
  } else if (metadata.manual) {
    metadata.manual = null;
  }

  return metadata;
}

/**
 * Sync requirement file with live test results
 * @param {string} filePath - Path to requirement file
 * @param {import('./types').Requirement[]} requirements - Requirements to sync
 * @returns {import('./types').SyncUpdate[]} Array of updates made
 */
function computeContentHash(content) {
  if (typeof content !== 'string') {
    return null;
  }
  return crypto.createHash('sha256').update(content, 'utf8').digest('hex');
}

function normalizeRelativePath(baseDir, filePath) {
  if (!baseDir) {
    return filePath;
  }
  return path.relative(baseDir, filePath) || filePath;
}

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

function captureFileSnapshot(filePath, parsed, serializedContent, context) {
  if (!context || !Array.isArray(context.snapshotFiles)) {
    return;
  }

  const effectiveContent = typeof serializedContent === 'string'
    ? serializedContent
    : `${JSON.stringify(parsed, null, 2)}\n`;
  const hash = computeContentHash(effectiveContent);
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

/**
 * Add missing vitest validation entries
 * @param {string} filePath - Path to requirements JSON file
 * @param {import('./types').Requirement[]} requirements - Requirements to check
 * @param {Map<string, import('./types').VitestFile[]>} vitestFiles - Vitest files from evidence
 * @param {string} scenarioRoot - Scenario directory path
 * @returns {import('./types').SyncUpdate[]} Array of changes made
 */
function addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot) {
  const changes = [];

  // Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return changes;
  }

  let modified = false;

  parsed.requirements.forEach((requirement) => {
    const liveTestFiles = vitestFiles.get(requirement.id) || [];

    if (liveTestFiles.length === 0) {
      return; // No vitest evidence for this requirement
    }

    // Ensure validation array exists
    if (!Array.isArray(requirement.validation)) {
      requirement.validation = [];
    }

    // Build set of existing vitest refs
    const existingRefs = new Set(
      requirement.validation
        .filter(v => v.type === 'test' && v.ref && v.ref.startsWith('ui/src/'))
        .map(v => v.ref)
    );

    // Add missing validations
    liveTestFiles.forEach(testFile => {
      if (existingRefs.has(testFile.ref)) {
        return; // Already exists
      }

      const newValidation = {
        type: 'test',
        ref: testFile.ref,
        phase: testFile.phase,
        status: testFile.status,
        notes: 'Auto-added from vitest evidence',
      };

      requirement.validation.push(newValidation);
      modified = true;

      changes.push({
        type: 'add_validation',
        requirement: requirement.id,
        validation: testFile.ref,
        status: testFile.status,
      });
    });
  });

  // Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return changes;
}

/**
 * Detect and optionally remove orphaned vitest validation entries
 * @param {string} filePath - Path to requirements JSON file
 * @param {string} scenarioRoot - Scenario directory
 * @param {import('./types').ParseOptions} options - Sync options (pruneStale flag)
 * @returns {{ orphaned: import('./types').OrphanedValidation[], removed: import('./types').OrphanedValidation[] }} Orphaned and removed validations
 */
function detectOrphanedValidations(filePath, scenarioRoot, options) {
  const orphaned = [];
  const removed = [];

  // Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return { orphaned, removed };
  }

  let modified = false;

  parsed.requirements.forEach(requirement => {
    if (!Array.isArray(requirement.validation)) {
      return;
    }

    const validValidations = [];

    requirement.validation.forEach(validation => {
      // PRESERVE all non-test validations (automation, manual, etc.)
      if (validation.type !== 'test') {
        validValidations.push(validation);
        return;
      }

      // PRESERVE Go/Python tests (not vitest scope)
      const ref = validation.ref || '';
      if (!ref.startsWith('ui/src/') || !ref.match(/\.test\.(ts|tsx)$/)) {
        validValidations.push(validation);
        return;
      }

      // Check if vitest test file exists
      const exists = fs.existsSync(path.join(scenarioRoot, ref));

      if (exists) {
        validValidations.push(validation);
      } else {
        // Orphaned vitest validation found!
        orphaned.push({
          requirement: requirement.id,
          ref,
          phase: validation.phase,
          file: filePath,
        });

        if (options.pruneStale) {
          removed.push({
            requirement: requirement.id,
            ref,
            file: filePath,
          });
          modified = true;
          // Don't push to validValidations (removes it)
        } else {
          // Keep but mark as orphaned
          validValidations.push(validation);
        }
      }
    });

    // Replace validation array if pruning
    if (modified && options.pruneStale) {
      requirement.validation = validValidations;
    }
  });

  // Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return { orphaned, removed };
}

/**
 * Sync entire requirement registry with live test results
 * @param {Map<string, import('./types').Requirement[]>} fileRequirementMap - Map of file paths to requirements
 * @param {string} scenarioRoot - Scenario root directory
 * @param {import('./types').ParseOptions} options - Sync options
 * @returns {import('./types').SyncResult} Sync results
 */
function syncRequirementRegistry(fileRequirementMap, scenarioRoot, options, extras = {}) {
  const updates = [];
  const addedValidations = [];
  const orphanedValidations = [];
  const removedValidations = [];
  const manualManifest = extras && extras.manualManifest ? extras.manualManifest : null;
  const manualEntries = manualManifest && manualManifest.latestByRequirement
    ? manualManifest.latestByRequirement
    : new Map();
  const manualManifestPath = manualManifest && manualManifest.manifestPath ? manualManifest.manifestPath : null;
  const manualManifestRelative = manualManifest && manualManifest.relativePath
    ? manualManifest.relativePath
    : (manualManifestPath ? normalizeRelativePath(scenarioRoot, manualManifestPath) : null);
  const syncContext = {
    testCommands: normalizeTestCommands(options && options.testCommands ? options.testCommands : []),
    snapshotFiles: [],
    scenarioRoot,
    manualEntries,
    manualManifestRelative,
  };

  // Extract live vitest evidence once (from phase-results)
  const vitestFiles = extractVitestFilesFromPhaseResults(scenarioRoot);

  for (const [filePath, requirements] of fileRequirementMap.entries()) {
    // Phase 1: Detect and optionally remove orphaned validations
    const orphanResult = detectOrphanedValidations(filePath, scenarioRoot, options);
    orphanedValidations.push(...orphanResult.orphaned);
    removedValidations.push(...orphanResult.removed);

    // Phase 2: Add missing validations from live evidence
    const added = addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot);
    addedValidations.push(...added);

    // Phase 3: Update status fields (existing logic)
    updates.push(...syncRequirementFile(filePath, requirements, syncContext));
  }

  return {
    statusUpdates: updates,
    addedValidations,
    orphanedValidations,
    removedValidations,
    fileSnapshots: syncContext.snapshotFiles,
  };
}

/**
 * Print sync summary to console
 * @param {import('./types').SyncResult} syncResult - Result from syncRequirementRegistry
 */
function printSyncSummary(syncResult) {
  console.log('\n📋 Requirements Sync Report:\n');

  // Status updates (existing)
  if (syncResult.statusUpdates.length > 0) {
    console.log('✏️  Status Updates:');
    syncResult.statusUpdates.forEach(update => {
      if (update.type === 'requirement') {
        console.log(`   ${update.requirement}: status → ${update.status}`);
      }
    });
    console.log();
  }

  // Added validations (new)
  if (syncResult.addedValidations.length > 0) {
    console.log('✅ Added Validations:');
    syncResult.addedValidations.forEach(added => {
      console.log(`   ${added.requirement}: + ${added.validation}`);
    });
    console.log();
  }

  // Orphaned validations (new)
  if (syncResult.orphanedValidations.length > 0) {
    console.log('⚠️  Orphaned Validations (file not found):');
    syncResult.orphanedValidations.forEach(orphan => {
      console.log(`   ${orphan.requirement}: × ${orphan.ref}`);
    });
    console.log();

    if (!syncResult.removedValidations.length) {
      console.log('   💡 Use --prune-stale to remove orphaned validations\n');
    }
  }

  // Removed validations (new)
  if (syncResult.removedValidations.length > 0) {
    console.log('🗑️  Removed Orphaned Validations:');
    syncResult.removedValidations.forEach(removed => {
      console.log(`   ${removed.requirement}: - ${removed.ref}`);
    });
    console.log();
  }

  // Summary
  const totalChanges = syncResult.statusUpdates.length +
                      syncResult.addedValidations.length +
                      syncResult.removedValidations.length;

  if (totalChanges === 0) {
    console.log('✅ No changes needed - all requirements are up to date\n');
  } else {
    console.log(`✅ Sync complete: ${totalChanges} total changes\n`);
  }
}

module.exports = {
  syncRequirementFile,
  addMissingValidations,
  detectOrphanedValidations,
  syncRequirementRegistry,
  printSyncSummary,
};
