import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { mergeCoverageRun } from '../merge-v8-coverage.mjs';
import { fileFingerprint, sourceFingerprint } from '../coverage-inputs.mjs';
import { parseArgs, runSuite } from '../run-vitest.mjs';

const loc = line => ({ start: { line, column: 0 }, end: { line, column: 10 } });
function fixture(root, branches) {
  const file = path.join(root, 'src/feature.ts');
  return { [file]: { path: file, statementMap: { 0: loc(1) }, s: { 0: 1 },
    fnMap: { 0: { name: 'feature', decl: loc(1), loc: loc(1), line: 1 } }, f: { 0: 1 },
    branchMap: Object.fromEntries(branches.map(([id, line]) => [id, { type: 'branch', line, loc: loc(line), locations: [loc(line)] }])),
    b: Object.fromEntries(branches.map(([id, , count]) => [id, [count]])) } };
}
function setup(t, entries = [[[0, 2, 1], [1, 3, 0]], [[0, 3, 1], [1, 2, 0]]]) {
  const root = mkdtempSync(path.join(tmpdir(), 'bas-coverage-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(path.join(root, 'src'));
  writeFileSync(path.join(root, 'src/feature.ts'), 'export const feature = 1;\n');
  const run = path.join(root, 'coverage');
  const projects = ['first', 'second'];
  const manifest = { version: 1, projects, sourceClosureProject: 'first', sourceFingerprint: sourceFingerprint(root), completed: {} };
  for (let i = 0; i < projects.length; i++) {
    const name = projects[i]; const file = path.join(run, 'raw', name, 'coverage-final.json');
    mkdirSync(path.dirname(file), { recursive: true });
    writeFileSync(file, JSON.stringify(fixture(root, entries[i])));
    manifest.completed[name] = { sourceFingerprint: manifest.sourceFingerprint, reportFingerprint: fileFingerprint(file), sourceClosure: i === 0 };
  }
  writeFileSync(path.join(run, 'manifest.json'), JSON.stringify(manifest));
  return { root, run, manifest };
}

test('merges differently numbered branches by source location without losing positive coverage', async t => {
  const { root, run } = setup(t);
  const result = await mergeCoverageRun(run, root);
  assert.equal(result.status, 'passed');
  assert.equal(result.summary.branches.total, 2);
  assert.equal(result.summary.branches.covered, 2);
  assert.equal(result.summary.branches.pct, 100);
});
test('positive unchanged branches survive an additional map with a new uncovered location', async t => {
  const { root, run } = setup(t, [[[0, 2, 1]], [[0, 2, 0], [1, 3, 0]]]);
  const result = await mergeCoverageRun(run, root);
  assert.equal(result.status, 'below-floor');
  assert.equal(result.floor, 85);
  assert.equal(result.summary.branches.total, 2);
  assert.equal(result.summary.branches.covered, 1);
  assert.ok(existsSync(path.join(run, 'coverage-summary.json')));
});
for (const kind of ['missing', 'unexpected', 'changed-source', 'changed-report', 'missing-closure']) {
  test(`refuses ${kind} input without publishing a combined summary`, async t => {
    const { root, run, manifest } = setup(t);
    if (kind === 'missing') rmSync(path.join(run, 'raw/second'), { recursive: true });
    if (kind === 'unexpected') mkdirSync(path.join(run, 'raw/unexpected'));
    if (kind === 'changed-source') writeFileSync(path.join(root, 'src/feature.ts'), 'changed');
    if (kind === 'changed-report') writeFileSync(path.join(run, 'raw/first/coverage-final.json'), '{}');
    if (kind === 'missing-closure') { manifest.completed.first.sourceClosure = false; writeFileSync(path.join(run, 'manifest.json'), JSON.stringify(manifest)); }
    await assert.rejects(mergeCoverageRun(run, root), /Incomplete coverage/);
    assert.equal(existsSync(path.join(run, 'coverage-summary.json')), false);
  });
}
test('coverage disabling flags remain behavior-only, respecting last explicit value', () => {
  assert.equal(parseArgs(['--coverage=false']).coverage, false);
  assert.deepEqual(parseArgs(['--suite', 'full', '--coverage']).forwarded, []);
  assert.deepEqual(parseArgs(['--coverage', 'false']).forwarded, []);
  assert.equal(parseArgs(['--coverage', 'false']).coverage, false);
  assert.equal(parseArgs(['--coverage.enabled']).coverage, true);
  assert.equal(parseArgs(['--no-coverage.enabled']).coverage, false);
  assert.throws(() => parseArgs(['--coverage.include=src/one.ts']), /owner-selected/);
  assert.throws(() => parseArgs(['--config=custom.ts']), /owner-selected/);
  assert.equal(parseArgs(['--coverage', '--no-coverage']).coverage, false);
  assert.equal(parseArgs(['--no-coverage', '--coverage']).coverage, true);
});
function mockProject(root, { belowFloor = false, fail = false } = {}) {
  return (_command, args, options) => {
    assert.equal(args.includes('--project'), true);
    const out = options.env.BAS_COVERAGE_REPORTS_DIRECTORY;
    const project = args[args.indexOf('--project') + 1];
    const native = { schema_version: 'vitest-native/v1', run_id: options.env.VROOLI_TEST_RUN_ID,
      runner_version: '3.2.4', support_profile: 'unsupported', limitations: ['unsupported fixture'], unhandled_errors: [],
      tests: [{ test_id: 'case-1', file: path.join(root, 'src/feature.test.ts'), project, name: 'does feature', state: fail ? 'fail' : 'pass', errors: [], require_assertions: false, assertion_status: 'unknown', reason: 'unsupported-version' }] };
    if (options.env.VROOLI_TEST_QUALITY_OUTPUT) writeFileSync(options.env.VROOLI_TEST_QUALITY_OUTPUT, JSON.stringify(native));
    writeFileSync(options.env.BAS_REQUIREMENTS_OUTPUT_FILE, JSON.stringify({ native_observations: [native], total_tests: 1, passed_tests: fail ? 0 : 1, failed_tests: fail ? 1 : 0, requirements: [{ id: 'BAS-TEST', status: fail ? 'failed' : 'passed' }] }));
    if (options.env.BAS_COLLECT_RAW_COVERAGE === '1') writeFileSync(path.join(out, 'coverage-final.json'), JSON.stringify(fixture(root, [[0, 2, belowFloor ? 0 : 1]])));
    return { status: fail ? 1 : 0 };
  };
}
for (const configuration of ['tsconfig.json', 'tsconfig.node.json']) {
  test(`mid-run ${configuration} changes invalidate coverage without replacing retained evidence`, async t => {
    const { root } = setup(t);
    writeFileSync(path.join(root, configuration), JSON.stringify({ compilerOptions: { strict: true } }));
    writeFileSync(path.join(root, 'coverage/coverage-summary.json'), 'retained');
    const executeProject = mockProject(root);
    let projects = 0;
    const code = await runSuite({ root, argv: ['--suite', 'full', '--coverage'], env: {}, execute: (...args) => {
      const result = executeProject(...args);
      if (++projects === 1) writeFileSync(path.join(root, configuration), JSON.stringify({ compilerOptions: { strict: false } }));
      return result;
    } });
    assert.equal(code, 2);
    assert.equal(projects, 1);
    const manifest = JSON.parse(readFileSync(path.join(root, 'coverage/vitest-run.json')));
    assert.equal(manifest.status, 'incomplete');
    assert.match(manifest.error, /Source inputs changed/);
    assert.equal(readFileSync(path.join(root, 'coverage/coverage-summary.json'), 'utf8'), 'retained');
    assert.equal(JSON.parse(readFileSync(path.join(root, 'coverage/vitest-requirements.json'))).passed_tests, 1);
  });
}
test('below-floor complete run retains merged behavior evidence and distinguishes coverage failure', async t => {
  const { root } = setup(t);
  const result = await runSuite({ root, argv: ['--suite', 'full', '--coverage'], env: {}, execute: mockProject(root, { belowFloor: true }) });
  assert.equal(result, 1);
  const manifest = JSON.parse(readFileSync(path.join(root, 'coverage/vitest-run.json')));
  assert.equal(manifest.status, 'below-floor');
  const requirements = JSON.parse(readFileSync(path.join(root, 'coverage/vitest-requirements.json')));
  assert.equal(requirements.total_tests, 15);
  assert.equal(requirements.failed_tests, 0);
  assert.equal(requirements.requirements[0].status, 'passed');
});
test('behavior-only run preserves previously retained coverage and records its own failure evidence', async t => {
  const { root } = setup(t);
  writeFileSync(path.join(root, 'coverage/coverage-summary.json'), 'retained');
  const result = await runSuite({ root, argv: ['--no-coverage'], env: {}, execute: mockProject(root, { fail: true }) });
  assert.equal(result, 1);
  assert.equal(readFileSync(path.join(root, 'coverage/coverage-summary.json'), 'utf8'), 'retained');
  assert.equal(JSON.parse(readFileSync(path.join(root, 'coverage/vitest-run.json'))).status, 'tests-failed');
  assert.equal(JSON.parse(readFileSync(path.join(root, 'coverage/vitest-requirements.json'))).failed_tests, 1);
});


test('missing successful-project behavior report is incomplete, without replacing retained coverage', async t => {
  const { root } = setup(t);
  writeFileSync(path.join(root, 'coverage/coverage-summary.json'), 'retained');
  const code = await runSuite({ root, argv: ['--coverage=false'], env: {}, execute: () => ({ status: 0 }) });
  assert.equal(code, 2);
  assert.equal(JSON.parse(readFileSync(path.join(root, 'coverage/vitest-run.json'))).status, 'incomplete');
  assert.equal(readFileSync(path.join(root, 'coverage/coverage-summary.json'), 'utf8'), 'retained');
});


test('full command-native handoff contains every project and preserves unsupported observations', async t => {
  const { root } = setup(t);
  const quality = path.join(root, 'quality/native.json');
  const inheritedRunId = '6a0a58ef-3f85-48a6-8c4e-3c552b5164c1';
  const code = await runSuite({ root, argv: ['--suite', 'full', '--coverage=false'],
    env: { VROOLI_TEST_RUN_ID: inheritedRunId, VROOLI_TEST_QUALITY_OUTPUT: quality }, execute: mockProject(root) });
  assert.equal(code, 0);
  const observation = JSON.parse(readFileSync(quality));
  assert.equal(observation.run_id, inheritedRunId);
  assert.equal(observation.runner_version, '3.2.4');
  assert.equal(observation.support_profile, 'unsupported');
  assert.equal(observation.tests.length, 15);
  assert.equal(new Set(observation.tests.map(value => value.project)).size, 15);
  assert.ok(observation.tests.every(value => value.assertion_status === 'unknown' && value.reason === 'unsupported-version'));
  assert.deepEqual(observation.unhandled_errors, []);
  const requirements = JSON.parse(readFileSync(path.join(root, 'coverage/vitest-requirements.json')));
  assert.equal(requirements.native_observations.length, 15);
});
test('partial failed command retains observed cases and explicitly identifies missing project reports', async t => {
  const { root } = setup(t);
  const quality = path.join(root, 'quality/native.json');
  const code = await runSuite({ root, argv: ['--suite', 'full'], env: { VROOLI_TEST_QUALITY_OUTPUT: quality }, execute: mockProject(root, { fail: true }) });
  assert.equal(code, 1);
  const observation = JSON.parse(readFileSync(quality));
  assert.match(observation.run_id, /^[0-9a-f-]{36}$/);
  assert.equal(observation.tests.length, 1);
  assert.equal(observation.tests[0].state, 'fail');
  assert.ok(observation.unhandled_errors.some(error => error.includes('missing native reports') && error.includes('shared')));
});
test('zero native reports produces an explicit unknown handoff rather than a stale or missing report', async t => {
  const { root } = setup(t);
  const quality = path.join(root, 'quality/native.json');
  const code = await runSuite({ root, argv: [], env: { VROOLI_TEST_QUALITY_OUTPUT: quality }, execute: () => ({ status: 1 }) });
  assert.equal(code, 1);
  const observation = JSON.parse(readFileSync(quality));
  assert.equal(observation.runner_version, 'unknown');
  assert.equal(observation.support_profile, 'unsupported');
  assert.equal(observation.tests.length, 0);
  assert.ok(observation.unhandled_errors.some(error => error.includes('No project-native')));
});
