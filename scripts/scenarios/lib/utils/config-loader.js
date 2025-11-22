'use strict';

/**
 * Configuration loader utility
 * Centralizes loading and access to completeness scoring configuration
 * @module scenarios/lib/utils/config-loader
 */

const fs = require('node:fs');
const path = require('node:path');

let cachedConfig = null;

/**
 * Load completeness configuration
 * @param {boolean} forceReload - Force reload even if cached
 * @returns {object} Complete configuration object
 * @throws {Error} If configuration cannot be loaded or parsed
 */
function loadConfig(forceReload = false) {
  if (cachedConfig && !forceReload) {
    return cachedConfig;
  }

  const configPath = path.join(__dirname, '..', 'completeness-config.json');

  try {
    const content = fs.readFileSync(configPath, 'utf8');
    cachedConfig = JSON.parse(content);
    return cachedConfig;
  } catch (error) {
    throw new Error(`Failed to load completeness configuration from ${configPath}: ${error.message}`);
  }
}

/**
 * Get penalty configuration
 * @param {string} penaltyType - Type of penalty (e.g., 'invalid_test_location')
 * @returns {object} Penalty configuration
 */
function getPenaltyConfig(penaltyType) {
  const config = loadConfig();
  return config.penalties[penaltyType] || null;
}

/**
 * Get validation configuration
 * @returns {object} Validation configuration
 */
function getValidationConfig() {
  const config = loadConfig();
  return config.validation || {};
}

/**
 * Get threshold configuration for a category
 * @param {string} category - Scenario category
 * @returns {object} Category-specific thresholds
 */
function getCategoryThresholds(category) {
  const config = loadConfig();
  const normalizedCategory = category || config.thresholds.default_category;
  return config.thresholds.categories[normalizedCategory] ||
         config.thresholds.categories[config.thresholds.default_category];
}

/**
 * Get classification thresholds
 * @returns {object} Classification thresholds
 */
function getClassifications() {
  const config = loadConfig();
  return config.classifications || {};
}

/**
 * Get scoring weights
 * @returns {object} Scoring weights and breakdowns
 */
function getScoringConfig() {
  const config = loadConfig();
  return config.scoring || {};
}

/**
 * Get staleness configuration
 * @returns {object} Staleness configuration
 */
function getStalenessConfig() {
  const config = loadConfig();
  return config.staleness || {};
}

module.exports = {
  loadConfig,
  getPenaltyConfig,
  getValidationConfig,
  getCategoryThresholds,
  getClassifications,
  getScoringConfig,
  getStalenessConfig
};
