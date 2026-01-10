import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { Execution, TimelineFrame } from "../store";
import type { Export } from "@/domains/exports";
import type { ReplayMovieSpec } from "@/types/export";
// Use config from the unified export domain (single source of truth)
import {
  DIMENSION_PRESET_CONFIG,
  EXPORT_EXTENSIONS,
  EXPORT_FORMAT_OPTIONS,
  EXPORT_RENDER_SOURCE_OPTIONS,
  EXPORT_STYLIZATION_OPTIONS,
  isBinaryFormat,
  sanitizeFileStem,
  type ExportDimensionPreset,
  type ExportFormat,
  type ExportStylization,
} from "@/domains/executions/export";
import { useReplayCustomization } from "./useReplayCustomization";
import { useReplaySpec } from "./useReplaySpec";
import { fetchExecutionExportPreview } from "./exportPreview";
import { formatSeconds } from "./useExecutionHeartbeat";
import {
  describePreviewStatusMessage,
  normalizePreviewStatus,
} from "@/domains/exports/presentation";
import { resolveUrl } from "@utils/executionTypeMappers";
import { toast } from "react-hot-toast";
import { useComposerApiBase } from "./useComposerApiBase";
import { applyReplayStyleToSpec, normalizeReplayStyle } from "@/domains/replay-style";
// Extracted export domain modules
import {
  DEFAULT_DIMENSIONS,
  scaleMovieSpec,
  extractCanvasDimensions,
  resolveDimensionPreset,
  parseDimensionInput,
  type DimensionPresetId,
} from "../export/transformations";
import {
  buildExportRequest,
  buildExportOverrides,
  executeServerExport,
  getLastOutputDir,
} from "../export/api";
import { useRecordedVideoStatus, useExportProgress } from "../export/hooks";

/**
 * Initial settings for pre-populating the export dialog (used when editing an existing export).
 */
type InitialExportSettings = {
  format?: ExportFormat;
  fileStem?: string;
};

type UseExecutionExportParams = {
  execution: Execution;
  replayFrames: TimelineFrame[];
  workflowName: string;
  replayCustomization: ReturnType<typeof useReplayCustomization>;
  createExport: (input: {
    executionId: string;
    workflowId?: string | null;
    name: string;
    format: "mp4" | "gif" | "json" | "html";
    settings?: Record<string, unknown>;
    fileSizeBytes?: number;
    durationMs?: number | null;
    frameCount?: number | null;
  }) => Promise<Export | null>;
  /** Optional initial settings for pre-populating the export dialog (used when editing). */
  initialSettings?: InitialExportSettings;
};

type ExecutionExportPreview = {
  executionId: string;
  specId?: string;
  status: ReturnType<typeof normalizePreviewStatus>;
  message?: string;
  capturedFrameCount: number;
  availableAssetCount: number;
  totalDurationMs: number;
  package?: ReplayMovieSpec;
};

