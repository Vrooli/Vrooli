import { test } from 'node:test';
import assert from 'node:assert/strict';
import { nativeObservations, nativeAssertionOptions } from '../dist/native-observations.js';
import RequirementReporter from '../dist/reporter.js';
import { readFileSync } from 'node:fs';

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
