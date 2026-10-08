import { describe, expect, it } from 'vitest';
import type { BootEntry } from '../types';
import { classifyShutdown, shutdownLabel } from './classifyShutdown';

const mk = (overrides: Partial<BootEntry>): BootEntry => ({
  index: -1,
  bootId: 'b',
  firstEntry: '2026-05-07T10:00:00Z',
  lastEntry: '2026-05-07T11:00:00Z',
  clean: false,
  ...overrides,
});

describe('classifyShutdown', () => {
  it('treats index 0 as in-progress regardless of clean flag', () => {
    expect(classifyShutdown(mk({ index: 0, clean: false }))).toBe('in-progress');
    expect(classifyShutdown(mk({ index: 0, clean: true }))).toBe('in-progress');
  });

  it('returns clean when clean flag is set', () => {
    expect(classifyShutdown(mk({ index: -1, clean: true }))).toBe('clean');
  });

  it('returns unclean when clean=false and reason is set', () => {
    expect(classifyShutdown(mk({ index: -1, clean: false, reason: 'no shutdown marker' }))).toBe('unclean');
  });

  it('returns unknown when clean=false and no reason is set', () => {
    expect(classifyShutdown(mk({ index: -1, clean: false }))).toBe('unknown');
  });
});

describe('shutdownLabel', () => {
  it('produces non-empty labels for all classes', () => {
    expect(shutdownLabel('clean')).toMatch(/clean/i);
    expect(shutdownLabel('unclean')).toMatch(/unclean/i);
    expect(shutdownLabel('in-progress')).toMatch(/current/i);
    expect(shutdownLabel('unknown')).toMatch(/unknown/i);
  });
});
