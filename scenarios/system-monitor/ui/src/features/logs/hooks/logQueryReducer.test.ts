import { describe, expect, it } from 'vitest';
import { initialState, logQueryReducer } from './logQueryReducer';
import { DEFAULT_LIMIT, MAX_LIMIT } from '../types';

describe('logQueryReducer', () => {
  it('set-filter merges patch into filters and resets pagination', () => {
    const seeded = {
      ...initialState,
      cursorStack: ['c1', 'c2'],
      currentCursor: 'c2',
      nextCursor: 'c3',
    };
    const next = logQueryReducer(seeded, {
      type: 'set-filter',
      patch: { units: ['nginx.service'], grep: 'oom' },
    });
    expect(next.filters.units).toEqual(['nginx.service']);
    expect(next.filters.grep).toBe('oom');
    expect(next.cursorStack).toEqual([]);
    expect(next.currentCursor).toBeUndefined();
    expect(next.nextCursor).toBeUndefined();
  });

  it('clamps limit on set-filter', () => {
    expect(
      logQueryReducer(initialState, { type: 'set-filter', patch: { limit: 99999 } }).filters.limit,
    ).toBe(MAX_LIMIT);
    expect(
      logQueryReducer(initialState, { type: 'set-filter', patch: { limit: 0 } }).filters.limit,
    ).toBe(DEFAULT_LIMIT);
    expect(
      logQueryReducer(initialState, { type: 'set-filter', patch: { limit: -5 } }).filters.limit,
    ).toBe(DEFAULT_LIMIT);
    expect(
      logQueryReducer(initialState, { type: 'set-filter', patch: { limit: 50 } }).filters.limit,
    ).toBe(50);
  });

  it('reset-filters wipes filters and pagination', () => {
    const seeded = {
      ...initialState,
      filters: { ...initialState.filters, grep: 'panic' },
      cursorStack: ['x'],
      currentCursor: 'x',
    };
    const next = logQueryReducer(seeded, { type: 'reset-filters' });
    expect(next.filters.grep).toBe('');
    expect(next.cursorStack).toEqual([]);
    expect(next.currentCursor).toBeUndefined();
  });

  it('next-page pushes nextCursor onto stack and sets currentCursor', () => {
    const seeded = { ...initialState, nextCursor: 'cur-1' };
    const after1 = logQueryReducer(seeded, { type: 'next-page' });
    expect(after1.cursorStack).toEqual(['cur-1']);
    expect(after1.currentCursor).toBe('cur-1');

    const after2 = logQueryReducer({ ...after1, nextCursor: 'cur-2' }, { type: 'next-page' });
    expect(after2.cursorStack).toEqual(['cur-1', 'cur-2']);
    expect(after2.currentCursor).toBe('cur-2');
  });

  it('next-page is a no-op when there is no nextCursor', () => {
    const next = logQueryReducer(initialState, { type: 'next-page' });
    expect(next).toBe(initialState);
  });

  it('prev-page pops the stack', () => {
    const seeded = {
      ...initialState,
      cursorStack: ['c1', 'c2'],
      currentCursor: 'c2',
    };
    const next = logQueryReducer(seeded, { type: 'prev-page' });
    expect(next.cursorStack).toEqual(['c1']);
    expect(next.currentCursor).toBe('c1');
  });

  it('prev-page back to head clears currentCursor', () => {
    const seeded = { ...initialState, cursorStack: ['c1'], currentCursor: 'c1' };
    const next = logQueryReducer(seeded, { type: 'prev-page' });
    expect(next.cursorStack).toEqual([]);
    expect(next.currentCursor).toBeUndefined();
  });

  it('prev-page is a no-op on an empty stack', () => {
    const next = logQueryReducer(initialState, { type: 'prev-page' });
    expect(next).toBe(initialState);
  });

  it('fetch-success populates entries, available, and reason', () => {
    const next = logQueryReducer(
      { ...initialState, isLoading: true },
      {
        type: 'fetch-success',
        entries: [
          {
            timestamp: '2026-05-07T10:00:00Z',
            realtime: 0,
            priority: 6,
            message: 'hello',
          },
        ],
        nextCursor: 'next-c',
        available: true,
      },
    );
    expect(next.isLoading).toBe(false);
    expect(next.entries).toHaveLength(1);
    expect(next.nextCursor).toBe('next-c');
    expect(next.available).toBe(true);
  });

  it('fetch-error captures the message without losing entries', () => {
    const seeded = { ...initialState, entries: [{ timestamp: 't', realtime: 0, priority: 6, message: 'm' }] };
    const next = logQueryReducer(seeded, { type: 'fetch-error', error: 'boom' });
    expect(next.error).toBe('boom');
    expect(next.entries).toHaveLength(1);
    expect(next.isLoading).toBe(false);
  });
});
