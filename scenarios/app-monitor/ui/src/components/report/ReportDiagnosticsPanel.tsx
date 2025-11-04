import { useMemo, useState, type ReactNode } from 'react';
import clsx from 'clsx';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  Loader2,
  RefreshCw,
} from 'lucide-react';

import type { App, BridgeRuleReport, BridgeRuleViolation } from '@/types';
import type { BrowserlessFallbackPageStatus } from '@/services/api';
import type { ReportDiagnosticsState } from './useReportIssueState';

interface ReportDiagnosticsPanelProps {
  diagnostics: ReportDiagnosticsState;
  includeSummary: boolean;
  onIncludeSummaryChange: (value: boolean) => void;
  disabled?: boolean;
  pageStatus?: BrowserlessFallbackPageStatus | null;
  activePreviewUrl?: string | null;
  app?: App | null;
}

const ReportDiagnosticsPanel = ({
  diagnostics,
  includeSummary,
  onIncludeSummaryChange,
  disabled = false,
  pageStatus = null,
  activePreviewUrl = null,
  app = null,
}: ReportDiagnosticsPanelProps) => {
  const {
    diagnosticsState,
    diagnosticsLoading,
    diagnosticsWarning,
    diagnosticsError,
    diagnosticsScannedFileCount,
    diagnosticsCheckedAt,
    refreshDiagnostics,
    diagnosticsRuleResults,
    diagnosticsWarnings,
    diagnosticsDescription,
  } = diagnostics;

  const hasScanFailure = diagnosticsState === 'fail' && diagnosticsScannedFileCount <= 0;

  const [isCollapsed, setIsCollapsed] = useState(true);

  const handleToggleCollapse = () => {
    setIsCollapsed(prev => !prev);
  };

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

    // Check if scenario exists but has no UI port
    // Show this diagnostic regardless of scenario state (running, stopped, etc.)
    // because the lack of UI port is a configuration issue that should be surfaced
    const hasNoPreviewUrl = !activePreviewUrl || activePreviewUrl.trim() === '';

    // Debug logging
    if (typeof window !== 'undefined' && (window as any).__DEBUG_NO_UI_PORT) {
      console.log('[ReportDiagnosticsPanel] No UI Port check:', {
        app: app ? { id: app.id, status: app.status, port_mappings: app.port_mappings } : null,
        activePreviewUrl,
        hasNoPreviewUrl,
        willAddCard: app && hasNoPreviewUrl,
      });
    }

    if (app && hasNoPreviewUrl) {
      cards.push({
        key: 'no-ui-port',
        status: 'warning',
        title: 'No UI port detected',
        summary: 'This scenario does not expose a web interface. The preview is unavailable, but logs and diagnostics can still be captured from the backend.',
      });
    }

    if (hasScanFailure) {
      cards.push({
        key: 'scan-missing',
        status: 'warning',
        title: 'No files were scanned',
        summary: 'Ensure the scenario UI imports the iframe bridge helpers and includes relevant UI/config files in the scan targets.',
      });
    }

    diagnosticsWarnings.forEach((message: string, index: number) => {
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

    // Add page status analysis from browserless fallback
    if (pageStatus) {
      if (pageStatus.whiteScreen) {
        cards.push({
          key: 'page-white-screen',
          status: 'error',
          title: 'White screen detected',
          summary: 'Page loaded but rendered no visible content. This typically indicates a bundling error or runtime exception before the UI initialized.',
        });
      }

      if (pageStatus.moduleError) {
        cards.push({
          key: 'page-module-error',
          status: 'error',
          title: 'Module loading error',
          summary: pageStatus.moduleError,
        });
      }

      if (pageStatus.loadError) {
        cards.push({
          key: 'page-load-error',
          status: 'error',
          title: 'Resource loading error',
          summary: pageStatus.loadError,
        });
      }

      if (pageStatus.httpError) {
        cards.push({
          key: 'page-http-error',
          status: 'error',
          title: `HTTP ${pageStatus.httpError.status} error`,
          summary: pageStatus.httpError.message,
        });
      }

      if (pageStatus.resourceCount === 0 && !pageStatus.whiteScreen) {
        cards.push({
          key: 'page-no-resources',
          status: 'warning',
          title: 'No resources loaded',
          summary: 'The page HTML loaded but no JavaScript or CSS resources were fetched. Possible bundling or server configuration issue.',
        });
      }
    }

    return cards;
  }, [diagnosticsError, diagnosticsState, diagnosticsWarning, diagnosticsWarnings, hasScanFailure, pageStatus, app, activePreviewUrl]);

  const ruleCards = useMemo(() => diagnosticsRuleResults.map((rule: BridgeRuleReport, index: number) => {
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
              {rule.violations.map((violation: BridgeRuleViolation) => (
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

  const cards = useMemo(() => {
    const entries: Array<{
      key: string;
      status: 'success' | 'warning' | 'error';
      title: string;
      summary?: string;
      meta?: string | null;
      body?: ReactNode;
    }> = [];

    globalCards.forEach((card: typeof globalCards[number]) => entries.push(card));
    ruleCards.forEach((card: typeof ruleCards[number]) => entries.push(card));

    return entries;
  }, [globalCards, ruleCards]);

  // Compute effective state considering both diagnostics and global cards
  const hasGlobalWarningsOrErrors = globalCards.length > 0;
  const effectiveHasWarnings = diagnosticsState === 'fail' || diagnosticsState === 'error' || hasGlobalWarningsOrErrors;
  const effectiveIsOk = diagnosticsState === 'pass' && !hasGlobalWarningsOrErrors;

  // Compute status icon and label based on effective state
  const effectiveStatusIcon = useMemo(() => {
    if (diagnosticsState === 'loading') {
      return <Loader2 aria-hidden size={18} className="spinning" />;
    }
    if (effectiveHasWarnings) {
      return <AlertTriangle aria-hidden size={18} />;
    }
    if (effectiveIsOk) {
      return <CheckCircle2 aria-hidden size={18} />;
    }
    return null;
  }, [diagnosticsState, effectiveHasWarnings, effectiveIsOk]);

  const effectiveStatusLabel = useMemo(() => {
    if (diagnosticsState === 'loading') {
      return 'Running diagnostics…';
    }
    if (diagnosticsState === 'error') {
      return 'Unable to run diagnostics';
    }
    if (effectiveHasWarnings) {
      return 'Diagnostics reported findings';
    }
    if (effectiveIsOk) {
      return 'Diagnostics passed';
    }
    return 'Diagnostics not run';
  }, [diagnosticsState, effectiveHasWarnings, effectiveIsOk]);

  return (
    <div
      className={clsx(
        'report-dialog__bridge',
        effectiveHasWarnings && 'report-dialog__bridge--warning',
        effectiveIsOk && 'report-dialog__bridge--ok',
      )}
    >
      <div className="report-dialog__bridge-head">
        {effectiveStatusIcon}
        <span>{effectiveStatusLabel}</span>
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
        </div>
      )}

      {diagnosticsDescription && (
        <div className="report-dialog__bridge-actions">
          <label className="report-dialog__checkbox">
            <input
              type="checkbox"
              checked={includeSummary}
              onChange={event => onIncludeSummaryChange(event.target.checked)}
              disabled={disabled}
            />
            <span>Include diagnostics summary in description</span>
          </label>
        </div>
      )}
    </div>
  );
};

export default ReportDiagnosticsPanel;
