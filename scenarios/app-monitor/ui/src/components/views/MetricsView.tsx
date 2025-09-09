import { useState, useEffect } from 'react';
import { appService } from '@/services/api';
import './MetricsView.css';

export default function MetricsView() {
  const [systemMonitorUrl, setSystemMonitorUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchSystemMonitorUrl();
  }, []);

  const fetchSystemMonitorUrl = async () => {
    try {
      // Get all running apps to find system-monitor
      const apps = await appService.getApps();
      
      // Find system-monitor scenario
      const systemMonitor = apps.find(app => 
        app.scenario_name === 'system-monitor' || 
        app.name === 'system-monitor' ||
        app.id === 'system-monitor'
      );
      
      if (systemMonitor && systemMonitor.port_mappings) {
        // Look for UI_PORT or ui port
        const uiPort = systemMonitor.port_mappings.UI_PORT || 
                      systemMonitor.port_mappings.ui ||
                      systemMonitor.port_mappings.port;
        
        if (uiPort) {
          setSystemMonitorUrl(`http://localhost:${uiPort}`);
        } else {
          setError('System Monitor is running but no UI port found');
        }
      } else {
        setError('System Monitor is not running. Please start it with: vrooli scenario run system-monitor');
      }
    } catch (err) {
      console.error('Failed to fetch system monitor URL:', err);
      setError('Failed to connect to system monitor');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="metrics-view">
        <div className="panel-header">
          <h2>SYSTEM METRICS</h2>
        </div>
        <div className="metrics-container loading-state">
          <div className="loading-spinner">⟳</div>
          <p>Locating System Monitor...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="metrics-view">
        <div className="panel-header">
          <h2>SYSTEM METRICS</h2>
        </div>
        <div className="metrics-container error-state">
          <div className="error-icon">⚠</div>
          <p>{error}</p>
          <button 
            className="control-btn refresh"
            onClick={() => {
              setLoading(true);
              setError(null);
              fetchSystemMonitorUrl();
            }}
          >
            RETRY
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="metrics-view">
      <div className="panel-header">
        <h2>SYSTEM METRICS</h2>
        <div className="panel-controls">
          <a 
            href={systemMonitorUrl!} 
            target="_blank" 
            rel="noopener noreferrer"
            className="control-btn"
            title="Open in new tab"
          >
            ↗
          </a>
        </div>
      </div>
      <div className="metrics-iframe-container">
        <iframe
          src={systemMonitorUrl!}
          title="System Monitor"
          className="metrics-iframe"
          style={{
            width: '100%',
            height: 'calc(100vh - 200px)',
            border: 'none',
            backgroundColor: '#0a0e0a'
          }}
        />
      </div>
    </div>
  );
}