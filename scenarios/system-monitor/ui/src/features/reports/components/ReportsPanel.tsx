import { useState, useEffect } from 'react';
import { BarChart, TrendingUp, RefreshCw, Loader } from 'lucide-react';
import type { SystemReport } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';

export const ReportsPanel = () => {
  const [reports, setReports] = useState<SystemReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState<string | null>(null); // Track which report is generating
  const [isRefreshing, setIsRefreshing] = useState(false);

  const loadReports = async (isRefresh = false) => {
    if (isRefresh) {
      setIsRefreshing(true);
    } else {
      setLoading(true);
    }
    
    try {
      // TODO: Implement API call to load reports
      // For now, show placeholder data
      await new Promise(resolve => setTimeout(resolve, 800));
      
      setReports([
        {
          id: 'daily-' + Date.now(),
          type: 'daily',
          generated_at: new Date().toISOString(),
          period_start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
          period_end: new Date().toISOString(),
          summary: {
            total_alerts: 3,
            avg_cpu_usage: 45.2,
            avg_memory_usage: 67.8,
            max_tcp_connections: 1247,
            uptime_percentage: 99.8
          },
          metrics: {
            cpu_trend: [40, 42, 45, 48, 46, 44, 45],
            memory_trend: [65, 66, 68, 70, 69, 67, 68],
            network_trend: [1200, 1180, 1220, 1247, 1230, 1210, 1225],
            timestamps: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
          },
          alerts: [
            {
              timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
              severity: 'medium',
              category: 'CPU',
              message: 'CPU usage exceeded 80% for 5 minutes',
              resolved: true
            }
          ],
          recommendations: [
            'Consider upgrading CPU for better performance',
            'Monitor memory usage patterns during peak hours'
          ]
        }
      ]);
    } catch (error) {
      console.error('Failed to load reports:', error);
    } finally {
      if (isRefresh) {
        setIsRefreshing(false);
      } else {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    loadReports();
  }, []);

  const generateReport = async (type: 'daily' | 'weekly') => {
    if (isGenerating || isRefreshing) return; // Prevent multiple concurrent operations
    
    setIsGenerating(type);
    
    try {
      console.log('Generating', type, 'report...');
      // TODO: Implement API call to generate report
      
      // Simulate report generation with realistic delay
      await new Promise(resolve => setTimeout(resolve, 2000 + Math.random() * 3000));
      
      // Generate new report and add to list
      const newReport = {
        id: type + '-' + Date.now(),
        type: type,
        generated_at: new Date().toISOString(),
        period_start: new Date(Date.now() - (type === 'daily' ? 24 : 168) * 60 * 60 * 1000).toISOString(),
        period_end: new Date().toISOString(),
        summary: {
          total_alerts: Math.floor(Math.random() * 10),
          avg_cpu_usage: Math.random() * 100,
          avg_memory_usage: Math.random() * 100,
          max_tcp_connections: Math.floor(Math.random() * 2000) + 1000,
          uptime_percentage: 95 + Math.random() * 5
        },
        metrics: {
          cpu_trend: Array(7).fill(0).map(() => Math.random() * 100),
          memory_trend: Array(7).fill(0).map(() => Math.random() * 100),
          network_trend: Array(7).fill(0).map(() => Math.floor(Math.random() * 1000) + 1000),
          timestamps: type === 'daily' 
            ? ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
            : ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
        },
        alerts: [
          {
            timestamp: new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000).toISOString(),
            severity: ['low', 'medium', 'high'][Math.floor(Math.random() * 3)] as 'low' | 'medium' | 'high',
            category: ['CPU', 'Memory', 'Network', 'Disk'][Math.floor(Math.random() * 4)],
            message: 'System resource alert detected',
            resolved: Math.random() > 0.3
          }
        ],
        recommendations: [
          'Monitor system performance during peak hours',
          'Consider resource optimization strategies',
          'Review alert thresholds for accuracy'
        ]
      };
      
      setReports(prev => [newReport, ...prev]);
      console.log(`${type} report generated successfully`);
      
    } catch (error) {
      console.error('Failed to generate report:', error);
    } finally {
      setIsGenerating(null);
    }
  };

  return (
    <section className="reports-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          PLAYBACK REPORTS
        </h2>
        
        <div className="report-controls" style={{
          display: 'flex',
          gap: 'var(--spacing-sm)'
        }}>
          <button 
            className={`btn btn-action ${(isGenerating || isRefreshing) ? 'disabled' : ''}`}
            onClick={() => generateReport('daily')}
            disabled={isGenerating !== null || isRefreshing}
            style={{
              opacity: (isGenerating !== null || isRefreshing) ? 0.6 : 1,
              cursor: (isGenerating !== null || isRefreshing) ? 'not-allowed' : 'pointer'
            }}
          >
            {isGenerating === 'daily' ? (
              <>
                <Loader size={16} className="spinning-loader" />
                GENERATING...
              </>
            ) : (
              <>
                <BarChart size={16} />
                GENERATE DAILY
              </>
            )}
          </button>
          <button 
            className={`btn btn-action ${(isGenerating || isRefreshing) ? 'disabled' : ''}`}
            onClick={() => generateReport('weekly')}
            disabled={isGenerating !== null || isRefreshing}
            style={{
              opacity: (isGenerating !== null || isRefreshing) ? 0.6 : 1,
              cursor: (isGenerating !== null || isRefreshing) ? 'not-allowed' : 'pointer'
            }}
          >
            {isGenerating === 'weekly' ? (
              <>
                <Loader size={16} className="spinning-loader" />
                GENERATING...
              </>
            ) : (
              <>
                <TrendingUp size={16} />
                GENERATE WEEKLY
              </>
            )}
          </button>
          <button 
            className={`btn btn-action ${(isGenerating || isRefreshing) ? 'disabled' : ''}`}
            onClick={() => loadReports(true)}
            disabled={isGenerating !== null || isRefreshing}
            style={{
              opacity: (isGenerating !== null || isRefreshing) ? 0.6 : 1,
              cursor: (isGenerating !== null || isRefreshing) ? 'not-allowed' : 'pointer'
            }}
          >
            {isRefreshing ? (
              <>
                <Loader size={16} className="spinning-loader" />
                REFRESHING...
              </>
            ) : (
              <>
                <RefreshCw size={16} />
                REFRESH
              </>
            )}
          </button>
        </div>
      </div>
      
      <div className="reports-list">
        {loading ? (
          <LoadingSkeleton variant="card" count={2} />
        ) : reports.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO REPORTS AVAILABLE
          </div>
        ) : (
          reports.map(report => (
            <div key={report.id} className="report-item card" style={{
              padding: 'var(--spacing-md)',
              background: 'rgba(0, 0, 0, 0.5)',
              marginBottom: 'var(--spacing-md)',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)'
            }}>
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'flex-start',
                marginBottom: 'var(--spacing-md)'
              }}>
                <div>
                  <h4 style={{ 
                    margin: '0 0 var(--spacing-xs) 0', 
                    color: 'var(--color-text-bright)',
                    textTransform: 'uppercase'
                  }}>
                    {report.type} Report
                  </h4>
                  <p style={{
                    margin: 0,
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    Generated: {new Date(report.generated_at).toLocaleString()}
                  </p>
                </div>
                
                <span style={{
                  padding: 'var(--spacing-xs) var(--spacing-sm)',
                  background: 'var(--color-accent)',
                  color: 'var(--color-background)',
                  borderRadius: 'var(--border-radius-sm)',
                  fontSize: 'var(--font-size-xs)',
                  fontWeight: 'bold',
                  textTransform: 'uppercase'
                }}>
                  {report.type}
                </span>
              </div>
              
              <div className="report-summary" style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                gap: 'var(--spacing-md)',
                marginBottom: 'var(--spacing-md)'
              }}>
                <div className="summary-item">
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Avg CPU Usage:
                  </span>
                  <span style={{ color: 'var(--color-text-bright)', fontWeight: 'bold' }}>
                    {report.summary.avg_cpu_usage.toFixed(1)}%
                  </span>
                </div>
                
                <div className="summary-item">
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Avg Memory Usage:
                  </span>
                  <span style={{ color: 'var(--color-text-bright)', fontWeight: 'bold' }}>
                    {report.summary.avg_memory_usage.toFixed(1)}%
                  </span>
                </div>
                
                <div className="summary-item">
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Total Alerts:
                  </span>
                  <span style={{ 
                    color: report.summary.total_alerts > 0 ? 'var(--color-warning)' : 'var(--color-success)',
                    fontWeight: 'bold'
                  }}>
                    {report.summary.total_alerts}
                  </span>
                </div>
                
                <div className="summary-item">
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Uptime:
                  </span>
                  <span style={{ color: 'var(--color-success)', fontWeight: 'bold' }}>
                    {report.summary.uptime_percentage.toFixed(1)}%
                  </span>
                </div>
              </div>
              
              {report.recommendations.length > 0 && (
                <div className="recommendations">
                  <h5 style={{ 
                    margin: '0 0 var(--spacing-sm) 0',
                    color: 'var(--color-accent)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    Key Recommendations:
                  </h5>
                  <ul style={{ 
                    margin: 0, 
                    paddingLeft: 'var(--spacing-lg)',
                    color: 'var(--color-text)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    {report.recommendations.slice(0, 2).map((rec, index) => (
                      <li key={index} style={{ marginBottom: 'var(--spacing-xs)' }}>
                        {rec}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </section>
  );
};
