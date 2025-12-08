// @ts-nocheck
import {
  useState,
  useEffect,
  useMemo,
  useCallback,
  useRef,
  useId,
  type MutableRefObject,
} from "react";
import { fromJson } from "@bufbuild/protobuf";
import {
  Activity,
  Pause,
  RotateCw,
  X,
  Square,
  Terminal,
  Image,
  Clock,
  CheckCircle,
  XCircle,
  Loader,
  PlayCircle,
  AlertTriangle,
  ChevronDown,
  Check,
  ListTree,
  Download,
  FolderOutput,
  Pencil,
} from "lucide-react";
import { format } from "date-fns";
import clsx from "clsx";
import type {
  ReplayFrame,
  ReplayChromeTheme,
  ReplayBackgroundTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
} from "./ReplayPlayer";
import { useExecutionStore } from "@stores/executionStore";
import type {
  Execution,
  TimelineFrame,
} from "@stores/executionStore";
import {
  type ExecutionExportPreview as ProtoExecutionExportPreview,
  ExecutionExportPreviewSchema,
} from "@vrooli/proto-types/browser-automation-studio/v1/execution_pb";
import { ExportStatus } from "@vrooli/proto-types/browser-automation-studio/v1/shared_pb";
import { useWorkflowStore } from "@stores/workflowStore";
import { useExportStore, type Export } from "@stores/exportStore";
import { ExportSuccessPanel } from "./ExportSuccessPanel";
import type { Screenshot, LogEntry } from "@stores/executionEventProcessor";
import { toast } from "react-hot-toast";
import { ResponsiveDialog } from "@shared/layout";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";
import type {
  ReplayMovieSpec,
  ReplayMovieFrameRect,
  ReplayMoviePresentation,
} from "@/types/export";
import { resolveUrl } from "@utils/executionTypeMappers";
import ExecutionHistory from "./ExecutionHistory";
import { selectors } from "@constants/selectors";
import { useExecutionEvents } from "./hooks/useExecutionEvents";
import { useExecutionActions } from "./hooks/useExecutionActions";
import {
  describePreviewStatusMessage,
  formatCapturedLabel,
  normalizePreviewStatus,
  stripApiSuffix,
  mapExportStatus,
  type ExportStatusLabel,
} from "./utils/exportHelpers";
import {
  DEFAULT_EXPORT_HEIGHT,
  DEFAULT_EXPORT_WIDTH,
  DIMENSION_PRESET_CONFIG,
  EXPORT_EXTENSIONS,
  EXPORT_FORMAT_OPTIONS,
  type ExportDimensionPreset,
  type ExportFormat,
  coerceMetricNumber,
  sanitizeFileStem,
} from "./viewer/exportConfig";
import { useReplayCustomization } from "./viewer/useReplayCustomization";
import ReplayCustomizationPanel from "./viewer/ReplayCustomizationPanel";
import ExportDialog from "./viewer/ExportDialog";
import ActiveExecutionTabs from "./viewer/ActiveExecutionTabs";
// Unsplash assets (IDs: m_7p45JfXQo, Tn29N3Hpf2E, KfFmwa7m5VQ) licensed for free use
const MOVIE_SPEC_POLL_INTERVAL_MS = 4000;

interface ActiveExecutionProps {
  execution: Execution;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

export type ViewerTab = "replay" | "screenshots" | "logs" | "executions";

const HEARTBEAT_WARN_SECONDS = 8;
const HEARTBEAT_STALL_SECONDS = 15;

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

export interface ExportPreviewMetrics {
  capturedFrames: number;
  assetCount: number;
  totalDurationMs: number;
}

type ExecutionExportPreview = {
  executionId: string;
  specId?: string;
  status: ExportStatusLabel;
  message?: string;
  capturedFrameCount: number;
  availableAssetCount: number;
  totalDurationMs: number;
  package?: ReplayMovieSpec;
};

const EXPORT_PREVIEW_PARSE_OPTIONS = {
  jsonOptions: { useProtoNames: true, ignoreUnknownFields: false },
} as const;

export const parseExportPreviewPayload = (
  raw: unknown,
): {
  preview: ProtoExecutionExportPreview;
  status: ExportStatusLabel;
  metrics: ExportPreviewMetrics;
  movieSpec: ReplayMovieSpec | null;
} => {
  const preview = fromJson(
    ExecutionExportPreviewSchema,
    raw as any,
    EXPORT_PREVIEW_PARSE_OPTIONS,
  );

  const status = mapExportStatus(preview.status);
  const metrics: ExportPreviewMetrics = {
    capturedFrames: typeof preview.capturedFrameCount === "number"
      ? preview.capturedFrameCount
      : 0,
    assetCount: typeof preview.availableAssetCount === "number"
      ? preview.availableAssetCount
      : 0,
    totalDurationMs: typeof preview.totalDurationMs === "number"
      ? preview.totalDurationMs
      : 0,
  };

  const movieSpec = preview.package
    ? (preview.package as unknown as ReplayMovieSpec)
    : null;

  return { preview, status, metrics, movieSpec };
};

function ActiveExecutionViewer({
  execution,
  onClose,
  showExecutionSwitcher = false,
}: ActiveExecutionProps) {
  useExecutionEvents({ id: execution.id, status: execution.status });
  const {
    refreshTimeline,
    stopExecution,
    startExecution,
    loadExecution,
    loadExecutions,
  } = useExecutionActions();
  const workflowName = useCurrentWorkflowName();
  const [activeTab, setActiveTab] = useState<ViewerTab>("replay");
  const [hasAutoSwitchedToReplay, setHasAutoSwitchedToReplay] =
    useState<boolean>(
      Boolean(execution.timeline && execution.timeline.length > 0),
    );
  const [selectedScreenshot, setSelectedScreenshot] =
    useState<Screenshot | null>(null);
  const [hasAutoOpenedScreenshots, setHasAutoOpenedScreenshots] =
    useState(false);
  const [, setHeartbeatTick] = useState(0);
  const [isStopping, setIsStopping] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isSwitchingExecution, setIsSwitchingExecution] = useState(false);
  const replayCustomization = useReplayCustomization({ executionId: execution.id });
  const {
    replayChromeTheme,
    replayBackgroundTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
    setReplayChromeTheme,
    setReplayBackgroundTheme,
    setReplayCursorTheme,
    setReplayCursorInitialPosition,
    setReplayCursorClickAnimation,
    setReplayCursorScale,
    selectedChromeOption,
    selectedBackgroundOption,
    selectedCursorOption,
    selectedCursorPositionOption,
    selectedCursorClickAnimationOption,
    backgroundOptionsByGroup,
    cursorOptionsByGroup,
    isCustomizationCollapsed,
    setIsCustomizationCollapsed,
    isBackgroundMenuOpen,
    setIsBackgroundMenuOpen,
    isCursorMenuOpen,
    setIsCursorMenuOpen,
    isCursorPositionMenuOpen,
    setIsCursorPositionMenuOpen,
    isCursorClickAnimationMenuOpen,
    setIsCursorClickAnimationMenuOpen,
    backgroundSelectorRef,
    cursorSelectorRef,
    cursorPositionSelectorRef,
    cursorClickAnimationSelectorRef,
    handleBackgroundSelect,
    handleCursorThemeSelect,
    handleCursorPositionSelect,
    handleCursorClickAnimationSelect,
    handleCursorScaleChange,
  } = replayCustomization;
  const screenshotRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const preloadedWorkflowRef = useRef<string | null>(null);
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
  const [previewMetrics, setPreviewMetrics] = useState<ExportPreviewMetrics>({
    capturedFrames: 0,
    assetCount: 0,
    totalDurationMs: 0,
  });
  const [activeSpecId, setActiveSpecId] = useState<string | null>(null);
  const [movieSpec, setMovieSpec] = useState<ReplayMovieSpec | null>(null);
  const [isMovieSpecLoading, setIsMovieSpecLoading] = useState(false);
  const [movieSpecError, setMovieSpecError] = useState<string | null>(null);
  const [composerApiBase, setComposerApiBase] = useState<string | null>(() => {
    if (typeof window === "undefined") {
      return null;
    }
    const existing = (
      window as typeof window & {
        __BAS_EXPORT_API_BASE__?: unknown;
      }
    ).__BAS_EXPORT_API_BASE__;
    return typeof existing === "string" && existing.trim().length > 0
      ? existing.trim()
      : null;
  });
  const composerRef = useRef<HTMLIFrameElement | null>(null);
  const composerWindowRef = useRef<Window | null>(null);
  const composerOriginRef = useRef<string | null>(null);
  const previewComposerRef = useRef<HTMLIFrameElement | null>(null);
  const composerPreviewWindowRef = useRef<Window | null>(null);
  const composerPreviewOriginRef = useRef<string | null>(null);
  const [isComposerReady, setIsComposerReady] = useState(false);
  const [isPreviewComposerReady, setIsPreviewComposerReady] = useState(false);
  const [previewComposerError, setPreviewComposerError] = useState<
    string | null
  >(null);
  const composerFrameStateRef = useRef({ frameIndex: 0, progress: 0 });
  const movieSpecRetryTimeoutRef = useRef<number | null>(null);
  const movieSpecAbortControllerRef = useRef<AbortController | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [showExportSuccess, setShowExportSuccess] = useState(false);
  const [lastCreatedExport, setLastCreatedExport] = useState<Export | null>(null);
  const { createExport } = useExportStore();
  const supportsFileSystemAccess =
    typeof window !== "undefined" &&
    typeof (window as typeof window & { showSaveFilePicker?: unknown })
      .showSaveFilePicker === "function";

