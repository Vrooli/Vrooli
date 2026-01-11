import { useMemo, type CSSProperties, type ReactNode } from 'react';
import { ExternalLink, Eye, RefreshCw, SlidersHorizontal, Target, Trash2 } from 'lucide-react';
import clsx from 'clsx';
import { AppsGridSkeleton, ResourcesGridSkeleton } from '@/components/LoadingSkeleton';
import { selectScreenshotBySurface, useSurfaceMediaStore, type SurfaceType } from '@/state/surfaceMediaStore';
import { resolveAppIdentifier } from '@/utils/appPreview';
import type { App, Resource } from '@/types';
import type { BrowserTabHistoryRecord, BrowserTabRecord } from '@/state/browserTabsStore';
import {
  buildThumbStyle,
  formatViewCount,
  getAppDisplayName,
  getCompletenessLevel,
  getFallbackInitial,
  getStatusClassName,
  getWebTabFallbackInitial,
  safeHostname,
} from './tabSwitcherUtils';

export type SortOption<T extends string> = { value: T; label: string };

export function Section({
  title,
  description,
  actions,
  children,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
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
  options: Array<SortOption<string>>;
  onChange(value: string): void;
};

export function SortControl({ label, value, options, onChange }: SortControlProps) {
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

export function AppGrid({
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
  const isBusy = Boolean(isLoading || isRefreshing);
  if (isBusy) {
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

export function ResourceGrid({
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

export function WebTabGrid({
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

export function WebHistoryList({
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

export function EmptyState({
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

function AppTabCard({ app, onSelect }: { app: App; onSelect(app: App): void }) {
  const {
    displayName,
    screenshot,
    thumbStyle,
    hasScreenshot,
    viewCountLabel,
    completenessScore,
    hasCompletenessScore,
    completenessLevel,
    showThumbOverlay,
    hasStatus,
    statusClassName,
    statusLabel,
    fallbackInitial,
    completenessTitle,
  } = useAppCardMeta(app);

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
        <AppCardOverlay
          show={showThumbOverlay}
          hasStatus={hasStatus}
          statusLabel={statusLabel}
          statusClassName={statusClassName}
          hasCompletenessScore={hasCompletenessScore}
          completenessScore={completenessScore}
          completenessLevel={completenessLevel}
          completenessTitle={completenessTitle}
          viewCountLabel={viewCountLabel}
        />
        {!hasScreenshot && (
          <div className="tab-card__thumb-fallback" aria-hidden>
            <span className="tab-card__thumb-placeholder">{fallbackInitial}</span>
          </div>
        )}
      </div>
      <div className="tab-card__body">
        <h4>{displayName}</h4>
      </div>
    </button>
  );
}

function ResourceTabCard({ resource, onSelect }: { resource: Resource; onSelect(resource: Resource): void }) {
  const { screenshot, thumbStyle, hasScreenshot } = useSurfaceThumbnail('resource', resource.id);
  const fallbackInitial = getFallbackInitial(resource.name ?? resource.id);
  const statusClassName = getStatusClassName(resource.status);
  const statusLabel = resource.status?.toUpperCase() ?? 'UNKNOWN';

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
          <span className={clsx('tab-card__status', statusClassName)}>
            {statusLabel}
          </span>
        </div>
      </div>
    </button>
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
  const fallbackInitial = useMemo(() => getWebTabFallbackInitial(tab), [tab]);

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

function useSurfaceThumbnail(type: SurfaceType, id: string | null | undefined) {
  const screenshotSelector = useMemo(() => selectScreenshotBySurface(type, id), [type, id]);
  const screenshot = useSurfaceMediaStore(screenshotSelector);
  const thumbStyle = useMemo<CSSProperties | undefined>(
    () => buildThumbStyle(screenshot?.dataUrl),
    [screenshot?.dataUrl],
  );
  return {
    screenshot,
    thumbStyle,
    hasScreenshot: Boolean(screenshot),
  };
}

function useAppCardMeta(app: App) {
  const displayName = getAppDisplayName(app);
  const identifier = useMemo(() => {
    const resolved = resolveAppIdentifier(app) ?? app.id;
    if (!resolved) {
      return null;
    }
    const trimmed = resolved.trim();
    return trimmed.length > 0 ? trimmed : null;
  }, [app]);
  const { screenshot, thumbStyle, hasScreenshot } = useSurfaceThumbnail('app', identifier);
  const viewCountLabel = formatViewCount(app.view_count);
  const completenessScore = app.completeness_score;
  const hasCompletenessScore = typeof completenessScore === 'number' && completenessScore >= 0;
  const completenessLevel = getCompletenessLevel(completenessScore);
  const hasStatus = Boolean(app.status);
  const statusClassName = getStatusClassName(app.status);
  const statusLabel = app.status ?? 'Unknown';
  const fallbackInitial = getFallbackInitial(displayName);
  const showThumbOverlay = Boolean(app.status || viewCountLabel || hasCompletenessScore);
  const completenessTitle = hasCompletenessScore
    ? `Completeness: ${completenessScore}/100${app.completeness_classification ? ` (${app.completeness_classification.replace(/_/g, ' ')})` : ''}`
    : null;

  return {
    displayName,
    screenshot,
    thumbStyle,
    hasScreenshot,
    viewCountLabel,
    completenessScore,
    hasCompletenessScore,
    completenessLevel,
    showThumbOverlay,
    hasStatus,
    statusClassName,
    statusLabel,
    fallbackInitial,
    completenessTitle,
  };
}

function AppCardOverlay({
  show,
  hasStatus,
  statusLabel,
  statusClassName,
  hasCompletenessScore,
  completenessScore,
  completenessLevel,
  completenessTitle,
  viewCountLabel,
}: {
  show: boolean;
  hasStatus: boolean;
  statusLabel: string;
  statusClassName: string;
  hasCompletenessScore: boolean;
  completenessScore: number | null | undefined;
  completenessLevel: 'none' | 'critical' | 'low' | 'medium' | 'high';
  completenessTitle: string | null;
  viewCountLabel: string | null;
}) {
  if (!show) {
    return null;
  }
  return (
    <div className="tab-card__thumb-overlay">
      {hasStatus && (
        <span className="tab-card__status-indicator">
          <span
            className={clsx('tab-card__status-dot', statusClassName)}
            aria-hidden
          />
          <span className="visually-hidden">Status: {statusLabel}</span>
        </span>
      )}
      <div className="tab-card__thumb-overlay-right">
        {hasCompletenessScore && (
          <span
            className={clsx(
              'tab-card__chip',
              'tab-card__chip--muted',
              'tab-card__chip--completeness',
              `tab-card__chip--completeness-${completenessLevel}`
            )}
            aria-label={`Completeness: ${completenessScore}%`}
            title={completenessTitle ?? undefined}
          >
            <Target size={14} aria-hidden />
            <span className="tab-card__completeness-score" aria-hidden>{completenessScore}</span>
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
    </div>
  );
}
