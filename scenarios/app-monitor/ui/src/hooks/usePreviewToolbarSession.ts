import { useCallback, useMemo } from 'react';
import type { MouseEvent as ReactMouseEvent } from 'react';
import { APP_OPEN_MODE_QUERY_KEY, type AppOpenMode } from '@/components/tabSwitcher/tabSwitcherOpenMode';
import type { useOverlayRouter } from '@/hooks/useOverlayRouter';
import type { App } from '@/types';
import { buildPreviewUrl } from '@/utils/appPreview';

type OpenOverlayFn = ReturnType<typeof useOverlayRouter>['openOverlay'];

interface UsePreviewToolbarSessionOptions {
  bridgeHref: string | null;
  previewUrl: string | null;
  history: string[];
  apps: App[];
  openOverlay: OpenOverlayFn;
  appOpenMode: AppOpenMode;
  onBeforeOpenScenarioSelector?: () => void;
}

export const buildPreviewUrlSuggestions = (history: string[], apps: App[]): string[] => {
  const seen = new Set<string>();
  const suggestions: string[] = [];
  const addSuggestion = (value: string | null | undefined) => {
    if (!value) {
      return;
    }
    const trimmed = value.trim();
    if (trimmed.length === 0 || seen.has(trimmed)) {
      return;
    }
    seen.add(trimmed);
    suggestions.push(trimmed);
  };

  for (let index = history.length - 1; index >= 0; index -= 1) {
    addSuggestion(history[index]);
    if (suggestions.length >= 10) {
      break;
    }
  }

  for (const app of apps) {
    addSuggestion(buildPreviewUrl(app));
    if (suggestions.length >= 16) {
      break;
    }
  }

  return suggestions;
};

export function usePreviewToolbarSession({
  bridgeHref,
  previewUrl,
  history,
  apps,
  openOverlay,
  appOpenMode,
  onBeforeOpenScenarioSelector,
}: UsePreviewToolbarSessionOptions) {
  const openPreviewTarget = useMemo(() => bridgeHref || previewUrl || '', [bridgeHref, previewUrl]);

  const urlSuggestions = useMemo(() => buildPreviewUrlSuggestions(history, apps), [apps, history]);

  const handleOpenScenarioSelector = useCallback(() => {
    onBeforeOpenScenarioSelector?.();
    openOverlay('tabs', {
      params: {
        segment: 'apps',
        [APP_OPEN_MODE_QUERY_KEY]: appOpenMode,
      },
    });
  }, [appOpenMode, onBeforeOpenScenarioSelector, openOverlay]);

  const handleOpenPreviewInNewTab = useCallback((event: ReactMouseEvent<HTMLButtonElement>) => {
    if (!openPreviewTarget || typeof window === 'undefined') {
      return;
    }

    event.preventDefault();
    window.open(openPreviewTarget, '_blank', 'noopener');
  }, [openPreviewTarget]);

  return {
    openPreviewTarget,
    urlSuggestions,
    handleOpenScenarioSelector,
    handleOpenPreviewInNewTab,
  };
}

export default usePreviewToolbarSession;
