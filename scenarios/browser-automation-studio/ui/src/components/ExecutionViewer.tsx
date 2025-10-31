import { useState, useEffect, useMemo, useCallback, useRef, type CSSProperties } from 'react';
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
} from 'lucide-react';
import { format } from 'date-fns';
import clsx from 'clsx';
import ReplayPlayer, {
  ReplayFrame,
  type ReplayChromeTheme,
  type ReplayBackgroundTheme,
} from './ReplayPlayer';
import { useExecutionStore } from '../stores/executionStore';
import type {
  Execution,
  TimelineFrame,
  TimelineArtifact,
} from '../stores/executionStore';
import type { Screenshot, LogEntry } from '../stores/executionEventProcessor';
import { toast } from 'react-hot-toast';
import {
  toNumber,
  toBoundingBox,
  toPoint,
  mapTrail,
  mapRegions,
  mapRetryHistory,
  mapAssertion,
  resolveUrl,
} from '../utils/executionTypeMappers';

interface ExecutionProps {
  execution: Execution;
  onClose?: () => void;
}

const HEARTBEAT_WARN_SECONDS = 8;
const HEARTBEAT_STALL_SECONDS = 15;

const REPLAY_CHROME_OPTIONS: Array<{ id: ReplayChromeTheme; label: string; subtitle: string }> = [
  { id: 'aurora', label: 'Aurora', subtitle: 'macOS-inspired chrome' },
  { id: 'chromium', label: 'Chromium', subtitle: 'Modern minimal controls' },
  { id: 'midnight', label: 'Midnight', subtitle: 'Gradient showcase frame' },
  { id: 'minimal', label: 'Minimal', subtitle: 'Hide browser chrome' },
];

const REPLAY_BACKGROUND_OPTIONS: Array<{
  id: ReplayBackgroundTheme;
  label: string;
  subtitle: string;
  previewStyle: CSSProperties;
  kind: 'abstract' | 'solid' | 'minimal';
}> = [
  {
    id: 'aurora',
    label: 'Aurora Glow',
    subtitle: 'Iridescent gradient wash',
    previewStyle: {
      backgroundImage: 'linear-gradient(135deg, rgba(56,189,248,0.7), rgba(129,140,248,0.7))',
    },
    kind: 'abstract',
  },
  {
    id: 'sunset',
    label: 'Sunset Bloom',
    subtitle: 'Fuchsia → amber ambience',
    previewStyle: {
      backgroundImage: 'linear-gradient(135deg, rgba(244,114,182,0.75), rgba(251,191,36,0.7))',
    },
    kind: 'abstract',
  },
  {
    id: 'ocean',
    label: 'Ocean Depths',
    subtitle: 'Cerulean blue gradient',
    previewStyle: {
      backgroundImage: 'linear-gradient(135deg, rgba(14,165,233,0.78), rgba(30,64,175,0.82))',
    },
    kind: 'abstract',
  },
  {
    id: 'nebula',
    label: 'Nebula Drift',
    subtitle: 'Cosmic violet haze',
    previewStyle: {
      backgroundImage: 'linear-gradient(135deg, rgba(147,51,234,0.78), rgba(99,102,241,0.78))',
    },
    kind: 'abstract',
  },
  {
    id: 'grid',
    label: 'Tech Grid',
    subtitle: 'Futuristic lattice backdrop',
    previewStyle: {
      backgroundColor: '#0f172a',
      backgroundImage:
        'linear-gradient(rgba(96,165,250,0.2) 1px, transparent 1px), linear-gradient(90deg, rgba(96,165,250,0.18) 1px, transparent 1px)',
      backgroundSize: '12px 12px',
    },
    kind: 'abstract',
  },
  {
    id: 'charcoal',
    label: 'Charcoal',
    subtitle: 'Deep neutral tone',
    previewStyle: {
      backgroundColor: '#0f172a',
    },
    kind: 'solid',
  },
  {
    id: 'steel',
    label: 'Steel Slate',
    subtitle: 'Cool slate finish',
    previewStyle: {
      backgroundColor: '#1f2937',
    },
    kind: 'solid',
  },
  {
    id: 'emerald',
    label: 'Evergreen',
    subtitle: 'Saturated green solid',
    previewStyle: {
      backgroundColor: '#064e3b',
    },
    kind: 'solid',
  },
  {
    id: 'none',
    label: 'No Background',
    subtitle: 'Edge-to-edge browser',
    previewStyle: {
      backgroundColor: 'transparent',
      backgroundImage:
        'linear-gradient(45deg, rgba(148,163,184,0.35) 25%, transparent 25%, transparent 50%, rgba(148,163,184,0.35) 50%, rgba(148,163,184,0.35) 75%, transparent 75%, transparent)',
      backgroundSize: '10px 10px',
    },
    kind: 'minimal',
  },
];

