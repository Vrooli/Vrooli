import { describe, expect, it } from 'vitest';
import { intervalLabel, normalizeInterval } from './pricingIntervals';

describe('pricing intervals', () => {
  it('normalizes legacy numeric, textual, and unknown interval values', () => {
    expect(normalizeInterval(1)).toBe('month');
    expect(normalizeInterval(2)).toBe('year');
    expect(normalizeInterval(3)).toBe('one_time');
    expect(normalizeInterval('MONTHLY')).toBe('month');
    expect(normalizeInterval('annual year')).toBe('year');
    expect(normalizeInterval('one-time purchase')).toBe('one_time');
    expect(normalizeInterval('onetime')).toBe('one_time');
    expect(normalizeInterval(null)).toBe('other');
    expect(normalizeInterval(99)).toBe('other');
  });

  it('returns stable customer-facing labels for every normalized interval', () => {
    expect(intervalLabel('month')).toBe('Monthly');
    expect(intervalLabel('year')).toBe('Yearly');
    expect(intervalLabel('one_time')).toBe('One-time');
    expect(intervalLabel('other')).toBe('Other');
  });
});
