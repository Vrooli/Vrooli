#!/usr/bin/env node
/**
 * Lighthouse Test Runner for Vrooli Scenarios
 *
 * Orchestrates Lighthouse audits, manages Chrome instances, checks thresholds,
 * and integrates with the requirements tracking system.
 *
 * Usage:
 *   node runner.js --config <path> --base-url <url> --output-dir <dir> \
 *                  --scenario <name> --phase-results-dir <dir>
 */

'use strict';

const fs = require('fs');
const path = require('path');
const lighthouse = require('lighthouse').default;
const chromeLauncher = require('chrome-launcher');

// Parse command line arguments
function parseArgs(argv) {
  const args = {
    config: '',
    baseUrl: '',
    outputDir: '',
    scenario: '',
    phaseResultsDir: '',
    lighthouseOnly: false,
    verbose: false,
  };

  for (let i = 0; i < argv.length; i++) {
    switch (argv[i]) {
      case '--config':
        args.config = argv[++i];
        break;
      case '--base-url':
        args.baseUrl = argv[++i];
        break;
      case '--output-dir':
        args.outputDir = argv[++i];
        break;
      case '--scenario':
        args.scenario = argv[++i];
        break;
      case '--phase-results-dir':
        args.phaseResultsDir = argv[++i];
        break;
      case '--lighthouse-only':
        args.lighthouseOnly = true;
        break;
      case '--verbose':
      case '-v':
        args.verbose = true;
        break;
      case '--help':
      case '-h':
        printHelp();
        process.exit(0);
        break;
    }
  }

  // Validate required arguments
  const required = ['config', 'baseUrl', 'outputDir', 'scenario', 'phaseResultsDir'];
  const missing = required.filter(key => !args[key]);
  if (missing.length > 0) {
    console.error(`Error: Missing required arguments: ${missing.join(', ')}`);
    console.error('Run with --help for usage information');
    process.exit(1);
  }

  return args;
}

function printHelp() {
  console.log(`
Lighthouse Test Runner for Vrooli Scenarios

Usage:
  node runner.js [options]

Required Options:
  --config <path>              Path to .vrooli/lighthouse.json
  --base-url <url>             Base URL of the scenario UI (e.g., http://localhost:3000)
  --output-dir <dir>           Directory to write Lighthouse reports
  --scenario <name>            Scenario name (for phase results)
  --phase-results-dir <dir>    Directory to write phase results JSON

Optional:
  --lighthouse-only            Only run Lighthouse (skip other performance tests)
  --verbose, -v                Enable verbose output
  --help, -h                   Show this help message

Example:
  node runner.js \\
    --config scenarios/browser-automation-studio/.vrooli/lighthouse.json \\
    --base-url http://localhost:3000 \\
    --output-dir scenarios/browser-automation-studio/test/artifacts/lighthouse \\
    --scenario browser-automation-studio \\
    --phase-results-dir scenarios/browser-automation-studio/coverage/phase-results
`);
}

// Build Lighthouse configuration from scenario config and page settings
function buildLighthouseConfig(globalConfig, pageConfig) {
  const config = {
    extends: 'lighthouse:default',
    settings: {
      onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
      throttlingMethod: 'simulate',
      formFactor: pageConfig.viewport === 'mobile' ? 'mobile' : 'desktop',
      screenEmulation: pageConfig.viewport === 'mobile'
        ? { mobile: true, width: 375, height: 667, deviceScaleFactor: 2 }
        : { mobile: false, width: 1440, height: 900, deviceScaleFactor: 1 },
      ...(globalConfig.global_options?.lighthouse?.settings || {})
    }
  };

  // Override with custom config if present
  const customConfigPath = path.join(path.dirname(globalConfig._configPath), 'custom-config.js');
  if (fs.existsSync(customConfigPath)) {
    try {
      const customConfig = require(customConfigPath);
      Object.assign(config, customConfig);
    } catch (err) {
      console.warn(`Warning: Failed to load custom config: ${err.message}`);
    }
  }

  return config;
}

