#!/usr/bin/env node
'use strict';

/**
 * CLI entry point for scenario completeness scoring
 * @module scenarios/lib/completeness-cli
 */

const path = require('node:path');
const {
  loadThresholds,
  getCategoryThresholds,
  calculateCompletenessScore,
  classifyScore,
  getClassificationPrefix,
  checkStaleness,
  generateRecommendations,
  formatBreakdownHuman
} = require('./completeness');

const { collectMetrics } = require('./completeness-data');
const { detectValidationQualityIssues } = require('./gaming-detection');
const {
  formatValidationIssues,
  formatScoreSummary,
  formatQualityAssessment,
  formatBaseMetrics,
  formatDetailedMetrics,
  formatActionPlan,
  formatComparison
} = require('./completeness-output-formatter');

function showHelp() {
  console.log(`
Usage: vrooli scenario completeness <scenario-name> [options]

Calculate objective completeness score for a scenario based on:
  Quality (50%):
    - Requirement pass rates (20%)
    - Operational target pass rates (15%)
    - Test pass rates (15%)
  Coverage (15%):
    - Test coverage ratio (8%)
    - Requirement depth (7%)
  Quantity (10%):
    - Absolute counts vs category thresholds (10%)
  UI Completeness (25%):
    - Template detection (10%)
    - Component complexity (5%)
    - API integration (5%)
    - Routing (2.5%)
    - Code volume (2.5%)

Options:
  --format <type>    Output format: json (machine-readable) or human (default)
  --verbose          Show detailed explanations and per-requirement breakdowns
  --metrics          Show full detailed metrics breakdown (implies --verbose)
  --help, -h         Show this help message

Examples:
  vrooli scenario completeness deployment-manager
  vrooli scenario completeness deployment-manager --verbose
  vrooli scenario completeness deployment-manager --metrics
  vrooli scenario completeness deployment-manager --format json
`);
}

function main() {
  const args = process.argv.slice(2);

  if (args.length === 0 || args.includes('--help') || args.includes('-h')) {
    showHelp();
    process.exit(0);
  }

  const scenarioName = args[0];
  const formatArg = args.indexOf('--format');
  const format = formatArg !== -1 && args[formatArg + 1] ? args[formatArg + 1] : 'human';
  const verbose = args.includes('--verbose');
  const showMetrics = args.includes('--metrics');

  if (!['json', 'human'].includes(format)) {
    console.error(`Invalid format: ${format}. Use 'json' or 'human'.`);
    process.exit(1);
  }

  // Determine scenario root
  const vrooliRoot = process.env.VROOLI_ROOT || path.join(__dirname, '../../..');
  const scenarioRoot = path.join(vrooliRoot, 'scenarios', scenarioName);

  // Load thresholds
  let thresholdsConfig;
  try {
    thresholdsConfig = loadThresholds();
  } catch (error) {
    console.error(`Failed to load thresholds: ${error.message}`);
    process.exit(1);
  }

  // Collect metrics
  let metrics;
  try {
    metrics = collectMetrics(scenarioRoot);
  } catch (error) {
    console.error(`Failed to collect metrics for ${scenarioName}: ${error.message}`);
    process.exit(1);
  }

  // Get category thresholds
  const thresholds = getCategoryThresholds(metrics.category, thresholdsConfig);

  // Detect validation quality issues first (needed for penalty calculation)
  const validationQualityAnalysis = detectValidationQualityIssues(
    metrics,
    metrics.rawRequirements,
    metrics.targets ? Object.values(metrics.targets) : [],
    scenarioRoot
  );

  // Calculate completeness score with validation penalties
  const breakdown = calculateCompletenessScore(
    metrics,
    metrics.rawRequirements,
    thresholds,
    validationQualityAnalysis
  );

  const totalScore = breakdown.score;
  const classification = classifyScore(totalScore);

  // Check staleness
  const stalenessCheck = checkStaleness(metrics.lastTestRun);
  const warnings = stalenessCheck.warning ? [stalenessCheck] : [];

  // Generate recommendations
  const recommendations = generateRecommendations(breakdown, thresholds);

  // Output based on format
  if (format === 'json') {
    const output = {
      scenario: metrics.scenario,
      category: metrics.category,
      base_score: breakdown.base_score,
      validation_penalty: breakdown.validation_penalty,
      final_score: totalScore,
      classification: classification,
      breakdown: {
        quality: breakdown.quality,
        coverage: breakdown.coverage,
        quantity: breakdown.quantity,
        ui: breakdown.ui
      },
      warnings: warnings,
      recommendations: recommendations,
      validation_quality_analysis: validationQualityAnalysis
    };
    console.log(JSON.stringify(output, null, 2));
  } else {
    // Human-readable output with new formatter
    console.log(`Scenario: ${metrics.scenario}`);
    console.log(`Category: ${metrics.category}`);

    // 1. Validation Issues (top priority - always shown)
    console.log(formatValidationIssues(validationQualityAnalysis, { verbose: verbose || showMetrics }));

    // 2. Score Summary
    console.log(formatScoreSummary(totalScore, breakdown, classification, validationQualityAnalysis));

    // 3. Base Metrics (always shown)
    console.log('');
    console.log(formatBaseMetrics(breakdown, thresholds));

    // 4. Detailed Metrics (only if --metrics flag - shows sub-breakdowns)
    if (showMetrics) {
      console.log('');
      console.log('━'.repeat(68));
      console.log(formatDetailedMetrics(breakdown, thresholds));
    }

    // 5. Warnings (if any)
    if (warnings.length > 0) {
      console.log('');
      console.log('Warnings:');
      warnings.forEach(w => {
        console.log(`  ⚠️  ${w.message}${w.action ? `. ${w.action}` : ''}`);
      });
    }

    // 6. Action Plan
    console.log('');
    console.log(formatActionPlan(breakdown, validationQualityAnalysis, thresholds));

    // 7. Comparison Context
    console.log('');
    const invalidRefCount = validationQualityAnalysis.patterns.invalid_test_location?.count || 0;
    const invalidRefRatio = invalidRefCount / Math.max(metrics.requirements.total, 1);
    console.log(formatComparison(metrics.scenario, totalScore, breakdown.validation_penalty, invalidRefRatio));

    // 8. Help text
    console.log('');
    console.log('Use --help to see all available flags and options.');
  }
}

main();
