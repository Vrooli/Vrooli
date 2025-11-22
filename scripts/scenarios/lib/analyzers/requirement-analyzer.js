'use strict';

/**
 * Requirement analyzer
 * Analyzes requirement pass rates and status with diversity enforcement
 * @module scenarios/lib/analyzers/requirement-analyzer
 */

const { PASSING_REQUIREMENT_STATUSES, PASSING_SYNC_STATUSES } = require('../constants');
const componentDetector = require('../validators/component-detector');
const layerDetector = require('../validators/layer-detector');

/**
 * Check if a requirement is considered passing (legacy function without diversity checks)
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
 * Calculate requirement pass with diversity enforcement
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Sync metadata
 * @param {string} scenarioRoot - Scenario root for component detection
 * @returns {object} Pass statistics with diversity metadata
 */
function calculateRequirementPass(requirements, syncData, scenarioRoot) {
  let passing = 0;
  let total = requirements.length;
  const syncMetadata = syncData.requirements || syncData;

  // If scenarioRoot not provided, fall back to legacy behavior
  if (!scenarioRoot) {
    for (const req of requirements) {
      const reqMeta = syncMetadata[req.id];
      if (isRequirementPassing(req, reqMeta)) {
        passing++;
      }
    }
    return { total, passing };
  }

  // Detect available components once for the scenario
  const scenarioComponents = componentDetector.detectScenarioComponents(scenarioRoot);
  const applicableLayers = componentDetector.getApplicableLayers(scenarioComponents);

  for (const req of requirements) {
    const reqMeta = syncMetadata[req.id];

    // Extract criticality from prd_ref (OT-P0-XXX → P0)
    const criticality = layerDetector.deriveRequirementCriticality(req);
    const status = req.status || 'pending';

    // Detect validation layers (returns {automated: Set, has_manual: boolean})
    const layerAnalysis = layerDetector.detectValidationLayers(req, scenarioRoot);

    // Filter to only applicable AUTOMATED layers based on scenario components
    const applicableAutomatedLayers = new Set();
    if (scenarioComponents.has('API') && layerAnalysis.automated.has('API')) {
      applicableAutomatedLayers.add('API');
    }
    if (scenarioComponents.has('UI') && layerAnalysis.automated.has('UI')) {
      applicableAutomatedLayers.add('UI');
    }
    // E2E is always applicable (AUTOMATED layer)
    if (layerAnalysis.automated.has('E2E')) {
      applicableAutomatedLayers.add('E2E');
    }
    // MANUAL is tracked separately, NOT counted toward diversity

    // Minimum AUTOMATED layer requirement based on criticality and available components
    // For API-only: Need API test + E2E
    // For UI-only: Need UI test + E2E
    // For Full-stack: Need (API + UI) OR (API + E2E) OR (UI + E2E)
    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;

    // Requirement passes if:
    // 1. Status indicates completion (validated/implemented/complete)
    // 2. Sync metadata shows completion
    // 3. Has sufficient AUTOMATED validation layer diversity (applicable layers only)
    const statusPasses = (
      status === 'validated' ||
      status === 'implemented' ||
      status === 'complete'
    );

    const syncPasses = reqMeta && (
      reqMeta.status === 'complete' ||
      reqMeta.status === 'validated' ||
      (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true)
    );

    const diversityPasses = applicableAutomatedLayers.size >= minLayers;

    // All three conditions must be true
    if ((statusPasses || syncPasses) && diversityPasses) {
      passing++;
    }
  }

  return { total, passing };
}

module.exports = {
  isRequirementPassing,
  calculateRequirementPass
};