// Check Lighthouse scores against thresholds
function checkThresholds(lhr, pageThresholds, reportingConfig) {
  const result = {
    status: 'passed',
    scores: {},
    failures: [],
    warnings: []
  };

  for (const [category, catData] of Object.entries(lhr.categories)) {
    const score = catData.score !== null ? catData.score : 0;
    result.scores[category] = score;

    const thresholds = pageThresholds?.[category];
    if (!thresholds) continue;

    if (score < thresholds.error) {
      result.failures.push({
        category,
        score: (score * 100).toFixed(1),
        threshold: (thresholds.error * 100).toFixed(1),
        level: 'error'
      });
      result.status = 'failed';
    } else if (score < thresholds.warn) {
      result.warnings.push({
        category,
        score: (score * 100).toFixed(1),
        threshold: (thresholds.warn * 100).toFixed(1),
        level: 'warn'
      });
      if (result.status !== 'failed') {
        result.status = 'warned';
      }
    }
  }

  return result;
}

// Rotate old reports to keep only the most recent N
function rotateReports(outputDir, keepCount) {
  try {
    const files = fs.readdirSync(outputDir)
      .filter(f => f.endsWith('.html') || f.endsWith('.json'))
      .map(f => ({
        name: f,
        path: path.join(outputDir, f),
        time: fs.statSync(path.join(outputDir, f)).mtime.getTime()
      }))
      .sort((a, b) => b.time - a.time);

    // Keep only the most recent keepCount reports
    const toDelete = files.slice(keepCount * 2); // *2 because we have both .html and .json
    for (const file of toDelete) {
      try {
        fs.unlinkSync(file.path);
      } catch (err) {
        // Ignore deletion errors
      }
    }
  } catch (err) {
    console.warn(`Warning: Failed to rotate reports: ${err.message}`);
  }
}

