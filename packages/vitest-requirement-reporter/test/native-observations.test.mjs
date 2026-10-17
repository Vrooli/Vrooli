import { test } from 'node:test';
import assert from 'node:assert/strict';
import { nativeObservations, nativeAssertionOptions } from '../dist/native-observations.js';
import RequirementReporter, { mergeNativeObservationReports } from '../dist/reporter.js';
import { readFileSync, mkdtempSync, mkdirSync, readdirSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, relative } from 'node:path';

test('default reporter tag grammar matches shared Go consumer cases', () => {
  const cases = JSON.parse(readFileSync(new URL('./tag-grammar.json', import.meta.url), 'utf8'));
  const reporter = new RequirementReporter({ verbose: false });
  assert.ok(cases.length >= 7);
  for (const entry of cases) {
    assert.deepEqual(reporter.extractRequirements({ name: entry.text }), entry.ids, entry.name);
  }
});

function observe(tasks, version = '2.1.9', enabled = true, errors = []) {
  return nativeObservations([{ filepath: '/project/example.test.ts', projectName: 'ui', tasks }], version, () => enabled, errors);
}
const task = (state, extra = {}) => ({ type: 'test', id: 'native-id', name: 'same display name', mode: 'run', result: { state, ...extra } });

function shard(project, version = '3.2.4') {
  return nativeObservations([{ filepath: '/project/example.test.ts', projectName: project,
    tasks: [task('pass')] }], version, () => true, [], 'one-command');
}

test('command merge retains distinct project identities and unsupported assertion evidence', () => {
  const a = shard('first'); const b = shard('second');
  b.tests[0].state = 'fail'; b.tests[0].errors = ['matcher failed'];
  b.unhandled_errors = ['second project import failed'];
  b.limitations = [...b.limitations, 'partial cohort'];
  const before = JSON.stringify([a, b]);
  const result = mergeNativeObservationReports([a, b]);
  assert.deepEqual(result.tests.map(t => [t.project, t.state, t.assertion_status, t.reason]),
    [['first', 'pass', 'unknown', 'unsupported-version'], ['second', 'fail', 'unknown', 'unsupported-version']]);
  assert.equal(result.support_profile, 'unsupported');
  assert.deepEqual(result.unhandled_errors, ['second project import failed']);
  assert.equal(result.limitations.filter(x => x === a.limitations[0]).length, 1);
  assert.ok(result.limitations.includes('partial cohort'));
  assert.equal(JSON.stringify([a, b]), before);
  result.tests[0].errors.push('consumer edit');
  assert.deepEqual(a.tests[0].errors, []);
});

test('command merge refuses conflicting provenance and duplicate test identities', () => {
  for (const [field, value] of [['schema_version', 'future/v2'], ['run_id', 'other-command'],
    ['runner_version', '2.1.9'], ['support_profile', 'react-vitest-v2']]) {
    const a = shard('first'); const b = { ...shard('second'), [field]: value };
    assert.throws(() => mergeNativeObservationReports([a, b]), new RegExp(field));
  }
  assert.throws(() => mergeNativeObservationReports([shard('same'), shard('same')]), /duplicate/);
  assert.throws(() => mergeNativeObservationReports([]), /at least one/);
  assert.throws(() => mergeNativeObservationReports([{ ...shard('first'), run_id: '' }]), /run_id/);
  assert.throws(() => mergeNativeObservationReports([{ ...shard('first'), tests: null }]), /tests/);
});

test('command merge rejects malformed JSON message and requirement arrays', () => {
  for (const [field, value] of [['errors', [42]], ['requirement_ids', 'REQ-ONE'],
    ['requirement_ids', [42]], ['requirement_ids', null]]) {
    const report = shard('json');
    report.tests[0][field] = value;
    assert.throws(() => mergeNativeObservationReports([report]), /string array/);
  }
  const report = shard('json');
  report.tests[0].requirement_ids = ['REQ-ONE'];
  const result = mergeNativeObservationReports([report]);
  result.tests[0].requirement_ids.push('REQ-TWO');
  assert.deepEqual(report.tests[0].requirement_ids, ['REQ-ONE']);
});

