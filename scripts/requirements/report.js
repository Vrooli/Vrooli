#!/usr/bin/env node

/**
 * requirements/report.js
 *
 * Lightweight coverage reporter for scenario requirements.
 * - Parses modular requirement registries (requirements/*.json)
 * - Computes aggregate counts by status and criticality gap
 * - Reads live phase results from coverage/phase-results
 * - Emits JSON, markdown, or trace outputs
 * - Supports sync mode to update requirement files based on live evidence
 *
 * Refactored to use modular architecture for better maintainability
 */

'use strict';

const fs = require('node:fs');
const path = require('node:path');

// Import modular components
const discovery = require('./lib/discovery');
const parser = require('./lib/parser');
const evidence = require('./lib/evidence');
const enrichment = require('./lib/enrichment');
const sync = require('./lib/sync');
const output = require('./lib/output');
const snapshot = require('./lib/snapshot');
const prdParser = require('../prd/parser');

/**
 * Parse command line arguments
 * @param {string[]} argv - Command line arguments
 * @returns {import('./lib/types').ParseOptions} Parsed options
 */
function parseArgs(argv) {
  const options = {
    scenario: '',
    format: 'json',
    includePending: false,
    output: '',
    mode: 'report',
    phase: '',
    pruneStale: false,
    testCommands: [],
    allowPartialSync: false,
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--scenario':
        options.scenario = argv[i + 1] || '';
        i += 1;
        break;
      case '--format': {
        const value = (argv[i + 1] || 'json').toLowerCase();
        if (value === 'markdown' || value === 'json' || value === 'trace') {
          options.format = value;
        }
        i += 1;
        break;
      }
      case '--mode': {
        const mode = (argv[i + 1] || '').toLowerCase();
        if (['report', 'phase-inspect', 'validate', 'sync'].includes(mode)) {
          options.mode = mode;
        }
        i += 1;
        break;
      }
      case '--phase':
        options.phase = argv[i + 1] || '';
        i += 1;
        break;
      case '--include-pending':
        options.includePending = true;
        break;
      case '--output':
        options.output = argv[i + 1] || '';
        i += 1;
        break;
      case '--prune-stale':
        options.pruneStale = true;
        break;
      case '--allow-partial-sync':
        options.allowPartialSync = true;
        break;
      default:
        break;
    }
  }

  const envCommands = process.env.REQUIREMENTS_SYNC_TEST_COMMANDS;
  if (envCommands) {
    try {
      const parsed = JSON.parse(envCommands);
      if (Array.isArray(parsed)) {
        options.testCommands = parsed;
      }
    } catch (error) {
      console.warn(`requirements/report: unable to parse REQUIREMENTS_SYNC_TEST_COMMANDS - ${error.message}`);
    }
  }

  if (!options.scenario) {
    throw new Error('Missing required --scenario argument');
  }

  return options;
}

function evaluatePhaseStatus(rawPayload) {
  if (!rawPayload) {
    return { ok: false, reason: 'missing_metadata', message: 'Phase execution metadata missing (set REQUIREMENTS_SYNC_PHASE_STATUS) or rerun via test/run-tests.sh.' };
  }
  let parsed;
  try {
    parsed = JSON.parse(rawPayload);
  } catch (error) {
    return { ok: false, reason: 'invalid_metadata', message: `Unable to parse REQUIREMENTS_SYNC_PHASE_STATUS (${error.message})` };
  }
  if (!Array.isArray(parsed) || parsed.length === 0) {
    return { ok: false, reason: 'empty_metadata', message: 'Phase execution metadata empty; rerun the full suite before syncing requirements.' };
  }

  const missing = [];
  const skipped = [];
  parsed.forEach((entry) => {
    if (!entry || entry.optional === true) {
      return;
    }
    const recorded = entry.recorded === true;
    const normalizedStatus = (entry.status || '').toLowerCase();
    if (!recorded) {
      missing.push(entry.phase || 'unknown');
      return;
    }
    if (['skipped', 'missing', 'not_executable', 'not_run', 'unknown'].includes(normalizedStatus)) {
      skipped.push(entry.phase || 'unknown');
    }
  });

  if (missing.length === 0 && skipped.length === 0) {
    return { ok: true, phases: parsed };
  }

  const problems = [];
  if (missing.length) {
    problems.push(`missing phases: ${missing.join(', ')}`);
  }
  if (skipped.length) {
    problems.push(`skipped phases: ${skipped.join(', ')}`);
  }

  return {
    ok: false,
    reason: 'partial_suite',
    missing,
    skipped,
    message: `Partial suite detected (${problems.join('; ')}). Run the full test suite before syncing requirements or pass --allow-partial-sync to override.`,
  };
}

/**
 * Main execution function
 */
