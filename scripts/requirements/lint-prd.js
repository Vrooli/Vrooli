#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const discovery = require('./lib/discovery');
const parser = require('./lib/parser');
const prdParser = require('../prd/parser');

function parseArgs(argv) {
  const options = {
    scenario: '',
    format: 'text',
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--scenario':
        options.scenario = argv[i + 1] || '';
        i += 1;
        break;
      case '--format': {
        const format = (argv[i + 1] || 'text').toLowerCase();
        if (format === 'json') {
          options.format = 'json';
        }
        i += 1;
        break;
      }
      case '--help':
      case '-h':
        options.showHelp = true;
        break;
      default:
        if (!options.scenario && !arg.startsWith('--')) {
          options.scenario = arg;
        }
        break;
    }
  }

  return options;
}

function extractTargetId(reference) {
  if (!reference) {
    return null;
  }
  const match = reference.match(/OT-[Pp][0-2]-\d{3}/);
  if (match && match[0]) {
    return match[0].toUpperCase();
  }
  return null;
}

function collectRequirements(requirementSources) {
  const results = [];
  requirementSources.forEach((source) => {
    const { requirements } = parser.parseRequirementFile(source.path);
    requirements.forEach((req) => {
      const meta = req.__meta || {};
      results.push({
        id: req.id,
        prd_ref: req.prd_ref || '',
        file: meta.filePath || source.path,
      });
    });
  });
  return results;
}

function relativize(filePath, baseDir) {
  try {
    return path.relative(baseDir, filePath);
  } catch (error) {
    return filePath;
  }
}

function lintScenario(options) {
  if (!options.scenario) {
    throw new Error('Scenario name required (pass --scenario <name>)');
  }

  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const prd = prdParser.loadOperationalTargets(scenarioRoot, 'scenario', options.scenario);
  const requirementSources = discovery.collectRequirementFiles(scenarioRoot);
  const requirements = collectRequirements(requirementSources);

  const result = {
    scenario: options.scenario,
    status: 'ok',
    targets_without_requirements: [],
    requirements_without_targets: [],
    missing_prd: false,
  };

  if (prd.status === 'missing') {
    result.status = 'missing_prd';
    result.missing_prd = true;
    return result;
  }

  const targetMap = new Map();
  prd.targets.forEach((target) => {
    if (!target || !target.id) {
      return;
    }
    targetMap.set(target.id.toUpperCase(), target);
  });

  const coverageMap = new Map();
  requirements.forEach((requirement) => {
    const targetId = extractTargetId(requirement.prd_ref);
    if (!targetId) {
      result.requirements_without_targets.push({
        requirement_id: requirement.id,
        file: relativize(requirement.file, scenarioRoot),
        reason: requirement.prd_ref ? 'missing OT-* id in prd_ref' : 'prd_ref not set',
      });
      return;
    }
    if (!targetMap.has(targetId)) {
      result.requirements_without_targets.push({
        requirement_id: requirement.id,
        file: relativize(requirement.file, scenarioRoot),
        reason: `PRD target ${targetId} not found`,
      });
      return;
    }
    if (!coverageMap.has(targetId)) {
      coverageMap.set(targetId, []);
    }
    coverageMap.get(targetId).push(requirement.id);
  });

  targetMap.forEach((target, key) => {
    if (!coverageMap.has(key)) {
      result.targets_without_requirements.push({
        target_id: key,
        title: target.title || '',
        criticality: target.criticality || '',
      });
    }
  });

  if (result.targets_without_requirements.length > 0 || result.requirements_without_targets.length > 0) {
    result.status = 'issues';
  }

  return result;
}

function printText(result) {
  if (result.status === 'missing_prd') {
    console.log('❌ PRD.md not found – cannot verify requirements mapping.');
    return 1;
  }

  if (result.status === 'ok') {
    console.log('✅ PRD ↔ requirements mapping looks healthy.');
    return 0;
  }

  console.log('❌ PRD ↔ requirements mismatch detected.');
  if (result.targets_without_requirements.length > 0) {
    console.log('  • Targets without linked requirements:');
    result.targets_without_requirements.forEach((target) => {
      console.log(`    - ${target.target_id} (${target.criticality || 'unknown'}) ${target.title || ''}`);
    });
  }
  if (result.requirements_without_targets.length > 0) {
    console.log('  • Requirements referencing missing PRD targets:');
    result.requirements_without_targets.forEach((entry) => {
      console.log(`    - ${entry.requirement_id} (${entry.file}): ${entry.reason}`);
    });
  }
  return 1;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.showHelp) {
    console.log('Usage: node scripts/requirements/lint-prd.js --scenario <name> [--format json]');
    process.exit(0);
  }

  const result = lintScenario(options);
  if (options.format === 'json') {
    console.log(JSON.stringify(result, null, 2));
    process.exit(result.status === 'ok' ? 0 : 1);
  }

  const exitCode = printText(result);
  process.exit(exitCode);
}

try {
  main();
} catch (error) {
  console.error(`requirements/lint-prd: ${error.message}`);
  process.exit(1);
}
