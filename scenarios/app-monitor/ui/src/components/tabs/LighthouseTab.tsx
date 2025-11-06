import { useState } from 'react';
import { Zap, TrendingUp, AlertCircle, Loader, ExternalLink, Play } from 'lucide-react';
import type { App } from '@/types';
import './LighthouseTab.css';

interface LighthouseTabProps {
  app: App | null | undefined;
  history: LighthouseHistory | null;
  loading: boolean;
  error: string | null;
  onRefetch?: () => Promise<void>;
}

interface LighthouseScore {
  performance: number;
  accessibility: number;
  'best-practices': number;
  seo?: number;
}

interface LighthouseReport {
  id: string;
  timestamp: string;
  page_id: string;
  page_label: string;
  url: string;
  viewport: string;
  status: string;
  scores: LighthouseScore;
  failures: Array<{ category: string; score: string; threshold: string; level: string }>;
  warnings: Array<{ category: string; score: string; threshold: string; level: string }>;
  report_url: string;
}

interface TrendPoint {
  timestamp: string;
  score: number;
  page_id: string;
}

interface LighthouseHistory {
  scenario: string;
  reports: LighthouseReport[];
  trend: {
    performance: TrendPoint[];
    accessibility: TrendPoint[];
    best_practices: TrendPoint[];
    seo: TrendPoint[];
  };
}

