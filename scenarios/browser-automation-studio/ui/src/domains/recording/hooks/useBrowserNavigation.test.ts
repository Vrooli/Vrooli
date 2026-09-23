import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useBrowserNavigation } from './useBrowserNavigation';

vi.mock('@/config', () => ({ getConfig: async () => ({ API_URL: 'http://fixture.test' }) }));

describe('browser history popup [REQ:BAS-RH-J04]', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('retains untitled browser entries without fabricated visit times and filters malformed entries', async () => {
    const back = { url: 'https://fixture.test/one', title: '' };
    const current = { url: 'https://fixture.test/two', title: 'Current' };
    const forward = { url: 'https://fixture.test/three', title: '', timestamp: '2026-09-22T00:00:00Z' };
    const fetch = vi.fn(async () => new Response(JSON.stringify({
      back_stack: [back, null, { url: 1, title: '' }, { url: 'https://invalid.test', title: null }],
      current, forward_stack: [forward],
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ sessionId: 'owned-session' }));
    await act(async () => {
      expect(await result.current.handleFetchNavigationStack()).toEqual({
        backStack: [{ ...back, timestamp: undefined }], current: { ...current, timestamp: undefined }, forwardStack: [forward],
      });
    });
    expect(fetch).toHaveBeenCalledWith('http://fixture.test/recordings/live/owned-session/navigation-stack');
  });

  it('returns no popup for a rejected session read', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', { status: 404 })));
    const { result } = renderHook(() => useBrowserNavigation({ sessionId: 'closed-session' }));
    await act(async () => expect(await result.current.handleFetchNavigationStack()).toBeNull());
  });
});
