#!/usr/bin/env node
'use strict';

/**
 * File discovery and scenario resolution for requirements system
 * @module requirements/lib/discovery
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Check if a directory is a scenario root
 * @param {string} directory - Directory to check
 * @param {string} scenario - Scenario name
 * @returns {boolean} True if directory is a scenario root
 */
function isScenarioRoot(directory, scenario) {
  if (!directory) {
    return false;
  }
  const basename = path.basename(directory);
  if (basename !== scenario) {
    return false;
  }
  const testDir = path.join(directory, 'test');
  return fs.existsSync(testDir);
}

/**
 * Resolve the root directory of a scenario
 * @param {string} baseDir - Starting directory
 * @param {string} scenario - Scenario name
 * @returns {string} Absolute path to scenario root
 * @throws {Error} If scenario cannot be located
 */
function resolveScenarioRoot(baseDir, scenario) {
  let current = path.resolve(baseDir);

  while (true) {
    if (isScenarioRoot(current, scenario)) {
      return current;
    }

    const candidate = path.join(current, 'scenarios', scenario);
    if (fs.existsSync(candidate)) {
      return candidate;
    }

    const parent = path.dirname(current);
    if (parent === current) {
      break;
    }
    current = parent;
  }

  if (process.env.VROOLI_ROOT) {
    const candidate = path.join(process.env.VROOLI_ROOT, 'scenarios', scenario);
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(`Unable to locate scenario '${scenario}'`);
}

/**
 * Parse imports array from index.json file
 * @param {string} indexPath - Path to index.json file
 * @returns {string[]} Array of import paths
 */
function parseImports(indexPath) {
  if (!fs.existsSync(indexPath)) {
    return [];
  }

  try {
    const content = fs.readFileSync(indexPath, 'utf8');
    const parsed = JSON.parse(content);
    return parsed.imports || [];
  } catch (error) {
    console.warn(`Failed to parse imports from ${indexPath}: ${error.message}`);
    return [];
  }
}

/**
 * Collect all requirement files from a scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {import('./types').RequirementFile[]} Array of requirement files
 * @throws {Error} If no requirements registry found
 */
function collectRequirementFiles(scenarioRoot) {
  // Check for requirements/ folder
  const requirementsDir = path.join(scenarioRoot, 'requirements');
  if (!fs.existsSync(requirementsDir)) {
    throw new Error('No requirements registry found (expected requirements/ folder)');
  }

  const results = [];
  const seen = new Set();

  /**
   * Enqueue a file for processing
   * @param {string} filePath - File path to enqueue
   * @param {boolean} [isIndex=false] - Whether this is an index file
   */
  const enqueue = (filePath, isIndex = false) => {
    if (!fs.existsSync(filePath)) {
      return;
    }
    const resolved = path.resolve(filePath);
    if (seen.has(resolved)) {
      return;
    }
    seen.add(resolved);
    const relative = path.relative(scenarioRoot, resolved);
    results.push({ path: resolved, relative, isIndex });
  };

  // Process index.json first if it exists
  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    enqueue(indexPath, true);
    const imports = parseImports(indexPath);
    imports.forEach((importEntry) => {
      const candidate = path.join(requirementsDir, importEntry);
      enqueue(candidate, false);
    });
  }

  // Walk the directory tree to find all JSON files
  const stack = [requirementsDir];
  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    entries.forEach((entry) => {
      if (entry.name.startsWith('.')) {
        return;
      }
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
        return;
      }
      if (!entry.isFile()) {
        return;
      }
      if (!entry.name.endsWith('.json')) {
        return;
      }
      enqueue(fullPath, path.resolve(fullPath) === path.resolve(indexPath));
    });
  }

  // Sort results with index first
  results.sort((a, b) => {
    if (a.isIndex && !b.isIndex) {
      return -1;
    }
    if (!a.isIndex && b.isIndex) {
      return 1;
    }
    return a.relative.localeCompare(b.relative);
  });

  return results;
}

module.exports = {
  isScenarioRoot,
  resolveScenarioRoot,
  parseImports,
  collectRequirementFiles,
};
