import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const ui = fileURLToPath(new URL('../../', import.meta.url));
test('designated source-closure config instruments unimported executable source without inventing type-only statements', t => {
  const root = mkdtempSync(path.join(ui, '.coverage-provider-fixture-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(path.join(root, 'src'));
  writeFileSync(path.join(root, 'src/loaded.ts'), 'export const loaded = () => 1;\n');
  writeFileSync(path.join(root, 'src/unimported.ts'), 'export function unimported(value: boolean) { return value ? 1 : 2; }\n');
  writeFileSync(path.join(root, 'src/types.ts'), 'export interface OnlyType { name: string }\n');
  writeFileSync(path.join(root, 'src/loaded.test.ts'), `import { it, expect } from ${JSON.stringify(path.join(ui, 'node_modules/vitest/dist/index.js'))};\nimport { loaded } from './loaded';\nit('executes the selected module', () => expect(loaded()).toBe(1));\n`);
  writeFileSync(path.join(root, 'config.mts'), `import base from ${JSON.stringify(path.join(ui, 'vitest.coverage.config.ts'))};\nexport default async env => { const c = await base(env); return { ...c, root: ${JSON.stringify(root)}, test: { ...c.test, projects: undefined, setupFiles: [], environment: 'node', include: ['src/*.test.ts'], reporters: ['default'], coverage: { ...c.test.coverage, reportsDirectory: ${JSON.stringify(path.join(root, 'coverage'))}, reporter: ['json'] } } }; };\n`);
  const result = spawnSync(process.execPath, [path.join(ui, 'node_modules/vitest/vitest.mjs'), 'run', '--config', path.join(root, 'config.mts'), '--coverage', '--maxWorkers=1', '--minWorkers=1'], {
    cwd: ui, encoding: 'utf8', timeout: 30000,
    env: { ...process.env, BAS_COLLECT_RAW_COVERAGE: '1', BAS_COVERAGE_SOURCE_CLOSURE: '1' },
  });
  assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`);
  const map = JSON.parse(readFileSync(path.join(root, 'coverage/coverage-final.json')));
  const missing = map[path.join(root, 'src/unimported.ts')];
  assert.ok(missing, 'unimported runtime module must remain in the denominator');
  assert.ok(Object.keys(missing.s).length > 0);
  assert.ok(Object.values(missing.s).every(value => value === 0));
  const types = map[path.join(root, 'src/types.ts')];
  assert.equal(types ? Object.keys(types.s).length : 0, 0, 'erased types have no executable statements');
});
