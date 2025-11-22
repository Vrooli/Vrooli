#!/usr/bin/env node
'use strict';

/**
 * Data collection for scenario completeness scoring
 * Gathers metrics from requirements, operational targets, and test results
 * @module scenarios/lib/completeness-data
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Load service.json to get scenario category
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Service configuration
 */
function loadServiceConfig(scenarioRoot) {
  const servicePath = path.join(scenarioRoot, '.vrooli', 'service.json');
  try {
    const content = fs.readFileSync(servicePath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    console.warn(`Unable to load service.json from ${servicePath}: ${error.message}`);
    return { category: 'utility' };  // Default category
  }
}

/**
 * Load requirements from index.json or module.json files
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {array} Array of requirement objects
 */
function loadRequirements(scenarioRoot) {
  const requirementsDir = path.join(scenarioRoot, 'requirements');

  if (!fs.existsSync(requirementsDir)) {
    return [];
  }

  const requirements = [];

  // Try to load from index.json first
  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    try {
      const content = fs.readFileSync(indexPath, 'utf8');
      const data = JSON.parse(content);
      if (Array.isArray(data.requirements)) {
        requirements.push(...data.requirements);
      }
    } catch (error) {
      console.warn(`Unable to parse ${indexPath}: ${error.message}`);
    }
  }

  // Also scan for module.json files (hierarchical requirements)
  const scanModules = (dir) => {
    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          scanModules(fullPath);
        } else if (entry.name === 'module.json') {
          try {
            const content = fs.readFileSync(fullPath, 'utf8');
            const data = JSON.parse(content);
            if (Array.isArray(data.requirements)) {
              requirements.push(...data.requirements);
            }
          } catch (error) {
            console.warn(`Unable to parse ${fullPath}: ${error.message}`);
          }
        }
      }
    } catch (error) {
      // Ignore directory read errors
    }
  };

  scanModules(requirementsDir);

  return requirements;
}

/**
 * Extract operational targets from requirements
 * @param {array} requirements - Array of requirement objects
 * @returns {array} Array of unique operational target IDs
 */
function extractOperationalTargets(requirements) {
  const targets = new Set();

  for (const req of requirements) {
    if (req.prd_ref) {
      // Extract OT-P0-001, OT-P1-002, etc. from prd_ref
      const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
      if (match) {
        targets.add(match[0].toUpperCase());
      }
    }
  }

  return Array.from(targets);
}

/**
 * Load sync metadata from coverage/sync directory
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Sync metadata mapping
 */
function loadSyncMetadata(scenarioRoot) {
  const syncDir = path.join(scenarioRoot, 'coverage', 'sync');

  if (!fs.existsSync(syncDir)) {
    return {};
  }

  const metadata = {};

  try {
    const files = fs.readdirSync(syncDir).filter(f => f.endsWith('.json'));
    for (const file of files) {
      const filePath = path.join(syncDir, file);
      try {
        const content = fs.readFileSync(filePath, 'utf8');
        const data = JSON.parse(content);
        if (data.requirements) {
          Object.assign(metadata, data.requirements);
        }
      } catch (error) {
        console.warn(`Unable to parse ${filePath}: ${error.message}`);
      }
    }
  } catch (error) {
    console.warn(`Unable to read sync directory ${syncDir}: ${error.message}`);
  }

  return metadata;
}

/**
 * Load test results from test phase outputs
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Test results summary
 */
function loadTestResults(scenarioRoot) {
  const testResultsPath = path.join(scenarioRoot, 'coverage', 'test-results.json');

  if (!fs.existsSync(testResultsPath)) {
    return { total: 0, passing: 0, failing: 0, lastRun: null };
  }

  try {
    const content = fs.readFileSync(testResultsPath, 'utf8');
    const data = JSON.parse(content);
    return {
      total: (data.passed || 0) + (data.failed || 0),
      passing: data.passed || 0,
      failing: data.failed || 0,
      lastRun: data.timestamp || null
    };
  } catch (error) {
    console.warn(`Unable to parse ${testResultsPath}: ${error.message}`);
    return { total: 0, passing: 0, failing: 0, lastRun: null };
  }
}

/**
 * Calculate requirement pass status from sync metadata
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncMetadata - Sync metadata mapping
 * @returns {object} Requirement pass statistics
 */
function calculateRequirementPass(requirements, syncMetadata) {
  let passing = 0;
  let total = requirements.length;

  for (const req of requirements) {
    const reqMeta = syncMetadata[req.id];

    // Requirement is passing if:
    // 1. Status is "validated" or "implemented"
    // 2. OR sync metadata shows all_tests_passing is true
    if (req.status === 'validated' || req.status === 'implemented') {
      passing++;
    } else if (reqMeta && reqMeta.all_tests_passing === true) {
      passing++;
    }
  }

  return { total, passing };
}

/**
 * Calculate operational target pass status
 * @param {array} targets - Array of operational target IDs
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncMetadata - Sync metadata mapping
 * @returns {object} Target pass statistics
 */
function calculateTargetPass(targets, requirements, syncMetadata) {
  const total = targets.length;
  let passing = 0;

  for (const targetId of targets) {
    // Find all requirements linked to this target
    const linkedReqs = requirements.filter(req => {
      if (!req.prd_ref) return false;
      const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
      return match && match[0].toUpperCase() === targetId;
    });

    // Target is passing if at least one linked requirement is passing
    const hasPassingReq = linkedReqs.some(req => {
      const reqMeta = syncMetadata[req.id];
      return req.status === 'validated' || req.status === 'implemented' ||
             (reqMeta && reqMeta.all_tests_passing === true);
    });

    if (hasPassingReq) {
      passing++;
    }
  }

  return { total, passing };
}

/**
 * Collect all metrics for a scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Complete metrics object
 */
function collectMetrics(scenarioRoot) {
  const serviceConfig = loadServiceConfig(scenarioRoot);
  const requirements = loadRequirements(scenarioRoot);
  const operationalTargets = extractOperationalTargets(requirements);
  const syncMetadata = loadSyncMetadata(scenarioRoot);
  const testResults = loadTestResults(scenarioRoot);

  const requirementPass = calculateRequirementPass(requirements, syncMetadata);
  const targetPass = calculateTargetPass(operationalTargets, requirements, syncMetadata);

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
    rawRequirements: requirements  // Include for depth calculation
  };
}

module.exports = {
  loadServiceConfig,
  loadRequirements,
  extractOperationalTargets,
  loadSyncMetadata,
  loadTestResults,
  calculateRequirementPass,
  calculateTargetPass,
  collectMetrics
};
