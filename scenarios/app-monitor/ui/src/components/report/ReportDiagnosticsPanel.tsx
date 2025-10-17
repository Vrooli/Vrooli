import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import clsx from 'clsx';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  Loader2,
  RefreshCw,
} from 'lucide-react';

import type { ReportDiagnosticsState } from './useReportIssueState';

interface ReportDiagnosticsPanelProps {
  diagnostics: ReportDiagnosticsState;
}

const ReportDiagnosticsPanel = ({ diagnostics }: ReportDiagnosticsPanelProps) => {
  const {
    diagnosticsState,
    diagnosticsLoading,
    diagnosticsWarning,
    diagnosticsError,
    diagnosticsScannedFileCount,
    diagnosticsCheckedAt,
    showDiagnosticsSetDescription,
    handleApplyDiagnosticsDescription,
    refreshDiagnostics,
    bridgeCompliance,
    bridgeComplianceCheckedAt,
    bridgeComplianceFailures,
    diagnosticsRuleResults,
    diagnosticsWarnings,
  } = diagnostics;

  const hasRuleViolations = useMemo(
    () => diagnosticsRuleResults.some(rule => rule.violations.length > 0),
    [diagnosticsRuleResults],
  );

  const hasRuleWarnings = useMemo(() => (
    diagnosticsRuleResults.some(rule => {
      const combinedWarnings = new Set(
        [rule.warning, ...(rule.warnings ?? [])]
          .filter((value): value is string => Boolean(value && value.trim()))
          .map(value => value.trim()),
      );
      return combinedWarnings.size > 0;
    })
  ), [diagnosticsRuleResults]);

  const hasRuntimeIssues = Boolean(bridgeCompliance && !bridgeCompliance.ok);
  const hasScanFailure = diagnosticsState === 'fail' && diagnosticsScannedFileCount <= 0;
  const hasAnyIssues = hasRuleViolations || hasRuleWarnings || hasRuntimeIssues || hasScanFailure || diagnosticsState === 'error' || diagnosticsState === 'fail';

  const prevHasIssuesRef = useRef(hasAnyIssues);
  const [isCollapsed, setIsCollapsed] = useState(() => !hasAnyIssues);

  useEffect(() => {
    if (!prevHasIssuesRef.current && hasAnyIssues) {
      setIsCollapsed(false);
    }
    prevHasIssuesRef.current = hasAnyIssues;
  }, [hasAnyIssues]);

  const handleToggleCollapse = () => {
    setIsCollapsed(prev => !prev);
  };

  const statusIcon = useMemo(() => {
    if (diagnosticsState === 'pass') {
      return <CheckCircle2 aria-hidden size={18} />;
    }
    if (diagnosticsState === 'loading') {
      return <Loader2 aria-hidden size={18} className="spinning" />;
    }
    if (diagnosticsState === 'fail' || diagnosticsState === 'error') {
      return <AlertTriangle aria-hidden size={18} />;
    }
    return null;
  }, [diagnosticsState]);

  const statusLabel = useMemo(() => {
    switch (diagnosticsState) {
      case 'loading':
        return 'Running diagnostics…';
      case 'pass':
        return 'Diagnostics passed';
      case 'fail':
        return 'Diagnostics reported findings';
      case 'error':
        return 'Unable to run diagnostics';
      case 'idle':
      default:
        return 'Diagnostics not run';
    }
  }, [diagnosticsState]);

  const globalCards = useMemo(() => {
    const cards: Array<{
      key: string;
      status: 'warning' | 'error';
      title: string;
      summary?: string;
      body?: ReactNode;
    }> = [];

    if (diagnosticsState === 'error' && diagnosticsError) {
      cards.push({
        key: 'diagnostics-error',
        status: 'error',
        title: 'Diagnostics failed',
        summary: diagnosticsError,
      });
      return cards;
    }

    if (hasScanFailure) {
      cards.push({
        key: 'scan-missing',
        status: 'warning',
        title: 'No files were scanned',
        summary: 'Ensure the scenario UI imports the iframe bridge helpers and includes relevant UI/config files in the scan targets.',
      });
    }

    diagnosticsWarnings.forEach((message, index) => {
      cards.push({
        key: `global-warning-${index}`,
        status: 'warning',
        title: 'Diagnostics warning',
        summary: message,
      });
    });

    if (diagnosticsWarning && diagnosticsWarnings.length === 0) {
      cards.push({
        key: 'global-warning-fallback',
        status: 'warning',
        title: 'Diagnostics warning',
        summary: diagnosticsWarning,
      });
    }

    return cards;
  }, [diagnosticsError, diagnosticsState, diagnosticsWarning, diagnosticsWarnings, hasScanFailure]);

  const ruleCards = useMemo(() => diagnosticsRuleResults.map((rule, index) => {
    const combinedWarnings = Array.from(new Set(
      [rule.warning, ...(rule.warnings ?? [])]
        .filter((value): value is string => Boolean(value && value.trim()))
        .map(value => value.trim()),
    ));
    const hasWarnings = combinedWarnings.length > 0;
    const hasViolations = rule.violations.length > 0;

    const status: 'success' | 'warning' | 'error' = hasViolations
      ? 'error'
      : hasWarnings
        ? 'warning'
        : 'success';

    const violationSummary = hasViolations
      ? `Found ${rule.violations.length} issue${rule.violations.length === 1 ? '' : 's'}.`
      : hasWarnings
        ? 'Returned warnings.'
        : 'All checks passed.';

    return {
      key: rule.rule_id || `rule-${index}`,
      status,
      title: rule.name ?? rule.rule_id ?? 'Diagnostics rule',
      summary: violationSummary,
      body: (
        <>
          {hasViolations && (
            <ul className="report-dialog__bridge-inline-list">
              {rule.violations.map(violation => (
                <li key={`${rule.rule_id}:${violation.file_path}:${violation.line}:${violation.title}`}>
                  <strong>{violation.title}</strong>
                  {violation.file_path && (
                    <span>
                      {' '}
                      ({violation.file_path}
                      {typeof violation.line === 'number' ? `:${violation.line}` : ''})
                    </span>
                  )}
                  {violation.description && <div>{violation.description}</div>}
                  {violation.recommendation && (
                    <div className="report-dialog__bridge-recommendation">{violation.recommendation}</div>
                  )}
                </li>
              ))}
            </ul>
          )}
          {combinedWarnings.map(message => (
            <p key={`${rule.rule_id}-warning-${message}`} className="report-dialog__bridge-inline-note">
              {message}
            </p>
          ))}
        </>
      ),
    };
  }), [diagnosticsRuleResults]);

  const runtimeCard = useMemo(() => {
    if (!bridgeCompliance) {
      return null;
    }

    const status: 'success' | 'warning' = bridgeCompliance.ok ? 'success' : 'warning';
    return {
      key: 'runtime-bridge',
      status,
      title: 'Runtime bridge check',
      summary: bridgeCompliance.ok
        ? 'Iframe handshake responded as expected.'
        : 'Runtime diagnostics detected missing signals.',
      meta: bridgeComplianceCheckedAt ? `Checked at ${bridgeComplianceCheckedAt}` : null,
      body: bridgeCompliance.ok ? null : (
        bridgeComplianceFailures.length > 0 ? (
          <ul className="report-dialog__bridge-inline-list">
            {bridgeComplianceFailures.map(message => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        ) : (
          <p className="report-dialog__bridge-inline-note">
            Runtime diagnostics could not identify specific missing signals. Refresh the preview and try again.
          </p>
        )
      ),
    };
  }, [bridgeCompliance, bridgeComplianceCheckedAt, bridgeComplianceFailures]);

  const cards = useMemo(() => {
    const entries: Array<{
      key: string;
      status: 'success' | 'warning' | 'error';
      title: string;
      summary?: string;
      meta?: string | null;
      body?: ReactNode;
    }> = [];

    globalCards.forEach(card => entries.push(card));
    ruleCards.forEach(card => entries.push(card));
    if (runtimeCard) {
      entries.push(runtimeCard);
    }

    return entries;
  }, [globalCards, ruleCards, runtimeCard]);

  return (
    <div
      className={clsx(
        'report-dialog__bridge',
        diagnosticsState === 'fail' && 'report-dialog__bridge--warning',
        diagnosticsState === 'pass' && 'report-dialog__bridge--ok',
        diagnosticsState === 'error' && 'report-dialog__bridge--warning',
      )}
    >
      <div className="report-dialog__bridge-head">
        {statusIcon}
        <span>{statusLabel}</span>
        <div className="report-dialog__bridge-head-actions">
          <button
            type="button"
            className="report-dialog__bridge-refresh"
            onClick={() => refreshDiagnostics()}
            disabled={diagnosticsLoading}
            aria-label="Refresh diagnostics"
          >
            {diagnosticsLoading ? (
              <Loader2 aria-hidden size={14} className="spinning" />
            ) : (
              <RefreshCw aria-hidden size={14} />
            )}
          </button>
          <button
            type="button"
            className={clsx(
              'report-dialog__bridge-toggle',
              isCollapsed && 'report-dialog__bridge-toggle--collapsed',
            )}
            onClick={handleToggleCollapse}
            aria-label={isCollapsed ? 'Expand diagnostics details' : 'Collapse diagnostics details'}
            aria-expanded={!isCollapsed}
          >
            <ChevronDown aria-hidden size={16} />
          </button>
        </div>
      </div>
      {diagnosticsCheckedAt && !diagnosticsLoading && (
        <p className="report-dialog__bridge-meta">Checked at {diagnosticsCheckedAt}</p>
      )}

      {!isCollapsed && (
        <div className="report-dialog__bridge-body">
          {cards.length > 0 && (
            <div className="report-dialog__bridge-card-group">
              {cards.map(card => (
                <div
                  key={card.key}
                  className={clsx(
                    'report-dialog__bridge-inline',
                    card.status === 'success' && 'report-dialog__bridge-inline--success',
                    card.status === 'warning' && 'report-dialog__bridge-inline--warning',
                    card.status === 'error' && 'report-dialog__bridge-inline--error',
                  )}
                >
                  {card.status === 'success' ? (
                    <CheckCircle2 aria-hidden size={18} />
                  ) : (
                    <AlertTriangle aria-hidden size={18} />
                  )}
                  <div className="report-dialog__bridge-inline-content">
                    <p className="report-dialog__bridge-inline-title">{card.title}</p>
                    {card.meta && (
                      <p className="report-dialog__bridge-inline-meta">{card.meta}</p>
                    )}
                    {card.summary && (
                      <p className="report-dialog__bridge-inline-note">{card.summary}</p>
                    )}
                    {card.body}
                  </div>
                </div>
              ))}
            </div>
          )}

          {diagnosticsState === 'pass' && diagnosticsScannedFileCount > 0 && cards.length === 0 && (
            <p className="report-dialog__bridge-note">
              Scanned {diagnosticsScannedFileCount} file{diagnosticsScannedFileCount === 1 ? '' : 's'} and found no violations.
            </p>
          )}

          {showDiagnosticsSetDescription && (
            <div className="report-dialog__bridge-actions">
              <button
                type="button"
                className="report-dialog__button report-dialog__button--ghost"
                onClick={handleApplyDiagnosticsDescription}
              >
                Use details in description
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ReportDiagnosticsPanel;
