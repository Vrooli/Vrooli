import { logger } from '@/services/logger';
import { FormEvent, useEffect, useId, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { Layers, Server, Globe, Search, SlidersHorizontal, X, ExternalLink, Trash2, Plus, Shuffle, Loader2, Eye, RefreshCw } from 'lucide-react';
import clsx from 'clsx';
import { selectScreenshotBySurface, useSurfaceMediaStore } from '@/state/surfaceMediaStore';
import { useAppCatalog, normalizeAppSort, type AppSortOption } from '@/hooks/useAppCatalog';
import { useResourcesCatalog, normalizeResourceSort, type ResourceSortOption } from '@/hooks/useResourcesCatalog';
import { useBrowserTabsStore, type BrowserTabRecord, type BrowserTabHistoryRecord } from '@/state/browserTabsStore';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { AppsGridSkeleton, ResourcesGridSkeleton } from '@/components/LoadingSkeleton';
import { resolveAppIdentifier } from '@/utils/appPreview';
import { ensureDataUrl } from '@/utils/dataUrl';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useAutoNextScenario } from '@/hooks/useAutoNextScenario';
import type { App, Resource } from '@/types';
import './TabSwitcherDialog.css';

const SEGMENTS = [
  { id: 'apps' as const, label: 'Scenarios', icon: Layers },
  { id: 'resources' as const, label: 'Resources', icon: Server },
  { id: 'web' as const, label: 'Web Tabs', icon: Globe },
];

type SegmentId = typeof SEGMENTS[number]['id'];

const SEGMENT_QUERY_KEY = 'segment';

const resolveSegment = (value: string | null): SegmentId => {
  if (value === 'resources' || value === 'web') {
    return value;
  }
  return 'apps';
};

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

const formatViewCount = (value?: number | string | null): string | null => {
  if (value == null) {
    return null;
  }
  const numeric = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return null;
  }
  if (numeric >= 1000) {
    return `${(numeric / 1000).toFixed(1)}k`;
  }
  return String(Math.round(numeric));
};

const matchesResourceSearch = (resource: Resource, query: string): boolean => {
  if (!query) {
    return true;
  }
  const haystacks = [
    resource.name,
    resource.type,
    resource.description,
    resource.status,
    resource.id,
  ];
  return haystacks.some(entry => typeof entry === 'string' && entry.toLowerCase().includes(query));
};

