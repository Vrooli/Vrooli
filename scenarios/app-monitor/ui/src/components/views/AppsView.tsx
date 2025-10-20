import React, { useState, useEffect, useCallback, useMemo, memo, useRef } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import clsx from 'clsx';
import { Search as SearchIcon, SlidersHorizontal, X } from 'lucide-react';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { App, AppViewMode } from '@/types';
import AppCard, { HealthIndicator } from '../AppCard';
import { orderedPortMetrics } from '@/utils/appPreview';
import AppModal from '../AppModal';
import { AppsGridSkeleton } from '../LoadingSkeleton';
import { useAppsStore } from '@/state/appsStore';
import {
  APP_SORT_OPTIONS,
  APP_SORT_OPTION_SET,
  DEFAULT_APP_SORT,
  filterAndSortApps,
  type AppSortOption,
} from '@/utils/appCollections';
import './AppsView.css';

// Memoized search input component with iconography and clear affordance
const SearchInput = memo(({ 
  value, 
  onChange,
  onClear,
  inputRef,
}: { 
  value: string; 
  onChange: (value: string) => void;
  onClear: () => void;
  inputRef: React.RefObject<HTMLInputElement>;
}) => (
  <div className="search-control">
    <SearchIcon className="search-icon" aria-hidden="true" />
    <input
      ref={inputRef}
      type="text"
      className="search-box"
      placeholder="Search applications"
      value={value}
      onChange={(event) => onChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === 'Escape' && value) {
          event.preventDefault();
          onClear();
        }
      }}
      aria-label="Search applications"
    />
    {value && (
      <button
        type="button"
        className="search-clear"
        onClick={onClear}
        aria-label="Clear search"
      >
        <X size={14} aria-hidden="true" />
      </button>
    )}
  </div>
));
SearchInput.displayName = 'SearchInput';

