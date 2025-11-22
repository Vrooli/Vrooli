#!/usr/bin/env node
'use strict';

/**
 * One-time migration script to remove _sync_metadata from all requirement files
 * Run this BEFORE deploying the new sync logic
 */

const fs = require('node:fs');
const path = require('node:path');
const discovery = require('./lib/discovery');

function parseArgs(argv) {
  return {
    scenario: argv.find((arg, i) => argv[i - 1] === '--scenario') || '',
    dryRun: argv.includes('--dry-run'),
    verify: argv.includes('--verify'),
  };
}

function removeSyncMetadata(parsed) {
  let changed = false;

  // Remove module-level last_synced_at
  if (parsed._metadata && parsed._metadata.last_synced_at) {
    delete parsed._metadata.last_synced_at;
    changed = true;
  }

  // Remove requirement-level and validation-level _sync_metadata
  if (Array.isArray(parsed.requirements)) {
    parsed.requirements.forEach(req => {
      if (req._sync_metadata) {
        delete req._sync_metadata;
        changed = true;
      }

      // NOTE: Property name is 'validation' (singular) in JSON
      if (Array.isArray(req.validation)) {
        req.validation.forEach(val => {
          if (val._sync_metadata) {
            delete val._sync_metadata;
            changed = true;
          }
        });
      }
    });
  }

  return changed;
}

function verifyCleanFile(parsed, filePath) {
  const issues = [];

  if (parsed._metadata && parsed._metadata.last_synced_at) {
    issues.push('Module-level _metadata.last_synced_at still present');
  }

  if (Array.isArray(parsed.requirements)) {
    parsed.requirements.forEach((req, reqIdx) => {
      if (req._sync_metadata) {
        issues.push(`Requirement[${reqIdx}] (${req.id}) has _sync_metadata`);
      }

      if (Array.isArray(req.validation)) {
        req.validation.forEach((val, valIdx) => {
          if (val._sync_metadata) {
            issues.push(`Requirement[${reqIdx}].validation[${valIdx}] has _sync_metadata`);
          }
        });
      }
    });
  }

  if (issues.length > 0) {
    console.error(`\n❌ Verification failed for ${filePath}:`);
    issues.forEach(issue => console.error(`   - ${issue}`));
    return false;
  }

  return true;
}

function main() {
  const options = parseArgs(process.argv.slice(2));

  if (!options.scenario) {
    throw new Error('Missing --scenario argument. Usage: node migrate-remove-sync-metadata.js --scenario <name> [--dry-run] [--verify]');
  }

  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const files = discovery.collectRequirementFiles(scenarioRoot);

  console.log(`\n📋 Found ${files.length} requirement files in ${options.scenario}\n`);

  let modifiedCount = 0;
  let verificationFailed = false;
  let errorCount = 0;

  files.forEach(file => {
    let content, parsed;

    try {
      content = fs.readFileSync(file.path, 'utf8');
    } catch (error) {
      console.error(`❌ Failed to read ${file.relative}: ${error.message}`);
      errorCount++;
      return;
    }

    try {
      parsed = JSON.parse(content);
    } catch (error) {
      console.error(`❌ Failed to parse ${file.relative}: ${error.message}`);
      errorCount++;
      return;
    }

    const changed = removeSyncMetadata(parsed);

    if (changed) {
      modifiedCount++;

      if (options.dryRun) {
        console.log(`[DRY RUN] Would clean: ${file.relative}`);
      } else {
        console.log(`Cleaning: ${file.relative}`);
        try {
          fs.writeFileSync(file.path, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
        } catch (error) {
          console.error(`❌ Failed to write ${file.relative}: ${error.message}`);
          errorCount++;
          return;
        }
      }
    }

    // Verify if requested
    if (options.verify && !options.dryRun) {
      if (!verifyCleanFile(parsed, file.relative)) {
        verificationFailed = true;
      }
    }
  });

  console.log(`\n✅ Total files ${options.dryRun ? 'that would be modified' : 'modified'}: ${modifiedCount}/${files.length}\n`);

  if (errorCount > 0) {
    throw new Error(`Encountered ${errorCount} error(s) during migration`);
  }

  if (verificationFailed) {
    throw new Error('Verification failed - some files still contain sync metadata');
  }

  if (options.dryRun) {
    console.log('💡 Run without --dry-run to apply changes\n');
  } else {
    console.log('✨ Migration complete! Run tests to regenerate sync metadata in coverage/sync/\n');
  }
}

try {
  main();
} catch (error) {
  console.error(`\n❌ Migration failed: ${error.message}\n`);
  process.exitCode = 1;
}
