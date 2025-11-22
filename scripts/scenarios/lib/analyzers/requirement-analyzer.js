'use strict';

/**
 * Requirement analyzer
 * Analyzes requirement pass rates and status
 * @module scenarios/lib/analyzers/requirement-analyzer
 */

const { PASSING_REQUIREMENT_STATUSES, PASSING_SYNC_STATUSES } = require('../constants');

/**
 * Check if a requirement is considered passing
 * @param {object} requirement - Requirement object
 * @param {object} syncMetadata - Sync metadata for this requirement
 * @returns {boolean} True if requirement is passing
 */
function isRequirementPassing(requirement, syncMetadata) {
  // Check requirement's own status
  if (requirement.status && PASSING_REQUIREMENT_STATUSES.includes(requirement.status)) {
    return true;
  }

  // Check sync metadata status
  if (syncMetadata) {
    if (syncMetadata.status && PASSING_SYNC_STATUSES.includes(syncMetadata.status)) {
      return true;
    }

    // Check if all tests are passing
    if (syncMetadata.sync_metadata && syncMetadata.sync_metadata.all_tests_passing === true) {
      return true;
    }
  }

  return false;
}

/**
 * Calculate requirement pass status from sync metadata
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Full sync metadata object
 * @returns {object} Requirement pass statistics
 */
function calculateRequirementPass(requirements, syncData) {
  let passing = 0;
  const total = requirements.length;

  const syncMetadata = syncData.requirements || syncData;

  for (const req of requirements) {
    const reqMeta = syncMetadata[req.id];
    if (isRequirementPassing(req, reqMeta)) {
      passing++;
    }
  }

  return { total, passing };
}

module.exports = {
  isRequirementPassing,
  calculateRequirementPass
};
