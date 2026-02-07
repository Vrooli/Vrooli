import { describe, expect, it } from 'vitest';
import {
  cycleAppOpenMode,
  resolveAppOpenModeShortcut,
  type AppOpenMode,
} from './tabSwitcherOpenMode';

const buildEvent = (overrides?: Partial<KeyboardEvent>): KeyboardEvent => {
  return {
    key: '',
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    ...overrides,
  } as KeyboardEvent;
};

describe('tabSwitcherOpenMode', () => {
  it('cycles through all open modes', () => {
    const cycle: AppOpenMode[] = ['single-preview', 'replace-focused', 'add-pane'];
    let mode: AppOpenMode = cycle[0];
    mode = cycleAppOpenMode(mode);
    expect(mode).toBe(cycle[1]);
    mode = cycleAppOpenMode(mode);
    expect(mode).toBe(cycle[2]);
    mode = cycleAppOpenMode(mode);
    expect(mode).toBe(cycle[0]);
  });

  it('resolves alt+o as cycle shortcut', () => {
    const result = resolveAppOpenModeShortcut(buildEvent({ key: 'o', altKey: true }));
    expect(result).toBe('cycle');
  });

  it('resolves direct mode shortcuts', () => {
    expect(resolveAppOpenModeShortcut(buildEvent({ key: '1', altKey: true }))).toBe('single-preview');
    expect(resolveAppOpenModeShortcut(buildEvent({ key: '2', altKey: true }))).toBe('replace-focused');
    expect(resolveAppOpenModeShortcut(buildEvent({ key: '3', altKey: true }))).toBe('add-pane');
  });

  it('ignores shortcuts when alt is not pressed', () => {
    expect(resolveAppOpenModeShortcut(buildEvent({ key: 'o' }))).toBeNull();
  });
});

