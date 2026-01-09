import { describe, expect, it } from 'vitest';
import {
  formatFileSize,
  formatDuration,
  formatExecutionDuration,
  formatCapturedLabel,
  formatSeconds,
} from './formatters';

describe('formatFileSize', () => {
  it('returns "Unknown size" for undefined', () => {
    expect(formatFileSize(undefined)).toBe('Unknown size');
  });

  it('returns "Unknown size" for 0', () => {
    expect(formatFileSize(0)).toBe('Unknown size');
  });

  it('formats bytes correctly', () => {
    expect(formatFileSize(512)).toBe('512 B');
  });

  it('formats kilobytes correctly', () => {
    expect(formatFileSize(1024)).toBe('1.0 KB');
    expect(formatFileSize(2560)).toBe('2.5 KB');
  });

  it('formats megabytes correctly', () => {
    expect(formatFileSize(1024 * 1024)).toBe('1.0 MB');
    expect(formatFileSize(1024 * 1024 * 2.5)).toBe('2.5 MB');
  });
});

describe('formatDuration', () => {
  it('returns empty string for undefined', () => {
    expect(formatDuration(undefined)).toBe('');
  });

  it('returns empty string for 0', () => {
    expect(formatDuration(0)).toBe('');
  });

  it('formats seconds correctly', () => {
    expect(formatDuration(45000)).toBe('45s');
  });

  it('formats minutes and seconds correctly', () => {
    expect(formatDuration(90000)).toBe('1:30');
    expect(formatDuration(125000)).toBe('2:05');
  });
});

describe('formatExecutionDuration', () => {
  it('returns "< 1s" for sub-second durations', () => {
    expect(formatExecutionDuration(500)).toBe('< 1s');
  });

  it('formats seconds correctly', () => {
    expect(formatExecutionDuration(45000)).toBe('45s');
  });

  it('formats minutes with seconds correctly', () => {
    expect(formatExecutionDuration(90000)).toBe('1m 30s');
  });

  it('formats minutes without seconds when even', () => {
    expect(formatExecutionDuration(120000)).toBe('2m');
  });
});

describe('formatCapturedLabel', () => {
  it('handles singular correctly', () => {
    expect(formatCapturedLabel(1, 'frame')).toBe('1 frame');
  });

  it('handles plural correctly', () => {
    expect(formatCapturedLabel(5, 'frame')).toBe('5 frames');
  });

  it('handles zero correctly', () => {
    expect(formatCapturedLabel(0, 'asset')).toBe('0 assets');
  });

  it('rounds decimals', () => {
    expect(formatCapturedLabel(2.7, 'frame')).toBe('3 frames');
  });
});

describe('formatSeconds', () => {
  it('formats sub-minute durations', () => {
    expect(formatSeconds(30)).toBe('30s');
  });

  it('formats minutes with seconds', () => {
    expect(formatSeconds(90)).toBe('1m 30s');
  });

  it('formats minutes without seconds when even', () => {
    expect(formatSeconds(120)).toBe('2m');
  });
});
