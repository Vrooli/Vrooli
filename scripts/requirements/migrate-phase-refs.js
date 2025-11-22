#!/usr/bin/env node
'use strict';

/**
 * Migration script to fix validation refs pointing to generic phase scripts.
 * Searches for BATS tests with [REQ:ID] tags and updates requirement files accordingly.
 *
 * Usage: node scripts/requirements/migrate-phase-refs.js --scenario <name> [--dry-run]
 */

const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const APP_ROOT = path.resolve(__dirname, '..', '..');

function parseArgs(argv) {
  const options = { scenario: '', dryRun: false };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--scenario') {
      options.scenario = argv[i + 1] || '';
      i += 1;
    } else if (arg === '--dry-run') {
      options.dryRun = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    }
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/requirements/migrate-phase-refs.js --scenario <name> [--dry-run]');
  console.log('');
  console.log('Migrates validation refs from generic phase scripts to specific BATS test files.');
  console.log('');
  console.log('Options:');
  console.log('  --scenario <name>  Scenario name to migrate');
  console.log('  --dry-run         Show what would be changed without modifying files');
  console.log('  --help, -h        Show this help message');
}

function resolveScenarioRoot(scenario) {
  if (!scenario) {
    throw new Error('Missing required --scenario argument');
  }
  const candidate = path.join(APP_ROOT, 'scenarios', scenario);
  if (fs.existsSync(candidate) && fs.statSync(candidate).isDirectory()) {
    return candidate;
  }
  throw new Error(`Unable to locate scenario '${scenario}'`);
}

/**
 * Find all BATS test files with [REQ:ID] tags
 * Returns: Map<requirementId, batsFilePath>
 */
function findBatsTestMappings(scenarioRoot) {
  const mappings = new Map();
  const cliTestDir = path.join(scenarioRoot, 'test', 'cli');

  if (!fs.existsSync(cliTestDir)) {
    console.log(`⚠️  No test/cli directory found at ${cliTestDir}`);
    return mappings;
  }

  const batsFiles = fs.readdirSync(cliTestDir)
    .filter(f => f.endsWith('.bats'))
    .map(f => path.join(cliTestDir, f));

  batsFiles.forEach(batsFile => {
    const content = fs.readFileSync(batsFile, 'utf8');
    const reqTagPattern = /\[REQ:([A-Z][A-Z0-9]+-[A-Z0-9-]+)\]/g;
    let match;

    while ((match = reqTagPattern.exec(content)) !== null) {
      const reqId = match[1];
      const relativePath = path.relative(scenarioRoot, batsFile);

      if (!mappings.has(reqId)) {
        mappings.set(reqId, relativePath);
      }
    }
  });

  return mappings;
}

/**
 * Collect all requirement JSON files
 */
function collectRequirementFiles(scenarioRoot) {
  const requirementsDir = path.join(scenarioRoot, 'requirements');
  if (!fs.existsSync(requirementsDir)) {
    throw new Error('No requirements/ directory found');
  }

  const results = [];
  const stack = [requirementsDir];

  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });

    entries.forEach(entry => {
      if (entry.name.startsWith('.')) return;

      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
      } else if (entry.isFile() && entry.name.endsWith('.json')) {
        results.push(fullPath);
      }
    });
  }

  return results;
}

/**
 * Migrate a single requirement file
 */
