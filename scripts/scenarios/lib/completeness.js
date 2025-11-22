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
 * Load threshold configuration from completeness-config.json
 * @returns {object} Threshold configuration object
 */
function loadThresholds() {
  const configPath = path.join(__dirname, 'completeness-config.json');
  try {
    const content = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(content);
    // Extract thresholds section from unified config
    return config.thresholds;
  } catch (error) {
    console.error(`Unable to load thresholds from ${configPath}: ${error.message}`);
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
 * Calculate quality score (50% weight)
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

  const reqPoints = Math.round(reqPassRate * 20);
  const targetPoints = Math.round(targetPassRate * 15);
  const testPoints = Math.round(testPassRate * 15);

  return {
    score: reqPoints + targetPoints + testPoints,
    max: 50,
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
 * Calculate coverage score (15% weight)
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
  const testCoveragePoints = Math.round((cappedRatio / 2.0) * 8);

  // Requirement depth score (cap at 3.0 levels = perfect score)
  let depthScore = 0;
  if (Array.isArray(requirements) && requirements.length > 0) {
    const depths = requirements.map(req => getMaxDepth(req));
    const avgDepth = depths.reduce((a, b) => a + b, 0) / depths.length;
    const cappedDepth = Math.min(avgDepth / 3.0, 1.0);
    depthScore = Math.round(cappedDepth * 7);
  }

  return {
    score: testCoveragePoints + depthScore,
    max: 15,
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
 * Calculate UI completeness score (25% weight)
 * Hybrid approach combining template detection, component complexity, and API integration
 * @param {object} uiMetrics - UI metrics from collectUIMetrics
 * @param {object} thresholds - Category thresholds
 * @returns {object} UI score breakdown
 */
function calculateUIScore(uiMetrics, thresholds) {
  if (!uiMetrics) {
    return {
      score: 0,
      max: 25,
      template_check: { is_template: true, penalty: 25, points: 0 },
      component_complexity: { file_count: 0, threshold: 'none', points: 0 },
      api_integration: { endpoint_count: 0, points: 0 },
      routing: { route_count: 0, points: 0 },
      code_volume: { total_loc: 0, points: 0 }
    };
  }

  const uiThresholds = thresholds.ui || {
    file_count: { ok: 15, good: 25, excellent: 40 },
    total_loc: { ok: 300, good: 600, excellent: 1200 },
    api_endpoints: { ok: 2, good: 4, excellent: 8 }
  };

  // Template signature check (40% = 10 points) - Binary: 0 if template, 10 if not
  const templatePoints = uiMetrics.is_template ? 0 : 10;

  // Component count (20% = 5 points) - Based on file count vs thresholds
  let componentPoints = 0;
  if (uiMetrics.file_count >= uiThresholds.file_count.excellent) {
    componentPoints = 5;
  } else if (uiMetrics.file_count >= uiThresholds.file_count.good) {
    componentPoints = 4;
  } else if (uiMetrics.file_count >= uiThresholds.file_count.ok) {
    componentPoints = 3;
  } else if (uiMetrics.file_count >= 10) {
    componentPoints = 2;
  } else if (uiMetrics.file_count >= 5) {
    componentPoints = 1;
  }

  // API integration depth (24% = 6 points) - Unique endpoints beyond /health
  let apiPoints = 0;
  if (uiMetrics.api_beyond_health >= uiThresholds.api_endpoints.excellent) {
    apiPoints = 6;
  } else if (uiMetrics.api_beyond_health >= uiThresholds.api_endpoints.good) {
    apiPoints = 5;
  } else if (uiMetrics.api_beyond_health >= uiThresholds.api_endpoints.ok) {
    apiPoints = 4;
  } else if (uiMetrics.api_beyond_health >= 2) {
    apiPoints = 3;
  } else if (uiMetrics.api_beyond_health >= 1) {
    apiPoints = 2;
  }

  // Router complexity (6% = 1.5 points) - Lower weight since SPAs can be complete
  let routerPoints = 0;
  if (uiMetrics.route_count >= 5) {
    routerPoints = 1.5;
  } else if (uiMetrics.route_count >= 3) {
    routerPoints = 1;
  } else if (uiMetrics.route_count >= 1) {
    routerPoints = 0.5;
  }

  // Code volume (10% = 2.5 points) - Total LOC vs threshold
  let volumePoints = 0;
  if (uiMetrics.total_loc >= uiThresholds.total_loc.excellent) {
    volumePoints = 2.5;
  } else if (uiMetrics.total_loc >= uiThresholds.total_loc.good) {
    volumePoints = 2;
  } else if (uiMetrics.total_loc >= uiThresholds.total_loc.ok) {
    volumePoints = 1.5;
  } else if (uiMetrics.total_loc >= 100) {
    volumePoints = 0.5;
  }

  const totalUIScore = Math.round(templatePoints + componentPoints + apiPoints + routerPoints + volumePoints);

  // Determine threshold level for file count
  const getFileCountThreshold = (count) => {
    if (count >= uiThresholds.file_count.excellent) return 'excellent';
    if (count >= uiThresholds.file_count.good) return 'good';
    if (count >= uiThresholds.file_count.ok) return 'ok';
    if (count > 0) return 'below';
    return 'none';
  };

  return {
    score: totalUIScore,
    max: 25,
    template_check: {
      is_template: uiMetrics.is_template,
      penalty: uiMetrics.is_template ? 10 : 0,
      points: templatePoints
    },
    component_complexity: {
      file_count: uiMetrics.file_count,
      component_count: uiMetrics.component_count,
      page_count: uiMetrics.page_count,
      threshold: getFileCountThreshold(uiMetrics.file_count),
      points: componentPoints
    },
    api_integration: {
      endpoint_count: uiMetrics.api_beyond_health,
      total_endpoints: uiMetrics.api_endpoints,
      points: apiPoints
    },
    routing: {
      has_routing: uiMetrics.has_routing,
      route_count: uiMetrics.route_count,
      points: routerPoints
    },
    code_volume: {
      total_loc: uiMetrics.total_loc,
      points: volumePoints
    }
  };
}

/**
 * Calculate overall completeness score
 * @param {object} metrics - Input metrics
 * @param {array} requirements - Array of requirement objects
 * @param {object} thresholds - Category thresholds
 * @param {object} validationQualityAnalysis - Optional validation quality analysis with penalties
 * @returns {object} Complete score breakdown
 */
function calculateCompletenessScore(metrics, requirements, thresholds, validationQualityAnalysis = null) {
  const quality = calculateQualityScore(metrics);
  const coverage = calculateCoverageScore(metrics, requirements);
  const quantity = calculateQuantityScore(metrics, thresholds);
  const ui = calculateUIScore(metrics.ui, thresholds);

  const baseScore = quality.score + coverage.score + quantity.score + ui.score;

  // Apply validation quality penalties if provided
  const validationPenalty = validationQualityAnalysis ? validationQualityAnalysis.total_penalty : 0;
  const finalScore = Math.max(baseScore - validationPenalty, 0);

  return {
    base_score: baseScore,
    validation_penalty: validationPenalty,
    score: finalScore,
    quality,
    coverage,
    quantity,
    ui
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

  // UI recommendations (highest priority if template detected)
  if (breakdown.ui && breakdown.ui.template_check.is_template) {
    recommendations.push('⚠️  Replace template UI with scenario-specific interface');
  }

  if (breakdown.ui && breakdown.ui.component_complexity.threshold === 'below') {
    const gap = thresholds.ui.file_count.ok - breakdown.ui.component_complexity.file_count;
    if (gap > 0) {
      recommendations.push(`Add ${gap} more UI files to reach minimum threshold (currently ${breakdown.ui.component_complexity.file_count} files)`);
    }
  }

  if (breakdown.ui && breakdown.ui.api_integration.endpoint_count === 0) {
    recommendations.push('Integrate UI with API endpoints beyond /health');
  } else if (breakdown.ui && breakdown.ui.api_integration.endpoint_count < thresholds.ui.api_endpoints.ok) {
    const gap = thresholds.ui.api_endpoints.ok - breakdown.ui.api_integration.endpoint_count;
    recommendations.push(`Add ${gap} more API endpoint integrations (currently ${breakdown.ui.api_integration.endpoint_count} beyond health)`);
  }

  if (breakdown.ui && !breakdown.ui.routing.has_routing && breakdown.ui.component_complexity.file_count > 0) {
    recommendations.push('Consider adding routing for multi-page navigation');
  }

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
  lines.push(`  ${breakdown.quality.requirement_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Requirements: ${breakdown.quality.requirement_pass_rate.total} total, ${breakdown.quality.requirement_pass_rate.passing} passing (${Math.round(breakdown.quality.requirement_pass_rate.rate * 100)}%) → ${breakdown.quality.requirement_pass_rate.points}/${20} pts`);
  lines.push(`  ${breakdown.quality.target_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Op Targets: ${breakdown.quality.target_pass_rate.total} total, ${breakdown.quality.target_pass_rate.passing} passing (${Math.round(breakdown.quality.target_pass_rate.rate * 100)}%) → ${breakdown.quality.target_pass_rate.points}/${15} pts`);
  lines.push(`  ${breakdown.quality.test_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Tests: ${breakdown.quality.test_pass_rate.total} total, ${breakdown.quality.test_pass_rate.passing} passing (${Math.round(breakdown.quality.test_pass_rate.rate * 100)}%) → ${breakdown.quality.test_pass_rate.points}/${15} pts${breakdown.quality.test_pass_rate.rate < 0.9 ? '  [Target: 90%+]' : ''}`);

  lines.push('');
  lines.push(`Coverage Metrics (${breakdown.coverage.score}/${breakdown.coverage.max}):`);
  lines.push(`  ${breakdown.coverage.test_coverage_ratio.ratio >= 2.0 ? '✅' : '⚠️ '} Test Coverage: ${breakdown.coverage.test_coverage_ratio.ratio.toFixed(1)}x → ${breakdown.coverage.test_coverage_ratio.points}/${8} pts${breakdown.coverage.test_coverage_ratio.ratio < 2.0 ? '  [Target: 2.0x]' : ''}`);
  lines.push(`  ${breakdown.coverage.depth_score.avg_depth >= 3.0 ? '✅' : '⚠️ '} Depth Score: ${breakdown.coverage.depth_score.avg_depth.toFixed(1)} avg levels → ${breakdown.coverage.depth_score.points}/${7} pts${breakdown.coverage.depth_score.avg_depth < 3.0 ? '  [Target: 3.0+]' : ''}`);

  lines.push('');
  lines.push(`Quantity Metrics (${breakdown.quantity.score}/${breakdown.quantity.max}):`);
  lines.push(`  ${breakdown.quantity.requirements.threshold === 'good' || breakdown.quantity.requirements.threshold === 'excellent' ? '✅' : '⚠️ '} Requirements: ${breakdown.quantity.requirements.count} (${breakdown.quantity.requirements.threshold}) → ${breakdown.quantity.requirements.points}/${4} pts${breakdown.quantity.requirements.threshold !== 'good' && breakdown.quantity.requirements.threshold !== 'excellent' ? `  [Target: ${thresholds.requirements.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.targets.threshold === 'good' || breakdown.quantity.targets.threshold === 'excellent' ? '✅' : '⚠️ '} Targets: ${breakdown.quantity.targets.count} (${breakdown.quantity.targets.threshold}) → ${breakdown.quantity.targets.points}/${3} pts${breakdown.quantity.targets.threshold !== 'good' && breakdown.quantity.targets.threshold !== 'excellent' ? `  [Target: ${thresholds.targets.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.tests.threshold === 'good' || breakdown.quantity.tests.threshold === 'excellent' ? '✅' : '⚠️ '} Tests: ${breakdown.quantity.tests.count} (${breakdown.quantity.tests.threshold}) → ${breakdown.quantity.tests.points}/${3} pts${breakdown.quantity.tests.threshold !== 'good' && breakdown.quantity.tests.threshold !== 'excellent' ? `  [Target: ${thresholds.tests.good}+]` : ''}`);

  // UI Metrics
  if (breakdown.ui) {
    lines.push('');
    lines.push(`UI Metrics (${breakdown.ui.score}/${breakdown.ui.max}):`);

    // Template check (most critical)
    const templateIcon = breakdown.ui.template_check.is_template ? '❌' : '✅';
    const templateStatus = breakdown.ui.template_check.is_template ? 'TEMPLATE' : 'Custom';
    lines.push(`  ${templateIcon} Template: ${templateStatus} → ${breakdown.ui.template_check.points}/${10} pts${breakdown.ui.template_check.is_template ? '  [CRITICAL: Replace template UI]' : ''}`);

    // Component complexity
    const compIcon = breakdown.ui.component_complexity.threshold === 'good' || breakdown.ui.component_complexity.threshold === 'excellent' ? '✅' : '⚠️ ';
    lines.push(`  ${compIcon} Files: ${breakdown.ui.component_complexity.file_count} files (${breakdown.ui.component_complexity.threshold}) → ${breakdown.ui.component_complexity.points}/${5} pts${breakdown.ui.component_complexity.threshold === 'below' || breakdown.ui.component_complexity.threshold === 'ok' ? `  [Target: ${thresholds.ui.file_count.good}+]` : ''}`);

    // API integration
    const apiIcon = breakdown.ui.api_integration.endpoint_count >= thresholds.ui.api_endpoints.good ? '✅' : '⚠️ ';
    lines.push(`  ${apiIcon} API Integration: ${breakdown.ui.api_integration.endpoint_count} endpoints beyond /health → ${breakdown.ui.api_integration.points}/${6} pts${breakdown.ui.api_integration.endpoint_count < thresholds.ui.api_endpoints.good ? `  [Target: ${thresholds.ui.api_endpoints.good}+]` : ''}`);

    // Routing
    const routeIcon = breakdown.ui.routing.route_count >= 3 ? '✅' : '⚠️ ';
    lines.push(`  ${routeIcon} Routing: ${breakdown.ui.routing.route_count} routes → ${breakdown.ui.routing.points}/${1.5} pts`);

    // Code volume
    const volumeIcon = breakdown.ui.code_volume.total_loc >= thresholds.ui.total_loc.good ? '✅' : '⚠️ ';
    lines.push(`  ${volumeIcon} LOC: ${breakdown.ui.code_volume.total_loc} total → ${breakdown.ui.code_volume.points}/${2.5} pts${breakdown.ui.code_volume.total_loc < thresholds.ui.total_loc.good ? `  [Target: ${thresholds.ui.total_loc.good}+]` : ''}`);
  }

  return lines.join('\n');
}

module.exports = {
  loadThresholds,
  getCategoryThresholds,
  calculateQualityScore,
  calculateCoverageScore,
  calculateQuantityScore,
  calculateUIScore,
  calculateCompletenessScore,
  classifyScore,
  getClassificationPrefix,
  checkStaleness,
  generateRecommendations,
  formatBreakdownHuman,
  getMaxDepth
};
