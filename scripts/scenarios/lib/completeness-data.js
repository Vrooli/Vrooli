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
 * Supports both imports-based (BAS) and module-based (deployment-manager) architectures
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {array} Array of requirement objects
 */
function loadRequirements(scenarioRoot) {
  const requirementsDir = path.join(scenarioRoot, 'requirements');

  if (!fs.existsSync(requirementsDir)) {
    return [];
  }

  const requirements = [];
  const loadedFiles = new Set(); // Prevent duplicate loading

  /**
   * Load requirements from a single JSON file
   */
  const loadFromFile = (filePath) => {
    if (loadedFiles.has(filePath)) return;
    loadedFiles.add(filePath);

    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const data = JSON.parse(content);
      if (Array.isArray(data.requirements)) {
        requirements.push(...data.requirements);
      }
    } catch (error) {
      console.warn(`Unable to parse ${filePath}: ${error.message}`);
    }
  };

  // Try to load from index.json first
  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    try {
      const content = fs.readFileSync(indexPath, 'utf8');
      const data = JSON.parse(content);

      // Load parent requirements from index.json
      if (Array.isArray(data.requirements)) {
        requirements.push(...data.requirements);
      }

      // Check for imports array (BAS architecture)
      if (Array.isArray(data.imports)) {
        for (const importPath of data.imports) {
          const fullPath = path.join(requirementsDir, importPath);
          loadFromFile(fullPath);
        }
      }
    } catch (error) {
      console.warn(`Unable to parse ${indexPath}: ${error.message}`);
    }
  }

  // Also scan for module.json files (hierarchical requirements - deployment-manager style)
  const scanModules = (dir) => {
    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          scanModules(fullPath);
        } else if (entry.name === 'module.json') {
          loadFromFile(fullPath);
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
 * Extract operational targets from requirements and sync metadata
 * Supports both OT-P0-001 format (deployment-manager) and folder-based format (BAS)
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncMetadata - Sync metadata object (may contain operational_targets)
 * @returns {array} Array of unique operational target objects
 */
function extractOperationalTargets(requirements, syncMetadata) {
  const targets = new Map(); // Use Map to store target objects with IDs as keys

  // Method 1: Extract OT-P0-001 format from requirement prd_ref (deployment-manager style)
  for (const req of requirements) {
    if (req.prd_ref) {
      const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
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

  // Method 2: Extract from sync metadata operational_targets (BAS style)
  if (syncMetadata && syncMetadata.operational_targets && Array.isArray(syncMetadata.operational_targets)) {
    for (const ot of syncMetadata.operational_targets) {
      const targetId = ot.key || ot.folder_hint || ot.target_id;
      if (targetId && !targets.has(targetId)) {
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

  return Array.from(targets.values());
}

/**
 * Load sync metadata from coverage/sync or coverage/requirements-sync directory
 * Supports both legacy (sync/) and new (requirements-sync/latest.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Full sync metadata object including requirements and operational_targets
 */
function loadSyncMetadata(scenarioRoot) {
  // Try new format first: coverage/requirements-sync/latest.json (BAS format)
  const newSyncPath = path.join(scenarioRoot, 'coverage', 'requirements-sync', 'latest.json');
  if (fs.existsSync(newSyncPath)) {
    try {
      const content = fs.readFileSync(newSyncPath, 'utf8');
      const data = JSON.parse(content);
      // Return full object with requirements, operational_targets, etc.
      return data;
    } catch (error) {
      console.warn(`Unable to parse ${newSyncPath}: ${error.message}`);
    }
  }

  // Fallback to legacy format: coverage/sync/*.json
  const legacySyncDir = path.join(scenarioRoot, 'coverage', 'sync');
  if (fs.existsSync(legacySyncDir)) {
    const metadata = { requirements: {} };
    try {
      const files = fs.readdirSync(legacySyncDir).filter(f => f.endsWith('.json'));
      for (const file of files) {
        const filePath = path.join(legacySyncDir, file);
        try {
          const content = fs.readFileSync(filePath, 'utf8');
          const data = JSON.parse(content);
          if (data.requirements) {
            Object.assign(metadata.requirements, data.requirements);
          }
        } catch (error) {
          console.warn(`Unable to parse ${filePath}: ${error.message}`);
        }
      }
    } catch (error) {
      console.warn(`Unable to read sync directory ${legacySyncDir}: ${error.message}`);
    }
    return metadata;
  }

  return { requirements: {} };
}

/**
 * Load test results from test phase outputs
 * Supports both single-file (test-results.json) and phase-based (phase-results/*.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Test results summary
 */
function loadTestResults(scenarioRoot) {
  let totalPassing = 0;
  let totalFailing = 0;
  let latestTimestamp = null;

  // Try phase-based results first (BAS format: coverage/phase-results/*.json)
  const phaseResultsDir = path.join(scenarioRoot, 'coverage', 'phase-results');
  if (fs.existsSync(phaseResultsDir)) {
    try {
      const files = fs.readdirSync(phaseResultsDir).filter(f => f.endsWith('.json'));
      let hasResults = false;

      for (const file of files) {
        const filePath = path.join(phaseResultsDir, file);
        try {
          const content = fs.readFileSync(filePath, 'utf8');
          const data = JSON.parse(content);

          // Aggregate requirement-level pass/fail from phase results
          if (Array.isArray(data.requirements)) {
            hasResults = true;
            for (const req of data.requirements) {
              if (req.status === 'passed') {
                totalPassing++;
              } else if (req.status === 'failed') {
                totalFailing++;
              }
              // Skip 'skipped', 'unknown', 'not_run' - don't count as tests
            }

            // Track latest timestamp
            if (data.updated_at) {
              const timestamp = new Date(data.updated_at);
              if (!latestTimestamp || timestamp > latestTimestamp) {
                latestTimestamp = timestamp;
              }
            }
          }
        } catch (error) {
          console.warn(`Unable to parse ${filePath}: ${error.message}`);
        }
      }

      if (hasResults) {
        return {
          total: totalPassing + totalFailing,
          passing: totalPassing,
          failing: totalFailing,
          lastRun: latestTimestamp ? latestTimestamp.toISOString() : null
        };
      }
    } catch (error) {
      console.warn(`Unable to read phase results directory ${phaseResultsDir}: ${error.message}`);
    }
  }

  // Fallback to single test-results.json (legacy format)
  const testResultsPath = path.join(scenarioRoot, 'coverage', 'test-results.json');
  if (fs.existsSync(testResultsPath)) {
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
    }
  }

  return { total: 0, passing: 0, failing: 0, lastRun: null };
}

/**
 * Calculate requirement pass status from sync metadata
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Full sync metadata object
 * @returns {object} Requirement pass statistics
 */
function calculateRequirementPass(requirements, syncData) {
  let passing = 0;
  let total = requirements.length;

  const syncMetadata = syncData.requirements || syncData;

  for (const req of requirements) {
    const reqMeta = syncMetadata[req.id];

    // Requirement is passing if:
    // 1. Requirement status is "validated", "implemented", or "complete"
    // 2. OR sync metadata exists with status "complete" or "validated"
    // 3. OR sync metadata shows all_tests_passing is true
    if (req.status === 'validated' || req.status === 'implemented' || req.status === 'complete') {
      passing++;
    } else if (reqMeta) {
      if (reqMeta.status === 'complete' || reqMeta.status === 'validated') {
        passing++;
      } else if (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true) {
        passing++;
      }
    }
  }

  return { total, passing };
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

  const syncMetadata = syncData.requirements || syncData;

  for (const target of targets) {
    let isTargetPassing = false;

    if (target.type === 'folder') {
      // Folder-based target (BAS style): use counts or status from sync metadata
      if (target.status === 'complete') {
        isTargetPassing = true;
      } else if (target.counts) {
        // Target is passing if >50% of requirements are complete
        const completeRatio = target.counts.complete / target.counts.total;
        isTargetPassing = completeRatio > 0.5;
      }
    } else {
      // OT-P0-001 style target: check linked requirements
      const targetId = target.id;
      const linkedReqs = requirements.filter(req => {
        if (!req.prd_ref) return false;
        const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
        return match && match[0].toUpperCase() === targetId;
      });

      // Target is passing if at least one linked requirement is passing
      isTargetPassing = linkedReqs.some(req => {
        const reqMeta = syncMetadata[req.id];

        // Check requirement status
        if (req.status === 'validated' || req.status === 'implemented' || req.status === 'complete') {
          return true;
        }

        // Check sync metadata
        if (reqMeta) {
          if (reqMeta.status === 'complete' || reqMeta.status === 'validated') {
            return true;
          }
          if (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true) {
            return true;
          }
        }

        return false;
      });
    }

    if (isTargetPassing) {
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
  const syncData = loadSyncMetadata(scenarioRoot);
  const operationalTargets = extractOperationalTargets(requirements, syncData);
  const testResults = loadTestResults(scenarioRoot);

  const requirementPass = calculateRequirementPass(requirements, syncData);
  const targetPass = calculateTargetPass(operationalTargets, requirements, syncData);

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
