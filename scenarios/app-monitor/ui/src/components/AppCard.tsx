import React, { memo, useCallback } from 'react';
import clsx from 'clsx';
import type { App } from '@/types';
import './AppCard.css';

interface AppCardProps {
  app: App;
  onClick: (app: App) => void;
  onStart: (appId: string) => void;
  onStop: (appId: string) => void;
  viewMode?: 'grid' | 'list';
}

// Memoized status badge component
const StatusBadge = memo(({ status }: { status: string }) => {
  const statusClass = clsx('status-badge', status.toLowerCase());
  return <span className={statusClass}>{status.toUpperCase()}</span>;
});
StatusBadge.displayName = 'StatusBadge';

// Memoized metric display component
const MetricDisplay = memo(({ label, value, unit = '' }: { label: string; value: string | number; unit?: string }) => (
  <div className="metric-item">
    <span className="metric-label">{label}:</span>
    <span className="metric-value">{value}{unit}</span>
  </div>
));
MetricDisplay.displayName = 'MetricDisplay';

// Memoized port display component
const PortDisplay = memo(({ ports }: { ports: Record<string, number> }) => {
  const portEntries = Object.entries(ports);
  if (portEntries.length === 0) return <MetricDisplay label="PORT" value="N/A" />;
  
  // Display first port, or multiple if space allows
  const primaryPort = portEntries[0];
  return (
    <MetricDisplay 
      label={primaryPort[0].toUpperCase()} 
      value={primaryPort[1]} 
    />
  );
});
PortDisplay.displayName = 'PortDisplay';

// Main AppCard component with memoization
const AppCard = memo<AppCardProps>(({ 
  app, 
  onClick, 
  onStart, 
  onStop, 
  viewMode = 'grid' 
}) => {
  // Memoize event handlers to prevent re-creation
  const handleClick = useCallback(() => {
    onClick(app);
  }, [onClick, app]);

  const handleStart = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onStart(app.id);
  }, [onStart, app.id]);

  const handleStop = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onStop(app.id);
  }, [onStop, app.id]);

  // Calculate uptime from created_at
  const calculateUptime = useCallback(() => {
    if (app.status !== 'running' || !app.created_at) return 'N/A';
    
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
  const isRunning = app.status === 'running' || app.status === 'healthy';
  
  return (
    <div 
      className={clsx('app-card', viewMode, app.status.toLowerCase())}
      onClick={handleClick}
    >
      <div className="app-header">
        <h3 className="app-name">{app.name}</h3>
        <StatusBadge status={app.status} />
      </div>
      
      <div className="app-metrics">
        <MetricDisplay label="CPU" value={app.cpu || 0} unit="%" />
        <MetricDisplay label="MEM" value={app.memory || 0} unit="%" />
        <MetricDisplay label="UPTIME" value={uptime} />
        <PortDisplay ports={app.port_mappings || {}} />
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
        <button className="action-btn details" onClick={handleClick}>
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
    prevProps.app.cpu === nextProps.app.cpu &&
    prevProps.app.memory === nextProps.app.memory &&
    prevProps.app.created_at === nextProps.app.created_at &&
    prevProps.viewMode === nextProps.viewMode &&
    JSON.stringify(prevProps.app.port_mappings) === JSON.stringify(nextProps.app.port_mappings)
  );
});

AppCard.displayName = 'AppCard';

export default AppCard;