const SortSelect = memo(({ value, onChange }: { value: AppSortOption; onChange: (value: AppSortOption) => void }) => (
  <div className="sort-control">
    <SlidersHorizontal className="sort-icon" aria-hidden="true" />
    <select
      className="sort-select"
      value={value}
      onChange={(event) => onChange(event.target.value as AppSortOption)}
      aria-label="Sort applications"
    >
      {APP_SORT_OPTIONS.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  </div>
));
SortSelect.displayName = 'SortSelect';

// Memoized view toggle button
const ViewToggle = memo(({ viewMode, onClick }: { viewMode: AppViewMode; onClick: () => void }) => (
  <button
    className="control-btn view-toggle"
    onClick={onClick}
    title="Toggle View"
  >
    {viewMode === 'grid' ? '☰' : '⊞'}
  </button>
));
ViewToggle.displayName = 'ViewToggle';

const formatStatusText = (status?: string, isPartial?: boolean) => {
  if (isPartial) {
    return 'SYNCING';
  }

  const normalized = status?.trim();
  if (!normalized) {
    return 'UNKNOWN';
  }

  return normalized.replace(/_/g, ' ').toUpperCase();
};

const runningStates = new Set(['running', 'healthy', 'degraded', 'unhealthy']);

const GRID_INITIAL_BATCH = 24;
const GRID_BATCH_SIZE = 24;

const AppListRow = memo(({ app, onPreview, onDetails, onAppAction }: {
  app: App;
  onPreview: (app: App) => void;
  onDetails: (app: App) => void;
  onAppAction: (appId: string, action: 'start' | 'stop') => void;
}) => {
  const isPartial = Boolean(app.is_partial);
  const normalizedStatus = (app.status ?? 'unknown').toLowerCase();
  const statusText = useMemo(() => formatStatusText(app.status, isPartial), [app.status, isPartial]);

  const uptime = useMemo(() => {
    if (isPartial) {
      return 'SYNCING';
    }

    if (app.uptime && app.uptime !== 'N/A') {
      return app.uptime;
    }

    if (!runningStates.has(normalizedStatus) || !app.created_at) {
      return 'N/A';
    }

    const start = new Date(app.created_at);
    if (Number.isNaN(start.getTime())) {
      return 'N/A';
    }

    const diff = Date.now() - start.getTime();
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }, [app.created_at, app.uptime, isPartial, normalizedStatus]);

  const ports = useMemo(() => orderedPortMetrics(app).slice(0, 3), [app]);
  const isRunning = runningStates.has(normalizedStatus);

  const viewCount = useMemo(() => {
    if (typeof app.view_count === 'number') {
      return app.view_count;
    }
    if (typeof app.view_count === 'string') {
      const parsed = Number(app.view_count);
      return Number.isFinite(parsed) ? parsed : null;
    }
    return null;
  }, [app.view_count]);

  const lastViewedLabel = useMemo(() => {
    const { last_viewed_at: lastViewedAt } = app;
    if (!lastViewedAt) {
      return null;
    }

    const timestamp = new Date(lastViewedAt);
    const time = timestamp.getTime();
    if (Number.isNaN(time)) {
      return null;
    }

    const diffMs = Date.now() - time;
    if (diffMs < 0) {
      return 'just now';
    }

    const seconds = Math.floor(diffMs / 1000);
    if (seconds < 60) {
      return 'just now';
    }
    if (seconds < 3600) {
      return `${Math.floor(seconds / 60)}m ago`;
    }
    if (seconds < 86400) {
      return `${Math.floor(seconds / 3600)}h ago`;
    }
    if (seconds < 604800) {
      return `${Math.floor(seconds / 86400)}d ago`;
    }

    return timestamp.toLocaleDateString();
  }, [app]);

  const viewMeta = useMemo(() => {
    const parts: string[] = [];
    if (viewCount !== null) {
      parts.push(`Views: ${viewCount.toLocaleString()}`);
    }
    if (lastViewedLabel) {
      parts.push(`Last viewed ${lastViewedLabel}`);
    }

    if (parts.length === 0) {
      return null;
    }

    return parts.join(' · ');
  }, [lastViewedLabel, viewCount]);

  const handleRowClick = useCallback(() => {
    onPreview(app);
  }, [app, onPreview]);

  const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onPreview(app);
    }
  }, [app, onPreview]);

  const handleStart = useCallback((event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onAppAction(app.id, 'start');
  }, [app.id, onAppAction]);

  const handleStop = useCallback((event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onAppAction(app.id, 'stop');
  }, [app.id, onAppAction]);

  const handleDetails = useCallback((event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onDetails(app);
  }, [app, onDetails]);

  return (
    <tr
      className={clsx('app-row', { partial: isPartial })}
      onClick={handleRowClick}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="button"
      aria-label={`Open ${app.name} preview`}
    >
      <td className="app-name-cell">
        <div className="app-name-details">
          <span className="app-name-text">{app.name}</span>
          {app.scenario_name && (
            <span className="app-name-subtext">{app.scenario_name}</span>
          )}
          {viewMeta && (
            <span className="app-name-subtext">{viewMeta}</span>
          )}
          {app.description && (
            <span className="app-name-description">{app.description}</span>
          )}
        </div>
      </td>
      <td className="app-status-cell">
        <HealthIndicator status={app.status} isPartial={isPartial} />
        <span className="app-status-text">{statusText}</span>
      </td>
      <td className="app-uptime">{uptime}</td>
      <td className="app-ports">
        {ports.length > 0 ? (
          ports.map(({ label, value }) => (
            <span key={label} className="port-tag">{label}: {value}</span>
          ))
        ) : (
          <span className="app-ports--empty">—</span>
        )}
      </td>
      <td className="actions-cell">
        {isRunning ? (
          <button type="button" className="app-btn small" onClick={handleStop}>
            Stop
          </button>
        ) : (
          <button type="button" className="app-btn small" onClick={handleStart}>
            Start
          </button>
        )}
        <button type="button" className="app-btn small" onClick={handleDetails}>
          Details
        </button>
      </td>
    </tr>
  );
});
AppListRow.displayName = 'AppListRow';

