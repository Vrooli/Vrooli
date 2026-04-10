/**
 * ExportDialog - Modal dialog for configuring and executing replay exports.
 *
 * This component uses the ExportDialogContext to access state, eliminating
 * prop drilling. It is composed of smaller section components for maintainability.
 *
 * @module export/components
 */

import { useMemo } from "react";
import { Download, FolderOpen, Loader, X } from "lucide-react";
import clsx from "clsx";
import { ResponsiveDialog } from "@shared/layout";
import { selectors } from "@constants/selectors";
import ReplayPlayer from "@/domains/exports/replay/ReplayPlayer";
import { normalizeReplayStyle } from "@/domains/replay-style";
import {
  useExportDialogContext,
  useExportFormatState,
  useExportDimensionState,
  useExportFileState,
  useExportRenderSourceState,
  useExportStylizationState,
  useExportPreviewState,
  useExportProgressState,
  useExportMetricsState,
  useExportDialogActions,
} from "../context";
import { EXPORT_EXTENSIONS, EXPORT_STYLIZATION_OPTIONS } from "../config";
import { ExportStylizationSidebar } from "./ExportStylizationSidebar";

// =============================================================================
// Sub-components for each section
// =============================================================================

function FormatSection() {
  const { formats, toggleFormat, formatOptions } = useExportFormatState();

  const selectedCount = formats.length;
  const formatLabel = selectedCount === 1
    ? formatOptions.find((o) => o.id === formats[0])?.label ?? formats[0]
    : `${selectedCount} formats`;

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-white">Format</h3>
          <p className="text-xs text-gray-400">Select one or more export formats.</p>
        </div>
        <span className="text-[11px] uppercase tracking-[0.2em] text-gray-500">
          {formatLabel}
        </span>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        {formatOptions.map((option) => {
          const isSelected = formats.includes(option.id);
          const Icon = option.icon;
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => toggleFormat(option.id)}
              className={clsx(
                "flex flex-col items-start gap-3 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                isSelected
                  ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_20px_45px_rgba(59,130,246,0.25)]"
                  : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
              )}
            >
              <span className="flex w-full items-center justify-between">
                <span
                  className={clsx(
                    "flex h-10 w-10 items-center justify-center rounded-lg",
                    isSelected ? "bg-flow-accent/80 text-white" : "bg-slate-900/70 text-flow-accent",
                  )}
                >
                  <Icon size={18} />
                </span>
                <span className="flex items-center gap-2">
                  {option.badge && (
                    <span
                      className={clsx(
                        "rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.2em]",
                        isSelected ? "bg-white/20 text-white" : "bg-slate-800 text-slate-300",
                      )}
                    >
                      {option.badge}
                    </span>
                  )}
                  {/* Checkbox indicator */}
                  <span
                    className={clsx(
                      "flex h-5 w-5 items-center justify-center rounded border",
                      isSelected
                        ? "border-flow-accent bg-flow-accent text-white"
                        : "border-gray-500 bg-slate-900/70",
                    )}
                  >
                    {isSelected && (
                      <svg className="h-3 w-3" viewBox="0 0 12 12" fill="none">
                        <path
                          d="M2 6L5 9L10 3"
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        />
                      </svg>
                    )}
                  </span>
                </span>
              </span>
              <div>
                <div className="text-sm font-semibold">{option.label}</div>
                <div className="text-xs text-slate-400">{option.description}</div>
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}

function RenderSourceSection() {
  const {
    source,
    setSource,
    sourceOptions,
    recordedVideoAvailable,
    recordedVideoCount,
    recordedVideoLoading,
  } = useExportRenderSourceState();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-white">Render source</h3>
          <p className="text-xs text-gray-400">Choose how the video is generated.</p>
        </div>
        <span className="text-[11px] uppercase tracking-[0.2em] text-gray-500">Source</span>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        {sourceOptions.map((option) => {
          const isSelected = option.id === source;
          const Icon = option.icon;
          const isUnavailable =
            option.id === "recorded_video" && !recordedVideoLoading && !recordedVideoAvailable;
          return (
            <button
              key={option.id}
              type="button"
              disabled={isUnavailable}
              onClick={() => setSource(option.id)}
              className={clsx(
                "flex flex-col items-start gap-3 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                isSelected && !isUnavailable
                  ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_20px_45px_rgba(59,130,246,0.25)]"
                  : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                isUnavailable && "cursor-not-allowed opacity-60",
              )}
            >
              <span
                className={clsx(
                  "flex h-10 w-10 items-center justify-center rounded-lg",
                  isSelected && !isUnavailable
                    ? "bg-flow-accent/80 text-white"
                    : "bg-slate-900/70 text-flow-accent",
                )}
              >
                <Icon size={18} />
              </span>
              <div>
                <div className="text-sm font-semibold">{option.label}</div>
                <div className="text-xs text-slate-400">{option.description}</div>
              </div>
            </button>
          );
        })}
      </div>
      <div className="text-xs text-slate-400">
        {recordedVideoLoading && "Checking for recorded video artifacts…"}
        {!recordedVideoLoading && recordedVideoAvailable && (
          <>Recorded video available ({recordedVideoCount}).</>
        )}
        {!recordedVideoLoading && !recordedVideoAvailable && (
          <>No recorded video found for this execution.</>
        )}
      </div>
    </section>
  );
}

