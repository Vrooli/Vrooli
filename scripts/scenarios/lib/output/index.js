'use strict';

/**
 * Completeness Output Module
 *
 * Provides formatting and output generation for completeness scoring.
 * This module acts as a facade, delegating to specialized formatters and generators.
 *
 * @module scenarios/lib/output
 */

// For now, re-export from the original monolithic file to maintain backward compatibility
// TODO: Gradually migrate to modular structure in output/formatters/ and output/generators/
const {
  formatValidationIssues,
  formatScoreSummary,
  formatQualityAssessment,
  formatBaseMetrics,
  formatDetailedMetrics,
  formatActionPlan,
  formatComparison
} = require('../completeness-output-formatter');

module.exports = {
  formatValidationIssues,
  formatScoreSummary,
  formatQualityAssessment,
  formatBaseMetrics,
  formatDetailedMetrics,
  formatActionPlan,
  formatComparison
};
