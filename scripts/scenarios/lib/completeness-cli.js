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

function showHelp() {
  console.log(`
Usage: vrooli scenario completeness <scenario-name> [--format json|human]

Calculate objective completeness score for a scenario based on:
  - Requirement pass rates (30% weight)
  - Operational target pass rates (20% weight)
  - Test pass rates (20% weight)
  - Test coverage ratio (10% weight)
  - Requirement depth (10% weight)
  - Absolute counts vs category thresholds (10% weight)

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
        quantity: breakdown.quantity
      },
      warnings: warnings,
      recommendations: recommendations
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
