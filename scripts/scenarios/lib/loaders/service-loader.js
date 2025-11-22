'use strict';

/**
 * Service configuration loader
 * Loads and validates .vrooli/service.json
 * @module scenarios/lib/loaders/service-loader
 */

const path = require('node:path');
const { readJsonFile } = require('../utils/file-utils');
const { DEFAULT_SERVICE_CATEGORY } = require('../constants');

/**
 * Load service.json to get scenario category and metadata
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Service configuration with defaults
 */
function loadServiceConfig(scenarioRoot) {
  const servicePath = path.join(scenarioRoot, '.vrooli', 'service.json');

  const config = readJsonFile(servicePath, {
    silent: false,
    defaultValue: {}
  });

  // Apply defaults
  return {
    category: config.category || DEFAULT_SERVICE_CATEGORY,
    name: config.name || path.basename(scenarioRoot),
    version: config.version || '0.0.0',
    ...config
  };
}

module.exports = {
  loadServiceConfig
};
