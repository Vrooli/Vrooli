import { useState, useEffect, useRef } from 'react';
import { BarChart, TrendingUp, RefreshCw, Loader } from 'lucide-react';
import type { SystemReport } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { useToast } from '../../../shared/components/ToastProvider';
import { extractErrorMessage, protoFetch } from '../../../shared/api/apiFetch';
import { parseListReportsResponse, parseGenerateReportResponse } from '../../../shared/api/proto-contracts';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { ReportInline } from '../../../shared/report/ReportBody';

export const ReportsPanel = () => {
  const { showApiError } = useToast();
  const [reports, setReports] = useState<SystemReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
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
      setError(null);
      const resp = await protoFetch('/reports', parseListReportsResponse);
      if (!mountedRef.current) return;
      setReports((resp.reports ?? []) as unknown as SystemReport[]);
    } catch (err) {
      if (mountedRef.current) {
        setError(extractErrorMessage(err, 'Failed to load reports'));
      }
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
      await protoFetch('/reports/generate', parseGenerateReportResponse, {
        method: 'POST',
        body: JSON.stringify({ type }),
      });

      if (!mountedRef.current) return;
      // Reload the full list to get consistent data
      await loadReports(true);
    } catch (err) {
      showApiError(err);
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
            onClick={() => { void generateReport('daily'); }}
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
            onClick={() => { void generateReport('weekly'); }}
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
            onClick={() => { void loadReports(true); }}
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
        ) : error && !loading ? (
          <div className="error-banner" data-sm-style="sm-style-c220bf04e8">
            FAILED TO LOAD REPORTS
            <br />
            <span data-sm-style="sm-style-634e37ebe1">{error}</span>
            <br />
            <button type="button" className="btn btn-action"
              onClick={() => { void loadReports(); }}
              data-sm-style="sm-style-9ed1054ccd">
              <RefreshCw size={14} /> RETRY
            </button>
          </div>
        ) : reports.length === 0 ? (
          <div className="text-center text-muted p-lg text-lg">
            NO REPORTS AVAILABLE
          </div>
        ) : (
          reports.map(report => (
            <div key={report.id} className="report-item section-card mb-md" data-sm-style="sm-style-667fe49ecb">
              <div className="flex-row-between mb-md" data-sm-style="sm-style-b27fbbaf32">
                <div>
                  <h4 className="section-heading" data-sm-style="sm-style-40e6416985">
                    {report.type} Report
                  </h4>
                  <p className="text-muted text-sm" data-sm-style="sm-style-2a0ca8350a">
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
                  <span className="summary-stat-value" data-sm-style="sm-style-dbed1e5364">
                    {report.summary?.avgCpuUsage.toFixed(1)}%
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Avg Memory Usage:</span>
                  <span className="summary-stat-value" data-sm-style="sm-style-dbed1e5364">
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
                  <span className="summary-stat-value" data-sm-style="sm-style-eab9fc4afc">
                    {report.summary?.uptimePercentage.toFixed(1)}%
                  </span>
                </div>
              </div>
              
              {report.recommendations.length > 0 && (
                <div className="recommendations">
                  <h5 className="text-accent text-sm" data-sm-style="sm-style-fef0e1ff97">
                    Key Recommendations:
                  </h5>
                  <ul className="text-sm" data-sm-style="sm-style-59a5e35c0f">
                    {report.recommendations.slice(0, 2).map((rec, index) => (
                      <li key={index} className="mb-sm">
                        <ReportInline text={rec} />
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
