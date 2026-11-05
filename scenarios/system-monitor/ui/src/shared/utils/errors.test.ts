import { describe, expect, it } from 'vitest';
import { isAbortError } from './errors';

describe('isAbortError', () => {
  it('recognizes DOM and standard error cancellation values', () => {
    expect(isAbortError(new DOMException('cancelled', 'AbortError'))).toBe(true);
    const error = new Error('cancelled');
    error.name = 'AbortError';
    expect(isAbortError(error)).toBe(true);
  });

  it('rejects unrelated and malformed thrown values', () => {
    expect(isAbortError(new Error('failure'))).toBe(false);
    expect(isAbortError({ name: 'TimeoutError' })).toBe(false);
    expect(isAbortError(null)).toBe(false);
    expect(isAbortError('AbortError')).toBe(false);
  });
});