test('command merge preserves failed and empty project evidence without claiming completeness', () => {
  const empty = nativeObservations([], '3.2.4', () => false, ['missing project: viewer'], 'one-command');
  const result = mergeNativeObservationReports([shard('first'), empty]);
  assert.equal(result.tests.length, 1);
  assert.deepEqual(result.unhandled_errors, ['missing project: viewer']);
  assert.equal(result.support_profile, 'unsupported');
});

test('native links reuse reporter grammar with inheritance and separate parameterized and skipped identities', () => {
  const reporter = new RequirementReporter({ verbose: false });
  const suite = { type: 'suite', name: '[REQ:UH-CORE-001, UH-CORE-010] shared contract' };
  const tasks = ['pass', 'skip'].map((state, index) => ({
    ...task(state), id: `parameter-${index}`, name: '[REQ:UH-CORE-002] case', suite,
  }));
  const report = nativeObservations([{ filepath: '/project/example.test.ts', tasks }],
    '2.1.9', () => true, [], 'current-run', value => reporter.extractRequirements(value));
  assert.deepEqual(report.tests.map(value => value.test_id), ['parameter-0', 'parameter-1']);
  for (const observation of report.tests) {
    assert.deepEqual(observation.requirement_ids, ['UH-CORE-002', 'UH-CORE-001', 'UH-CORE-010']);
  }
  assert.deepEqual(report.tests.map(value => value.state), ['pass', 'skip']);
  assert.equal(report.tests[1].assertion_status, 'unknown');
  assert.equal(report.run_id, 'current-run');
  assert.equal(observe([task('pass')]).tests[0].requirement_ids, undefined);
});

test('supported pass reports only scoped assertion activity with limitations', () => {
  const report = observe([task('pass')]);
  assert.equal(report.tests[0].assertion_status, 'checked_clean');
  assert.equal(report.tests[0].test_id, 'native-id');
  assert.equal(report.tests[0].project, 'ui');
  assert.match(report.limitations.join(' '), /Bare expect/);
  assert.match(report.limitations.join(' '), /Alternate assertion/);
});
test('unprobed versions and disabled native check remain unknown', () => {
  assert.deepEqual(nativeAssertionOptions('2.1.9'), { requireAssertions: true });
  assert.deepEqual(nativeAssertionOptions('3.2.4'), {});
  assert.equal(observe([task('pass')], '3.2.4').tests[0].reason, 'unsupported-version');
  assert.equal(observe([task('pass')], '2.1.9', false).tests[0].assertion_status, 'unknown');
});
test('expected-failure inversion never converts missing assertions into clean evidence', () => {
  const result = observe([{ ...task('pass'), fails: true }]).tests[0];
  assert.equal(result.assertion_status, 'unknown');
  assert.equal(result.reason, 'unsupported-test-kind');
});
test('skipped, todo and unexecuted tests never establish assertion activity', () => {
  for (const state of ['skip', 'todo', 'run', 'only']) {
    assert.equal(observe([task(state)]).tests[0].assertion_status, 'unknown');
  }
});
test('native assertion absence differs from failed setup or failed matcher', () => {
  const absence = task('fail', { errors: [{ message: 'expected any number of assertion, but got none' }] });
  assert.equal(observe([absence]).tests[0].assertion_status, 'violation');
  for (const message of ['fixture setup failure', 'expected 4 to be five']) {
    assert.equal(observe([task('fail', { errors: [{ message }] })]).tests[0].assertion_status, 'unknown');
  }
});
test('retry preserves final pass, retry count and prior errors independently', () => {
  const result = observe([task('pass', { retryCount: 1, errors: [{ message: 'first attempt failed' }] })]).tests[0];
  assert.equal(result.state, 'pass');
  assert.equal(result.retry_count, 1);
  assert.deepEqual(result.errors, ['first attempt failed']);
});
test('absent retry metadata remains absent and configured seed is retained', () => {
  const report = nativeObservations([{ filepath: '/project/example.test.ts', projectName: 'ui', tasks: [task('pass')] }],
    '2.1.9', () => true, [], 'run', undefined, project => project === 'ui' ? 0 : undefined);
  assert.equal(report.tests[0].seed, '0');
  assert.equal(report.tests[0].retry_count, undefined);
  assert.equal(report.tests[0].retry_ordinal, undefined);
  assert.equal(observe([task('pass', { retryCount: 0 })]).tests[0].retry_count, 0);
});
test('nested identical names retain native IDs and global errors', () => {
  const first = task('pass');
  const second = { ...task('pass'), id: 'second-id' };
  const report = observe([{ type: 'suite', tasks: [first, second] }], '2.1.9', true, [new Error('unhandled rejection')]);
  assert.deepEqual(report.tests.map(test => test.test_id), ['native-id', 'second-id']);
  assert.deepEqual(report.unhandled_errors, ['unhandled rejection']);
});


