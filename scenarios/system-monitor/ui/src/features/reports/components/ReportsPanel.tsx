import { useState, useEffect, useRef, useMemo } from 'react';
import { BarChart, TrendingUp, RefreshCw, Loader } from 'lucide-react';
import type { EnhancedSystemReport } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { extractErrorMessage, protoFetch } from '../../../shared/api/apiFetch';
import { parseListReportsResponse, parseGenerateReportResponse, parseGetReportResponse } from '../../../shared/api/proto-contracts';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { ReportInline } from '../../../shared/report/ReportBody';

type ReportsLoadState =
  | { kind: 'loading' }
  | { kind: 'ready' }
  | { kind: 'refreshing' }
  | { kind: 'failed'; message: string };

type ReportGenerationState =
  | { kind: 'idle' }
  | { kind: 'generating'; reportType: 'daily' | 'weekly'; requestId: string }
  | { kind: 'unknown'; reportType: 'daily' | 'weekly'; requestId: string; message: string };

type ReportDetailState =
  | { kind: 'closed' }
  | { kind: 'loading'; reportId: string }
  | { kind: 'ready'; reportId: string; report: EnhancedSystemReport }
  | { kind: 'failed'; reportId: string; message: string };

export const ReportsPanel = () => {
  const [reports, setReports] = useState<EnhancedSystemReport[]>([]);
  const [loadState, setLoadState] = useState<ReportsLoadState>({ kind: 'loading' });
  const [generationState, setGenerationState] = useState<ReportGenerationState>({ kind: 'idle' });
  const [search, setSearch] = useState('');
  const [detail, setDetail] = useState<ReportDetailState>({ kind: 'closed' });
  const isRefreshing = loadState.kind === 'refreshing';
  const generatingType = generationState.kind === 'generating' ? generationState.reportType : null;
  const isGenerating = generatingType !== null;
  const hasUnknownGeneration = generationState.kind === 'unknown';
  const filteredReports = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return reports;
    return reports.filter(report => {
      const searchable = [
        report.reportId, report.reportType, report.dateRangeDisplay, report.actualDuration,
        report.executiveSummary?.overallHealth, report.executiveSummary?.timeDescription,
        ...(report.executiveSummary?.keyFindings ?? []), ...report.recommendations, ...report.highlights,
        ...report.trends.map(trend => `${trend.name} ${trend.direction} ${trend.changePercent}%`),
      ];
      return searchable.some(value => String(value ?? '').toLocaleLowerCase().includes(query));
    });
  }, [reports, search]);
  const mountedRef = useRef(true);
  const reportListRequestRef = useRef(0);
  const detailRequestRef = useRef(0);
  const detailHeadingRef = useRef<HTMLHeadingElement>(null);
  const detailTriggerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  useEffect(() => {
    if (detail.kind === 'closed') return;
    const frame = window.requestAnimationFrame(() => {
      detailHeadingRef.current?.scrollIntoView({ block: 'start', behavior: 'instant' });
      detailHeadingRef.current?.focus({ preventScroll: true });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [detail.kind, detail.kind === 'closed' ? '' : detail.reportId]);

  const loadReports = async (isRefresh = false) => {
    const requestId = ++reportListRequestRef.current;
    setLoadState({ kind: isRefresh ? 'refreshing' : 'loading' });
    try {
      const resp = await protoFetch('/reports', parseListReportsResponse);
      if (!mountedRef.current || requestId !== reportListRequestRef.current) return;
      setReports(resp.reports ?? []);
      setLoadState({ kind: 'ready' });
    } catch (err) {
      if (mountedRef.current && requestId === reportListRequestRef.current) {
        setLoadState({ kind: 'failed', message: extractErrorMessage(err, 'Failed to load reports') });
      }
    }
  };

  useEffect(() => {
    void loadReports();
  }, []);

  const generateReport = async (type: 'daily' | 'weekly') => {
    if (isGenerating || isRefreshing || hasUnknownGeneration) return;
    const requestId = `report-${type}-${crypto.randomUUID()}`;
    let outcomeUnknown = false;
    setGenerationState({ kind: 'generating', reportType: type, requestId });
    
    try {
      const response = await protoFetch('/reports/generate', parseGenerateReportResponse, {
        method: 'POST',
        body: JSON.stringify({ type }),
      });

      if (!mountedRef.current) return;
      if (response.report) {
        const generated = response.report;
        setReports(previous => [generated, ...previous.filter(report => report.reportId !== generated.reportId)]);
        setGenerationState({ kind: 'idle' });
      } else {
        outcomeUnknown = true;
        setGenerationState({ kind: 'unknown', reportType: type, requestId, message: 'The request completed without a report identity. Refresh the report list to reconcile before generating another.' });
        await loadReports(true);
        return;
      }
      // Reload the full list to get consistent data
      await loadReports(true);
    } catch {
      if (mountedRef.current) {
        outcomeUnknown = true;
        setGenerationState({ kind: 'unknown', reportType: type, requestId, message: 'The response was not received. Refresh the report list and reconcile the generated report before trying again.' });
        await loadReports(true);
      }
    } finally {
      if (mountedRef.current && !outcomeUnknown) {
        setGenerationState({ kind: 'idle' });
      }
    }
  };

  const openReport = async (reportId: string, trigger?: HTMLButtonElement) => {
    if (trigger) detailTriggerRef.current = trigger;
    const requestId = ++detailRequestRef.current;
    setDetail({ kind: 'loading', reportId });
    try {
      const response = await protoFetch(`/reports/${encodeURIComponent(reportId)}`, parseGetReportResponse);
      if (!mountedRef.current || requestId !== detailRequestRef.current) return;
      if (!response.report) throw new Error('The report service returned no report for this identity.');
      setDetail({ kind: 'ready', reportId, report: response.report });
    } catch (err) {
      if (mountedRef.current && requestId === detailRequestRef.current) setDetail({ kind: 'failed', reportId, message: extractErrorMessage(err, 'Unable to load report details') });
    }
  };

  return (
    <section className="reports-panel card">
      <div className="flex-row-between mb-md">
        <h2 className="section-heading">
          REPORTS
        </h2>

        <div className="flex-row-center gap-sm">
          <button
            className="btn btn-action"
            onClick={() => { void generateReport('daily'); }}
            disabled={isGenerating || isRefreshing || hasUnknownGeneration}
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
            disabled={isGenerating || isRefreshing || hasUnknownGeneration}
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
      <p className="reports-description">Daily and weekly summaries of collected telemetry. These are separate from investigation findings.</p>

      {generationState.kind === 'unknown' && (
        <div className="error-banner text-sm report-generation-unknown" role="status">
          <p>{generationState.message} Local attempt ID: {generationState.requestId}. A successful list refresh does not prove that the original request failed.</p>
          <p>Review the refreshed reports above before clearing this hold. The report service does not support an idempotency key for this attempt.</p>
          <button
            type="button"
            className="btn btn-action"
            disabled={isRefreshing || loadState.kind !== 'ready'}
            onClick={() => { setGenerationState({ kind: 'idle' }); }}
          >
            I reviewed the list; allow another request
          </button>
        </div>
      )}
      
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
          <>
            <p className="reports-count" role="status">{filteredReports.length} of {reports.length} saved {reports.length === 1 ? 'report' : 'reports'}</p>
            <label className="report-search-label" htmlFor="report-search">Search reports</label>
            <input id="report-search" className="form-control report-search" type="search" value={search} onChange={event => setSearch(event.target.value)} placeholder="Type, date, finding, trend, or recommendation" />
            {filteredReports.length === 0 ? <div className="text-center text-muted p-lg" role="status">No reports match “{search.trim()}”. <button type="button" className="btn btn-action" onClick={() => setSearch('')}>Clear search</button></div> : filteredReports.map(report => (
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
              <button type="button" className="btn btn-action report-view-details" onClick={event => { void openReport(report.reportId, event.currentTarget); }} aria-label={`View details for ${report.reportType} report from ${report.dateRangeDisplay || 'unknown date'}`}>
                View report details
              </button>
            </div>
            ))}
          </>
        )}
      </div>
      {detail.kind !== 'closed' && <section className="report-detail-inline card" role="region" aria-labelledby="report-detail-title">
          <div className="flex-row-between mb-md">
            <h3 id="report-detail-title" ref={detailHeadingRef} tabIndex={-1} className="section-heading">{detail.kind === 'ready' ? `${detail.report.reportType} report details` : 'Report details'}</h3>
            <button type="button" className="btn btn-action" onClick={() => {
              detailRequestRef.current += 1;
              setDetail({ kind: 'closed' });
              requestAnimationFrame(() => {
                if (detailTriggerRef.current?.isConnected) detailTriggerRef.current.focus();
                else document.getElementById('report-search')?.focus();
              });
            }}>Close report</button>
          </div>
          {detail.kind === 'loading' ? <p role="status">Loading the saved report…</p> : detail.kind === 'failed' ? <div role="alert" className="error-banner"><p>Report details could not be loaded: {detail.message}</p><button type="button" className="btn btn-action" onClick={() => { void openReport(detail.reportId); }}>Retry details</button></div> : <ReportDetail report={detail.report} />}
      </section>}
    </section>
  );
};

