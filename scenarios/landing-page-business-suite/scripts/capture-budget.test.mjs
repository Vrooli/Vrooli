import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CAPTURE_TIMING_DEFAULTS,
  CAPTURE_TIMING_LIMITS,
  CaptureBudgetDisposedError,
  CaptureBudgetExpiredError,
  createCaptureBudget,
  validateCaptureTiming,
} from './capture-budget.mjs';

const wait = (milliseconds) => new Promise(resolve => setTimeout(resolve, milliseconds));

test('normalizes defaults and preserves finite bounded timing values', () => {
  assert.deepEqual(validateCaptureTiming({}), CAPTURE_TIMING_DEFAULTS);
  assert.deepEqual(validateCaptureTiming({ timeoutMs: 1, settleMs: 2, recordMs: 3, maxCaptureMs: 4 }), {
    timeoutMs: 1,
    settleMs: 2,
    recordMs: 3,
    maxCaptureMs: 4,
  });
  const bounded = validateCaptureTiming({ journeyLength: 2, journeyLengthCap: 3, timeoutMs: 10, settleMs: 20, recordMs: 30, maxCaptureMs: 100 });
  assert.equal(bounded.estimatedJourneyMs, 80);
  assert.ok(Object.isFrozen(bounded));
});

test('rejects non-finite, non-positive, and over-limit timing values', () => {
  for (const field of ['timeoutMs', 'settleMs', 'recordMs', 'maxCaptureMs']) {
    for (const value of [Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY, -1, 0, '10']) {
      assert.throws(() => validateCaptureTiming({ [field]: value }), /capture timing/);
    }
    assert.throws(() => validateCaptureTiming({ [field]: CAPTURE_TIMING_LIMITS[field] + 1 }), /must be <=/);
  }
  assert.throws(() => validateCaptureTiming({ journeyLength: 0 }), /journeyLength/);
  assert.throws(() => validateCaptureTiming({ journeyLength: 1.5 }), /journeyLength/);
  assert.throws(() => validateCaptureTiming({ journeyLength: 4, journeyLengthCap: 3 }), /journeyLength/);
  assert.throws(() => validateCaptureTiming({ journeyLength: 10, timeoutMs: 50, settleMs: 50, recordMs: 50, maxCaptureMs: 100 }), /maxCaptureMs/);
});

test('expires once, aborts its signal, and invokes the expiry callback once', async () => {
  let expiryCount = 0;
  let expiryError;
  const budget = createCaptureBudget(15, error => { expiryCount += 1; expiryError = error; });
  assert.equal(budget.signal.aborted, false);
  assert.equal(budget.assertActive(), true);
  await wait(40);
  assert.equal(budget.signal.aborted, true);
  assert.equal(expiryCount, 1);
  assert.ok(expiryError instanceof CaptureBudgetExpiredError);
  assert.ok(budget.signal.reason instanceof CaptureBudgetExpiredError);
  assert.throws(() => budget.assertActive(), CaptureBudgetExpiredError);
  budget.dispose();
  await wait(10);
  assert.equal(expiryCount, 1);
});

test('dispose cancels the timer and prevents expiry notification', async () => {
  let expiryCount = 0;
  const budget = createCaptureBudget(20, () => { expiryCount += 1; });
  budget.dispose();
  await wait(45);
  assert.equal(expiryCount, 0);
  assert.equal(budget.signal.aborted, false);
  assert.throws(() => budget.assertActive(), CaptureBudgetDisposedError);
  budget.dispose();
});

test('pre-expired budget fails active assertions without a second latch', async () => {
  let expiryCount = 0;
  const budget = createCaptureBudget(5, () => { expiryCount += 1; });
  await wait(25);
  assert.throws(() => budget.assertActive(), CaptureBudgetExpiredError);
  assert.throws(() => budget.assertActive(), CaptureBudgetExpiredError);
  assert.equal(expiryCount, 1);
});

test('invalid budget callback is rejected before starting a timer', () => {
  assert.throws(() => createCaptureBudget(10, 'not-a-callback'), /onExpired/);
});

