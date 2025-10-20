import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Layers, Server, Globe, Search, SlidersHorizontal, X, ExternalLink, Trash2 } from 'lucide-react';
import clsx from 'clsx';
import { selectScreenshotBySurface, useSurfaceMediaStore } from '@/state/surfaceMediaStore';
import { useAppCatalog, normalizeAppSort, type AppSortOption } from '@/hooks/useAppCatalog';
import { useResourcesCatalog } from '@/hooks/useResourcesCatalog';
import { useBrowserTabsStore, type BrowserTabRecord, type BrowserTabHistoryRecord } from '@/state/browserTabsStore';
import { resolveAppIdentifier } from '@/utils/appPreview';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
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
  const [searchParams, setSearchParams] = useSearchParams();
  const segmentParam = searchParams.get(SEGMENT_QUERY_KEY);
  const { closeOverlay } = useOverlayRouter();
  const [search, setSearch] = useState('');
  const [activeSegment, setActiveSegment] = useState<SegmentId>(() => resolveSegment(segmentParam));
  const [sortOption, setSortOption] = useState<AppSortOption>('status');
  const normalizedSearch = search.trim().toLowerCase();
  const { filteredApps, recentApps } = useAppCatalog({ search, sort: sortOption, historyLimit: 12 });
  const { sortedResources } = useResourcesCatalog();
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

  useEffect(() => {
    setActiveSegment(resolveSegment(segmentParam));
  }, [segmentParam]);

  const handleSegmentSelect = (segmentId: SegmentId) => {
    setActiveSegment(segmentId);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.set(SEGMENT_QUERY_KEY, segmentId);
    setSearchParams(nextParams, { replace: true });
  };

  const handleClose = () => {
    closeOverlay({ preserve: ['segment'] });
  };

  const handleAppSelect = (app: App) => {
    const identifier = resolveAppIdentifier(app) ?? app.id;
    if (!identifier) {
      return;
    }
    closeOverlay({ replace: true });
    navigate(`/apps/${encodeURIComponent(identifier)}/preview`);
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
    if (entry.screenshotData) {
      updateBrowserTab(reopened.id, {
        screenshotData: entry.screenshotData,
        screenshotWidth: entry.screenshotWidth,
        screenshotHeight: entry.screenshotHeight,
        screenshotNote: entry.screenshotNote,
      });
      setSurfaceScreenshot('web', reopened.id, {
        dataUrl: entry.screenshotData,
        width: entry.screenshotWidth ?? 0,
        height: entry.screenshotHeight ?? 0,
        capturedAt: Date.now(),
        note: entry.screenshotNote ?? 'Restored from history',
        source: 'restored',
      });
    }
    handleWebTabOpen(reopened);
  };

  const showAppHistory = !normalizedSearch && recentApps.length > 0;
  const showResourceHistory = false;
  const showWebHistory = filteredHistoryTabs.length > 0;

  const activeSegmentLabel = SEGMENTS.find(segment => segment.id === activeSegment)?.label ?? '';

  return (
    <div className="tab-switcher">
      <header className="tab-switcher__header">
        <div>
          <h2>{activeSegmentLabel}</h2>
          <p>Quickly jump between managed surfaces.</p>
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
              >
                <Icon size={16} aria-hidden />
                <span>{segment.label}</span>
              </button>
            );
          })}
        </div>
        {activeSegment === 'apps' && (
          <div className="tab-switcher__sort">
            <SlidersHorizontal size={16} aria-hidden />
            <select
              value={sortOption}
              onChange={event => setSortOption(normalizeAppSort(event.target.value))}
              aria-label="Sort scenarios"
            >
              <option value="status">Active first</option>
              <option value="recently-viewed">Recently viewed</option>
              <option value="least-recently-viewed">Least recently viewed</option>
              <option value="recently-updated">Recently updated</option>
              <option value="recently-added">Recently added</option>
              <option value="most-viewed">Most viewed</option>
              <option value="least-viewed">Least viewed</option>
              <option value="name-asc">A → Z</option>
              <option value="name-desc">Z → A</option>
            </select>
          </div>
        )}
      </div>

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
            >
              <AppGrid apps={filteredApps} onSelect={handleAppSelect} emptyMessage="No scenarios match your search." />
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
            >
              <ResourceGrid resources={filteredResources} onSelect={handleResourceSelect} emptyMessage="No resources match your search." />
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

function Section({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <section className="tab-switcher__group">
      <header>
        <h3>{title}</h3>
        {description && <p>{description}</p>}
      </header>
      {children}
    </section>
  );
}

function AppGrid({ apps, onSelect, emptyMessage }: { apps: App[]; onSelect(app: App): void; emptyMessage?: string }) {
  if (apps.length === 0) {
    return <EmptyState message={emptyMessage ?? 'No scenarios available.'} />;
  }
  return (
    <div className="tab-switcher__grid">
      {apps.map(app => (
        <AppTabCard key={app.id} app={app} onSelect={onSelect} />
      ))}
    </div>
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
  const thumbStyle = screenshot ? { backgroundImage: `url(${screenshot.dataUrl})` } : undefined;
  const viewCountLabel = formatViewCount(app.view_count);

  return (
    <button
      type="button"
      className="tab-card"
      onClick={() => onSelect(app)}
    >
      <div
        className={clsx('tab-card__thumb', screenshot && 'tab-card__thumb--image')}
        aria-hidden
        style={thumbStyle}
        title={screenshot?.note ?? undefined}
      />
      <div className="tab-card__body">
        <h4>{app.scenario_name ?? app.name ?? app.id}</h4>
        <div className="tab-card__meta">
          {app.status && <span className="tab-card__chip">{app.status}</span>}
          {viewCountLabel && (
            <span className="tab-card__chip tab-card__chip--muted">
              👁 {viewCountLabel}
            </span>
          )}
        </div>
      </div>
    </button>
  );
}

function ResourceGrid({ resources, onSelect, emptyMessage }: { resources: Resource[]; onSelect(resource: Resource): void; emptyMessage?: string }) {
  if (resources.length === 0) {
    return <EmptyState message={emptyMessage ?? 'No resources available.'} />;
  }
  return (
    <div className="tab-switcher__grid">
      {resources.map(resource => (
        <button
          key={resource.id}
          type="button"
          className="tab-card"
          onClick={() => onSelect(resource)}
        >
          <div className="tab-card__thumb tab-card__thumb--resource" aria-hidden />
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
      ))}
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="tab-switcher__empty">
      <p>{message}</p>
    </div>
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
  const thumbStyle = previewImage ? { backgroundImage: `url(${previewImage})` } : undefined;

  return (
    <div className="tab-card tab-card--web">
      <button
        type="button"
        className={clsx(
          'tab-card__thumb',
          'tab-card__thumb--web',
          previewImage && 'tab-card__thumb--image',
        )}
        onClick={() => onOpen(tab)}
        aria-label={`Open ${tab.title}`}
        style={thumbStyle}
        title={storedScreenshot?.note ?? undefined}
      />
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