  useEffect(() => {
    if (composerApiBase) {
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const configData = await getConfig();
        if (cancelled) {
          return;
        }
        const derived = stripApiSuffix(configData.API_URL);
        if (derived) {
          setComposerApiBase((current) =>
            current && current.length > 0 ? current : derived,
          );
        }
      } catch (error) {
        if (!cancelled) {
          logger.warn(
            "Failed to derive API base for replay composer",
            { component: "ExecutionViewer", executionId: execution.id },
            error,
          );
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [composerApiBase, execution.id]);

  useEffect(() => {
    if (typeof window === "undefined" || !composerApiBase) {
      return;
    }
    (
      window as typeof window & { __BAS_EXPORT_API_BASE__?: string }
    ).__BAS_EXPORT_API_BASE__ = composerApiBase;
  }, [composerApiBase]);

  const specCanvasWidth =
    movieSpec?.presentation?.canvas?.width ??
    movieSpec?.presentation?.viewport?.width ??
    DEFAULT_EXPORT_WIDTH;
  const specCanvasHeight =
    movieSpec?.presentation?.canvas?.height ??
    movieSpec?.presentation?.viewport?.height ??
    DEFAULT_EXPORT_HEIGHT;

  const defaultExportFileStem = useMemo(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
    [execution.id],
  );

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
    if (typeof window === "undefined") {
      composerOriginRef.current = null;
      return;
    }
    try {
      composerOriginRef.current = new URL(
        composerUrl,
        window.location.href,
      ).origin;
    } catch (error) {
      composerOriginRef.current = null;
      logger.warn(
        "Failed to derive composer origin",
        { component: "ExecutionViewer", executionId: execution.id },
        error,
      );
    }
  }, [composerUrl, execution.id]);

  useEffect(() => {
    if (typeof window === "undefined") {
      composerPreviewOriginRef.current = null;
      return;
    }
    try {
      composerPreviewOriginRef.current = new URL(
        composerPreviewUrl,
        window.location.href,
      ).origin;
    } catch (error) {
      composerPreviewOriginRef.current = null;
      logger.warn(
        "Failed to derive preview composer origin",
        { component: "ExecutionViewer", executionId: execution.id },
        error,
      );
    }
  }, [composerPreviewUrl, execution.id]);

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

  const heartbeatTimestamp = execution.lastHeartbeat?.timestamp?.valueOf();
  const executionError = execution.error ?? undefined;

  useEffect(() => {
    if (execution.status !== "running" || !heartbeatTimestamp) {
      return;
    }
    const interval = window.setInterval(() => {
      setHeartbeatTick((tick) => tick + 1);
    }, 1000);
    return () => {
      window.clearInterval(interval);
    };
  }, [execution.status, heartbeatTimestamp]);

  useEffect(() => {
    setHasAutoSwitchedToReplay(
      Boolean(execution.timeline && execution.timeline.length > 0),
    );
    setActiveTab("replay");
    setSelectedScreenshot(null);
  }, [execution.id]);

  useEffect(() => {
    setIsExportDialogOpen(false);
    setExportPreview(null);
    setExportPreviewExecutionId(null);
    setExportPreviewError(null);
    setExportFileStem(defaultExportFileStem);
    setUseNativeFilePicker(false);
  }, [defaultExportFileStem]);

  useEffect(() => {
    setCustomWidthInput(String(specCanvasWidth));
    setCustomHeightInput(String(specCanvasHeight));
    setDimensionPreset("spec");
  }, [execution.id, specCanvasWidth, specCanvasHeight]);

  const clearMovieSpecRetryTimeout = useCallback(() => {
    if (movieSpecRetryTimeoutRef.current != null) {
      window.clearTimeout(movieSpecRetryTimeoutRef.current);
      movieSpecRetryTimeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [clearMovieSpecRetryTimeout]);

  useEffect(() => {
    let isCancelled = false;

    if (movieSpecAbortControllerRef.current) {
      movieSpecAbortControllerRef.current.abort();
      movieSpecAbortControllerRef.current = null;
    }
    clearMovieSpecRetryTimeout();

    async function fetchMovieSpec() {
      if (isCancelled) {
        return;
      }

      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
      }
      const abortController = new AbortController();
      movieSpecAbortControllerRef.current = abortController;

      let shouldRetry = false;
      try {
        setIsMovieSpecLoading(true);
        const configData = await getConfig();
        const response = await fetch(
          `${configData.API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: JSON.stringify({ format: "json" }),
            signal: abortController.signal,
          },
        );
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }
        const raw = await response.json();
        if (isCancelled) {
          return;
        }
        const {
          preview,
          status,
          metrics,
          movieSpec,
        } = parseExportPreviewPayload(raw);
        const specId =
          (preview.specId && preview.specId.trim()) ||
          preview.executionId ||
          execution.id;

        setPreviewMetrics(metrics);
        setActiveSpecId(specId);

        const message = describePreviewStatusMessage(
          status,
          preview.message,
          metrics,
        );
        if (status !== "ready") {
          setMovieSpec(null);
          setMovieSpecError(message);
          shouldRetry = status === "pending";
          return;
        }
        if (!movieSpec) {
          setMovieSpec(null);
          setMovieSpecError(
            "Replay export unavailable – missing export package",
          );
          return;
        }
        clearMovieSpecRetryTimeout();
        setMovieSpec(movieSpec);
        setMovieSpecError(null);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }
        const message =
          error instanceof Error ? error.message : "Failed to load replay spec";
        setMovieSpecError(message);
        setMovieSpec(null);
        logger.error(
          "Failed to load replay movie spec",
          { component: "ExecutionViewer", executionId: execution.id },
          error,
        );
        shouldRetry = execution.status === "running";
      } finally {
        if (!isCancelled) {
          setIsMovieSpecLoading(false);
        }
        if (!isCancelled) {
          movieSpecAbortControllerRef.current = null;
        }
        if (!isCancelled && shouldRetry) {
          clearMovieSpecRetryTimeout();
          movieSpecRetryTimeoutRef.current = window.setTimeout(() => {
            void fetchMovieSpec();
          }, MOVIE_SPEC_POLL_INTERVAL_MS);
        }
      }
    }

    setMovieSpecError(null);
    setMovieSpec(null);
    setIsMovieSpecLoading(true);
    void fetchMovieSpec();

    return () => {
      isCancelled = true;
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [
    execution.id,
    execution.status,
    execution.timeline?.length,
    clearMovieSpecRetryTimeout,
  ]);

  useEffect(() => {
    if (exportFormat !== "json" && useNativeFilePicker) {
      setUseNativeFilePicker(false);
    }
  }, [exportFormat, useNativeFilePicker]);

  useEffect(() => {
    if (!showExecutionSwitcher || !execution.workflowId) {
      return;
    }
    if (preloadedWorkflowRef.current === execution.workflowId) {
      return;
    }
    preloadedWorkflowRef.current = execution.workflowId;
    void loadExecutions(execution.workflowId);
  }, [execution.workflowId, loadExecutions, showExecutionSwitcher]);

  useEffect(() => {
    if (!showExecutionSwitcher && activeTab === "executions") {
      setActiveTab("replay");
    }
  }, [showExecutionSwitcher, activeTab]);

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
        const configData = await getConfig();
        const response = await fetch(
          `${configData.API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ format: "json" }),
          },
        );
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }
        const raw = await response.json();
        if (isCancelled) {
          return;
        }
        const {
          preview,
          status,
          metrics,
          movieSpec,
        } = parseExportPreviewPayload(raw);

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

  useEffect(() => {
    if (
      !hasAutoSwitchedToReplay &&
      execution.timeline &&
      execution.timeline.length > 0
    ) {
      setActiveTab("replay");
      setHasAutoSwitchedToReplay(true);
    }
  }, [execution.timeline, hasAutoSwitchedToReplay]);

  const derivedHeartbeatTimestamp = useMemo(() => {
    if (execution.lastHeartbeat?.timestamp) {
      return execution.lastHeartbeat.timestamp;
    }
    const frames = execution.timeline ?? [];
    for (let idx = frames.length - 1; idx >= 0; idx -= 1) {
      const frame = frames[idx];
      const rawTimestamp =
        frame.completed_at ||
        frame.completedAt ||
        frame.started_at ||
        frame.startedAt;
      if (rawTimestamp) {
        const parsed = new Date(rawTimestamp);
        if (!Number.isNaN(parsed.getTime())) {
          return parsed;
        }
      }
    }
    return undefined;
  }, [execution.lastHeartbeat, execution.timeline]);

  const heartbeatAgeSeconds = useMemo(() => {
    if (!derivedHeartbeatTimestamp) {
      return null;
    }
    const age = (Date.now() - derivedHeartbeatTimestamp.getTime()) / 1000;
    return age < 0 ? 0 : age;
  }, [derivedHeartbeatTimestamp]);

  const inStepSeconds =
    execution.lastHeartbeat?.elapsedMs != null
      ? Math.max(0, execution.lastHeartbeat.elapsedMs / 1000)
      : null;

  const formatSeconds = (value: number) => {
    if (Number.isNaN(value) || !Number.isFinite(value)) {
      return "0s";
    }
    if (value >= 10) {
      return `${Math.round(value)}s`;
    }
    return `${value.toFixed(1)}s`;
  };

  const heartbeatAgeLabel =
    heartbeatAgeSeconds == null
      ? null
      : heartbeatAgeSeconds < 0.75
        ? "just now"
        : `${formatSeconds(heartbeatAgeSeconds)} ago`;

  const inStepLabel =
    inStepSeconds != null ? formatSeconds(inStepSeconds) : null;

  type HeartbeatState = "idle" | "awaiting" | "healthy" | "delayed" | "stalled";

  const heartbeatState: HeartbeatState = useMemo(() => {
    const isRunning = execution.status === "running";
    const hasHeartbeat = Boolean(
      execution.lastHeartbeat || derivedHeartbeatTimestamp,
    );
    if (!hasHeartbeat) {
      return isRunning ? "awaiting" : "idle";
    }
    if (heartbeatAgeSeconds == null) {
      return "awaiting";
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_STALL_SECONDS) {
      return "stalled";
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_WARN_SECONDS) {
      return "delayed";
    }
    return "healthy";
  }, [
    execution.status,
    execution.lastHeartbeat,
    heartbeatAgeSeconds,
    derivedHeartbeatTimestamp,
  ]);

  const heartbeatDescriptor = useMemo(() => {
    const baseDescriptor = (() => {
      switch (heartbeatState) {
        case "idle":
          if (execution.status === "running") {
            return null;
          }
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "No heartbeats recorded for this run",
          };
        case "awaiting":
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "Awaiting first heartbeat…",
          };
        case "healthy":
          return {
            tone: "healthy" as const,
            iconClass: "text-blue-400",
            textClass: "text-blue-200",
            label: `Heartbeat ${heartbeatAgeLabel ?? "just now"}`,
          };
        case "delayed":
          return {
            tone: "delayed" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200",
            label: `Heartbeat delayed (${formatSeconds(heartbeatAgeSeconds ?? 0)} since last update)`,
          };
        case "stalled":
          return {
            tone: "stalled" as const,
            iconClass: "text-red-400",
            textClass: "text-red-200",
            label: `Heartbeat stalled (${formatSeconds(heartbeatAgeSeconds ?? 0)} without update)`,
          };
        default:
          return null;
      }
    })();

    if (!baseDescriptor) {
      return null;
    }

    const labelWithSource = execution.lastHeartbeat
      ? baseDescriptor.label
      : `${baseDescriptor.label} (timeline activity)`;

    if (execution.status !== "running") {
      return {
        ...baseDescriptor,
        label: `${labelWithSource} • Final heartbeat snapshot`,
      };
    }

    return {
      ...baseDescriptor,
      label: labelWithSource,
    };
  }, [
    heartbeatState,
    heartbeatAgeLabel,
    heartbeatAgeSeconds,
    execution.status,
    execution.lastHeartbeat,
  ]);

  const statusMessage = useMemo(() => {
    const label =
      typeof execution.currentStep === "string"
        ? execution.currentStep.trim()
        : "";
    if (label.length > 0) {
      return label;
    }
    switch (execution.status) {
      case "pending":
        return "Pending...";
      case "running":
        return "Running...";
      case "completed":
        return "Completed";
      case "failed":
        return execution.error ? "Failed" : "Failed";
      case "cancelled":
        return "Cancelled";
      default:
        return "Initializing...";
    }
  }, [execution.currentStep, execution.status, execution.error]);

  const isRunning = execution.status === "running";
  const isFailed = execution.status === "failed";
  const isCancelled = execution.status === "cancelled";
  const canRestart =
    Boolean(execution.workflowId) && execution.status !== "running";

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

  const replayFrames = useMemo<ReplayFrame[]>(() => {
    return timelineForReplay.map((frame: TimelineFrame, index: number) => {
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
    });
  }, [timelineForReplay]);

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

  useEffect(() => {
    if (!activeMovieSpec) {
      return;
    }
    setPreviewMetrics((current) => {
      const frameCount =
        activeMovieSpec.summary?.frame_count ??
        activeMovieSpec.frames?.length ??
        current.capturedFrames;
      const assetCount =
        Array.isArray(activeMovieSpec.assets) &&
        activeMovieSpec.assets.length >= 0
          ? activeMovieSpec.assets.length
          : current.assetCount;
      const totalDuration =
        activeMovieSpec.summary?.total_duration_ms ??
        activeMovieSpec.playback?.duration_ms ??
        current.totalDurationMs;
      return {
        capturedFrames: frameCount,
        assetCount,
        totalDurationMs: totalDuration,
      };
    });
    const specIdFromSpec = activeMovieSpec.execution?.execution_id;
    if (specIdFromSpec && specIdFromSpec !== activeSpecId) {
      setActiveSpecId(specIdFromSpec);
    }
  }, [activeMovieSpec, activeSpecId]);

  const firstFramePreviewUrl = useMemo(() => {
    const frame = replayFrames[0];
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
  }, [activeMovieSpec, replayFrames]);

  const firstFrameLabel = useMemo(() => {
    const frame = replayFrames[0];
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
  }, [replayFrames]);

  const hasTimeline = replayFrames.length > 0;

  const exportDialogTitleId = useId();
  const exportDialogDescriptionId = useId();
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
    replayFrames.length;
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

  const timelineScreenshots = useMemo(() => {
    const frames = execution.timeline ?? [];
    const items: Screenshot[] = [];
    frames.forEach((frame: TimelineFrame, index: number) => {
      const resolved = resolveUrl(frame?.screenshot?.url);
      if (!resolved) {
        return;
      }
      items.push({
        id: frame?.screenshot?.artifactId || `timeline-${index}`,
        url: resolved,
        stepName:
          frame?.nodeId ||
          (typeof frame?.stepType === "string" ? frame.stepType : undefined) ||
          (typeof frame?.stepIndex === "number"
            ? `Step ${frame.stepIndex + 1}`
            : "Step"),
        timestamp: new Date(),
      });
    });
    return items;
  }, [execution.timeline]);

  const postToComposer = useCallback(
    (message: Record<string, unknown>) => {
      const targets: Array<{
        windowRef: MutableRefObject<Window | null>;
        originRef: MutableRefObject<string | null>;
        url: string;
        label: string;
      }> = [
        {
          windowRef: composerWindowRef,
          originRef: composerOriginRef,
          url: composerUrl,
          label: "composer",
        },
      ];
      if (composerPreviewWindowRef.current) {
        targets.push({
          windowRef: composerPreviewWindowRef,
          originRef: composerPreviewOriginRef,
          url: composerPreviewUrl,
          label: "preview-composer",
        });
      }

      targets.forEach(({ windowRef, originRef, url, label }) => {
        const targetWindow = windowRef.current;
        if (!targetWindow) {
          return;
        }
        let targetOrigin = originRef.current;
        if (!targetOrigin && typeof window !== "undefined") {
          try {
            targetOrigin = new URL(url, window.location.href).origin;
            originRef.current = targetOrigin;
          } catch (error) {
            logger.warn(
              "Failed to resolve composer origin",
              {
                component: "ExecutionViewer",
                executionId: execution.id,
                context: label,
              },
              error,
            );
          }
        }
        try {
          targetWindow.postMessage(message, targetOrigin ?? "*");
        } catch (error) {
          logger.warn(
            "Failed to post message to composer",
            { component: "ExecutionViewer", context: label },
            error,
          );
        }
      });
    },
    [composerUrl, composerPreviewUrl, execution.id],
  );

  const sendSpecToComposer = useCallback(
    (spec: ReplayMovieSpec | null) => {
      if (!spec) {
        return;
      }
      const specId =
        spec.execution?.execution_id ?? activeSpecId ?? execution.id;
      const summaryFrames =
        spec.summary?.frame_count ?? spec.frames?.length ?? 0;
      const summaryDuration =
        spec.summary?.total_duration_ms ?? spec.playback?.duration_ms ?? 0;
      const summaryAssets = Array.isArray(spec.assets) ? spec.assets.length : 0;
      postToComposer({
        type: "bas:spec:set",
        spec,
        executionId: execution.id,
        specId,
        apiBase: composerApiBase ?? undefined,
        summary: {
          frames: summaryFrames,
          totalDurationMs: summaryDuration,
          assets: summaryAssets,
        },
      });
    },
    [activeSpecId, composerApiBase, execution.id, postToComposer],
  );

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleMessage = (event: MessageEvent) => {
      const fromComposer = event.source === composerWindowRef.current;
      const fromPreview = event.source === composerPreviewWindowRef.current;
      if (!fromComposer && !fromPreview) {
        return;
      }
      const payload = event.data;
      if (!payload || typeof payload !== "object") {
        return;
      }
      const { type } = payload as { type?: unknown };
      if (typeof type !== "string" || !type.startsWith("bas:")) {
        return;
      }
      if (type === "bas:ready") {
        if (fromComposer) {
          setIsComposerReady(true);
          setMovieSpecError(null);
          composerOriginRef.current = event.origin || composerOriginRef.current;
        } else {
          setIsPreviewComposerReady(true);
          setPreviewComposerError(null);
          composerPreviewOriginRef.current =
            event.origin || composerPreviewOriginRef.current;
        }
        if (preparedMovieSpec) {
          sendSpecToComposer(preparedMovieSpec);
        }
      } else if (type === "bas:state") {
        if (fromComposer) {
          const data = payload as { frameIndex?: unknown; progress?: unknown };
          const frameIndex =
            typeof data.frameIndex === "number" ? data.frameIndex : 0;
          const progress =
            typeof data.progress === "number" ? data.progress : 0;
          composerFrameStateRef.current = { frameIndex, progress };
        }
      } else if (type === "bas:error") {
        const data = payload as {
          message?: unknown;
          status?: unknown;
          frames?: unknown;
          assets?: unknown;
          totalDurationMs?: unknown;
          specId?: unknown;
        };
        const status = normalizePreviewStatus(
          typeof data.status === "string" ? data.status : undefined,
        );
        const message = describePreviewStatusMessage(
          status,
          typeof data.message === "string" ? data.message : undefined,
          {
            capturedFrames:
              typeof data.frames !== "undefined"
                ? coerceMetricNumber(data.frames)
                : previewMetrics.capturedFrames,
            assetCount:
              typeof data.assets !== "undefined"
                ? coerceMetricNumber(data.assets)
                : previewMetrics.assetCount,
          },
        );
        if (fromComposer) {
          setIsComposerReady(false);
          setMovieSpecError(message);
        } else {
          setIsPreviewComposerReady(false);
          setPreviewComposerError(message);
        }
        const framesValue =
          typeof data.frames !== "undefined"
            ? coerceMetricNumber(data.frames)
            : null;
        const assetsValue =
          typeof data.assets !== "undefined"
            ? coerceMetricNumber(data.assets)
            : null;
        const durationValue =
          typeof data.totalDurationMs !== "undefined"
            ? coerceMetricNumber(data.totalDurationMs)
            : null;
        if (
          framesValue !== null ||
          assetsValue !== null ||
          durationValue !== null
        ) {
          setPreviewMetrics((current) => {
            const next = { ...current };
            let changed = false;
            if (framesValue !== null) {
              next.capturedFrames = framesValue;
              changed = true;
            }
            if (assetsValue !== null) {
              next.assetCount = assetsValue;
              changed = true;
            }
            if (durationValue !== null) {
              next.totalDurationMs = durationValue;
              changed = true;
            }
            return changed ? next : current;
          });
        }
        if (typeof data.specId === "string" && data.specId.trim()) {
          setActiveSpecId(data.specId.trim());
        }
      } else if (type === "bas:error-clear") {
        if (fromComposer) {
          setMovieSpecError(null);
        } else {
          setPreviewComposerError(null);
        }
      } else if (type === "bas:metrics") {
        if (!fromComposer) {
          return;
        }
        const data = payload as {
          frames?: unknown;
          assets?: unknown;
          totalDurationMs?: unknown;
          specId?: unknown;
        };
        const framesValue =
          typeof data.frames !== "undefined"
            ? coerceMetricNumber(data.frames)
            : null;
        const assetsValue =
          typeof data.assets !== "undefined"
            ? coerceMetricNumber(data.assets)
            : null;
        const durationValue =
          typeof data.totalDurationMs !== "undefined"
            ? coerceMetricNumber(data.totalDurationMs)
            : null;
        if (
          framesValue !== null ||
          assetsValue !== null ||
          durationValue !== null
        ) {
          setPreviewMetrics((current) => {
            const next = { ...current };
            let changed = false;
            if (framesValue !== null) {
              next.capturedFrames = framesValue;
              changed = true;
            }
            if (assetsValue !== null) {
              next.assetCount = assetsValue;
              changed = true;
            }
            if (durationValue !== null) {
              next.totalDurationMs = durationValue;
              changed = true;
            }
            return changed ? next : current;
          });
        }
        if (typeof data.specId === "string" && data.specId.trim()) {
          setActiveSpecId(data.specId.trim());
        }
      }
    };
    window.addEventListener("message", handleMessage);
    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [preparedMovieSpec, previewMetrics, sendSpecToComposer]);

