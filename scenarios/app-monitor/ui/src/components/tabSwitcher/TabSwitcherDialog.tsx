import { logger } from '@/services/logger';
import { useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { useAppCatalog, normalizeAppSort, type AppSortOption } from '@/hooks/useAppCatalog';
import { useResourcesCatalog, normalizeResourceSort, type ResourceSortOption } from '@/hooks/useResourcesCatalog';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { resolveAppIdentifier } from '@/utils/appPreview';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useAutoNextScenario } from '@/hooks/useAutoNextScenario';
import { isIosSafariUserAgent, primePreviewGuardForNavigation } from '@/components/views/useIosAutobackGuard';
import type { App, Resource } from '@/types';
import { TabSwitcherControls, TabSwitcherHeader } from './TabSwitcherControls';
import { AppsSection, ResourcesSection } from './TabSwitcherSections';
import { SEGMENT_QUERY_KEY } from './tabSwitcherConstants';
import { useTabSwitcherFiltering, useTabSwitcherSegment } from './tabSwitcherHooks';
import type { SortOption } from './TabSwitcherCards';
import './TabSwitcherDialog.css';

const APP_SORT_OPTIONS: Array<SortOption<AppSortOption>> = [
  { value: 'status', label: 'Active first' },
  { value: 'completeness-high', label: 'Completeness: High → Low' },
  { value: 'completeness-low', label: 'Completeness: Low → High' },
  { value: 'recently-viewed', label: 'Recently viewed' },
  { value: 'least-recently-viewed', label: 'Least recently viewed' },
  { value: 'recently-updated', label: 'Recently updated' },
  { value: 'recently-added', label: 'Recently added' },
  { value: 'most-viewed', label: 'Most viewed' },
  { value: 'least-viewed', label: 'Least viewed' },
  { value: 'name-asc', label: 'A → Z' },
  { value: 'name-desc', label: 'Z → A' },
];

const RESOURCE_SORT_OPTIONS: Array<SortOption<ResourceSortOption>> = [
  { value: 'status', label: 'Status (Active first)' },
  { value: 'name-asc', label: 'Name A → Z' },
  { value: 'name-desc', label: 'Name Z → A' },
  { value: 'type', label: 'Type' },
];

