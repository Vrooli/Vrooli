import type { CompleteDiagnostics } from '@/types';
import { Activity, AlertTriangle, CheckCircle, Info, XCircle, ExternalLink } from 'lucide-react';
import './DiagnosticsTab.css';

interface DiagnosticsTabProps {
  diagnostics: CompleteDiagnostics | null | undefined;
  loading?: boolean;
  error?: string | null;
  onRetry?: () => void;
}

export default function DiagnosticsTab({ diagnostics, loading, error, onRetry }: DiagnosticsTabProps) {
  if (loading) {
    return (
      <div className="diagnostics-tab">
        <div className="diagnostics-tab__loading">
          <Activity size={32} className="diagnostics-tab__loading-icon" />
          <p>Running diagnostics...</p>
        </div>
      </div>
    );
  }

  if (error && !diagnostics) {
    return (
      <div className="diagnostics-tab">
        <div className="diagnostics-tab__error">
          <AlertTriangle size={32} />
          <p>{error}</p>
          {onRetry && (
            <button type="button" className="diagnostics-tab__retry" onClick={onRetry}>
              Retry
            </button>
          )}
        </div>
      </div>
    );
  }

  if (!diagnostics) {
    return (
      <div className="diagnostics-tab">
        <div className="diagnostics-tab__empty">
          <Activity size={32} />
          <p>No diagnostics available</p>
        </div>
      </div>
    );
  }

  const healthChecks = diagnostics.health_checks;
  const hasHealthChecks = healthChecks != null && Array.isArray(healthChecks.checks) && healthChecks.checks.length > 0;
  const bridgeRules = diagnostics.bridge_rules;
  const hasBridgeViolations = bridgeRules != null && Array.isArray(bridgeRules.violations) && bridgeRules.violations.length > 0;
  const localhostUsage = diagnostics.localhost_usage;
  const hasLocalhostFindings = localhostUsage != null && Array.isArray(localhostUsage.findings) && localhostUsage.findings.length > 0;
  const auditorSummary = diagnostics.auditor_summary;
  const hasAuditorSummary = Boolean(auditorSummary && auditorSummary.total > 0);
  const issues = diagnostics.issues;
  const hasIssues = issues != null && issues.total_count > 0;
  const warnings = Array.isArray(diagnostics.warnings) ? diagnostics.warnings : [];
  const hasWarnings = warnings.length > 0;

  // Severity badge color
  const severityClass = diagnostics.severity === 'ok' ? 'success' :
    diagnostics.severity === 'warn' ? 'warning' :
      diagnostics.severity === 'error' ? 'error' : 'unknown';

  return (
    <div className="diagnostics-tab">
      {/* Overall Summary */}
      <section className="diagnostics-section diagnostics-section--summary">
        <div className="diagnostics-summary">
          <div className="diagnostics-summary__status">
            <span className={`diagnostics-severity diagnostics-severity--${severityClass}`}>
              {(diagnostics.severity ?? 'unknown').toUpperCase()}
            </span>
            {diagnostics.summary && (
              <p className="diagnostics-summary__text">{diagnostics.summary}</p>
            )}
          </div>
          {hasWarnings && (
            <div className="diagnostics-summary__count">
              <AlertTriangle size={20} />
              <span>{warnings.length} warning{warnings.length !== 1 ? 's' : ''}</span>
            </div>
          )}
        </div>
      </section>

      {/* Scenario-Auditor Summary */}
      {hasAuditorSummary && auditorSummary && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <AlertTriangle size={18} />
            <span>Scenario-auditor Summary ({auditorSummary.total} issues)</span>
          </h3>
          <div className="diagnostics-auditor">
            <div className="diagnostics-auditor__overview">
              <div>
                <p className="diagnostics-auditor__severity">Highest severity: <strong>{auditorSummary.highest_severity || 'unknown'}</strong></p>
                <p className="diagnostics-auditor__counts">
                  {Object.entries(auditorSummary.by_severity || {}).map(([severity, count]) => (
                    <span key={`severity-${severity}`} className={`diagnostics-tag diagnostics-tag--${severity}`}>
                      {severity}: {count}
                    </span>
                  ))}
                </p>
              </div>
              {auditorSummary.recommended_steps && auditorSummary.recommended_steps.length > 0 && (
                <div className="diagnostics-auditor__steps">
                  <p>Next steps:</p>
                  <ul>
                    {auditorSummary.recommended_steps.slice(0, 3).map((step, index) => (
                      <li key={`step-${index}`}>{step}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
            {auditorSummary.top_violations && auditorSummary.top_violations.length > 0 && (
              <div className="diagnostics-auditor__violations">
                {auditorSummary.top_violations.slice(0, 5).map((violation) => (
                  <div key={violation.id} className="diagnostics-violation">
                    <div className="diagnostics-violation__header">
                      <span className="diagnostics-violation__title">{violation.title || violation.rule_id || violation.id}</span>
                      <span className={`diagnostics-violation__severity diagnostics-violation__severity--${violation.severity}`}>
                        {violation.severity}
                      </span>
                    </div>
                    {violation.file_path && (
                      <code className="diagnostics-violation__file">
                        {violation.file_path}{violation.line_number ? `:${violation.line_number}` : ''}
                      </code>
                    )}
                    {violation.recommendation && (
                      <div className="diagnostics-violation__recommendation">
                        <Info size={14} />
                        <span>{violation.recommendation}</span>
                      </div>
                    )}
                  </div>
                ))}
                {auditorSummary.top_violations.length > 5 && (
                  <p className="diagnostics-violations__more">
                    ...and {auditorSummary.top_violations.length - 5} more prioritized violations
                  </p>
                )}
              </div>
            )}
            {auditorSummary.artifact?.path && (
              <div className="diagnostics-auditor__artifact">
                <a href={`/${auditorSummary.artifact.path}`} target="_blank" rel="noreferrer">
                  <ExternalLink size={14} /> View full artifact
                </a>
              </div>
            )}
          </div>
        </section>
      )}

      {/* Aggregated Warnings */}
      {hasWarnings && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <AlertTriangle size={18} />
            <span>Warnings</span>
          </h3>
          <div className="diagnostics-warnings">
            {warnings.map((warning, index) => (
              <div
                key={`warning-${index}`}
                className={`diagnostics-warning diagnostics-warning--${warning.severity}`}
              >
                <div className="diagnostics-warning__header">
                  <span className="diagnostics-warning__source">{warning.source}</span>
                  <span className={`diagnostics-warning__severity diagnostics-warning__severity--${warning.severity}`}>
                    {warning.severity}
                  </span>
                </div>
                <p className="diagnostics-warning__message">{warning.message}</p>
                {warning.file_path && (
                  <div className="diagnostics-warning__location">
                    <code>{warning.file_path}{warning.line ? `:${warning.line}` : ''}</code>
                  </div>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Health Checks */}
      {hasHealthChecks && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <Activity size={18} />
            <span>Health Checks</span>
          </h3>
          <div className="diagnostics-health-checks">
            {healthChecks.checks.map((check) => {
              const isPassing = check.status === 'pass';
              const icon = isPassing ? <CheckCircle size={16} /> : <XCircle size={16} />;

              return (
                <div
                  key={check.id}
                  className={`diagnostics-health-check diagnostics-health-check--${check.status}`}
                >
                  <div className="diagnostics-health-check__header">
                    {icon}
                    <span className="diagnostics-health-check__name">{check.name}</span>
                    {check.latency_ms !== undefined && (
                      <span className="diagnostics-health-check__latency">{check.latency_ms}ms</span>
                    )}
                  </div>
                  {check.message && (
                    <p className="diagnostics-health-check__message">{check.message}</p>
                  )}
                  {check.endpoint && (
                    <code className="diagnostics-health-check__endpoint">{check.endpoint}</code>
                  )}
                </div>
              );
            })}
          </div>
        </section>
      )}

      {/* Bridge Rules Violations */}
      {hasBridgeViolations && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <AlertTriangle size={18} />
            <span>Bridge Compliance Issues ({bridgeRules.violations.length})</span>
          </h3>
          <div className="diagnostics-violations">
            {bridgeRules.violations.slice(0, 5).map((violation, index) => (
              <div key={`violation-${index}`} className="diagnostics-violation">
                <div className="diagnostics-violation__header">
                  <span className="diagnostics-violation__title">{violation.title}</span>
                  <span className={`diagnostics-violation__severity diagnostics-violation__severity--${violation.severity}`}>
                    {violation.severity}
                  </span>
                </div>
                <p className="diagnostics-violation__description">{violation.description}</p>
                {violation.file_path && (
                  <code className="diagnostics-violation__file">
                    {violation.file_path}{violation.line ? `:${violation.line}` : ''}
                  </code>
                )}
                {violation.recommendation && (
                  <div className="diagnostics-violation__recommendation">
                    <Info size={14} />
                    <span>{violation.recommendation}</span>
                  </div>
                )}
              </div>
            ))}
            {bridgeRules.violations.length > 5 && (
              <p className="diagnostics-violations__more">
                ...and {bridgeRules.violations.length - 5} more violations
              </p>
            )}
          </div>
        </section>
      )}

      {/* Localhost Usage */}
      {hasLocalhostFindings && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <AlertTriangle size={18} />
            <span>Localhost References ({localhostUsage.findings.length})</span>
          </h3>
          <div className="diagnostics-localhost">
            <div className="diagnostics-localhost__alert">
              <AlertTriangle size={16} />
              <p>
                {localhostUsage.findings.length} hard-coded localhost reference
                {localhostUsage.findings.length !== 1 ? 's' : ''} detected.
                Update requests to use the scenario proxy base.
              </p>
            </div>
            <div className="diagnostics-localhost__findings">
              {localhostUsage.findings.slice(0, 6).map((finding, index) => (
                <div key={`finding-${index}`} className="diagnostics-localhost__finding">
                  <code className="diagnostics-localhost__file">
                    {finding.file_path}:{finding.line}
                  </code>
                  {finding.pattern && (
                    <span className="diagnostics-localhost__pattern">({finding.pattern})</span>
                  )}
                  <pre className="diagnostics-localhost__snippet">{finding.snippet}</pre>
                </div>
              ))}
              {localhostUsage.findings.length > 6 && (
                <p className="diagnostics-localhost__more">
                  ...and {localhostUsage.findings.length - 6} more occurrences
                </p>
              )}
            </div>
          </div>
        </section>
      )}

      {/* Issues Summary */}
      {hasIssues && (
        <section className="diagnostics-section">
          <h3 className="diagnostics-section__title">
            <Info size={18} />
            <span>Issues Tracker</span>
          </h3>
          <div className="diagnostics-issues">
            <div className="diagnostics-issues__summary">
              <div className="diagnostics-issues__stat">
                <span className="diagnostics-issues__stat-value">{issues.open_count}</span>
                <span className="diagnostics-issues__stat-label">Open</span>
              </div>
              <div className="diagnostics-issues__stat">
                <span className="diagnostics-issues__stat-value">{issues.active_count}</span>
                <span className="diagnostics-issues__stat-label">Active</span>
              </div>
              <div className="diagnostics-issues__stat">
                <span className="diagnostics-issues__stat-value">{issues.total_count}</span>
                <span className="diagnostics-issues__stat-label">Total</span>
              </div>
            </div>
            {issues.tracker_url && (
              <a
                href={issues.tracker_url}
                target="_blank"
                rel="noopener noreferrer"
                className="diagnostics-issues__link"
              >
                <span>View in Issue Tracker</span>
                <ExternalLink size={14} />
              </a>
            )}
          </div>
        </section>
      )}

      {/* All Good Message */}
      {!hasWarnings && !hasHealthChecks && !hasBridgeViolations && !hasLocalhostFindings && (
        <div className="diagnostics-tab__success">
          <CheckCircle size={48} />
          <p>All diagnostics passed! No issues detected.</p>
        </div>
      )}
    </div>
  );
}
