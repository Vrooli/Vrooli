#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function normalizePlaybookPath(scenarioRoot, absPath) {
  const rel = path.relative(scenarioRoot, absPath);
  return rel.split(path.sep).join('/');
}

function collectRequirementValidations(requirementsDir) {
  const indexPath = path.join(requirementsDir, 'index.json');
  if (!fs.existsSync(indexPath)) {
    return new Map();
  }
  const indexData = readJSON(indexPath);
  const imports = indexData.imports || [];
  const validationsByFile = new Map();

  for (const relModule of imports) {
    const modulePath = path.join(requirementsDir, relModule);
    if (!fs.existsSync(modulePath)) continue;
    const moduleData = readJSON(modulePath);
    for (const req of moduleData.requirements || []) {
      for (const validation of req.validation || []) {
        if (validation.type !== 'automation' || typeof validation.ref !== 'string') continue;
        if (!validation.ref.startsWith('test/playbooks/')) continue;
        const key = validation.ref;
        if (!validationsByFile.has(key)) {
          validationsByFile.set(key, []);
        }
        validationsByFile.get(key).push({
          requirement: req.id,
          requirementTitle: req.title,
          phase: validation.phase || 'unknown',
          status: validation.status || 'unknown'
        });
      }
    }
  }

  return validationsByFile;
}

function extractFixtureSlug(workflowId) {
  if (typeof workflowId !== 'string' || !workflowId.startsWith('@fixture/')) {
    return null;
  }
  const match = workflowId.trim().match(/^@fixture\/([A-Za-z0-9_.-]+)/);
  return match ? match[1] : null;
}

function getFixtures(playbook) {
  const fixtures = new Set();
  const nodes = Array.isArray(playbook.nodes) ? playbook.nodes : [];
  for (const node of nodes) {
    const workflowId = node?.data?.workflowId;
    const slug = extractFixtureSlug(workflowId);
    if (slug) {
      fixtures.add(slug);
    }
  }
  return Array.from(fixtures).sort();
}

function collectFixtureMetadata(fixturesDir) {
  const metadataMap = new Map();
  if (!fs.existsSync(fixturesDir)) {
    return metadataMap;
  }
  const files = [];
  const stack = [fixturesDir];
  while (stack.length) {
    const current = stack.pop();
    if (!fs.existsSync(current)) continue;
    const stat = fs.statSync(current);
    if (stat.isDirectory()) {
      for (const child of fs.readdirSync(current)) {
        stack.push(path.join(current, child));
      }
    } else if (path.extname(current) === '.json') {
      files.push(current);
    }
  }

  for (const filePath of files) {
    try {
      const data = readJSON(filePath);
      const metadata = data.metadata || {};
      const fixtureId = metadata.fixture_id || metadata.fixtureId;
      if (!fixtureId || typeof fixtureId !== 'string') {
        continue;
      }
      const description = typeof metadata.description === 'string' ? metadata.description : '';
      const requirements = Array.isArray(metadata.requirements)
        ? metadata.requirements.filter((req) => typeof req === 'string')
        : [];
      const parameters = Array.isArray(metadata.parameters)
        ? metadata.parameters
            .map((param) => {
              if (!param || typeof param !== 'object') {
                return null;
              }
              const name = typeof param.name === 'string' ? param.name : '';
              if (!name) {
                return null;
              }
              const type = typeof param.type === 'string' ? param.type : 'string';
              const paramInfo = {
                name,
                type,
                required: Boolean(param.required),
                default: Object.prototype.hasOwnProperty.call(param, 'default') ? param.default : undefined,
                enumValues: Array.isArray(param.enumValues)
                  ? param.enumValues
                  : Array.isArray(param.enum_values)
                  ? param.enum_values
                  : undefined,
                description: typeof param.description === 'string' ? param.description : '',
              };
              return paramInfo;
            })
            .filter(Boolean)
        : [];

      metadataMap.set(fixtureId, {
        description,
        requirements,
        parameters,
      });
    } catch (err) {
      // eslint-disable-next-line no-console
      console.warn(`[playbook-registry] Failed parsing fixture metadata ${filePath}: ${err.message}`);
    }
  }

  return metadataMap;
}

