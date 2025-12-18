// eslint-disable-next-line @typescript-eslint/ban-ts-comment
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
import { useExecutionStore } from "@stores/executionStore";
import type { Execution, TimelineFrame } from "@stores/executionStore";
import { useWorkflowStore } from "@stores/workflowStore";
import { useExportStore } from "@stores/exportStore";
import { ExportSuccessPanel } from "./ExportSuccessPanel";
import type { Screenshot, LogEntry } from "@stores/executionEventProcessor";
import { toast } from "react-hot-toast";
import { logger } from "@utils/logger";
import { resolveUrl } from "@utils/executionTypeMappers";
import ExecutionHistory from "./ExecutionHistory";
import { selectors } from "@constants/selectors";
import { useExecutionEvents } from "./hooks/useExecutionEvents";
import { useExecutionActions } from "./hooks/useExecutionActions";
import {
  describePreviewStatusMessage,
  formatCapturedLabel,
  normalizePreviewStatus,
} from "./utils/exportHelpers";
import { coerceMetricNumber } from "./viewer/exportConfig";
import { useReplayCustomization } from "./viewer/useReplayCustomization";
import ReplayCustomizationPanel from "./viewer/ReplayCustomizationPanel";
import ExportDialog from "./viewer/ExportDialog";
import ActiveExecutionTabs from "./viewer/ActiveExecutionTabs";
import { useExecutionHeartbeat, formatSeconds } from "./viewer/useExecutionHeartbeat";
import { useExecutionExport } from "./viewer/useExecutionExport";

