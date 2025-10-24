import clsx from 'clsx';
import { Loader2, Trash2 } from 'lucide-react';
import {
  useEffect,
  useMemo,
  useState,
  useRef,
  type ChangeEvent,
  type ReactNode,
} from 'react';

import type { ReportElementCapture } from './reportTypes';
import type { ReportScreenshotState } from './useReportIssueState';

const PRIMARY_TAB_ID = 'page';

interface ReportScreenshotPanelProps {
  screenshot: ReportScreenshotState;
  canCaptureScreenshot: boolean;
  reportIncludeScreenshot: boolean;
  reportSubmitting: boolean;
  elementCaptures: ReportElementCapture[];
  onElementNoteChange: (captureId: string, note: string) => void;
  onElementRemove: (captureId: string) => void;
}

type CaptureTab =
  | { id: string; label: string; type: 'page' }
  | { id: string; label: string; type: 'element'; capture: ReportElementCapture; index: number };

const ReportScreenshotPanel = ({
  screenshot,
  canCaptureScreenshot,
  reportIncludeScreenshot,
  reportSubmitting,
  elementCaptures,
  onElementNoteChange,
  onElementRemove,
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

  const tabs = useMemo<CaptureTab[]>(() => {
    const pageTab: CaptureTab = { id: PRIMARY_TAB_ID, label: 'Preview', type: 'page' };
    const elementTabs = elementCaptures.map<CaptureTab>((capture, index) => {
      const label = capture.metadata.selector?.trim()
        || capture.metadata.label?.trim()
        || capture.metadata.tagName
        || `Capture ${index + 1}`;
      return {
        id: capture.id,
        label,
        type: 'element',
        capture,
        index,
      };
    });
    return [pageTab, ...elementTabs];
  }, [elementCaptures]);

  const [activeTab, setActiveTab] = useState<string>(() => (
    reportIncludeScreenshot || elementCaptures.length === 0
      ? PRIMARY_TAB_ID
      : elementCaptures[0]?.id ?? PRIMARY_TAB_ID
  ));
  const previousIncludeRef = useRef(reportIncludeScreenshot);

  useEffect(() => {
    const tabIds = tabs.map(tab => tab.id);
    if (!tabIds.includes(activeTab)) {
      const fallback = tabIds.includes(PRIMARY_TAB_ID)
        ? PRIMARY_TAB_ID
        : elementCaptures[0]?.id ?? PRIMARY_TAB_ID;
      setActiveTab(fallback);
    }
  }, [activeTab, elementCaptures, tabs]);

  useEffect(() => {
    const previousInclude = previousIncludeRef.current;
    if (reportIncludeScreenshot && !previousInclude) {
      setActiveTab(PRIMARY_TAB_ID);
    } else if (!reportIncludeScreenshot && previousInclude && activeTab === PRIMARY_TAB_ID) {
      const firstCaptureId = elementCaptures[0]?.id;
      if (firstCaptureId) {
        setActiveTab(firstCaptureId);
      }
    }
    previousIncludeRef.current = reportIncludeScreenshot;
  }, [activeTab, elementCaptures, reportIncludeScreenshot]);

  const showPreviewContent = reportIncludeScreenshot && canCaptureScreenshot;

  const renderPrimaryPanel = () => (
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
  );

  const renderElementPanel = (capture: ReportElementCapture, index: number) => {
    const selector = capture.metadata.selector?.trim() ?? null;
    const label = capture.metadata.label?.trim() ?? null;
    const tagLabel = capture.metadata.tagName ?? null;
    const title = capture.metadata.title ?? null;
    const ariaDescription = capture.metadata.ariaDescription ?? null;
    const role = capture.metadata.role ?? null;
    const classes = (capture.metadata.classes ?? []).filter(Boolean);
    const textValue = capture.metadata.text?.trim() ?? null;
    const truncatedText = textValue && textValue.length > 180
      ? `${textValue.slice(0, 180)}…`
      : textValue;
    const boundingBox = capture.metadata.boundingBox ?? null;
    const dimensionLabel = `${capture.width} × ${capture.height} px`;
    const createdAtLabel = Number.isFinite(capture.createdAt)
      ? new Date(capture.createdAt).toLocaleString()
      : null;
    const noteFieldId = `report-element-note-${capture.id}`;
    const imageSrc = capture.data.startsWith('data:')
      ? capture.data
      : `data:image/png;base64,${capture.data}`;

    const metadataRows = [
      selector && { label: 'Selector', value: <code>{selector}</code> },
      label && { label: 'Label', value: label },
      role && { label: 'Role', value: role },
      truncatedText && { label: 'Text', value: truncatedText },
      classes.length > 0 && { label: 'Classes', value: classes.map(token => `.${token}`).join(' ') },
      title && { label: 'Title', value: title },
      ariaDescription && { label: 'ARIA description', value: ariaDescription },
      boundingBox && {
        label: 'Bounding box',
        value: `${Math.round(boundingBox.width)} × ${Math.round(boundingBox.height)} px at (${Math.round(boundingBox.x)}, ${Math.round(boundingBox.y)})`,
      },
    ].filter(Boolean) as { label: string; value: ReactNode }[];

    const handleNoteChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
      onElementNoteChange(capture.id, event.target.value);
    };

    return (
      <div className="report-element-capture">
        <div className="report-element-capture__header">
          <div className="report-element-capture__heading">
            <p className="report-element-capture__title">
              {selector || label || tagLabel || `Capture ${index + 1}`}
            </p>
            <p className="report-element-capture__meta">
              {dimensionLabel}
              {createdAtLabel ? ` · ${createdAtLabel}` : ''}
            </p>
          </div>
          <button
            type="button"
            className="report-element-capture__remove"
            onClick={() => onElementRemove(capture.id)}
            title="Remove capture"
            aria-label="Remove capture"
          >
            <Trash2 aria-hidden size={16} />
          </button>
        </div>

        <div className="report-element-capture__image">
          <img src={imageSrc} alt="Element capture" loading="lazy" />
        </div>

        {metadataRows.length > 0 && (
          <dl className="report-element-capture__details">
            {metadataRows.map(row => (
              <div key={row.label} className="report-element-capture__detail">
                <dt>{row.label}</dt>
                <dd>{row.value}</dd>
              </div>
            ))}
          </dl>
        )}

        <label className="report-element-capture__note-label" htmlFor={noteFieldId}>
          Capture note
        </label>
        <textarea
          id={noteFieldId}
          className="report-element-capture__note"
          rows={3}
          value={capture.note}
          onChange={handleNoteChange}
          placeholder="Add context for this capture"
        />
      </div>
    );
  };

  return (
    <div className="report-dialog__lane report-dialog__lane--secondary">
      <div className="report-dialog__captures">
        <div className="report-dialog__captures-tabs" role="tablist" aria-orientation="horizontal">
          {tabs.map(tab => {
            const tabId = `${tab.id}-tab`;
            const panelId = `report-capture-panel-${tab.id}`;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                id={tabId}
                role="tab"
                aria-selected={isActive}
                aria-controls={panelId}
                className={clsx(
                  'report-dialog__capture-tab',
                  isActive && 'report-dialog__capture-tab--active',
                )}
                onClick={() => setActiveTab(tab.id)}
              >
                {tab.label}
              </button>
            );
          })}
        </div>

        {tabs.map(tab => {
          const panelId = `report-capture-panel-${tab.id}`;
          const isActive = activeTab === tab.id;
          return (
            <div
              key={tab.id}
              id={panelId}
              role="tabpanel"
              aria-labelledby={`${tab.id}-tab`}
              className={clsx(
                'report-dialog__capture-panel',
                isActive && 'report-dialog__capture-panel--active',
              )}
              hidden={!isActive}
            >
              {tab.type === 'page'
                ? renderPrimaryPanel()
                : renderElementPanel(tab.capture, tab.index)}
            </div>
          );
        })}

        {tabs.length === 1 && !reportIncludeScreenshot && elementCaptures.length === 0 && canCaptureScreenshot && (
          <p className="report-dialog__preview-hint report-dialog__preview-hint--inactive">
            Use the inspector to add element captures or enable the toggle to include the preview screenshot.
          </p>
        )}
      </div>
    </div>
  );
};

export default ReportScreenshotPanel;
