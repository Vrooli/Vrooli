import { useState, useEffect, useMemo } from 'react';
import {
  Activity,
  Pause,
  RotateCw,
  X,
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
import ReplayPlayer, { ReplayFrame } from './ReplayPlayer';
import { useExecutionStore } from '../stores/executionStore';
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

interface Screenshot {
  id: string;
  timestamp: Date;
  url: string;
  stepName: string;
}

interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'info' | 'warning' | 'error' | 'success';
  message: string;
}

interface TimelineArtifact {
  id: string;
  type: string;
  label?: string;
  storage_url?: string;
  thumbnail_url?: string;
  content_type?: string;
  size_bytes?: number;
  step_index?: number;
  payload?: Record<string, unknown> | null;
}

interface TimelineFrame {
  screenshot?: { url?: string; artifact_id?: string; thumbnail_url?: string; width?: number; height?: number; content_type?: string; size_bytes?: number } | null;
  step_index?: number;
  node_id?: string;
  step_type?: string;
  started_at?: string | Date;
  focused_element?: { selector?: string; bounding_box?: unknown; boundingBox?: unknown };
  focusedElement?: { selector?: string; bounding_box?: unknown; boundingBox?: unknown };
  element_bounding_box?: unknown;
  elementBoundingBox?: unknown;
  click_position?: unknown;
  clickPosition?: unknown;
  cursor_trail?: unknown;
  cursorTrail?: unknown;
  highlight_regions?: unknown;
  highlightRegions?: unknown;
  mask_regions?: unknown;
  maskRegions?: unknown;
  status?: string;
  success?: boolean;
  duration_ms?: number;
  durationMs?: number;
  total_duration_ms?: number;
  totalDurationMs?: number;
  progress?: number;
  final_url?: string;
  finalUrl?: string;
  error?: string;
  extracted_data_preview?: unknown;
  extractedDataPreview?: unknown;
  console_log_count?: number;
  consoleLogCount?: number;
  network_event_count?: number;
  networkEventCount?: number;
  timeline_artifact_id?: string;
  zoom_factor?: number;
  zoomFactor?: number;
  retry_attempt?: number;
  retryAttempt?: number;
  retry_max_attempts?: number;
  retryMaxAttempts?: number;
  retry_configured?: number;
  retryConfigured?: number;
  retry_delay_ms?: number;
  retryDelayMs?: number;
  retry_backoff_factor?: number;
  retryBackoffFactor?: number;
  retry_history?: unknown;
  retryHistory?: unknown;
  dom_snapshot_preview?: string;
  domSnapshotPreview?: string;
  dom_snapshot_artifact_id?: string;
  domSnapshotArtifactId?: string;
  assertion?: unknown;
  artifacts?: TimelineArtifact[];
}

interface ExecutionProps {
  execution: {
    id: string;
    workflowId: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    startedAt: Date;
    completedAt?: Date;
    screenshots: Screenshot[];
    timeline?: TimelineFrame[];
    logs: LogEntry[];
    currentStep?: string;
    progress: number;
    lastHeartbeat?: {
      step?: string;
      elapsedMs?: number;
      timestamp: Date;
    };
  };
}

const HEARTBEAT_WARN_SECONDS = 8;
const HEARTBEAT_STALL_SECONDS = 15;

function ExecutionViewer({ execution }: ExecutionProps) {
  const refreshTimeline = useExecutionStore((state) => state.refreshTimeline);
  const [activeTab, setActiveTab] = useState<'replay' | 'screenshots' | 'logs'>(
    execution.timeline && execution.timeline.length > 0 ? 'replay' : 'screenshots'
  );
  const [selectedScreenshot, setSelectedScreenshot] = useState<Screenshot | null>(null);
  const [, setHeartbeatTick] = useState(0);

  const heartbeatTimestamp = execution.lastHeartbeat?.timestamp?.valueOf();

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

  const replayFrames = useMemo<ReplayFrame[]>(() => {
    return (execution.timeline ?? []).map((frame: TimelineFrame, index: number) => {
      const screenshotData = frame?.screenshot ?? undefined;
      const screenshotUrl = resolveUrl(screenshotData?.url);
      const thumbnailUrl = resolveUrl(screenshotData?.thumbnail_url);

      const focused = frame?.focused_element ?? frame?.focusedElement;
      const focusedBoundingBox = toBoundingBox(focused?.bounding_box ?? focused?.boundingBox);
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

      return {
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
          focused || focusedBoundingBox
            ? {
                selector: typeof focused?.selector === 'string' ? focused.selector : undefined,
                boundingBox: focusedBoundingBox,
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
      } as ReplayFrame;
    });
  }, [execution.timeline]);

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
    if (hasTimeline) {
      if (activeTab === 'screenshots' && screenshots.length === 0) {
        setActiveTab('replay');
      }
    } else if (activeTab === 'replay') {
      setActiveTab(screenshots.length > 0 ? 'screenshots' : 'logs');
    }
  }, [hasTimeline, activeTab, screenshots.length]);

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
    <div className="h-full flex flex-col bg-flow-node">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          {getStatusIcon()}
          <div>
            <div className="text-sm font-medium text-white">
              Execution #{execution.id.slice(0, 8)}
            </div>
            <div className="text-xs text-gray-500">{execution.currentStep || 'Initializing...'}</div>
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
          <button className="toolbar-button p-1.5" title="Pause">
            <Pause size={14} />
          </button>
          <button className="toolbar-button p-1.5" title="Restart">
            <RotateCw size={14} />
          </button>
          <button className="toolbar-button p-1.5 text-red-400" title="Stop">
            <X size={14} />
          </button>
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
              : 'text-gray-400 hover:text-white'
          } ${hasTimeline ? '' : 'opacity-50 cursor-not-allowed'}`}
          onClick={() => hasTimeline && setActiveTab('replay')}
          disabled={!hasTimeline}
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

      <div className="flex-1 overflow-hidden flex flex-col">
        {activeTab === 'replay' ? (
          hasTimeline ? (
            <div className="flex-1 overflow-auto p-3">
              <ReplayPlayer frames={replayFrames} />
            </div>
          ) : (
            <div className="flex flex-1 items-center justify-center p-6 text-sm text-gray-400">
              Timeline data will appear once replay artifacts are captured for this execution.
            </div>
          )
        ) : activeTab === 'screenshots' ? (
          <>
            {selectedScreenshot && (
              <div className="flex-1 p-3 overflow-auto">
                <div className="screenshot-viewer">
                  <div className="bg-gray-800 px-3 py-2 flex items-center justify-between">
                    <span className="text-xs text-gray-400">{selectedScreenshot.stepName}</span>
                    <span className="text-xs text-gray-500">
                      {format(selectedScreenshot.timestamp, 'HH:mm:ss.SSS')}
                    </span>
                  </div>
                  <img
                    src={selectedScreenshot.url}
                    alt={selectedScreenshot.stepName}
                    className="w-full"
                  />
                </div>
              </div>
            )}

            <div className="border-t border-gray-800 p-2 overflow-x-auto">
              <div className="flex gap-2">
                {screenshots.map((screenshot) => (
                  <div
                    key={screenshot.id}
                    onClick={() => setSelectedScreenshot(screenshot)}
                    className={`flex-shrink-0 w-20 h-20 rounded overflow-hidden cursor-pointer border-2 transition-all ${
                      selectedScreenshot?.id === screenshot.id
                        ? 'border-flow-accent'
                        : 'border-gray-700 hover:border-gray-600'
                    }`}
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