const FOCUSABLE_SELECTOR = [
  'a[href]:not([tabindex="-1"])',
  'button:not([disabled]):not([tabindex="-1"])',
  'input:not([disabled]):not([type="hidden"]):not([tabindex="-1"])',
  'select:not([disabled]):not([tabindex="-1"])',
  'textarea:not([disabled]):not([tabindex="-1"])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

const getFocusable = (root: HTMLElement): HTMLElement[] => (
  Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter((element) => {
    if (element.hasAttribute('disabled')) {
      return false;
    }
    const ariaHidden = element.getAttribute('aria-hidden');
    if (ariaHidden === 'true') {
      return false;
    }
    const rect = element.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  })
);

export default function TabSwitcherDialog() {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const segmentParam = searchParams.get(SEGMENT_QUERY_KEY);
  const { closeOverlay } = useOverlayRouter();
  const [search, setSearch] = useState('');
  const [sortOption, setSortOption] = useState<AppSortOption>('status');
  const [resourceSortOption, setResourceSortOption] = useState<ResourceSortOption>('status');
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const appLoadState = useAppsStore(state => ({
    loadingInitial: state.loadingInitial,
    hasInitialized: state.hasInitialized,
    appsLength: state.apps.length,
    error: state.error,
  }));
  const loadApps = useAppsStore(state => state.loadApps);
  const resourceLoadState = useResourcesStore(state => ({
    loading: state.loading,
    hasInitialized: state.hasInitialized,
    resourcesLength: state.resources.length,
    error: state.error,
  }));
  const { apps, filteredApps, recentApps } = useAppCatalog({ search, sort: sortOption, historyLimit: 12 });
  const { sortedResources } = useResourcesCatalog({ sort: resourceSortOption });

  const {
    autoSelect: autoSelectScenario,
    status: autoNextStatus,
    message: autoNextMessage,
    resetAutoNextMessage,
  } = useAutoNextScenario();

  const currentAppId = useMemo(() => {
    const match = location.pathname.match(/\/apps\/([^/]+)\//);
    if (!match) {
      return null;
    }
    try {
      return decodeURIComponent(match[1]);
    } catch (error) {
      logger.warn('[tabSwitcher] Failed to decode current app id', error);
      return match[1];
    }
  }, [location.pathname]);

  const {
    activeSegment,
    activeSegmentLabel,
    handleSegmentSelect,
  } = useTabSwitcherSegment({ segmentParam, searchParams, setSearchParams });

  const {
    normalizedSearch,
    filteredResources,
    showAppHistory,
  } = useTabSwitcherFiltering({
    search,
    sortedResources,
    recentApps,
  });

  // Consider loading if:
  // 1. Initial load is in progress AND has not initialized yet
  // This prevents infinite loading when the store has initialized but returned empty results
  const isLoadingApps = appLoadState.loadingInitial && !appLoadState.hasInitialized;

  const isLoadingResources = (!resourceLoadState.hasInitialized && !resourceLoadState.error)
    && (resourceLoadState.loading || resourceLoadState.resourcesLength === 0);

  useDialogFocus(dialogRef, searchInputRef);

  const handleClose = () => {
    closeOverlay({ preserve: ['segment'] });
  };

  const handleAppSelect = (app: App, options?: { autoSelected?: boolean; navigationId?: string }) => {
    resetAutoNextMessage();
    const identifier = options?.navigationId ?? resolveAppIdentifier(app) ?? app.id;
    if (!identifier) {
      return;
    }
    closeOverlay({ replace: true });
    const targetPath = `/apps/${encodeURIComponent(identifier)}/preview`;
    const navigationState = {
      fromAppsList: true,
      originAppId: app.id,
      navTimestamp: Date.now(),
      autoSelected: Boolean(options?.autoSelected),
      autoSelectedAt: options?.autoSelected ? Date.now() : undefined,
    } as const;

    if (isIosSafariUserAgent()) {
      const guardAppId = app.id ?? identifier;
      primePreviewGuardForNavigation({
        appId: guardAppId,
        recoverPath: targetPath,
      });
    }

    navigate(targetPath, {
      state: navigationState,
    });
  };

  const handleResourceSelect = (resource: Resource) => {
    closeOverlay({ replace: true });
    navigate(`/resources/${encodeURIComponent(resource.id)}`);
  };

  const isAutoNextRunning = autoNextStatus === 'running';

  const handleAutoNext = async () => {
    if (isAutoNextRunning) {
      return;
    }
    await autoSelectScenario({
      apps,
      currentAppId,
      onSelect: (app, navigationId) => handleAppSelect(app, { autoSelected: true, navigationId }),
    });
  };

  const handleRetryLoadApps = async (): Promise<void> => {
    setIsRefreshing(true);
    setRefreshError(null);
    try {
      await loadApps({ force: true });
      // Give the store a moment to update before checking for success
      await new Promise(resolve => setTimeout(resolve, 100));
      const currentError = useAppsStore.getState().error;
      if (currentError) {
        setRefreshError(currentError);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to refresh scenarios';
      setRefreshError(message);
    } finally {
      setIsRefreshing(false);
    }
  };

  return (
    <div className="tab-switcher" ref={dialogRef}>
      <TabSwitcherHeader title={activeSegmentLabel} onClose={handleClose} />

      <TabSwitcherControls
        activeSegment={activeSegment}
        onSegmentSelect={handleSegmentSelect}
        search={search}
        onSearchChange={setSearch}
        onSearchClear={() => setSearch('')}
        searchInputRef={searchInputRef}
        showAutoNext={activeSegment === 'apps'}
        isAutoNextRunning={isAutoNextRunning}
        onAutoNext={handleAutoNext}
        disableAutoNext={apps.length === 0}
      />

      {activeSegment === 'apps' && autoNextMessage && (
        <div className="tab-switcher__auto-next-message" role="status">
          {autoNextMessage}
        </div>
      )}

      <div className="tab-switcher__content">
        {activeSegment === 'apps' && (
          <div className="tab-switcher__section">
            <AppsSection
              showHistory={showAppHistory}
              recentApps={recentApps}
              apps={filteredApps}
              normalizedSearch={normalizedSearch}
              sortOption={sortOption}
              onSortChange={value => setSortOption(normalizeAppSort(value))}
              onSelect={handleAppSelect}
              isLoading={isLoadingApps}
              onRetry={handleRetryLoadApps}
              errorMessage={refreshError}
              isRefreshing={isRefreshing}
              sortOptions={APP_SORT_OPTIONS}
            />
          </div>
        )}

        {activeSegment === 'resources' && (
          <div className="tab-switcher__section">
            <ResourcesSection
              resources={filteredResources}
              normalizedSearch={normalizedSearch}
              sortOption={resourceSortOption}
              onSortChange={value => setResourceSortOption(normalizeResourceSort(value))}
              onSelect={handleResourceSelect}
              isLoading={isLoadingResources}
              sortOptions={RESOURCE_SORT_OPTIONS}
            />
          </div>
        )}
      </div>
    </div>
  );
}
function useDialogFocus(dialogRef: RefObject<HTMLElement>, searchInputRef: RefObject<HTMLInputElement>) {
  const lastFocusedElementRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }
    lastFocusedElementRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
    const rafId = window.requestAnimationFrame(() => {
      searchInputRef.current?.focus({ preventScroll: true });
    });

    return () => {
      window.cancelAnimationFrame(rafId);
      const previous = lastFocusedElementRef.current;
      if (previous && typeof previous.focus === 'function') {
        previous.focus({ preventScroll: true });
      }
    };
  }, [searchInputRef]);

  useEffect(() => {
    const root = dialogRef.current;
    if (!root || typeof document === 'undefined') {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Tab') {
        return;
      }
      const focusable = getFocusable(root);
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const activeElement = document.activeElement as HTMLElement | null;

      if (event.shiftKey) {
        if (!activeElement || activeElement === first || !root.contains(activeElement)) {
          event.preventDefault();
          last.focus({ preventScroll: true });
        }
      } else if (activeElement === last) {
        event.preventDefault();
        first.focus({ preventScroll: true });
      }
    };

    root.addEventListener('keydown', handleKeyDown);
    return () => {
      root.removeEventListener('keydown', handleKeyDown);
    };
  }, [dialogRef]);
}
