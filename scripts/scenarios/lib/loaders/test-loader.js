'use strict';

/**
 * Test results loader
 * Loads test results from test phase outputs
 * Supports both single-file (test-results.json) and phase-based (phase-results/*.json) formats
 * @module scenarios/lib/loaders/test-loader
 */

const fs = require('node:fs');
const path = require('node:path');
const { readJsonFile } = require('../utils/file-utils');

/**
 * Load test results from phase-based format (BAS format)
 * Path: coverage/phase-results/*.json
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object|null} Test results summary or null if not found
 */
function loadPhaseBasedResults(scenarioRoot) {
  const phaseResultsDir = path.join(scenarioRoot, 'coverage', 'phase-results');

  if (!fs.existsSync(phaseResultsDir)) {
    return null;
  }

  let totalPassing = 0;
  let totalFailing = 0;
  let latestTimestamp = null;
  let hasResults = false;

  try {
    const files = fs.readdirSync(phaseResultsDir).filter(f => f.endsWith('.json'));

    for (const file of files) {
      const filePath = path.join(phaseResultsDir, file);
      const data = readJsonFile(filePath, { silent: true, defaultValue: {} });

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

  return null;
}

/**
 * Load test results from single-file format (legacy format)
 * Path: coverage/test-results.json
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object|null} Test results summary or null if not found
 */
function loadSingleFileResults(scenarioRoot) {
  const testResultsPath = path.join(scenarioRoot, 'coverage', 'test-results.json');

  if (!fs.existsSync(testResultsPath)) {
    return null;
  }

  const data = readJsonFile(testResultsPath, { silent: true, defaultValue: {} });

  return {
    total: (data.passed || 0) + (data.failed || 0),
    passing: data.passed || 0,
    failing: data.failed || 0,
    lastRun: data.timestamp || null
  };
}

/**
 * Load test results from test phase outputs
 * Supports both single-file (test-results.json) and phase-based (phase-results/*.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Test results summary
 */
function loadTestResults(scenarioRoot) {
  // Try phase-based results first (preferred format)
  const phaseResults = loadPhaseBasedResults(scenarioRoot);
  if (phaseResults !== null) {
    return phaseResults;
  }

  // Fallback to single-file results
  const singleFileResults = loadSingleFileResults(scenarioRoot);
  if (singleFileResults !== null) {
    return singleFileResults;
  }

  // No test results found
  return {
    total: 0,
    passing: 0,
    failing: 0,
    lastRun: null
  };
}

module.exports = {
  loadTestResults
};
