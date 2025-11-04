#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { spawnSync } = require('node:child_process');

const APP_ROOT = path.resolve(__dirname, '..', '..');
const SCHEMA_DIR = path.join(APP_ROOT, 'scripts', 'requirements', 'schemas');
const CRITICAL_LEVELS = new Set(['P0', 'P1']);
const OPTIONAL_VALIDATION_STATUSES = new Set(['planned', 'not_implemented', 'skipped', 'todo']);
const VALID_PHASES = new Set(['structure', 'dependencies', 'unit', 'integration', 'business', 'performance', 'cli']);

function parseArgs(argv) {
  const options = { scenario: '', quiet: false };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--scenario') {
      options.scenario = argv[i + 1] || '';
      i += 1;
    } else if (arg === '--quiet') {
      options.quiet = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    }
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/requirements/validate.js --scenario <name>');
  console.log('Validates scenario requirement files against JSON schemas, checks child references, and ensures imports exist.');
}

function resolveScenarioRoot(baseDir, scenario) {
  if (!scenario) {
    throw new Error('Missing required --scenario argument');
  }
  let current = path.resolve(baseDir);
  while (true) {
    const candidate = path.join(current, 'scenarios', scenario);
    if (fs.existsSync(candidate) && fs.statSync(candidate).isDirectory()) {
      return candidate;
    }
    if (path.basename(current) === scenario) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) {
      break;
    }
    current = parent;
  }
  if (process.env.VROOLI_ROOT) {
    const candidate = path.join(process.env.VROOLI_ROOT, 'scenarios', scenario);
    if (fs.existsSync(candidate) && fs.statSync(candidate).isDirectory()) {
      return candidate;
    }
  }
  throw new Error(`Unable to locate scenario '${scenario}'. Start from the repo root or export VROOLI_ROOT.`);
}

function collectRequirementFiles(scenarioRoot) {
  const docsPath = path.join(scenarioRoot, 'docs', 'requirements.yaml');
  if (fs.existsSync(docsPath)) {
    return [{ path: docsPath, isIndex: true }];
  }
  const requirementsDir = path.join(scenarioRoot, 'requirements');
  if (!fs.existsSync(requirementsDir)) {
    throw new Error('No requirements registry found (expected docs/requirements.yaml or requirements/ folder).');
  }
  const results = [];
  const stack = [requirementsDir];
  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    entries.forEach((entry) => {
      if (entry.name.startsWith('.')) {
        return;
      }
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
        return;
      }
      if (!entry.isFile()) {
        return;
      }
      if (!/\.ya?ml$/i.test(entry.name)) {
        return;
      }
      const relativeToBase = path.relative(requirementsDir, fullPath);
      results.push({
        path: fullPath,
        isIndex: relativeToBase === 'index.yaml',
      });
    });
  }
  if (results.length === 0) {
    throw new Error('requirements/ exists but no YAML files were found.');
  }
  return results;
}

function gatherValidations(requirement) {
  const results = [];
  if (Array.isArray(requirement.validation)) {
    results.push(...requirement.validation);
  }
  if (Array.isArray(requirement.validations)) {
    results.push(...requirement.validations);
  }
  return results;
}

function runCommand(command, args, label) {
  const result = spawnSync(command, args, { encoding: 'utf8' });
  if (result.error) {
    throw new Error(`${label} failed to run: ${result.error.message}`);
  }
  return result;
}

function convertYamlToJson(filePath) {
  const result = runCommand('js-yaml', [filePath], `js-yaml (${filePath})`);
  if (result.status !== 0) {
    const errorOutput = result.stderr || result.stdout || 'unknown parser error';
    throw new Error(`Unable to parse ${filePath}: ${errorOutput.trim()}`);
  }
  const jsonText = result.stdout;
  try {
    return { jsonText, data: JSON.parse(jsonText) };
  } catch (error) {
    throw new Error(`js-yaml output for ${filePath} was not valid JSON: ${error.message}`);
  }
}

function runSchemaValidation(schemaPath, jsonText, sourcePath, tempDir) {
  const dataFile = path.join(tempDir, `${path.basename(sourcePath)}.json`);
  fs.writeFileSync(dataFile, jsonText, 'utf8');
  const result = runCommand('ajv', ['validate', '-s', schemaPath, '-d', dataFile], `ajv (${sourcePath})`);
  fs.unlinkSync(dataFile);
  if (result.status !== 0) {
    const output = (result.stderr || result.stdout || '').trim();
    throw new Error(`Schema validation failed for ${sourcePath}: ${output || 'ajv returned a non-zero exit code'}`);
  }
}

