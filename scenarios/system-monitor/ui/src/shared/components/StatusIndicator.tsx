import { useState, useCallback, useRef } from 'react';
import { Loader2 } from 'lucide-react';
import { useClickOutside } from '../hooks/useClickOutside';
import type { SystemHealthStatus } from '../../features/monitoring/hooks/useSystemMonitor';

interface StatusIndicatorProps {
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  onToggleMonitoring: () => Promise<void>;
  onRefreshHealth: () => Promise<void>;
  isLoading: boolean;
}

export const StatusIndicator = ({
  healthStatus,
  healthError,
  onToggleMonitoring,
  onRefreshHealth,
  isLoading
}: StatusIndicatorProps) => {
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isToggling, setIsToggling] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const dotButtonRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  const closePopover = useCallback(() => setIsPopoverOpen(false), []);
  useClickOutside([dotButtonRef, popoverRef], closePopover, isPopoverOpen);

  const isActive = healthStatus?.processor_active ?? (healthStatus?.maintenance_state === 'active');

  const onlineStatus = (() => {
    if (healthError) {
      return 'error';
    }

    const status = healthStatus?.status?.toLowerCase();
    if (status === 'unhealthy') {
      return 'offline';
    }
    if (status) {
      return status;
    }

    return 'offline';
  })();

  const dotClass = onlineStatus === 'offline'
    ? 'offline'
    : onlineStatus === 'error'
      ? 'error'
      : '';

  const timestamp = healthStatus?.timestamp;
  const formattedTimestamp = typeof timestamp === 'number'
    ? new Date(timestamp * 1000).toLocaleString()
    : timestamp
      ? new Date(timestamp).toLocaleString()
      : undefined;

  const apiConnectivity = healthStatus?.api_connectivity;

  const handleTogglePopover = () => {
    if (!isLoading) {
      setIsPopoverOpen(prev => !prev);
    }
  };

  const handleToggle = async () => {
    setIsToggling(true);
    try {
      await onToggleMonitoring();
    } finally {
      setIsToggling(false);
    }
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await onRefreshHealth();
    } finally {
      setIsRefreshing(false);
    }
  };

  return (
    <div className="status-control-group">
      <button
        ref={dotButtonRef}
        className={`header-button status-dot-button ${isLoading ? 'loading' : ''}`}
        onClick={handleTogglePopover}
        type="button"
        title="View status details"
        aria-haspopup="dialog"
        aria-expanded={isPopoverOpen}
        disabled={isLoading && !healthStatus}
      >
        {isLoading && !healthStatus ? (
          <Loader2 size={16} className="animate-spin" />
        ) : (
          <span className={`status-dot ${dotClass}`} aria-hidden="true" />
        )}
      </button>

      {isPopoverOpen && (
        <div ref={popoverRef} className="status-popover" role="dialog" aria-label="System status details">
          <div className="status-popover-section">
            <span className="status-popover-label">Overall</span>
            <span className="status-popover-value">{healthStatus?.status ?? (healthError ? 'Unavailable' : 'Unknown')}</span>
          </div>
          {healthStatus?.service && (
            <div className="status-popover-section">
              <span className="status-popover-label">Service</span>
              <span className="status-popover-value">{healthStatus.service}</span>
            </div>
          )}
          <div className="status-popover-section">
            <span className="status-popover-label">Processor</span>
            <span className="status-popover-value">{isActive ? 'Active' : 'Inactive'}</span>
          </div>
          {healthStatus?.maintenance_state && (
            <div className="status-popover-section">
              <span className="status-popover-label">Mode</span>
              <span className="status-popover-value">{healthStatus.maintenance_state}</span>
            </div>
          )}
          {apiConnectivity && (
            <div className="status-popover-section">
              <span className="status-popover-label">API</span>
              <span className="status-popover-value">
                {apiConnectivity.connected ? 'Connected' : 'Unavailable'}
                {typeof apiConnectivity.latency_ms === 'number' && ` · ${Math.round(apiConnectivity.latency_ms)}ms`}
              </span>
            </div>
          )}
          {formattedTimestamp && (
            <div className="status-popover-section">
              <span className="status-popover-label">Updated</span>
              <span className="status-popover-value">{formattedTimestamp}</span>
            </div>
          )}
          {healthError && (
            <div className="status-popover-error">{healthError}</div>
          )}
          <button
            className="status-popover-refresh"
            onClick={handleRefresh}
            type="button"
            disabled={isRefreshing}
          >
            {isRefreshing ? <Loader2 size={14} className="animate-spin" /> : 'Refresh status'}
          </button>
        </div>
      )}

      <button
        className={`header-button status-toggle ${isActive ? 'active' : 'inactive'}`}
        onClick={handleToggle}
        type="button"
        title={isActive ? 'Pause monitoring' : 'Activate monitoring'}
        disabled={isLoading || isToggling}
      >
        {isToggling && <Loader2 size={14} className="animate-spin" />}
        <span>{isActive ? 'Active' : 'Inactive'}</span>
      </button>
    </div>
  );
};
