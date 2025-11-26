import { strict as assert } from 'node:assert';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const registryScript = path.resolve(__dirname, '../build-registry.mjs');

const writeJSON = (baseDir, relativePath, payload) => {
  const target = path.join(baseDir, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, `${JSON.stringify(payload, null, 2)}\n`);
};

const readJSON = (target) => JSON.parse(fs.readFileSync(target, 'utf8'));

const runRegistry = (scenarioDir) => {
  const result = spawnSync('node', [registryScript, '--scenario', scenarioDir], {
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, `build-registry.mjs failed: ${result.stderr || result.stdout}`);
};

test('build-registry emits depth-first order, reset, and requirements', () => {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'bas-registry-'));
  const scenarioDir = path.join(tmpRoot, 'scenario');

  // Minimal requirements registry referencing both workflows
  writeJSON(scenarioDir, 'requirements/index.json', {
    imports: ['module.json'],
  });
  writeJSON(scenarioDir, 'requirements/module.json', {
    requirements: [
      {
        id: 'BAS-TEST-001',
        validation: [
          {
            type: 'automation',
            ref: 'test/playbooks/capabilities/01-foundation/alpha.json',
          },
        ],
      },
      {
        id: 'BAS-TEST-002',
        validation: [
          {
            type: 'automation',
            ref: 'test/playbooks/capabilities/02-builder/beta.json',
          },
        ],
      },
    ],
  });

  // Workflow JSONs
  writeJSON(scenarioDir, 'test/playbooks/capabilities/01-foundation/alpha.json', {
    metadata: {
      description: 'Alpha flow',
      reset: 'full',
    },
    nodes: [],
    edges: [],
  });

  writeJSON(scenarioDir, 'test/playbooks/capabilities/02-builder/beta.json', {
    metadata: {
      description: 'Beta flow',
      reset: 'none',
    },
    nodes: [],
    edges: [],
  });

  runRegistry(scenarioDir);

  const registry = readJSON(path.join(scenarioDir, 'test/playbooks/registry.json'));
  assert.equal(registry.playbooks.length, 2, 'expected two playbooks');

  const alpha = registry.playbooks.find((entry) => entry.file.endsWith('alpha.json'));
  const beta = registry.playbooks.find((entry) => entry.file.endsWith('beta.json'));

  assert.ok(alpha.order.startsWith('01.'), 'alpha should be first in traversal');
  assert.deepEqual(alpha.requirements, ['BAS-TEST-001']);
  assert.equal(alpha.reset, 'full');

  assert.ok(beta.order > alpha.order, 'beta should execute after alpha');
  assert.deepEqual(beta.requirements, ['BAS-TEST-002']);
  assert.equal(beta.reset, 'none');
});
