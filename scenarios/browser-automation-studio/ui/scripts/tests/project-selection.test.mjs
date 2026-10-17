import assert from 'node:assert/strict';
import { readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { inventory } from '../run-vitest.mjs';

const ui = fileURLToPath(new URL('../../', import.meta.url));
test('every existing ordinary UI test file is selected exactly once, including viewer and formerly unassigned domains', () => {
  const result = spawnSync(process.execPath, [path.join(ui, 'node_modules/vitest/vitest.mjs'), 'list', '--filesOnly', '--json'], { cwd: ui, encoding: 'utf8', timeout: 15000 });
  assert.equal(result.status, 0, result.stderr);
  const selected = JSON.parse(result.stdout);
  const owned = new Map();
  for (const { file, projectName } of selected) {
    assert.equal(owned.has(file), false, `duplicate ${file}: ${owned.get(file)} / ${projectName}`);
    owned.set(file, projectName);
  }
  function visit(directory) {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const full = path.join(directory, entry.name);
      if (entry.isDirectory()) visit(full);
      else if (/\.(?:test|spec)\.(?:ts|tsx|mjs)$/.test(entry.name)) assert.ok(owned.has(full), `unassigned ordinary test: ${full}`);
    }
  }
  visit(path.join(ui, 'src'));
  visit(path.join(ui, 'vitest'));
  assert.ok(selected.some(item => item.projectName === 'execution-viewer'));
  assert.deepEqual(inventory.coverage.thresholds, { lines: 85, functions: 85, branches: 85, statements: 85 });
  assert.deepEqual(inventory.coverage.include, ['src/**/*.{ts,tsx}']);
  assert.deepEqual(inventory.coverage.exclude, ['src/**/*.test.{ts,tsx}', 'src/**/*.test.{ts,tsx,mjs}', 'src/**/*.spec.{ts,tsx}', 'src/**/*.spec.{ts,tsx,mjs}', 'src/**/*.d.ts', 'src/main.tsx', 'src/bootstrap.tsx', 'src/test-setup.ts', 'src/test-utils/**', 'src/consts/strings.generated.ts', 'src/i18n/locales/**', 'src/**/generated/**']);
  const declared = inventory.projects.map(project => project.name);
  assert.equal(new Set(declared).size, declared.length);
  assert.ok(declared.includes(inventory.sourceClosureProject));
  for (const name of inventory.smoke) assert.ok(declared.includes(name));
});