// Main AppsView component with optimizations
const AppsView = memo(() => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const loadingDetailed = useAppsStore(state => state.loadingDetailed);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const storeError = useAppsStore(state => state.error);
  const clearError = useAppsStore(state => state.clearError);
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [search, setSearch] = useState(() => searchParams.get('q') ?? '');
  const initialSortParam = searchParams.get('sort');
  const [sortOption, setSortOption] = useState<AppSortOption>(
    initialSortParam && APP_SORT_OPTION_SET.has(initialSortParam as AppSortOption)
      ? (initialSortParam as AppSortOption)
      : DEFAULT_APP_SORT,
  );
  const [viewMode, setViewMode] = useState<AppViewMode>('grid');
  const [selectedApp, setSelectedApp] = useState<App | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [visibleCount, setVisibleCount] = useState(GRID_INITIAL_BATCH);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const emptyReloadRef = useRef(false);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const prevFiltersRef = useRef({ search, sortOption, viewMode });
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const recordNavigateEvent = useCallback((info: Record<string, unknown>) => {
    try {
      const payload = {
        event: 'navigate-event',
        timestamp: Date.now(),
        detail: {
          pathname: location.pathname,
          search: location.search,
          ...info,
        },
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
      };
      const body = JSON.stringify(payload);
      if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
        const blob = new Blob([body], { type: 'application/json' });
        navigator.sendBeacon('/__debug/client-event', blob);
      } else {
        void fetch('/__debug/client-event', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body,
          keepalive: true,
        });
      }
    } catch (error) {
      // best-effort debug logging
    }
  }, [location.pathname, location.search]);

  const loading = loadingInitial && apps.length === 0;
  const hydrating = loadingDetailed;

  useEffect(() => {
    setErrorMessage(storeError ?? null);
  }, [storeError]);

  useEffect(() => () => {
    clearError();
  }, [clearError]);

  useEffect(() => () => {
    if (observerRef.current) {
      observerRef.current.disconnect();
      observerRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (viewMode !== 'grid' && observerRef.current) {
      observerRef.current.disconnect();
      observerRef.current = null;
    }
  }, [viewMode]);

  useEffect(() => {
    if (hasInitialized && !loadingInitial && apps.length === 0) {
      if (!emptyReloadRef.current) {
        emptyReloadRef.current = true;
        void loadApps({ force: true });
      }
    } else if (apps.length > 0) {
      emptyReloadRef.current = false;
    }
  }, [apps.length, hasInitialized, loadApps, loadingInitial]);

  useEffect(() => {
    const nextSearch = searchParams.get('q') ?? '';
    if (nextSearch !== search) {
      setSearch(nextSearch);
    }
  }, [searchParams, search]);

  const updateSearchParam = useCallback((key: 'q' | 'sort', value: string | null) => {
    const next = new URLSearchParams(searchParams);
    if (value && value.length > 0) {
      next.set(key, value);
    } else {
      next.delete(key);
    }
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  useEffect(() => {
    const param = searchParams.get('sort');
    if (param && APP_SORT_OPTION_SET.has(param as AppSortOption)) {
      const normalized = param as AppSortOption;
      if (normalized !== sortOption) {
        setSortOption(normalized);
      }
      return;
    }

    if (param && !APP_SORT_OPTION_SET.has(param as AppSortOption)) {
      if (sortOption !== DEFAULT_APP_SORT) {
        setSortOption(DEFAULT_APP_SORT);
      }
      updateSearchParam('sort', null);
      return;
    }

    if (!param && sortOption !== DEFAULT_APP_SORT) {
      setSortOption(DEFAULT_APP_SORT);
    }
  }, [searchParams, sortOption, updateSearchParam]);

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  const filteredApps = useMemo(
    () => filterAndSortApps(apps, { search, sort: sortOption }),
    [apps, search, sortOption],
  );

  useEffect(() => {
    const previous = prevFiltersRef.current;
    const filtersChanged = previous.search !== search || previous.sortOption !== sortOption;
    const viewChanged = previous.viewMode !== viewMode;

    if (viewMode === 'grid') {
      if (filtersChanged || viewChanged) {
        setVisibleCount(Math.min(filteredApps.length, GRID_INITIAL_BATCH));
      } else if (filteredApps.length < visibleCount) {
        setVisibleCount(filteredApps.length);
      }
    } else if (visibleCount !== filteredApps.length) {
      setVisibleCount(filteredApps.length);
    }

    prevFiltersRef.current = { search, sortOption, viewMode };
  }, [filteredApps.length, search, sortOption, viewMode, visibleCount]);

  const handleAppAction = useCallback(async (appId: string, action: 'start' | 'stop' | 'restart') => {
    logger.info(`App action: ${action} for ${appId}`);

    try {
      const success = await appService.controlApp(appId, action);
      if (success) {
        setAppsState(prev => prev.map(app => {
          if (app.id === appId) {
            return {
              ...app,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString()
            };
          }
          return app;
        }));
        logger.info(`App ${appId} ${action} successful`);
      }
    } catch (error) {
      logger.error(`Failed to ${action} app ${appId}`, error);
    }
  }, [setAppsState]);

  const handleAppDetails = useCallback((app: App) => {
    setSelectedApp(app);
    setModalOpen(true);
  }, []);

  const handleAppPreview = useCallback((app: App) => {
    const currentSearch = searchParams.toString();
    recordNavigateEvent({
      source: 'apps-grid',
      action: 'preview',
      appId: app.id,
      targetPath: `/apps/${encodeURIComponent(app.id)}/preview`,
      targetSearch: currentSearch ? `?${currentSearch}` : undefined,
    });
    navigate({
      pathname: `/apps/${encodeURIComponent(app.id)}/preview`,
      search: currentSearch ? `?${currentSearch}` : undefined,
    }, {
      state: {
        fromAppsList: true,
        originAppId: app.id,
        navTimestamp: Date.now(),
      },
    });
  }, [navigate, recordNavigateEvent, searchParams]);

  const handleViewLogs = useCallback((appId: string) => {
    setModalOpen(false);
    setSelectedApp(null);
    recordNavigateEvent({
      source: 'apps-grid',
      action: 'view-logs',
      appId,
      targetPath: `/logs/${appId}`,
    });
    navigate(`/logs/${appId}`);
  }, [navigate, recordNavigateEvent]);

  const toggleViewMode = useCallback(() => {
    setViewMode(prev => prev === 'grid' ? 'list' : 'grid');
  }, []);

  const handleSearchChange = useCallback((value: string) => {
    setSearch(value);
    updateSearchParam('q', value ? value : null);
  }, [updateSearchParam]);

  const handleSearchClear = useCallback(() => {
    setSearch('');
    updateSearchParam('q', null);
    if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(() => {
        searchInputRef.current?.focus();
      });
    } else {
      searchInputRef.current?.focus();
    }
  }, [updateSearchParam]);

  const handleSortChange = useCallback((value: AppSortOption) => {
    setSortOption(value);
    updateSearchParam('sort', value === DEFAULT_APP_SORT ? null : value);
  }, [updateSearchParam]);

  const isGridView = viewMode === 'grid';
  const gridHasMore = isGridView && visibleCount < filteredApps.length;
  const displayedApps = isGridView ? filteredApps.slice(0, visibleCount) : filteredApps;

  const loadMoreRef = useCallback((node: HTMLDivElement | null) => {
    if (observerRef.current) {
      observerRef.current.disconnect();
      observerRef.current = null;
    }

    if (!node || !isGridView || !gridHasMore) {
      return;
    }

    observerRef.current = new IntersectionObserver((entries) => {
      const [entry] = entries;
      if (entry.isIntersecting) {
        setVisibleCount(prev => {
          if (prev >= filteredApps.length) {
            return prev;
          }
          return Math.min(filteredApps.length, prev + GRID_BATCH_SIZE);
        });
      }
    }, { root: null, rootMargin: '200px 0px', threshold: 0.1 });

    observerRef.current.observe(node);
  }, [filteredApps.length, gridHasMore, isGridView]);

  const handleErrorDismiss = useCallback(() => {
    setErrorMessage(null);
    clearError();
  }, [clearError]);

  return (
    <div className="apps-view">
      <div className="panel-header">
        <div className="panel-title-spacer" aria-hidden="true" />
        <div className="panel-controls">
          <SearchInput
            value={search}
            onChange={handleSearchChange}
            onClear={handleSearchClear}
            inputRef={searchInputRef}
          />
          <SortSelect value={sortOption} onChange={handleSortChange} />
          <ViewToggle viewMode={viewMode} onClick={toggleViewMode} />
          {hydrating && !loading && (
            <span className="control-status" title="Fetching live orchestrator data">
              SYNCING…
            </span>
          )}
        </div>
      </div>

      {errorMessage && (
        <div className="app-notice error" role="alert">
          <span className="app-notice-message">{errorMessage}</span>
          <button
            type="button"
            className="app-notice-dismiss"
            onClick={handleErrorDismiss}
            aria-label="Dismiss error message"
          >
            <X size={16} aria-hidden />
          </button>
        </div>
      )}

      {loading && filteredApps.length === 0 ? (
        <AppsGridSkeleton count={6} viewMode={viewMode} />
      ) : filteredApps.length === 0 ? (
        <div className="empty-state">
          <p>No applications found.</p>
          <p className="hint">
            {search ? 'Try adjusting your search.' : 'Launch an application to see it appear here.'}
          </p>
        </div>
      ) : viewMode === 'list' ? (
        <div className="apps-list">
          <table className="apps-table">
            <thead>
              <tr>
                <th scope="col">Application</th>
                <th scope="col">Status</th>
                <th scope="col">Uptime</th>
                <th scope="col">Ports</th>
                <th scope="col" className="actions-header">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredApps.map(app => (
                <AppListRow
                  key={app.id}
                  app={app}
                  onPreview={handleAppPreview}
                  onDetails={handleAppDetails}
                  onAppAction={handleAppAction}
                />
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <>
          <div className={clsx('apps-grid', viewMode)}>
            {displayedApps.map(app => (
              <AppCard
                key={app.id}
                app={app}
                viewMode={viewMode}
                onCardClick={handleAppPreview}
                onDetails={handleAppDetails}
                onStart={(id) => handleAppAction(id, 'start')}
                onStop={(id) => handleAppAction(id, 'stop')}
              />
            ))}
          </div>
          {gridHasMore && (
            <div className="apps-grid__sentinel" ref={loadMoreRef} aria-hidden="true" />
          )}
        </>
      )}

      {modalOpen && selectedApp && (
        <AppModal
          app={selectedApp}
          isOpen={modalOpen}
          onClose={() => {
            setModalOpen(false);
            setSelectedApp(null);
          }}
          onAction={handleAppAction}
          onViewLogs={handleViewLogs}
        />
      )}
    </div>
  );
});

AppsView.displayName = 'AppsView';

export default AppsView;
