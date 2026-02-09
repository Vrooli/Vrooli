import { describe, expect, it } from 'vitest';
import {
  previewNavigationActions,
  reducePreviewNavigationState,
  type PreviewNavigationState,
} from './previewNavigationStateMachine';

const baseState = (): PreviewNavigationState => ({
  previewUrl: 'http://localhost:3000/apps/scenario-1/proxy/',
  previewUrlInput: 'http://localhost:3000/apps/scenario-1/proxy/',
  hasCustomPreviewUrl: false,
  history: ['http://localhost:3000/apps/scenario-1/proxy/'],
  historyIndex: 0,
  initialPreviewUrl: 'http://localhost:3000/apps/scenario-1/proxy/',
});

describe('reducePreviewNavigationState', () => {
  it('exposes typed action creators for all reducer transitions', () => {
    expect(previewNavigationActions.reset(true)).toEqual({ type: 'reset', force: true });
    expect(previewNavigationActions.setInput('/apps/scenario-1/proxy/')).toEqual({
      type: 'set-input',
      value: '/apps/scenario-1/proxy/',
    });
    expect(previewNavigationActions.markDefaultCleared()).toEqual({ type: 'mark-default-cleared' });
    expect(previewNavigationActions.applyDefaultUrl('/apps/scenario-1/proxy/')).toEqual({
      type: 'apply-default-url',
      url: '/apps/scenario-1/proxy/',
    });
    expect(previewNavigationActions.applyLocalNavigation('http://localhost:4310/settings')).toEqual({
      type: 'apply-local-navigation',
      url: 'http://localhost:4310/settings',
    });
    expect(previewNavigationActions.travelHistory('back')).toEqual({
      type: 'travel-history',
      direction: 'back',
    });
    expect(previewNavigationActions.syncFromBridge('http://localhost:4310/settings')).toEqual({
      type: 'sync-from-bridge',
      href: 'http://localhost:4310/settings',
    });
  });

  it('does not reset non-forced state while custom navigation is active', () => {
    const input: PreviewNavigationState = {
      ...baseState(),
      hasCustomPreviewUrl: true,
      previewUrl: 'http://localhost:3000/apps/scenario-2/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/scenario-2/proxy/',
    };

    const next = reducePreviewNavigationState(input, {
      type: 'reset',
      force: false,
    });

    expect(next).toEqual(input);
  });

  it('resets state when forced', () => {
    const next = reducePreviewNavigationState(baseState(), {
      type: 'reset',
      force: true,
    });

    expect(next).toEqual({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
      initialPreviewUrl: null,
    });
  });

  it('applies default URL and appends to history when changed', () => {
    const next = reducePreviewNavigationState(baseState(), {
      type: 'apply-default-url',
      url: 'http://localhost:3000/apps/scenario-2/proxy/',
    });

    expect(next.previewUrl).toBe('http://localhost:3000/apps/scenario-2/proxy/');
    expect(next.previewUrlInput).toBe('http://localhost:3000/apps/scenario-2/proxy/');
    expect(next.hasCustomPreviewUrl).toBe(false);
    expect(next.initialPreviewUrl).toBe('http://localhost:3000/apps/scenario-2/proxy/');
    expect(next.history).toEqual([
      'http://localhost:3000/apps/scenario-1/proxy/',
      'http://localhost:3000/apps/scenario-2/proxy/',
    ]);
    expect(next.historyIndex).toBe(1);
  });

  it('applies local navigation as custom URL and truncates forward history', () => {
    const input: PreviewNavigationState = {
      ...baseState(),
      history: [
        'http://localhost:3000/apps/scenario-1/proxy/',
        'http://localhost:3000/apps/scenario-1/proxy/?path=a',
        'http://localhost:3000/apps/scenario-1/proxy/?path=b',
      ],
      historyIndex: 1,
    };

    const next = reducePreviewNavigationState(input, {
      type: 'apply-local-navigation',
      url: 'http://localhost:3000/apps/scenario-1/proxy/?path=c',
    });

    expect(next.hasCustomPreviewUrl).toBe(true);
    expect(next.previewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=c');
    expect(next.history).toEqual([
      'http://localhost:3000/apps/scenario-1/proxy/',
      'http://localhost:3000/apps/scenario-1/proxy/?path=a',
      'http://localhost:3000/apps/scenario-1/proxy/?path=c',
    ]);
    expect(next.historyIndex).toBe(2);
  });

  it('travels history back/forward for local mode', () => {
    const input: PreviewNavigationState = {
      ...baseState(),
      hasCustomPreviewUrl: true,
      history: [
        'http://localhost:3000/apps/scenario-1/proxy/',
        'http://localhost:3000/apps/scenario-1/proxy/?path=a',
        'http://localhost:3000/apps/scenario-1/proxy/?path=b',
      ],
      historyIndex: 2,
      previewUrl: 'http://localhost:3000/apps/scenario-1/proxy/?path=b',
      previewUrlInput: 'http://localhost:3000/apps/scenario-1/proxy/?path=b',
    };

    const back = reducePreviewNavigationState(input, {
      type: 'travel-history',
      direction: 'back',
    });
    expect(back.historyIndex).toBe(1);
    expect(back.previewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=a');
    expect(back.previewUrlInput).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=a');

    const forward = reducePreviewNavigationState(back, {
      type: 'travel-history',
      direction: 'forward',
    });
    expect(forward.historyIndex).toBe(2);
    expect(forward.previewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=b');
    expect(forward.previewUrlInput).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=b');
  });

  it('syncs bridge href into persisted navigation state without replacing iframe source URL', () => {
    const input = baseState();
    const next = reducePreviewNavigationState(input, {
      type: 'sync-from-bridge',
      href: 'http://localhost:3000/apps/scenario-1/proxy/?path=README.md',
    });

    expect(next.previewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/');
    expect(next.previewUrlInput).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=README.md');
    expect(next.hasCustomPreviewUrl).toBe(true);
    expect(next.initialPreviewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/?path=README.md');
    expect(next.history).toEqual([
      'http://localhost:3000/apps/scenario-1/proxy/',
      'http://localhost:3000/apps/scenario-1/proxy/?path=README.md',
    ]);
    expect(next.historyIndex).toBe(1);
  });
});
