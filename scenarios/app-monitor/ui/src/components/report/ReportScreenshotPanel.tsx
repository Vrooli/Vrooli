import clsx from 'clsx';
import { Loader2 } from 'lucide-react';

import type { ReportScreenshotState } from './useReportIssueState';

interface ReportScreenshotPanelProps {
  screenshot: ReportScreenshotState;
  canCaptureScreenshot: boolean;
  reportIncludeScreenshot: boolean;
  reportSubmitting: boolean;
}

const ReportScreenshotPanel = ({
  screenshot,
  canCaptureScreenshot,
  reportIncludeScreenshot,
  reportSubmitting,
}: ReportScreenshotPanelProps) => {
  const {
    reportScreenshotData,
    reportScreenshotLoading,
    reportScreenshotError,
    reportScreenshotOriginalDimensions,
    reportSelectionRect,
    reportScreenshotClip,
    reportScreenshotInfo,
    reportScreenshotCountdown,
    assignScreenshotContainerRef,
    reportScreenshotContainerHandlers,
    selectionDimensionLabel,
    handleReportIncludeScreenshotChange,
    handleRetryScreenshotCapture,
    handleResetScreenshotSelection,
    handleDelayedScreenshotCapture,
    handleScreenshotImageLoad,
  } = screenshot;

  const showPreviewContent = reportIncludeScreenshot && canCaptureScreenshot;

  return (
    <div className="report-dialog__lane report-dialog__lane--secondary">
      <div className="report-dialog__preview" aria-live="polite">
        <div className="report-dialog__preview-head">
          <label className="report-dialog__checkbox report-dialog__preview-toggle">
            <input
              type="checkbox"
              checked={reportIncludeScreenshot && canCaptureScreenshot}
              onChange={handleReportIncludeScreenshotChange}
              disabled={!canCaptureScreenshot || reportSubmitting}
            />
            <span>Include screenshot of the current preview</span>
          </label>
        </div>

        {!canCaptureScreenshot && (
          <p className="report-dialog__hint">Load the preview to capture a screenshot.</p>
        )}

        {showPreviewContent ? (
          <>
            {reportScreenshotLoading && (
              <div className="report-dialog__preview-loading">
                <Loader2 aria-hidden size={18} className="spinning" />
                <span>Capturing screenshot…</span>
              </div>
            )}

            {!reportScreenshotLoading && reportScreenshotError && (
              <div className="report-dialog__preview-error">
                <p>{reportScreenshotError}</p>
                <button
                  type="button"
                  className="report-dialog__button report-dialog__button--ghost"
                  onClick={handleRetryScreenshotCapture}
                  disabled={reportScreenshotLoading}
                >
                  Retry capture
                </button>
              </div>
            )}

            {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotInfo && (
              <p className="report-dialog__preview-info">{reportScreenshotInfo}</p>
            )}

            {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotData && (
              <>
                <div
                  className={clsx(
                    'report-dialog__preview-image',
                    'report-dialog__preview-image--selectable',
                  )}
                  ref={assignScreenshotContainerRef}
                  {...reportScreenshotContainerHandlers}
                >
                  <img
                    src={`data:image/png;base64,${reportScreenshotData}`}
                    alt="Preview screenshot"
                    loading="lazy"
                    draggable={false}
                    onLoad={handleScreenshotImageLoad}
                  />
                  {reportSelectionRect && (
                    <div
                      className="report-dialog__selection"
                      style={{
                        left: `${reportSelectionRect.x}px`,
                        top: `${reportSelectionRect.y}px`,
                        width: `${reportSelectionRect.width}px`,
                        height: `${reportSelectionRect.height}px`,
                      }}
                    >
                      {selectionDimensionLabel && (
                        <span className="report-dialog__selection-label">{selectionDimensionLabel}</span>
                      )}
                    </div>
                  )}
                </div>
                <div className="report-dialog__preview-actions">
                  <div className="report-dialog__preview-buttons">
                    <button
                      type="button"
                      className="report-dialog__button report-dialog__button--ghost"
                      onClick={handleDelayedScreenshotCapture}
                      disabled={reportScreenshotLoading}
                    >
                      {reportScreenshotCountdown !== null
                        ? `Cancel timer (${reportScreenshotCountdown}s)`
                        : 'Capture in 5s'}
                    </button>
                    <button
                      type="button"
                      className="report-dialog__button report-dialog__button--ghost"
                      onClick={handleRetryScreenshotCapture}
                      disabled={reportScreenshotLoading}
                    >
                      {reportScreenshotLoading ? 'Capturing…' : 'Capture now'}
                    </button>
                    <button
                      type="button"
                      className="report-dialog__button report-dialog__button--ghost"
                      onClick={handleResetScreenshotSelection}
                      disabled={reportScreenshotLoading || !reportScreenshotClip}
                    >
                      Reset area
                    </button>
                  </div>
                  <p className="report-dialog__preview-hint">
                    {reportScreenshotClip === null
                      ? 'Drag on the screenshot to focus on the area that needs attention. Use the timer to stage transient UI before the snap.'
                      : 'Crop saved. Drag again to fine-tune or reset to capture a different view.'}
                  </p>
                </div>
                {reportScreenshotOriginalDimensions && (
                  <p className="report-dialog__preview-meta">
                    {`Original size: ${reportScreenshotOriginalDimensions.width}×${reportScreenshotOriginalDimensions.height}`}
                  </p>
                )}
              </>
            )}

            {!reportScreenshotLoading && !reportScreenshotError && !reportScreenshotData && (
              <div className="report-dialog__preview-actions">
                <div className="report-dialog__preview-buttons">
                  <button
                    type="button"
                    className="report-dialog__button report-dialog__button--ghost"
                    onClick={handleDelayedScreenshotCapture}
                    disabled={reportScreenshotLoading}
                  >
                    {reportScreenshotCountdown !== null
                      ? `Cancel timer (${reportScreenshotCountdown}s)`
                      : 'Capture in 5s'}
                  </button>
                  <button
                    type="button"
                    className="report-dialog__button report-dialog__button--ghost"
                    onClick={handleRetryScreenshotCapture}
                    disabled={reportScreenshotLoading}
                  >
                    {reportScreenshotLoading ? 'Capturing…' : 'Capture now'}
                  </button>
                </div>
                <p className="report-dialog__preview-hint">
                  Ready to capture the current preview. Use the delay to open menus or tooltips before the screenshot fires.
                </p>
              </div>
            )}
          </>
        ) : canCaptureScreenshot ? (
          <p className="report-dialog__preview-hint report-dialog__preview-hint--inactive">
            Enable the toggle above to include a screenshot with this report.
          </p>
        ) : null}
      </div>
    </div>
  );
};

export default ReportScreenshotPanel;