function StylizationSection() {
  const { stylization, setStylization, isStylized } = useExportStylizationState();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-white">Style</h3>
          <p className="text-xs text-gray-400">Choose how the output looks.</p>
        </div>
        <span className="text-[11px] uppercase tracking-[0.2em] text-gray-500">
          {isStylized ? "Enhanced" : "Original"}
        </span>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        {EXPORT_STYLIZATION_OPTIONS.map((option) => {
          const isSelected = option.id === stylization;
          const Icon = option.icon;
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => setStylization(option.id)}
              className={clsx(
                "flex flex-col items-start gap-3 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                isSelected
                  ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_20px_45px_rgba(59,130,246,0.25)]"
                  : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
              )}
            >
              <span
                className={clsx(
                  "flex h-10 w-10 items-center justify-center rounded-lg",
                  isSelected
                    ? "bg-flow-accent/80 text-white"
                    : "bg-slate-900/70 text-flow-accent",
                )}
              >
                <Icon size={18} />
              </span>
              <div>
                <div className="text-sm font-semibold">{option.label}</div>
                <div className="text-xs text-slate-400">{option.description}</div>
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}

function PreviewSection() {
  const { selectedDimensions } = useExportDimensionState();
  const { source, recordedVideoAvailable } = useExportRenderSourceState();
  const { isStylized } = useExportStylizationState();
  const {
    replayFrames,
    replayStyle,
    recordedVideoUrl,
    firstFramePreviewUrl,
    firstFrameLabel,
  } = useExportPreviewState();

  // Determine effective source (resolve "auto" to recording or slideshow)
  const effectiveSource = useMemo(() => {
    if (source === "auto") {
      return recordedVideoAvailable ? "recorded_video" : "replay_frames";
    }
    return source;
  }, [source, recordedVideoAvailable]);

  // Build preview style based on stylization toggle
  // When stylized, use full replayStyle; when raw, use minimal style
  const previewStyle = useMemo(() => {
    if (isStylized) {
      return replayStyle;
    }
    // Raw style: no chrome, no background, no cursor styling
    return normalizeReplayStyle({
      chromeTheme: "none",
      presentation: "plain",
      background: "none",
      cursorTheme: "system",
      cursorClickAnimation: "none",
    });
  }, [isStylized, replayStyle]);

  // Determine what type of preview we're showing
  const showRecordedVideo = effectiveSource === "recorded_video" && recordedVideoUrl;
  const showReplayPlayer = replayFrames.length > 0;

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-white">Preview</h3>
          <p className="text-xs text-gray-400">
            {effectiveSource === "recorded_video" ? "Screen recording" : "Slideshow replay"}
          </p>
        </div>
        {firstFrameLabel && <span className="text-xs text-slate-400">{firstFrameLabel}</span>}
      </div>
      <div className="overflow-hidden rounded-xl border border-white/10 bg-slate-900/60">
        {showRecordedVideo ? (
          // Video player for recorded video
          <div
            className="relative w-full"
            style={{
              aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
            }}
          >
            <video
              src={recordedVideoUrl}
              controls
              autoPlay={false}
              loop
              className="absolute inset-0 h-full w-full object-contain bg-black"
            />
          </div>
        ) : showReplayPlayer ? (
          // ReplayPlayer for slideshow
          <div
            className="relative w-full"
            style={{
              aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
            }}
          >
            <ReplayPlayer
              frames={replayFrames}
              replayStyle={previewStyle}
              presentationMode="default"
              presentationFit="contain"
              autoPlay={false}
              loop={true}
            />
          </div>
        ) : firstFramePreviewUrl ? (
          // Fallback to static image
          <img
            src={firstFramePreviewUrl}
            alt="First frame preview"
            className="block w-full object-cover"
            style={{
              aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
            }}
          />
        ) : (
          // No preview available
          <div className="flex h-40 items-center justify-center text-sm text-slate-400">
            Preview unavailable
          </div>
        )}
        <div className="flex items-center justify-between border-t border-white/5 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-slate-500">
          <span>
            {selectedDimensions.width}×{selectedDimensions.height} px
          </span>
          <span>{effectiveSource === "recorded_video" ? "Recording" : "Slideshow"}</span>
        </div>
      </div>
    </section>
  );
}