interface ActiveExecutionProps {
  execution: Execution;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

export type ViewerTab = "replay" | "screenshots" | "logs" | "executions";

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
  const [isStopping, setIsStopping] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isSwitchingExecution, setIsSwitchingExecution] = useState(false);
  const replayCustomization = useReplayCustomization({ executionId: execution.id });
  const { heartbeatDescriptor, inStepLabel } = useExecutionHeartbeat(execution);
  const screenshotRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const preloadedWorkflowRef = useRef<string | null>(null);
  const composerRef = useRef<HTMLIFrameElement | null>(null);
  const composerWindowRef = useRef<Window | null>(null);
  const composerOriginRef = useRef<string | null>(null);
  const [isComposerReady, setIsComposerReady] = useState(false);
  const composerFrameStateRef = useRef({ frameIndex: 0, progress: 0 });
  const { createExport } = useExportStore();
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
  const exportController = useExecutionExport({
    execution,
    replayFrames: execution.timeline ?? [],
    workflowName,
    replayCustomization,
    createExport,
  });
  const {
    composerApiBase,
    composerUrl,
    composerPreviewUrl,
    movieSpecError,
    setMovieSpecError,
    isMovieSpecLoading,
    preparedMovieSpec,
    previewMetrics,
    setPreviewMetrics,
    activeSpecId,
    setActiveSpecId,
    exportDialogProps,
    isExportDialogOpen,
    openExportDialog,
    closeExportDialog,
    confirmExport,
    showExportSuccess,
    lastCreatedExport,
    dismissExportSuccess,
    replayFrames,
  } = exportController;

  const selectedDimensions = useMemo(() => {
    const fallback = { width: 1440, height: 900 };
    const width =
      preparedMovieSpec?.presentation?.canvas?.width ??
      preparedMovieSpec?.presentation?.viewport?.width ??
      fallback.width;
    const height =
      preparedMovieSpec?.presentation?.canvas?.height ??
      preparedMovieSpec?.presentation?.viewport?.height ??
      fallback.height;
    return {
      width: typeof width === "number" && width > 0 ? width : fallback.width,
      height: typeof height === "number" && height > 0 ? height : fallback.height,
    };
  }, [preparedMovieSpec]);
  const {
    previewComposerRef,
    composerPreviewWindowRef,
    composerPreviewOriginRef,
    setIsPreviewComposerReady,
    setPreviewComposerError,
    isPreviewComposerReady,
    previewComposerError,
  } = exportDialogProps;

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

  const executionError = execution.error ?? undefined;

  useEffect(() => {
    setHasAutoSwitchedToReplay(
      Boolean(execution.timeline && execution.timeline.length > 0),
    );
    setActiveTab("replay");
    setSelectedScreenshot(null);
  }, [execution.id]);

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
    if (
      !hasAutoSwitchedToReplay &&
      execution.timeline &&
      execution.timeline.length > 0
    ) {
      setActiveTab("replay");
      setHasAutoSwitchedToReplay(true);
    }
  }, [execution.timeline, hasAutoSwitchedToReplay]);

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
  const hasTimeline = replayFrames.length > 0;
  const exportDialogTitleId = useId();
  const exportDialogDescriptionId = useId();

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
            <div className="text-sm font-medium text-surface">
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
            onClick={openExportDialog}
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
                <div className="rounded-lg border border-dashed border-flow-border/70 bg-flow-node/70 px-4 py-3 text-sm text-flow-text-secondary">
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
              <div className="relative w-full overflow-hidden rounded-2xl border border-flow-border bg-flow-node/70 shadow-[0_25px_70px_rgba(0,0,0,0.35)]">
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
                  <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/50">
                    <span className="text-xs uppercase tracking-[0.3em] text-flow-text-muted">
                      {isMovieSpecLoading
                        ? "Loading replay spec…"
                        : "Initialising player…"}
                    </span>
                  </div>
                )}
                {movieSpecError && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/60 p-6 text-center">
                    <div className="rounded-xl border border-flow-border bg-flow-node/80 px-6 py-4 text-sm text-flow-text shadow-[0_15px_45px_rgba(0,0,0,0.4)]">
                      <div className="mb-1 font-semibold text-flow-text">
                        Failed to load replay spec
                      </div>
                      <div className="text-xs text-flow-text-secondary">
                        {movieSpecError}
                      </div>
                      {(previewMetrics.capturedFrames > 0 ||
                        previewMetrics.totalDurationMs > 0) && (
                        <div className="mt-3 text-[11px] text-flow-text-muted">
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
                        <div className="mt-1 text-[10px] uppercase tracking-[0.24em] text-flow-text-muted">
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
                        <div className="bg-flow-node/80 px-3 py-2 flex items-center justify-between text-xs text-flow-text-secondary">
                          <span className="truncate font-medium">
                            {screenshot.stepName}
                          </span>
                          <span className="text-flow-text-muted">
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
        onClose={closeExportDialog}
        onConfirm={confirmExport}
        dialogTitleId={exportDialogTitleId}
        dialogDescriptionId={exportDialogDescriptionId}
        {...exportDialogProps}
      />

      {/* Export Success Panel */}
      {showExportSuccess && lastCreatedExport && (
        <ExportSuccessPanel
          export_={lastCreatedExport}
          onClose={dismissExportSuccess}
          onViewInLibrary={() => {
            dismissExportSuccess();
            window.dispatchEvent(new CustomEvent("navigate-to-exports"));
          }}
          onViewExecution={dismissExportSuccess}
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
        <h3 className="text-lg font-semibold text-surface">{title}</h3>
        <p className="text-sm text-flow-text-muted">{description}</p>
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
            <div className="text-sm font-medium text-surface">{workflowName}</div>
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
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : "text-subtle hover:text-surface"
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
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : "text-subtle hover:text-surface"
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
              ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
              : "text-subtle hover:text-surface"
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
                ? "bg-flow-bg text-surface border-b-2 border-flow-accent"
                : "text-subtle hover:text-surface"
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
                <h3 className="text-sm font-medium text-surface">
                  Execution history
                </h3>
                <p className="text-xs text-flow-text-muted">
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

export { parseExportPreviewPayload } from "./viewer/exportPreview";
export default ExecutionViewer;
