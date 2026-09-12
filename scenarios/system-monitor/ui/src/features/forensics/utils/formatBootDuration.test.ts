import { describe, expect, it } from 'vitest';
import { formatBootDuration } from './formatBootDuration';

describe('formatBootDuration', () => {
  it('returns em-dash for missing or invalid timestamps', () => {
    expect(formatBootDuration('', '2026-05-07T10:00:00Z')).toBe('—');
    expect(formatBootDuration('2026-05-07T10:00:00Z', '')).toBe('—');
    expect(formatBootDuration('not-a-date', '2026-05-07T10:00:00Z')).toBe('—');
  });

  it('returns em-dash for negative durations (clock skew)', () => {
    expect(formatBootDuration('2026-05-07T11:00:00Z', '2026-05-07T10:00:00Z')).toBe('—');
  });

  it('formats sub-minute durations as seconds', () => {
    expect(formatBootDuration('2026-05-07T10:00:00Z', '2026-05-07T10:00:42Z')).toBe('42s');
  });

  it('formats sub-hour durations as minutes and seconds', () => {
    expect(formatBootDuration('2026-05-07T10:00:00Z', '2026-05-07T10:05:30Z')).toBe('5m 30s');
  });

  it('formats sub-day durations as hours and minutes', () => {
    expect(formatBootDuration('2026-05-07T10:00:00Z', '2026-05-07T13:30:00Z')).toBe('3h 30m');
  });

  it('formats multi-day durations as days and hours', () => {
    expect(formatBootDuration('2026-05-01T10:00:00Z', '2026-05-07T13:00:00Z')).toBe('6d 3h');
  });
});