function ReportDetail({ report }: { report: EnhancedSystemReport }) {
  return <div className="report-detail-content">
    <dl className="detail-grid detail-grid-md">
      <div><dt>Report identity</dt><dd><code>{report.reportId}</code></dd></div>
      <div><dt>Generated</dt><dd>{report.generatedAt ? timestampDate(report.generatedAt).toLocaleString() : 'Time not reported'}</dd></div>
      <div><dt>Coverage</dt><dd>{report.dateRangeDisplay || report.actualDuration || 'Range not reported'}</dd></div>
      <div><dt>Health</dt><dd>{report.executiveSummary?.overallHealth || 'Not reported'}</dd></div>
      <div><dt>Alerts in period</dt><dd>{report.alertsCount}</dd></div>
      <div><dt>Recommendations</dt><dd>{report.recommendations.length}</dd></div>
      <div><dt>Investigations</dt><dd>{report.investigationsCount}</dd></div>
      <div><dt>Metrics analyzed</dt><dd>{report.metricsCount}</dd></div>
    </dl>
    {report.performance && <section><h4 className="section-heading">Performance summary</h4><p className="text-muted">{report.performance.timeRange || report.actualDuration || 'Report period'}</p><div className="detail-grid detail-grid-md">
      {report.performance.cpu && <MetricStatsSummary label="CPU" stats={report.performance.cpu} />}
      {report.performance.memory && <MetricStatsSummary label="Memory" stats={report.performance.memory} />}
    </div></section>}
    {report.executiveSummary?.keyFindings?.length ? <section><h4 className="section-heading">Key findings</h4><ul>{report.executiveSummary.keyFindings.map((finding, index) => <li key={`${index}-${finding}`}><ReportInline text={finding} /></li>)}</ul></section> : null}
    {report.highlights.length > 0 && <section><h4 className="section-heading">Highlights</h4><ul>{report.highlights.map((item, index) => <li key={`${index}-${item}`}><ReportInline text={item} /></li>)}</ul></section>}
    {report.trends.length > 0 && <section><h4 className="section-heading">Metric trends</h4><ul>{report.trends.map((trend, index) => <li key={`${index}-${trend.name}`}><strong>{trend.name}</strong>: {trend.direction}, {trend.changePercent.toFixed(1)}% change ({trend.change.toFixed(1)})</li>)}</ul></section>}
    {report.recommendations.length > 0 && <section><h4 className="section-heading">Recommendations</h4><ul>{report.recommendations.map((item, index) => <li key={`${index}-${item}`}><ReportInline text={item} /></li>)}</ul></section>}
    <p className="text-muted text-sm">Read-only saved report. Opening details does not generate or replay a report.</p>
  </div>;
}

function MetricStatsSummary({ label, stats }: { label: string; stats: NonNullable<EnhancedSystemReport['performance']>['cpu'] }) {
  if (!stats) return null;
  return <div className="report-metric-stats card">
    <h5 className="section-heading">{label}</h5>
    <dl>
      <div><dt>Average</dt><dd>{stats.average.toFixed(1)}%</dd></div>
      <div><dt>Minimum</dt><dd>{stats.min.toFixed(1)}%</dd></div>
      <div><dt>Maximum</dt><dd>{stats.max.toFixed(1)}%</dd></div>
      <div><dt>Standard deviation</dt><dd>{stats.stdDev.toFixed(1)} percentage points</dd></div>
      <div><dt>Peak</dt><dd>{stats.peakValue.toFixed(1)}%{stats.peakTime ? ` · ${timestampDate(stats.peakTime).toLocaleString()}` : ''}</dd></div>
    </dl>
  </div>;
}
