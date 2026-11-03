import { useState, useEffect, useRef } from 'react';
import { BarChart, TrendingUp, RefreshCw, Loader } from 'lucide-react';
import type { EnhancedSystemReport } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { useToast } from '../../../shared/components/ToastProvider';
import { extractErrorMessage, protoFetch } from '../../../shared/api/apiFetch';
import { parseListReportsResponse, parseGenerateReportResponse } from '../../../shared/api/proto-contracts';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { ReportInline } from '../../../shared/report/ReportBody';

type ReportsLoadState =
  | { kind: 'loading' }
  | { kind: 'ready' }
  | { kind: 'refreshing' }
  | { kind: 'failed'; message: string };

type ReportGenerationState =
  | { kind: 'idle' }
  | { kind: 'generating'; reportType: 'daily' | 'weekly' };

export const ReportsPanel = () => {
  const { showApiError } = useToast();
  const [reports, setReports] = useState<EnhancedSystemReport[]>([]);
  const [loadState, setLoadState] = useState<ReportsLoadState>({ kind: 'loading' });
  const [generationState, setGenerationState] = useState<ReportGenerationState>({ kind: 'idle' });
  const isRefreshing = loadState.kind === 'refreshing';
  const generatingType = generationState.kind === 'generating' ? generationState.reportType : null;
  const isGenerating = generatingType !== null;
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  const loadReports = async (isRefresh = false) => {
    setLoadState({ kind: isRefresh ? 'refreshing' : 'loading' });
    try {
      const resp = await protoFetch('/reports', parseListReportsResponse);
      if (!mountedRef.current) return;
      setReports(resp.reports ?? []);
      setLoadState({ kind: 'ready' });
    } catch (err) {
      if (mountedRef.current) {
        setLoadState({ kind: 'failed', message: extractErrorMessage(err, 'Failed to load reports') });
      }
    }
  };

  useEffect(() => {
    void loadReports();
  }, []);

  const generateReport = async (type: 'daily' | 'weekly') => {
    if (isGenerating || isRefreshing) return;
    setGenerationState({ kind: 'generating', reportType: type });
    
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
        setGenerationState({ kind: 'idle' });
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
            disabled={isGenerating || isRefreshing}
          >
            {generatingType === 'daily' ? (
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
            disabled={isGenerating || isRefreshing}
          >
            {generatingType === 'weekly' ? (
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
            disabled={isGenerating || isRefreshing}
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
        {loadState.kind === 'loading' ? (
          <LoadingSkeleton variant="card" count={2} />
        ) : loadState.kind === 'failed' ? (
          <div className="error-banner" data-sm-style="sm-style-c220bf04e8">
            FAILED TO LOAD REPORTS
            <br />
            <span data-sm-style="sm-style-634e37ebe1">{loadState.message}</span>
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
            <div key={report.reportId} className="report-item section-card mb-md" data-sm-style="sm-style-667fe49ecb">
              <div className="flex-row-between mb-md" data-sm-style="sm-style-b27fbbaf32">
                <div>
                  <h4 className="section-heading" data-sm-style="sm-style-40e6416985">
                    {report.reportType} Report
                  </h4>
                  <p className="text-muted text-sm" data-sm-style="sm-style-2a0ca8350a">
                    Generated: {report.generatedAt ? timestampDate(report.generatedAt).toLocaleString() : 'Unknown'}
                  </p>
                </div>

                <span className="badge badge-success">
                  {report.reportType}
                </span>
              </div>

              <div className="report-summary detail-grid detail-grid-md mb-md">
                <div className="summary-stat">
                  <span className="summary-stat-label">Avg CPU Usage:</span>
                  <span className="summary-stat-value" data-sm-style="sm-style-dbed1e5364">
                    {report.performance?.cpu?.average.toFixed(1) ?? '—'}%
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Avg Memory Usage:</span>
                  <span className="summary-stat-value" data-sm-style="sm-style-dbed1e5364">
                    {report.performance?.memory?.average.toFixed(1) ?? '—'}%
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Total Alerts:</span>
                  <span className="summary-stat-value" style={{
                    color: report.alertsCount > 0 ? 'var(--color-warning)' : 'var(--color-success)'
                  }}>
                    {report.alertsCount}
                  </span>
                </div>

                <div className="summary-stat">
                  <span className="summary-stat-label">Overall Health:</span>
                  <span className="summary-stat-value" data-sm-style="sm-style-eab9fc4afc">
                    {report.executiveSummary?.overallHealth || 'Unknown'}
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
