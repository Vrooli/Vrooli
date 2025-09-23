import React, { memo, useCallback, useMemo } from 'react';
import clsx from 'clsx';
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

// Memoized status badge component
const StatusBadge = memo(({ status }: { status: string }) => {
  const normalized = status?.toLowerCase() || 'unknown';
  const statusClass = clsx('status-badge', normalized);
  return <span className={statusClass}>{normalized.toUpperCase()}</span>;
});
StatusBadge.displayName = 'StatusBadge';

// Memoized metric display component
const COMPACT_VALUE_STYLE: React.CSSProperties = {
  fontSize: '0.68rem',
  letterSpacing: '0.07em',
};

const COMPACT_LABEL_STYLE: React.CSSProperties = {
  fontSize: '0.5rem',
  letterSpacing: '0.14em',
};

const MetricDisplay = memo(({ label, value }: { label: string; value?: string | number }) => {
  let displayValue = '—';

  if (typeof value === 'number' && Number.isFinite(value)) {
    displayValue = `${value}`;
  } else if (typeof value === 'string') {
    const trimmed = value.trim();
    if (trimmed) {
      displayValue = trimmed;
    }
  }

  return (
    <div className="metric-item">
      <span className="metric-label" style={COMPACT_LABEL_STYLE}>{label}</span>
      <span className="metric-value" style={COMPACT_VALUE_STYLE}>{displayValue}</span>
    </div>
  );
});
MetricDisplay.displayName = 'MetricDisplay';

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

// Main AppCard component with memoization
const AppCard = memo<AppCardProps>(({ 
  app, 
  onCardClick, 
  onDetails, 
  onStart, 
  onStop, 
  viewMode = 'grid',
  isActive = false,
}) => {
  // Memoize event handlers to prevent re-creation
  const handleClick = useCallback(() => {
    onCardClick(app);
  }, [onCardClick, app]);

  const handleDetails = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onDetails(app);
  }, [onDetails, app]);

  const handleStart = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onStart(app.id);
  }, [onStart, app.id]);

  const handleStop = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onStop(app.id);
  }, [onStop, app.id]);

  // Calculate uptime from provided uptime field or created_at timestamp
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
  }, [app.created_at, app.status]);

  const uptime = calculateUptime();
  const isRunning = ['running', 'healthy', 'degraded', 'unhealthy'].includes(app.status);

  const metrics = useMemo(() => {
    const items: { label: string; value?: string }[] = [{ label: 'UPTIME', value: uptime }];

    const portMetrics = orderedPortMetrics(app)
      .map(({ label, value }) => ({ label, value }))
      .slice(0, 3);

    return [...items, ...portMetrics].slice(0, 4);
  }, [app, uptime]);
  
  return (
    <div 
      className={clsx('app-card', viewMode, app.status.toLowerCase(), { active: isActive })}
      onClick={handleClick}
    >
      <div className="app-header">
        <h3 className="app-name">{app.name}</h3>
        <StatusBadge status={app.status} />
      </div>
      
      <div className="app-metrics">
        {metrics.map(({ label, value }) => (
          <MetricDisplay key={label} label={label} value={value} />
        ))}
      </div>
      
      <div className="app-actions">
        {isRunning ? (
          <button className="action-btn stop" onClick={handleStop}>
            STOP
          </button>
        ) : (
          <button className="action-btn start" onClick={handleStart}>
            START
          </button>
        )}
        <button className="action-btn details" onClick={handleDetails}>
          DETAILS
        </button>
      </div>
    </div>
  );
}, (prevProps, nextProps) => {
  // Custom comparison function for deep equality check
  // Only re-render if relevant props actually changed
  return (
    prevProps.app.id === nextProps.app.id &&
    prevProps.app.status === nextProps.app.status &&
    prevProps.app.runtime === nextProps.app.runtime &&
    prevProps.app.created_at === nextProps.app.created_at &&
    prevProps.viewMode === nextProps.viewMode &&
    prevProps.isActive === nextProps.isActive &&
    JSON.stringify(prevProps.app.port_mappings) === JSON.stringify(nextProps.app.port_mappings)
  );
});

AppCard.displayName = 'AppCard';

export default AppCard;
