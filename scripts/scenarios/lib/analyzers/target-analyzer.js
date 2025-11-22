'use strict';

/**
 * Operational target analyzer
 * Extracts and analyzes operational targets from requirements and sync metadata
 * @module scenarios/lib/analyzers/target-analyzer
 */

const { OPERATIONAL_TARGETS } = require('../utils/pattern-matchers');
const { TARGET_PASS_THRESHOLD } = require('../constants');
const { isRequirementPassing } = require('./requirement-analyzer');

/**
 * Extract operational targets from prd_ref fields (deployment-manager style)
 * @param {array} requirements - Array of requirement objects
 * @returns {Map<string, object>} Map of target ID to target object
 */
function extractFromPrdRef(requirements) {
  const targets = new Map();

  for (const req of requirements) {
    if (req.prd_ref) {
      const match = req.prd_ref.match(OPERATIONAL_TARGETS.prdRef);
      if (match) {
        const targetId = match[0].toUpperCase();
        if (!targets.has(targetId)) {
          targets.set(targetId, {
            id: targetId,
            type: 'prd_ref',
            requirements: []
          });
        }
        targets.get(targetId).requirements.push(req.id);
      }
    }
  }

  return targets;
}

/**
 * Extract operational targets from sync metadata (BAS style)
 * @param {object} syncMetadata - Sync metadata object
 * @returns {Map<string, object>} Map of target ID to target object
 */
function extractFromSyncMetadata(syncMetadata) {
  const targets = new Map();

  if (syncMetadata && syncMetadata.operational_targets && Array.isArray(syncMetadata.operational_targets)) {
    for (const ot of syncMetadata.operational_targets) {
      const targetId = ot.key || ot.folder_hint || ot.target_id;
      if (targetId) {
        targets.set(targetId, {
          id: targetId,
          type: 'folder',
          status: ot.status,
          criticality: ot.criticality,
          counts: ot.counts,
          requirements: (ot.requirements || []).map(r => r.id)
        });
      }
    }
  }

  return targets;
}

/**
 * Extract operational targets from requirements and sync metadata
 * Supports both OT-P0-001 format (deployment-manager) and folder-based format (BAS)
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncMetadata - Sync metadata object (may contain operational_targets)
 * @returns {array} Array of unique operational target objects
 */
function extractOperationalTargets(requirements, syncMetadata) {
  // Extract from both sources
  const prdTargets = extractFromPrdRef(requirements);
  const syncTargets = extractFromSyncMetadata(syncMetadata);

  // Merge targets (sync targets take precedence for duplicates)
  const allTargets = new Map([...prdTargets, ...syncTargets]);

  return Array.from(allTargets.values());
}

/**
 * Check if a folder-based target is passing
 * @param {object} target - Target object with type 'folder'
 * @returns {boolean} True if target is passing
 */
function isFolderTargetPassing(target) {
  // If target status is explicitly 'complete'
  if (target.status === 'complete') {
    return true;
  }

  // If target has counts, check completion ratio
  if (target.counts && target.counts.total > 0) {
    const completeRatio = target.counts.complete / target.counts.total;
    return completeRatio > TARGET_PASS_THRESHOLD;
  }

  return false;
}

/**
 * Check if a prd_ref-based target is passing
 * @param {object} target - Target object with type 'prd_ref'
 * @param {array} requirements - Array of all requirements
 * @param {object} syncData - Full sync metadata object
 * @returns {boolean} True if target is passing
 */
function isPrdRefTargetPassing(target, requirements, syncData) {
  const targetId = target.id;
  const syncMetadata = syncData.requirements || syncData;

  // Find all requirements linked to this target
  const linkedReqs = requirements.filter(req => {
    if (!req.prd_ref) return false;
    const match = req.prd_ref.match(OPERATIONAL_TARGETS.prdRef);
    return match && match[0].toUpperCase() === targetId;
  });

  // Target is passing if at least one linked requirement is passing
  return linkedReqs.some(req => {
    const reqMeta = syncMetadata[req.id];
    return isRequirementPassing(req, reqMeta);
  });
}

/**
 * Check if a target is passing
 * @param {object} target - Target object
 * @param {array} requirements - Array of all requirements
 * @param {object} syncData - Full sync metadata object
 * @returns {boolean} True if target is passing
 */
function isTargetPassing(target, requirements, syncData) {
  if (target.type === 'folder') {
    return isFolderTargetPassing(target);
  } else {
    return isPrdRefTargetPassing(target, requirements, syncData);
  }
}

/**
 * Calculate operational target pass status
 * Supports both OT-P0-001 format and folder-based targets
 * @param {array} targets - Array of operational target objects
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Full sync metadata object
 * @returns {object} Target pass statistics
 */
function calculateTargetPass(targets, requirements, syncData) {
  const total = targets.length;
  let passing = 0;

  for (const target of targets) {
    if (isTargetPassing(target, requirements, syncData)) {
      passing++;
    }
  }

  return { total, passing };
}

module.exports = {
  extractOperationalTargets,
  isTargetPassing,
  calculateTargetPass
};
