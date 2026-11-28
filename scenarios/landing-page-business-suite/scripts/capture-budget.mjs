const DEFAULT_TIMEOUT_MS = 30_000;
const DEFAULT_SETTLE_MS = 900;
const DEFAULT_RECORD_MS = 1_200;
const DEFAULT_MAX_CAPTURE_MS = 180_000;

export const CAPTURE_TIMING_LIMITS = Object.freeze({
  timeoutMs: 60_000,
  settleMs: 10_000,
  recordMs: 30_000,
  maxCaptureMs: 300_000,
  journeyLength: 100,
});

export const CAPTURE_TIMING_DEFAULTS = Object.freeze({
  timeoutMs: DEFAULT_TIMEOUT_MS,
  settleMs: DEFAULT_SETTLE_MS,
  recordMs: DEFAULT_RECORD_MS,
  maxCaptureMs: DEFAULT_MAX_CAPTURE_MS,
});

const timingFields = Object.freeze(['timeoutMs', 'settleMs', 'recordMs', 'maxCaptureMs']);

function invalidTiming(field, value, detail) {
  return new RangeError(`capture timing ${field} is invalid (${String(value)}): ${detail}`);
}

function finitePositiveTiming(field, value, maximum) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    throw invalidTiming(field, value, 'must be a finite positive number');
  }
  if (value > maximum) throw invalidTiming(field, value, `must be <= ${maximum}ms`);
  return value;
}

function optionalPositiveInteger(field, value, maximum) {
  if (value === undefined) return undefined;
  if (!Number.isInteger(value) || value <= 0) {
    throw invalidTiming(field, value, 'must be a positive integer');
  }
  if (value > maximum) throw invalidTiming(field, value, `must be <= ${maximum}`);
  return value;
}

/**
 * Normalize and validate timing values before a browser or media process is
 * started. All values are milliseconds except the optional journey count.
 * `journeyLength` enables an additional conservative budget check; callers
 * that do not know their checkpoint count may omit it.
 */
export function validateCaptureTiming(options = {}) {
  if (options === null || typeof options !== 'object' || Array.isArray(options)) {
    throw new TypeError('capture timing options must be an object');
  }

  const normalized = {};
  for (const field of timingFields) {
    const value = options[field] === undefined ? CAPTURE_TIMING_DEFAULTS[field] : options[field];
    normalized[field] = finitePositiveTiming(field, value, CAPTURE_TIMING_LIMITS[field]);
  }

  const journeyLength = optionalPositiveInteger('journeyLength', options.journeyLength, CAPTURE_TIMING_LIMITS.journeyLength);
  const journeyLengthCap = optionalPositiveInteger('journeyLengthCap', options.journeyLengthCap, CAPTURE_TIMING_LIMITS.journeyLength);
  if (journeyLength !== undefined && journeyLengthCap !== undefined && journeyLength > journeyLengthCap) {
    throw invalidTiming('journeyLength', journeyLength, `must be <= journeyLengthCap (${journeyLengthCap})`);
  }

  // A journey can spend one settle interval per checkpoint, in addition to
  // its final recording interval. The navigation timeout is retained as one
  // conservative outer-stage allowance rather than multiplied by checkpoints.
  const estimatedJourneyMs = journeyLength === undefined
    ? undefined
    : normalized.timeoutMs + (normalized.settleMs * journeyLength) + normalized.recordMs;
  if (estimatedJourneyMs !== undefined && estimatedJourneyMs > normalized.maxCaptureMs) {
    throw invalidTiming('maxCaptureMs', normalized.maxCaptureMs,
      `must cover the estimated journey duration (${estimatedJourneyMs}ms)`);
  }

  return Object.freeze({
    ...normalized,
    ...(journeyLength === undefined ? {} : { journeyLength }),
    ...(journeyLengthCap === undefined ? {} : { journeyLengthCap }),
    ...(estimatedJourneyMs === undefined ? {} : { estimatedJourneyMs }),
  });
}

export class CaptureBudgetExpiredError extends Error {
  constructor(message = 'capture budget expired') {
    super(message);
    this.name = 'CaptureBudgetExpiredError';
    this.code = 'CAPTURE_BUDGET_EXPIRED';
  }
}

export class CaptureBudgetDisposedError extends Error {
  constructor(message = 'capture budget is disposed') {
    super(message);
    this.name = 'CaptureBudgetDisposedError';
    this.code = 'CAPTURE_BUDGET_DISPOSED';
  }
}

/**
 * Create a one-shot, bounded cancellation budget for a capture operation.
 * Expiry aborts the returned signal and invokes onExpired once. The callback
 * is notification-only: context/process cleanup remains the caller's job.
 */
export function createCaptureBudget(maxCaptureMs = DEFAULT_MAX_CAPTURE_MS, onExpired) {
  const { maxCaptureMs: boundedMaxCaptureMs } = validateCaptureTiming({ maxCaptureMs });
  if (onExpired !== undefined && typeof onExpired !== 'function') {
    throw new TypeError('capture budget onExpired must be a function when provided');
  }

  const controller = new AbortController();
  let disposed = false;
  let expired = false;
  let timer;

  const expire = () => {
    if (disposed || expired) return;
    expired = true;
    const error = new CaptureBudgetExpiredError();
    controller.abort(error);
    try {
      Promise.resolve(onExpired?.(error)).catch(() => {});
    } catch {
      // Cleanup notification must not prevent the abort signal from reaching
      // the parent capture stages or turn expiry into an uncaught exception.
    }
  };

  timer = setTimeout(expire, boundedMaxCaptureMs);
  timer.unref?.();

  const assertActive = () => {
    if (disposed) throw new CaptureBudgetDisposedError();
    if (expired || controller.signal.aborted) throw new CaptureBudgetExpiredError();
    return true;
  };

  const dispose = () => {
    if (disposed) return;
    disposed = true;
    clearTimeout(timer);
  };

  return Object.freeze({ signal: controller.signal, assertActive, dispose });
}