export const useExecutionExport = ({
  execution,
  replayFrames,
  workflowName: _workflowName,
  replayCustomization,
  createExport: _createExport,
  initialSettings,
}: UseExecutionExportParams) => {
  const {
    replayChromeTheme,
    replayPresentation,
    replayBackground,
    replayBackgroundTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    replayBrowserScale,
    replayRenderSource,
    setReplayRenderSource,
  } = replayCustomization;
  const {
    movieSpec,
    movieSpecError,
    isMovieSpecLoading,
    previewMetrics,
    setPreviewMetrics,
    activeSpecId,
    setActiveSpecId,
    setMovieSpecError,
  } = useReplaySpec({
    executionId: execution.id,
    executionStatus: execution.status,
    timelineLength: execution.timeline?.length,
  });
  const { composerApiBase, setComposerApiBase } = useComposerApiBase(execution.id);
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<ExportFormat>(
    () => initialSettings?.format ?? "mp4",
  );
  const [exportFormats, setExportFormats] = useState<ExportFormat[]>(
    () => initialSettings?.format ? [initialSettings.format] : ["mp4"],
  );
  const [exportFileStem, setExportFileStem] = useState<string>(
    () => initialSettings?.fileStem ?? `browser-automation-replay-${execution.id.slice(0, 8)}`,
  );
  // Output directory for server-side exports (loaded from localStorage)
  const [outputDir, setOutputDir] = useState<string>(() => getLastOutputDir());
  // Track active export ID for progress subscription
  const [activeExportId, setActiveExportId] = useState<string | null>(null);
  const [dimensionPreset, setDimensionPreset] =
    useState<ExportDimensionPreset>("spec");
  const [customWidthInput, setCustomWidthInput] = useState<string>(() =>
    String(DEFAULT_DIMENSIONS.width),
  );
  const [customHeightInput, setCustomHeightInput] = useState<string>(() =>
    String(DEFAULT_DIMENSIONS.height),
  );
  const [exportPreview, setExportPreview] =
    useState<ExecutionExportPreview | null>(null);
  const [exportPreviewExecutionId, setExportPreviewExecutionId] = useState<
    string | null
  >(null);
  const [isExportPreviewLoading, setIsExportPreviewLoading] = useState(false);
  const [exportPreviewError, setExportPreviewError] = useState<string | null>(
    null,
  );
  const previewComposerRef = useRef<HTMLIFrameElement | null>(null);
  const composerPreviewWindowRef = useRef<Window | null>(null);
  const composerPreviewOriginRef = useRef<string | null>(null);
  const [isPreviewComposerReady, setIsPreviewComposerReady] = useState(false);
  const [previewComposerError, setPreviewComposerError] = useState<
    string | null
  >(null);
  const [isExporting, setIsExporting] = useState(false);
  const [showExportSuccess, setShowExportSuccess] = useState(false);
  const [lastCreatedExport, setLastCreatedExport] = useState<Export | null>(
    null,
  );
  // Use extracted hook for recorded video status
  const recordedVideoStatus = useRecordedVideoStatus({
    executionId: execution.id,
  });

  // Export progress tracking via WebSocket
  const { progress: exportProgress, reset: resetExportProgress } = useExportProgress({
    exportId: activeExportId,
    executionId: activeExportId ? undefined : execution.id,
    onComplete: (progress) => {
      toast.success(`Export saved to ${progress.storage_url}`);
      setIsExportDialogOpen(false);
      setActiveExportId(null);
      setIsExporting(false);
    },
    onError: (error) => {
      toast.error(error);
      setActiveExportId(null);
      setIsExporting(false);
    },
  });

  // Stylization state - defaults to "stylized" to match current behavior
  const [stylization, setStylization] = useState<ExportStylization>("stylized");
  const isStylized = stylization === "stylized";

  const defaultExportFileStem = useMemo(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
    [execution.id],
  );

  // Use extracted function for canvas dimension extraction
  const specDimensions = useMemo(
    () => extractCanvasDimensions(movieSpec?.presentation),
    [movieSpec?.presentation],
  );
  const specCanvasWidth = specDimensions.width;
  const specCanvasHeight = specDimensions.height;

  const composerUrl = useMemo(() => {
    const embedOrigin =
      typeof window !== "undefined"
        ? window.location.origin
        : "http://localhost";
    const url = new URL("/export/composer.html", embedOrigin);
    url.searchParams.set("mode", "embedded");
    url.searchParams.set("executionId", execution.id);
    url.searchParams.set("apiBase", composerApiBase ?? embedOrigin);
    return url.toString();
  }, [composerApiBase, execution.id]);

  const composerPreviewUrl = useMemo(() => {
    const embedOrigin =
      typeof window !== "undefined"
        ? window.location.origin
        : "http://localhost";
    const url = new URL("/export/composer.html", embedOrigin);
    url.searchParams.set("mode", "embedded");
    url.searchParams.set("executionId", execution.id);
    url.searchParams.set("apiBase", composerApiBase ?? embedOrigin);
    url.searchParams.set("context", "preview");
    return url.toString();
  }, [composerApiBase, execution.id]);

  useEffect(() => {
    setCustomWidthInput(String(specCanvasWidth));
    setCustomHeightInput(String(specCanvasHeight));
    setDimensionPreset("spec");
  }, [execution.id, specCanvasWidth, specCanvasHeight]);

  useEffect(() => {
    setIsExportDialogOpen(false);
    setExportPreview(null);
    setExportPreviewExecutionId(null);
    setExportPreviewError(null);
    setExportFileStem(defaultExportFileStem);
    setActiveExportId(null);
    resetExportProgress();
  }, [defaultExportFileStem, resetExportProgress]);

  // Fallback render source to "auto" when recorded video becomes unavailable
  useEffect(() => {
    if (!recordedVideoStatus.loading && !recordedVideoStatus.available && replayRenderSource === "recorded_video") {
      setReplayRenderSource("auto");
    }
  }, [recordedVideoStatus.loading, recordedVideoStatus.available, replayRenderSource, setReplayRenderSource]);

  useEffect(() => {
    if (!isExportDialogOpen) {
      return;
    }
    if (exportPreview && exportPreviewExecutionId === execution.id) {
      return;
    }
    let isCancelled = false;
    setIsExportPreviewLoading(true);
    setExportPreviewError(null);
    void (async () => {
      try {
        const {
          preview,
          status,
          metrics,
          movieSpec,
        } = await fetchExecutionExportPreview(execution.id);

        if (isCancelled) {
          return;
        }

        const executionIdForPreview =
          preview.executionId || execution.id;
        const specId = (preview.specId && preview.specId.trim()) || executionIdForPreview;

        const parsed: ExecutionExportPreview = {
          executionId: executionIdForPreview,
          specId,
          status,
          message: describePreviewStatusMessage(
            status,
            preview.message,
            metrics,
          ),
          capturedFrameCount: metrics.capturedFrames,
          availableAssetCount: metrics.assetCount,
          totalDurationMs: metrics.totalDurationMs,
          package: movieSpec ?? undefined,
        };
        setExportPreview(parsed);
        setPreviewMetrics(metrics);
        setActiveSpecId(specId);
        setExportPreviewExecutionId(parsed.executionId);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        const message =
          error instanceof Error
            ? error.message
            : "Failed to prepare export preview";
        setExportPreviewError(message);
        setExportPreview(null);
      } finally {
        if (!isCancelled) {
          setIsExportPreviewLoading(false);
        }
      }
    })();
    return () => {
      isCancelled = true;
    };
  }, [
    execution.id,
    exportPreview,
    exportPreviewExecutionId,
    isExportDialogOpen,
  ]);

  const getSanitizedFileStem = useCallback(
    () => sanitizeFileStem(exportFileStem, defaultExportFileStem),
    [exportFileStem, defaultExportFileStem],
  );

  const buildOutputFileName = useCallback(
    (extension: string) => {
      const stem = getSanitizedFileStem();
      return `${stem}.${extension}`;
    },
    [getSanitizedFileStem],
  );

  const finalFileName = useMemo(
    () => buildOutputFileName(EXPORT_EXTENSIONS[exportFormat]),
    [buildOutputFileName, exportFormat],
  );

  const timelineForReplay = useMemo(() => {
    return (execution.timeline ?? []).filter((frame) => {
      if (!frame) {
        return false;
      }
      const type =
        typeof frame.stepType === "string" ? frame.stepType.toLowerCase() : "";
      return type !== "screenshot";
    });
  }, [execution.timeline]);

  const replayFramesWithFallback = useMemo<TimelineFrame[]>(() => {
    return (replayFrames.length > 0 ? replayFrames : timelineForReplay).map(
      (frame: TimelineFrame, index: number) => {
        const resolvedScreenshot = frame.screenshot
          ? {
              ...frame.screenshot,
              url: resolveUrl(frame.screenshot.url),
              thumbnailUrl: resolveUrl(frame.screenshot.thumbnailUrl),
            }
          : undefined;

        const hasScreenshot = Boolean(
          resolvedScreenshot?.url || resolvedScreenshot?.thumbnailUrl,
        );

        return {
          ...frame,
          id: frame.id || `frame-${index}`,
          screenshot: hasScreenshot ? resolvedScreenshot : undefined,
        };
      },
    );
  }, [replayFrames, timelineForReplay]);

  const replayStyle = useMemo(
    () =>
      normalizeReplayStyle({
        chromeTheme: replayChromeTheme,
        presentation: replayPresentation,
        background: replayBackground,
        cursorTheme: replayCursorTheme,
        cursorInitialPosition: replayCursorInitialPosition,
        cursorClickAnimation: replayCursorClickAnimation,
        cursorScale: replayCursorScale,
        browserScale: replayBrowserScale,
      }),
    [
      replayBackground,
      replayBrowserScale,
      replayChromeTheme,
      replayCursorClickAnimation,
      replayCursorInitialPosition,
      replayCursorScale,
      replayCursorTheme,
      replayPresentation,
    ],
  );

  const decoratedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    if (!movieSpec) {
      return null;
    }
    return applyReplayStyleToSpec(movieSpec, replayStyle);
  }, [
    movieSpec,
    replayStyle,
  ]);

  const dimensionPresetOptions = useMemo(
    () => [
      {
        id: "spec" as ExportDimensionPreset,
        label: `Scenario (${specCanvasWidth}×${specCanvasHeight})`,
        width: specCanvasWidth,
        height: specCanvasHeight,
        description: "Match recorded canvas",
      },
      {
        id: "1080p" as ExportDimensionPreset,
        label: DIMENSION_PRESET_CONFIG["1080p"].label,
        width: DIMENSION_PRESET_CONFIG["1080p"].width,
        height: DIMENSION_PRESET_CONFIG["1080p"].height,
        description: "Great for polished exports",
      },
      {
        id: "720p" as ExportDimensionPreset,
        label: DIMENSION_PRESET_CONFIG["720p"].label,
        width: DIMENSION_PRESET_CONFIG["720p"].width,
        height: DIMENSION_PRESET_CONFIG["720p"].height,
        description: "Smaller file size",
      },
      {
        id: "custom" as ExportDimensionPreset,
        label: "Custom size",
        width: Number.parseInt(customWidthInput, 10) || DEFAULT_DIMENSIONS.width,
        height: Number.parseInt(customHeightInput, 10) || DEFAULT_DIMENSIONS.height,
        description: "Set explicit pixel dimensions",
      },
    ],
    [customHeightInput, customWidthInput, specCanvasHeight, specCanvasWidth],
  );

  // Use extracted pure function for dimension preset resolution
  const customDimensions = useMemo(
    () => ({
      width: parseDimensionInput(customWidthInput, DEFAULT_DIMENSIONS.width),
      height: parseDimensionInput(customHeightInput, DEFAULT_DIMENSIONS.height),
    }),
    [customWidthInput, customHeightInput],
  );
  const selectedDimensions = useMemo(
    () => resolveDimensionPreset(
      dimensionPreset as DimensionPresetId,
      specDimensions,
      customDimensions,
    ),
    [dimensionPreset, specDimensions, customDimensions],
  );

  // Use extracted pure function for movie spec scaling
  const preparedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    const base = decoratedMovieSpec ?? movieSpec;
    if (!base) {
      return null;
    }
    // Use the extracted scaleMovieSpec pure function
    return scaleMovieSpec(base, { targetDimensions: selectedDimensions });
  }, [decoratedMovieSpec, movieSpec, selectedDimensions]);

  const activeMovieSpec = preparedMovieSpec ?? decoratedMovieSpec ?? movieSpec;

  const firstFramePreviewUrl = useMemo(() => {
    const frame = replayFramesWithFallback[0];
    const screenshotUrl = frame?.screenshot?.url
      ? (resolveUrl(frame.screenshot.url) ?? frame.screenshot.url)
      : null;
    if (screenshotUrl) {
      return screenshotUrl;
    }
    const specFrame = activeMovieSpec?.frames?.[0];
    if (specFrame?.screenshot_asset_id && activeMovieSpec?.assets) {
      const asset = activeMovieSpec.assets.find(
        (candidate) => candidate?.id === specFrame.screenshot_asset_id,
      );
      if (asset?.source) {
        return resolveUrl(asset.source) ?? asset.source;
      }
    }
    return null;
  }, [activeMovieSpec, replayFramesWithFallback]);

  const firstFrameLabel = useMemo(() => {
    const frame = replayFramesWithFallback[0];
    if (!frame) {
      return null;
    }
    return (
      frame.nodeId ||
      frame.stepType ||
      (typeof frame.stepIndex === "number"
        ? `Step ${frame.stepIndex + 1}`
        : null)
    );
  }, [replayFramesWithFallback]);

  const exportSummary =
    exportPreview?.package?.summary ?? activeMovieSpec?.summary ?? null;
  const normalizedExportStatus = exportPreview
    ? normalizePreviewStatus(exportPreview.status)
    : movieSpec
      ? "ready"
      : "";
  const exportStatusMessage = isExportPreviewLoading
    ? "Preparing replay data…"
    : describePreviewStatusMessage(
        normalizedExportStatus,
        exportPreviewError ?? exportPreview?.message,
        previewMetrics,
      );
  const movieSpecFrameCount =
    movieSpec?.frames != null ? movieSpec.frames.length : undefined;
  const previewFrameCount =
    previewMetrics.capturedFrames || previewMetrics.capturedFrames === 0
      ? previewMetrics.capturedFrames
      : undefined;
  const estimatedFrameCount =
    exportSummary?.frame_count ??
    movieSpecFrameCount ??
    previewFrameCount ??
    replayFramesWithFallback.length;
  const previewDurationMs =
    previewMetrics.totalDurationMs || previewMetrics.totalDurationMs === 0
      ? previewMetrics.totalDurationMs
      : undefined;
  const specDurationMs = activeMovieSpec?.playback?.duration_ms;
  const estimatedTotalDurationMs =
    exportSummary?.total_duration_ms ??
    specDurationMs ??
    previewDurationMs ??
    null;
  const estimatedDurationSeconds =
    estimatedTotalDurationMs != null ? estimatedTotalDurationMs / 1000 : null;
  // Check if ANY selected format is binary (mp4/gif) - used for stylization applicability
  const isBinaryExport = exportFormats.some((f) => isBinaryFormat(f));

  // Toggle format in the multi-select array
  const toggleExportFormat = useCallback((format: ExportFormat) => {
    setExportFormats((prev) => {
      if (prev.includes(format)) {
        // Don't allow removing the last format
        if (prev.length === 1) {
          return prev;
        }
        return prev.filter((f) => f !== format);
      }
      return [...prev, format];
    });
    // Also update the single format to the first selected (for backwards compatibility)
    setExportFormat(format);
  }, []);

  const handleOpenExportDialog = useCallback(() => {
    if (replayFramesWithFallback.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }
    setExportFormat("mp4");
    setExportFileStem(defaultExportFileStem);
    setOutputDir(getLastOutputDir());
    setActiveExportId(null);
    resetExportProgress();
    setIsExportDialogOpen(true);
  }, [defaultExportFileStem, replayFramesWithFallback.length, resetExportProgress]);

  const handleCloseExportDialog = useCallback(() => {
    setIsExportDialogOpen(false);
  }, []);

  const handleConfirmExport = useCallback(async () => {
    if (replayFramesWithFallback.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }

    if (!outputDir.trim()) {
      toast.error("Please specify an output directory");
      return;
    }

    setIsExporting(true);
    try {
      const baselineSpec = exportPreview?.package ?? movieSpec;
      const exportSpec = preparedMovieSpec ?? baselineSpec;

      // Build overrides only when stylization is enabled and it's a binary format
      const isBinaryFormat = exportFormat === "mp4" || exportFormat === "gif";
      const overrides = isStylized && isBinaryFormat
        ? buildExportOverrides({
            chromeTheme: replayChromeTheme,
            backgroundTheme: replayBackgroundTheme,
            cursorTheme: replayCursorTheme,
            cursorInitialPosition: replayCursorInitialPosition,
            cursorScale: replayCursorScale,
            cursorClickAnimation: replayCursorClickAnimation,
          })
        : undefined;

      // Build the request with output_dir for server-side saving
      const requestPayload = buildExportRequest({
        format: exportFormat,
        fileName: finalFileName,
        renderSource: replayRenderSource,
        movieSpec: exportSpec,
        overrides,
        outputDir: outputDir.trim(),
      });

      // Execute server-side export (returns immediately for binary formats)
      const result = await executeServerExport({
        executionId: execution.id,
        payload: requestPayload,
      });

      // Set active export ID to subscribe to progress updates
      setActiveExportId(result.export_id);
      toast.success("Export started...");

      // Note: Dialog stays open to show progress. It will close when onComplete fires.
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to start export";
      toast.error(message);
      setIsExporting(false);
    }
  }, [
    preparedMovieSpec,
    exportFormat,
    exportPreview,
    execution.id,
    finalFileName,
    movieSpec,
    outputDir,
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorClickAnimation,
    replayCursorInitialPosition,
    replayCursorScale,
    replayCursorTheme,
    replayRenderSource,
    replayFramesWithFallback.length,
    isStylized,
  ]);

  const exportDialogProps = {
    exportFormat,
    setExportFormat,
    exportFormats,
    toggleExportFormat,
    isBinaryExport,
    exportFormatOptions: EXPORT_FORMAT_OPTIONS,
    exportRenderSourceOptions: EXPORT_RENDER_SOURCE_OPTIONS,
    renderSource: replayRenderSource,
    setRenderSource: setReplayRenderSource,
    stylization,
    setStylization,
    isStylized,
    exportStylizationOptions: EXPORT_STYLIZATION_OPTIONS,
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
    outputDir,
    setOutputDir,
    preparedMovieSpec,
    // Export progress state
    exportProgress,
    activeExportId,
    // New preview state for ReplayPlayer
    replayFrames: replayFramesWithFallback,
    replayStyle,
    recordedVideoUrl: null as string | null, // TODO: Fetch from recorded video status
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
    replayFramesLength: replayFramesWithFallback.length,
    exportStatusMessage,
    estimatedFrameCount,
    estimatedDurationSeconds,
    activeSpecId,
    previewMetrics,
    formatSeconds,
    recordedVideoAvailable: recordedVideoStatus.available,
    recordedVideoCount: recordedVideoStatus.count,
    recordedVideoLoading: recordedVideoStatus.loading,
  } as const;

  return {
    composerApiBase,
    setComposerApiBase,
    composerUrl,
    composerPreviewUrl,
    movieSpec,
    movieSpecError,
    setMovieSpecError,
    isMovieSpecLoading,
    preparedMovieSpec,
    activeMovieSpec,
    previewMetrics,
    setPreviewMetrics,
    activeSpecId,
    setActiveSpecId,
    exportDialogProps,
    isExportDialogOpen,
    openExportDialog: handleOpenExportDialog,
    closeExportDialog: handleCloseExportDialog,
    confirmExport: handleConfirmExport,
    isExporting,
    exportStatusMessage,
    showExportSuccess,
    lastCreatedExport,
    dismissExportSuccess: () => {
      setShowExportSuccess(false);
      setLastCreatedExport(null);
    },
    replayFrames: replayFramesWithFallback,
  };
};

export default useExecutionExport;
