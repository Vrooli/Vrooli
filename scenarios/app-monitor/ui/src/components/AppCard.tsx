import React, { memo, useCallback, useMemo } from 'react';
import clsx from 'clsx';
import type { LucideIcon } from 'lucide-react';
import { ActivitySquare, Cable, Clock3, Cpu, Info, Play, Square } from 'lucide-react';
import type { App } from '@/types';
import './AppCard.css';

interface AppCardProps {
  app: App;
  onCardClick: (app: App) => void;
  onDetails: (app: App) => void;
  onStart: (appId: string) => void;
  onStop: (appId: string) => void;
  viewMode?: 'grid' | 'list';
  isActive?: boolean;
}

const StatusBadge = memo(({ status }: { status: string }) => {
  const normalized = status?.toLowerCase() || 'unknown';
  const statusClass = clsx('status-badge', normalized);
  return <span className={statusClass}>{normalized.toUpperCase()}</span>;
});
StatusBadge.displayName = 'StatusBadge';

interface MetricProps {
  label: string;
  value?: string | number;
  Icon: LucideIcon;
}

const MetricChip = memo(({ label, value, Icon }: MetricProps) => {
  const displayValue = typeof value === 'number'
    ? value.toString()
    : (value?.toString().trim() || '—');

  return (
    <div className="metric-chip">
      <Icon aria-hidden className="metric-chip__icon" />
      <div className="metric-chip__content">
        <span className="metric-chip__label">{label}</span>
        <span className="metric-chip__value">{displayValue}</span>
      </div>
    </div>
  );
});
MetricChip.displayName = 'MetricChip';

const orderedPortMetrics = (app: App) => {
  const entries = Object.entries(app.port_mappings || {})
    .map(([label, value]) => ({
      label: label.toUpperCase(),
      value: typeof value === 'number' ? String(value) : String(value ?? ''),
    }))
    .filter(({ value }) => value !== '');

  const priorityOrder = ['UI_PORT', 'API_PORT'];
  const prioritized = entries
    .filter((entry) => priorityOrder.includes(entry.label))
    .sort((a, b) => priorityOrder.indexOf(a.label) - priorityOrder.indexOf(b.label));

  const remaining = entries
    .filter((entry) => !priorityOrder.includes(entry.label))
    .sort((a, b) => a.label.localeCompare(b.label));

  return [...prioritized, ...remaining];
};

const AppCard = memo<AppCardProps>(({ 
  app,
  onCardClick,
  onDetails,
  onStart,
  onStop,
  viewMode = 'grid',
  isActive = false,
}) => {
  const isPartial = Boolean(app.is_partial);

  const handleClick = useCallback(() => {
    onCardClick(app);
  }, [onCardClick, app]);

  const handleDetails = useCallback((event: React.MouseEvent) => {
    event.stopPropagation();
    onDetails(app);
  }, [onDetails, app]);

  const handleStart = useCallback((event: React.MouseEvent) => {
    event.stopPropagation();
    onStart(app.id);
  }, [onStart, app.id]);

  const handleStop = useCallback((event: React.MouseEvent) => {
    event.stopPropagation();
    onStop(app.id);
  }, [onStop, app.id]);

  const calculateUptime = useCallback(() => {
    if (app.uptime && app.uptime !== 'N/A') {
      return app.uptime;
    }

    const runningStates = new Set(['running', 'healthy', 'degraded', 'unhealthy']);
    if (!runningStates.has(app.status) || !app.created_at) {
      return 'N/A';
    }

    const start = new Date(app.created_at);
    const now = new Date();
    const diff = now.getTime() - start.getTime();

    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }, [app.created_at, app.status, app.uptime]);

  const uptime = isPartial ? 'SYNCING' : calculateUptime();
  const isRunning = ['running', 'healthy', 'degraded', 'unhealthy'].includes(app.status);

  const metrics = useMemo(() => {
    const items: MetricProps[] = [
      { label: 'UPTIME', value: uptime, Icon: Clock3 },
    ];

    const portMetrics = orderedPortMetrics(app)
      .slice(0, 2)
      .map(({ label, value }) => ({
        label: label.replace(/_/g, ' '),
        value,
        Icon: Cable,
      }));

    items.push(...portMetrics);

    const processCount = Number(app.config?.process_count ?? NaN);
    if (Number.isFinite(processCount) && processCount > 0) {
      items.push({ label: 'PROCESSES', value: processCount, Icon: Cpu });
    }

    if (!isPartial && app.runtime && app.runtime.trim()) {
      items.push({ label: 'RUNTIME', value: app.runtime, Icon: ActivitySquare });
    }

    return items.slice(0, 4);
  }, [app, uptime, isPartial]);

  const cardClasses = clsx(
    'app-card',
    viewMode,
    app.status.toLowerCase(),
    {
      active: isActive,
      partial: isPartial,
    }
  );

  const description = (app.description || '').trim() || (isPartial ? 'Fetching scenario details…' : 'No description provided yet.');
  const tags = app.tags || [];
  const displayedTags = tags.slice(0, 4);
  const overflowCount = Math.max(0, tags.length - displayedTags.length);
  const rawVersion = app.config?.version;
  const version = typeof rawVersion === 'string' ? rawVersion.trim() : '';

  return (
    <div className={cardClasses} onClick={handleClick}>
      <div className="app-card__glow" aria-hidden />
      <div className="app-card__header">
        <div className="app-card__identity">
          <h3 className="app-name">{app.name}</h3>
          {version && <span className="app-meta">v{version}</span>}
        </div>
        <div className="app-card__status-group">
          {isPartial && <span className="app-card__sync" aria-live="polite">SYNCING…</span>}
          <StatusBadge status={app.status} />
        </div>
      </div>

      <p className="app-card__description">{description}</p>

      {displayedTags.length > 0 && (
        <div className="app-card__tags">
          {displayedTags.map((tag) => (
            <span key={tag} className="app-card__tag">{tag}</span>
          ))}
          {overflowCount > 0 && <span className="app-card__tag extra">+{overflowCount}</span>}
        </div>
      )}

      <div className="app-card__metrics">
        {metrics.map(({ label, value, Icon }) => (
          <MetricChip key={label} label={label} value={value} Icon={Icon} />
        ))}
      </div>

      <div className="app-card__footer">
        {isRunning ? (
          <button type="button" className="action-btn stop" onClick={handleStop}>
            <Square aria-hidden className="action-icon" />
            <span>STOP</span>
          </button>
        ) : (
          <button type="button" className="action-btn start" onClick={handleStart}>
            <Play aria-hidden className="action-icon" />
            <span>START</span>
          </button>
        )}
        <button type="button" className="action-btn details" onClick={handleDetails}>
          <Info aria-hidden className="action-icon" />
          <span>DETAILS</span>
        </button>
      </div>
    </div>
  );
}, (prevProps, nextProps) => (
  prevProps.app.id === nextProps.app.id &&
  prevProps.app.status === nextProps.app.status &&
  prevProps.app.runtime === nextProps.app.runtime &&
  prevProps.app.created_at === nextProps.app.created_at &&
  prevProps.viewMode === nextProps.viewMode &&
  prevProps.isActive === nextProps.isActive &&
  prevProps.app.is_partial === nextProps.app.is_partial &&
  JSON.stringify(prevProps.app.port_mappings) === JSON.stringify(nextProps.app.port_mappings)
));

AppCard.displayName = 'AppCard';

export default AppCard;
