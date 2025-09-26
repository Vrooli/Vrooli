import { useState, useEffect, useCallback, useRef } from 'react';
import { Loader2 } from 'lucide-react';

type MaintenanceState = 'active' | 'inactive' | string;

interface StatusIndicatorProps {
  fallbackOnline?: boolean;
}

interface ApiConnectivityStatus {
  connected?: boolean;
  latency_ms?: number;
  error?: {
    code?: string;
    message?: string;
  } | null;
}

interface SystemStatus {
  status?: string;
  service?: string;
  timestamp?: number | string;
  uptime?: number;
  processor_active?: boolean;
  maintenance_state?: MaintenanceState;
  api_connectivity?: ApiConnectivityStatus;
  checks?: Record<string, unknown>;
}

export const StatusIndicator = ({ fallbackOnline = true }: StatusIndicatorProps) => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isToggling, setIsToggling] = useState(false);

  const dotButtonRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  const fetchSystemStatus = useCallback(async () => {
    try {
      setFetching(true);
      setError(null);
      const response = await fetch('/health', {
        headers: {
          'Accept': 'application/json'
        }
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch system status (HTTP ${response.status})`);
      }

      const data = await response.json();
      setSystemStatus(data);
    } catch (err) {
      console.error('Failed to fetch system status:', err);
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
      setFetching(false);
    }
  }, []);

  const toggleSystemStatus = useCallback(async () => {
    if (!systemStatus) {
      await fetchSystemStatus();
      return;
    }

    const isCurrentlyActive = systemStatus.processor_active ?? (systemStatus.maintenance_state === 'active');
    const nextState: MaintenanceState = isCurrentlyActive ? 'inactive' : 'active';

    try {
      setIsToggling(true);

      const response = await fetch('/api/maintenance/state', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          maintenanceState: nextState
        })
      });

      if (!response.ok) {
        throw new Error('Failed to update system status');
      }

      const data = await response.json();
      if (data.success === false) {
        throw new Error(data.error || 'Failed to update status');
      }

      setSystemStatus(prev => prev ? {
        ...prev,
        processor_active: nextState === 'active',
        maintenance_state: nextState
      } : {
        processor_active: nextState === 'active',
        maintenance_state: nextState
      });

      // Refresh from server to confirm
      setTimeout(fetchSystemStatus, 500);
    } catch (err) {
      console.error('Failed to toggle system status:', err);
      setError(err instanceof Error ? err.message : 'Failed to update status');
      fetchSystemStatus();
    } finally {
      setIsToggling(false);
    }
  }, [fetchSystemStatus, systemStatus]);

  useEffect(() => {
    fetchSystemStatus();
    const interval = setInterval(fetchSystemStatus, 10000);
    return () => clearInterval(interval);
  }, [fetchSystemStatus]);

  useEffect(() => {
    if (!isPopoverOpen) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const targetNode = event.target as Node;
      if (dotButtonRef.current?.contains(targetNode)) {
        return;
      }
      if (popoverRef.current?.contains(targetNode)) {
        return;
      }
      setIsPopoverOpen(false);
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isPopoverOpen]);

  const isActive = systemStatus?.processor_active ?? (systemStatus?.maintenance_state === 'active');

  const onlineStatus = (() => {
    if (error) {
      return 'error';
    }

    const status = systemStatus?.status?.toLowerCase();
    if (status === 'unhealthy') {
      return 'offline';
    }
    if (status) {
      return status;
    }

    return fallbackOnline ? 'online' : 'offline';
  })();

  const dotClass = onlineStatus === 'offline'
    ? 'offline'
    : onlineStatus === 'error'
      ? 'error'
      : '';

  const timestamp = systemStatus?.timestamp;
  const formattedTimestamp = typeof timestamp === 'number'
    ? new Date(timestamp * 1000).toLocaleString()
    : timestamp
      ? new Date(timestamp).toLocaleString()
      : undefined;

  const apiConnectivity = systemStatus?.api_connectivity;

  const handleTogglePopover = () => {
    if (!loading) {
      setIsPopoverOpen(prev => !prev);
    }
  };

  return (
    <div className="status-control-group">
      <button
        ref={dotButtonRef}
        className={`header-button status-dot-button ${loading ? 'loading' : ''}`}
        onClick={handleTogglePopover}
        type="button"
        title="View status details"
        aria-haspopup="dialog"
        aria-expanded={isPopoverOpen}
        disabled={loading && !systemStatus}
      >
        {loading && !systemStatus ? (
          <Loader2 size={16} className="animate-spin" />
        ) : (
          <span className={`status-dot ${dotClass}`} aria-hidden="true" />
        )}
      </button>

      {isPopoverOpen && (
        <div ref={popoverRef} className="status-popover" role="dialog" aria-label="System status details">
          <div className="status-popover-section">
            <span className="status-popover-label">Overall</span>
            <span className="status-popover-value">{systemStatus?.status ?? (error ? 'Unavailable' : 'Unknown')}</span>
          </div>
          {systemStatus?.service && (
            <div className="status-popover-section">
              <span className="status-popover-label">Service</span>
              <span className="status-popover-value">{systemStatus.service}</span>
            </div>
          )}
          <div className="status-popover-section">
            <span className="status-popover-label">Processor</span>
            <span className="status-popover-value">{isActive ? 'Active' : 'Inactive'}</span>
          </div>
          {systemStatus?.maintenance_state && (
            <div className="status-popover-section">
              <span className="status-popover-label">Mode</span>
              <span className="status-popover-value">{systemStatus.maintenance_state}</span>
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
          {error && (
            <div className="status-popover-error">{error}</div>
          )}
          <button
            className="status-popover-refresh"
            onClick={fetchSystemStatus}
            type="button"
            disabled={fetching}
          >
            {fetching ? <Loader2 size={14} className="animate-spin" /> : 'Refresh status'}
          </button>
        </div>
      )}

      <button
        className={`header-button status-toggle ${isActive ? 'active' : 'inactive'}`}
        onClick={toggleSystemStatus}
        type="button"
        title={isActive ? 'Pause monitoring' : 'Activate monitoring'}
        disabled={loading || isToggling}
      >
        {isToggling && <Loader2 size={14} className="animate-spin" />}
        <span>{isActive ? 'Active' : 'Inactive'}</span>
      </button>
    </div>
  );
};
