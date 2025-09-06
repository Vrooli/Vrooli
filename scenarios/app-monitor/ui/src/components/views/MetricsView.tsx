import { useState, useEffect } from 'react';
import { metricsService } from '@/services/api';
import type { SystemMetrics } from '@/types';
import './MetricsView.css';

interface MetricsViewProps {
  metrics: SystemMetrics | null;
}

export default function MetricsView({ metrics }: MetricsViewProps) {
  const [interval, setInterval] = useState<'1m' | '5m' | '15m' | '1h'>('5m');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchMetricsHistory();
  }, [interval]);

  const fetchMetricsHistory = async () => {
    setLoading(true);
    try {
      // For now, just log - can be used for historical charts later
      const data = await metricsService.getMetricsHistory(interval);
      console.log('Metrics history:', data);
    } catch (error) {
      console.error('Failed to fetch metrics history:', error);
    } finally {
      setLoading(false);
    }
  };

  // Show loading state when metrics haven't loaded yet
  if (!metrics) {
    return (
      <div className="metrics-view">
        <div className="panel-header">
          <h2>PERFORMANCE METRICS</h2>
          <div className="panel-controls">
            <select className="metric-select" disabled>
              <option>Loading...</option>
            </select>
          </div>
        </div>

        <div className="metrics-container">
          <div className="metric-card loading">
            <h3>CPU USAGE</h3>
            <div className="metric-chart">
              <div className="metric-bar loading-bar"></div>
            </div>
            <div className="metric-value">Loading...</div>
          </div>
          
          <div className="metric-card loading">
            <h3>MEMORY</h3>
            <div className="metric-chart">
              <div className="metric-bar loading-bar"></div>
            </div>
            <div className="metric-value">Loading...</div>
          </div>
          
          <div className="metric-card loading">
            <h3>NETWORK I/O</h3>
            <div className="metric-chart">
              <div className="metric-bar loading-bar"></div>
            </div>
            <div className="metric-value">Loading...</div>
          </div>
          
          <div className="metric-card loading">
            <h3>DISK USAGE</h3>
            <div className="metric-chart">
              <div className="metric-bar loading-bar"></div>
            </div>
            <div className="metric-value">Loading...</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="metrics-view">
      <div className="panel-header">
        <h2>PERFORMANCE METRICS</h2>
        <div className="panel-controls">
          <select
            className="metric-select"
            value={interval}
            onChange={(e) => setInterval(e.target.value as any)}
          >
            <option value="1m">1 MIN</option>
            <option value="5m">5 MIN</option>
            <option value="15m">15 MIN</option>
            <option value="1h">1 HOUR</option>
          </select>
        </div>
      </div>

      <div className="metrics-container">
        <div className="metric-card">
          <h3>CPU USAGE</h3>
          <div className="metric-chart">
            <div className="metric-bar" style={{ height: `${metrics.cpu}%` }}></div>
          </div>
          <div className="metric-value">{metrics.cpu.toFixed(1)}%</div>
        </div>
        
        <div className="metric-card">
          <h3>MEMORY</h3>
          <div className="metric-chart">
            <div className="metric-bar" style={{ height: `${metrics.memory}%` }}></div>
          </div>
          <div className="metric-value">{metrics.memory.toFixed(1)}%</div>
        </div>
        
        <div className="metric-card">
          <h3>NETWORK I/O</h3>
          <div className="metric-chart">
            <div className="metric-bar" style={{ height: `${Math.min(metrics.network / 10, 100)}%` }}></div>
          </div>
          <div className="metric-value">{metrics.network.toFixed(1)} KB/s</div>
        </div>
        
        <div className="metric-card">
          <h3>DISK USAGE</h3>
          <div className="metric-chart">
            <div className="metric-bar" style={{ height: `${metrics.disk}%` }}></div>
          </div>
          <div className="metric-value">{metrics.disk.toFixed(1)}%</div>
        </div>
      </div>

      {loading && (
        <div className="loading-overlay">
          <div className="spinner small"></div>
        </div>
      )}
    </div>
  );
}