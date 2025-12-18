import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { Execution, TimelineFrame } from "@stores/executionStore";
import type { Export } from "@stores/exportStore";
import type {
  ReplayMovieFrameRect,
  ReplayMoviePresentation,
  ReplayMovieSpec,
} from "@/types/export";
import { sanitizeFileStem } from "./exportConfig";
import {
  DEFAULT_EXPORT_HEIGHT,
  DEFAULT_EXPORT_WIDTH,
  DIMENSION_PRESET_CONFIG,
  EXPORT_EXTENSIONS,
  EXPORT_FORMAT_OPTIONS,
  type ExportDimensionPreset,
  type ExportFormat,
} from "./exportConfig";
import { useReplayCustomization } from "./useReplayCustomization";
import { useReplaySpec } from "./useReplaySpec";
import { fetchExecutionExportPreview } from "./exportPreview";
import { formatSeconds } from "./useExecutionHeartbeat";
import {
  describePreviewStatusMessage,
  normalizePreviewStatus,
} from "../utils/exportHelpers";
import { resolveUrl } from "@utils/executionTypeMappers";
import { toast } from "react-hot-toast";
import { getConfig } from "@/config";
import { useComposerApiBase } from "./useComposerApiBase";

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
    replayBackgroundTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
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
    String(DEFAULT_EXPORT_WIDTH),
  );
  const [customHeightInput, setCustomHeightInput] = useState<string>(() =>
    String(DEFAULT_EXPORT_HEIGHT),
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
  const supportsFileSystemAccess =
    typeof window !== "undefined" &&
    typeof (window as typeof window & { showSaveFilePicker?: unknown })
      .showSaveFilePicker === "function";

  const defaultExportFileStem = useMemo(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
    [execution.id],
  );

  const specCanvasWidth =
    movieSpec?.presentation?.canvas?.width ??
    movieSpec?.presentation?.viewport?.width ??
    DEFAULT_EXPORT_WIDTH;
  const specCanvasHeight =
    movieSpec?.presentation?.canvas?.height ??
    movieSpec?.presentation?.viewport?.height ??
    DEFAULT_EXPORT_HEIGHT;

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

  const decoratedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    if (!movieSpec) {
      return null;
    }
    const cursorSpec = movieSpec.cursor ?? {};
    const decor = movieSpec.decor ?? {};
    const motion = movieSpec.cursor_motion ?? {};
    return {
      ...movieSpec,
      cursor: {
        ...cursorSpec,
        scale: replayCursorScale,
        initial_position: replayCursorInitialPosition,
        click_animation: replayCursorClickAnimation,
      },
      decor: {
        ...decor,
        chrome_theme: replayChromeTheme,
        background_theme: replayBackgroundTheme,
        cursor_theme: replayCursorTheme,
        cursor_initial_position: replayCursorInitialPosition,
        cursor_click_animation: replayCursorClickAnimation,
        cursor_scale: replayCursorScale,
      },
      cursor_motion: {
        ...motion,
        initial_position: replayCursorInitialPosition,
        click_animation: replayCursorClickAnimation,
        cursor_scale: replayCursorScale,
      },
    };
  }, [
    movieSpec,
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
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
        width: Number.parseInt(customWidthInput, 10) || DEFAULT_EXPORT_WIDTH,
        height: Number.parseInt(customHeightInput, 10) || DEFAULT_EXPORT_HEIGHT,
        description: "Set explicit pixel dimensions",
      },
    ],
    [customHeightInput, customWidthInput, specCanvasHeight, specCanvasWidth],
  );

  const selectedDimensions = useMemo(() => {
    switch (dimensionPreset) {
      case "custom": {
        const parsedWidth = Number.parseInt(customWidthInput, 10);
        const parsedHeight = Number.parseInt(customHeightInput, 10);
        return {
          width:
            Number.isFinite(parsedWidth) && parsedWidth > 0
              ? parsedWidth
              : DEFAULT_EXPORT_WIDTH,
          height:
            Number.isFinite(parsedHeight) && parsedHeight > 0
              ? parsedHeight
              : DEFAULT_EXPORT_HEIGHT,
        };
      }
      case "1080p":
      case "720p": {
        const preset = DIMENSION_PRESET_CONFIG[dimensionPreset];
        return { width: preset.width, height: preset.height };
      }
      case "spec":
      default:
        return { width: specCanvasWidth, height: specCanvasHeight };
    }
  }, [
    customHeightInput,
    customWidthInput,
    dimensionPreset,
    specCanvasHeight,
    specCanvasWidth,
  ]);

  const preparedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    const base = decoratedMovieSpec ?? movieSpec;
    if (!base) {
      return null;
    }
    const cloned = JSON.parse(JSON.stringify(base)) as ReplayMovieSpec;
    const width = selectedDimensions.width;
    const height = selectedDimensions.height;

    const basePresentation: ReplayMoviePresentation | undefined =
      cloned.presentation ?? base.presentation;

    const baseCanvasWidth =
      basePresentation?.canvas?.width && basePresentation.canvas.width > 0
        ? basePresentation.canvas.width
        : width;
    const baseCanvasHeight =
      basePresentation?.canvas?.height && basePresentation.canvas.height > 0
        ? basePresentation.canvas.height
        : height;

    const scaleX = baseCanvasWidth > 0 ? width / baseCanvasWidth : 1;
    const scaleY = baseCanvasHeight > 0 ? height / baseCanvasHeight : 1;

    const scaleDimensions = (
      dims?: { width?: number; height?: number },
      fallback?: { width: number; height: number },
    ): { width: number; height: number } => {
      const fallbackWidth = fallback?.width ?? baseCanvasWidth;
      const fallbackHeight = fallback?.height ?? baseCanvasHeight;
      const sourceWidth =
        dims?.width && dims.width > 0 ? dims.width : fallbackWidth;
      const sourceHeight =
        dims?.height && dims.height > 0 ? dims.height : fallbackHeight;
      return {
        width: Math.round(sourceWidth * scaleX),
        height: Math.round(sourceHeight * scaleY),
      };
    };

    const scaleFrameRect = (
      rect?: ReplayMovieFrameRect | null,
    ): ReplayMovieFrameRect => {
      const source: ReplayMovieFrameRect = rect
        ? { ...rect }
        : {
            x: 0,
            y: 0,
            width: basePresentation?.browser_frame?.width ?? baseCanvasWidth,
            height: basePresentation?.browser_frame?.height ?? baseCanvasHeight,
            radius: basePresentation?.browser_frame?.radius ?? 24,
          };
      return {
        x: Math.round((source.x ?? 0) * scaleX),
        y: Math.round((source.y ?? 0) * scaleY),
        width: Math.round((source.width ?? baseCanvasWidth) * scaleX),
        height: Math.round((source.height ?? baseCanvasHeight) * scaleY),
        radius: source.radius ?? basePresentation?.browser_frame?.radius ?? 24,
      };
    };

    const presentation: ReplayMoviePresentation = {
      canvas: {
        width,
        height,
      },
      viewport: scaleDimensions(basePresentation?.viewport, {
        width: basePresentation?.viewport?.width ?? baseCanvasWidth,
        height: basePresentation?.viewport?.height ?? baseCanvasHeight,
      }),
      browser_frame: scaleFrameRect(basePresentation?.browser_frame),
      device_scale_factor:
        basePresentation?.device_scale_factor &&
        basePresentation.device_scale_factor > 0
          ? basePresentation.device_scale_factor
          : 1,
    };

    cloned.presentation = presentation;

    if (Array.isArray(cloned.frames)) {
      cloned.frames = cloned.frames.map((frame) => ({
        ...frame,
        viewport: scaleDimensions(frame.viewport, {
          width:
            frame.viewport?.width ??
            basePresentation?.viewport?.width ??
            baseCanvasWidth,
          height:
            frame.viewport?.height ??
            basePresentation?.viewport?.height ??
            baseCanvasHeight,
        }),
      }));
    }
    return cloned;
  }, [
    decoratedMovieSpec,
    movieSpec,
    selectedDimensions.height,
    selectedDimensions.width,
  ]);

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
        const { API_URL } = await getConfig();
        const baselineSpec = exportPreview?.package ?? movieSpec;
        const exportSpec = preparedMovieSpec ?? baselineSpec;
        const hasExportFrames =
          exportSpec && Array.isArray(exportSpec.frames)
            ? exportSpec.frames.length > 0
            : false;
        const requestPayload: Record<string, unknown> = {
          format: exportFormat,
          file_name: finalFileName,
        };
        if (hasExportFrames && exportSpec) {
          requestPayload.movie_spec = exportSpec;
        } else {
          requestPayload.overrides = {
            theme_preset: {
              chrome_theme: replayChromeTheme,
              background_theme: replayBackgroundTheme,
            },
            cursor_preset: {
              theme: replayCursorTheme,
              initial_position: replayCursorInitialPosition,
              scale: Number.isFinite(replayCursorScale) ? replayCursorScale : 1,
              click_animation: replayCursorClickAnimation,
            },
          } satisfies Record<string, unknown>;
        }

        const response = await fetch(
          `${API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Accept: exportFormat === "gif" ? "image/gif" : "video/mp4",
            },
            body: JSON.stringify(requestPayload),
          },
        );

        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }

        const blob = await response.blob();
        const downloadUrl = URL.createObjectURL(blob);
        const anchor = document.createElement("a");
        anchor.href = downloadUrl;
        anchor.download = finalFileName;
        document.body.appendChild(anchor);
        anchor.click();
        document.body.removeChild(anchor);
        URL.revokeObjectURL(downloadUrl);

        const exportName = exportFileStem || `${workflowName} Export`;
        const createdExport = await createExport({
          executionId: execution.id,
          workflowId: execution.workflowId,
          name: exportName,
          format: exportFormat as "mp4" | "gif" | "json" | "html",
          settings: {
            chromeTheme: replayChromeTheme,
            backgroundTheme: replayBackgroundTheme,
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

    if (useNativeFilePicker && !supportsFileSystemAccess) {
      toast.error(
        "This browser does not support choosing a destination. Disable “Choose save location”.",
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

      const blob = new Blob([JSON.stringify(payload, null, 2)], {
        type: "application/json",
      });

      if (useNativeFilePicker && supportsFileSystemAccess) {
        try {
          const picker = await (
            window as typeof window & {
              showSaveFilePicker?: (
                options?: SaveFilePickerOptions,
              ) => Promise<FileSystemFileHandle>;
            }
          ).showSaveFilePicker?.({
            suggestedName: finalFileName,
            types: [
              {
                description: "Replay export (JSON)",
                accept: { "application/json": [".json"] },
              },
            ],
          });
          if (!picker) {
            throw new Error("Unable to open save dialog");
          }
          const writable = await picker.createWritable();
          await writable.write(blob);
          await writable.close();
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
        const url = URL.createObjectURL(blob);
        try {
          const anchor = document.createElement("a");
          anchor.href = url;
          anchor.download = finalFileName;
          document.body.appendChild(anchor);
          anchor.click();
          document.body.removeChild(anchor);
          toast.success("Replay download started");
        } finally {
          URL.revokeObjectURL(url);
        }
      }

      const exportName = exportFileStem || `${workflowName} Export`;
      const createdExport = await createExport({
        executionId: execution.id,
        workflowId: execution.workflowId,
        name: exportName,
        format: "json",
        settings: {
          chromeTheme: replayChromeTheme,
          backgroundTheme: replayBackgroundTheme,
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
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorClickAnimation,
    replayCursorInitialPosition,
    replayCursorScale,
    replayCursorTheme,
    replayFramesWithFallback.length,
    supportsFileSystemAccess,
    useNativeFilePicker,
    createExport,
  ]);

  const exportDialogProps = {
    exportFormat,
    setExportFormat,
    isBinaryExport,
    exportFormatOptions: EXPORT_FORMAT_OPTIONS,
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
    replayFramesLength: replayFramesWithFallback.length,
    exportStatusMessage,
    estimatedFrameCount,
    estimatedDurationSeconds,
    activeSpecId,
    previewMetrics,
    formatSeconds,
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

type SaveFilePickerOptions = {
  suggestedName?: string;
  types?: Array<{
    description?: string;
    accept: Record<string, string[]>;
  }>;
};

type FileSystemWritableFileStream = {
  write: (data: BlobPart) => Promise<void>;
  close: () => Promise<void>;
};

type FileSystemFileHandle = {
  createWritable: () => Promise<FileSystemWritableFileStream>;
};

export default useExecutionExport;
