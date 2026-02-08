import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { MouseEvent as ReactMouseEvent } from 'react';
import type { App } from '@/types';
import { buildPreviewUrlSuggestions, usePreviewToolbarSession } from './usePreviewToolbarSession';

const createApp = (id: string, port: number): App => ({
  id,
  name: id,
  scenario_name: id,
  path: `/tmp/${id}`,
  created_at: '2026-02-07T00:00:00Z',
  updated_at: '2026-02-07T00:00:00Z',
  status: 'running',
  port_mappings: { UI_PORT: port },
  environment: {},
  config: {},
});

describe('buildPreviewUrlSuggestions', () => {
  it('prioritizes newest history entries and de-duplicates values', () => {
    const apps = [
      createApp('scenario-a', 4310),
      createApp('scenario-b', 4311),
    ];
    const suggestions = buildPreviewUrlSuggestions(
      ['http://localhost:4310', 'http://localhost:9999', 'http://localhost:4310'],
      apps,
      'http://localhost:3000/apps/app-monitor/proxy/',
    );

    expect(suggestions[0]).toBe('http://localhost:4310/');
    expect(suggestions[1]).toBe('http://localhost:9999/');
    expect(new Set(suggestions).size).toBe(suggestions.length);
  });

  it('normalizes relative proxy paths to absolute URLs', () => {
    const apps = [createApp('scenario-a', 4310)];
    const suggestions = buildPreviewUrlSuggestions(
      ['/apps/git-control-tower/proxy/'],
      apps,
      'http://localhost:3000/apps/app-monitor/proxy/',
    );

    expect(suggestions).toContain('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(suggestions).toContain('http://localhost:3000/apps/scenario-a/proxy/');
  });
});

describe('usePreviewToolbarSession', () => {
  it('opens the tab switcher with the requested open mode', () => {
    const openOverlay = vi.fn();
    const { result } = renderHook(() => usePreviewToolbarSession({
      bridgeHref: null,
      previewUrl: 'http://localhost:4310',
      history: [],
      apps: [createApp('scenario-a', 4310)],
      openOverlay,
      appOpenMode: 'replace-focused',
    }));

    act(() => {
      result.current.handleOpenScenarioSelector();
    });

    expect(openOverlay).toHaveBeenCalledWith('tabs', {
      params: {
        segment: 'apps',
        appOpenMode: 'replace-focused',
      },
    });
  });

  it('opens preview target in a new tab when available', () => {
    const openOverlay = vi.fn();
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    const { result } = renderHook(() => usePreviewToolbarSession({
      bridgeHref: null,
      previewUrl: 'http://localhost:4310',
      history: [],
      apps: [createApp('scenario-a', 4310)],
      openOverlay,
      appOpenMode: 'single-preview',
    }));

    const preventDefault = vi.fn();
    act(() => {
      result.current.handleOpenPreviewInNewTab({
        preventDefault,
      } as unknown as ReactMouseEvent<HTMLButtonElement>);
    });

    expect(preventDefault).toHaveBeenCalled();
    expect(openSpy).toHaveBeenCalledWith('http://localhost:4310/', '_blank', 'noopener');
    openSpy.mockRestore();
  });

  it('opens relative preview targets in a new tab as absolute URLs', () => {
    const openOverlay = vi.fn();
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    const { result } = renderHook(() => usePreviewToolbarSession({
      bridgeHref: null,
      previewUrl: '/apps/prompt-manager/proxy/?skill=scientific-debugging',
      history: [],
      apps: [createApp('scenario-a', 4310)],
      openOverlay,
      appOpenMode: 'single-preview',
    }));

    const preventDefault = vi.fn();
    act(() => {
      result.current.handleOpenPreviewInNewTab({
        preventDefault,
      } as unknown as ReactMouseEvent<HTMLButtonElement>);
    });

    expect(preventDefault).toHaveBeenCalled();
    expect(openSpy).toHaveBeenCalledWith(
      'http://localhost:3000/apps/prompt-manager/proxy/?skill=scientific-debugging',
      '_blank',
      'noopener',
    );
    openSpy.mockRestore();
  });
});