function DimensionsSection() {
  const {
    preset,
    setPreset,
    presetOptions,
    selectedDimensions,
    customWidthInput,
    customHeightInput,
    setCustomWidthInput,
    setCustomHeightInput,
  } = useExportDimensionState();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">Dimensions</h3>
        <span className="text-xs text-slate-400">
          {selectedDimensions.width}×{selectedDimensions.height} px
        </span>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        {presetOptions.map((option) => {
          const isSelected = preset === option.id;
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => setPreset(option.id)}
              className={clsx(
                "flex flex-col items-start gap-1.5 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                isSelected
                  ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_18px_40px_rgba(59,130,246,0.25)]"
                  : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
              )}
            >
              <span className="text-sm font-semibold">{option.label}</span>
              <span className="text-xs text-slate-400">
                {option.id === "custom" ? "Define width & height" : `${option.width}×${option.height} px`}
              </span>
              {option.description && (
                <span className="text-[11px] text-slate-500">{option.description}</span>
              )}
            </button>
          );
        })}
      </div>
      {preset === "custom" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="flex flex-col gap-1">
            <span className="text-xs text-slate-400">Width (px)</span>
            <input
              type="number"
              min={320}
              step={10}
              value={customWidthInput}
              onChange={(event) => {
                setPreset("custom");
                setCustomWidthInput(event.target.value.replace(/[^0-9]/g, ""));
              }}
              className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
              placeholder="1280"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs text-slate-400">Height (px)</span>
            <input
              type="number"
              min={320}
              step={10}
              value={customHeightInput}
              onChange={(event) => {
                setPreset("custom");
                setCustomHeightInput(event.target.value.replace(/[^0-9]/g, ""));
              }}
              className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
              placeholder="720"
            />
          </label>
        </div>
      )}
    </section>
  );
}

