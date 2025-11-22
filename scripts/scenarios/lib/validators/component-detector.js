'use strict';

/**
 * Component detector - Identifies which components (API, UI) exist in a scenario
 * @module scenarios/lib/validators/component-detector
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Detect which components exist in the scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {Set<string>} Set of component types (API, UI)
 */
function detectScenarioComponents(scenarioRoot) {
  const components = new Set();

  // Check for API component
  const apiDir = path.join(scenarioRoot, 'api');
  if (fs.existsSync(apiDir)) {
    const hasGoFiles = fs.readdirSync(apiDir).some(f => f.endsWith('.go'));
    if (hasGoFiles) components.add('API');
  }

  // Check for UI component
  const uiDir = path.join(scenarioRoot, 'ui');
  if (fs.existsSync(uiDir)) {
    const pkgJson = path.join(uiDir, 'package.json');
    if (fs.existsSync(pkgJson)) components.add('UI');
  }

  return components;
}

/**
 * Get applicable validation layers for scenario based on components
 * @param {Set<string>} components - Detected components
 * @returns {Set<string>} Applicable layer names (API, UI, E2E)
 */
function getApplicableLayers(components) {
  const layers = new Set(['E2E']);  // E2E always applicable

  if (components.has('API')) {
    layers.add('API');
  }
  if (components.has('UI')) {
    layers.add('UI');
  }

  return layers;
}

module.exports = {
  detectScenarioComponents,
  getApplicableLayers
};
