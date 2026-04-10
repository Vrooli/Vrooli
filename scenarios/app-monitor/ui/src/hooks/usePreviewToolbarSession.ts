import { useCallback, useMemo } from 'react';
import type { MouseEvent as ReactMouseEvent } from 'react';
import { APP_OPEN_MODE_QUERY_KEY, type AppOpenMode } from '@/components/tabSwitcher/tabSwitcherOpenMode';
import type { useOverlayRouter } from '@/hooks/useOverlayRouter';
import type { App } from '@/types';
import { resolvePreviewUrlCandidate } from '@/utils/previewUrl';
import {
  buildPreviewSuggestionSections,
  type PreviewSuggestionSection,
} from '@/utils/workspaceDiscovery';

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

export const buildPreviewUrlSuggestions = (
  history: string[],
  apps: App[],
  referenceUrl: string | null = null,
): string[] => {
  const sections = buildPreviewSuggestionSections({
    apps,
    history,
    query: '',
    referenceUrl,
  });
  return sections.flatMap((section) => section.items.map((item) => item.value));
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
  const openPreviewTarget = useMemo(() => {
    const preferredTarget = bridgeHref || previewUrl || '';
    return resolvePreviewUrlCandidate(preferredTarget, previewUrl || bridgeHref) ?? preferredTarget;
  }, [bridgeHref, previewUrl]);

  const buildUrlSuggestionSections = useCallback((query: string): PreviewSuggestionSection[] => (
    buildPreviewSuggestionSections({
      apps,
      history,
      query,
      referenceUrl: bridgeHref || previewUrl,
    })
  ), [apps, bridgeHref, history, previewUrl]);

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
    buildUrlSuggestionSections,
    handleOpenScenarioSelector,
    handleOpenPreviewInNewTab,
  };
}

export default usePreviewToolbarSession;
