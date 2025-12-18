import { Download, FolderOutput, Loader, X } from "lucide-react";
import clsx from "clsx";
import { ResponsiveDialog } from "@shared/layout";
import { selectors } from "@constants/selectors";
import type { MutableRefObject } from "react";
import type { ReplayMovieSpec } from "@/types/export";
import type { ExportPreviewMetrics } from "./exportPreview";
import type { ExportDimensionPreset, ExportFormat, ExportFormatOption } from "./exportConfig";
import { DEFAULT_EXPORT_HEIGHT, DEFAULT_EXPORT_WIDTH, EXPORT_EXTENSIONS } from "./exportConfig";

interface ExportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  dialogTitleId: string;
  dialogDescriptionId: string;
  exportFormat: ExportFormat;
  setExportFormat: (format: ExportFormat) => void;
  isBinaryExport: boolean;
  exportFormatOptions: ExportFormatOption[];
  dimensionPresetOptions: Array<{
    id: ExportDimensionPreset;
    label: string;
    width?: number;
    height?: number;
    description?: string;
  }>;
  dimensionPreset: ExportDimensionPreset;
  setDimensionPreset: (value: ExportDimensionPreset) => void;
  selectedDimensions: { width: number; height: number };
  customWidthInput: string;
  customHeightInput: string;
  setCustomWidthInput: (value: string) => void;
  setCustomHeightInput: (value: string) => void;
  exportFileStem: string;
  setExportFileStem: (value: string) => void;
  defaultExportFileStem: string;
  finalFileName: string;
  supportsFileSystemAccess: boolean;
  useNativeFilePicker: boolean;
  setUseNativeFilePicker: (value: boolean) => void;
  preparedMovieSpec: ReplayMovieSpec | null;
  composerPreviewUrl: string;
  firstFramePreviewUrl: string | null;
  firstFrameLabel: string | null;
  previewComposerRef: MutableRefObject<HTMLIFrameElement | null>;
  composerPreviewWindowRef: MutableRefObject<Window | null>;
  setIsPreviewComposerReady: (value: boolean) => void;
  setPreviewComposerError: (value: string | null) => void;
  composerPreviewOriginRef: MutableRefObject<string | null>;
  isPreviewComposerReady: boolean;
  previewComposerError: string | null;
  isExporting: boolean;
  isExportPreviewLoading: boolean;
  replayFramesLength: number;
  exportStatusMessage: string;
  estimatedFrameCount?: number;
  estimatedDurationSeconds: number | null;
  activeSpecId: string | null;
  previewMetrics: ExportPreviewMetrics;
  formatSeconds: (value: number) => string;
}

/**
  * ExportDialog isolates the replay export modal from the execution viewer, keeping the
  * export UX cohesive while allowing the parent to own state and orchestration.
  */
