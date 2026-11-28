import { describe, expect, it, vi } from 'vitest';
import { parseGrep } from './parseGrep';

describe('parseGrep', () => {
  it('returns empty pattern with no error for blank/whitespace input', () => {
    expect(parseGrep('')).toEqual({ pattern: '' });
    expect(parseGrep('   ')).toEqual({ pattern: '' });
  });

  it('trims whitespace and accepts a valid regex', () => {
    expect(parseGrep('  oom_killer  ')).toEqual({ pattern: 'oom_killer' });
  });

  it('returns the trimmed pattern plus an error for an invalid regex', () => {
    const r = parseGrep('foo[unclosed');
    expect(r.pattern).toBe('foo[unclosed');
    expect(r.error).toBeDefined();
  });

  it('accepts complex valid patterns', () => {
    const r = parseGrep('Reached target.*Shutdown|systemd-shutdown');
    expect(r.error).toBeUndefined();
    expect(r.pattern).toBe('Reached target.*Shutdown|systemd-shutdown');
  });

  it('uses the generic validation message for non-Error regex failures', () => {
    vi.stubGlobal('RegExp', function BadRegExp() {
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw 'invalid';
    });
    expect(parseGrep('anything')).toEqual({ pattern: 'anything', error: 'invalid regex' });
    vi.unstubAllGlobals();
  });
});