function migrateRequirementFile(filePath, batsMapping, dryRun) {
  const content = fs.readFileSync(filePath, 'utf8');
  const data = JSON.parse(content);

  if (!Array.isArray(data.requirements)) {
    return { changed: false, updates: [] };
  }

  let changed = false;
  const updates = [];

  data.requirements.forEach(req => {
    if (!req.validation || !Array.isArray(req.validation)) {
      return;
    }

    req.validation.forEach((validation, idx) => {
      const isTestType = (validation.type || '').toLowerCase() === 'test';
      const phaseScriptPattern = /^test\/phases\/test-[a-z]+\.sh$/;
      const isPhaseScript = validation.ref && phaseScriptPattern.test(validation.ref);

      if (isTestType && isPhaseScript) {
        const batsFile = batsMapping.get(req.id);

        if (batsFile) {
          updates.push({
            requirementId: req.id,
            oldRef: validation.ref,
            newRef: batsFile,
            file: path.relative(APP_ROOT, filePath)
          });

          validation.ref = batsFile;
          validation.notes = validation.notes || `CLI integration tests with [REQ:${req.id}] tags`;
          changed = true;
        } else {
          updates.push({
            requirementId: req.id,
            oldRef: validation.ref,
            newRef: null,
            file: path.relative(APP_ROOT, filePath),
            warning: `No BATS test found with [REQ:${req.id}] tag`
          });
        }
      }
    });
  });

  if (changed && !dryRun) {
    // Write with 2-space indentation to match existing format
    fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n', 'utf8');
  }

  return { changed, updates };
}

function main() {
  const options = parseArgs(process.argv.slice(2));

  if (options.help) {
    printHelp();
    process.exit(0);
  }

  if (!options.scenario) {
    console.error('❌ Missing required --scenario argument');
    printHelp();
    process.exit(1);
  }

  console.log(`🔍 Analyzing scenario: ${options.scenario}`);

  const scenarioRoot = resolveScenarioRoot(options.scenario);
  const batsMapping = findBatsTestMappings(scenarioRoot);

  console.log(`✅ Found ${batsMapping.size} BATS test mappings`);

  const requirementFiles = collectRequirementFiles(scenarioRoot);
  console.log(`📋 Checking ${requirementFiles.length} requirement files\n`);

  let totalChanges = 0;
  let totalWarnings = 0;
  const allUpdates = [];

  requirementFiles.forEach(filePath => {
    const result = migrateRequirementFile(filePath, batsMapping, options.dryRun);

    if (result.updates.length > 0) {
      allUpdates.push(...result.updates);
      totalChanges += result.updates.filter(u => u.newRef).length;
      totalWarnings += result.updates.filter(u => !u.newRef).length;
    }
  });

  // Print summary
  if (allUpdates.length === 0) {
    console.log('✅ No phase script references found - all validations already specific!');
    process.exit(0);
  }

  console.log(`\n${'='.repeat(80)}`);
  console.log(`MIGRATION SUMMARY${options.dryRun ? ' (DRY RUN)' : ''}`);
  console.log(`${'='.repeat(80)}\n`);

  // Group by file
  const byFile = new Map();
  allUpdates.forEach(update => {
    if (!byFile.has(update.file)) {
      byFile.set(update.file, []);
    }
    byFile.get(update.file).push(update);
  });

  byFile.forEach((updates, file) => {
    console.log(`📄 ${file}`);
    updates.forEach(update => {
      if (update.newRef) {
        console.log(`   ✅ ${update.requirementId}`);
        console.log(`      ${update.oldRef} → ${update.newRef}`);
      } else {
        console.log(`   ⚠️  ${update.requirementId}`);
        console.log(`      ${update.oldRef} → NO BATS TEST FOUND`);
        console.log(`      ${update.warning}`);
      }
    });
    console.log();
  });

  console.log(`${'='.repeat(80)}`);
  console.log(`✅ Successfully migrated: ${totalChanges}`);
  if (totalWarnings > 0) {
    console.log(`⚠️  Warnings: ${totalWarnings}`);
  }
  console.log(`${'='.repeat(80)}\n`);

  if (options.dryRun) {
    console.log('💡 This was a dry run. No files were modified.');
    console.log('   Run without --dry-run to apply changes.\n');
  } else {
    console.log('✅ Migration complete! Files have been updated.\n');
    console.log('💡 Next steps:');
    console.log('   1. Review changes: git diff scenarios/' + options.scenario + '/requirements/');
    console.log('   2. Validate: node scripts/requirements/validate.js --scenario ' + options.scenario);
    console.log('   3. Commit: git add scenarios/' + options.scenario + '/requirements/ && git commit\n');
  }
}

try {
  main();
} catch (error) {
  console.error(`❌ ${error.message}`);
  process.exit(1);
}