export default function TabSwitcherDialog() {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const segmentParam = searchParams.get(SEGMENT_QUERY_KEY);
  const { closeOverlay } = useOverlayRouter();
  const [search, setSearch] = useState('');
  const [activeSegment, setActiveSegment] = useState<SegmentId>(() => resolveSegment(segmentParam));
  const [sortOption, setSortOption] = useState<AppSortOption>('status');
  const [resourceSortOption, setResourceSortOption] = useState<ResourceSortOption>('status');
  const [newWebTabUrl, setNewWebTabUrl] = useState('');
  const [newWebTabError, setNewWebTabError] = useState<string | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const webTabErrorId = useId();
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const lastFocusedElementRef = useRef<HTMLElement | null>(null);
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
  const normalizedSearch = search.trim().toLowerCase();
  const { apps, filteredApps, recentApps } = useAppCatalog({ search, sort: sortOption, historyLimit: 12 });
  const { sortedResources } = useResourcesCatalog({ sort: resourceSortOption });
  const setSurfaceScreenshot = useSurfaceMediaStore(state => state.setScreenshot);
  const {
    tabs: browserTabs,
    history: browserHistory,
    activateTab: activateBrowserTab,
    closeTab: closeBrowserTab,
    openTab: openBrowserTab,
    updateTab: updateBrowserTab,
  } = useBrowserTabsStore(state => ({
    tabs: state.tabs,
    history: state.history,
    activateTab: state.activateTab,
    closeTab: state.closeTab,
    openTab: state.openTab,
    updateTab: state.updateTab,
  }));

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

  const filteredResources = useMemo(() => (
    sortedResources.filter(resource => matchesResourceSearch(resource, normalizedSearch))
  ), [sortedResources, normalizedSearch]);

  const filteredActiveWebTabs = useMemo(() => {
    const items = [...browserTabs].sort((a, b) => b.lastActiveAt - a.lastActiveAt);
    if (!normalizedSearch) {
      return items;
    }
    return items.filter(tab => tab.title.toLowerCase().includes(normalizedSearch)
      || tab.url.toLowerCase().includes(normalizedSearch));
  }, [browserTabs, normalizedSearch]);

  const filteredHistoryTabs = useMemo(() => (
    !normalizedSearch
      ? browserHistory
      : browserHistory.filter(tab => tab.title.toLowerCase().includes(normalizedSearch)
        || tab.url.toLowerCase().includes(normalizedSearch))
  ), [browserHistory, normalizedSearch]);

  // Consider loading if:
  // 1. Initial load is in progress AND has not initialized yet
  // This prevents infinite loading when the store has initialized but returned empty results
  const isLoadingApps = appLoadState.loadingInitial && !appLoadState.hasInitialized;

  const isLoadingResources = (!resourceLoadState.hasInitialized && !resourceLoadState.error)
    && (resourceLoadState.loading || resourceLoadState.resourcesLength === 0);

  useEffect(() => {
    setActiveSegment(resolveSegment(segmentParam));
  }, [segmentParam]);

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
  }, []);

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
  }, []);

  const handleSegmentSelect = (segmentId: SegmentId) => {
    setActiveSegment(segmentId);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set(SEGMENT_QUERY_KEY, segmentId);
    setSearchParams(nextParams, { replace: true });
  };

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
    navigate(`/apps/${encodeURIComponent(identifier)}/preview`, {
      state: {
        fromAppsList: true,
        originAppId: app.id,
        navTimestamp: Date.now(),
        autoSelected: Boolean(options?.autoSelected),
        autoSelectedAt: options?.autoSelected ? Date.now() : undefined,
      },
    });
  };

  const handleResourceSelect = (resource: Resource) => {
    closeOverlay({ replace: true });
    navigate(`/resources/${encodeURIComponent(resource.id)}`);
  };

  const handleWebTabOpen = (tab: BrowserTabRecord) => {
    activateBrowserTab(tab.id);
    window.open(tab.url, '_blank', 'noopener,noreferrer');
    closeOverlay();
  };

  const handleWebTabClose = (tab: BrowserTabRecord) => {
    closeBrowserTab(tab.id);
  };

  const handleHistoryReopen = (entry: BrowserTabHistoryRecord) => {
    const reopened = openBrowserTab({ url: entry.url, title: entry.title });
    const screenshotDataUrl = ensureDataUrl(entry.screenshotData);
    if (screenshotDataUrl) {
      updateBrowserTab(reopened.id, {
        screenshotData: screenshotDataUrl,
        screenshotWidth: entry.screenshotWidth,
        screenshotHeight: entry.screenshotHeight,
        screenshotNote: entry.screenshotNote,
      });
      setSurfaceScreenshot('web', reopened.id, {
        dataUrl: screenshotDataUrl,
        width: entry.screenshotWidth ?? 0,
        height: entry.screenshotHeight ?? 0,
        capturedAt: Date.now(),
        note: entry.screenshotNote ?? 'Restored from history',
        source: 'restored',
      });
    }
    handleWebTabOpen(reopened);
  };

  const handleWebTabCreate = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = parseWebTabInput(newWebTabUrl);
    if (!parsed) {
      setNewWebTabError('Enter a valid URL');
      return;
    }

    const record = openBrowserTab({ url: parsed.url, title: parsed.title });
    setNewWebTabUrl('');
    setNewWebTabError(null);
    activateBrowserTab(record.id);
  };

  const handleNewWebTabInput = (value: string) => {
    setNewWebTabUrl(value);
    if (newWebTabError) {
      setNewWebTabError(null);
    }
  };

  const disableWebTabSubmit = newWebTabUrl.trim().length === 0;

  const showAppHistory = !normalizedSearch && recentApps.length > 0;
  const showResourceHistory = false;
  const showWebHistory = filteredHistoryTabs.length > 0;

  const activeSegmentLabel = SEGMENTS.find(segment => segment.id === activeSegment)?.label ?? '';
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
      <header className="tab-switcher__header">
        <div className="tab-switcher__header-text">
          <h2>{activeSegmentLabel}</h2>
        </div>
        <button
          type="button"
          className="tab-switcher__close"
          aria-label="Close tab switcher"
          onClick={handleClose}
        >
          <X aria-hidden />
        </button>
      </header>

      <div className="tab-switcher__controls">
        <div className="tab-switcher__search">
          <Search size={16} aria-hidden />
          <input
            type="text"
            ref={searchInputRef}
            value={search}
            onChange={event => setSearch(event.target.value)}
            placeholder="Search scenarios, resources, or tabs"
            aria-label="Search"
          />
          {search && (
            <button type="button" onClick={() => setSearch('')} aria-label="Clear search">
              <X size={14} aria-hidden />
            </button>
          )}
        </div>
        <div className="tab-switcher__segment">
          {SEGMENTS.map(segment => {
            const Icon = segment.icon;
            const isActive = segment.id === activeSegment;
            return (
              <button
                key={segment.id}
                type="button"
                className={clsx('tab-switcher__segment-btn', isActive && 'active')}
                onClick={() => handleSegmentSelect(segment.id)}
                aria-pressed={isActive}
                aria-label={segment.label}
              >
                <Icon size={16} aria-hidden />
                <span className="tab-switcher__segment-label">{segment.label}</span>
              </button>
            );
          })}
        </div>
        {activeSegment === 'apps' && (
          <button
            type="button"
            className="tab-switcher__auto-next"
            onClick={handleAutoNext}
            disabled={isAutoNextRunning || apps.length === 0}
          >
            <span className="tab-switcher__auto-next-icon">
              {isAutoNextRunning ? (
                <Loader2 size={24} aria-hidden className="tab-switcher__auto-next-spinner" />
              ) : (
                <Shuffle size={24} aria-hidden />
              )}
            </span>
            <span className="tab-switcher__auto-next-text">
              {isAutoNextRunning ? 'Selecting next scenario…' : 'Auto-next scenario'}
            </span>
          </button>
        )}
        {activeSegment === 'web' && (
          <>
            <form className="tab-switcher__web-form" onSubmit={handleWebTabCreate}>
              <Globe size={16} aria-hidden />
              <input
                type="text"
                inputMode="url"
                autoComplete="off"
                autoCorrect="off"
                autoCapitalize="none"
                spellCheck={false}
                value={newWebTabUrl}
                onChange={event => handleNewWebTabInput(event.target.value)}
                placeholder="https://example.com"
                aria-label="Open a new web tab"
                aria-invalid={newWebTabError ? 'true' : 'false'}
                aria-describedby={newWebTabError ? webTabErrorId : undefined}
              />
              {newWebTabUrl && (
                <button
                  type="button"
                  className="tab-switcher__clear"
                  onClick={() => handleNewWebTabInput('')}
                  aria-label="Clear URL"
                >
                  <X size={14} aria-hidden />
                </button>
              )}
              <button
                type="submit"
                className="tab-switcher__web-submit"
                disabled={disableWebTabSubmit}
              >
                <Plus size={16} aria-hidden />
                <span>Add tab</span>
              </button>
            </form>
            {newWebTabError && (
              <p className="tab-switcher__web-error" role="alert" id={webTabErrorId}>
                {newWebTabError}
              </p>
            )}
          </>
        )}
      </div>

      {activeSegment === 'apps' && autoNextMessage && (
        <div className="tab-switcher__auto-next-message" role="status">
          {autoNextMessage}
        </div>
      )}

      <div className="tab-switcher__content">
        {activeSegment === 'apps' && (
          <div className="tab-switcher__section">
            {showAppHistory && (
              <Section
                title="Recently opened"
                description="Your most visited scenarios."
              >
                <AppGrid apps={recentApps} onSelect={handleAppSelect} />
              </Section>
            )}
            <Section
              title={normalizedSearch ? 'Search results' : 'All scenarios'}
              description={normalizedSearch ? undefined : 'Browse the full catalog alphabetically or by status.'}
              actions={(
                <SortControl
                  label="Sort scenarios"
                  value={sortOption}
                  options={APP_SORT_OPTIONS}
                  onChange={value => setSortOption(normalizeAppSort(value))}
                />
              )}
            >
              <AppGrid
                apps={filteredApps}
                onSelect={handleAppSelect}
                emptyMessage="No scenarios match your search."
                isLoading={isLoadingApps}
                onRetry={handleRetryLoadApps}
                errorMessage={refreshError}
                isRefreshing={isRefreshing}
              />
            </Section>
          </div>
        )}

        {activeSegment === 'resources' && (
          <div className="tab-switcher__section">
            {showResourceHistory && (
              <Section title="Recent" description="Recently interacted resources.">
                <EmptyState message="Resource history coming soon." />
              </Section>
            )}
            <Section
              title={normalizedSearch ? 'Search results' : 'All resources'}
              description={normalizedSearch ? undefined : 'Operational tooling and shared services.'}
              actions={(
                <SortControl
                  label="Sort resources"
                  value={resourceSortOption}
                  options={RESOURCE_SORT_OPTIONS}
                  onChange={value => setResourceSortOption(normalizeResourceSort(value))}
                />
              )}
            >
              <ResourceGrid
                resources={filteredResources}
                onSelect={handleResourceSelect}
                emptyMessage="No resources match your search."
                isLoading={isLoadingResources}
              />
            </Section>
          </div>
        )}

        {activeSegment === 'web' && (
          <div className="tab-switcher__section">
            <Section
              title="Active tabs"
              description={filteredActiveWebTabs.length === 0 ? undefined : 'Tabs opened through App Monitor.'}
            >
              {filteredActiveWebTabs.length === 0
                ? <EmptyState message="No active web tabs yet." />
                : (
                  <WebTabGrid
                    tabs={filteredActiveWebTabs}
                    onOpen={handleWebTabOpen}
                    onClose={handleWebTabClose}
                  />
                )}
            </Section>
            <Section
              title="History"
              description={showWebHistory ? 'Most recently closed tabs.' : undefined}
            >
              {showWebHistory
                ? (
                  <WebHistoryList
                    entries={filteredHistoryTabs}
                    onReopen={handleHistoryReopen}
                  />
                )
                : <EmptyState message="No web browsing history yet." />}
            </Section>
          </div>
        )}
      </div>
    </div>
  );
}