export default function LighthouseTab({ app, history, loading, error, onRefetch }: LighthouseTabProps) {
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);

  const scenarioName = app?.id || '';

  const runLighthouse = async () => {
    if (!scenarioName || running) return;

    setRunning(true);
    setRunError(null);

    try {
      const response = await fetch(`/api/v1/scenarios/${scenarioName}/lighthouse/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to run Lighthouse audits');
      }

      // Refresh history after run completes
      if (onRefetch) {
        await onRefetch();
      }
    } catch (err) {
      setRunError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setRunning(false);
    }
  };

  if (loading) {
    return (
      <div className="lighthouse-tab">
        <div className="lighthouse-tab__loading">
          <Loader size={32} className="lighthouse-tab__loading-icon" />
          <p>Loading Lighthouse reports...</p>
        </div>
      </div>
    );
  }

  if (error && !history) {
    return (
      <div className="lighthouse-tab">
        <div className="lighthouse-tab__error">
          <AlertCircle size={32} />
          <p>{error}</p>
          <p className="lighthouse-tab__error-hint">
            This scenario may not have a .lighthouse/config.json file configured.
          </p>
        </div>
      </div>
    );
  }

  if (!history || history.reports.length === 0) {
    return (
      <div className="lighthouse-tab">
        <div className="lighthouse-tab__empty">
          <Zap size={32} />
          <p>No Lighthouse reports available</p>
          {runError && (
            <div className="lighthouse-tab__error-banner" style={{ marginTop: '1rem', marginBottom: '1rem' }}>
              <AlertCircle size={16} />
              <span>{runError}</span>
            </div>
          )}
          <button
            onClick={runLighthouse}
            disabled={running}
            className="lighthouse-tab__run-button"
          >
            {running ? (
              <>
                <Loader size={16} className="lighthouse-tab__run-button-icon--spinning" />
                Running audits...
              </>
            ) : (
              <>
                <Play size={16} />
                Run Lighthouse Audits
              </>
            )}
          </button>
        </div>
      </div>
    );
  }

  // Group reports by page_id (latest per page)
  const latestReports = history.reports.reduce((acc, report) => {
    if (!acc[report.page_id] || new Date(report.timestamp) > new Date(acc[report.page_id].timestamp)) {
      acc[report.page_id] = report;
    }
    return acc;
  }, {} as Record<string, LighthouseReport>);

  const latestReportsList = Object.values(latestReports);

  return (
    <div className="lighthouse-tab">
      {/* Header with Run button */}
      <div className="lighthouse-tab__header">
        <div className="lighthouse-tab__header-info">
          <h3 className="lighthouse-tab__title">
            <Zap size={18} />
            <span>Lighthouse Performance</span>
          </h3>
          <p className="lighthouse-tab__subtitle">
            {history.reports.length} report{history.reports.length !== 1 ? 's' : ''} •{' '}
            Last run: {new Date(history.reports[0]?.timestamp).toLocaleString()}
          </p>
        </div>
        <button
          onClick={runLighthouse}
          disabled={running}
          className="lighthouse-tab__run-button lighthouse-tab__run-button--primary"
        >
          {running ? (
            <>
              <Loader size={16} className="lighthouse-tab__run-button-icon--spinning" />
              Running...
            </>
          ) : (
            <>
              <Play size={16} />
              Run Audits
            </>
          )}
        </button>
      </div>

      {(error || runError) && (
        <div className="lighthouse-tab__error-banner">
          <AlertCircle size={16} />
          <span>{error || runError}</span>
        </div>
      )}

      {/* Latest Scores Grid */}
      <section className="lighthouse-section">
        <h4 className="lighthouse-section__title">Latest Scores</h4>
        <div className="lighthouse-scores-grid">
          {latestReportsList.map((report) => (
            <div key={report.id} className="lighthouse-score-card">
              <div className="lighthouse-score-card__header">
                <span className="lighthouse-score-card__page">{report.page_id}</span>
                <span className="lighthouse-score-card__viewport">{report.viewport}</span>
              </div>
              <div className="lighthouse-score-card__scores">
                <ScoreBadge
                  label="Performance"
                  score={report.scores.performance}
                  icon="⚡"
                />
                <ScoreBadge
                  label="Accessibility"
                  score={report.scores.accessibility}
                  icon="♿"
                />
                <ScoreBadge
                  label="Best Practices"
                  score={report.scores['best-practices']}
                  icon="✓"
                />
                {report.scores.seo !== undefined && (
                  <ScoreBadge label="SEO" score={report.scores.seo} icon="🔍" />
                )}
              </div>
              <a
                href={report.report_url}
                target="_blank"
                rel="noopener noreferrer"
                className="lighthouse-score-card__link"
              >
                <span>View Full Report</span>
                <ExternalLink size={14} />
              </a>
            </div>
          ))}
        </div>
      </section>

      {/* Performance Trend */}
      {history.trend.performance.length > 1 && (
        <section className="lighthouse-section">
          <h4 className="lighthouse-section__title">
            <TrendingUp size={18} />
            <span>Performance Trend</span>
          </h4>
          <div className="lighthouse-trend">
            <TrendMiniChart data={history.trend.performance} />
          </div>
        </section>
      )}

      {/* Recent Reports Table */}
      <section className="lighthouse-section">
        <h4 className="lighthouse-section__title">Report History</h4>
        <div className="lighthouse-reports-table">
          <table>
            <thead>
              <tr>
                <th>Page</th>
                <th>Timestamp</th>
                <th>Performance</th>
                <th>Accessibility</th>
                <th>Best Practices</th>
                <th>Report</th>
              </tr>
            </thead>
            <tbody>
              {history.reports.slice(0, 10).map((report) => (
                <tr key={report.id}>
                  <td>
                    <div className="lighthouse-reports-table__page">
                      <span>{report.page_id}</span>
                      <span className="lighthouse-reports-table__viewport">
                        {report.viewport}
                      </span>
                    </div>
                  </td>
                  <td>{new Date(report.timestamp).toLocaleString()}</td>
                  <td>
                    <ScoreChip score={report.scores.performance} />
                  </td>
                  <td>
                    <ScoreChip score={report.scores.accessibility} />
                  </td>
                  <td>
                    <ScoreChip score={report.scores['best-practices']} />
                  </td>
                  <td>
                    <a
                      href={report.report_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="lighthouse-reports-table__link"
                    >
                      <ExternalLink size={14} />
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

// Score badge component with color coding
function ScoreBadge({ label, score, icon }: { label: string; score: number; icon: string }) {
  const percentage = Math.round(score * 100);
  const level = percentage >= 90 ? 'good' : percentage >= 50 ? 'average' : 'poor';

  return (
    <div className={`lighthouse-score-badge lighthouse-score-badge--${level}`}>
      <span className="lighthouse-score-badge__icon">{icon}</span>
      <div className="lighthouse-score-badge__content">
        <span className="lighthouse-score-badge__label">{label}</span>
        <span className="lighthouse-score-badge__score">{percentage}</span>
      </div>
    </div>
  );
}

// Score chip for table
function ScoreChip({ score }: { score: number }) {
  const percentage = Math.round(score * 100);
  const level = percentage >= 90 ? 'good' : percentage >= 50 ? 'average' : 'poor';

  return (
    <span className={`lighthouse-score-chip lighthouse-score-chip--${level}`}>
      {percentage}
    </span>
  );
}

// Mini trend chart (simplified ASCII-style visualization)
function TrendMiniChart({ data }: { data: TrendPoint[] }) {
  if (data.length === 0) return null;

  // Group by page_id and take last 10 points per page
  const byPage = data.reduce((acc, point) => {
    if (!acc[point.page_id]) acc[point.page_id] = [];
    acc[point.page_id].push(point);
    return acc;
  }, {} as Record<string, TrendPoint[]>);

  return (
    <div className="lighthouse-mini-chart">
      {Object.entries(byPage).map(([pageId, points]) => {
        const recent = points.slice(-10);
        const scores = recent.map((p) => p.score);
        const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
        const trend = scores.length > 1 ? scores[scores.length - 1] - scores[0] : 0;

        return (
          <div key={pageId} className="lighthouse-mini-chart__row">
            <span className="lighthouse-mini-chart__label">{pageId}</span>
            <div className="lighthouse-mini-chart__bars">
              {scores.map((score, idx) => (
                <div
                  key={idx}
                  className="lighthouse-mini-chart__bar"
                  style={{
                    height: `${score * 100}%`,
                    backgroundColor:
                      score >= 0.9 ? '#0cce6b' : score >= 0.5 ? '#ffa400' : '#ff4e42',
                  }}
                  title={`${Math.round(score * 100)}%`}
                />
              ))}
            </div>
            <div className="lighthouse-mini-chart__stats">
              <span className="lighthouse-mini-chart__avg">{Math.round(avg * 100)}%</span>
              {trend !== 0 && (
                <span
                  className={`lighthouse-mini-chart__trend ${trend > 0 ? 'lighthouse-mini-chart__trend--up' : 'lighthouse-mini-chart__trend--down'}`}
                >
                  {trend > 0 ? '↑' : '↓'} {Math.abs(Math.round(trend * 100))}%
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
