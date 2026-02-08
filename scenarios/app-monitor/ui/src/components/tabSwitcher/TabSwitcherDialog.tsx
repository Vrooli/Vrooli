import { useCallback, useEffect, useRef, useState, type RefObject } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import clsx from 'clsx';
import { useAppCatalog, normalizeAppSort, type AppSortOption } from '@/hooks/useAppCatalog';
import { useResourcesCatalog, normalizeResourceSort, type ResourceSortOption } from '@/hooks/useResourcesCatalog';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { resolveAppIdentifier } from '@/utils/appPreview';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useKeyboardScope } from '@/hooks/useKeyboardScopes';
import { resolveTabSwitcherShortcut, type ShortcutState } from '@/utils/tabSwitcherShortcut';
import type { App, Resource } from '@/types';
import {
  WORKSPACE_INTENT_APP_ID_KEY,
  WORKSPACE_INTENT_MODE_KEY,
} from '@/features/preview-workspace/utils/navigationIntent';
import { TabSwitcherControls, TabSwitcherHeader } from './TabSwitcherControls';
import { AppsSection, ResourcesSection } from './TabSwitcherSections';
import { SEGMENT_QUERY_KEY } from './tabSwitcherConstants';
import { useTabSwitcherFiltering, useTabSwitcherSegment } from './tabSwitcherHooks';
import type { SortOption } from './TabSwitcherCards';
import {
  APP_OPEN_MODE_QUERY_KEY,
  APP_OPEN_MODE_LABELS,
  APP_OPEN_MODES,
  cycleAppOpenMode,
  isAppOpenMode,
  resolveAppOpenModeShortcut,
  type AppOpenMode,
} from './tabSwitcherOpenMode';
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
  const [searchParams, setSearchParams] = useSearchParams();
  const segmentParam = searchParams.get(SEGMENT_QUERY_KEY);
  const { closeOverlay } = useOverlayRouter();
  const [search, setSearch] = useState('');
  const [sortOption, setSortOption] = useState<AppSortOption>('status');
  const [resourceSortOption, setResourceSortOption] = useState<ResourceSortOption>('status');
  const requestedOpenMode = searchParams.get(APP_OPEN_MODE_QUERY_KEY);
  const [appOpenMode, setAppOpenMode] = useState<AppOpenMode>(
    () => (isAppOpenMode(requestedOpenMode) ? requestedOpenMode : 'single-preview'),
  );
  const [shortcut, setShortcut] = useState<ShortcutState | null>(null);
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
  const { filteredApps, recentApps } = useAppCatalog({ search, sort: sortOption, historyLimit: 12 });
  const { sortedResources } = useResourcesCatalog({ sort: resourceSortOption });

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

  useEffect(() => {
    setShortcut(resolveTabSwitcherShortcut());
  }, []);

  useEffect(() => {
    if (isAppOpenMode(requestedOpenMode)) {
      setAppOpenMode(requestedOpenMode);
    }
  }, [requestedOpenMode]);

  const handleClose = () => {
    closeOverlay({
      preserve: ['segment'],
      params: {
        [APP_OPEN_MODE_QUERY_KEY]: null,
      },
    });
  };

  const moveFocusByStep = useCallback((step: number) => {
    const root = dialogRef.current;
    if (!root) {
      return;
    }

    const focusable = getFocusable(root);
    if (focusable.length === 0) {
      return;
    }

    const active = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const currentIndex = active ? focusable.indexOf(active) : -1;
    const baseIndex = currentIndex === -1 ? (step > 0 ? -1 : 0) : currentIndex;
    const nextIndex = (baseIndex + step + focusable.length) % focusable.length;
    const next = focusable[nextIndex];
    next?.focus({ preventScroll: true });
    next?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, []);

  const moveGridCardFocus = useCallback((direction: 1 | -1, target: HTMLElement): boolean => {
    const focusedCard = target.closest<HTMLElement>('.tab-card');
    const grid = focusedCard?.closest<HTMLElement>('.tab-switcher__grid');
    if (!focusedCard || !grid) {
      return false;
    }

    const cards = Array.from(grid.querySelectorAll<HTMLElement>('.tab-card')).filter(
      card => !card.hasAttribute('disabled') && card.tabIndex !== -1,
    );
    if (cards.length === 0) {
      return false;
    }

    const currentIndex = cards.indexOf(focusedCard);
    if (currentIndex === -1) {
      return false;
    }

    const firstRowTop = Math.round(cards[0]?.getBoundingClientRect().top ?? 0);
    let columns = 0;
    for (const card of cards) {
      const top = Math.round(card.getBoundingClientRect().top);
      if (Math.abs(top - firstRowTop) > 2) {
        break;
      }
      columns += 1;
    }

    const step = Math.max(1, columns);
    const nextIndex = currentIndex + (step * direction);
    const next = cards[nextIndex];
    if (!next) {
      return false;
    }

    next.focus({ preventScroll: true });
    next.scrollIntoView({ block: 'nearest', inline: 'nearest' });
    return true;
  }, []);

  useKeyboardScope({
    id: 'tab-switcher-dialog-keys',
    priority: 900,
    onKeyDown: (event) => {
      const target = event.target;
      if (!dialogRef.current || !(target instanceof Node) || !dialogRef.current.contains(target)) {
        return false;
      }

      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        handleClose();
        return true;
      }

      const openModeShortcut = resolveAppOpenModeShortcut(event);
      if (openModeShortcut) {
        event.preventDefault();
        event.stopPropagation();

        if (openModeShortcut === 'cycle') {
          setAppOpenMode((current) => cycleAppOpenMode(current));
        } else {
          setAppOpenMode(openModeShortcut);
        }
        return true;
      }

      const key = event.key;
      const isArrowKey = key === 'ArrowLeft' || key === 'ArrowRight' || key === 'ArrowUp' || key === 'ArrowDown';
      if (!isArrowKey) {
        return false;
      }

      const keyTarget = event.target as HTMLElement | null;
      if (!keyTarget) {
        return false;
      }

      if (keyTarget.tagName === 'INPUT' || keyTarget.tagName === 'TEXTAREA' || keyTarget.tagName === 'SELECT' || keyTarget.isContentEditable) {
        return false;
      }

      if (key === 'ArrowUp' || key === 'ArrowDown') {
        const moved = moveGridCardFocus(key === 'ArrowDown' ? 1 : -1, keyTarget);
        if (!moved) {
          return false;
        }
        event.preventDefault();
        return true;
      }

      event.preventDefault();
      moveFocusByStep(key === 'ArrowRight' ? 1 : -1);
      return true;
    },
  });

  const handleAppSelect = (app: App, options?: { navigationId?: string }) => {
    const identifier = options?.navigationId ?? resolveAppIdentifier(app) ?? app.id;
    if (!identifier) {
      return;
    }
    closeOverlay({ replace: true });
    const encodedIdentifier = encodeURIComponent(identifier);
    const targetPath = appOpenMode === 'single-preview'
      ? `/apps/${encodedIdentifier}/preview`
      : '/apps/workspace';
    const navigationState = {
      fromAppsList: true,
      originAppId: app.id,
      navTimestamp: Date.now(),
    } as const;

    let workspaceTarget: string | null = null;
    if (appOpenMode !== 'single-preview') {
      const workspaceParams = new URLSearchParams();
      workspaceParams.set(WORKSPACE_INTENT_APP_ID_KEY, identifier);
      workspaceParams.set(WORKSPACE_INTENT_MODE_KEY, appOpenMode);
      workspaceTarget = `/apps/workspace?${workspaceParams.toString()}`;
    }

    if (appOpenMode === 'single-preview') {
      navigate(targetPath, {
        state: navigationState,
      });
      return;
    }

    navigate(workspaceTarget ?? '/apps/workspace', {
      state: navigationState,
    });
  };

  const handleResourceSelect = (resource: Resource) => {
    closeOverlay({ replace: true });
    navigate(`/resources/${encodeURIComponent(resource.id)}`);
  };

  const handleSearchEnter = () => {
    if (activeSegment === 'apps') {
      const firstApp = showAppHistory ? (recentApps[0] ?? filteredApps[0]) : filteredApps[0];
      if (firstApp) {
        handleAppSelect(firstApp);
      }
      return;
    }

    const firstResource = filteredResources[0];
    if (firstResource) {
      handleResourceSelect(firstResource);
    }
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
      <TabSwitcherHeader title={activeSegmentLabel} shortcut={shortcut} onClose={handleClose} />

      <TabSwitcherControls
        activeSegment={activeSegment}
        onSegmentSelect={handleSegmentSelect}
        search={search}
        onSearchChange={setSearch}
        onSearchClear={() => setSearch('')}
        onSearchEnter={handleSearchEnter}
        searchInputRef={searchInputRef}
      />

      {activeSegment === 'apps' && (
        <section className="tab-switcher__open-mode" aria-label="Scenario open destination">
          <span className="tab-switcher__open-mode-label">
            Open Scenarios In
            <span className="tab-switcher__open-mode-hint" aria-label="Shortcut hints">
              <kbd>Alt+O</kbd>
              <span>/</span>
              <kbd>Alt+1..3</kbd>
            </span>
          </span>
          <div className="tab-switcher__open-mode-options" role="radiogroup" aria-label="Scenario open mode">
            {APP_OPEN_MODES.map((mode) => {
              const checked = appOpenMode === mode;
              return (
                <button
                  key={mode}
                  type="button"
                  role="radio"
                  aria-checked={checked}
                  className={clsx(
                    'tab-switcher__open-mode-btn',
                    checked && 'tab-switcher__open-mode-btn--active',
                  )}
                  onClick={() => setAppOpenMode(mode)}
                >
                  {APP_OPEN_MODE_LABELS[mode]}
                </button>
              );
            })}
          </div>
        </section>
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
      if (!first || !last) return;
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
