#!/usr/bin/env node
'use strict';

/**
 * Requirement registry synchronization operations
 * @module requirements/lib/sync
 */

const fs = require('node:fs');
const path = require('node:path');
const { deriveValidationStatus, deriveRequirementStatus } = require('./utils');
const { extractVitestFilesFromPhaseResults } = require('./evidence');

/**
 * Sync requirement file with live test results
 * @param {string} filePath - Path to requirement file
 * @param {import('./types').Requirement[]} requirements - Requirements to sync
 * @returns {import('./types').SyncUpdate[]} Array of updates made
 */
function syncRequirementFile(filePath, requirements) {
  if (!requirements || requirements.length === 0) {
    return [];
  }

  // Read the original JSON file
  const originalContent = fs.readFileSync(filePath, 'utf8');
  const parsed = JSON.parse(originalContent);
  const updates = [];

  // Create a map of requirement IDs to their index in the file
  const requirementMap = new Map();
  (parsed.requirements || []).forEach((req, idx) => {
    requirementMap.set(req.id, idx);
  });

  requirements.forEach((requirement) => {
    const requirementMeta = requirement.__meta;
    const validationStatuses = [];
    const originalStatus = requirementMeta && typeof requirementMeta.originalStatus === 'string'
      ? requirementMeta.originalStatus
      : requirement.status;

    // Update validation statuses
    if (Array.isArray(requirement.validations)) {
      requirement.validations.forEach((validation, index) => {
        const derivedStatus = deriveValidationStatus(validation);
        validationStatuses.push(derivedStatus);

        if (derivedStatus && derivedStatus !== validation.status) {
          validation.status = derivedStatus;
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

    // Update requirement status based on validation rollup
    const nextStatus = deriveRequirementStatus(requirement.status, validationStatuses);
    if (nextStatus && nextStatus !== originalStatus) {
      requirement.status = nextStatus;
      if (requirementMeta) {
        requirementMeta.originalStatus = nextStatus;
      }
      updates.push({
        type: 'requirement',
        requirement: requirement.id,
        status: nextStatus,
        file: filePath,
      });
    }

    // Update the parsed JSON with new statuses
    const reqIndex = requirementMap.get(requirement.id);
    if (reqIndex !== undefined && parsed.requirements[reqIndex]) {
      parsed.requirements[reqIndex].status = requirement.status;

      // Update validation statuses in parsed JSON
      if (Array.isArray(requirement.validations) && Array.isArray(parsed.requirements[reqIndex].validation)) {
        requirement.validations.forEach((validation, idx) => {
          if (parsed.requirements[reqIndex].validation[idx]) {
            parsed.requirements[reqIndex].validation[idx].status = validation.status;
          }
        });
      }
    }
  });

  // Write back to file if there were updates
  if (updates.length > 0) {
    // Update metadata
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    // Write with 2-space indentation
    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

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
function syncRequirementRegistry(fileRequirementMap, scenarioRoot, options) {
  const updates = [];
  const addedValidations = [];
  const orphanedValidations = [];
  const removedValidations = [];

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
    updates.push(...syncRequirementFile(filePath, requirements));
  }

  return {
    statusUpdates: updates,
    addedValidations,
    orphanedValidations,
    removedValidations,
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
