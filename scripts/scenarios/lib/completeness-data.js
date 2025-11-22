#!/usr/bin/env node
'use strict';

/**
 * Data collection for scenario completeness scoring
 * Gathers metrics from requirements, operational targets, and test results
 *
 * REFACTORED: This module now acts as an orchestrator, delegating to specialized modules:
 * - loaders/ - Data loading (service config, requirements, sync metadata, test results)
 * - analyzers/ - Data analysis (requirements, targets, UI metrics)
 * - utils/ - Shared utilities (file operations, pattern matching)
 *
 * @module scenarios/lib/completeness-data
 */

const path = require('node:path');

// Loaders
const { loadServiceConfig } = require('./loaders/service-loader');
const { loadRequirements } = require('./loaders/requirements-loader');
const { loadSyncMetadata } = require('./loaders/sync-loader');
const { loadTestResults } = require('./loaders/test-loader');

// Analyzers
const { calculateRequirementPass } = require('./analyzers/requirement-analyzer');
const { extractOperationalTargets, calculateTargetPass } = require('./analyzers/target-analyzer');
const { collectUIMetrics } = require('./analyzers/ui-analyzer');

/**
 * Collect all metrics for a scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Complete metrics object
 */
function collectMetrics(scenarioRoot) {
  // Load data from various sources
  const serviceConfig = loadServiceConfig(scenarioRoot);
  const requirements = loadRequirements(scenarioRoot);
  const syncData = loadSyncMetadata(scenarioRoot);
  const testResults = loadTestResults(scenarioRoot);

  // Analyze the data
  const operationalTargets = extractOperationalTargets(requirements, syncData);
  const requirementPass = calculateRequirementPass(requirements, syncData);
  const targetPass = calculateTargetPass(operationalTargets, requirements, syncData);
  const uiMetrics = collectUIMetrics(scenarioRoot);

  return {
    scenario: path.basename(scenarioRoot),
    category: serviceConfig.category || 'utility',
    requirements: requirementPass,
    targets: targetPass,
    tests: {
      total: testResults.total,
      passing: testResults.passing
    },
    lastTestRun: testResults.lastRun,
    rawRequirements: requirements,  // Include for depth calculation
    ui: uiMetrics  // Include UI metrics
  };
}

module.exports = {
  // Main entry point
  collectMetrics,

  // Re-export loaders for backward compatibility and testing
  loadServiceConfig,
  loadRequirements,
  loadSyncMetadata,
  loadTestResults,

  // Re-export analyzers for backward compatibility and testing
  calculateRequirementPass,
  extractOperationalTargets,
  calculateTargetPass
};
