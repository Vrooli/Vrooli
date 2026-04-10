import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { App } from '@/types';
import { usePaneMetadata } from './usePaneMetadata';

const { loggerDebugMock, loggerWarnMock } = vi.hoisted(() => ({
  loggerDebugMock: vi.fn(),
  loggerWarnMock: vi.fn(),
}));

vi.mock('@/services/logger', () => ({
  logger: {
    debug: loggerDebugMock,
    warn: loggerWarnMock,
    error: vi.fn(),
    info: vi.fn(),
  },
}));

const createApp = (overrides?: Partial<App>): App => ({
  id: 'scenario-1',
  name: 'Scenario One',
  scenario_name: 'scenario-one',
  path: '/tmp/scenario-one',
  created_at: '2026-02-07T00:00:00Z',
  updated_at: '2026-02-07T00:00:00Z',
  status: 'running',
  port_mappings: { UI_PORT: 4310 },
  environment: {},
  config: {},
  ...overrides,
});

describe('usePaneMetadata', () => {
  const hasLoggedEvent = (event: string): boolean => (
    loggerDebugMock.mock.calls.some((call) => {
      const payload = call[1] as { event?: string } | undefined;
      return payload?.event === event;
    })
  );

  beforeEach(() => {
    vi.useRealTimers();
    loggerDebugMock.mockReset();
    loggerWarnMock.mockReset();
  });

  it('deduplicates in-flight hydration requests for the same pane identifier', async () => {
    let resolveFetch: ((value: App | null) => void) | null = null;
    const getApp = vi.fn(() => new Promise<App | null>((resolve) => {
      resolveFetch = resolve;
    }));
    const setAppsState = vi.fn();
    const setStatusMessage = vi.fn();
    const onResetForMissingIdentifier = vi.fn();
    const shouldPreferExistingApp = vi.fn((existing: App | null) => Boolean(existing));

    const partial = createApp({ status: 'unknown', is_partial: true });
    const { result, rerender } = renderHook((props: { apps: App[] }) => usePaneMetadata({
      paneId: 'pane-1',
      apps: props.apps,
      resolvedAppIdentifier: partial.id,
      scenarioIdentifierFromUrl: null,
      shouldPreferExistingApp,
      setAppsState,
      getApp,
      setStatusMessage,
      onResetForMissingIdentifier,
      cooldownMs: 1_500,
    }), {
      initialProps: { apps: [partial] },
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledTimes(1);
      expect(result.current.isMetadataLoading).toBe(true);
      expect(hasLoggedEvent('requested')).toBe(true);
    });

    rerender({
      apps: [{ ...partial, updated_at: '2026-02-08T00:00:01Z' }],
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledTimes(1);
      expect(hasLoggedEvent('skippedInFlight')).toBe(true);
    });

    expect(resolveFetch).not.toBeNull();
    act(() => {
      resolveFetch?.(null);
    });

    await waitFor(() => {
      expect(result.current.isMetadataLoading).toBe(false);
      expect(hasLoggedEvent('notFound')).toBe(true);
    });
  });

  it('throttles repeat hydration calls within cooldown window', async () => {
    const getApp = vi.fn().mockResolvedValue(null);
    const setAppsState = vi.fn();
    const setStatusMessage = vi.fn();
    const onResetForMissingIdentifier = vi.fn();
    const shouldPreferExistingApp = vi.fn(() => false);
    const partial = createApp({ status: 'unknown', is_partial: true });

    const { result, rerender } = renderHook((props: { apps: App[] }) => usePaneMetadata({
      paneId: 'pane-1',
      apps: props.apps,
      resolvedAppIdentifier: partial.id,
      scenarioIdentifierFromUrl: null,
      shouldPreferExistingApp,
      setAppsState,
      getApp,
      setStatusMessage,
      onResetForMissingIdentifier,
      cooldownMs: 100_000,
    }), {
      initialProps: { apps: [partial] },
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledTimes(1);
      expect(hasLoggedEvent('notFound')).toBe(true);
    });

    rerender({
      apps: [{ ...partial, updated_at: '2026-02-08T00:00:02Z' }],
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledTimes(1);
      expect(hasLoggedEvent('skippedCooldown')).toBe(true);
    });
  });

  it('ignores stale request results after identifier switch', async () => {
    let resolveFirst: ((value: App | null) => void) | null = null;
    const appOne = createApp({ id: 'scenario-1', scenario_name: 'scenario-one', status: 'unknown', is_partial: true });
    const appTwo = createApp({ id: 'scenario-2', scenario_name: 'scenario-two', status: 'unknown', is_partial: true });
    const appTwoHydrated = createApp({ id: 'scenario-2', scenario_name: 'scenario-two', status: 'running', is_partial: false });
    const getApp = vi.fn((identifier: string) => {
      if (identifier === appOne.id) {
        return new Promise<App | null>((resolve) => {
          resolveFirst = resolve;
        });
      }
      if (identifier === appTwo.id) {
        return Promise.resolve(appTwoHydrated);
      }
      return Promise.resolve(null);
    });
    const setAppsState = vi.fn();
    const setStatusMessage = vi.fn();
    const onResetForMissingIdentifier = vi.fn();
    const shouldPreferExistingApp = vi.fn(() => false);

    const { result, rerender } = renderHook((props: { apps: App[]; resolvedId: string | null }) => usePaneMetadata({
      paneId: 'pane-1',
      apps: props.apps,
      resolvedAppIdentifier: props.resolvedId,
      scenarioIdentifierFromUrl: null,
      shouldPreferExistingApp,
      setAppsState,
      getApp,
      setStatusMessage,
      onResetForMissingIdentifier,
      cooldownMs: 1_500,
    }), {
      initialProps: { apps: [appOne], resolvedId: appOne.id },
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledWith(appOne.id);
      expect(result.current.isMetadataLoading).toBe(true);
    });

    rerender({
      apps: [appTwo],
      resolvedId: appTwo.id,
    });

    await waitFor(() => {
      expect(getApp).toHaveBeenCalledWith(appTwo.id);
      expect(result.current.currentApp?.id).toBe(appTwoHydrated.id);
      expect(hasLoggedEvent('completed')).toBe(true);
    });

    expect(resolveFirst).not.toBeNull();
    act(() => {
      resolveFirst?.(createApp({ ...appOne, status: 'running', is_partial: false }));
    });

    await waitFor(() => {
      expect(result.current.currentApp?.id).toBe(appTwoHydrated.id);
      expect(hasLoggedEvent('ignoredStale')).toBe(true);
    });
  });
});
