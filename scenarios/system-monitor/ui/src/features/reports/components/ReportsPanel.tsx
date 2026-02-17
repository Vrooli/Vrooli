import { useState, useEffect, useRef } from 'react';
import { BarChart, TrendingUp, RefreshCw, Loader } from 'lucide-react';
import type { SystemReport } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { timestampDate, timestampFromDate } from '@bufbuild/protobuf/wkt';

export const ReportsPanel = () => {
  const [reports, setReports] = useState<SystemReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

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
      
      if (!mountedRef.current) return;
      setReports([
        {
          id: 'daily-' + Date.now(),
          type: 'daily',
          generatedAt: timestampFromDate(new Date()),
          periodStart: timestampFromDate(new Date(Date.now() - 24 * 60 * 60 * 1000)),
          periodEnd: timestampFromDate(new Date()),
          summary: {
            totalAlerts: 3,
            avgCpuUsage: 45.2,
            avgMemoryUsage: 67.8,
            maxTcpConnections: 1247,
            uptimePercentage: 99.8
          },
          metrics: {
            cpuTrend: [40, 42, 45, 48, 46, 44, 45],
            memoryTrend: [65, 66, 68, 70, 69, 67, 68],
            networkTrend: [1200, 1180, 1220, 1247, 1230, 1210, 1225],
            timestamps: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
          },
          alerts: [
            {
              timestamp: timestampFromDate(new Date(Date.now() - 6 * 60 * 60 * 1000)),
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
        } as SystemReport
      ]);
    } catch (error) {
      console.error('Failed to load reports:', error);
    } finally {
      if (mountedRef.current) {
        if (isRefresh) {
          setIsRefreshing(false);
        } else {
          setLoading(false);
        }
      }
    }
  };

  useEffect(() => {
    void loadReports();
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
        generatedAt: timestampFromDate(new Date()),
        periodStart: timestampFromDate(new Date(Date.now() - (type === 'daily' ? 24 : 168) * 60 * 60 * 1000)),
        periodEnd: timestampFromDate(new Date()),
        summary: {
          totalAlerts: Math.floor(Math.random() * 10),
          avgCpuUsage: Math.random() * 100,
          avgMemoryUsage: Math.random() * 100,
          maxTcpConnections: Math.floor(Math.random() * 2000) + 1000,
          uptimePercentage: 95 + Math.random() * 5
        },
        metrics: {
          cpuTrend: Array(7).fill(0).map(() => Math.random() * 100),
          memoryTrend: Array(7).fill(0).map(() => Math.random() * 100),
          networkTrend: Array(7).fill(0).map(() => Math.floor(Math.random() * 1000) + 1000),
          timestamps: type === 'daily'
            ? ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
            : ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
        },
        alerts: [
          {
            timestamp: timestampFromDate(new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000)),
            severity: (['low', 'medium', 'high'] as const)[Math.floor(Math.random() * 3)] ?? 'low',
            category: (['CPU', 'Memory', 'Network', 'Disk'] as const)[Math.floor(Math.random() * 4)] ?? 'CPU',
            message: 'System resource alert detected',
            resolved: Math.random() > 0.3
          }
        ],
        recommendations: [
          'Monitor system performance during peak hours',
          'Consider resource optimization strategies',
          'Review alert thresholds for accuracy'
        ]
      } as SystemReport;

      if (!mountedRef.current) return;
      setReports(prev => [newReport, ...prev]);
      console.log(`${type} report generated successfully`);

    } catch (error) {
      console.error('Failed to generate report:', error);
    } finally {
      if (mountedRef.current) {
        setIsGenerating(null);
      }
    }
  };

  return (
    <section className="reports-panel card">
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">
          PLAYBACK REPORTS
        </h2>

        <div className="flex-row-center gap-sm">
          <button
            className="btn btn-action"
            onClick={() => generateReport('daily')}
            disabled={isGenerating !== null || isRefreshing}
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
            className="btn btn-action"
            onClick={() => generateReport('weekly')}
            disabled={isGenerating !== null || isRefreshing}
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
            className="btn btn-action"
            onClick={() => loadReports(true)}
            disabled={isGenerating !== null || isRefreshing}
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
          <div className="text-center text-muted p-lg text-lg">
            NO REPORTS AVAILABLE
          </div>
        ) : (
          reports.map(report => (
            <div key={report.id} className="report-item section-card mb-md" style={{ cursor: 'pointer' }}>
              <div className="flex-row-between mb-md" style={{ alignItems: 'flex-start' }}>
                <div>
                  <h4 className="section-heading" style={{ textTransform: 'uppercase' }}>
                    {report.type} Report
                  </h4>
                  <p className="text-muted text-sm" style={{ margin: 0 }}>
                    Generated: {report.generatedAt ? timestampDate(report.generatedAt).toLocaleString() : 'Unknown'}
                  </p>
                </div>

                <span className="badge badge-success">
                  {report.type}
                </span>
              </div>

              <div className="report-summary detail-grid detail-grid-md mb-md">
                <div className="summary-stat">
                  <span className="summary-stat-label">Avg CPU Usage:</span>
                  <span className="summary-stat-value" style={{ color: 'var(--color-text-bright)' }}>
                    {report.summary?.avgCpuUsage.toFixed(1)}%
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Avg Memory Usage:</span>
                  <span className="summary-stat-value" style={{ color: 'var(--color-text-bright)' }}>
                    {report.summary?.avgMemoryUsage.toFixed(1)}%
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Total Alerts:</span>
                  <span className="summary-stat-value" style={{
                    color: (report.summary?.totalAlerts ?? 0) > 0 ? 'var(--color-warning)' : 'var(--color-success)'
                  }}>
                    {report.summary?.totalAlerts}
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Uptime:</span>
                  <span className="summary-stat-value" style={{ color: 'var(--color-success)' }}>
                    {report.summary?.uptimePercentage.toFixed(1)}%
                  </span>
                </div>
              </div>
              
              {report.recommendations.length > 0 && (
                <div className="recommendations">
                  <h5 className="text-accent text-sm" style={{ margin: '0 0 var(--spacing-sm) 0' }}>
                    Key Recommendations:
                  </h5>
                  <ul className="text-sm" style={{ margin: 0, paddingLeft: 'var(--spacing-lg)' }}>
                    {report.recommendations.slice(0, 2).map((rec, index) => (
                      <li key={index} className="mb-sm">
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
