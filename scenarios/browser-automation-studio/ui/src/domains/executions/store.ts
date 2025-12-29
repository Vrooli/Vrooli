/**
 * Executions Domain Store
 *
 * Manages execution state including:
 * - Executions list and current execution
 * - Screenshots, timeline, and logs
 * - Execution lifecycle (start, stop, load)
 *
 * Note: Real-time updates are handled by useExecutionEvents hook,
 * which bridges the shared WebSocketContext to this store.
 */

import { fromJson } from '@bufbuild/protobuf';
import {
  ExecutionSchema,
  GetScreenshotsResponseSchema,
  type GetScreenshotsResponse as ProtoGetScreenshotsResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import {
  ExecutionTimelineSchema,
  type ExecutionTimeline as ProtoExecutionTimeline,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/container_pb';
import type {
  TimelineEntry as ProtoTimelineEntry,
  TimelineLog as ProtoTimelineLog,
  TimelineArtifact as ProtoTimelineArtifact,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import type { TelemetryArtifact as ProtoTelemetryArtifact } from '@vrooli/proto-types/browser-automation-studio/v1/domain/telemetry_pb';
import { ExecuteWorkflowResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { AssertionMode } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { create } from 'zustand';
import { getConfig } from '../../config';
import { logger } from '../../utils/logger';
import { parseProtoStrict } from '../../utils/proto';
import type { ReplayFrame } from '@/domains/exports/replay/ReplayPlayer';
import {
  mapArtifactType,
  mapExecutionStatus,
  mapProtoLogLevel,
  mapStepStatus,
  mapStepType,
  timestampToDate,
} from '@/domains/executions/utils/mappers';
import {
  createId,
  parseTimestamp,
  type Screenshot,
  type LogEntry,
} from './utils/eventProcessor';

const coerceDate = (value: unknown): Date | undefined => {
  if (value instanceof Date) return value;
  if (typeof value === 'string' || typeof value === 'number') {
    const d = new Date(value);
    return Number.isNaN(d.valueOf()) ? undefined : d;
  }
  return undefined;
};

const toNumber = (value?: number | bigint | null): number | undefined => {
  if (value == null) return undefined;
  return typeof value === 'bigint' ? Number(value) : value;
};

const mapAssertionMode = (mode?: AssertionMode | null): string | undefined => {
  if (mode == null) return undefined;
  return AssertionMode[mode] ?? String(mode);
};

export interface TimelineBoundingBox {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface TimelineRegion {
  selector?: string;
  boundingBox?: TimelineBoundingBox;
  padding?: number;
  color?: string;
  opacity?: number;
}

export interface TimelineRetryHistoryEntry {
  attempt?: number;
  success?: boolean;
  durationMs?: number;
  callDurationMs?: number;
  error?: string;
}

export interface TimelineAssertion {
  mode?: string;
  selector?: string;
  expected?: unknown;
  actual?: unknown;
  success?: boolean;
  message?: string;
  negated?: boolean;
  caseSensitive?: boolean;
}

export interface TimelineArtifact {
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

export interface TimelineLog {
  id?: string;
  level?: string;
  message?: string;
  step_name?: string;
  timestamp?: string;
}

export type TimelineFrame = ReplayFrame & {
  artifacts?: TimelineArtifact[];
  domSnapshotArtifactId?: string;
};

/** Page definition for multi-page workflow executions. */
export interface ExecutionPage {
  id: string;
  url: string;
  title: string;
  isInitial: boolean;
  openerId?: string;
  /** Whether this page has been closed during execution. */
  closed?: boolean;
}

export interface Execution {
  id: string;
  workflowId: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  startedAt: Date;
  completedAt?: Date;
  screenshots: Screenshot[];
  timeline: TimelineFrame[];
  logs: LogEntry[];
  currentStep?: string;
  progress: number;
  error?: string;
  lastHeartbeat?: {
    step?: string;
    elapsedMs?: number;
    timestamp: Date;
  };
  /** Pages for multi-page workflow executions. */
  pages?: ExecutionPage[];
  /** Currently active page ID during playback. */
  activePageId?: string;
}

/**
 * Artifact collection profile names.
 * These map to preset configurations on the backend.
 */
export type ArtifactProfile = 'full' | 'standard' | 'minimal' | 'debug' | 'none' | 'custom';

/**
 * Artifact collection configuration.
 * Controls what artifacts are collected during workflow execution.
 */
export interface ArtifactCollectionConfig {
  /** Preset profile: "full" (default), "standard", "minimal", "debug", "none", "custom" */
  profile?: ArtifactProfile;
  /** Capture screenshots at each step (only used when profile is "custom") */
  collectScreenshots?: boolean;
  /** Capture DOM snapshots (only used when profile is "custom") */
  collectDomSnapshots?: boolean;
  /** Capture browser console logs (only used when profile is "custom") */
  collectConsoleLogs?: boolean;
  /** Capture network events (only used when profile is "custom") */
  collectNetworkEvents?: boolean;
  /** Capture extracted data (only used when profile is "custom") */
  collectExtractedData?: boolean;
  /** Capture assertion results (only used when profile is "custom") */
  collectAssertions?: boolean;
  /** Capture cursor trail positions (only used when profile is "custom") */
  collectCursorTrails?: boolean;
  /** Emit real-time telemetry events (only used when profile is "custom") */
  collectTelemetry?: boolean;
}

/**
 * Options for starting a workflow execution.
 */
export interface StartExecutionOptions {
  /** Artifact collection configuration */
  artifactConfig?: ArtifactCollectionConfig;
  /** Force native video capture for this execution */
  requiresVideo?: boolean;
  /** Function to save the workflow before execution */
  saveWorkflowFn?: () => Promise<void>;
}

/** Profile descriptions for UI display */
export const ARTIFACT_PROFILE_DESCRIPTIONS: Record<ArtifactProfile, { label: string; description: string }> = {
  full: { label: 'Full', description: 'Collect all artifacts (screenshots, DOM, console, network, etc.)' },
  standard: { label: 'Standard', description: 'Screenshots, console logs, extracted data, and assertions' },
  minimal: { label: 'Minimal', description: 'Screenshots and assertions only (fastest)' },
  debug: { label: 'Debug', description: 'All artifacts with larger size limits for troubleshooting' },
  none: { label: 'None', description: 'No artifacts collected (execution status only)' },
  custom: { label: 'Custom', description: 'Customize individual artifact settings' },
};

interface ExecutionStore {
  executions: Execution[];
  currentExecution: Execution | null;
  viewerWorkflowId: string | null;
  /** Current artifact collection profile (persisted in localStorage) */
  artifactProfile: ArtifactProfile;

  openViewer: (workflowId: string) => void;
  closeViewer: () => void;
  startExecution: (workflowId: string, options?: StartExecutionOptions) => Promise<string>;
  setArtifactProfile: (profile: ArtifactProfile) => void;
  stopExecution: (executionId: string) => Promise<void>;
  loadExecutions: (workflowId?: string) => Promise<void>;
  loadExecution: (executionId: string) => Promise<void>;
  refreshTimeline: (executionId: string) => Promise<void>;
  addScreenshot: (screenshot: Screenshot) => void;
  addLog: (log: LogEntry) => void;
  updateExecutionStatus: (status: Execution['status'], error?: string) => void;
  updateProgress: (progress: number, currentStep?: string) => void;
  recordHeartbeat: (step?: string, elapsedMs?: number) => void;
  clearCurrentExecution: () => void;
}

const parseExecuteWorkflowResponse = (raw: unknown) =>
  parseProtoStrict(ExecuteWorkflowResponseSchema, raw) as ReturnType<typeof fromJson<typeof ExecuteWorkflowResponseSchema>>;

const parseExecutionProto = (raw: unknown): Execution => {
  const proto = parseProtoStrict(ExecutionSchema, raw) as ReturnType<typeof fromJson<typeof ExecutionSchema>>;
  const startedAt = timestampToDate(proto.startedAt) ?? new Date();
  const completedAt = timestampToDate(proto.completedAt);
  // Note: proto field is now lastHeartbeatAt (not lastHeartbeat)
  const lastHeartbeatAt = proto.lastHeartbeatAt ? timestampToDate(proto.lastHeartbeatAt) : undefined;

  return {
    id: proto.executionId || '',
    workflowId: proto.workflowId || '',
    status: mapExecutionStatus(proto.status),
    startedAt,
    completedAt: completedAt || undefined,
    screenshots: [],
    timeline: [],
    logs: [],
    currentStep: proto.currentStep || undefined,
    progress: typeof proto.progress === 'number' && proto.progress > 0 ? proto.progress : 0,
    error: proto.error ?? undefined,
    lastHeartbeat: lastHeartbeatAt
      ? {
          timestamp: lastHeartbeatAt,
        }
      : undefined,
  };
};

const mapBoundingBoxFromProto = (bbox?: { x?: number; y?: number; width?: number; height?: number } | null) => {
  if (!bbox) return undefined;
  const { x, y, width, height } = bbox;
  if ([x, y, width, height].every((v) => v == null)) return undefined;
  return { x, y, width, height };
};

/**
 * Map a TimelineArtifact from proto to our internal format.
 */
const mapTimelineArtifactFromProto = (artifact?: ProtoTimelineArtifact): TimelineArtifact | undefined => {
  if (!artifact) return undefined;
  const stepIndex = artifact.stepIndex ?? undefined;
  // Convert payload map to Record<string, unknown>
  const payload: Record<string, unknown> = {};
  if (artifact.payload) {
    for (const [key, value] of Object.entries(artifact.payload)) {
      payload[key] = value;
    }
  }
  return {
    id: artifact.id,
    type: mapArtifactType(artifact.type) ?? 'unknown',
    label: artifact.label || undefined,
    storage_url: artifact.storageUrl || undefined,
    thumbnail_url: artifact.thumbnailUrl || undefined,
    content_type: artifact.contentType || undefined,
    size_bytes: toNumber(artifact.sizeBytes),
    step_index: stepIndex,
    payload: Object.keys(payload).length > 0 ? payload : undefined,
  };
};

const mapTelemetryArtifactFromProto = (
  type: string,
  artifact?: ProtoTelemetryArtifact | null,
): TimelineArtifact | undefined => {
  if (!artifact) return undefined;
  const payload: Record<string, unknown> = {};
  if (artifact.path) {
    payload.path = artifact.path;
  }
  return {
    id: artifact.artifactId || createId(),
    type,
    label: type,
    storage_url: artifact.storageUrl || undefined,
    content_type: artifact.contentType || undefined,
    size_bytes: toNumber(artifact.sizeBytes),
    payload: Object.keys(payload).length > 0 ? payload : undefined,
  };
};

/**
 * Map a TimelineEntry from proto to our TimelineFrame format.
 * This is the new unified format - TimelineFrame is now just a wrapper.
 *
 * Exported for targeted unit testing to ensure API↔UI contract stays aligned.
 */
export const mapTimelineEntryToFrame = (entry: ProtoTimelineEntry): TimelineFrame => {
  const context = entry.context;
  const aggregates = entry.aggregates;
  const telemetry = entry.telemetry;

  // Map artifacts from aggregates
  const artifacts = aggregates?.artifacts?.map((a) => mapTimelineArtifactFromProto(a)!).filter(Boolean) ?? [];
  const telemetryArtifacts = [
    mapTelemetryArtifactFromProto('dom_snapshot', telemetry?.domSnapshot),
    mapTelemetryArtifactFromProto('console', telemetry?.consoleLogArtifact),
    mapTelemetryArtifactFromProto('network', telemetry?.networkEventArtifact),
  ].filter(Boolean) as TimelineArtifact[];

  // Map screenshot from telemetry
  const screenshot = telemetry?.screenshot
    ? {
        artifactId: telemetry.screenshot.artifactId || createId(),
        url: telemetry.screenshot.url,
        thumbnailUrl: telemetry.screenshot.thumbnailUrl,
        width: toNumber(telemetry.screenshot.width),
        height: toNumber(telemetry.screenshot.height),
        contentType: telemetry.screenshot.contentType || undefined,
        sizeBytes: toNumber(telemetry.screenshot.sizeBytes),
      }
    : undefined;

  // Map focused element from aggregates
  const focusedElement = aggregates?.focusedElement
    ? {
        selector: aggregates.focusedElement.selector || undefined,
        boundingBox: mapBoundingBoxFromProto(aggregates.focusedElement.boundingBox) ?? undefined,
      }
    : null;

  // Map element bounding box from telemetry
  const elementBoundingBox = mapBoundingBoxFromProto(telemetry?.elementBoundingBox) ?? null;

  // Map assertion from context
  const assertion = context?.assertion
    ? {
        mode: mapAssertionMode(context.assertion.mode),
        selector: context.assertion.selector,
        expected: context.assertion.expected,
        actual: context.assertion.actual,
        success: context.assertion.success,
        message: context.assertion.message || undefined,
        negated: context.assertion.negated,
        caseSensitive: context.assertion.caseSensitive,
      }
    : undefined;

  // Map retry from context
  const retryStatus = context?.retryStatus;

  return {
    id: entry.id || `entry-${entry.stepIndex ?? 0}`,
    stepIndex: entry.stepIndex ?? 0,
    nodeId: entry.nodeId || undefined,
    stepType: entry.action?.type ? mapStepType(entry.action.type) : undefined,
    status: aggregates ? mapStepStatus(aggregates.status) : undefined,
    success: context?.success ?? false,
    durationMs: entry.durationMs ?? undefined,
    totalDurationMs: entry.totalDurationMs ?? undefined,
    progress: aggregates?.progress ?? undefined,
    finalUrl: aggregates?.finalUrl || undefined,
    error: context?.error || undefined,
    extractedDataPreview: aggregates?.extractedDataPreview,
    consoleLogCount: aggregates?.consoleLogCount ?? undefined,
    networkEventCount: aggregates?.networkEventCount ?? undefined,
    screenshot: screenshot ?? undefined,
    highlightRegions: [], // Not in new proto - would come from telemetry.highlightRegions if added
    maskRegions: [], // Not in new proto - would come from telemetry.maskRegions if added
    focusedElement,
    elementBoundingBox,
    clickPosition: telemetry?.cursorPosition ? { x: telemetry.cursorPosition.x, y: telemetry.cursorPosition.y } : null,
    cursorTrail: [], // Not in new proto - would come from telemetry.cursorTrail if added
    zoomFactor: undefined, // Not in new proto
    assertion,
    retryAttempt: retryStatus?.currentAttempt ?? undefined,
    retryMaxAttempts: retryStatus?.maxAttempts ?? undefined,
    retryConfigured: retryStatus?.configured ?? undefined,
    retryDelayMs: retryStatus?.delayMs ?? undefined,
    retryBackoffFactor: retryStatus?.backoffFactor ?? undefined,
    retryHistory: retryStatus?.history?.map((h) => ({
      attempt: h.attempt ?? undefined,
      success: h.success ?? undefined,
      durationMs: h.durationMs ?? undefined,
      callDurationMs: undefined, // Not in new proto
      error: h.error || undefined,
    })) ?? [],
    domSnapshotArtifactId: telemetry?.domSnapshot?.artifactId || undefined,
    domSnapshotHtml: undefined, // Would need to fetch from artifact
    artifacts: [...artifacts, ...telemetryArtifacts],
  };
};

// Backwards compatibility - old name
export const mapTimelineFrameFromProto = mapTimelineEntryToFrame;

const mapTimelineLogFromProto = (log: ProtoTimelineLog): LogEntry | null => {
  const baseMessage = typeof log.message === 'string' ? log.message.trim() : '';
  const stepName = typeof log.stepName === 'string' ? log.stepName : '';
  const composedMessage = stepName && baseMessage ? `${stepName}: ${baseMessage}` : baseMessage || stepName;
  if (!composedMessage) {
    return null;
  }

  const rawTimestamp = timestampToDate(log.timestamp) ?? coerceDate(log.timestamp ?? undefined);
  const id = log.id && log.id.trim().length > 0 ? log.id.trim() : `${stepName}|${composedMessage}|${rawTimestamp?.toISOString() ?? ''}`;

  return {
    id,
    level: mapProtoLogLevel(log.level),
    message: composedMessage,
    timestamp: rawTimestamp ?? new Date(),
  };
};

const mapScreenshotsFromProto = (raw: unknown): Screenshot[] => {
  const proto = parseProtoStrict(GetScreenshotsResponseSchema, raw) as ProtoGetScreenshotsResponse;
  if (!proto.screenshots || proto.screenshots.length === 0) {
    return [];
  }
  return proto.screenshots
    .map((shot) => {
      const ts = timestampToDate(shot.timestamp) ?? coerceDate(shot.timestamp) ?? parseTimestamp(shot.timestamp as any);
      const url = shot.screenshot?.url || shot.screenshot?.thumbnailUrl || '';
      if (!url) {
        return null;
      }
      return {
        id: shot.screenshot?.artifactId || createId(),
        timestamp: ts,
        url,
        stepName: shot.stepLabel || `Step ${shot.stepIndex + 1}`,
      } satisfies Screenshot;
    })
    .filter((screenshot): screenshot is Screenshot => screenshot !== null);
};

// Load artifact profile from localStorage
const loadArtifactProfile = (): ArtifactProfile => {
  try {
    const stored = localStorage.getItem('bas_artifact_profile');
    if (stored && ['full', 'standard', 'minimal', 'debug', 'none', 'custom'].includes(stored)) {
      return stored as ArtifactProfile;
    }
  } catch {
    // Ignore localStorage errors
  }
  return 'full'; // Default to full collection
};

const resolveRequiresVideoPreference = (): boolean => {
  if (typeof window === 'undefined') {
    return false;
  }
  const keys = [
    'browserAutomation.settings.replay.exportRenderSource',
    'browserAutomation.replayRenderSource',
  ];
  for (const key of keys) {
    try {
      const stored = window.localStorage.getItem(key);
      if (stored === 'recorded_video' || stored === 'auto') {
        return true;
      }
      if (stored === 'replay_frames') {
        return false;
      }
    } catch {
      // Ignore localStorage errors
    }
  }
  return false;
};

export const useExecutionStore = create<ExecutionStore>((set, get) => ({
  executions: [],
  currentExecution: null,
  viewerWorkflowId: null,
  artifactProfile: loadArtifactProfile(),

  openViewer: (workflowId: string) => {
    const state = get();
    if (state.currentExecution && state.currentExecution.workflowId !== workflowId) {
      set({ currentExecution: null });
    }
    set({ viewerWorkflowId: workflowId });
  },

  closeViewer: () => {
    set({ currentExecution: null, viewerWorkflowId: null });
  },

  setArtifactProfile: (profile: ArtifactProfile) => {
    try {
      localStorage.setItem('bas_artifact_profile', profile);
    } catch {
      // Ignore localStorage errors
    }
    set({ artifactProfile: profile });
  },

  startExecution: async (workflowId: string, options?: StartExecutionOptions) => {
    try {
      // Save workflow first if save function provided (to ensure latest changes are used)
      if (options?.saveWorkflowFn) {
        await options.saveWorkflowFn();
      }

      set({ viewerWorkflowId: workflowId });
      const state = get();

      const config = await getConfig();

      // Build artifact config for the request
      // Use provided config, fall back to store's profile, or default to 'full'
      const artifactConfig = options?.artifactConfig ?? { profile: state.artifactProfile };
      const requiresVideo = options?.requiresVideo ?? resolveRequiresVideoPreference();

      const executeURL = new URL(`${config.API_URL}/workflows/${workflowId}/execute`);
      if (requiresVideo) {
        executeURL.searchParams.set('requires_video', 'true');
      }

      const response = await fetch(executeURL.toString(), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          wait_for_completion: false,
          parameters: {
            artifact_config: {
              profile: artifactConfig.profile ?? 'full',
              ...(artifactConfig.profile === 'custom' && {
                collect_screenshots: artifactConfig.collectScreenshots,
                collect_dom_snapshots: artifactConfig.collectDomSnapshots,
                collect_console_logs: artifactConfig.collectConsoleLogs,
                collect_network_events: artifactConfig.collectNetworkEvents,
                collect_extracted_data: artifactConfig.collectExtractedData,
                collect_assertions: artifactConfig.collectAssertions,
                collect_cursor_trails: artifactConfig.collectCursorTrails,
                collect_telemetry: artifactConfig.collectTelemetry,
              }),
            },
          },
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to start execution: ${response.status}`);
      }

      const data = await response.json();
      const protoPayload = parseExecuteWorkflowResponse(data);

      const execution: Execution = {
        id: protoPayload.executionId || '',
        workflowId,
        status: mapExecutionStatus(protoPayload.status),
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
        lastHeartbeat: undefined,
      };

      set({ currentExecution: execution });
      void get().refreshTimeline(execution.id);
      return execution.id;
    } catch (error) {
      logger.error('Failed to start execution', { component: 'ExecutionStore', action: 'startExecution', workflowId }, error);
      throw error;
    }
  },

  stopExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}/stop`, {
        method: 'POST',
      });

      if (!response.ok) {
        throw new Error(`Failed to stop execution: ${response.status}`);
      }

      set((state) => ({
        currentExecution:
          state.currentExecution?.id === executionId
            ? {
                ...state.currentExecution,
                status: 'cancelled' as const,
                error: 'Execution cancelled by user',
                currentStep: 'Stopped by user',
                completedAt: new Date(),
              }
            : state.currentExecution,
      }));
    } catch (error) {
      logger.error('Failed to stop execution', { component: 'ExecutionStore', action: 'stopExecution', executionId }, error);
      throw error;
    }
  },

  loadExecutions: async (workflowId?: string) => {
    try {
      const config = await getConfig();
      const url = workflowId
        ? `${config.API_URL}/executions?workflow_id=${encodeURIComponent(workflowId)}`
        : `${config.API_URL}/executions`;
      const response = await fetch(url);

      if (response.status === 404) {
        set({ executions: [] });
        return;
      }

      if (!response.ok) {
        throw new Error(`Failed to load executions: ${response.status}`);
      }

      const data = await response.json();
      const executions = Array.isArray(data?.executions) ? data.executions : [];
      const normalizedExecutions = executions
        .map((item: unknown) => {
          try {
            return parseExecutionProto(item);
          } catch (err) {
            logger.error('Failed to parse execution proto', { component: 'ExecutionStore', action: 'loadExecutions' }, err);
            return null;
          }
        })
        .filter((entry: Execution | null): entry is Execution => entry !== null);
      set({ executions: normalizedExecutions });
    } catch (error) {
      logger.error('Failed to load executions', { component: 'ExecutionStore', action: 'loadExecutions', workflowId }, error);
    }
  },

  loadExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}`);

      if (!response.ok) {
        throw new Error(`Failed to load execution: ${response.status}`);
      }

      const data = await response.json();
      const execution = parseExecutionProto(data);
      let screenshots: Screenshot[] = [];
      try {
        const shotsResponse = await fetch(`${config.API_URL}/executions/${executionId}/screenshots`);
        if (shotsResponse.ok) {
          const shotsJson = await shotsResponse.json();
          screenshots = mapScreenshotsFromProto(shotsJson);
        }
      } catch (shotsErr) {
        logger.error('Failed to load execution screenshots', { component: 'ExecutionStore', action: 'loadExecution', executionId }, shotsErr);
      }

      set({ currentExecution: { ...execution, screenshots }, viewerWorkflowId: execution.workflowId });
      void get().refreshTimeline(executionId);
    } catch (error) {
      logger.error('Failed to load execution', { component: 'ExecutionStore', action: 'loadExecution', executionId }, error);
    }
  },

  refreshTimeline: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}/timeline`);

      if (!response.ok) {
        throw new Error(`Failed to load execution timeline: ${response.status}`);
      }

      const data = await response.json();

      // Strict proto parsing; bubble errors so contract drift is visible.
      // Preserve raw error for UI since the proto contract omits it.
      const { error: rawError, ...timelinePayload } = (data && typeof data === 'object'
        ? (data as Record<string, unknown>)
        : {}) as Record<string, unknown>;

      const protoTimeline = parseProtoStrict(
        ExecutionTimelineSchema,
        timelinePayload
      ) as ProtoExecutionTimeline;

      const frames = (protoTimeline.entries ?? []).map((entry) => mapTimelineFrameFromProto(entry));

      const normalizedLogs = (protoTimeline.logs ?? [])
        .map((log) => mapTimelineLogFromProto(log))
        .filter((entry): entry is LogEntry => Boolean(entry));

      // Extract status with fallback to raw data
      const mappedStatus = mapExecutionStatus(protoTimeline.status);

      // Extract progress with fallback to raw data
      const progressValue = typeof protoTimeline.progress === 'number'
        ? protoTimeline.progress
        : undefined;

      // Extract completedAt with fallback to raw data
      const completedAt = timestampToDate(protoTimeline.completedAt);

      // Extract error from raw data (ExecutionTimeline proto doesn't have error field)
      const errorValue = rawError ? String(rawError) : undefined;

      set((state) => {
        if (!state.currentExecution || state.currentExecution.id !== executionId) {
          return {};
        }

        const updated: Execution = {
          ...state.currentExecution,
          timeline: frames,
          logs: [...(state.currentExecution.logs ?? [])],
        };

        if (normalizedLogs.length > 0) {
          const merged = new Map<string, LogEntry>();
          for (const existingLog of updated.logs) {
            merged.set(existingLog.id, existingLog);
          }
          for (const newLog of normalizedLogs) {
            merged.set(newLog.id, newLog);
          }
          updated.logs = Array.from(merged.values()).sort(
            (a, b) => a.timestamp.getTime() - b.timestamp.getTime(),
          );
        }

        if (mappedStatus && updated.status !== mappedStatus) {
          updated.status = mappedStatus;
        }

        if (progressValue != null) {
          updated.progress = progressValue;
        }

        if (completedAt && (updated.status === 'completed' || updated.status === 'failed')) {
          updated.completedAt = completedAt;
        }

        if (errorValue && updated.status === 'failed') {
          updated.error = errorValue;
        }

        if (frames.length > 0) {
          const lastFrame = frames[frames.length - 1];
          const label = lastFrame?.nodeId ?? lastFrame?.stepType;
          if (typeof label === 'string' && label.trim().length > 0) {
            updated.currentStep = label.trim();
          }
        }

        return { currentExecution: updated };
      });
    } catch (error) {
      logger.error('Failed to load execution timeline', { component: 'ExecutionStore', action: 'refreshTimeline', executionId }, error);
    }
  },

  addScreenshot: (screenshot: Screenshot) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            screenshots: [...state.currentExecution.screenshots, screenshot],
          }
        : state.currentExecution,
    }));
  },

  addLog: (log: LogEntry) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            logs: [...state.currentExecution.logs, log],
          }
        : state.currentExecution,
    }));
  },

  updateExecutionStatus: (status: Execution['status'], error?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            status,
            error,
            completedAt:
              status === 'completed' || status === 'failed' || status === 'cancelled'
                ? new Date()
                : state.currentExecution.completedAt,
          }
        : state.currentExecution,
    }));
  },

  updateProgress: (progress: number, currentStep?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            progress,
            currentStep: currentStep ?? state.currentExecution.currentStep,
            lastHeartbeat: {
              step: currentStep ?? state.currentExecution.lastHeartbeat?.step,
              elapsedMs: 0,
              timestamp: new Date(),
            },
          }
        : state.currentExecution,
    }));
  },
  recordHeartbeat: (step?: string, elapsedMs?: number) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            lastHeartbeat: {
              step,
              elapsedMs,
              timestamp: new Date(),
            },
          }
        : state.currentExecution,
    }));
  },

  clearCurrentExecution: () => {
    set({ currentExecution: null });
  },
}));

// Export types for consumers
export type { ExecutionStore, Screenshot, LogEntry };
