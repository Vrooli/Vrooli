import type { PreviewLocationState } from '@/types/preview';
import type { MutableRefObject } from 'react';
import { useEffect, useMemo, useRef } from 'react';
import type { Location, NavigateFunction } from 'react-router-dom';
import { PREVIEW_TIMEOUTS } from './previewConstants';

export const isIosSafariUserAgent = (): boolean => {
  if (typeof navigator === 'undefined') {
    return false;
  }

  const ua = navigator.userAgent || '';
  const isTouchCapableMac = /Macintosh/i.test(ua)
    && typeof navigator.maxTouchPoints === 'number'
    && navigator.maxTouchPoints > 1;
  const isClassicIos = /iP(ad|hone|od)/i.test(ua);
  const isWebKit = /WebKit/i.test(ua);
  const isExcludedBrowser = /CriOS|FxiOS|OPiOS|EdgiOS/i.test(ua);

  return (isClassicIos || isTouchCapableMac) && isWebKit && !isExcludedBrowser;
};

type PreviewGuardState = NonNullable<Window['__appMonitorPreviewGuard']>;

const createDefaultPreviewGuardState = (): PreviewGuardState => ({
  active: false,
  armedAt: 0,
  ttl: PREVIEW_TIMEOUTS.IOS_AUTOBACK_GUARD,
  key: null,
  appId: null,
  recoverPath: null,
  ignoreNextPopstate: false,
  lastSuppressedAt: 0,
  recoverState: null,
});

export const ensurePreviewGuard = (): PreviewGuardState | null => {
  if (typeof window === 'undefined') {
    return null;
  }

  if (!window.__appMonitorPreviewGuard) {
    window.__appMonitorPreviewGuard = createDefaultPreviewGuardState();
  }

  return window.__appMonitorPreviewGuard as PreviewGuardState;
};

export const primePreviewGuardForNavigation = (params: {
  appId: string | null;
  recoverPath: string;
  guardTtlMs?: number;
  recoverState?: unknown;
}) => {
  const guard = ensurePreviewGuard();
  if (!guard) {
    return;
  }

  const now = Date.now();
  const ttl = typeof params.guardTtlMs === 'number'
    ? params.guardTtlMs
    : PREVIEW_TIMEOUTS.IOS_AUTOBACK_GUARD;

  guard.active = true;
  guard.armedAt = now;
  guard.ttl = ttl;
  guard.key = null;
  guard.appId = params.appId;
  guard.recoverPath = params.recoverPath;
  guard.recoverState = params.recoverState ?? null;
  guard.ignoreNextPopstate = false;

  window.__appMonitorPreviewGuard = guard;
};

interface ExtendedPreviewLocationState extends PreviewLocationState {
  fromAppsList?: boolean;
  originAppId?: string;
  navTimestamp?: number;
  suppressedAutoBack?: boolean;
  autoSelectedAt?: number;
}

interface UseIosAutobackGuardOptions {
  appId: string | null;
  guardTtlMs: number;
  isIosSafari: boolean;
  lastAppId: string | null;
  location: Location;
  locationState: ExtendedPreviewLocationState | null;
  navigate: NavigateFunction;
  recordDebugEvent: (event: string, detail?: Record<string, unknown>) => void;
  recordNavigateEvent: (detail: Record<string, unknown>) => void;
  updatePreviewGuard: (patch: Record<string, unknown>) => void;
}

type IosPopGuard = {
  active: boolean;
  timeoutId: number | null;
  activatedAt: number;
  handledCount: number;
};

const clearGuardTimeout = (guard: IosPopGuard) => {
  if (typeof window === 'undefined') {
    guard.timeoutId = null;
    return;
  }

  if (guard.timeoutId !== null) {
    window.clearTimeout(guard.timeoutId);
    guard.timeoutId = null;
  }
};

