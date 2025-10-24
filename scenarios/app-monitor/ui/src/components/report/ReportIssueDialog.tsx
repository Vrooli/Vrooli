import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import type { App } from '@/types';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
  BridgeScreenshotMode,
  BridgeScreenshotOptions,
} from '@vrooli/iframe-bridge';
import clsx from 'clsx';
import {
  AlertTriangle,
  Bug,
  ChevronDown,
  Circle,
  CircleDot,
  ExternalLink,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react';
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type RefObject,
} from 'react';

import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import ReportDiagnosticsPanel from './ReportDiagnosticsPanel';
import ReportScreenshotPanel from './ReportScreenshotPanel';
import type { ReportElementCapture } from './reportTypes';
import useReportIssueState, { type BridgePreviewState } from './useReportIssueState';

const REPORT_APP_LOGS_PANEL_ID = 'app-report-dialog-logs';
const REPORT_CONSOLE_LOGS_PANEL_ID = 'app-report-dialog-console';
const REPORT_NETWORK_PANEL_ID = 'app-report-dialog-network';
const REPORT_HEALTH_PANEL_ID = 'app-report-dialog-health';
const REPORT_STATUS_PANEL_ID = 'app-report-dialog-status';

interface ReportIssueDialogProps {
  isOpen: boolean;
  onClose: () => void;
  appId?: string;
  app: App | null;
  activePreviewUrl: string | null;
  canCaptureScreenshot: boolean;
  previewContainerRef: RefObject<HTMLDivElement | null>;
  iframeRef: RefObject<HTMLIFrameElement | null>;
  isPreviewSameOrigin: boolean;
  bridgeSupportsScreenshot: boolean;
  requestScreenshot: (options?: BridgeScreenshotOptions) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
    mode?: BridgeScreenshotMode;
    clip?: { x: number; y: number; width: number; height: number };
  }>;
  bridgeState: BridgePreviewState;
  logState: BridgeLogStreamState | null;
  configureLogs: ((config: { enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }) => boolean) | null;
  getRecentLogs: () => BridgeLogEvent[];
  requestLogBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeLogEvent[]>;
  networkState: BridgeNetworkStreamState | null;
  configureNetwork: ((config: { enable?: boolean; streaming?: boolean; bufferSize?: number }) => boolean) | null;
  getRecentNetworkEvents: () => BridgeNetworkEvent[];
  requestNetworkBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeNetworkEvent[]>;
  bridgeCompliance: BridgeComplianceResult | null;
  elementCaptures: ReportElementCapture[];
  onElementCaptureNoteChange: (captureId: string, note: string) => void;
  onElementCaptureRemove: (captureId: string) => void;
  onElementCapturesReset: () => void;
  onPrimaryCaptureDraftChange?: (hasCapture: boolean) => void;
}