function relativeToRoot(filePath) {
  return path.relative(APP_ROOT, filePath) || filePath;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    printHelp();
    process.exit(0);
  }
  if (!options.scenario) {
    throw new Error('Missing required --scenario argument.');
  }
  const scenarioRoot = resolveScenarioRoot(process.cwd(), options.scenario);
  const requirementSources = collectRequirementFiles(scenarioRoot);
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vrooli-req-'));

  const errors = [];
  const warnings = [];
  const requirementIndex = new Map();
  const childRefs = [];
  const importChecks = [];

  try {
    requirementSources.forEach((source) => {
      const schemaName = source.isIndex ? 'index.schema.json' : 'module.schema.json';
      const schemaPath = path.join(SCHEMA_DIR, schemaName);
      if (!fs.existsSync(schemaPath)) {
        errors.push(`Missing schema definition ${schemaName} (expected at ${schemaPath}).`);
        return;
      }
      let parsed;
      try {
        parsed = convertYamlToJson(source.path);
      } catch (error) {
        errors.push(error.message);
        return;
      }
      try {
        runSchemaValidation(schemaPath, parsed.jsonText, source.path, tmpDir);
      } catch (error) {
        errors.push(error.message);
        return;
      }

      const requirements = Array.isArray(parsed.data.requirements) ? parsed.data.requirements : [];
      requirements.forEach((req, idx) => {
        if (!req || typeof req !== 'object') {
          errors.push(`Requirement entry ${idx + 1} in ${relativeToRoot(source.path)} is not an object.`);
          return;
        }
        const requirementId = req.id;
        if (!requirementId || typeof requirementId !== 'string') {
          errors.push(`Requirement entry ${idx + 1} in ${relativeToRoot(source.path)} is missing an id.`);
          return;
        }
        if (requirementIndex.has(requirementId)) {
          const previous = requirementIndex.get(requirementId);
          errors.push(`Duplicate requirement id '${requirementId}' in ${relativeToRoot(source.path)} (first defined in ${relativeToRoot(previous.file)}).`);
          return;
        }
        requirementIndex.set(requirementId, { file: source.path });
        const validations = gatherValidations(req);
        const hasChildren = Array.isArray(req.children) && req.children.length > 0;
        const criticality = (req.criticality || '').toUpperCase();

        if (CRITICAL_LEVELS.has(criticality) && !hasChildren && validations.length === 0) {
          errors.push(`Critical requirement '${requirementId}' in ${relativeToRoot(source.path)} is missing validation entries or child requirements.`);
        }

        validations.forEach((validation, validationIndex) => {
          if (!validation || typeof validation !== 'object') {
            errors.push(`Requirement '${requirementId}' in ${relativeToRoot(source.path)} has an invalid validation entry (#${validationIndex + 1}).`);
            return;
          }

          const outputLabel = `Requirement '${requirementId}' validation ${validationIndex + 1} (${relativeToRoot(source.path)})`;
          const vType = (validation.type || '').toLowerCase();
          const validationStatus = (validation.status || '').toLowerCase();
          const ref = validation.ref || '';
          const workflowId = validation.workflow_id || '';

          if ((vType === 'test' || vType === 'automation') && !validation.phase) {
            errors.push(`${outputLabel} must specify a phase when type is '${vType}'.`);
          }

          if (validation.phase) {
            const normalizedPhase = validation.phase.toLowerCase();
            if (!VALID_PHASES.has(normalizedPhase)) {
              warnings.push(`${outputLabel} references unknown phase '${validation.phase}'. Valid phases: ${Array.from(VALID_PHASES).join(', ')}.`);
            }
          }

          if (!ref && !workflowId) {
            errors.push(`${outputLabel} must define either 'ref' (file path) or 'workflow_id'.`);
          }

          const shouldCheckRef = Boolean(ref && !workflowId);
          if (shouldCheckRef) {
            const lowerStatus = validationStatus.toLowerCase();
            const skipFileCheck = OPTIONAL_VALIDATION_STATUSES.has(lowerStatus);
            const hasScheme = /^[a-zA-Z]+:/.test(ref) || ref.startsWith('//');

            if (!skipFileCheck && !hasScheme) {
              const resolved = path.resolve(scenarioRoot, ref);
              if (!fs.existsSync(resolved)) {
                errors.push(`${outputLabel} references '${ref}' but the file does not exist relative to the scenario root.`);
              }
            }
          }
        });

        if (Array.isArray(req.children)) {
          req.children.forEach((childId) => {
            if (typeof childId === 'string' && childId.trim().length > 0) {
              childRefs.push({ parent: requirementId, child: childId, file: source.path });
            } else {
              errors.push(`Requirement '${requirementId}' in ${relativeToRoot(source.path)} contains an invalid child entry.`);
            }
          });
        }
      });

      if (source.isIndex && Array.isArray(parsed.data.imports) && parsed.data.imports.length > 0) {
        importChecks.push({ file: source.path, imports: parsed.data.imports });
      }
    });

    if (importChecks.length > 0) {
      const requirementsDir = path.join(scenarioRoot, 'requirements');
      if (!fs.existsSync(requirementsDir)) {
        errors.push(`Index file declares imports but ${relativeToRoot(requirementsDir)} is missing.`);
      } else {
        importChecks.forEach((entry) => {
          entry.imports.forEach((importPath) => {
            if (typeof importPath !== 'string' || importPath.trim().length === 0) {
              errors.push(`Empty import entry detected in ${relativeToRoot(entry.file)}.`);
              return;
            }
            const resolved = path.join(requirementsDir, importPath);
            if (!fs.existsSync(resolved)) {
              errors.push(`Import '${importPath}' referenced in ${relativeToRoot(entry.file)} does not exist under requirements/.`);
            }
          });
        });
      }
    }

    childRefs.forEach((ref) => {
      if (!requirementIndex.has(ref.child)) {
        errors.push(`Requirement '${ref.parent}' references unknown child '${ref.child}' (${relativeToRoot(ref.file)}).`);
      }
    });
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

  if (errors.length > 0) {
    console.error('❌ Requirement validation failed:');
    const uniqueErrors = [...new Set(errors)];
    uniqueErrors.forEach((message) => {
      console.error(`  - ${message}`);
    });
    process.exit(1);
  }

  if (warnings.length > 0) {
    const uniqueWarnings = [...new Set(warnings)];
    uniqueWarnings.forEach((message) => {
      console.warn(`⚠️  ${message}`);
    });
  }

  if (!options.quiet) {
    console.log(`✅ Requirements for '${options.scenario}' passed schema and reference validation (${requirementSources.length} file${requirementSources.length === 1 ? '' : 's'} checked).`);
  }
}

try {
  main();
} catch (error) {
  console.error(`❌ ${error.message}`);
  process.exit(1);
}