test('failure artifacts retain distinct long, punctuation and cross-file identities', async (t) => {
  const root = mkdtempSync(join(tmpdir(), 'vitest-artifact-identities-'));
  const previous = process.cwd();
  const ui = join(root, 'scenario', 'ui');
  mkdirSync(ui, { recursive: true });
  process.chdir(ui);
  t.after(() => { process.chdir(previous); rmSync(root, { recursive: true, force: true }); });
  const logs = [];
  t.mock.method(console, 'log', (...parts) => logs.push(parts.join(' ')));
  const cases = [
    ['first.test.ts', 'same long name '.repeat(12) + 'true'],
    ['first.test.ts', 'same long name '.repeat(12) + 'false'],
    ['first.test.ts', 'punctuation/a'], ['first.test.ts', 'punctuation?a'],
    ['first.test.ts', 'shared name'], ['second.test.ts', 'shared name'],
    ['first.test.ts', 'repeated row'], ['first.test.ts', 'repeated row'],
    ['first.test.ts', '[REQ:CASE-ONE] same tagged row'], ['first.test.ts', '[REQ:CASE-TWO] same tagged row'],
  ];
  const expected = new Map();
  const files = cases.map(([file, name], index) => {
    const filepath = join(root, file);
    const message = `distinct failure ${index}`;
    expected.set(`case-${index}`, message);
    return { filepath, projectName: 'ui', tasks: [{type: 'test', id: `case-${index}`, name,
      result: {state: 'fail', errors: [{message}], duration: 1}}] };
  });
  const reports = [];
  for (const run of ['one', 'two']) {
    const artifactsDir = run === 'one' ? join(root, run) : relative(process.cwd(), join(root, run));
    const reporter = new RequirementReporter({outputFile: join(root, `${run}.json`),
      artifactsDir, scenario: 'fixture', conciseMode: true, emitStdout: false, autoClear: false});
    // Avoid unrelated host/git discovery; artifact publication is exercised through onFinished.
    t.mock.method(reporter, 'getGitContext', () => ({recentChanges: [], deletedFiles: []}));
    reporter.onInit();
    await reporter.onFinished(files, []);
    const dirs = readdirSync(join(artifactsDir, 'ui')).sort();
    assert.equal(dirs.length, cases.length, 'no failure may overwrite another diagnostic');
    for (const dir of dirs) {
      assert.ok(dir.length <= 100, 'diagnostic leaf remains bounded');
      const base = join(artifactsDir, 'ui', dir);
      const context = JSON.parse(readFileSync(join(base, 'test-context.json'), 'utf8'));
      assert.equal(readFileSync(join(base, 'error.txt'), 'utf8'), expected.get(context.taskId));
    }
    reports.push(dirs);
  }
  assert.deepEqual(reports[0], reports[1], 'the same identities keep stable artifact paths');
  const links = logs.filter(line => line.includes('↳ Read ')).map(line => line.split('↳ Read ')[1]);
  assert.equal(links.length, cases.length * 2);
  assert.ok(links.every(path => existsSync(path)), 'every emitted diagnostic link resolves');
});
