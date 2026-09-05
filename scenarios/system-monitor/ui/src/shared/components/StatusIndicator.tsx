import { useState, useCallback, useRef } from 'react';
import { Activity, Loader2 } from 'lucide-react';
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

  const closePopover = useCallback(() => { setIsPopoverOpen(false); }, []);
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

  /*
   * The dot carries system health in COLOUR ALONE and is `aria-hidden`, so the
   * state reached neither assistive technology nor a reader who cannot
   * separate the hues. The button's name was the static string "View status
   * details", which says what the control does but never what it is
   * reporting. Naming the current state here is the text alternative.
   */
  const statusWord = (() => {
    if (healthError) return 'error';
    if (isLoading && !healthStatus) return 'loading';
    if (onlineStatus === 'offline') return 'offline';
    return onlineStatus;
  })();

  const statusLabel = `System status: ${statusWord}. View status details`;

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
        title={statusLabel}
        aria-label={statusLabel}
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
            onClick={() => { void handleRefresh(); }}
            type="button"
            disabled={isRefreshing}
          >
            {isRefreshing ? <Loader2 size={14} className="animate-spin" /> : 'Refresh status'}
          </button>
        </div>
      )}

      {/*
        * The visible label is the STATE ("Active"), while the control's effect
        * is the opposite ("Pause monitoring"). Sighted users resolve that from
        * the tooltip, but the accessible name was just "Active", which
        * announces as "Active, button" and says nothing about what pressing it
        * does. `aria-pressed` makes the toggle semantics explicit and the
        * label names both the state and the action, so the visible text can
        * stay the at-a-glance state without stranding non-visual users.
        */}
      <button
        className={`header-button status-toggle ${isActive ? 'active' : 'inactive'}`}
        onClick={() => { void handleToggle(); }}
        type="button"
        aria-pressed={isActive}
        aria-label={isActive ? 'Monitoring active. Pause monitoring' : 'Monitoring inactive. Activate monitoring'}
        title={isActive ? 'Pause monitoring' : 'Activate monitoring'}
        disabled={isLoading || isToggling}
      >
        {isToggling
          ? <Loader2 size={14} className="animate-spin" aria-hidden="true" />
          : <Activity size={14} aria-hidden="true" />}
        <span className="hdr-label">{isActive ? 'Active' : 'Inactive'}</span>
      </button>
    </div>
  );
};