function FileNameSection() {
  const { formats } = useExportFormatState();
  const { fileStem, setFileStem, defaultFileStem, outputDir } = useExportFileState();

  // Build list of all files that will be created with full paths
  const effectiveStem = fileStem || defaultFileStem;
  const dir = outputDir.endsWith('/') ? outputDir.slice(0, -1) : outputDir;
  const filesToShow = formats.map((fmt) => `${dir}/${effectiveStem}.${EXPORT_EXTENSIONS[fmt]}`);

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">File name</h3>
        <span className="text-xs text-gray-500">
          {formats.length === 1 ? "1 file" : `${formats.length} files`}
        </span>
      </div>
      <input
        type="text"
        value={fileStem}
        onChange={(event) => setFileStem(event.target.value)}
        className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-gray-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
        placeholder={defaultFileStem}
      />
      <div className="space-y-1">
        <ul className="space-y-0.5">
          {filesToShow.map((filePath) => (
            <li key={filePath} className="text-xs text-gray-300 font-mono truncate" title={filePath}>
              {filePath}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function DestinationSection() {
  const { outputDir, setOutputDir } = useExportFileState();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">Save location</h3>
        <span className="text-[10px] uppercase tracking-wider text-gray-500">Server</span>
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <FolderOpen size={14} className="text-gray-400 flex-shrink-0" />
          <input
            type="text"
            value={outputDir}
            onChange={(event) => setOutputDir(event.target.value)}
            placeholder="data/exports"
            className="flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white font-mono placeholder:text-gray-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
          />
        </div>
        <p className="text-[10px] text-gray-500 pl-6">
          Enter the path where exports should be saved. Directory will be created if it doesn't exist.
        </p>
      </div>
    </section>
  );
}

function SummarySection() {
  const { statusMessage } = useExportProgressState();
  const {
    estimatedFrameCount,
    estimatedDurationSeconds,
    activeSpecId,
    previewMetrics,
  } = useExportMetricsState();
  const { formatSeconds } = useExportDialogContext();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">Replay summary</h3>
        <span className="text-xs text-gray-500">{statusMessage}</span>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
          <div className="text-xs text-gray-400">Frames</div>
          <div className="text-lg font-semibold text-white">{estimatedFrameCount}</div>
        </div>
        <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
          <div className="text-xs text-gray-400">Duration</div>
          <div className="text-lg font-semibold text-white">
            {estimatedDurationSeconds != null ? formatSeconds(estimatedDurationSeconds) : "—"}
          </div>
        </div>
        <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
          <div className="text-xs text-gray-400">Status</div>
          <div className="text-sm text-slate-300">{statusMessage}</div>
          {activeSpecId && (
            <div className="mt-1 text-[10px] uppercase tracking-[0.22em] text-slate-500">
              Spec {activeSpecId.slice(0, 8)}
            </div>
          )}
          {previewMetrics.assetCount > 0 && (
            <div className="mt-1 text-[11px] text-slate-500">
              {previewMetrics.assetCount} {previewMetrics.assetCount === 1 ? "asset" : "assets"}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

/**
 * Export progress bar - shows real-time progress during export rendering.
 */
function ExportProgressSection() {
  const { isExporting, exportProgress, activeExportId } = useExportProgressState();

  // Only show when actively exporting
  if (!isExporting || !activeExportId) {
    return null;
  }

  const stage = exportProgress?.stage ?? "preparing";
  const percent = exportProgress?.progress_percent ?? 0;
  const status = exportProgress?.status ?? "processing";

  // Stage labels for display
  const stageLabels: Record<string, string> = {
    preparing: "Preparing export…",
    capturing: "Capturing frames…",
    encoding: "Encoding video…",
    finalizing: "Finalizing…",
    completed: "Export complete",
    failed: "Export failed",
  };

  const stageLabel = stageLabels[stage] ?? "Processing…";
  const isComplete = status === "completed";
  const isFailed = status === "failed";

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-white">Export progress</h3>
        <span className="text-xs text-gray-500">
          {isComplete ? "Done" : isFailed ? "Error" : `${Math.round(percent)}%`}
        </span>
      </div>

      <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4">
        {/* Progress bar */}
        <div className="relative h-2 w-full overflow-hidden rounded-full bg-slate-800">
          <div
            className={clsx(
              "absolute left-0 top-0 h-full rounded-full transition-all duration-300",
              isComplete
                ? "bg-green-500"
                : isFailed
                  ? "bg-red-500"
                  : "bg-flow-accent",
            )}
            style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
          />
        </div>

        {/* Stage label */}
        <div className="mt-3 flex items-center justify-between">
          <span className={clsx(
            "text-sm",
            isComplete ? "text-green-400" : isFailed ? "text-red-400" : "text-slate-300",
          )}>
            {stageLabel}
          </span>
          {exportProgress?.error && (
            <span className="text-xs text-red-400 max-w-xs truncate" title={exportProgress.error}>
              {exportProgress.error}
            </span>
          )}
        </div>

        {/* File info when complete */}
        {isComplete && exportProgress?.storage_url && (
          <div className="mt-2 text-xs text-slate-400 font-mono truncate" title={exportProgress.storage_url}>
            Saved to: {exportProgress.storage_url}
          </div>
        )}
      </div>
    </section>
  );
}

// =============================================================================
// Main Component
// =============================================================================

interface ExportDialogProps {
  /**
   * Whether the dialog is open.
   */
  isOpen: boolean;
}

/**
 * ExportDialog displays the replay export configuration modal.
 *
 * Uses ExportDialogContext for state management, so must be wrapped
 * in ExportDialogProvider.
 */
export function ExportDialog({ isOpen }: ExportDialogProps) {
  const { titleId, descriptionId, isEditMode } = useExportDialogContext();
  const { formats, isBinaryExport } = useExportFormatState();
  const { isStylized } = useExportStylizationState();
  const { isExporting, isPreviewLoading } = useExportProgressState();
  const { replayFramesLength } = useExportMetricsState();
  const { onClose, onConfirm } = useExportDialogActions();

  // Format type checks for conditional rendering
  const hasJsonFormat = formats.includes("json");

  // Sidebar visibility only depends on stylization toggle - not format
  // This prevents the confusing behavior where selecting JSON closes the sidebar
  const showSidebar = isStylized;
  // Whether stylization will actually apply: true if ANY selected format is binary
  const stylizationApplies = isBinaryExport;

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size={showSidebar ? "export" : "wide"}
      overlayClassName="z-50"
      className="bg-flow-node border border-gray-800 shadow-2xl max-h-[90vh] flex flex-col"
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-gray-800 px-6 py-4">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-flow-accent/15 text-flow-accent">
            <Download size={18} />
          </span>
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-white">
              {isEditMode ? 'Edit export' : 'Export replay'}
            </h2>
            <p id={descriptionId} className="text-sm text-gray-400">
              {isEditMode
                ? 'Modify settings and re-export. A new file will be generated.'
                : 'Choose format, naming, and destination for this execution.'}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded-full p-2 text-gray-400 transition hover:bg-white/5 hover:text-white"
          aria-label="Close export dialog"
        >
          <X size={16} />
        </button>
      </div>

      {/* Content with optional sidebar */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Sidebar - shown when stylization is enabled */}
        {showSidebar && <ExportStylizationSidebar stylizationApplies={stylizationApplies} />}

        {/* Main Content */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6" aria-describedby={descriptionId}>
          <FormatSection />
          {/* Render source and stylization only apply to binary formats (mp4/gif) */}
          {isBinaryExport && (
            <>
              <RenderSourceSection />
              <StylizationSection />
            </>
          )}
          <PreviewSection />
          <DimensionsSection />
          <FileNameSection />
          <DestinationSection />
          <SummarySection />
          <ExportProgressSection />
        </div>
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between border-t border-gray-800 bg-flow-bg/60 px-6 py-4">
        <div className="text-xs text-gray-400">
          {formats.length} {formats.length === 1 ? "format" : "formats"} selected
          {isBinaryExport && " · Video renders on server"}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-white/10 px-4 py-2 text-sm text-gray-300 transition hover:border-white/20 hover:text-white"
            disabled={isExporting}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={clsx(
              "flex items-center gap-2 rounded-lg bg-flow-accent px-4 py-2 text-sm font-semibold text-white transition disabled:cursor-not-allowed disabled:opacity-60",
              isBinaryExport && "bg-gradient-to-r from-flow-accent to-sky-500",
            )}
            disabled={
              isExporting ||
              (hasJsonFormat && isPreviewLoading) ||
              (isBinaryExport && replayFramesLength === 0)
            }
            data-testid={
              isExporting
                ? selectors.executions.export.inProgress
                : selectors.executions.actions.exportConfirmButton
            }
          >
            {isExporting || isPreviewLoading ? (
              <>
                <Loader size={16} className="animate-spin" />
                {isPreviewLoading ? "Preparing…" : isBinaryExport ? "Rendering…" : "Exporting…"}
              </>
            ) : (
              <>
                <Download size={16} />
                Export {formats.length === 1 ? "replay" : `${formats.length} formats`}
              </>
            )}
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default ExportDialog;
