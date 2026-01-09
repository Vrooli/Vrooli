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
  sanitizeFileStem,
  type ExportDimensionPreset,
  type ExportFormat,
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
  executeBinaryExport,
  createJsonBlob,
  triggerBlobDownload,
  saveWithNativeFilePicker,
  supportsFileSystemAccess,
} from "../export/api";
import { useRecordedVideoStatus } from "../export/hooks";

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
  workflowName,
  replayCustomization,
  createExport,
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
  const [exportFormat, setExportFormat] = useState<ExportFormat>("mp4");
  const [exportFileStem, setExportFileStem] = useState<string>(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
  );
  const [useNativeFilePicker, setUseNativeFilePicker] = useState(false);
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
  const fileSystemAccessSupported = supportsFileSystemAccess();

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
    setUseNativeFilePicker(false);
  }, [defaultExportFileStem]);

  useEffect(() => {
    if (exportFormat !== "json" && useNativeFilePicker) {
      setUseNativeFilePicker(false);
    }
  }, [exportFormat, useNativeFilePicker]);

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
  const isBinaryExport = exportFormat !== "json";

  const handleOpenExportDialog = useCallback(() => {
    if (replayFramesWithFallback.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }
    setExportFormat("mp4");
    setExportFileStem(defaultExportFileStem);
    setUseNativeFilePicker(false);
    setIsExportDialogOpen(true);
  }, [defaultExportFileStem, replayFramesWithFallback.length]);

  const handleCloseExportDialog = useCallback(() => {
    setIsExportDialogOpen(false);
  }, []);

  const handleConfirmExport = useCallback(async () => {
    if (exportFormat !== "json") {
      if (replayFramesWithFallback.length === 0) {
        toast.error("Replay not ready to export yet");
        return;
      }

      setIsExporting(true);
      try {
        const baselineSpec = exportPreview?.package ?? movieSpec;
        const exportSpec = preparedMovieSpec ?? baselineSpec;

        // Build overrides for when we don't have a spec with frames
        const overrides = buildExportOverrides({
          chromeTheme: replayChromeTheme,
          backgroundTheme: replayBackgroundTheme,
          cursorTheme: replayCursorTheme,
          cursorInitialPosition: replayCursorInitialPosition,
          cursorScale: replayCursorScale,
          cursorClickAnimation: replayCursorClickAnimation,
        });

        // Use extracted pure function to build the request
        const requestPayload = buildExportRequest({
          format: exportFormat as "mp4" | "gif",
          fileName: finalFileName,
          renderSource: replayRenderSource,
          movieSpec: exportSpec,
          overrides,
        });

        // Use extracted API function for binary export
        const { blob } = await executeBinaryExport({
          executionId: execution.id,
          payload: requestPayload,
        });

        // Use extracted utility for download
        triggerBlobDownload(blob, finalFileName);

        const exportName = exportFileStem || `${workflowName} Export`;
        const createdExport = await createExport({
          executionId: execution.id,
          workflowId: execution.workflowId,
          name: exportName,
          format: exportFormat as "mp4" | "gif" | "json" | "html",
          settings: {
            chromeTheme: replayChromeTheme,
            background: replayBackground,
            cursorTheme: replayCursorTheme,
            cursorScale: replayCursorScale,
            browserScale: replayBrowserScale,
            renderSource: replayRenderSource,
          },
          fileSizeBytes: blob.size,
          durationMs: exportPreview?.totalDurationMs,
          frameCount: exportPreview?.capturedFrameCount,
        });

        if (createdExport) {
          setLastCreatedExport(createdExport);
          setShowExportSuccess(true);
        }
        toast.success("Replay export ready");
        setIsExportDialogOpen(false);
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to export replay";
        toast.error(message);
      } finally {
        setIsExporting(false);
      }
      return;
    }

    if (replayFramesWithFallback.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }

    if (useNativeFilePicker && !fileSystemAccessSupported) {
      toast.error(
        "This browser does not support choosing a destination. Disable \"Choose save location\".",
      );
      return;
    }

    setIsExporting(true);
    try {
      const payload = preparedMovieSpec ?? exportPreview?.package ?? movieSpec;
      if (!payload) {
        toast.error("Replay not ready to export yet");
        setIsExporting(false);
        return;
      }

      // Use extracted utility to create JSON blob
      const blob = createJsonBlob(payload);

      if (useNativeFilePicker && fileSystemAccessSupported) {
        try {
          // Use extracted utility for native file picker
          await saveWithNativeFilePicker(
            blob,
            finalFileName,
            "Replay export (JSON)",
            { "application/json": [".json"] },
          );
          toast.success("Replay export saved");
        } catch (error) {
          if (error instanceof DOMException && error.name === "AbortError") {
            setIsExporting(false);
            return;
          }
          const message =
            error instanceof Error
              ? error.message
              : "Failed to save replay export";
          toast.error(message);
          setIsExporting(false);
          return;
        }
      } else {
        // Use extracted utility for download
        triggerBlobDownload(blob, finalFileName);
        toast.success("Replay download started");
      }

      const exportName = exportFileStem || `${workflowName} Export`;
      const createdExport = await createExport({
        executionId: execution.id,
        workflowId: execution.workflowId,
        name: exportName,
        format: "json",
        settings: {
          chromeTheme: replayChromeTheme,
          background: replayBackground,
          cursorTheme: replayCursorTheme,
          cursorScale: replayCursorScale,
        },
        fileSizeBytes: blob.size,
        durationMs: exportPreview?.totalDurationMs,
        frameCount: exportPreview?.capturedFrameCount,
      });

      if (createdExport) {
        setLastCreatedExport(createdExport);
        setShowExportSuccess(true);
      }

      setIsExportDialogOpen(false);
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to export replay";
      toast.error(message);
    } finally {
      setIsExporting(false);
    }
  }, [
    preparedMovieSpec,
    exportFormat,
    exportPreview,
    execution.id,
    execution.workflowId,
    workflowName,
    exportFileStem,
    finalFileName,
    movieSpec,
    replayBackground,
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorClickAnimation,
    replayCursorInitialPosition,
    replayCursorScale,
    replayCursorTheme,
    replayRenderSource,
    replayFramesWithFallback.length,
    fileSystemAccessSupported,
    useNativeFilePicker,
    createExport,
  ]);

  const exportDialogProps = {
    exportFormat,
    setExportFormat,
    isBinaryExport,
    exportFormatOptions: EXPORT_FORMAT_OPTIONS,
    exportRenderSourceOptions: EXPORT_RENDER_SOURCE_OPTIONS,
    renderSource: replayRenderSource,
    setRenderSource: setReplayRenderSource,
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
    supportsFileSystemAccess: fileSystemAccessSupported,
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
