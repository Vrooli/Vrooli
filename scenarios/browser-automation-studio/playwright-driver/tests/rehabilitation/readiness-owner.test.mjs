import test from 'node:test';
import assert from 'node:assert/strict';
import { assessWarmCohort, nearestRankPercentile, sessionsAreIdle } from './readiness-owner.mjs';

test('nearest-rank percentile preserves the boundary observation', () => {
  assert.equal(nearestRankPercentile([10, 30, 20, 40], 0.75), 30);
});

test('a full warm cohort passes only when every trial succeeded and p95 meets the contract', () => {
  const attempts = Array.from({ length: 100 }, (_, index) => ({
    status: 'passed',
    latency_ms: index < 95 ? 999 : 1000,
  }));
  assert.deepEqual(assessWarmCohort(attempts), {
    complete: true,
    sample_count: 100,
    successful_count: 100,
    p95_ms: 999,
    within_band: true,
  });
});

test('missing or failed attempts cannot pass by being omitted from the percentile', () => {
  const attempts = Array.from({ length: 99 }, () => ({ status: 'passed', latency_ms: 1 }));
  attempts.push({ status: 'failed', latency_ms: null });
  const reading = assessWarmCohort(attempts);
  assert.equal(reading.complete, false);
  assert.equal(reading.p95_ms, null);
  assert.equal(reading.within_band, false);
});

test('the live owner refuses a session snapshot with active or unreported capacity', () => {
  assert.equal(sessionsAreIdle({ summary: { active: 0, total: 0 } }), true);
  assert.equal(sessionsAreIdle({ summary: { active: 1, total: 1 } }), false);
  assert.equal(sessionsAreIdle({ summary: { total: 0 } }), false);
});
