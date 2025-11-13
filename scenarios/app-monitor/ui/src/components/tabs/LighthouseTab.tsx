import { useEffect, useState } from 'react';
import { Zap, TrendingUp, AlertCircle, Loader, ExternalLink, Play, CheckSquare, Square } from 'lucide-react';
import type { App } from '@/types';
import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import { appService, type LighthouseMissingScenario, type ReportIssuePayload } from '@/services/api';
import { useSnackPublisher } from '@/notifications/useSnackPublisher';
import type { SnackPublisherApi } from '@/notifications/snackPublisherContext';
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
  const [missingConfigInfo, setMissingConfigInfo] = useState<MissingConfigMeta | null>(null);
  const [requestingIssue, setRequestingIssue] = useState(false);
  const [bulkDialogOpen, setBulkDialogOpen] = useState(false);
  const snackPublisher = useSnackPublisher();

  const scenarioName = app?.id || '';

  useEffect(() => {
    setMissingConfigInfo(null);
  }, [scenarioName]);

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
        const errorData = await response.json().catch(() => ({}));
        if (response.status === 404 && errorData?.missing_config) {
          setMissingConfigInfo({
            scenario: scenarioName,
            expectedPath: typeof errorData.expected_path === 'string' ? errorData.expected_path : undefined,
            hint: typeof errorData.hint === 'string' ? errorData.hint : undefined,
          });
        }
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

  useEffect(() => {
    if (!error || !scenarioName) {
      return;
    }
    if (/lighthouse/i.test(error)) {
      setMissingConfigInfo(current => current ?? {
        scenario: scenarioName,
        expectedPath: `scenarios/${scenarioName}/.vrooli/lighthouse.json`,
      });
    }
  }, [error, scenarioName]);

  const handleRequestSetupIssue = async () => {
    if (!scenarioName || requestingIssue) {
      return;
    }

    const info = missingConfigInfo;
    const friendlyName = app?.name ?? scenarioName;
    const lines = [
      `Scenario ${scenarioName} is missing Lighthouse coverage (.vrooli/lighthouse.json).`,
    ];
    if (info?.expectedPath) {
      lines.push(`Expected config path: ${info.expectedPath}`);
    }
    lines.push('Please initialize the config with scripts/scenarios/testing/lighthouse/config.sh init.');

    const payload: ReportIssuePayload = {
      message: lines.join('\n'),
      primaryDescription: `Add Lighthouse config for ${friendlyName}`,
      source: 'app-monitor:lighthouse',
      appName: friendlyName,
      scenarioName: app?.scenario_name ?? scenarioName,
    };

    try {
      setRequestingIssue(true);
      const response = await appService.reportAppIssue(scenarioName, payload);
      const issueId = response.data?.issue_id;
      snackPublisher.publish({
        variant: 'success',
        title: 'Issue created',
        message: issueId
          ? `Requested Lighthouse setup (issue ${issueId}).`
          : 'Requested Lighthouse setup.',
        autoDismiss: true,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to file issue.';
      snackPublisher.publish({
        variant: 'error',
        title: 'Request failed',
        message,
        autoDismiss: false,
      });
    } finally {
      setRequestingIssue(false);
    }
  };

  const showMissingConfigActions = Boolean(missingConfigInfo && scenarioName);

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
            {missingConfigInfo?.expectedPath
              ? `Missing Lighthouse config at ${missingConfigInfo.expectedPath}`
              : 'This scenario does not appear to have a Lighthouse config yet (.vrooli/lighthouse.json).'}
          </p>
          {showMissingConfigActions && (
            <MissingConfigActions
              requestingIssue={requestingIssue}
              onRequestIssue={handleRequestSetupIssue}
              onOpenBulk={() => setBulkDialogOpen(true)}
            />
          )}
        </div>
        <BulkMissingConfigDialog
          isOpen={bulkDialogOpen}
          onClose={() => setBulkDialogOpen(false)}
          snackPublisher={snackPublisher}
        />
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
          {showMissingConfigActions && (
            <MissingConfigActions
              requestingIssue={requestingIssue}
              onRequestIssue={handleRequestSetupIssue}
              onOpenBulk={() => setBulkDialogOpen(true)}
            />
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
        <BulkMissingConfigDialog
          isOpen={bulkDialogOpen}
          onClose={() => setBulkDialogOpen(false)}
          snackPublisher={snackPublisher}
        />
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
      <BulkMissingConfigDialog
        isOpen={bulkDialogOpen}
        onClose={() => setBulkDialogOpen(false)}
        snackPublisher={snackPublisher}
      />
    </div>
  );
}

function MissingConfigActions({
  requestingIssue,
  onRequestIssue,
  onOpenBulk,
}: {
  requestingIssue: boolean;
  onRequestIssue: () => void;
  onOpenBulk: () => void;
}) {
  return (
    <div className="lighthouse-tab__error-actions">
      <button
        className="lighthouse-tab__error-button"
        onClick={onRequestIssue}
        disabled={requestingIssue}
      >
        {requestingIssue ? (
          <>
            <Loader size={16} className="lighthouse-tab__run-button-icon--spinning" />
            Filing issue...
          </>
        ) : (
          <>
            <AlertCircle size={16} />
            Request setup issue
          </>
        )}
      </button>
      <button
        className="lighthouse-tab__error-button lighthouse-tab__error-button--secondary"
        onClick={onOpenBulk}
      >
        <Play size={16} />
        Bulk request
      </button>
    </div>
  );
}

interface MissingConfigMeta {
  scenario: string;
  expectedPath?: string;
  hint?: string;
}

interface BulkDialogProps {
  isOpen: boolean;
  onClose: () => void;
  snackPublisher: SnackPublisherApi;
}

function BulkMissingConfigDialog({ isOpen, onClose, snackPublisher }: BulkDialogProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [scenarios, setScenarios] = useState<LighthouseMissingScenario[]>([]);
  const [selection, setSelection] = useState<Set<string>>(new Set());
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!isOpen) {
      return;
    }
    let isCancelled = false;
    setLoading(true);
    setError(null);
    appService
      .listScenariosMissingLighthouseConfigs()
      .then((list) => {
        if (isCancelled) {
          return;
        }
        setScenarios(list);
        setSelection(new Set(list.map(item => item.scenario)));
      })
      .catch((err) => {
        if (isCancelled) {
          return;
        }
        const message = err instanceof Error ? err.message : 'Failed to load scenarios.';
        setError(message);
      })
      .finally(() => {
        if (!isCancelled) {
          setLoading(false);
        }
      });

    return () => {
      isCancelled = true;
    };
  }, [isOpen]);

  const toggleScenario = (name: string) => {
    setSelection((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  const handleSelectAll = () => {
    setSelection(new Set(scenarios.map(item => item.scenario)));
  };

  const handleSelectNone = () => {
    setSelection(new Set());
  };

  const handleSubmit = async () => {
    if (selection.size === 0) {
      setError('Select at least one scenario to include in the issue.');
      return;
    }

    const selected = Array.from(selection).sort();
    const description = `Add .vrooli/lighthouse.json for ${selected.length} scenario(s).`;
    const lines = [description, '', 'Scenarios requiring setup:', ...selected.map(name => `- ${name}`), '', 'Recommended command:', 'scripts/scenarios/testing/lighthouse/config.sh init'];

    const payload: ReportIssuePayload = {
      message: lines.join('\n'),
      primaryDescription: description,
      source: 'app-monitor:lighthouse-bulk',
      appName: 'App Monitor',
      scenarioName: 'app-monitor',
    };

    try {
      setSubmitting(true);
      const response = await appService.reportAppIssue('app-monitor', payload);
      const issueId = response.data?.issue_id;
      snackPublisher.publish({
        variant: 'success',
        title: 'Bulk request created',
        message: issueId
          ? `Filed issue ${issueId} for ${selected.length} scenario(s).`
          : `Filed request for ${selected.length} scenario(s).`,
        autoDismiss: true,
      });
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to file bulk issue.';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      className="lighthouse-bulk-dialog"
      size="wide"
      ariaLabel="Request Lighthouse setup for multiple scenarios"
    >
      <header className="lighthouse-bulk-dialog__header">
        <div>
          <h3>Bulk Lighthouse setup request</h3>
          <p>Select the scenarios missing .vrooli/lighthouse.json to include in a single issue.</p>
        </div>
        <button className="lighthouse-bulk-dialog__close" onClick={onClose} aria-label="Close dialog">
          &times;
        </button>
      </header>

      {error && (
        <div className="lighthouse-bulk-dialog__error">
          <AlertCircle size={16} />
          <span>{error}</span>
        </div>
      )}

      <div className="lighthouse-bulk-dialog__list">
        {loading ? (
          <div className="lighthouse-bulk-dialog__loading">
            <Loader size={20} className="lighthouse-tab__run-button-icon--spinning" />
            <span>Loading scenarios...</span>
          </div>
        ) : scenarios.length === 0 ? (
          <p>All scenarios already have Lighthouse configs.</p>
        ) : (
          scenarios.map((item) => {
            const selected = selection.has(item.scenario);
            return (
              <button
                key={item.scenario}
                type="button"
                className="lighthouse-bulk-dialog__list-item"
                onClick={() => toggleScenario(item.scenario)}
              >
                <div className="lighthouse-bulk-dialog__list-info">
                  <strong>{item.scenario}</strong>
                  <span>{item.expected_path || `.vrooli/lighthouse.json`}</span>
                </div>
                {selected ? <CheckSquare size={18} /> : <Square size={18} />}
              </button>
            );
          })
        )}
      </div>

      {scenarios.length > 0 && (
        <div className="lighthouse-bulk-dialog__selection-actions">
          <button type="button" onClick={handleSelectAll}>Select all</button>
          <button type="button" onClick={handleSelectNone}>Clear</button>
          <span>{selection.size} selected</span>
        </div>
      )}

      <footer className="lighthouse-bulk-dialog__footer">
        <button className="lighthouse-tab__error-button lighthouse-tab__error-button--secondary" onClick={onClose} disabled={submitting}>
          Cancel
        </button>
        <button className="lighthouse-tab__error-button" onClick={handleSubmit} disabled={submitting || loading || scenarios.length === 0}>
          {submitting ? (
            <>
              <Loader size={16} className="lighthouse-tab__run-button-icon--spinning" />
              Filing issue...
            </>
          ) : (
            'Create issue'
          )}
        </button>
      </footer>
    </ResponsiveDialog>
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
                      score >= 0.9
                        ? 'var(--color-success, #33e29a)'
                        : score >= 0.5
                          ? 'var(--color-warning, #ffd166)'
                          : 'var(--color-danger, #ff6f91)',
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
