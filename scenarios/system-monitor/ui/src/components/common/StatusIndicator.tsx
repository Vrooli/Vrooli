import React, { useState, useEffect } from 'react';
import { Play, Pause, Settings, Loader2 } from 'lucide-react';

interface StatusIndicatorProps {
  onOpenSettings?: () => void;
}

interface SystemStatus {
  processor_active: boolean;
  maintenance_state: 'active' | 'inactive';
  service: string;
  status: string;
}

export const StatusIndicator = ({ onOpenSettings }: StatusIndicatorProps) => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchSystemStatus();
    // Poll for status updates every 10 seconds
    const interval = setInterval(fetchSystemStatus, 10000);
    return () => clearInterval(interval);
  }, []);

  const fetchSystemStatus = async () => {
    try {
      setError(null);
      const response = await fetch('/health');
      if (response.ok) {
        const data = await response.json();
        setSystemStatus(data);
      } else {
        throw new Error('Failed to fetch system status');
      }
    } catch (error) {
      console.error('Failed to fetch system status:', error);
      setError(error instanceof Error ? error.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const toggleSystemStatus = async () => {
    if (!systemStatus) return;
    
    try {
      const newState = systemStatus.processor_active ? 'inactive' : 'active';
      
      const response = await fetch('/api/maintenance/state', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          maintenanceState: newState
        })
      });

      if (response.ok) {
        const data = await response.json();
        if (data.success) {
          // Update local state immediately
          setSystemStatus(prev => prev ? {
            ...prev,
            processor_active: newState === 'active',
            maintenance_state: newState
          } : null);
          
          // Refresh status from server to be sure
          setTimeout(fetchSystemStatus, 500);
        } else {
          throw new Error(data.error || 'Failed to update status');
        }
      } else {
        throw new Error('Failed to update system status');
      }
    } catch (error) {
      console.error('Failed to toggle system status:', error);
      setError(error instanceof Error ? error.message : 'Failed to update status');
      // Refresh status to ensure UI is in sync
      fetchSystemStatus();
    }
  };

  if (loading) {
    return (
      <div className="status-indicator loading" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--spacing-sm)',
        padding: 'var(--spacing-sm) var(--spacing-md)',
        background: 'rgba(0, 255, 0, 0.1)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-md)',
        fontSize: 'var(--font-size-sm)'
      }}>
        <Loader2 size={16} className="animate-spin" style={{ color: 'var(--color-accent)' }} />
        <span style={{ color: 'var(--color-text)' }}>Loading status...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="status-indicator error" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--spacing-sm)',
        padding: 'var(--spacing-sm) var(--spacing-md)',
        background: 'rgba(255, 0, 0, 0.1)',
        border: '1px solid var(--color-error)',
        borderRadius: 'var(--border-radius-md)',
        fontSize: 'var(--font-size-sm)',
        color: 'var(--color-error)'
      }}>
        <Pause size={16} />
        <span>Status Error</span>
        <button
          onClick={fetchSystemStatus}
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--color-error)',
            cursor: 'pointer',
            textDecoration: 'underline',
            fontSize: 'inherit'
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  if (!systemStatus) {
    return null;
  }

  const isActive = systemStatus.processor_active;
  const statusColor = isActive ? 'var(--color-success)' : 'var(--color-warning)';
  const statusIcon = isActive ? Play : Pause;
  const statusText = isActive ? 'Active' : 'Paused';

  return (
    <div className="status-indicator" style={{
      display: 'flex',
      alignItems: 'center',
      gap: 'var(--spacing-md)',
      padding: 'var(--spacing-sm) var(--spacing-md)',
      background: isActive ? 'rgba(0, 255, 0, 0.1)' : 'rgba(255, 255, 0, 0.1)',
      border: `1px solid ${statusColor}`,
      borderRadius: 'var(--border-radius-md)',
      fontSize: 'var(--font-size-sm)',
      boxShadow: isActive ? '0 0 10px rgba(0, 255, 0, 0.2)' : '0 0 10px rgba(255, 255, 0, 0.2)'
    }}>
      {/* Status Display */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--spacing-sm)'
      }}>
        {React.createElement(statusIcon, {
          size: 16,
          style: { color: statusColor, filter: `drop-shadow(0 0 4px ${statusColor})` }
        })}
        <span style={{
          color: 'var(--color-text-bright)',
          fontWeight: 'bold',
          textTransform: 'uppercase',
          letterSpacing: '0.5px'
        }}>
          {statusText}
        </span>
      </div>

      {/* Divider */}
      <div style={{
        width: '1px',
        height: '20px',
        background: 'var(--color-accent)',
        opacity: 0.3
      }} />

      {/* Toggle Button */}
      <button
        onClick={toggleSystemStatus}
        className="btn btn-secondary"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 'var(--spacing-xs)',
          padding: 'var(--spacing-xs) var(--spacing-sm)',
          fontSize: 'var(--font-size-xs)',
          minWidth: 'auto',
          textTransform: 'uppercase',
          letterSpacing: '0.5px'
        }}
        title={`Click to ${isActive ? 'pause' : 'activate'} system monitoring`}
      >
        {isActive ? 'Pause' : 'Activate'}
      </button>

      {/* Settings Button */}
      {onOpenSettings && (
        <>
          <div style={{
            width: '1px',
            height: '20px',
            background: 'var(--color-accent)',
            opacity: 0.3
          }} />
          
          <button
            onClick={onOpenSettings}
            className="btn btn-secondary"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-xs)',
              padding: 'var(--spacing-xs) var(--spacing-sm)',
              fontSize: 'var(--font-size-xs)',
              minWidth: 'auto'
            }}
            title="Open system monitor settings"
          >
            <Settings size={14} />
            Settings
          </button>
        </>
      )}
    </div>
  );
};