  useEffect(() => {
    setIsComposerReady(false);
    composerFrameStateRef.current = { frameIndex: 0, progress: 0 };
    setHasAutoOpenedScreenshots(false);
  }, [execution.id]);

  useEffect(() => {
    if (!isComposerReady || !preparedMovieSpec) {
      return;
    }
    sendSpecToComposer(preparedMovieSpec);
  }, [isComposerReady, preparedMovieSpec, sendSpecToComposer]);

  const screenshots =
    timelineScreenshots.length > 0
      ? timelineScreenshots
      : execution.screenshots;

  useEffect(() => {
    if (screenshots.length === 0) {
      return;
    }
    const alreadySelected =
      selectedScreenshot &&
      screenshots.some((shot) => shot.id === selectedScreenshot.id);
    if (!alreadySelected) {
      setSelectedScreenshot(screenshots[screenshots.length - 1]);
    }
    if (
      screenshots.length > 0 &&
      !hasAutoOpenedScreenshots &&
      activeTab !== "screenshots"
    ) {
      setActiveTab("screenshots");
      setHasAutoOpenedScreenshots(true);
    }
  }, [screenshots, selectedScreenshot, hasAutoOpenedScreenshots, activeTab]);

  useEffect(() => {
    if (!selectedScreenshot) {
      return;
    }
    const element = screenshotRefs.current[selectedScreenshot.id];
    if (element) {
      element.scrollIntoView({
        behavior: "smooth",
        block: "start",
        inline: "nearest",
      });
    }
  }, [selectedScreenshot]);

