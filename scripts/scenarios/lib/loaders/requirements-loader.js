'use strict';

/**
 * Requirements loader
 * Loads requirements from index.json or module.json files
 * Supports both imports-based (BAS) and module-based (deployment-manager) architectures
 * @module scenarios/lib/loaders/requirements-loader
 */

const fs = require('node:fs');
const path = require('node:path');
const { readJsonFile } = require('../utils/file-utils');

/**
 * Load requirements from a single JSON file
 * @param {string} filePath - Path to requirements file
 * @param {Set} loadedFiles - Set to track already loaded files (prevents duplicates)
 * @returns {array} Array of requirements from this file
 */
function loadFromFile(filePath, loadedFiles) {
  if (loadedFiles.has(filePath)) {
    return [];
  }
  loadedFiles.add(filePath);

  const data = readJsonFile(filePath, { silent: true, defaultValue: {} });
  return Array.isArray(data.requirements) ? data.requirements : [];
}

/**
 * Scan directory recursively for module.json files
 * @param {string} dir - Directory to scan
 * @param {Set} loadedFiles - Set to track already loaded files
 * @returns {array} Array of all requirements found
 */
function scanModules(dir, loadedFiles) {
  const requirements = [];

  try {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        // Recursively scan subdirectories
        requirements.push(...scanModules(fullPath, loadedFiles));
      } else if (entry.name === 'module.json') {
        // Load requirements from module.json
        requirements.push(...loadFromFile(fullPath, loadedFiles));
      }
    }
  } catch (error) {
    // Silently ignore directory read errors
  }

  return requirements;
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

  // Try to load from index.json first
  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    const data = readJsonFile(indexPath, { silent: true, defaultValue: {} });

    // Load parent requirements from index.json
    if (Array.isArray(data.requirements)) {
      requirements.push(...data.requirements);
    }
    loadedFiles.add(indexPath);

    // Check for imports array (BAS architecture)
    if (Array.isArray(data.imports)) {
      for (const importPath of data.imports) {
        const fullPath = path.join(requirementsDir, importPath);
        requirements.push(...loadFromFile(fullPath, loadedFiles));
      }
    }
  }

  // Also scan for module.json files (hierarchical requirements - deployment-manager style)
  requirements.push(...scanModules(requirementsDir, loadedFiles));

  return requirements;
}

module.exports = {
  loadRequirements
};
