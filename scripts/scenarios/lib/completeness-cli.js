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
const { detectGamingPatterns } = require('./gaming-detection');

function showHelp() {
  console.log(`
Usage: vrooli scenario completeness <scenario-name> [--format json|human]

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
  --help, -h         Show this help message

Examples:
  vrooli scenario completeness deployment-manager
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

  // Calculate completeness score
  const breakdown = calculateCompletenessScore(
    metrics,
    metrics.rawRequirements,
    thresholds
  );

  const totalScore = breakdown.score;
  const classification = classifyScore(totalScore);

  // Check staleness
  const stalenessCheck = checkStaleness(metrics.lastTestRun);
  const warnings = stalenessCheck.warning ? [stalenessCheck] : [];

  // PHASE 5: Detect gaming patterns
  const gamingAnalysis = detectGamingPatterns(
    metrics,
    metrics.rawRequirements,
    metrics.targets ? Object.values(metrics.targets) : [],
    scenarioRoot
  );

  // Generate recommendations
  const recommendations = generateRecommendations(breakdown, thresholds);

  // Output based on format
  if (format === 'json') {
    const output = {
      scenario: metrics.scenario,
      category: metrics.category,
      score: totalScore,
      classification: classification,
      breakdown: {
        quality: breakdown.quality,
        coverage: breakdown.coverage,
        quantity: breakdown.quantity,
        ui: breakdown.ui
      },
      warnings: warnings,
      recommendations: recommendations,
      gaming_analysis: gamingAnalysis  // Include gaming analysis in JSON output
    };
    console.log(JSON.stringify(output, null, 2));
  } else {
    // Human-readable output
    console.log(`Scenario: ${metrics.scenario}`);
    console.log(`Category: ${metrics.category}`);
    console.log(`Completeness Score: ${totalScore}/100 (${classification.replace(/_/g, ' ')})`);
    console.log('');
    console.log(formatBreakdownHuman(breakdown, thresholds));

    if (warnings.length > 0) {
      console.log('');
      console.log('Warnings:');
      warnings.forEach(w => {
        console.log(`  ⚠️  ${w.message}${w.action ? `. ${w.action}` : ''}`);
      });
    }

    // PHASE 5: Display gaming pattern warnings
    if (gamingAnalysis.has_warnings) {
      console.log('');
      console.log('⚠️  Gaming Pattern Warnings:');
      console.log('');
      gamingAnalysis.warnings.forEach(warning => {
        const icon = warning.severity === 'high' ? '🔴' : '🟡';
        console.log(`${icon} ${warning.message}`);
        console.log(`   💡 ${warning.recommendation}`);
        console.log('');
      });
    }

    if (recommendations.length > 0) {
      console.log('');
      console.log('Priority Actions:');
      recommendations.forEach(rec => {
        console.log(`  • ${rec}`);
      });
    }

    console.log('');
    console.log(`Classification: ${classification}`);
    console.log(getClassificationPrefix(classification, totalScore));
  }
}

main();
