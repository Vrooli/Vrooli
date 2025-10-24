import clsx from 'clsx';
import { BoxSelect, ChevronDown, Download, Image, Inspect, Loader2, PlusCircle, X } from 'lucide-react';
import type { BridgeInspectState } from '@/hooks/useIframeBridge';
import type { PreviewInspectorState } from './usePreviewInspector';

export type PreviewInspectorPanelProps = {
  inspectState: BridgeInspectState;
  previewUrl: string | null;
  inspector: PreviewInspectorState;
};

const PreviewInspectorPanel = ({
  inspectState,
  previewUrl,
  inspector,
}: PreviewInspectorPanelProps) => {
  if (!inspector.shouldRenderInspectorDialog) {
    return null;
  }

  const {
    inspectorDialogRef,
    inspectorDialogPosition,
    isInspectorDragging,
    inspectorDialogTitleId,
    inspectorDetailsSectionId,
    inspectorDetailsContentId,
    inspectorScreenshotSectionId,
    inspectorScreenshotContentId,
    inspectorReportNoteId,
    inspectorDetailsExpanded,
    toggleInspectorDetails,
    inspectStatusMessage,
    inspectCopyFeedback,
    inspectTarget,
    inspectActiveChipLabel,
    inspectMeta,
    inspectRect,
    inspectSizeLabel,
    inspectPositionLabel,
    inspectClassTokens,
    inspectSelectorValue,
    inspectLabelValue,
    inspectTextValue,
    inspectTextPreview,
    inspectAriaDescription,
    inspectTitleValue,
    inspectMethodLabel,
    inspectorScreenshot,
    isInspectorScreenshotCapturing,
    inspectorScreenshotExpanded,
    toggleInspectorScreenshotExpanded,
    inspectorCaptureNote,
    handleInspectorCaptureNoteChange,
    handleAddInspectorCaptureToReport,
    handleCopySelector,
    handleCopyText,
    handleDownloadInspectorScreenshot,
    handleToggleInspectMode,
    handleInspectorDialogClose,
    handleInspectorPointerDown,
    handleInspectorPointerMove,
    handleInspectorPointerUp,
    handleInspectorClickCapture,
  } = inspector;

  const handleDialogRef = (node: HTMLDivElement | null) => {
    inspectorDialogRef.current = node;
  };

  const showDetailsSection = !inspectState.active && Boolean(inspectTarget);
  const showScreenshotSection = !inspectState.active;
  const activeChipLabel = inspectActiveChipLabel ?? 'Move finger to highlight an element';

  return (
    <div
      ref={handleDialogRef}
      className={clsx(
        'preview-inspector-dialog',
        isInspectorDragging && 'preview-inspector-dialog--dragging',
      )}
      style={{ transform: `translate3d(${Math.round(inspectorDialogPosition.x)}px, ${Math.round(inspectorDialogPosition.y)}px, 0)` }}
      role="dialog"
      aria-modal="false"
      aria-labelledby={inspectorDialogTitleId}
      onPointerDown={handleInspectorPointerDown}
      onPointerMove={handleInspectorPointerMove}
      onPointerUp={handleInspectorPointerUp}
      onPointerCancel={handleInspectorPointerUp}
      onClickCapture={handleInspectorClickCapture}
    >
      <section
        className={clsx(
          'preview-inspector',
          inspectState.active && 'preview-inspector--active',
          !inspectState.supported && 'preview-inspector--unsupported',
        )}
        aria-live="polite"
      >
        <header className="preview-inspector__header">
          <div className="preview-inspector__handle">
            <span className="preview-inspector__title" id={inspectorDialogTitleId}>
              {inspectState.active
                ? 'Inspecting element'
                : inspectState.supported
                  ? 'Element inspector'
                  : 'Element inspector unavailable'}
            </span>
            <div className="preview-inspector__handle-actions">
              <button
                type="button"
                className={clsx(
                  'preview-inspector__icon-button',
                  inspectState.active && 'preview-inspector__icon-button--active',
                )}
                onClick={handleToggleInspectMode}
                disabled={!inspectState.supported || (!inspectState.active && !previewUrl)}
                aria-pressed={inspectState.active}
                aria-label={inspectState.active ? 'Stop element inspection' : 'Start element inspection'}
                title={inspectState.active ? 'Stop inspecting elements' : 'Inspect elements'}
              >
                <Inspect aria-hidden size={16} />
              </button>
              <button
                type="button"
                className="preview-inspector__close"
                onClick={handleInspectorDialogClose}
                aria-label="Close element inspector"
              >
                <X aria-hidden size={16} />
              </button>
            </div>
          </div>
        </header>
        {inspectStatusMessage && (
          <p
            className={clsx(
              'preview-inspector__status',
              (inspectState.error || (inspectStatusMessage && inspectStatusMessage.toLowerCase().includes('failed')))
                && 'preview-inspector__status--error',
            )}
          >
            {inspectStatusMessage}
          </p>
        )}
        {inspectState.active && (
          <div className="preview-inspector__active-chip" role="status">
            <span className="preview-inspector__active-chip-label" title={activeChipLabel}>
              {activeChipLabel}
            </span>
          </div>
        )}
        {showDetailsSection && (
          <div className="preview-inspector__section">
            <button
              type="button"
              className={clsx(
                'preview-inspector__section-toggle',
                inspectorDetailsExpanded && 'preview-inspector__section-toggle--expanded',
              )}
              onClick={toggleInspectorDetails}
              aria-expanded={inspectorDetailsExpanded}
              aria-controls={inspectorDetailsContentId}
              id={inspectorDetailsSectionId}
            >
              <span className="preview-inspector__section-label">
                <BoxSelect aria-hidden size={16} className="preview-inspector__section-symbol" />
                Element details
              </span>
              <ChevronDown aria-hidden size={16} className="preview-inspector__section-icon" />
            </button>
            <div
              id={inspectorDetailsContentId}
              className="preview-inspector__section-content"
              hidden={!inspectorDetailsExpanded}
              role="region"
              aria-labelledby={inspectorDetailsSectionId}
            >
              <dl className="preview-inspector__details">
                <div className="preview-inspector__detail">
                  <dt>Element</dt>
                  <dd>
                    <code>{inspectMeta?.tag ?? 'unknown'}</code>
                    {inspectMeta?.id && (
                      <span className="preview-inspector__element-token">#{inspectMeta.id}</span>
                    )}
                    {inspectClassTokens.map(token => (
                      <span key={token} className="preview-inspector__element-token">.{token}</span>
                    ))}
                  </dd>
                </div>
                {inspectSelectorValue && (
                  <div className="preview-inspector__detail">
                    <dt>Selector</dt>
                    <dd>
                      <code>{inspectSelectorValue}</code>
                      <button
                        type="button"
                        className="preview-inspector__copy"
                        onClick={handleCopySelector}
                        aria-label="Copy CSS selector"
                      >
                        Copy
                      </button>
                      {inspectCopyFeedback === 'selector' && (
                        <span className="preview-inspector__copy-feedback">Copied</span>
                      )}
                    </dd>
                  </div>
                )}
                {inspectLabelValue && (
                  <div className="preview-inspector__detail">
                    <dt>Label</dt>
                    <dd>{inspectLabelValue}</dd>
                  </div>
                )}
                {inspectTitleValue && (
                  <div className="preview-inspector__detail">
                    <dt>Title</dt>
                    <dd>{inspectTitleValue}</dd>
                  </div>
                )}
                {inspectAriaDescription && (
                  <div className="preview-inspector__detail">
                    <dt>ARIA description</dt>
                    <dd>{inspectAriaDescription}</dd>
                  </div>
                )}
                {inspectTextValue && (
                  <div className="preview-inspector__detail">
                    <dt>Text</dt>
                    <dd className="preview-inspector__text">
                      <span title={inspectTextValue}>{inspectTextPreview ?? inspectTextValue}</span>
                      <button
                        type="button"
                        className="preview-inspector__copy"
                        onClick={handleCopyText}
                        aria-label="Copy element text"
                      >
                        Copy
                      </button>
                      {inspectCopyFeedback === 'text' && (
                        <span className="preview-inspector__copy-feedback">Copied</span>
                      )}
                    </dd>
                  </div>
                )}
                {inspectPositionLabel && (
                  <div className="preview-inspector__detail">
                    <dt>Position</dt>
                    <dd>{inspectPositionLabel}</dd>
                  </div>
                )}
                {inspectSizeLabel && (
                  <div className="preview-inspector__detail">
                    <dt>Size</dt>
                    <dd>{inspectSizeLabel}</dd>
                  </div>
                )}
                {inspectMethodLabel && (
                  <div className="preview-inspector__detail">
                    <dt>Selection method</dt>
                    <dd>{inspectMethodLabel}</dd>
                  </div>
                )}
                {inspectRect && (
                  <div className="preview-inspector__detail">
                    <dt>Bounding box</dt>
                    <dd>
                      <code>{`x ${Math.round(inspectRect.x)}, y ${Math.round(inspectRect.y)}, w ${Math.round(inspectRect.width)}, h ${Math.round(inspectRect.height)}`}</code>
                    </dd>
                  </div>
                )}
              </dl>
            </div>
          </div>
        )}

        {showScreenshotSection && (
          <div className="preview-inspector__section">
            <button
              type="button"
              className={clsx(
              'preview-inspector__section-toggle',
              inspectorScreenshotExpanded && 'preview-inspector__section-toggle--expanded',
            )}
            onClick={toggleInspectorScreenshotExpanded}
            aria-expanded={inspectorScreenshotExpanded}
            aria-controls={inspectorScreenshotContentId}
            id={inspectorScreenshotSectionId}
          >
            <span className="preview-inspector__section-label">
              <Image aria-hidden size={16} className="preview-inspector__section-symbol" />
              Screenshot
            </span>
            <ChevronDown aria-hidden size={16} className="preview-inspector__section-icon" />
          </button>
          <div
            id={inspectorScreenshotContentId}
            className="preview-inspector__section-content"
            hidden={!inspectorScreenshotExpanded}
            role="region"
            aria-labelledby={inspectorScreenshotSectionId}
          >
            <figure className="preview-inspector__screenshot">
              <div
                className={clsx(
                  'preview-inspector__screenshot-frame',
                  !inspectorScreenshot && 'preview-inspector__screenshot-frame--empty',
                )}
              >
                <div
                  className={clsx(
                    'preview-inspector__screenshot-content',
                    !inspectorScreenshot && 'preview-inspector__screenshot-content--empty',
                  )}
                >
                  {inspectorScreenshot ? (
                    <img
                      src={inspectorScreenshot.dataUrl}
                      alt={`Captured element screenshot (${inspectorScreenshot.width} × ${inspectorScreenshot.height} pixels)`}
                    />
                  ) : (
                    <div className="preview-inspector__screenshot-placeholder" role="presentation">
                      <Image aria-hidden size={36} className="preview-inspector__screenshot-placeholder-icon" />
                      <span>No screenshot captured yet</span>
                    </div>
                  )}
                </div>
                {isInspectorScreenshotCapturing && (
                  <div className="preview-inspector__screenshot-overlay" role="status" aria-live="polite">
                    <Loader2 aria-hidden size={28} className="spinning preview-inspector__screenshot-spinner" />
                    <span className="preview-inspector__screenshot-spinner-label">Capturing screenshot...</span>
                  </div>
                )}
                {inspectorScreenshot && (
                  <button
                    type="button"
                    className="preview-inspector__screenshot-download"
                    onClick={handleDownloadInspectorScreenshot}
                    aria-label="Download element screenshot"
                    title="Download element screenshot"
                    disabled={isInspectorScreenshotCapturing}
                  >
                    <Download aria-hidden size={16} />
                  </button>
                )}
              </div>
              <figcaption className="preview-inspector__screenshot-meta">
                {inspectorScreenshot ? (
                  <>
                    <span className="preview-inspector__screenshot-dimensions">
                      {inspectorScreenshot.width} × {inspectorScreenshot.height} px
                    </span>
                    {inspectorScreenshot.note && (
                      <span className="preview-inspector__screenshot-note">{inspectorScreenshot.note}</span>
                    )}
                    <span
                      className="preview-inspector__screenshot-filename"
                      title={inspectorScreenshot.filename}
                    >
                      {inspectorScreenshot.filename}
                    </span>
                  </>
                ) : (
                  <span className="preview-inspector__screenshot-placeholder-text">
                    Capture an element to generate a screenshot.
                  </span>
                )}
              </figcaption>
              {inspectorScreenshot && (
                <div className="preview-inspector__report">
                  <label className="preview-inspector__report-label" htmlFor={inspectorReportNoteId}>
                    Report note
                  </label>
                  <div className="preview-inspector__report-input">
                    <input
                      id={inspectorReportNoteId}
                      type="text"
                      className="preview-inspector__report-field"
                      value={inspectorCaptureNote}
                      onChange={handleInspectorCaptureNoteChange}
                      placeholder="Add note for issue report"
                    />
                    <button
                      type="button"
                      className="preview-inspector__report-action"
                      onClick={handleAddInspectorCaptureToReport}
                      title="Add capture to issue report"
                      aria-label="Add capture to issue report"
                    >
                      <PlusCircle aria-hidden size={16} />
                    </button>
                  </div>
                </div>
              )}
            </figure>
          </div>
          </div>
        )}
      </section>
    </div>
  );
};

export default PreviewInspectorPanel;