const ReportIssueDialog = (props: ReportIssueDialogProps) => {
  const {
    isOpen,
    bridgeState,
    canCaptureScreenshot,
    appId,
    app,
    elementCaptures,
    onElementCaptureNoteChange,
    onElementCaptureRemove,
    onElementCapturesReset,
  } = props;

  const {
    textareaRef,
    form,
    modal,
    logs,
    consoleLogs,
    network,
    health,
    status,
    existingIssues,
    diagnostics,
    screenshot,
    diagnosticsSummaryIncluded,
    setIncludeDiagnosticsSummary,
  } = useReportIssueState(props);

  const sendDisabled = form.submitting || (
    screenshot.reportIncludeScreenshot
    && canCaptureScreenshot
    && (screenshot.reportScreenshotLoading || !screenshot.reportScreenshotData)
  );

  const existingIssuesLoading = existingIssues.status === 'loading';
  const existingIssuesError = existingIssues.status === 'error';
  const existingIssuesShouldWarn = existingIssues.status === 'ready'
    && (existingIssues.openCount > 0 || existingIssues.activeCount > 0);
  const existingIssuesErrorMessage = existingIssues.error ?? 'Unable to load existing issues.';
  const relevantIssues = useMemo(
    () => existingIssues.issues.filter(issue => {
      const status = issue.status?.toLowerCase();
      return status === 'open' || status === 'active';
    }),
    [existingIssues.issues],
  );
  const issuesToDisplay = existingIssuesShouldWarn
    ? relevantIssues.slice(0, 4)
    : relevantIssues.slice(0, 3);
  const issueSummaryLabel = `Active issues exist for this app`;

  let existingIssuesCheckedLabel: string | null = null;
  if (existingIssues.lastFetched) {
    const parsed = new Date(existingIssues.lastFetched);
    if (!Number.isNaN(parsed.getTime())) {
      existingIssuesCheckedLabel = parsed.toLocaleTimeString();
    }
  }

  const existingIssuesMeta = existingIssues.stale
    ? 'Snapshot may be out of date.'
    : existingIssuesCheckedLabel
      ? `Checked at ${existingIssuesCheckedLabel}${existingIssues.fromCache ? ' (cached)' : ''}`
      : existingIssues.fromCache
        ? 'Showing cached results.'
        : null;
  const captureNoteCount = useMemo(
    () => elementCaptures.reduce(
      (total, capture) => (capture.note.trim() ? total + 1 : total),
      0,
    ),
    [elementCaptures],
  );
  const hasDescription = form.message.trim().length > 0;
  const descriptionRequired = !hasDescription && captureNoteCount === 0 && !diagnosticsSummaryIncluded;
  const [issuesCollapsed, setIssuesCollapsed] = useState(false);
  const prevIssuesWarningRef = useRef(existingIssuesShouldWarn);
  useEffect(() => {
    if (!prevIssuesWarningRef.current && existingIssuesShouldWarn) {
      setIssuesCollapsed(false);
    }
    prevIssuesWarningRef.current = existingIssuesShouldWarn;
  }, [existingIssuesShouldWarn]);
  const handleToggleIssuesCollapse = () => {
    setIssuesCollapsed(prev => !prev);
  };
  const existingIssuesMetaLabel = useMemo(() => {
    if (!existingIssuesShouldWarn) {
      return existingIssuesMeta ?? null;
    }

    const parts: string[] = [];
    if (existingIssuesMeta) {
      parts.push(existingIssuesMeta);
    }

    return parts.join(' • ');
  }, [
    existingIssuesShouldWarn,
    existingIssuesMeta,
  ]);

  const issueDisplayName = useMemo(() => {
    const candidates = [
      app?.name,
      app?.scenario_name,
      appId,
    ];
    const match = candidates
      .map(value => (value ?? '').toString().trim())
      .find(value => value.length > 0);
    return match ?? 'Unknown app';
  }, [app?.name, app?.scenario_name, appId]);

  if (!isOpen) {
    return null;
  }

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={modal.handleDismiss}
      ariaLabelledBy="app-report-dialog-title"
      size="xl"
      className="report-dialog"
      overlayClassName="report-dialog__overlay"
    >
      <div className="report-dialog__header">
          <h2 id="app-report-dialog-title" className="report-dialog__title">
            <Bug aria-hidden size={20} />
            <span>Report an Issue</span>
            {issueDisplayName && (
              <span className="report-dialog__app-chip" title={issueDisplayName}>
                {issueDisplayName}
              </span>
            )}
          </h2>
          <button
            type="button"
            className="report-dialog__close"
            onClick={modal.handleDismiss}
            aria-label="Close report dialog"
          >
            <X aria-hidden size={16} />
          </button>
      </div>
      <form className="report-dialog__form" onSubmit={form.handleSubmit}>
        <div className="report-dialog__body">
          <div className="report-dialog__layout">
            <div className="report-dialog__lane report-dialog__lane--primary">
        {existingIssuesLoading && (
          <div className="report-dialog__issues-alert report-dialog__issues-alert--loading">
            <Loader2 aria-hidden size={16} className="spinning" />
            <span>Checking existing issues…</span>
          </div>
        )}

                {existingIssuesError && (
                  <div className="report-dialog__issues-alert report-dialog__issues-alert--error">
                    <AlertTriangle aria-hidden size={16} />
                    <div className="report-dialog__issues-error">
                      <p>{existingIssuesErrorMessage}</p>
                      <button
                        type="button"
                        className="report-dialog__button report-dialog__button--ghost"
                        onClick={existingIssues.refresh}
                      >
                        <RefreshCw aria-hidden size={14} />
                        <span>Retry</span>
                      </button>
                    </div>
                  </div>
                )}

                {existingIssuesShouldWarn && (
                  <div
                    className={clsx(
                      'report-dialog__bridge',
                      'report-dialog__bridge--warning',
                      'report-dialog__issues-panel',
                      existingIssues.stale && 'report-dialog__bridge--stale',
                    )}
                  >
                    <div className="report-dialog__bridge-head">
                      <AlertTriangle aria-hidden size={18} />
                      <span>{issueSummaryLabel}</span>
                      <div className="report-dialog__bridge-head-actions">
                        <button
                          type="button"
                          className="report-dialog__bridge-icon"
                          onClick={existingIssues.refresh}
                          aria-label="Refresh matching issues"
                          title="Refresh matching issues"
                        >
                          <RefreshCw aria-hidden size={16} />
                        </button>
                        {existingIssues.trackerUrl && (
                          <button
                            type="button"
                            className="report-dialog__bridge-icon"
                            onClick={() => {
                              if (existingIssues.trackerUrl) {
                                // Keep Referer so proxy affinity still maps assets for the opened tracker view.
                                window.open(existingIssues.trackerUrl, '_blank', 'noopener');
                              }
                            }}
                            aria-label="Open issue tracker"
                            title="Open issue tracker"
                          >
                            <ExternalLink aria-hidden size={16} />
                          </button>
                        )}
                        <button
                          type="button"
                          className={clsx(
                            'report-dialog__bridge-toggle',
                            issuesCollapsed && 'report-dialog__bridge-toggle--collapsed',
                          )}
                          onClick={handleToggleIssuesCollapse}
                          aria-label={issuesCollapsed ? 'Expand issues list' : 'Collapse issues list'}
                          aria-expanded={!issuesCollapsed}
                        >
                          <ChevronDown aria-hidden size={16} />
                        </button>
                      </div>
                    </div>

                    {existingIssuesMetaLabel && (
                      <p className="report-dialog__bridge-meta">{existingIssuesMetaLabel}</p>
                    )}

                    {!issuesCollapsed && issuesToDisplay.length > 0 && (
                      <ul className="report-dialog__issues-list">
                        {issuesToDisplay.map((issue, index) => {
                          const key = issue.id || issue.issue_url || issue.title || `issue-${index}`;
                          const title = issue.title?.trim() || issue.id?.trim() || 'Unnamed issue';
                          const normalizedStatus = issue.status?.toLowerCase();
                          const statusText = normalizedStatus === 'active' ? 'Active issue' : 'Open issue';
                          const StatusIcon = normalizedStatus === 'active' ? CircleDot : Circle;

                          return (
                            <li key={key} className="report-dialog__issues-entry">
                              <div className="report-dialog__issues-entry-content">
                                <span
                                  className={clsx(
                                    'report-dialog__issues-entry-title',
                                    normalizedStatus === 'active' && 'report-dialog__issues-entry-title--active',
                                  )}
                                >
                                  <span
                                    className={clsx(
                                      'report-dialog__issues-entry-icon',
                                      normalizedStatus === 'active'
                                        ? 'report-dialog__issues-entry-icon--active'
                                        : 'report-dialog__issues-entry-icon--open',
                                    )}
                                    title={statusText}
                                    aria-label={statusText}
                                  >
                                    <StatusIcon aria-hidden size={14} />
                                  </span>
                                  {title}
                                </span>
                                {issue.id && issue.id.trim() && issue.id.trim() !== title ? (
                                  <span className="report-dialog__issues-entry-meta">{issue.id.trim()}</span>
                                ) : null}
                              </div>
                              {issue.issue_url && (
                                <button
                                  type="button"
                                  className="report-dialog__bridge-icon report-dialog__issues-link"
                                  onClick={() => {
                                    if (issue.issue_url) {
                                      // Maintain Referer for consistent proxy routing when the issue opens externally.
                                      window.open(issue.issue_url, '_blank', 'noopener');
                                    }
                                  }}
                                  aria-label="View issue"
                                  title="View issue"
                                >
                                  <ExternalLink aria-hidden size={16} />
                                </button>
                              )}
                            </li>
                          );
                        })}
                      </ul>
                    )}
                  </div>
                )}

                <label htmlFor="app-report-message" className="report-dialog__label">
                  Describe the issue
                </label>
                <textarea
                  ref={textareaRef}
                  id="app-report-message"
                  className="report-dialog__textarea"
                  value={form.message}
                  onChange={form.handleMessageChange}
                  onKeyDown={form.handleMessageKeyDown}
                  rows={6}
                  placeholder="What are you seeing? Include steps to reproduce if possible."
                  disabled={form.submitting}
                  required={descriptionRequired}
                />

                <ReportDiagnosticsPanel
                  diagnostics={diagnostics}
                  includeSummary={diagnosticsSummaryIncluded}
                  onIncludeSummaryChange={setIncludeDiagnosticsSummary}
                  disabled={form.submitting}
                />
              </div>

              <ReportScreenshotPanel
                screenshot={screenshot}
                canCaptureScreenshot={canCaptureScreenshot}
                reportIncludeScreenshot={screenshot.reportIncludeScreenshot}
                reportSubmitting={form.submitting}
                elementCaptures={elementCaptures}
                onElementNoteChange={onElementCaptureNoteChange}
                onElementRemove={onElementCaptureRemove}
                logsPanel={{
                  logs,
                  consoleLogs,
                  network,
                  health,
                  status,
                  bridgeCaps: bridgeState.caps,
                  appLogsPanelId: REPORT_APP_LOGS_PANEL_ID,
                  consoleLogsPanelId: REPORT_CONSOLE_LOGS_PANEL_ID,
                  networkPanelId: REPORT_NETWORK_PANEL_ID,
                  healthPanelId: REPORT_HEALTH_PANEL_ID,
                  statusPanelId: REPORT_STATUS_PANEL_ID,
                }}
              />
          </div>
        </div>

        <div className="report-dialog__footer">
          {form.error && (
            <p className="report-dialog__error" role="alert">
              {form.error}
            </p>
          )}

          <div className="report-dialog__actions">
            <button
              type="button"
              className="report-dialog__button"
              onClick={() => {
                onElementCapturesReset();
                modal.handleReset();
              }}
            >
              Close
            </button>
            <button
              type="submit"
              className="report-dialog__button report-dialog__button--primary"
              disabled={sendDisabled}
            >
              {form.submitting ? (
                <>
                  <Loader2 aria-hidden size={16} className="spinning" />
                  Sending…
                </>
              ) : (
                'Send Report'
              )}
            </button>
          </div>
        </div>
      </form>

  </ResponsiveDialog>
  );
};

export type { ReportIssueDialogProps };
export default ReportIssueDialog;
