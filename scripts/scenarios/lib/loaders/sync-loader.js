'use strict';

/**
 * Sync metadata loader
 * Loads sync metadata from coverage/sync or coverage/requirements-sync directory
 * Supports both legacy (sync/) and new (requirements-sync/latest.json) formats
 * @module scenarios/lib/loaders/sync-loader
 */

const fs = require('node:fs');
const path = require('node:path');
const { readJsonFile } = require('../utils/file-utils');

/**
 * Load sync metadata from new format (BAS format)
 * Path: coverage/requirements-sync/latest.json
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object|null} Full sync metadata object or null if not found
 */
function loadNewFormat(scenarioRoot) {
  const newSyncPath = path.join(scenarioRoot, 'coverage', 'requirements-sync', 'latest.json');

  if (!fs.existsSync(newSyncPath)) {
    return null;
  }

  return readJsonFile(newSyncPath, {
    silent: true,
    defaultValue: null
  });
}

/**
 * Load sync metadata from legacy format
 * Path: coverage/sync/*.json
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object|null} Aggregated sync metadata object or null if not found
 */
function loadLegacyFormat(scenarioRoot) {
  const legacySyncDir = path.join(scenarioRoot, 'coverage', 'sync');

  if (!fs.existsSync(legacySyncDir)) {
    return null;
  }

  const metadata = { requirements: {} };

  try {
    const files = fs.readdirSync(legacySyncDir).filter(f => f.endsWith('.json'));

    if (files.length === 0) {
      return null;
    }

    for (const file of files) {
      const filePath = path.join(legacySyncDir, file);
      const data = readJsonFile(filePath, { silent: true, defaultValue: {} });

      if (data.requirements) {
        Object.assign(metadata.requirements, data.requirements);
      }
    }

    return metadata;
  } catch (error) {
    console.warn(`Unable to read sync directory ${legacySyncDir}: ${error.message}`);
    return null;
  }
}

/**
 * Load sync metadata from coverage/sync or coverage/requirements-sync directory
 * Supports both legacy (sync/) and new (requirements-sync/latest.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Full sync metadata object including requirements and operational_targets
 */
function loadSyncMetadata(scenarioRoot) {
  // Try new format first
  const newFormat = loadNewFormat(scenarioRoot);
  if (newFormat !== null) {
    return newFormat;
  }

  // Fallback to legacy format
  const legacyFormat = loadLegacyFormat(scenarioRoot);
  if (legacyFormat !== null) {
    return legacyFormat;
  }

  // No sync data found - return empty structure
  return { requirements: {} };
}

module.exports = {
  loadSyncMetadata
};
