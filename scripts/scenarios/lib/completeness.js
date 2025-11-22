#!/usr/bin/env node
'use strict';

/**
 * Scenario completeness scoring system
 * Provides objective metrics-based classification to replace AI-based completion estimates
 * @module scenarios/lib/completeness
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Load threshold configuration
 * @returns {object} Threshold configuration object
 */
function loadThresholds() {
  const thresholdsPath = path.join(__dirname, 'completeness-thresholds.json');
  try {
    const content = fs.readFileSync(thresholdsPath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    console.error(`Unable to load thresholds from ${thresholdsPath}: ${error.message}`);
    throw error;
  }
}

/**
 * Get category thresholds for a scenario
 * @param {string} category - Scenario category
 * @param {object} thresholdsConfig - Loaded threshold configuration
 * @returns {object} Category-specific thresholds
 */
function getCategoryThresholds(category, thresholdsConfig) {
  const normalizedCategory = category || thresholdsConfig.default_category;
  return thresholdsConfig.categories[normalizedCategory] || thresholdsConfig.categories[thresholdsConfig.default_category];
}

/**
 * Calculate quality score (70% weight)
 * Based on requirement/target/test pass rates
 * @param {object} metrics - Input metrics
 * @returns {object} Quality score breakdown
 */
function calculateQualityScore(metrics) {
  const reqPassRate = metrics.requirements.total > 0
    ? metrics.requirements.passing / metrics.requirements.total
    : 0;
  const targetPassRate = metrics.targets.total > 0
    ? metrics.targets.passing / metrics.targets.total
    : 0;
  const testPassRate = metrics.tests.total > 0
    ? metrics.tests.passing / metrics.tests.total
    : 0;

  const reqPoints = Math.round(reqPassRate * 30);
  const targetPoints = Math.round(targetPassRate * 20);
  const testPoints = Math.round(testPassRate * 20);

  return {
    score: reqPoints + targetPoints + testPoints,
    max: 70,
    requirement_pass_rate: {
      passing: metrics.requirements.passing,
      total: metrics.requirements.total,
      rate: reqPassRate,
      points: reqPoints
    },
    target_pass_rate: {
      passing: metrics.targets.passing,
      total: metrics.targets.total,
      rate: targetPassRate,
      points: targetPoints
    },
    test_pass_rate: {
      passing: metrics.tests.passing,
      total: metrics.tests.total,
      rate: testPassRate,
      points: testPoints
    }
  };
}

/**
 * Calculate maximum depth of requirement tree
 * @param {object} requirement - Requirement object
 * @returns {number} Maximum depth
 */
function getMaxDepth(requirement) {
  if (!requirement.children || requirement.children.length === 0) {
    return 1;
  }
  return 1 + Math.max(...requirement.children.map(getMaxDepth));
}

/**
 * Calculate coverage score (20% weight)
 * Based on test coverage ratio and requirement depth
 * @param {object} metrics - Input metrics
 * @param {array} requirements - Array of requirement objects
 * @returns {object} Coverage score breakdown
 */
function calculateCoverageScore(metrics, requirements) {
  // Test coverage ratio (cap at 2.0x = perfect score)
  const testCoverageRatio = metrics.requirements.total > 0
    ? metrics.tests.total / metrics.requirements.total
    : 0;
  const cappedRatio = Math.min(testCoverageRatio, 2.0);
  const testCoveragePoints = Math.round((cappedRatio / 2.0) * 10);

  // Requirement depth score (cap at 3.0 levels = perfect score)
  let depthScore = 0;
  if (Array.isArray(requirements) && requirements.length > 0) {
    const depths = requirements.map(req => getMaxDepth(req));
    const avgDepth = depths.reduce((a, b) => a + b, 0) / depths.length;
    const cappedDepth = Math.min(avgDepth / 3.0, 1.0);
    depthScore = Math.round(cappedDepth * 10);
  }

  return {
    score: testCoveragePoints + depthScore,
    max: 20,
    test_coverage_ratio: {
      ratio: testCoverageRatio,
      points: testCoveragePoints
    },
    depth_score: {
      avg_depth: Array.isArray(requirements) && requirements.length > 0
        ? requirements.map(req => getMaxDepth(req)).reduce((a, b) => a + b, 0) / requirements.length
        : 0,
      points: depthScore
    }
  };
}

/**
 * Calculate quantity score (10% weight)
 * Based on category-specific thresholds
 * @param {object} metrics - Input metrics
 * @param {object} thresholds - Category thresholds
 * @returns {object} Quantity score breakdown
 */
function calculateQuantityScore(metrics, thresholds) {
  // Requirements score (4 points max)
  const reqRatio = Math.min(metrics.requirements.total / thresholds.requirements.good, 1.0);
  const reqPoints = Math.round(reqRatio * 4);

  // Targets score (3 points max)
  const targetRatio = Math.min(metrics.targets.total / thresholds.targets.good, 1.0);
  const targetPoints = Math.round(targetRatio * 3);

  // Tests score (3 points max)
  const testRatio = Math.min(metrics.tests.total / thresholds.tests.good, 1.0);
  const testPoints = Math.round(testRatio * 3);

  // Determine threshold level for each metric
  const getThresholdLevel = (count, thresholds) => {
    if (count >= thresholds.excellent) return 'excellent';
    if (count >= thresholds.good) return 'good';
    if (count >= thresholds.ok) return 'ok';
    return 'below';
  };

  return {
    score: reqPoints + targetPoints + testPoints,
    max: 10,
    requirements: {
      count: metrics.requirements.total,
      threshold: getThresholdLevel(metrics.requirements.total, thresholds.requirements),
      points: reqPoints
    },
    targets: {
      count: metrics.targets.total,
      threshold: getThresholdLevel(metrics.targets.total, thresholds.targets),
      points: targetPoints
    },
    tests: {
      count: metrics.tests.total,
      threshold: getThresholdLevel(metrics.tests.total, thresholds.tests),
      points: testPoints
    }
  };
}

/**
 * Calculate overall completeness score
 * @param {object} metrics - Input metrics
 * @param {array} requirements - Array of requirement objects
 * @param {object} thresholds - Category thresholds
 * @returns {object} Complete score breakdown
 */
function calculateCompletenessScore(metrics, requirements, thresholds) {
  const quality = calculateQualityScore(metrics);
  const coverage = calculateCoverageScore(metrics, requirements);
  const quantity = calculateQuantityScore(metrics, thresholds);

  const totalScore = quality.score + coverage.score + quantity.score;

  return {
    score: totalScore,
    quality,
    coverage,
    quantity
  };
}

/**
 * Map score to classification level
 * @param {number} score - Completeness score (0-100)
 * @returns {string} Classification level
 */
function classifyScore(score) {
  if (score >= 96) return 'production_ready';
  if (score >= 81) return 'nearly_ready';
  if (score >= 61) return 'mostly_complete';
  if (score >= 41) return 'functional_incomplete';
  if (score >= 21) return 'foundation_laid';
  return 'early_stage';
}

/**
 * Get human-readable classification prefix
 * @param {string} classification - Classification level
 * @param {number} score - Completeness score
 * @returns {string} Human-readable prefix
 */
function getClassificationPrefix(classification, score) {
  const prefixes = {
    production_ready: `**Score: ${score}/100** - Production ready, excellent validation coverage`,
    nearly_ready: `**Score: ${score}/100** - Nearly ready, final polish and edge cases`,
    mostly_complete: `**Score: ${score}/100** - Mostly complete, needs refinement and validation`,
    functional_incomplete: `**Score: ${score}/100** - Functional but incomplete, needs more features/tests`,
    foundation_laid: `**Score: ${score}/100** - Foundation laid, core features in progress`,
    early_stage: `**Score: ${score}/100** - Just starting, needs significant development`
  };
  return prefixes[classification] || `**Score: ${score}/100** - Status unclear`;
}

/**
 * Check if test results are stale
 * @param {string} lastTestRun - ISO timestamp of last test run
 * @returns {object} Staleness check result
 */
function checkStaleness(lastTestRun) {
  if (!lastTestRun) {
    return {
      warning: true,
      type: 'staleness',
      message: 'No test results found',
      action: 'Run `make test` to generate test results',
      hoursStale: null
    };
  }

  const hoursSinceTest = (Date.now() - Date.parse(lastTestRun)) / (1000 * 60 * 60);

  if (hoursSinceTest > 48) {
    return {
      warning: true,
      type: 'staleness',
      message: `Test results stale (${Math.floor(hoursSinceTest)}h old)`,
      action: 'Run `make test` to refresh',
      hoursStale: Math.floor(hoursSinceTest)
    };
  }

  return { warning: false };
}

/**
 * Generate actionable recommendations
 * @param {object} breakdown - Score breakdown
 * @param {object} thresholds - Category thresholds
 * @returns {array} Array of recommendation strings
 */
function generateRecommendations(breakdown, thresholds) {
  const recommendations = [];

  // Quality recommendations
  if (breakdown.quality.test_pass_rate.rate < 0.9) {
    const currentPercent = Math.round(breakdown.quality.test_pass_rate.rate * 100);
    recommendations.push(`Increase test pass rate from ${currentPercent}% to 90%+`);
  }

  if (breakdown.quality.requirement_pass_rate.rate < 0.9) {
    const currentPercent = Math.round(breakdown.quality.requirement_pass_rate.rate * 100);
    recommendations.push(`Increase requirement pass rate from ${currentPercent}% to 90%+`);
  }

  if (breakdown.quality.target_pass_rate.rate < 0.9) {
    const currentPercent = Math.round(breakdown.quality.target_pass_rate.rate * 100);
    recommendations.push(`Increase operational target pass rate from ${currentPercent}% to 90%+`);
  }

  // Quantity recommendations
  if (breakdown.quantity.requirements.threshold === 'below' || breakdown.quantity.requirements.threshold === 'ok') {
    const gap = thresholds.requirements.good - breakdown.quantity.requirements.count;
    if (gap > 0) {
      recommendations.push(`Add ${gap} more requirements to reach 'good' threshold (${thresholds.requirements.good})`);
    }
  }

  if (breakdown.quantity.targets.threshold === 'below' || breakdown.quantity.targets.threshold === 'ok') {
    const gap = thresholds.targets.good - breakdown.quantity.targets.count;
    if (gap > 0) {
      recommendations.push(`Add ${gap} more operational targets to reach 'good' threshold (${thresholds.targets.good})`);
    }
  }

  if (breakdown.quantity.tests.threshold === 'below' || breakdown.quantity.tests.threshold === 'ok') {
    const gap = thresholds.tests.good - breakdown.quantity.tests.count;
    if (gap > 0) {
      recommendations.push(`Add ${gap} more tests to reach 'good' threshold (${thresholds.tests.good})`);
    }
  }

  // Coverage recommendations
  if (breakdown.coverage.test_coverage_ratio.ratio < 2.0) {
    const gap = Math.ceil((2.0 * breakdown.quality.requirement_pass_rate.total) - breakdown.quality.test_pass_rate.total);
    if (gap > 0) {
      recommendations.push(`Add ${gap} more tests to reach optimal 2:1 test-to-requirement ratio`);
    }
  }

  return recommendations;
}

/**
 * Format breakdown for human output
 * @param {object} breakdown - Score breakdown
 * @param {object} thresholds - Category thresholds
 * @returns {string} Formatted breakdown
 */
function formatBreakdownHuman(breakdown, thresholds) {
  const lines = [];

  lines.push(`Quality Metrics (${breakdown.quality.score}/${breakdown.quality.max}):`);
  lines.push(`  ${breakdown.quality.requirement_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Requirements: ${breakdown.quality.requirement_pass_rate.total} total, ${breakdown.quality.requirement_pass_rate.passing} passing (${Math.round(breakdown.quality.requirement_pass_rate.rate * 100)}%) → ${breakdown.quality.requirement_pass_rate.points}/${30} pts`);
  lines.push(`  ${breakdown.quality.target_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Op Targets: ${breakdown.quality.target_pass_rate.total} total, ${breakdown.quality.target_pass_rate.passing} passing (${Math.round(breakdown.quality.target_pass_rate.rate * 100)}%) → ${breakdown.quality.target_pass_rate.points}/${20} pts`);
  lines.push(`  ${breakdown.quality.test_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Tests: ${breakdown.quality.test_pass_rate.total} total, ${breakdown.quality.test_pass_rate.passing} passing (${Math.round(breakdown.quality.test_pass_rate.rate * 100)}%) → ${breakdown.quality.test_pass_rate.points}/${20} pts${breakdown.quality.test_pass_rate.rate < 0.9 ? '  [Target: 90%+]' : ''}`);

  lines.push('');
  lines.push(`Coverage Metrics (${breakdown.coverage.score}/${breakdown.coverage.max}):`);
  lines.push(`  ${breakdown.coverage.test_coverage_ratio.ratio >= 2.0 ? '✅' : '⚠️ '} Test Coverage: ${breakdown.coverage.test_coverage_ratio.ratio.toFixed(1)}x → ${breakdown.coverage.test_coverage_ratio.points}/${10} pts${breakdown.coverage.test_coverage_ratio.ratio < 2.0 ? '  [Target: 2.0x]' : ''}`);
  lines.push(`  ${breakdown.coverage.depth_score.avg_depth >= 3.0 ? '✅' : '⚠️ '} Depth Score: ${breakdown.coverage.depth_score.avg_depth.toFixed(1)} avg levels → ${breakdown.coverage.depth_score.points}/${10} pts${breakdown.coverage.depth_score.avg_depth < 3.0 ? '  [Target: 3.0+]' : ''}`);

  lines.push('');
  lines.push(`Quantity Metrics (${breakdown.quantity.score}/${breakdown.quantity.max}):`);
  lines.push(`  ${breakdown.quantity.requirements.threshold === 'good' || breakdown.quantity.requirements.threshold === 'excellent' ? '✅' : '⚠️ '} Requirements: ${breakdown.quantity.requirements.count} (${breakdown.quantity.requirements.threshold}) → ${breakdown.quantity.requirements.points}/${4} pts${breakdown.quantity.requirements.threshold !== 'good' && breakdown.quantity.requirements.threshold !== 'excellent' ? `  [Target: ${thresholds.requirements.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.targets.threshold === 'good' || breakdown.quantity.targets.threshold === 'excellent' ? '✅' : '⚠️ '} Targets: ${breakdown.quantity.targets.count} (${breakdown.quantity.targets.threshold}) → ${breakdown.quantity.targets.points}/${3} pts${breakdown.quantity.targets.threshold !== 'good' && breakdown.quantity.targets.threshold !== 'excellent' ? `  [Target: ${thresholds.targets.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.tests.threshold === 'good' || breakdown.quantity.tests.threshold === 'excellent' ? '✅' : '⚠️ '} Tests: ${breakdown.quantity.tests.count} (${breakdown.quantity.tests.threshold}) → ${breakdown.quantity.tests.points}/${3} pts${breakdown.quantity.tests.threshold !== 'good' && breakdown.quantity.tests.threshold !== 'excellent' ? `  [Target: ${thresholds.tests.good}+]` : ''}`);

  return lines.join('\n');
}

module.exports = {
  loadThresholds,
  getCategoryThresholds,
  calculateQualityScore,
  calculateCoverageScore,
  calculateQuantityScore,
  calculateCompletenessScore,
  classifyScore,
  getClassificationPrefix,
  checkStaleness,
  generateRecommendations,
  formatBreakdownHuman,
  getMaxDepth
};