const isReplayChromeTheme = (value: string | null | undefined): value is ReplayChromeTheme =>
  Boolean(value && REPLAY_CHROME_OPTIONS.some((option) => option.id === value));

const isReplayBackgroundTheme = (
  value: string | null | undefined,
): value is ReplayBackgroundTheme =>
  Boolean(value && REPLAY_BACKGROUND_OPTIONS.some((option) => option.id === value));

function ExecutionViewer({ execution, onClose }: ExecutionProps) {
  const refreshTimeline = useExecutionStore((state) => state.refreshTimeline);
  const stopExecution = useExecutionStore((state) => state.stopExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const [activeTab, setActiveTab] = useState<'replay' | 'screenshots' | 'logs'>('replay');
  const [hasAutoSwitchedToReplay, setHasAutoSwitchedToReplay] = useState<boolean>(
    Boolean(execution.timeline && execution.timeline.length > 0)
  );
  const [selectedScreenshot, setSelectedScreenshot] = useState<Screenshot | null>(null);
  const [, setHeartbeatTick] = useState(0);
  const [isStopping, setIsStopping] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [replayChromeTheme, setReplayChromeTheme] = useState<ReplayChromeTheme>(() => {
    if (typeof window === 'undefined') {
      return 'aurora';
    }
    const stored = window.localStorage.getItem('browserAutomation.replayChromeTheme');
    return isReplayChromeTheme(stored) ? stored : 'aurora';
  });
  const [replayBackgroundTheme, setReplayBackgroundTheme] = useState<ReplayBackgroundTheme>(() => {
    if (typeof window === 'undefined') {
      return 'aurora';
    }
    const stored = window.localStorage.getItem('browserAutomation.replayBackgroundTheme');
    return isReplayBackgroundTheme(stored) ? stored : 'aurora';
  });
  const screenshotRefs = useRef<Record<string, HTMLDivElement | null>>({});

  const heartbeatTimestamp = execution.lastHeartbeat?.timestamp?.valueOf();
  const executionError = execution.error ?? undefined;

  useEffect(() => {
    if (execution.status !== 'running' || !heartbeatTimestamp) {
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
    setHasAutoSwitchedToReplay(Boolean(execution.timeline && execution.timeline.length > 0));
    setActiveTab('replay');
    setSelectedScreenshot(null);
  }, [execution.id]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    try {
      window.localStorage.setItem('browserAutomation.replayChromeTheme', replayChromeTheme);
    } catch (err) {
      console.warn('Failed to persist replay chrome theme', err);
    }
  }, [replayChromeTheme]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    try {
      window.localStorage.setItem('browserAutomation.replayBackgroundTheme', replayBackgroundTheme);
    } catch (err) {
      console.warn('Failed to persist replay background theme', err);
    }
  }, [replayBackgroundTheme]);

  useEffect(() => {
    if (!hasAutoSwitchedToReplay && execution.timeline && execution.timeline.length > 0) {
      setActiveTab('replay');
      setHasAutoSwitchedToReplay(true);
    }
  }, [execution.timeline, hasAutoSwitchedToReplay]);

  const heartbeatAgeSeconds = useMemo(() => {
    if (!execution.lastHeartbeat || !execution.lastHeartbeat.timestamp) {
      return null;
    }
    const age = (Date.now() - execution.lastHeartbeat.timestamp.getTime()) / 1000;
    return age < 0 ? 0 : age;
  }, [execution.lastHeartbeat]);

  const inStepSeconds = execution.lastHeartbeat?.elapsedMs != null
    ? Math.max(0, execution.lastHeartbeat.elapsedMs / 1000)
    : null;

  const formatSeconds = (value: number) => {
    if (Number.isNaN(value) || !Number.isFinite(value)) {
      return '0s';
    }
    if (value >= 10) {
      return `${Math.round(value)}s`;
    }
    return `${value.toFixed(1)}s`;
  };

  const heartbeatAgeLabel = heartbeatAgeSeconds == null
    ? null
    : heartbeatAgeSeconds < 0.75
      ? 'just now'
      : `${formatSeconds(heartbeatAgeSeconds)} ago`;

  const inStepLabel = inStepSeconds != null ? formatSeconds(inStepSeconds) : null;

  type HeartbeatState = 'idle' | 'awaiting' | 'healthy' | 'delayed' | 'stalled';

  const heartbeatState: HeartbeatState = useMemo(() => {
    if (execution.status !== 'running') {
      return 'idle';
    }
    if (!execution.lastHeartbeat) {
      return 'awaiting';
    }
    if (heartbeatAgeSeconds == null) {
      return 'awaiting';
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_STALL_SECONDS) {
      return 'stalled';
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_WARN_SECONDS) {
      return 'delayed';
    }
    return 'healthy';
  }, [execution.status, execution.lastHeartbeat, heartbeatAgeSeconds]);

  const heartbeatDescriptor = useMemo(() => {
    switch (heartbeatState) {
      case 'idle':
        return null;
      case 'awaiting':
        return {
          tone: 'awaiting' as const,
          iconClass: 'text-amber-400',
          textClass: 'text-amber-200/90',
          label: 'Awaiting first heartbeat…',
        };
      case 'healthy':
        return {
          tone: 'healthy' as const,
          iconClass: 'text-blue-400',
          textClass: 'text-blue-200',
          label: `Heartbeat ${heartbeatAgeLabel ?? 'just now'}`,
        };
      case 'delayed':
        return {
          tone: 'delayed' as const,
          iconClass: 'text-amber-400',
          textClass: 'text-amber-200',
          label: `Heartbeat delayed (${formatSeconds(heartbeatAgeSeconds ?? 0)} since last update)`,
        };
      case 'stalled':
        return {
          tone: 'stalled' as const,
          iconClass: 'text-red-400',
          textClass: 'text-red-200',
          label: `Heartbeat stalled (${formatSeconds(heartbeatAgeSeconds ?? 0)} without update)`,
        };
      default:
        return null;
    }
  }, [heartbeatState, heartbeatAgeLabel, heartbeatAgeSeconds]);

  const statusMessage = useMemo(() => {
    const label = typeof execution.currentStep === 'string' ? execution.currentStep.trim() : '';
    if (label.length > 0) {
      return label;
    }
    switch (execution.status) {
      case 'pending':
        return 'Pending...';
      case 'running':
        return 'Running...';
      case 'completed':
        return 'Completed';
      case 'failed':
        return execution.error ? 'Failed' : 'Failed';
      default:
        return 'Initializing...';
    }
  }, [execution.currentStep, execution.status, execution.error]);

  const isRunning = execution.status === 'running';
  const canRestart = Boolean(execution.workflowId) && execution.status !== 'running';

  const timelineForReplay = useMemo(() => {
    return (execution.timeline ?? []).filter((frame) => {
      if (!frame) {
        return false;
      }
      const type = (typeof frame.step_type === 'string' ? frame.step_type : typeof frame.stepType === 'string' ? frame.stepType : '').toLowerCase();
      return type !== 'screenshot';
    });
  }, [execution.timeline]);

  const replayFrames = useMemo<ReplayFrame[]>(() => {
    return timelineForReplay.map((frame: TimelineFrame, index: number) => {
      const screenshotData = frame?.screenshot ?? undefined;
      const screenshotUrl = resolveUrl(screenshotData?.url);
      const thumbnailUrl = resolveUrl(screenshotData?.thumbnail_url);

      const focused = frame?.focused_element ?? frame?.focusedElement;
      const focusedRaw = (focused ?? undefined) as (Record<string, unknown> | undefined);
      const focusedBoundingBox = toBoundingBox(
        (focusedRaw?.bounding_box as unknown) ?? (focusedRaw?.boundingBox as unknown)
      );
      const normalizedFocusedBoundingBox = focusedBoundingBox ?? undefined;
      const totalDuration = toNumber(frame?.total_duration_ms ?? frame?.totalDurationMs);
      const retryAttempt = toNumber(frame?.retry_attempt ?? frame?.retryAttempt);
      const retryMaxAttempts = toNumber(frame?.retry_max_attempts ?? frame?.retryMaxAttempts);
      const retryConfigured = toNumber(frame?.retry_configured ?? frame?.retryConfigured);
      const retryDelayMs = toNumber(frame?.retry_delay_ms ?? frame?.retryDelayMs);
      const retryBackoffFactor = toNumber(frame?.retry_backoff_factor ?? frame?.retryBackoffFactor);
      const retryHistory = mapRetryHistory(frame?.retry_history ?? frame?.retryHistory);
      const domSnapshotArtifact = Array.isArray(frame?.artifacts)
        ? frame.artifacts.find((artifact: TimelineArtifact) => artifact?.type === 'dom_snapshot')
        : undefined;
      const domSnapshotHtml = domSnapshotArtifact?.payload && typeof domSnapshotArtifact.payload === 'object'
        ? (() => {
            const payload = domSnapshotArtifact.payload as Record<string, unknown>;
            const html = payload?.html;
            return typeof html === 'string' ? html : undefined;
          })()
        : undefined;
      const domSnapshotPreview = typeof (frame?.dom_snapshot_preview ?? frame?.domSnapshotPreview) === 'string'
        ? (frame.dom_snapshot_preview ?? frame.domSnapshotPreview)
        : undefined;
      const domSnapshotArtifactId = typeof (frame?.dom_snapshot_artifact_id ?? frame?.domSnapshotArtifactId) === 'string'
        ? (frame.dom_snapshot_artifact_id ?? frame.domSnapshotArtifactId)
        : (typeof domSnapshotArtifact?.id === 'string' ? domSnapshotArtifact.id : undefined);

      const mappedFrame: ReplayFrame = {
        id: screenshotData?.artifact_id || frame?.timeline_artifact_id || `frame-${index}`,
        stepIndex: typeof frame?.step_index === 'number' ? frame.step_index : index,
        nodeId: typeof frame?.node_id === 'string' ? frame.node_id : undefined,
        stepType: typeof frame?.step_type === 'string' ? frame.step_type : undefined,
        status: typeof frame?.status === 'string' ? frame.status : undefined,
        success: Boolean(frame?.success),
        durationMs: toNumber(frame?.duration_ms ?? frame?.durationMs),
        totalDurationMs: totalDuration,
        progress: toNumber(frame?.progress),
        finalUrl:
          typeof frame?.final_url === 'string'
            ? frame.final_url
            : typeof frame?.finalUrl === 'string'
              ? frame.finalUrl
              : undefined,
        error: typeof frame?.error === 'string' ? frame.error : undefined,
        extractedDataPreview: frame?.extracted_data_preview ?? frame?.extractedDataPreview,
        consoleLogCount: toNumber(frame?.console_log_count ?? frame?.consoleLogCount),
        networkEventCount: toNumber(frame?.network_event_count ?? frame?.networkEventCount),
        screenshot: screenshotData
          ? {
              artifactId: screenshotData.artifact_id || `artifact-${index}`,
              url: screenshotUrl,
              thumbnailUrl,
              width: toNumber(screenshotData.width),
              height: toNumber(screenshotData.height),
              contentType: typeof screenshotData.content_type === 'string' ? screenshotData.content_type : undefined,
              sizeBytes: toNumber(screenshotData.size_bytes),
            }
          : undefined,
        highlightRegions: mapRegions(frame?.highlight_regions ?? frame?.highlightRegions),
        maskRegions: mapRegions(frame?.mask_regions ?? frame?.maskRegions),
        focusedElement:
          focused || normalizedFocusedBoundingBox
            ? {
                selector: typeof focused?.selector === 'string' ? focused.selector : undefined,
                boundingBox: normalizedFocusedBoundingBox,
              }
            : null,
        elementBoundingBox: toBoundingBox(frame?.element_bounding_box ?? frame?.elementBoundingBox) ?? null,
        clickPosition: toPoint(frame?.click_position ?? frame?.clickPosition) ?? null,
        cursorTrail: mapTrail(frame?.cursor_trail ?? frame?.cursorTrail),
        zoomFactor: toNumber(frame?.zoom_factor ?? frame?.zoomFactor),
        assertion: mapAssertion(frame?.assertion) ?? undefined,
        retryAttempt,
        retryMaxAttempts,
        retryConfigured,
        retryDelayMs,
        retryBackoffFactor,
        retryHistory,
        domSnapshotHtml,
        domSnapshotPreview,
        domSnapshotArtifactId,
      };

      const hasScreenshot = Boolean(mappedFrame.screenshot?.url || mappedFrame.screenshot?.thumbnailUrl);
      return hasScreenshot ? mappedFrame : { ...mappedFrame, screenshot: undefined };
    });
  }, [timelineForReplay]);

  const hasTimeline = replayFrames.length > 0;

  const timelineScreenshots = useMemo(() => {
    const frames = execution.timeline ?? [];
    const items: Screenshot[] = [];
    frames.forEach((frame: TimelineFrame, index: number) => {
      const resolved = resolveUrl(frame?.screenshot?.url);
      if (!resolved) {
        return;
      }
      items.push({
        id: frame?.screenshot?.artifact_id || `timeline-${index}`,
        url: resolved,
        stepName:
          frame?.node_id ||
          frame?.step_type ||
          (typeof frame?.step_index === 'number' ? `Step ${frame.step_index + 1}` : 'Step'),
        timestamp: frame?.started_at ? new Date(frame.started_at) : new Date(),
      });
    });
    return items;
  }, [execution.timeline]);

  const backgroundOptionGroups = useMemo(
    () => ({
      abstract: REPLAY_BACKGROUND_OPTIONS.filter((option) => option.kind === 'abstract'),
      solid: REPLAY_BACKGROUND_OPTIONS.filter((option) => option.kind === 'solid'),
      minimal: REPLAY_BACKGROUND_OPTIONS.filter((option) => option.kind === 'minimal'),
    }),
    [],
  );

  const renderBackgroundOption = (option: (typeof REPLAY_BACKGROUND_OPTIONS)[number]) => (
    <button
      key={option.id}
      type="button"
      onClick={() => setReplayBackgroundTheme(option.id)}
      title={option.subtitle}
      className={clsx(
        'group flex items-center gap-2 rounded-full border px-2.5 py-1.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900',
        replayBackgroundTheme === option.id
          ? 'border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]'
          : 'border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/50 hover:text-white',
      )}
    >
      <span
        className="h-6 w-12 rounded-full border border-white/10 shadow-inner transition-transform duration-200 group-hover:scale-[1.03]"
        style={option.previewStyle}
      />
      <span>{option.label}</span>
    </button>
  );

  const screenshots = timelineScreenshots.length > 0 ? timelineScreenshots : execution.screenshots;

  useEffect(() => {
    if (screenshots.length === 0) {
      return;
    }
    const alreadySelected = selectedScreenshot && screenshots.some((shot) => shot.id === selectedScreenshot.id);
    if (!alreadySelected) {
      setSelectedScreenshot(screenshots[screenshots.length - 1]);
    }
  }, [screenshots, selectedScreenshot]);

  useEffect(() => {
    if (!selectedScreenshot) {
      return;
    }
    const element = screenshotRefs.current[selectedScreenshot.id];
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start', inline: 'nearest' });
    }
  }, [selectedScreenshot]);

  useEffect(() => {
    if (activeTab === 'screenshots' && screenshots.length === 0 && replayFrames.length > 0) {
      setActiveTab('replay');
    }
  }, [activeTab, screenshots.length, replayFrames.length]);

  useEffect(() => {
    let interval: number | undefined;
    if (execution.status === 'running') {
      interval = window.setInterval(() => {
        void refreshTimeline(execution.id);
      }, 2000);
    } else if (execution.status === 'completed' || execution.status === 'failed') {
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
      toast.success('Execution stopped');
      await refreshTimeline(execution.id);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to stop execution';
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
      toast.error('Workflow identifier missing for restart');
      return;
    }
    setIsRestarting(true);
    try {
      await startExecution(execution.workflowId);
      toast.success('Workflow restarted');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to restart workflow';
      toast.error(message);
    } finally {
      setIsRestarting(false);
    }
  }, [canRestart, execution.workflowId, isRestarting, startExecution]);

  const getStatusIcon = () => {
    switch (execution.status) {
      case 'running':
        return <Loader size={16} className="animate-spin text-blue-400" />;
      case 'completed':
        return <CheckCircle size={16} className="text-green-400" />;
      case 'failed':
        return <XCircle size={16} className="text-red-400" />;
      default:
        return <Clock size={16} className="text-gray-400" />;
    }
  };

  const getLogColor = (level: LogEntry['level']) => {
    switch (level) {
      case 'error':
        return 'text-red-400';
      case 'warning':
        return 'text-yellow-400';
      case 'success':
        return 'text-green-400';
      default:
        return 'text-gray-300';
    }
  };

  return (
    <div className="h-full flex flex-col bg-flow-node min-h-0">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          {getStatusIcon()}
          <div>
            <div className="text-sm font-medium text-white">
              Execution #{execution.id.slice(0, 8)}
            </div>
            <div className="text-xs text-gray-500">{statusMessage}</div>
            {heartbeatDescriptor && (
              <div
                className="mt-1 flex items-center gap-2 text-[11px]"
              >
                {heartbeatDescriptor.tone === 'stalled' ? (
                  <AlertTriangle size={12} className={heartbeatDescriptor.iconClass} />
                ) : (
                  <Activity size={12} className={heartbeatDescriptor.iconClass} />
                )}
                <span className={heartbeatDescriptor.textClass}>{heartbeatDescriptor.label}</span>
                {inStepLabel && execution.lastHeartbeat && (
                  <span className={`${heartbeatDescriptor.textClass} opacity-80`}>
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
            title={canRestart ? 'Restart workflow' : 'Stop execution before restarting'}
            onClick={handleRestart}
            disabled={!canRestart || isRestarting || isStopping}
          >
            {isRestarting ? <Loader size={14} className="animate-spin" /> : <RotateCw size={14} />}
          </button>
          <button
            className="toolbar-button p-1.5 text-red-400 disabled:text-red-400/50 disabled:cursor-not-allowed"
            title={isRunning ? 'Stop execution' : 'Execution not running'}
            onClick={handleStop}
            disabled={!isRunning || isStopping}
          >
            {isStopping ? <Loader size={14} className="animate-spin" /> : <Square size={14} />}
          </button>
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
          style={{ width: `${execution.progress}%` }}
        />
      </div>

      <div className="flex border-b border-gray-800">
        <button
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'replay'
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent'
              : hasTimeline
                ? 'text-gray-400 hover:text-white'
                : 'text-gray-500 hover:text-white/80'
          }`}
          onClick={() => setActiveTab('replay')}
        >
          <PlayCircle size={14} />
          Replay ({replayFrames.length})
        </button>
        <button
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'screenshots'
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent'
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('screenshots')}
        >
          <Image size={14} />
          Screenshots ({screenshots.length})
        </button>
        <button
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'logs'
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent'
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('logs')}
        >
          <Terminal size={14} />
          Logs ({execution.logs.length})
        </button>
      </div>

      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {activeTab === 'replay' ? (
          <div className="flex-1 overflow-auto p-3 space-y-3">
            {!hasTimeline && (
              <div className="rounded-lg border border-dashed border-slate-700/60 bg-slate-900/60 px-4 py-3 text-sm text-slate-200/80">
                Replay frames stream in as each action runs. Leave this tab open to tailor the final cut in real time.
              </div>
            )}
            {execution.status === 'failed' && execution.progress < 100 && (
              <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 flex items-start gap-3">
                <AlertTriangle size={18} className="text-rose-400 flex-shrink-0 mt-0.5" />
                <div className="flex-1 text-sm">
                  <div className="font-medium text-rose-200 mb-1">
                    Execution Failed - Replay Incomplete
                  </div>
                  <div className="text-rose-100/80">
                    This replay shows only {replayFrames.length} of the workflow's steps.
                    Execution failed at: {execution.currentStep || 'unknown step'}
                  </div>
                  {executionError && (
                    <div className="mt-2 text-xs font-mono text-rose-100/70 bg-rose-950/30 px-2 py-1 rounded">
                      {executionError}
                    </div>
                  )}
                </div>
              </div>
            )}
            <div className="rounded-2xl border border-white/5 bg-slate-950/40 px-4 py-3 shadow-inner space-y-4">
              <div>
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">Browser frame</span>
                  <span className="text-[11px] text-slate-500">Customize the replay window</span>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {REPLAY_CHROME_OPTIONS.map((option) => (
                    <button
                      key={option.id}
                      type="button"
                      onClick={() => setReplayChromeTheme(option.id)}
                      title={option.subtitle}
                      className={clsx(
                        'rounded-full px-3 py-1.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900',
                        replayChromeTheme === option.id
                          ? 'bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]'
                          : 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80',
                      )}
                    >
                      {option.label}
                    </button>
                  ))}
                </div>
              </div>

              <div className="border-t border-white/5 pt-3 space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">Background</span>
                  <span className="text-[11px] text-slate-500">Set the stage behind the browser</span>
                </div>
                <div className="space-y-3">
                  {backgroundOptionGroups.abstract.length > 0 && (
                    <div>
                      <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">Abstract</span>
                      <div className="mt-1 flex flex-wrap gap-2">
                        {backgroundOptionGroups.abstract.map(renderBackgroundOption)}
                      </div>
                    </div>
                  )}
                  {backgroundOptionGroups.solid.length > 0 && (
                    <div>
                      <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">Solid</span>
                      <div className="mt-1 flex flex-wrap gap-2">
                        {backgroundOptionGroups.solid.map(renderBackgroundOption)}
                      </div>
                    </div>
                  )}
                  {backgroundOptionGroups.minimal.length > 0 && (
                    <div>
                      <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">Minimal</span>
                      <div className="mt-1 flex flex-wrap gap-2">
                        {backgroundOptionGroups.minimal.map(renderBackgroundOption)}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
            <ReplayPlayer
              frames={replayFrames}
              executionStatus={execution.status}
              chromeTheme={replayChromeTheme}
              backgroundTheme={replayBackgroundTheme}
            />
          </div>
        ) : activeTab === 'screenshots' ? (
          screenshots.length === 0 ? (
            <div className="flex flex-1 items-center justify-center p-6 text-center">
              <div>
                <Image size={32} className="mx-auto mb-3 text-gray-600" />
                <div className="text-sm text-gray-400 mb-1">No screenshots captured</div>
                {execution.status === 'failed' && (
                  <div className="text-xs text-gray-500">
                    Execution failed before screenshot steps could run
                  </div>
                )}
                {execution.status === 'completed' && (
                  <div className="text-xs text-gray-500">
                    This workflow does not include screenshot steps
                  </div>
                )}
              </div>
            </div>
          ) : (
            <>
              <div className="flex-1 p-3 overflow-auto">
                <div className="space-y-4">
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
                        'cursor-pointer overflow-hidden rounded-xl border transition-all duration-200',
                        selectedScreenshot?.id === screenshot.id
                          ? 'border-flow-accent/80 shadow-[0_22px_50px_rgba(59,130,246,0.35)]'
                          : 'border-gray-800 hover:border-flow-accent/50 hover:shadow-[0_15px_40px_rgba(59,130,246,0.2)]',
                      )}
                    >
                      <div className="bg-slate-900/80 px-3 py-2 flex items-center justify-between text-xs text-slate-300">
                        <span className="truncate font-medium">{screenshot.stepName}</span>
                        <span className="text-slate-400">{format(screenshot.timestamp, 'HH:mm:ss.SSS')}</span>
                      </div>
                      <img
                        src={screenshot.url}
                        alt={screenshot.stepName}
                        className="block w-full"
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
                        'flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-150',
                        selectedScreenshot?.id === screenshot.id
                          ? 'border-flow-accent shadow-[0_12px_30px_rgba(59,130,246,0.35)]'
                          : 'border-gray-700 hover:border-flow-accent/60',
                      )}
                    >
                      <img
                        src={screenshot.url}
                        alt={screenshot.stepName}
                        className="w-full h-full object-cover"
                      />
                    </div>
                  ))}
                </div>
              </div>
            </>
          )
        ) : (
          <div className="flex-1 overflow-auto p-3">
            <div className="terminal-output">
              {execution.logs.map((log) => (
                <div key={log.id} className="flex gap-2 mb-1">
                  <span className="text-xs text-gray-600">{format(log.timestamp, 'HH:mm:ss')}</span>
                  <span className={`flex-1 text-xs ${getLogColor(log.level)}`}>
                    {log.message}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ExecutionViewer;
