import React, { useState, useEffect, useCallback, useMemo, memo, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import clsx from 'clsx';
import { Search as SearchIcon, SlidersHorizontal, X } from 'lucide-react';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { App, AppViewMode } from '@/types';
import AppCard, { HealthIndicator, orderedPortMetrics } from '../AppCard';
import AppModal from '../AppModal';
import { AppsGridSkeleton } from '../LoadingSkeleton';
import './AppsView.css';

interface AppsViewProps {
  apps: App[];
  setApps: React.Dispatch<React.SetStateAction<App[]>>;
}

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

type SortOption = 'status' | 'name-asc' | 'name-desc';

const SORT_OPTIONS: Array<{ value: SortOption; label: string }> = [
  { value: 'status', label: 'Status · active first' },
  { value: 'name-asc', label: 'Name · A → Z' },
  { value: 'name-desc', label: 'Name · Z → A' },
];

const SORT_OPTION_SET = new Set<SortOption>(SORT_OPTIONS.map(({ value }) => value));
const DEFAULT_SORT: SortOption = 'status';

const STATUS_PRIORITY: Record<string, number> = {
  running: 0,
  healthy: 0,
  starting: 0,
  booting: 0,
  initializing: 0,
  unknown: 0,
  degraded: 1,
  unhealthy: 1,
  warning: 1,
  syncing: 1,
  partial: 1,
  stopping: 2,
  paused: 2,
  error: 2,
  failed: 2,
  crashed: 2,
  offline: 4,
  stopped: 4,
};

const SortSelect = memo(({ value, onChange }: { value: SortOption; onChange: (value: SortOption) => void }) => (
  <div className="sort-control">
    <SlidersHorizontal className="sort-icon" aria-hidden="true" />
    <select
      className="sort-select"
      value={value}
      onChange={(event) => onChange(event.target.value as SortOption)}
      aria-label="Sort applications"
    >
      {SORT_OPTIONS.map((option) => (
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

// Virtual scrolling component for large lists
const VirtualAppList = memo(({ 
  apps, 
  viewMode, 
  onPreview, 
  onDetails,
  onAppAction,
}: {
  apps: App[];
  viewMode: AppViewMode;
  onPreview: (app: App) => void;
  onDetails: (app: App) => void;
  onAppAction: (appId: string, action: 'start' | 'stop') => void;
}) => {
  // For lists with many items, implement virtual scrolling
  const ITEMS_PER_PAGE = 50;
  const [visibleRange, setVisibleRange] = useState({ start: 0, end: ITEMS_PER_PAGE });
  
  useEffect(() => {
    const handleScroll = () => {
      const scrollTop = window.scrollY;
      const itemHeight = viewMode === 'grid' ? 200 : 100; // Approximate heights
      const start = Math.floor(scrollTop / itemHeight);
      const end = start + Math.ceil(window.innerHeight / itemHeight) + 10; // Buffer
      
      setVisibleRange({ start, end });
    };
    
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [viewMode]);
  
  // Only render visible items for performance
  const visibleApps = apps.slice(visibleRange.start, visibleRange.end);
  
  return (
    <div className={clsx('apps-grid', viewMode)}>
      {/* Spacer for virtual scrolling */}
      {visibleRange.start > 0 && (
        <div style={{ height: `${visibleRange.start * (viewMode === 'grid' ? 200 : 100)}px` }} />
      )}
      
      {visibleApps.map(app => (
        <AppCard
          key={app.id}
          app={app}
          viewMode={viewMode}
          onCardClick={onPreview}
          onDetails={onDetails}
          onStart={(id) => onAppAction(id, 'start')}
          onStop={(id) => onAppAction(id, 'stop')}
        />
      ))}
      
      {/* Spacer for virtual scrolling */}
      {visibleRange.end < apps.length && (
        <div style={{ height: `${(apps.length - visibleRange.end) * (viewMode === 'grid' ? 200 : 100)}px` }} />
      )}
    </div>
  );
});
VirtualAppList.displayName = 'VirtualAppList';

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
const AppsView = memo<AppsViewProps>(({ apps, setApps }) => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [hydrating, setHydrating] = useState(false);
  const [search, setSearch] = useState(() => searchParams.get('q') ?? '');
  const initialSortParam = searchParams.get('sort');
  const [sortOption, setSortOption] = useState<SortOption>(
    initialSortParam && SORT_OPTION_SET.has(initialSortParam as SortOption)
      ? (initialSortParam as SortOption)
      : DEFAULT_SORT,
  );
  const [viewMode, setViewMode] = useState<AppViewMode>('grid');
  const [selectedApp, setSelectedApp] = useState<App | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const isMountedRef = useRef(true);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const initialPositionsRef = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    const nextSearch = searchParams.get('q') ?? '';
    if (nextSearch !== search) {
      setSearch(nextSearch);
    }
  }, [searchParams, search]);

  useEffect(() => {
    apps.forEach((app, index) => {
      if (!initialPositionsRef.current.has(app.id)) {
        initialPositionsRef.current.set(app.id, index);
      }
    });
  }, [apps]);

  const mergeAppData = useCallback((existing: App[], updates: App[]) => {
    if (updates.length === 0) {
      return existing;
    }

    const updateMap = new Map<string, App>();
    updates.forEach((app) => updateMap.set(app.id, app));

    const merged = existing.map((app) => {
      const update = updateMap.get(app.id);
      if (!update) {
        return app;
      }

      updateMap.delete(app.id);

      const mergedConfig = {
        ...(app.config || {}),
        ...(update.config || {}),
      };

      const mergedPorts = Object.keys(update.port_mappings || {}).length > 0
        ? update.port_mappings
        : app.port_mappings;

      return {
        ...app,
        ...update,
        description: update.description || app.description,
        tags: update.tags && update.tags.length > 0 ? update.tags : app.tags,
        uptime: update.uptime || app.uptime,
        config: mergedConfig,
        port_mappings: mergedPorts,
        is_partial: Boolean(update.is_partial),
      };
    });

    updateMap.forEach((app) => {
      merged.push({
        ...app,
        is_partial: Boolean(app.is_partial),
      });
    });

    return merged;
  }, []);

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
    if (param && SORT_OPTION_SET.has(param as SortOption)) {
      const normalized = param as SortOption;
      if (normalized !== sortOption) {
        setSortOption(normalized);
      }
      return;
    }

    if (param && !SORT_OPTION_SET.has(param as SortOption)) {
      if (sortOption !== DEFAULT_SORT) {
        setSortOption(DEFAULT_SORT);
      }
      updateSearchParam('sort', null);
      return;
    }

    if (!param && sortOption !== DEFAULT_SORT) {
      setSortOption(DEFAULT_SORT);
    }
  }, [searchParams, sortOption, updateSearchParam]);

  const fetchApps = useCallback(async () => {
    setLoading(true);
    logger.time('fetchApps.summary');

    try {
      const summaries = await appService.getAppSummaries();
      logger.info(`Fetched ${summaries.length} app summaries`);
      if (isMountedRef.current) {
        setApps((prev) => (summaries.length > 0 ? summaries : prev));
      }
    } catch (error) {
      logger.error('Failed to fetch app summaries', error);
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
      }
      logger.timeEnd('fetchApps.summary');
    }

    setHydrating(true);
    logger.time('fetchApps.detail');

    try {
      const detailedApps = await appService.getApps();
      logger.info(`Hydrated ${detailedApps.length} apps with orchestrator data`);
      if (isMountedRef.current && detailedApps.length > 0) {
        setApps((prev) => mergeAppData(prev, detailedApps));
      }
    } catch (error) {
      logger.error('Failed to hydrate apps with orchestrator data', error);
    } finally {
      if (isMountedRef.current) {
        setHydrating(false);
      }
      logger.timeEnd('fetchApps.detail');
    }
  }, [mergeAppData, setApps]);

  useEffect(() => {
    fetchApps();
  }, [fetchApps]);

  const filteredApps = useMemo(() => {
    const normalizedSearch = search.trim().toLowerCase();

    const matchesSearch = (app: App) => {
      if (!normalizedSearch) {
        return true;
      }

      const haystacks: string[] = [
        app.name,
        app.id,
        app.status ?? '',
        app.scenario_name ?? '',
        app.description ?? '',
        ...(app.tags ?? []),
      ];

      Object.entries(app.port_mappings ?? {}).forEach(([key, mapping]) => {
        haystacks.push(key);
        if (mapping == null) {
          return;
        }
        if (typeof mapping === 'string' || typeof mapping === 'number') {
          haystacks.push(String(mapping));
          return;
        }

        if (typeof mapping === 'object') {
          const record = mapping as Record<string, unknown>;
          ['label', 'name', 'value', 'port', 'description', 'url'].forEach((field) => {
            const candidate = record[field];
            if (typeof candidate === 'string' || typeof candidate === 'number') {
              haystacks.push(String(candidate));
            }
          });
        }
      });

      return haystacks.some((item) => item && item.toLowerCase().includes(normalizedSearch));
    };

    const baseList = apps.filter(matchesSearch);

    const compareByInitialPosition = (a: App, b: App) => {
      const positionA = initialPositionsRef.current.get(a.id) ?? 0;
      const positionB = initialPositionsRef.current.get(b.id) ?? 0;
      if (positionA !== positionB) {
        return positionA - positionB;
      }
      return a.name.localeCompare(b.name);
    };

    const compareByNameAsc = (a: App, b: App) => a.name.localeCompare(b.name);
    const compareByNameDesc = (a: App, b: App) => b.name.localeCompare(a.name);

    const compareByStatus = (a: App, b: App) => {
      const statusA = (a.status ?? '').toLowerCase();
      const statusB = (b.status ?? '').toLowerCase();
      const priorityA = STATUS_PRIORITY[statusA] ?? 3;
      const priorityB = STATUS_PRIORITY[statusB] ?? 3;
      if (priorityA !== priorityB) {
        return priorityA - priorityB;
      }
      return compareByInitialPosition(a, b);
    };

    switch (sortOption) {
      case 'name-asc':
        return baseList.slice().sort((a, b) => {
          const result = compareByNameAsc(a, b);
          return result === 0 ? compareByInitialPosition(a, b) : result;
        });
      case 'name-desc':
        return baseList.slice().sort((a, b) => {
          const result = compareByNameDesc(a, b);
          return result === 0 ? compareByInitialPosition(a, b) : result;
        });
      case 'status':
      default:
        return baseList.slice().sort(compareByStatus);
    }
  }, [apps, search, sortOption]);

  const handleAppAction = useCallback(async (appId: string, action: 'start' | 'stop' | 'restart') => {
    logger.info(`App action: ${action} for ${appId}`);

    try {
      const success = await appService.controlApp(appId, action);
      if (success) {
        setApps(prev => prev.map(app => {
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
  }, [setApps]);

  const handleAppDetails = useCallback((app: App) => {
    setSelectedApp(app);
    setModalOpen(true);
  }, []);

  const handleAppPreview = useCallback((app: App) => {
    navigate(`/apps/${encodeURIComponent(app.id)}/preview`);
  }, [navigate]);

  const handleViewLogs = useCallback((appId: string) => {
    setModalOpen(false);
    setSelectedApp(null);
    navigate(`/logs/${appId}`);
  }, [navigate]);

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

  const handleSortChange = useCallback((value: SortOption) => {
    setSortOption(value);
    updateSearchParam('sort', value === DEFAULT_SORT ? null : value);
  }, [updateSearchParam]);

  const useVirtualScrolling = viewMode === 'grid' && filteredApps.length > 100;

  return (
    <div className="apps-view">
      <div className="panel-header">
        <h2>ACTIVE APPLICATIONS</h2>
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
          <button
            className="control-btn refresh"
            onClick={fetchApps}
            disabled={loading}
            title="Refresh"
          >
            {loading ? '⟳' : hydrating ? '⧗' : '⟲'}
          </button>
        </div>
      </div>

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
      ) : useVirtualScrolling ? (
        <VirtualAppList
          apps={filteredApps}
          viewMode={viewMode}
          onPreview={handleAppPreview}
          onDetails={handleAppDetails}
          onAppAction={handleAppAction}
        />
      ) : (
        <div className={clsx('apps-grid', viewMode)}>
          {filteredApps.map(app => (
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