const resetGuardState = (
  guard: IosPopGuard,
  iosGuardedLocationKeyRef: MutableRefObject<string | null>,
) => {
  iosGuardedLocationKeyRef.current = null;
  clearGuardTimeout(guard);
  guard.active = false;
  guard.handledCount = 0;
  guard.activatedAt = 0;
};

const deactivateGuard = (
  guard: IosPopGuard,
  iosGuardedLocationKeyRef: MutableRefObject<string | null>,
  updatePreviewGuard: (patch: Record<string, unknown>) => void,
) => {
  resetGuardState(guard, iosGuardedLocationKeyRef);
  updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
};

export const useIosAutobackGuard = ({
  appId,
  guardTtlMs,
  isIosSafari,
  lastAppId,
  location,
  locationState,
  navigate,
  recordDebugEvent,
  recordNavigateEvent,
  updatePreviewGuard,
}: UseIosAutobackGuardOptions) => {
  const iosPopGuardRef = useRef<IosPopGuard>({
    active: false,
    timeoutId: null,
    activatedAt: 0,
    handledCount: 0,
  });
  const iosGuardedLocationKeyRef = useRef<string | null>(null);

  // Compute preview location from current appId
  const previewLocation = useMemo(() => {
    return appId ? {
      pathname: `/apps/${encodeURIComponent(appId)}/preview`,
      search: location.search || '',
    } : null;
  }, [appId, location.search]);

  useEffect(() => {
    if (!isIosSafari) {
      deactivateGuard(iosPopGuardRef.current, iosGuardedLocationKeyRef, updatePreviewGuard);
      return;
    }

    const previewGuard = ensurePreviewGuard();
    const recoverPath = `${location.pathname}${location.search ?? ''}`;
    const guardMatchesRoute = Boolean(
      previewGuard?.active
      && typeof previewGuard.recoverPath === 'string'
      && previewGuard.recoverPath === recoverPath,
    );
    const shouldArmFromState = Boolean(locationState?.fromAppsList && !locationState?.suppressedAutoBack);
    const shouldArm = shouldArmFromState || guardMatchesRoute;

    if (!shouldArm) {
      deactivateGuard(iosPopGuardRef.current, iosGuardedLocationKeyRef, updatePreviewGuard);
      return;
    }

    if (iosGuardedLocationKeyRef.current === location.key) {
      return;
    }

    iosGuardedLocationKeyRef.current = location.key;
    const guard = iosPopGuardRef.current;
    guard.active = true;
    guard.activatedAt = Date.now();
    guard.handledCount = 0;
    clearGuardTimeout(guard);

    if (typeof window !== 'undefined') {
      guard.timeoutId = window.setTimeout(() => {
        guard.active = false;
        guard.timeoutId = null;
        updatePreviewGuard({ active: false, key: null });
      }, guardTtlMs);
    }

    recordDebugEvent('ios-guard-armed', {
      key: location.key,
      appId,
      navTimestamp: locationState?.navTimestamp,
    });

    updatePreviewGuard({
      active: true,
      armedAt: Date.now(),
      ttl: guardTtlMs,
      key: location.key,
      appId,
      recoverPath,
      ignoreNextPopstate: false,
      recoverState: typeof window !== 'undefined' ? window.history.state : undefined,
    });
  }, [
    appId,
    guardTtlMs,
    isIosSafari,
    location.key,
    location.pathname,
    location.search,
    locationState,
    recordDebugEvent,
    updatePreviewGuard,
  ]);

  useEffect(() => {
    if (!isIosSafari || typeof window === 'undefined') {
      return () => { };
    }

    const guard = iosPopGuardRef.current;

    const handlePopState = () => {
      if (!guard.active) {
        return;
      }

      const elapsed = Date.now() - guard.activatedAt;
      const shouldSuppress = guard.handledCount === 0 && elapsed >= 0 && elapsed <= guardTtlMs;

      if (shouldSuppress) {
        guard.handledCount = 1;
        guard.active = false;
        clearGuardTimeout(guard);

        const targetLocation = previewLocation;
        const hasTargetPath = Boolean(targetLocation?.pathname);

        recordDebugEvent('ios-popstate-suppressed', {
          elapsed,
          appId,
          targetPath: targetLocation?.pathname,
        });

        if (!hasTargetPath || !targetLocation) {
          return;
        }

        const nextState: PreviewLocationState = {
          ...(locationState ?? {}),
          fromAppsList: true,
          originAppId: locationState?.originAppId ?? lastAppId ?? undefined,
          navTimestamp: locationState?.navTimestamp ?? Date.now(),
          suppressedAutoBack: true,
        };

        if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
          window.requestAnimationFrame(() => {
            recordNavigateEvent({
              reason: 'ios-guard-restore',
              targetPath: targetLocation.pathname,
              targetSearch: targetLocation.search,
              replace: true,
            });
            navigate(
              {
                pathname: targetLocation.pathname,
                search: targetLocation.search || undefined,
              },
              {
                replace: true,
                state: nextState,
              },
            );
            recordDebugEvent('ios-guard-restore', {
              pathname: targetLocation.pathname,
              search: targetLocation.search,
              appId,
            });
          });
        } else {
          recordNavigateEvent({
            reason: 'ios-guard-restore',
            targetPath: targetLocation.pathname,
            targetSearch: targetLocation.search,
            replace: true,
          });
          navigate(
            {
              pathname: targetLocation.pathname,
              search: targetLocation.search || undefined,
            },
            {
              replace: true,
              state: nextState,
            },
          );
          recordDebugEvent('ios-guard-restore', {
            pathname: targetLocation.pathname,
            search: targetLocation.search,
            appId,
          });
        }

        return;
      }

      guard.active = false;
      clearGuardTimeout(guard);
      recordDebugEvent('ios-popstate-allowed', {
        elapsed,
        appId,
      });
    };

    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
      clearGuardTimeout(guard);
      guard.active = false;
      guard.handledCount = 0;
      recordDebugEvent('ios-guard-disposed', {
        appId,
      });
      updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
    };
  }, [
    appId,
    guardTtlMs,
    isIosSafari,
    lastAppId,
    locationState,
    navigate,
    previewLocation,
    recordDebugEvent,
    recordNavigateEvent,
    updatePreviewGuard,
  ]);

  useEffect(() => {
    if (!isIosSafari) {
      return;
    }

    if (!locationState?.fromAppsList || !locationState?.suppressedAutoBack) {
      return;
    }

    // Defer the state reset to avoid race condition with the popstate handler's restore navigation.
    // The popstate handler sets suppressedAutoBack=true, then navigates to restore the preview.
    // We need to wait for that navigation to complete before resetting the state flags.
    if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
      return;
    }

    const resetStateAfterRestore = () => {
      recordNavigateEvent({
        reason: 'ios-guard-reset-state',
        targetPath: location.pathname,
        targetSearch: location.search,
        replace: true,
      });

      navigate(
        {
          pathname: location.pathname,
          search: location.search || undefined,
        },
        {
          replace: true,
          state: {
            ...(locationState ?? {}),
            fromAppsList: false,
            suppressedAutoBack: false,
          },
        },
      );

      recordDebugEvent('ios-guard-reset-state', {
        pathname: location.pathname,
        search: location.search,
        appId,
      });

      updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
    };

    // Use double rAF to ensure the restore navigation has been processed
    const frame1 = window.requestAnimationFrame(() => {
      const frame2 = window.requestAnimationFrame(resetStateAfterRestore);
      // Store frame2 for cleanup if needed
      return () => window.cancelAnimationFrame(frame2);
    });

    return () => {
      window.cancelAnimationFrame(frame1);
    };
  }, [
    appId,
    isIosSafari,
    location.pathname,
    location.search,
    locationState,
    navigate,
    recordDebugEvent,
    recordNavigateEvent,
    updatePreviewGuard,
  ]);
};

export type { UseIosAutobackGuardOptions };