  useEffect(() => {
    if (
      activeTab === "screenshots" &&
      screenshots.length === 0 &&
      replayFrames.length > 0
    ) {
      setActiveTab("replay");
    }
    if (activeTab === "screenshots") {
      setHasAutoOpenedScreenshots(true);
    }
  }, [activeTab, screenshots.length, replayFrames.length]);

  useEffect(() => {
    let interval: number | undefined;
    if (execution.status === "running") {
      interval = window.setInterval(() => {
        void refreshTimeline(execution.id);
      }, 2000);
    } else if (
      execution.status === "completed" ||
      execution.status === "failed" ||
      execution.status === "cancelled"
    ) {
      void refreshTimeline(execution.id);
    }

    return () => {
      if (interval) {
        window.clearInterval(interval);
      }
    };
  }, [execution.status, execution.id, refreshTimeline]);

  const handleStop = useCallback(async () => {
    if (!isRunning || isStopping) {
      return;
    }
    setIsStopping(true);
    try {
      await stopExecution(execution.id);
      toast.success("Execution stopped");
      await refreshTimeline(execution.id);
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to stop execution";
      toast.error(message);
    } finally {
      setIsStopping(false);
    }
  }, [execution.id, isRunning, isStopping, stopExecution, refreshTimeline]);

  const handleRestart = useCallback(async () => {
    if (!canRestart || isRestarting) {
      return;
    }
    if (!execution.workflowId) {
      toast.error("Workflow identifier missing for restart");
      return;
    }
    setIsRestarting(true);
    try {
      await startExecution(execution.workflowId);
      toast.success("Workflow restarted");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to restart workflow";
      toast.error(message);
    } finally {
      setIsRestarting(false);
    }
  }, [canRestart, execution.workflowId, isRestarting, startExecution]);