function main() {
  const options = parseArgs(process.argv.slice(2));
  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const requirementSources = discovery.collectRequirementFiles(scenarioRoot);

  if (requirementSources.length === 0) {
    throw new Error('No requirements files discovered');
  }

  // Parse all requirement files and build indices
  const requirements = [];
  const fileRequirementMap = new Map();
  const requirementIndex = new Map();

  requirementSources.forEach((source) => {
    const { requirements: fileRequirements } = parser.parseRequirementFile(source.path);
    if (!fileRequirements || fileRequirements.length === 0) {
      return;
    }

    if (!fileRequirementMap.has(source.path)) {
      fileRequirementMap.set(source.path, []);
    }

    fileRequirements.forEach((requirement) => {
      if (!requirement || !requirement.id) {
        return;
      }

      // Check for duplicate requirement IDs
      if (requirementIndex.has(requirement.id)) {
        const previous = requirementIndex.get(requirement.id);
        const previousPath = previous.__meta && previous.__meta.filePath ? previous.__meta.filePath : 'unknown';
        throw new Error(`Duplicate requirement id '${requirement.id}' in ${source.path} (already defined in ${previousPath})`);
      }

      // Update metadata with file path
      if (requirement.__meta) {
        requirement.__meta.filePath = source.path;
      }
      if (Array.isArray(requirement.validations)) {
        requirement.validations.forEach((validation) => {
          if (validation && validation.__meta) {
            validation.__meta.filePath = source.path;
          }
        });
      }

      fileRequirementMap.get(source.path).push(requirement);
      requirementIndex.set(requirement.id, requirement);
      requirements.push(requirement);
    });
  });

  // Handle phase-inspect mode (early return)
  if (options.mode === 'phase-inspect') {
    const phaseOutput = output.renderPhaseInspect(options, requirements, scenarioRoot);
    if (options.output) {
      const resolvedOutput = path.resolve(process.cwd(), options.output);
      fs.mkdirSync(path.dirname(resolvedOutput), { recursive: true });
      fs.writeFileSync(resolvedOutput, phaseOutput, 'utf8');
    } else {
      process.stdout.write(phaseOutput);
    }
    return;
  }

  // Load phase results and enrich requirements with live evidence
  const { phaseResults, requirementEvidence } = evidence.loadPhaseResults(scenarioRoot);
  const manualManifest = evidence.loadManualManifest(scenarioRoot);
  if (manualManifest && manualManifest.latestByRequirement.size > 0) {
    evidence.applyManualManifest(requirementEvidence, manualManifest);
  }
  enrichment.enrichValidationResults(requirements, { phaseResults, requirementEvidence });
  enrichment.aggregateRequirementStatuses(requirements, requirementIndex);

  // Handle sync mode
  if (options.mode === 'sync') {
    const allowPartialSync = options.allowPartialSync || process.env.REQUIREMENTS_SYNC_ALLOW_PARTIAL === '1';
    if (!allowPartialSync) {
      const phaseGuard = evaluatePhaseStatus(process.env.REQUIREMENTS_SYNC_PHASE_STATUS || '');
      if (!phaseGuard.ok) {
        throw new Error(phaseGuard.message || 'Run the full test suite before syncing requirements.');
      }
    }

    const syncResult = sync.syncRequirementRegistry(
      fileRequirementMap,
      scenarioRoot,
      options,
      { manualManifest }
    );

    const summary = enrichment.computeSummary(requirements);
    const manifestEntry = snapshot.parseManifestEntry(
      process.env.REQUIREMENTS_SYNC_MANIFEST_ENTRY,
      process.env.REQUIREMENTS_SYNC_RUN_LOG
    );

    const snapshotResult = snapshot.writeSnapshot({
      scenarioRoot,
      scenarioName: options.scenario,
      summary,
      testCommands: options.testCommands,
      fileSnapshots: syncResult.fileSnapshots,
      manifestEntry,
      manifestPath: process.env.REQUIREMENTS_SYNC_RUN_LOG || null,
      manualManifest,
    });

    if (snapshotResult && snapshotResult.payload) {
      const statusMap = new Map();
      const targets = snapshotResult.payload.operational_targets || [];
      targets.forEach((target) => {
        if (target && target.target_id) {
          statusMap.set(target.target_id.toUpperCase(), target.status || 'pending');
        }
      });
      if (statusMap.size > 0) {
        const prdSyncResult = prdParser.syncOperationalTargetCheckboxes(scenarioRoot, statusMap);
        if (prdSyncResult && prdSyncResult.updated) {
          console.log(`🔄 PRD checkboxes updated (${prdSyncResult.changedTargets.length} target(s))`);
        }
      }
    }

    sync.printSyncSummary(syncResult);
    return;
  }

  // Generate report output
  const summary = enrichment.computeSummary(requirements);

  let outputContent = '';
  if (options.format === 'markdown') {
    outputContent = output.renderMarkdown(summary, requirements, options.includePending);
  } else if (options.format === 'trace') {
    outputContent = output.renderTrace(requirements, scenarioRoot);
  } else {
    outputContent = output.renderJSON(summary, requirements);
  }

  if (options.output) {
    const resolvedOutput = path.resolve(process.cwd(), options.output);
    fs.mkdirSync(path.dirname(resolvedOutput), { recursive: true });
    fs.writeFileSync(resolvedOutput, outputContent, 'utf8');
  } else {
    process.stdout.write(outputContent);
  }
}

// Execute main function with error handling
try {
  main();
} catch (error) {
  console.error(`requirements/report failed: ${error.message}`);
  process.exitCode = 1;
}