function collectPlaybooks(playbooksRoot) {
  const targets = [
    path.join(playbooksRoot, 'capabilities'),
    path.join(playbooksRoot, 'journeys')
  ];
  const files = [];
  for (const target of targets) {
    if (!fs.existsSync(target)) continue;
    const stack = [target];
    while (stack.length) {
      const current = stack.pop();
      if (!fs.existsSync(current)) continue;
      const stat = fs.statSync(current);
      if (stat.isDirectory()) {
        for (const child of fs.readdirSync(current)) {
          stack.push(path.join(current, child));
        }
      } else if (path.extname(current) === '.json') {
        files.push(current);
      }
    }
  }
  return files;
}

function parseArgs() {
  const args = process.argv.slice(2);
  const config = { scenario: null };
  for (let i = 0; i < args.length; i += 1) {
    const arg = args[i];
    if (arg === '--scenario' && i + 1 < args.length) {
      config.scenario = path.resolve(args[i + 1]);
      i += 1;
    }
  }
  if (!config.scenario) {
    config.scenario = process.cwd();
  }
  return config;
}

function ensureDirExists(dir, label) {
  if (!fs.existsSync(dir)) {
    console.warn(`[playbook-registry] Skipping: ${label} not found at ${dir}`);
    process.exit(0);
  }
}

(function main() {
  const { scenario } = parseArgs();
  const scenarioRoot = scenario;
  const playbooksRoot = path.join(scenarioRoot, 'test', 'playbooks');
  const requirementsDir = path.join(scenarioRoot, 'requirements');
  const registryPath = path.join(playbooksRoot, 'registry.json');
  const fixturesDir = path.join(playbooksRoot, '__subflows');

  ensureDirExists(playbooksRoot, 'test/playbooks directory');
  ensureDirExists(requirementsDir, 'requirements directory');

  const validationsByFile = collectRequirementValidations(requirementsDir);
  const playbookFiles = collectPlaybooks(playbooksRoot);
  const fixtureMetadata = collectFixtureMetadata(fixturesDir);

  const registry = {
    _note: 'AUTO-GENERATED by scripts/scenarios/testing/playbooks/build-registry.mjs — run this script (or make test) to refresh. Do not edit manually.',
    scenario: path.basename(scenarioRoot),
    generated_at: new Date().toISOString(),
    playbooks: []
  };

  for (const absPath of playbookFiles.sort()) {
    const relPath = normalizePlaybookPath(scenarioRoot, absPath);
    const data = readJSON(absPath);
    const fixtures = getFixtures(data);
    const metadata = data.metadata || {};
    const validations = validationsByFile.get(relPath) || [];
    const fixtureDetails = fixtures.map((id) => ({
      id,
      description: fixtureMetadata.get(id)?.description || '',
      requirements: fixtureMetadata.get(id)?.requirements || [],
      parameters: fixtureMetadata.get(id)?.parameters || [],
    }));

    const fixtureRequirementSet = new Set();
    const inheritedRequirements = Array.isArray(metadata.requirementsFromFixtures)
      ? metadata.requirementsFromFixtures.filter((req) => typeof req === 'string')
      : [];
    inheritedRequirements.forEach((req) => fixtureRequirementSet.add(req));
    for (const detail of fixtureDetails) {
      for (const req of detail.requirements || []) {
        if (typeof req === 'string') {
          fixtureRequirementSet.add(req);
        }
      }
    }
    const fixtureRequirements = Array.from(fixtureRequirementSet);

    registry.playbooks.push({
      file: relPath,
      requirement: metadata.requirement || null,
      description: metadata.description || '',
      version: metadata.version ?? null,
      fixtures,
      fixture_details: fixtureDetails,
      fixture_requirements: fixtureRequirements,
      validations,
    });
  }

  fs.writeFileSync(registryPath, JSON.stringify(registry, null, 2));
  console.log(`[playbook-registry] Updated ${path.relative(scenarioRoot, registryPath)} (${registry.playbooks.length} workflows)`);
})();
