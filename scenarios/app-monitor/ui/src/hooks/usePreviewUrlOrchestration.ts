import { useCallback } from 'react';
import type { MutableRefObject } from 'react';
import type { App } from '@/types';
import { buildPreviewUrl } from '@/utils/appPreview';

interface UsePreviewUrlOrchestrationOptions {
  hasCustomPreviewUrl: boolean;
  previewUrl: string | null;
  applyDefaultPreviewUrl: (url: string) => void;
  resetPreviewState: (options?: { force?: boolean }) => void;
  setPreviewUrl: (url: string | null) => void;
  initialPreviewUrlRef: MutableRefObject<string | null>;
}

interface SyncPreviewUrlOptions {
  appForPreview: App | null;
  fallbackPreviewUrl?: string | null;
  forceResetWhenMissingApp?: boolean;
}

interface SyncPreviewUrlResult {
  hasPreviewCandidate: boolean;
  defaultPreviewUrl: string | null;
}

const COMPARE_BASE_URL = 'http://localhost';

const normalizePathname = (value: string): string => {
  if (value === '/') {
    return value;
  }
  return value.endsWith('/') ? value.slice(0, -1) : value;
};

const shouldPreserveInAppNavigation = (
  currentUrl: string | null,
  defaultUrl: string | null,
): boolean => {
  if (!currentUrl || !defaultUrl) {
    return false;
  }

  try {
    const current = new URL(currentUrl, COMPARE_BASE_URL);
    const fallback = new URL(defaultUrl, COMPARE_BASE_URL);

    if (current.origin !== fallback.origin) {
      return false;
    }

    const currentPath = normalizePathname(current.pathname);
    const fallbackPath = normalizePathname(fallback.pathname);

    if (currentPath === fallbackPath) {
      return current.search !== fallback.search || current.hash !== fallback.hash;
    }

    return currentPath.startsWith(`${fallbackPath}/`);
  } catch {
    return false;
  }
};

export function usePreviewUrlOrchestration({
  hasCustomPreviewUrl,
  previewUrl,
  applyDefaultPreviewUrl,
  resetPreviewState,
  setPreviewUrl,
  initialPreviewUrlRef,
}: UsePreviewUrlOrchestrationOptions) {
  return useCallback(({
    appForPreview,
    fallbackPreviewUrl = null,
    forceResetWhenMissingApp = false,
  }: SyncPreviewUrlOptions): SyncPreviewUrlResult => {
    if (!appForPreview) {
      if (!hasCustomPreviewUrl) {
        if (fallbackPreviewUrl && previewUrl !== fallbackPreviewUrl) {
          resetPreviewState({ force: forceResetWhenMissingApp });
          applyDefaultPreviewUrl(fallbackPreviewUrl);
        } else if (!fallbackPreviewUrl) {
          resetPreviewState({ force: forceResetWhenMissingApp });
        }
      }
      return {
        hasPreviewCandidate: false,
        defaultPreviewUrl: fallbackPreviewUrl,
      };
    }

    const nextPreviewUrl = buildPreviewUrl(appForPreview);
    const hasPreviewCandidate = Boolean(nextPreviewUrl);
    const defaultPreviewUrl = hasPreviewCandidate ? (nextPreviewUrl as string) : fallbackPreviewUrl;

    if (!hasCustomPreviewUrl) {
      if (
        defaultPreviewUrl
        && previewUrl !== defaultPreviewUrl
        && !shouldPreserveInAppNavigation(previewUrl, defaultPreviewUrl)
      ) {
        applyDefaultPreviewUrl(defaultPreviewUrl);
      } else if (!defaultPreviewUrl) {
        resetPreviewState();
      }
    } else if (hasPreviewCandidate && previewUrl === null) {
      const resolvedUrl = nextPreviewUrl as string;
      initialPreviewUrlRef.current = resolvedUrl;
      setPreviewUrl(resolvedUrl);
    }

    return {
      hasPreviewCandidate,
      defaultPreviewUrl,
    };
  }, [
    applyDefaultPreviewUrl,
    hasCustomPreviewUrl,
    initialPreviewUrlRef,
    previewUrl,
    resetPreviewState,
    setPreviewUrl,
  ]);
}

export default usePreviewUrlOrchestration;
