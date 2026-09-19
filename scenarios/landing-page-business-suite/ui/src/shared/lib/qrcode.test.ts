import { describe, expect, it } from 'vitest';
import { encodeQr } from './qrcode';

function finderAt(modules: boolean[][], x: number, y: number): boolean {
  // Ring of dark modules around a light ring around a dark 3×3 core.
  const row = (dy: number) => modules[y + dy] ?? [];
  return [0, 6].every((d) => row(d).slice(x, x + 7).every(Boolean))
    && row(1).slice(x + 1, x + 6).every((dark) => !dark)
    && row(3).slice(x + 2, x + 5).every(Boolean);
}

describe('encodeQr', () => {
  it('chooses the smallest version that fits and draws all three finder patterns', () => {
    const small = encodeQr('hi');
    expect(small.size).toBe(21);
    expect(finderAt(small.modules, 0, 0)).toBe(true);
    expect(finderAt(small.modules, small.size - 7, 0)).toBe(true);
    expect(finderAt(small.modules, 0, small.size - 7)).toBe(true);
  });

  it('grows to fit an enrollment URI and stays deterministic', () => {
    const uri = 'otpauth://totp/Site:admin%40example.com?algorithm=SHA1&digits=6&issuer=Site&period=30&secret=JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP';
    const first = encodeQr(uri);
    expect(first.size).toBeGreaterThan(21);
    expect(encodeQr(uri)).toEqual(first);
  });

  it('refuses payloads beyond QR capacity', () => {
    expect(() => encodeQr('x'.repeat(3000))).toThrow(RangeError);
  });
});