// Main execution function
async function main() {
  const args = parseArgs(process.argv.slice(2));

  // Load and parse config
  let config;
  try {
    config = JSON.parse(fs.readFileSync(args.config, 'utf8'));
    config._configPath = args.config; // Store path for custom config lookup
  } catch (err) {
    console.error(`Error: Failed to parse config file: ${err.message}`);
    process.exit(1);
  }

  // Prepare results structure
  const results = {
    scenario: args.scenario,
    phase: 'performance',
    subphase: 'lighthouse',
    timestamp: new Date().toISOString(),
    pages: [],
    requirements: [],
    summary: {
      total_pages: 0,
      passed: 0,
      warned: 0,
      failed: 0
    }
  };

  let allPassed = true;
  const pages = config.pages || [];

  if (pages.length === 0) {
    console.warn('Warning: No pages defined in config');
    process.exit(0);
  }

  console.log(`\n🔍 Running Lighthouse audits for ${pages.length} page(s)...\n`);

  for (const page of pages) {
    results.summary.total_pages++;
    console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
    console.log(`📄 ${page.label || page.path} (${page.viewport || 'desktop'})`);
    console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`);

    const url = `${args.baseUrl}${page.path}`;
    let chrome;

    try {
      // Launch Chrome
      const chromeFlags = config.global_options?.chrome_flags || [
        '--headless',
        '--no-sandbox',
        '--disable-gpu',
        '--disable-dev-shm-usage'
      ];

      if (args.verbose) {
        console.log(`Launching Chrome with flags: ${chromeFlags.join(' ')}`);
      }

      chrome = await chromeLauncher.launch({ chromeFlags });

      // Build Lighthouse config
      const lhConfig = buildLighthouseConfig(config, page);

      if (args.verbose) {
        console.log(`Auditing: ${url}`);
        console.log(`Viewport: ${page.viewport || 'desktop'}`);
      }

      // Run Lighthouse audit
      const runnerResult = await lighthouse(url, {
        port: chrome.port,
        ...lhConfig
      });

      const lhr = runnerResult.lhr;

      // Save reports
      const timestamp = Date.now();
      const reportBasename = `${page.id}_${timestamp}`;

      const formats = config.reporting?.formats || ['json', 'html'];

      if (formats.includes('json')) {
        const jsonPath = path.join(args.outputDir, `${reportBasename}.json`);
        fs.writeFileSync(jsonPath, JSON.stringify(lhr, null, 2));
        if (args.verbose) {
          console.log(`Saved JSON report: ${jsonPath}`);
        }
      }

      if (formats.includes('html')) {
        const htmlPath = path.join(args.outputDir, `${reportBasename}.html`);
        fs.writeFileSync(htmlPath, runnerResult.report);
        if (args.verbose) {
          console.log(`Saved HTML report: ${htmlPath}`);
        }
      }

      // Check thresholds
      const pageResult = checkThresholds(lhr, page.thresholds, config.reporting);
      pageResult.page_id = page.id;
      pageResult.page_label = page.label || page.path;
      pageResult.url = url;
      pageResult.viewport = page.viewport || 'desktop';
      pageResult.timestamp = new Date(timestamp).toISOString();

      results.pages.push(pageResult);

      // Print scores
      console.log('Scores:');
      for (const [category, score] of Object.entries(pageResult.scores)) {
        const scorePercent = (score * 100).toFixed(1);
        const threshold = page.thresholds?.[category];
        let emoji = '✅';
        let status = 'PASS';

        if (threshold) {
          if (score < threshold.error) {
            emoji = '❌';
            status = 'FAIL';
          } else if (score < threshold.warn) {
            emoji = '⚠️ ';
            status = 'WARN';
          }
        }

        const thresholdStr = threshold
          ? ` (threshold: ${(threshold.error * 100).toFixed(0)}%)`
          : '';
        console.log(`  ${emoji} ${category}: ${scorePercent}%${thresholdStr} [${status}]`);
      }

      // Print failures and warnings
      if (pageResult.failures.length > 0) {
        console.log('\n❌ Failures:');
        for (const failure of pageResult.failures) {
          console.log(`  - ${failure.category}: ${failure.score}% < ${failure.threshold}% (threshold)`);
        }
      }

      if (pageResult.warnings.length > 0) {
        console.log('\n⚠️  Warnings:');
        for (const warning of pageResult.warnings) {
          console.log(`  - ${warning.category}: ${warning.score}% < ${warning.threshold}% (threshold)`);
        }
      }

      // Update summary
      if (pageResult.status === 'failed') {
        results.summary.failed++;
        allPassed = false;
        console.log(`\n❌ ${page.label || page.path} FAILED\n`);
      } else if (pageResult.status === 'warned') {
        results.summary.warned++;
        if (config.reporting?.fail_on_warn) {
          allPassed = false;
        }
        console.log(`\n⚠️  ${page.label || page.path} has warnings\n`);
      } else {
        results.summary.passed++;
        console.log(`\n✅ ${page.label || page.path} passed\n`);
      }

      // Link to requirements
      if (page.requirements && Array.isArray(page.requirements)) {
        for (const reqId of page.requirements) {
          const perfScore = (lhr.categories.performance?.score || 0).toFixed(2);
          const a11yScore = (lhr.categories.accessibility?.score || 0).toFixed(2);

          results.requirements.push({
            id: reqId,
            status: pageResult.status === 'failed' ? 'failed' : 'passed',
            evidence: `Lighthouse: ${page.label || page.path} - Performance: ${perfScore}, Accessibility: ${a11yScore}`,
            page_id: page.id,
            updated_at: pageResult.timestamp
          });
        }
      }

    } catch (err) {
      console.error(`\n❌ Error auditing ${page.label || page.path}:`);
      console.error(`   ${err.message}`);
      if (args.verbose) {
        console.error(err.stack);
      }

      results.pages.push({
        page_id: page.id,
        page_label: page.label || page.path,
        url,
        status: 'error',
        error: err.message
      });

      results.summary.failed++;
      allPassed = false;

    } finally {
      if (chrome) {
        await chrome.kill();
      }
    }
  }

  // Print summary
  console.log(`\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
  console.log(`📊 Summary`);
  console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n`);
  console.log(`  Total pages:   ${results.summary.total_pages}`);
  console.log(`  ✅ Passed:     ${results.summary.passed}`);
  console.log(`  ⚠️  Warned:     ${results.summary.warned}`);
  console.log(`  ❌ Failed:     ${results.summary.failed}\n`);

  // Write phase results for requirements sync
  try {
    const phaseResultPath = path.join(args.phaseResultsDir, 'lighthouse.json');
    fs.writeFileSync(phaseResultPath, JSON.stringify(results, null, 2));

    if (args.verbose) {
      console.log(`Phase results written to: ${phaseResultPath}`);
    }
  } catch (err) {
    console.error(`Warning: Failed to write phase results: ${err.message}`);
  }

  // Rotate old reports
  const keepCount = config.reporting?.keep_reports || 10;
  rotateReports(args.outputDir, keepCount);

  // Exit with appropriate code
  if (!allPassed) {
    console.log(`\n❌ Lighthouse audits failed\n`);
    process.exit(1);
  } else {
    console.log(`\n✅ All Lighthouse audits passed\n`);
    process.exit(0);
  }
}

// Run main function
main().catch(err => {
  console.error('Fatal error:', err);
  process.exit(1);
});