function Section({ title, description, actions, children }: { title: string; description?: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <section className="tab-switcher__group">
      <header>
        <div className="tab-switcher__group-text">
          <h3>{title}</h3>
          {description && <p>{description}</p>}
        </div>
        {actions}
      </header>
      {children}
    </section>
  );
}

type SortControlProps = {
  label: string;
  value: string;
  options: Array<{ value: string; label: string }>;
  onChange(value: string): void;
};

function SortControl({ label, value, options, onChange }: SortControlProps) {
  return (
    <div className="tab-switcher__sort">
      <SlidersHorizontal size={16} aria-hidden />
      <select
        value={value}
        onChange={event => onChange(event.target.value)}
        aria-label={label}
      >
        {options.map(option => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

function AppGrid({
  apps,
  onSelect,
  emptyMessage,
  isLoading,
  skeletonCount = 8,
  onRetry,
  errorMessage,
  isRefreshing,
}: {
  apps: App[];
  onSelect(app: App): void;
  emptyMessage?: string;
  isLoading?: boolean;
  skeletonCount?: number;
  onRetry?: () => void;
  errorMessage?: string | null;
  isRefreshing?: boolean;
}) {
  if (isLoading || isRefreshing) {
    return <AppsGridSkeleton count={skeletonCount} viewMode="grid" />;
  }

  // Show error banner even if we have cached apps
  const hasError = Boolean(errorMessage);

  if (apps.length === 0) {
    const displayMessage = errorMessage || emptyMessage || 'No scenarios available.';
    return (
      <EmptyState
        message={displayMessage}
        onRetry={onRetry}
        retryLabel="Try loading scenarios again"
        isError={hasError}
      />
    );
  }

  return (
    <>
      {hasError && (
        <div className="tab-switcher__error-banner" role="alert">
          <p>{errorMessage}</p>
          {onRetry && (
            <button
              type="button"
              className="tab-switcher__error-retry"
              onClick={onRetry}
              aria-label="Retry loading"
            >
              <RefreshCw size={16} aria-hidden />
              <span>Retry</span>
            </button>
          )}
        </div>
      )}
      <div className="tab-switcher__grid">
        {apps.map(app => (
          <AppTabCard key={app.id} app={app} onSelect={onSelect} />
        ))}
      </div>
    </>
  );
}

function AppTabCard({ app, onSelect }: { app: App; onSelect(app: App): void }) {
  const identifier = useMemo(() => {
    const resolved = resolveAppIdentifier(app) ?? app.id;
    if (!resolved) {
      return null;
    }
    const trimmed = resolved.trim();
    return trimmed.length > 0 ? trimmed : null;
  }, [app]);
  const screenshotSelector = useMemo(() => selectScreenshotBySurface('app', identifier), [identifier]);
  const screenshot = useSurfaceMediaStore(screenshotSelector);
  const thumbStyle = useMemo<CSSProperties | undefined>(() => (
    screenshot
      ? {
          backgroundImage: `url(${screenshot.dataUrl})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          backgroundRepeat: 'no-repeat',
        }
      : undefined
  ), [screenshot]);
  const hasScreenshot = Boolean(screenshot);
  const viewCountLabel = formatViewCount(app.view_count);
  const statusClassName = useMemo(() => {
    const source = (app.status ?? 'unknown').toLowerCase();
    const normalized = source.replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
    return `status-${normalized || 'unknown'}`;
  }, [app.status]);
  const statusLabel = app.status ?? 'Unknown';
  const fallbackInitial = useMemo(() => {
    const base = (app.scenario_name ?? app.name ?? app.id ?? '').trim();
    if (!base) {
      return '?';
    }
    const alphanumeric = base.replace(/[^a-zA-Z0-9]/g, '');
    const candidate = alphanumeric.charAt(0) || base.charAt(0);
    return candidate.toUpperCase();
  }, [app.id, app.name, app.scenario_name]);

  return (
    <button
      type="button"
      className="tab-card"
      onClick={() => onSelect(app)}
    >
      <div
        className={clsx(
          'tab-card__thumb',
          'tab-card__thumb--scenario',
          hasScreenshot && 'tab-card__thumb--image',
        )}
        aria-hidden
        style={thumbStyle}
        title={screenshot?.note ?? undefined}
      >
        {(app.status || viewCountLabel) && (
          <div className="tab-card__thumb-overlay">
            {app.status && (
              <span className="tab-card__status-indicator">
                <span
                  className={clsx('tab-card__status-dot', statusClassName)}
                  aria-hidden
                />
                <span className="visually-hidden">Status: {statusLabel}</span>
              </span>
            )}
            {viewCountLabel && (
              <span
                className="tab-card__chip tab-card__chip--muted tab-card__chip--views"
                aria-label={`Views: ${viewCountLabel}`}
              >
                <Eye size={14} aria-hidden />
                <span className="tab-card__views-count" aria-hidden>{viewCountLabel}</span>
              </span>
            )}
          </div>
        )}
        {!hasScreenshot && (
          <div className="tab-card__thumb-fallback" aria-hidden>
            <span className="tab-card__thumb-placeholder">{fallbackInitial}</span>
          </div>
        )}
      </div>
      <div className="tab-card__body">
        <h4>{app.scenario_name ?? app.name ?? app.id}</h4>
      </div>
    </button>
  );
}

function ResourceGrid({
  resources,
  onSelect,
  emptyMessage,
  isLoading,
  skeletonCount = 6,
}: {
  resources: Resource[];
  onSelect(resource: Resource): void;
  emptyMessage?: string;
  isLoading?: boolean;
  skeletonCount?: number;
}) {
  if (isLoading) {
    return <ResourcesGridSkeleton count={skeletonCount} />;
  }
  if (resources.length === 0) {
    return <EmptyState message={emptyMessage ?? 'No resources available.'} />;
  }
  return (
    <div className="tab-switcher__grid">
      {resources.map(resource => (
        <ResourceTabCard key={resource.id} resource={resource} onSelect={onSelect} />
      ))}
    </div>
  );
}

function EmptyState({
  message,
  onRetry,
  retryLabel = 'Retry loading',
  isError = false,
}: {
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
  isError?: boolean;
}) {
  return (
    <div className={clsx('tab-switcher__empty', isError && 'tab-switcher__empty--error')}>
      <p>{message}</p>
      {onRetry && (
        <button
          type="button"
          className="tab-switcher__empty-refresh"
          onClick={onRetry}
          aria-label={retryLabel}
          title={retryLabel}
        >
          <RefreshCw size={20} aria-hidden />
          <span className="visually-hidden">{retryLabel}</span>
        </button>
      )}
    </div>
  );
}

function ResourceTabCard({ resource, onSelect }: { resource: Resource; onSelect(resource: Resource): void }) {
  const screenshotSelector = useMemo(() => selectScreenshotBySurface('resource', resource.id), [resource.id]);
  const screenshot = useSurfaceMediaStore(screenshotSelector);
  const thumbStyle = useMemo<CSSProperties | undefined>(() => (
    screenshot
      ? {
          backgroundImage: `url(${screenshot.dataUrl})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          backgroundRepeat: 'no-repeat',
        }
      : undefined
  ), [screenshot]);
  const hasScreenshot = Boolean(screenshot);
  const fallbackInitial = useMemo(() => {
    const base = (resource.name ?? resource.id ?? '').toString().trim();
    if (!base) {
      return '?';
    }
    const alphanumeric = base.replace(/[^a-zA-Z0-9]/g, '');
    const candidate = alphanumeric.charAt(0) || base.charAt(0);
    return candidate.toUpperCase();
  }, [resource.id, resource.name]);

  return (
    <button
      type="button"
      className="tab-card"
      onClick={() => onSelect(resource)}
    >
      <div
        className={clsx(
          'tab-card__thumb',
          'tab-card__thumb--resource',
          hasScreenshot && 'tab-card__thumb--image',
        )}
        aria-hidden
        style={thumbStyle}
        title={screenshot?.note ?? undefined}
      >
        {!hasScreenshot && (
          <div className="tab-card__thumb-fallback" aria-hidden>
            <span className="tab-card__thumb-placeholder">{fallbackInitial}</span>
          </div>
        )}
      </div>
      <div className="tab-card__body">
        <h4>{resource.name}</h4>
        <div className="tab-card__meta">
          <span className="tab-card__chip">{resource.type}</span>
          <span className={clsx('tab-card__status', `status-${(resource.status ?? 'unknown').toLowerCase()}`)}>
            {resource.status?.toUpperCase() ?? 'UNKNOWN'}
          </span>
        </div>
      </div>
    </button>
  );
}

function WebTabGrid({
  tabs,
  onOpen,
  onClose,
}: {
  tabs: BrowserTabRecord[];
  onOpen(tab: BrowserTabRecord): void;
  onClose(tab: BrowserTabRecord): void;
}) {
  return (
    <div className="tab-switcher__grid">
      {tabs.map(tab => (
        <WebTabCard key={tab.id} tab={tab} onOpen={onOpen} onClose={onClose} />
      ))}
    </div>
  );
}

function WebTabCard({
  tab,
  onOpen,
  onClose,
}: {
  tab: BrowserTabRecord;
  onOpen(tab: BrowserTabRecord): void;
  onClose(tab: BrowserTabRecord): void;
}) {
  const screenshotSelector = useMemo(() => selectScreenshotBySurface('web', tab.id), [tab.id]);
  const storedScreenshot = useSurfaceMediaStore(screenshotSelector);
  const previewImage = storedScreenshot?.dataUrl ?? tab.screenshotData ?? null;
  const hasPreviewImage = Boolean(previewImage);
  const faviconUrl = tab.faviconUrl && tab.faviconUrl.trim().length > 0 ? tab.faviconUrl.trim() : null;
  const fallbackInitial = useMemo(() => {
    const hostname = safeHostname(tab.url);
    if (!hostname) {
      const candidate = tab.title?.trim();
      return candidate && candidate.length > 0 ? candidate.slice(0, 1).toUpperCase() : null;
    }
    const alpha = hostname.replace(/[^a-zA-Z]/g, '');
    if (!alpha) {
      return hostname.slice(0, 1).toUpperCase();
    }
    return alpha.slice(0, 1).toUpperCase();
  }, [tab.title, tab.url]);

  return (
    <div className="tab-card tab-card--web">
      <button
        type="button"
        className={clsx(
          'tab-card__thumb',
          'tab-card__thumb--web',
          hasPreviewImage && 'tab-card__thumb--image',
        )}
        onClick={() => onOpen(tab)}
        aria-label={`Open ${tab.title}`}
        title={storedScreenshot?.note ?? undefined}
      >
        {hasPreviewImage ? (
          <img src={previewImage ?? undefined} alt="" className="tab-card__thumb-media" />
        ) : (
          <div className="tab-card__thumb-fallback" aria-hidden>
            {faviconUrl ? (
              <img src={faviconUrl} alt="" className="tab-card__thumb-favicon" />
            ) : null}
            {fallbackInitial ? (
              <span className="tab-card__thumb-placeholder">{fallbackInitial}</span>
            ) : null}
          </div>
        )}
      </button>
      <div className="tab-card__body">
        <div className="tab-card__heading">
          <h4>{tab.title}</h4>
          <button
            type="button"
            className="tab-card__icon-btn"
            onClick={() => onClose(tab)}
            aria-label={`Close ${tab.title}`}
          >
            <Trash2 size={14} aria-hidden />
          </button>
        </div>
        <div className="tab-card__meta tab-card__meta--spread">
          <span className="tab-card__chip tab-card__chip--muted">{safeHostname(tab.url)}</span>
          <button
            type="button"
            className="tab-card__icon-btn"
            onClick={() => onOpen(tab)}
            aria-label={`Open ${tab.title} in new tab`}
          >
            <ExternalLink size={16} aria-hidden />
          </button>
        </div>
      </div>
    </div>
  );
}

function WebHistoryList({
  entries,
  onReopen,
}: {
  entries: BrowserTabHistoryRecord[];
  onReopen(entry: BrowserTabHistoryRecord): void;
}) {
  if (entries.length === 0) {
    return <EmptyState message="No history yet." />;
  }
  return (
    <div className="tab-switcher__history">
      {entries.slice(0, 15).map(entry => (
        <button
          key={`${entry.id}-${entry.closedAt}`}
          type="button"
          className="tab-switcher__history-row"
          onClick={() => onReopen(entry)}
        >
          <div className="tab-switcher__history-title">
            <span>{entry.title}</span>
            <span className="tab-switcher__history-url">{entry.url}</span>
          </div>
          <span className="tab-switcher__history-time">
            {new Date(entry.closedAt).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}
          </span>
        </button>
      ))}
    </div>
  );
}

const safeHostname = (value: string) => {
  try {
    return new URL(value).hostname;
  } catch (error) {
    return value;
  }
};

function parseWebTabInput(value: string): { url: string; title: string } | null {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  const tryParse = (candidate: string) => {
    try {
      const normalized = new URL(candidate);
      return normalized.toString();
    } catch (error) {
      return null;
    }
  };

  const normalized = tryParse(trimmed)
    ?? (!trimmed.includes('://') ? tryParse(`https://${trimmed}`) : null);

  if (!normalized) {
    return null;
  }

  const hostname = safeHostname(normalized);
  return {
    url: normalized,
    title: hostname && hostname !== normalized ? hostname : normalized,
  };
}
const APP_SORT_OPTIONS: Array<{ value: AppSortOption; label: string }> = [
  { value: 'status', label: 'Active first' },
  { value: 'recently-viewed', label: 'Recently viewed' },
  { value: 'least-recently-viewed', label: 'Least recently viewed' },
  { value: 'recently-updated', label: 'Recently updated' },
  { value: 'recently-added', label: 'Recently added' },
  { value: 'most-viewed', label: 'Most viewed' },
  { value: 'least-viewed', label: 'Least viewed' },
  { value: 'name-asc', label: 'A → Z' },
  { value: 'name-desc', label: 'Z → A' },
];

const RESOURCE_SORT_OPTIONS: Array<{ value: ResourceSortOption; label: string }> = [
  { value: 'status', label: 'Status (Active first)' },
  { value: 'name-asc', label: 'Name A → Z' },
  { value: 'name-desc', label: 'Name Z → A' },
  { value: 'type', label: 'Type' },
];
