#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const discovery = require('./lib/discovery');
const parser = require('./lib/parser');
const manual = require('./lib/manual');

function usage(message) {
  if (message) {
    console.error(message);
  }
  console.error('Usage: node scripts/requirements/manual-log.js --scenario <name> --requirement <REQ-ID> [options]');
  console.error('\nOptions:');
  console.error('  --status <passed|failed>        Manual validation result (default: passed)');
  console.error('  --notes "text"                 Notes or summary of what was validated');
  console.error('  --artifact <path>              Relative path to supporting artifact (screenshots, docs, etc.)');
  console.error('  --validated-by <name>          Person or agent who performed the validation (default: $USER/VROOLI_AGENT_ID)');
  console.error('  --validated-at <ISO8601>       Override timestamp (defaults to now)');
  console.error('  --expires-in <days>            Validity window in days (default: 30)');
  console.error('  --expires-at <ISO8601>         Explicit expiration timestamp (overrides --expires-in)');
  console.error('  --manifest <relative-or-abs>   Override manifest path (defaults to coverage/manual-validations/log.jsonl)');
  console.error('  --dry-run                      Print the manifest entry without writing to disk');
  process.exit(1);
}

function parseArgs(argv) {
  const options = {
    scenario: '',
    requirement: '',
    status: 'passed',
    notes: '',
    artifact: '',
    validatedBy: process.env.VROOLI_AGENT_ID || process.env.USER || 'manual-validator',
    validatedAt: null,
    expiresInDays: 30,
    expiresAt: null,
    manifest: '',
    dryRun: false,
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--scenario':
        options.scenario = argv[++i] || '';
        break;
      case '--requirement':
        options.requirement = argv[++i] || '';
        break;
      case '--status':
        options.status = argv[++i] || 'passed';
        break;
      case '--notes':
        options.notes = argv[++i] || '';
        break;
      case '--artifact':
        options.artifact = argv[++i] || '';
        break;
      case '--validated-by':
        options.validatedBy = argv[++i] || options.validatedBy;
        break;
      case '--validated-at':
        options.validatedAt = argv[++i] || null;
        break;
      case '--expires-in':
        options.expiresInDays = parseInt(argv[++i] || '30', 10) || 30;
        break;
      case '--expires-at':
        options.expiresAt = argv[++i] || null;
        break;
      case '--manifest':
        options.manifest = argv[++i] || '';
        break;
      case '--dry-run':
        options.dryRun = true;
        break;
      case '--help':
      case '-h':
        usage();
        break;
      default:
        usage(`Unknown argument: ${arg}`);
        break;
    }
  }

  if (!options.scenario || !options.requirement) {
    usage('Scenario and requirement are required.');
  }

  return options;
}

function findRequirement(scenarioRoot, requirementId) {
  const sources = discovery.collectRequirementFiles(scenarioRoot);
  for (const source of sources) {
    const { requirements } = parser.parseRequirementFile(source.path);
    if (!Array.isArray(requirements)) {
      continue;
    }
    for (const requirement of requirements) {
      if (requirement && requirement.id === requirementId) {
        return requirement;
      }
    }
  }
  return null;
}

function computeExpiration(validatedAt, options) {
  if (options.expiresAt) {
    return options.expiresAt;
  }
  const base = validatedAt ? Date.parse(validatedAt) : Date.now();
  const expiresMs = base + (options.expiresInDays || 30) * 24 * 60 * 60 * 1000;
  return new Date(expiresMs).toISOString();
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const requirement = findRequirement(scenarioRoot, options.requirement);
  if (!requirement) {
    throw new Error(`Requirement '${options.requirement}' not found under ${options.scenario}`);
  }

  const validatedAt = options.validatedAt || new Date().toISOString();
  const expiresAt = computeExpiration(validatedAt, options);
  const status = manual.normalizeManualStatus(options.status);

  const entry = {
    scenario: options.scenario,
    requirement_id: options.requirement,
    status,
    validated_at: validatedAt,
    expires_at: expiresAt,
    validated_by: options.validatedBy,
    notes: options.notes || undefined,
    artifact_path: options.artifact || undefined,
    recorded_at: new Date().toISOString(),
  };

  const manifestPath = manual.resolveManifestPath(scenarioRoot, options.manifest);
  const relativePath = path.relative(scenarioRoot, manifestPath);

  if (options.dryRun) {
    console.log('Dry run: manual validation entry');
    console.log(JSON.stringify(entry, null, 2));
    console.log(`Manifest path: ${manifestPath}`);
    return;
  }

  fs.mkdirSync(path.dirname(manifestPath), { recursive: true });
  fs.appendFileSync(manifestPath, `${JSON.stringify(entry)}\n`, 'utf8');

  console.log(`Logged manual validation for ${options.requirement} (${status})`);
  console.log(`  Scenario:      ${options.scenario}`);
  console.log(`  Manifest:      ${relativePath || manifestPath}`);
  console.log(`  Validated at:  ${validatedAt}`);
  console.log(`  Expires at:    ${expiresAt}`);
}

try {
  main();
} catch (error) {
  console.error(`manual-log failed: ${error.message}`);
  process.exitCode = 1;
}