export function ExportDialog({
  isOpen,
  onClose,
  onConfirm,
  dialogTitleId,
  dialogDescriptionId,
  exportFormat,
  setExportFormat,
  isBinaryExport,
  exportFormatOptions,
  dimensionPresetOptions,
  dimensionPreset,
  setDimensionPreset,
  selectedDimensions,
  customWidthInput,
  customHeightInput,
  setCustomWidthInput,
  setCustomHeightInput,
  exportFileStem,
  setExportFileStem,
  defaultExportFileStem,
  finalFileName,
  supportsFileSystemAccess,
  useNativeFilePicker,
  setUseNativeFilePicker,
  preparedMovieSpec,
  composerPreviewUrl,
  firstFramePreviewUrl,
  firstFrameLabel,
  previewComposerRef,
  composerPreviewWindowRef,
  setIsPreviewComposerReady,
  setPreviewComposerError,
  composerPreviewOriginRef,
  isPreviewComposerReady,
  previewComposerError,
  isExporting,
  isExportPreviewLoading,
  replayFramesLength,
  exportStatusMessage,
  estimatedFrameCount,
  estimatedDurationSeconds,
  activeSpecId,
  previewMetrics,
  formatSeconds,
}: ExportDialogProps) {
  const isJsonExport = exportFormat === "json";

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={dialogTitleId}
      size="wide"
      overlayClassName="z-50"
      className="bg-flow-node border border-gray-800 shadow-2xl max-h-[90vh] flex flex-col"
    >
      <div className="flex items-center justify-between border-b border-gray-800 px-6 py-4">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-flow-accent/15 text-flow-accent">
            <Download size={18} />
          </span>
          <div>
            <h2 id={dialogTitleId} className="text-lg font-semibold text-white">
              Export replay
            </h2>
            <p id={dialogDescriptionId} className="text-sm text-gray-400">
              Choose format, naming, and destination for this execution.
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

      <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6" aria-describedby={dialogDescriptionId}>
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-semibold text-white">Format</h3>
              <p className="text-xs text-gray-400">Pick the export target that fits your workflow.</p>
            </div>
            <span className="text-[11px] uppercase tracking-[0.2em] text-gray-500">
              {isBinaryExport ? "Direct download" : "Data bundle"}
            </span>
          </div>
          <div className="grid gap-3 md:grid-cols-3">
            {exportFormatOptions.map((option) => {
              const isSelected = option.id === exportFormat;
              const Icon = option.icon;
              return (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => setExportFormat(option.id)}
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

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">First frame preview</h3>
            {firstFrameLabel && <span className="text-xs text-slate-400">{firstFrameLabel}</span>}
          </div>
          <div className="overflow-hidden rounded-xl border border-white/10 bg-slate-900/60">
            {preparedMovieSpec ? (
              <div
                className="relative w-full"
                style={{
                  aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                }}
              >
                <iframe
                  key={`${preparedMovieSpec?.execution?.execution_id ?? "preview"}-composer`}
                  ref={(node) => {
                    previewComposerRef.current = node;
                    const nextWindow = node?.contentWindow ?? null;
                    if (composerPreviewWindowRef.current !== nextWindow) {
                      composerPreviewWindowRef.current = nextWindow;
                      if (!nextWindow) {
                        setIsPreviewComposerReady(false);
                        setPreviewComposerError(null);
                        composerPreviewOriginRef.current = null;
                      }
                    }
                  }}
                  src={composerPreviewUrl}
                  title="Replay preview"
                  className="absolute inset-0 h-full w-full border-0"
                  allow="clipboard-read; clipboard-write"
                />
                {!isPreviewComposerReady && firstFramePreviewUrl && (
                  <img
                    src={firstFramePreviewUrl}
                    alt="First frame snapshot"
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                )}
                {!isPreviewComposerReady && (
                  <div className="absolute inset-0 flex items-center justify-center bg-slate-950/60 text-xs uppercase tracking-[0.3em] text-slate-300">
                    {previewComposerError ?? "Loading preview…"}
                  </div>
                )}
              </div>
            ) : firstFramePreviewUrl ? (
              <img
                src={firstFramePreviewUrl}
                alt="First frame preview"
                className="block w-full object-cover"
                style={{
                  aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                }}
              />
            ) : (
              <div className="flex h-40 items-center justify-center text-sm text-slate-400">Preview unavailable</div>
            )}
            <div className="flex items-center justify-between border-t border-white/5 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-slate-500">
              <span>
                {selectedDimensions.width}×{selectedDimensions.height} px
              </span>
              <span>Canvas</span>
            </div>
          </div>
        </section>

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">Dimensions</h3>
            <span className="text-xs text-slate-400">
              {selectedDimensions.width}×{selectedDimensions.height} px
            </span>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {dimensionPresetOptions.map((option) => {
              const isSelected = dimensionPreset === option.id;
              return (
                <button
                  key={option.id}
                  type="button"
                  onClick={() => setDimensionPreset(option.id)}
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
          {dimensionPreset === "custom" && (
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="flex flex-col gap-1">
                <span className="text-xs text-slate-400">Width (px)</span>
                <input
                  type="number"
                  min={320}
                  step={10}
                  value={customWidthInput}
                  onChange={(event) => {
                    setDimensionPreset("custom");
                    setCustomWidthInput(event.target.value.replace(/[^0-9]/g, ""));
                  }}
                  className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                  placeholder={String(DEFAULT_EXPORT_WIDTH)}
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
                    setDimensionPreset("custom");
                    setCustomHeightInput(event.target.value.replace(/[^0-9]/g, ""));
                  }}
                  className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                  placeholder={String(DEFAULT_EXPORT_HEIGHT)}
                />
              </label>
            </div>
          )}
        </section>

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">File name</h3>
            <span className="text-xs text-gray-500">Extension follows format</span>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="text"
              value={exportFileStem}
              onChange={(event) => setExportFileStem(event.target.value)}
              className="flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-gray-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
              placeholder={defaultExportFileStem}
            />
            <span className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-xs text-gray-300">
              .{EXPORT_EXTENSIONS[exportFormat]}
            </span>
          </div>
          <p className="text-xs text-gray-500">
            Final file name: <code className="text-gray-300">{finalFileName}</code>
          </p>
        </section>

        {isJsonExport ? (
          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">Destination</h3>
              {!supportsFileSystemAccess && <span className="text-xs text-amber-400">Browser will download to defaults</span>}
            </div>
            <label
              className={clsx(
                "flex items-center gap-3 rounded-lg border px-4 py-3 text-sm transition",
                useNativeFilePicker && supportsFileSystemAccess
                  ? "border-flow-accent/60 bg-flow-accent/10 text-white"
                  : "border-white/10 bg-slate-900/60 text-slate-300",
                !supportsFileSystemAccess && "opacity-60",
              )}
            >
              <input
                type="checkbox"
                className="h-4 w-4 rounded border-white/20 bg-slate-900 text-flow-accent focus:ring-flow-accent"
                checked={useNativeFilePicker && supportsFileSystemAccess}
                onChange={(event) => setUseNativeFilePicker(event.target.checked)}
                disabled={!supportsFileSystemAccess}
              />
              <div className="flex flex-col">
                <span className="font-medium">
                  {supportsFileSystemAccess ? "Choose save location on export" : "Use browser default download folder"}
                </span>
                <span className="text-xs text-slate-400">
                  {supportsFileSystemAccess
                    ? "Open your OS save dialog to pick the destination each time."
                    : "Your browser will download using its configured destination."}
                </span>
              </div>
            </label>
          </section>
        ) : (
          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">Download</h3>
              <span className="text-xs text-slate-400">Server-rendered</span>
            </div>
            <div className="space-y-2 rounded-lg border border-white/10 bg-slate-900/60 p-3 text-xs text-slate-300">
              <p>Your replay renders on the API using the same metadata that powers the in-app player.</p>
              <p>We’ll trigger a download automatically once rendering finishes.</p>
            </div>
          </section>
        )}

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-white">Replay summary</h3>
            <span className="text-xs text-gray-500">{exportStatusMessage}</span>
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
              <div className="text-sm text-slate-300">{exportStatusMessage}</div>
              {activeSpecId && (
                <div className="mt-1 text-[10px] uppercase tracking-[0.22em] text-slate-500">Spec {activeSpecId.slice(0, 8)}</div>
              )}
              {previewMetrics.assetCount > 0 && (
                <div className="mt-1 text-[11px] text-slate-500">
                  {previewMetrics.assetCount} {previewMetrics.assetCount === 1 ? "asset" : "assets"}
                </div>
              )}
            </div>
          </div>
        </section>
      </div>

      <div className="flex items-center justify-between border-t border-gray-800 bg-flow-bg/60 px-6 py-4">
        <div className="text-xs text-gray-400">
          {isJsonExport
            ? "Exports the replay package with chosen theming so tooling can recreate animations."
            : "Rendering runs on the API and downloads the finished media when ready."}
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
              !isJsonExport && "bg-gradient-to-r from-flow-accent to-sky-500",
            )}
            disabled={isJsonExport ? isExporting || isExportPreviewLoading : isExporting || replayFramesLength === 0}
            data-testid={
              isExporting ? selectors.executions.export.inProgress : selectors.executions.actions.exportConfirmButton
            }
          >
            {isJsonExport ? (
              isExporting || isExportPreviewLoading ? (
                <>
                  <Loader size={16} className="animate-spin" />
                  {isExportPreviewLoading ? "Preparing…" : "Exporting…"}
                </>
              ) : (
                <>
                  <FolderOutput size={16} />
                  Export replay
                </>
              )
            ) : isExporting ? (
              <>
                <Loader size={16} className="animate-spin" />
                Rendering…
              </>
            ) : (
              <>
                <Download size={16} />
                Export replay
              </>
            )}
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}

export default ExportDialog;