  const handleExecutionSwitch = useCallback(
    async (candidate: { id: string }) => {
      if (!candidate?.id) {
        return;
      }
      if (candidate.id === execution.id) {
        setActiveTab("replay");
        return;
      }
      setIsSwitchingExecution(true);
      try {
        await loadExecution(candidate.id);
        setActiveTab("replay");
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to load execution";
        toast.error(message);
      } finally {
        setIsSwitchingExecution(false);
      }
    },
    [execution.id, loadExecution],
  );

  const handleOpenExportDialog = useCallback(() => {
    if (replayFrames.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }
    setExportFormat("mp4");
    setExportFileStem(defaultExportFileStem);
    setUseNativeFilePicker(false);
    setIsExportDialogOpen(true);
  }, [defaultExportFileStem, replayFrames.length]);

  const handleCloseExportDialog = useCallback(() => {
    setIsExportDialogOpen(false);
  }, []);

  const handleConfirmExport = useCallback(async () => {
    if (exportFormat !== "json") {
      if (replayFrames.length === 0) {
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

        // Create export record in the library
        const exportName = exportFileStem || `${workflowName} Export`;
        const createdExport = await createExport({
          executionId: execution.id,
          workflowId: execution.workflowId,
          name: exportName,
          format: exportFormat as 'mp4' | 'gif' | 'json' | 'html',
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

    if (replayFrames.length === 0) {
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

      // Create export record in the library for JSON exports
      const exportName = exportFileStem || `${workflowName} Export`;
      const createdExport = await createExport({
        executionId: execution.id,
        workflowId: execution.workflowId,
        name: exportName,
        format: 'json',
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
    replayFrames.length,
    supportsFileSystemAccess,
    useNativeFilePicker,
    createExport,
  ]);

  const getStatusIcon = () => {
    const statusTestId = `execution-status-${execution.status}`;
    // Add both general and specific status selectors for test flexibility
    const testIds = `${selectors.executions.viewer.status} ${statusTestId}`;
    switch (execution.status) {
      case "running":
        return (
          <Loader
            size={16}
            className="animate-spin text-blue-400"
            data-testid={testIds}
          />
        );
      case "completed":
        return (
          <CheckCircle
            size={16}
            className="text-green-400"
            data-testid={testIds}
          />
        );
      case "failed":
        return (
          <XCircle
            size={16}
            className="text-red-400"
            data-testid={testIds}
          />
        );
      case "cancelled":
        return (
          <AlertTriangle
            size={16}
            className="text-yellow-400"
            data-testid={testIds}
          />
        );
      default:
        return (
          <Clock
            size={16}
            className="text-gray-400"
            data-testid={testIds}
          />
        );
    }
  };

  const getLogColor = (level: LogEntry["level"]) => {
    switch (level) {
      case "error":
        return "text-red-400";
      case "warning":
        return "text-yellow-400";
      case "success":
        return "text-green-400";
      default:
        return "text-gray-300";
    }
  };

  return (
    <div
      className="h-full flex flex-col bg-flow-node min-h-0"
      data-testid={selectors.executions.viewer.root}
    >
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          {getStatusIcon()}
          <div>
            <div className="text-sm font-medium text-white">
              Execution #{execution.id.slice(0, 8)}
            </div>
            <div
              className="text-xs text-gray-500"
              data-testid={selectors.executions.viewer.status}
            >
              {statusMessage}
            </div>
            {heartbeatDescriptor && (
              <div
                className="mt-1 flex items-center gap-2 text-[11px]"
                data-testid={selectors.heartbeat.indicator}
              >
                {heartbeatDescriptor.tone === "stalled" ? (
                  <AlertTriangle
                    size={12}
                    className={heartbeatDescriptor.iconClass}
                    data-testid={
                      heartbeatDescriptor.tone === "stalled"
                        ? selectors.heartbeat.lagWarning
                        : undefined
                    }
                  />
                ) : (
                  <Activity
                    size={12}
                    className={heartbeatDescriptor.iconClass}
                  />
                )}
                <span
                  className={heartbeatDescriptor.textClass}
                  data-testid={selectors.heartbeat.status}
                >
                  {heartbeatDescriptor.label}
                </span>
                {inStepLabel && execution.lastHeartbeat && (
                  <span
                    className={`${heartbeatDescriptor.textClass} opacity-80`}
                  >
                    • {inStepLabel} in step
                  </span>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            className="toolbar-button p-1.5 text-gray-500 opacity-50 cursor-not-allowed"
            title="Pause (coming soon)"
            disabled
            aria-disabled="true"
          >
            <Pause size={14} />
          </button>
          <button
            className="toolbar-button p-1.5"
            title={
              canRestart
                ? "Re-run workflow"
                : "Stop execution before re-running"
            }
            onClick={handleRestart}
            disabled={!canRestart || isRestarting || isStopping}
            data-testid={selectors.executions.viewer.rerunButton}
          >
            {isRestarting ? (
              <Loader size={14} className="animate-spin" />
            ) : (
              <RotateCw size={14} />
            )}
          </button>
          <button
            className="toolbar-button p-1.5 disabled:text-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
            title={
              replayFrames.length === 0
                ? "Replay not ready to export"
                : "Export replay"
            }
            onClick={handleOpenExportDialog}
            disabled={replayFrames.length === 0}
            data-testid={selectors.executions.actions.exportReplayButton}
          >
            <Download size={14} />
          </button>
          <button
            className="toolbar-button p-1.5 text-red-400 disabled:text-red-400/50 disabled:cursor-not-allowed"
            title={isRunning ? "Stop execution" : "Execution not running"}
            onClick={handleStop}
            disabled={!isRunning || isStopping}
            data-testid={selectors.executions.viewer.stopButton}
          >
            {isStopping ? (
              <Loader size={14} className="animate-spin" />
            ) : (
              <Square size={14} />
            )}
          </button>
          {onClose && (
            <>
              <button
                className="toolbar-button p-1.5 ml-2 border-l border-gray-700 pl-3 text-blue-400 hover:text-blue-300"
                title="Edit workflow"
                onClick={onClose}
                data-testid={selectors.executions.viewer.editWorkflowButton}
              >
                <Pencil size={14} />
              </button>
              <button
                className="toolbar-button p-1.5"
                title="Close"
                onClick={onClose}
              >
                <X size={14} />
              </button>
            </>
          )}
        </div>
      </div>

      <div className="h-2 bg-flow-bg">
        <div
          className="h-full bg-flow-accent transition-all duration-300"
          style={{ width: `${execution.progress}%` }}
        />
      </div>

      <ActiveExecutionTabs
        activeTab={activeTab}
        onTabChange={setActiveTab}
        showExecutionSwitcher={showExecutionSwitcher}
        isSwitchingExecution={isSwitchingExecution}
        hasTimeline={hasTimeline}
        counts={{
          replay: replayFrames.length,
          screenshots: screenshots.length,
          logs: execution.logs.length,
        }}
        tabs={{
          replay: (
            <div
              className="flex-1 overflow-auto p-3 space-y-3"
              data-testid={selectors.replay.player}
            >
              {!hasTimeline && (
                <div className="rounded-lg border border-dashed border-slate-700/60 bg-slate-900/60 px-4 py-3 text-sm text-slate-200/80">
                  Replay frames stream in as each action runs. Leave this tab open
                  to tailor the final cut in real time.
                </div>
              )}
              {(isFailed || isCancelled) && execution.progress < 100 && (
                <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 flex items-start gap-3">
                  <AlertTriangle
                    size={18}
                    className="text-rose-400 flex-shrink-0 mt-0.5"
                  />
                  <div className="flex-1 text-sm">
                    <div className="font-medium text-rose-200 mb-1">
                      Execution {isFailed ? "Failed" : "Cancelled"} - Replay
                      Incomplete
                    </div>
                    <div className="text-rose-100/80">
                      This replay shows only {replayFrames.length} of the
                      workflow's steps. Execution{" "}
                      {isFailed ? "failed" : "was cancelled"}
                      at {execution.currentStep || "an unknown step"}.
                    </div>
                    {isFailed && executionError && (
                      <div className="mt-2 text-xs font-mono text-rose-100/70 bg-rose-950/30 px-2 py-1 rounded">
                        {executionError}
                      </div>
                    )}
                  </div>
                </div>
              )}
              <ReplayCustomizationPanel controller={replayCustomization} />
              <div className="relative w-full overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-[0_25px_70px_rgba(15,23,42,0.45)]">
                <iframe
                  key={execution.id}
                  ref={(node) => {
                    composerRef.current = node;
                    composerWindowRef.current = node?.contentWindow ?? null;
                  }}
                  src={composerUrl}
                  title="Replay Composer"
                  className="w-full border-0"
                  style={{
                    aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                    minHeight: "360px",
                  }}
                  allow="clipboard-read; clipboard-write"
                />
                {(isMovieSpecLoading || !isComposerReady) && !movieSpecError && (
                  <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-slate-950/55">
                    <span className="text-xs uppercase tracking-[0.3em] text-slate-400">
                      {isMovieSpecLoading
                        ? "Loading replay spec…"
                        : "Initialising player…"}
                    </span>
                  </div>
                )}
                {movieSpecError && (
                  <div className="absolute inset-0 flex items-center justify-center bg-slate-950/65 p-6 text-center">
                    <div className="rounded-xl border border-slate-800 bg-slate-900/70 px-6 py-4 text-sm text-slate-200 shadow-[0_15px_45px_rgba(15,23,42,0.5)]">
                      <div className="mb-1 font-semibold text-slate-100">
                        Failed to load replay spec
                      </div>
                      <div className="text-xs text-slate-300/80">
                        {movieSpecError}
                      </div>
                      {(previewMetrics.capturedFrames > 0 ||
                        previewMetrics.totalDurationMs > 0) && (
                        <div className="mt-3 text-[11px] text-slate-400">
                          {previewMetrics.capturedFrames > 0 && (
                            <span>
                              {formatCapturedLabel(
                                previewMetrics.capturedFrames,
                                "frame",
                              )}
                            </span>
                          )}
                          {previewMetrics.capturedFrames > 0 &&
                            previewMetrics.totalDurationMs > 0 && (
                              <span> • </span>
                            )}
                          {previewMetrics.totalDurationMs > 0 && (
                            <span>
                              {formatSeconds(
                                previewMetrics.totalDurationMs / 1000,
                              )}{" "}
                              recorded
                            </span>
                          )}
                        </div>
                      )}
                      {activeSpecId && (
                        <div className="mt-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                          Spec {activeSpecId.slice(0, 8)}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ),
          screenshots:
            screenshots.length === 0 ? (
              <div className="flex flex-1 items-center justify-center p-6 text-center">
                <div>
                  <Image size={32} className="mx-auto mb-3 text-gray-600" />
                  <div className="text-sm text-gray-400 mb-1">
                    No screenshots captured
                  </div>
                  {isFailed && (
                    <div className="text-xs text-gray-500">
                      Execution failed before screenshot steps could run
                    </div>
                  )}
                  {isCancelled && (
                    <div className="text-xs text-gray-500">
                      Execution was cancelled before screenshot steps could run
                    </div>
                  )}
                  {execution.status === "completed" && (
                    <div className="text-xs text-gray-500">
                      This workflow does not include screenshot steps
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <>
                <div className="flex-1 p-3 overflow-auto">
                  <div className="space-y-4" data-testid={selectors.executions.viewer.screenshots}>
                    {screenshots.map((screenshot) => (
                      <div
                        key={screenshot.id}
                        ref={(node) => {
                          if (node) {
                            screenshotRefs.current[screenshot.id] = node;
                          } else {
                            delete screenshotRefs.current[screenshot.id];
                          }
                        }}
                        onClick={() => setSelectedScreenshot(screenshot)}
                        className={clsx(
                          "cursor-pointer overflow-hidden rounded-xl border transition-all duration-200",
                          selectedScreenshot?.id === screenshot.id
                            ? "border-flow-accent/80 shadow-[0_22px_50px_rgba(59,130,246,0.35)]"
                            : "border-gray-800 hover:border-flow-accent/50 hover:shadow-[0_15px_40px_rgba(59,130,246,0.2)]",
                        )}
                        data-testid={selectors.timeline.frame}
                      >
                        <div className="bg-slate-900/80 px-3 py-2 flex items-center justify-between text-xs text-slate-300">
                          <span className="truncate font-medium">
                            {screenshot.stepName}
                          </span>
                          <span className="text-slate-400">
                            {format(screenshot.timestamp, "HH:mm:ss.SSS")}
                          </span>
                        </div>
                        <img
                          src={screenshot.url}
                          alt={screenshot.stepName}
                          loading="lazy"
                          className="block w-full"
                          data-testid={selectors.executions.viewer.screenshot}
                        />
                      </div>
                    ))}
                  </div>
                </div>

                <div className="border-t border-gray-800 p-2 overflow-x-auto">
                  <div className="flex gap-2">
                    {screenshots.map((screenshot) => (
                      <div
                        key={screenshot.id}
                        onClick={() => setSelectedScreenshot(screenshot)}
                        className={clsx(
                          "flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-150",
                          selectedScreenshot?.id === screenshot.id
                            ? "border-flow-accent shadow-[0_12px_30px_rgba(59,130,246,0.35)]"
                            : "border-gray-700 hover:border-flow-accent/60",
                        )}
                      >
                        <img
                          src={screenshot.url}
                          alt={screenshot.stepName}
                          loading="lazy"
                          className="w-full h-full object-cover"
                        />
                      </div>
                    ))}
                  </div>
                </div>
              </>
            ),
          logs: (
            <div
              className="flex-1 overflow-auto p-3"
              data-testid={selectors.executions.viewer.logs}
            >
              <div className="terminal-output">
                {execution.logs.map((log) => (
                  <div
                    key={log.id}
                    className="flex gap-2 mb-1"
                    data-testid={selectors.executions.logEntry}
                  >
                    <span className="text-xs text-gray-600">
                      {format(log.timestamp, "HH:mm:ss")}
                    </span>
                    <span className={`flex-1 text-xs ${getLogColor(log.level)}`}>
                      {log.message}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ),
          executions: (
            <div className="flex-1 overflow-hidden p-3">
              {execution.workflowId ? (
                <div className="h-full overflow-hidden rounded-xl border border-gray-800 bg-flow-node/40">
                  <ExecutionHistory
                    workflowId={execution.workflowId}
                    onSelectExecution={handleExecutionSwitch}
                  />
                </div>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-gray-400">
                  Workflow identifier unavailable for this execution.
                </div>
              )}
            </div>
          ),
        }}
      />

      <ExportDialog
        isOpen={isExportDialogOpen}
        onClose={handleCloseExportDialog}
        onConfirm={handleConfirmExport}
        dialogTitleId={exportDialogTitleId}
        dialogDescriptionId={exportDialogDescriptionId}
        exportFormat={exportFormat}
        setExportFormat={setExportFormat}
        isBinaryExport={isBinaryExport}
        exportFormatOptions={EXPORT_FORMAT_OPTIONS}
        dimensionPresetOptions={dimensionPresetOptions}
        dimensionPreset={dimensionPreset}
        setDimensionPreset={setDimensionPreset}
        selectedDimensions={selectedDimensions}
        customWidthInput={customWidthInput}
        customHeightInput={customHeightInput}
        setCustomWidthInput={setCustomWidthInput}
        setCustomHeightInput={setCustomHeightInput}
        exportFileStem={exportFileStem}
        setExportFileStem={setExportFileStem}
        defaultExportFileStem={defaultExportFileStem}
        finalFileName={finalFileName}
        supportsFileSystemAccess={supportsFileSystemAccess}
        useNativeFilePicker={useNativeFilePicker}
        setUseNativeFilePicker={setUseNativeFilePicker}
        preparedMovieSpec={preparedMovieSpec}
        composerPreviewUrl={composerPreviewUrl}
        firstFramePreviewUrl={firstFramePreviewUrl}
        firstFrameLabel={firstFrameLabel}
        previewComposerRef={previewComposerRef}
        composerPreviewWindowRef={composerPreviewWindowRef}
        setIsPreviewComposerReady={setIsPreviewComposerReady}
        setPreviewComposerError={setPreviewComposerError}
        composerPreviewOriginRef={composerPreviewOriginRef}
        isPreviewComposerReady={isPreviewComposerReady}
        previewComposerError={previewComposerError}
        isExporting={isExporting}
        isExportPreviewLoading={isExportPreviewLoading}
        replayFramesLength={replayFrames.length}
        exportStatusMessage={exportStatusMessage}
        estimatedFrameCount={estimatedFrameCount}
        estimatedDurationSeconds={estimatedDurationSeconds}
        activeSpecId={activeSpecId}
        previewMetrics={previewMetrics}
        formatSeconds={formatSeconds}
      />

      {/* Export Success Panel */}
      {showExportSuccess && lastCreatedExport && (
        <ExportSuccessPanel
          export_={lastCreatedExport}
          onClose={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
          }}
          onViewInLibrary={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
            // Navigate to exports tab - this will be handled by parent component
            window.dispatchEvent(new CustomEvent('navigate-to-exports'));
          }}
          onViewExecution={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
          }}
        />
      )}
    </div>
  );
}

interface EmptyExecutionViewerProps {
  workflowId: string;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

interface ExecutionViewerProps {
  workflowId: string;
  execution: Execution | null;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

const useCurrentWorkflowName = () =>
  useWorkflowStore((state) => state.currentWorkflow?.name ?? "Workflow");

const useWorkflowSave = () => useWorkflowStore((state) => state.saveWorkflow);

function EmptyExecutionViewer({
  workflowId,
  onClose,
  showExecutionSwitcher = false,
}: EmptyExecutionViewerProps) {
  const [activeTab, setActiveTab] = useState<ViewerTab>("replay");
  const [isStarting, setIsStarting] = useState(false);
  const { startExecution, loadExecution } = useExecutionActions();
  const workflowName = useCurrentWorkflowName();
  const saveWorkflow = useWorkflowSave();

  const handleStart = useCallback(async () => {
    if (!workflowId) {
      toast.error("Unable to determine workflow to execute");
      return;
    }

    setIsStarting(true);
    try {
      await startExecution(workflowId, () =>
        saveWorkflow({
          source: "execution-run",
          changeDescription: "Autosave before execution",
        }),
      );
      toast.success("Workflow execution started");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to start workflow";
      toast.error(message);
    } finally {
      setIsStarting(false);
    }
  }, [startExecution, workflowId, saveWorkflow]);

  const handleHistorySelect = useCallback(
    async (candidate: { id: string }) => {
      if (!candidate?.id) {
        return;
      }
      try {
        await loadExecution(candidate.id);
        setActiveTab("replay");
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to load execution";
        toast.error(message);
      }
    },
    [loadExecution],
  );

  const renderStartButton = (options?: { variant?: "compact" | "large" }) => {
    const variant = options?.variant ?? "compact";
    const baseClasses =
      "inline-flex items-center gap-2 rounded-md font-medium transition focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-60";
    const palette =
      variant === "large"
        ? "bg-flow-accent px-5 py-2.5 text-base text-white shadow-lg shadow-flow-accent/30 hover:bg-flow-accent/90"
        : "bg-flow-accent px-3 py-1.5 text-sm text-white shadow-md shadow-flow-accent/30 hover:bg-flow-accent/85";

    return (
      <button
        type="button"
        onClick={handleStart}
        disabled={isStarting}
        className={`${baseClasses} ${palette}`}
      >
        {isStarting ? (
          <Loader size={16} className="animate-spin" />
        ) : (
          <PlayCircle size={16} />
        )}
        <span>{isStarting ? "Starting…" : "Start workflow"}</span>
      </button>
    );
  };

  const renderEmptyTab = (title: string, description: string) => (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 px-6 text-center">
      <div className="max-w-md space-y-2">
        <h3 className="text-lg font-semibold text-white">{title}</h3>
        <p className="text-sm text-slate-400">{description}</p>
      </div>
      {renderStartButton({ variant: "large" })}
    </div>
  );

  return (
    <div data-testid={selectors.executions.viewer.root} className="h-full flex flex-col bg-flow-node min-h-0">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          <PlayCircle size={20} className="text-flow-accent" />
          <div>
            <div className="text-sm font-medium text-white">{workflowName}</div>
            <div className="text-xs text-gray-500">
              Execution viewer ready — no runs started yet
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {renderStartButton()}
          {onClose && (
            <button
              className="toolbar-button p-1.5 ml-2 border-l border-gray-700 pl-3"
              title="Close"
              onClick={onClose}
            >
              <X size={14} />
            </button>
          )}
        </div>
      </div>

      <div className="h-2 bg-flow-bg">
        <div
          className="h-full bg-flow-accent transition-all duration-300"
          style={{ width: "0%" }}
        />
      </div>

      <div className="flex border-b border-gray-800">
        <button
          data-testid={selectors.executions.tabs.replay}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "replay"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("replay")}
        >
          <PlayCircle size={14} />
          Replay (0)
        </button>
        <button
          data-testid={selectors.executions.tabs.screenshots}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "screenshots"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("screenshots")}
        >
          <Image size={14} />
          Screenshots (0)
        </button>
        <button
          data-testid={selectors.executions.tabs.logs}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "logs"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("logs")}
        >
          <Terminal size={14} />
          Logs (0)
        </button>
        {showExecutionSwitcher && (
          <button
            data-testid={selectors.executions.tabs.executions}
            className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
              activeTab === "executions"
                ? "bg-flow-bg text-white border-b-2 border-flow-accent"
                : "text-gray-400 hover:text-white"
            }`}
            onClick={() => setActiveTab("executions")}
          >
            <ListTree size={14} />
            Executions
          </button>
        )}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {activeTab === "replay" ? (
          renderEmptyTab(
            "No execution in progress",
            "Start the workflow to capture a replay of each automation step.",
          )
        ) : activeTab === "screenshots" ? (
          renderEmptyTab(
            "Screenshots will appear here",
            "As the workflow runs, screenshots from each step are collected for review.",
          )
        ) : activeTab === "logs" ? (
          renderEmptyTab(
            "Execution logs are empty",
            "Run the workflow to stream live log output, retries, and console messages.",
          )
        ) : showExecutionSwitcher && workflowId ? (
          <div className="flex-1 overflow-auto p-4 space-y-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-medium text-white">
                  Execution history
                </h3>
                <p className="text-xs text-slate-400">
                  Select a past execution to inspect its replay, screenshots,
                  and logs.
                </p>
              </div>
              {renderStartButton()}
            </div>
            <ExecutionHistory
              workflowId={workflowId}
              onSelectExecution={handleHistorySelect}
            />
          </div>
        ) : (
          renderEmptyTab(
            "Execution history unavailable",
            "Enable execution history to review previous workflow runs.",
          )
        )}
      </div>
    </div>
  );
}

function ExecutionViewer({
  workflowId,
  execution,
  onClose,
  showExecutionSwitcher,
}: ExecutionViewerProps) {
  if (!workflowId) return null;

  if (execution && execution.id) {
    return (
      <ActiveExecutionViewer
        execution={execution}
        onClose={onClose}
        showExecutionSwitcher={showExecutionSwitcher}
      />
    );
  }

  return (
    <EmptyExecutionViewer
      workflowId={workflowId}
      onClose={onClose}
      showExecutionSwitcher={showExecutionSwitcher}
    />
  );
}

export default ExecutionViewer;
