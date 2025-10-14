import type { RefObject } from 'react';
import {
  Bug,
  ExternalLink,
  Loader2,
  X,
} from 'lucide-react';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
} from '@vrooli/iframe-bridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import type { App } from '@/types';

import ReportDiagnosticsPanel from './ReportDiagnosticsPanel';
import ReportLogsSection from './ReportLogsSection';
import ReportScreenshotPanel from './ReportScreenshotPanel';
import useReportIssueState, { type BridgePreviewState } from './useReportIssueState';

const REPORT_APP_LOGS_PANEL_ID = 'app-report-dialog-logs';
const REPORT_CONSOLE_LOGS_PANEL_ID = 'app-report-dialog-console';
const REPORT_NETWORK_PANEL_ID = 'app-report-dialog-network';

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
  requestScreenshot: (options?: Record<string, unknown>) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
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
}

const ReportIssueDialog = (props: ReportIssueDialogProps) => {
  const {
    isOpen,
    bridgeState,
    canCaptureScreenshot,
  } = props;

  const {
    textareaRef,
    form,
    modal,
    logs,
    consoleLogs,
    network,
    diagnostics,
    screenshot,
  } = useReportIssueState(props);

  if (!isOpen) {
    return null;
  }

  const { result } = form;
  const sendDisabled = form.submitting || (
    screenshot.reportIncludeScreenshot
    && canCaptureScreenshot
    && (screenshot.reportScreenshotLoading || !screenshot.reportScreenshotData)
  );

  return (
    <div
      className="report-dialog__overlay"
      role="presentation"
      onClick={modal.handleOverlayClick}
    >
      <div
        className="report-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="app-report-dialog-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="report-dialog__header">
          <h2 id="app-report-dialog-title" className="report-dialog__title">
            <Bug aria-hidden size={20} />
            <span>Report an Issue</span>
          </h2>
          <button
            type="button"
            className="report-dialog__close"
            onClick={modal.handleDialogClose}
            disabled={form.submitting}
            aria-label="Close report dialog"
          >
            <X aria-hidden size={16} />
          </button>
        </div>

        {result ? (
          <div className="report-dialog__state">
            <p className="report-dialog__success">
              {result.message ?? 'Issue report sent successfully.'}
            </p>
            {result.issueId && (
              <p className="report-dialog__success-id">
                Tracking ID: <span>{result.issueId}</span>
              </p>
            )}
            {result.issueUrl && (
              <div className="report-dialog__success-link">
                <button
                  type="button"
                  className="report-dialog__button"
                  onClick={() => window.open(result.issueUrl, '_blank', 'noopener,noreferrer')}
                >
                  <ExternalLink aria-hidden size={16} />
                  <span>Open in Issue Tracker</span>
                </button>
              </div>
            )}
            <div className="report-dialog__actions">
              <button
                type="button"
                className="report-dialog__button report-dialog__button--primary"
                onClick={modal.handleDialogClose}
              >
                Close
              </button>
            </div>
          </div>
        ) : (
          <form className="report-dialog__form" onSubmit={form.handleSubmit}>
            <div className="report-dialog__layout">
              <div className="report-dialog__lane report-dialog__lane--primary">
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
                  required
                />

                <ReportDiagnosticsPanel diagnostics={diagnostics} />

                <ReportLogsSection
                  logs={logs}
                  consoleLogs={consoleLogs}
                  network={network}
                  bridgeCaps={bridgeState.caps}
                  appLogsPanelId={REPORT_APP_LOGS_PANEL_ID}
                  consoleLogsPanelId={REPORT_CONSOLE_LOGS_PANEL_ID}
                  networkPanelId={REPORT_NETWORK_PANEL_ID}
                />
              </div>

              <ReportScreenshotPanel
                screenshot={screenshot}
                canCaptureScreenshot={canCaptureScreenshot}
                reportIncludeScreenshot={screenshot.reportIncludeScreenshot}
                reportSubmitting={form.submitting}
              />
            </div>

            {form.error && (
              <p className="report-dialog__error" role="alert">
                {form.error}
              </p>
            )}

            <div className="report-dialog__actions">
              <button
                type="button"
                className="report-dialog__button"
                onClick={modal.handleDialogClose}
                disabled={form.submitting}
              >
                Cancel
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
          </form>
        )}

      </div>
    </div>
  );
};

export type { ReportIssueDialogProps };
export default ReportIssueDialog;
