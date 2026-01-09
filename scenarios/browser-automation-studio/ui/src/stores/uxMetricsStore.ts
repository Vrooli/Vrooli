/**
 * UX Metrics Store
 *
 * Manages UX metrics data for workflow executions, including:
 * - Friction scores and signals
 * - Cursor path analysis
 * - Step-level metrics
 * - Workflow-level aggregations
 *
 * UX metrics are gated to Pro tier and above.
 */
import { create } from 'zustand';
import { getApiBase } from '../config';
import { logger } from '../utils/logger';

// =============================================================================
// Types
// =============================================================================

export interface Point {
  x: number;
  y: number;
}

export interface TimedPoint extends Point {
  timestamp: string;
}

export interface CursorPath {
  stepIndex: number;
  points: TimedPoint[];
  totalDistancePx: number;
  durationMs: number;
  directDistancePx: number;
  directness: number;
  zigzagScore: number;
  averageSpeedPxMs: number;
  maxSpeedPxMs: number;
  hesitationCount: number;
}

export type FrictionType =
  | 'excessive_time'
  | 'zigzag_path'
  | 'multiple_retries'
  | 'rapid_clicks'
  | 'long_hesitation'
  | 'back_navigation'
  | 'element_miss';

export type Severity = 'low' | 'medium' | 'high';

export interface FrictionSignal {
  type: FrictionType;
  stepIndex: number;
  severity: Severity;
  score: number;
  description: string;
  evidence?: Record<string, unknown>;
}

export interface StepMetrics {
  stepIndex: number;
  nodeId: string;
  stepType: string;
  timeToActionMs: number;
  actionDurationMs: number;
  totalDurationMs: number;
  cursorPath?: CursorPath;
  retryCount: number;
  frictionSignals: FrictionSignal[];
  frictionScore: number;
}

export interface MetricsSummary {
  highFrictionSteps: number[];
  slowestSteps: number[];
  topFrictionTypes: string[];
  recommendedActions: string[];
}

export interface ExecutionMetrics {
  executionId: string;
  workflowId: string;
  computedAt: string;
  totalDurationMs: number;
  stepCount: number;
  successfulSteps: number;
  failedSteps: number;
  totalRetries: number;
  avgStepDurationMs: number;
  totalCursorDistancePx: number;
  overallFrictionScore: number;
  frictionSignals: FrictionSignal[];
  stepMetrics: StepMetrics[];
  summary?: MetricsSummary;
}

export interface WorkflowMetricsAggregate {
  workflowId: string;
  executionCount: number;
  avgFrictionScore: number;
  avgDurationMs: number;
  trendDirection: 'improving' | 'degrading' | 'stable';
  highFrictionStepFrequency: Record<number, number>;
}

// =============================================================================
// Store State & Actions
// =============================================================================

interface UXMetricsState {
  // Per-execution metrics cache
  executionMetrics: Map<string, ExecutionMetrics>;

  // Per-workflow aggregate cache
  workflowAggregates: Map<string, WorkflowMetricsAggregate>;

  // Real-time step metrics (updated via WebSocket)
  currentStepMetrics: StepMetrics | null;

  // Loading states
  isLoading: boolean;
  isComputing: boolean;
  error: string | null;

  // Actions
  fetchExecutionMetrics: (executionId: string) => Promise<ExecutionMetrics | null>;
  computeMetrics: (executionId: string) => Promise<ExecutionMetrics | null>;
  fetchWorkflowAggregate: (workflowId: string, limit?: number) => Promise<WorkflowMetricsAggregate | null>;
  handleMetricsUpdate: (payload: { stepIndex: number; frictionScore: number; signals?: FrictionSignal[] }) => void;
  handleFrictionAlert: (payload: { stepIndex: number; signal: FrictionSignal }) => void;
  clearMetrics: (executionId: string) => void;
  clearWorkflowAggregate: (workflowId: string) => void;
  reset: () => void;
}

// =============================================================================
// API Helpers
// =============================================================================

interface ApiResponse<T> {
  data?: T;
  error?: string;
  status: number;
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
  try {
    const apiBase = getApiBase();
    // Ensure proper URL construction with trailing slash handling
    const normalizedBase = apiBase.endsWith('/') ? apiBase : `${apiBase}/`;
    const normalizedEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
    const url = `${normalizedBase}${normalizedEndpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (response.status === 403) {
      return { status: 403, error: 'UX metrics requires Pro plan or higher' };
    }

    if (response.status === 404) {
      return { status: 404, error: 'Not found' };
    }

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      return { status: response.status, error: errorData.error || 'Request failed' };
    }

    const data = await response.json();
    return { status: response.status, data };
  } catch (err) {
    logger.error('UX metrics API error:', { error: err });
    return { status: 0, error: (err as Error).message };
  }
}

// =============================================================================
// JSON Mapping
// =============================================================================

function mapExecutionMetrics(raw: Record<string, unknown>): ExecutionMetrics {
  return {
    executionId: String(raw.execution_id ?? ''),
    workflowId: String(raw.workflow_id ?? ''),
    computedAt: String(raw.computed_at ?? ''),
    totalDurationMs: Number(raw.total_duration_ms ?? 0),
    stepCount: Number(raw.step_count ?? 0),
    successfulSteps: Number(raw.successful_steps ?? 0),
    failedSteps: Number(raw.failed_steps ?? 0),
    totalRetries: Number(raw.total_retries ?? 0),
    avgStepDurationMs: Number(raw.avg_step_duration_ms ?? 0),
    totalCursorDistancePx: Number(raw.total_cursor_distance_px ?? 0),
    overallFrictionScore: Number(raw.overall_friction_score ?? 0),
    frictionSignals: mapFrictionSignals(raw.friction_signals),
    stepMetrics: mapStepMetricsList(raw.step_metrics),
    summary: raw.summary ? mapMetricsSummary(raw.summary as Record<string, unknown>) : undefined,
  };
}

function mapFrictionSignals(raw: unknown): FrictionSignal[] {
  if (!Array.isArray(raw)) return [];
  return raw.map((s: Record<string, unknown>) => ({
    type: String(s.type ?? 'excessive_time') as FrictionType,
    stepIndex: Number(s.step_index ?? 0),
    severity: String(s.severity ?? 'low') as Severity,
    score: Number(s.score ?? 0),
    description: String(s.description ?? ''),
    evidence: s.evidence as Record<string, unknown> | undefined,
  }));
}

function mapStepMetricsList(raw: unknown): StepMetrics[] {
  if (!Array.isArray(raw)) return [];
  return raw.map((sm: Record<string, unknown>) => ({
    stepIndex: Number(sm.step_index ?? 0),
    nodeId: String(sm.node_id ?? ''),
    stepType: String(sm.step_type ?? ''),
    timeToActionMs: Number(sm.time_to_action_ms ?? 0),
    actionDurationMs: Number(sm.action_duration_ms ?? 0),
    totalDurationMs: Number(sm.total_duration_ms ?? 0),
    cursorPath: sm.cursor_path ? mapCursorPath(sm.cursor_path as Record<string, unknown>) : undefined,
    retryCount: Number(sm.retry_count ?? 0),
    frictionSignals: mapFrictionSignals(sm.friction_signals),
    frictionScore: Number(sm.friction_score ?? 0),
  }));
}

function mapCursorPath(raw: Record<string, unknown>): CursorPath {
  return {
    stepIndex: Number(raw.step_index ?? 0),
    points: mapTimedPoints(raw.points),
    totalDistancePx: Number(raw.total_distance_px ?? 0),
    durationMs: Number(raw.duration_ms ?? 0),
    directDistancePx: Number(raw.direct_distance_px ?? 0),
    directness: Number(raw.directness ?? 1),
    zigzagScore: Number(raw.zigzag_score ?? 0),
    averageSpeedPxMs: Number(raw.average_speed_px_ms ?? 0),
    maxSpeedPxMs: Number(raw.max_speed_px_ms ?? 0),
    hesitationCount: Number(raw.hesitation_count ?? 0),
  };
}

function mapTimedPoints(raw: unknown): TimedPoint[] {
  if (!Array.isArray(raw)) return [];
  return raw.map((p: Record<string, unknown>) => ({
    x: Number(p.x ?? 0),
    y: Number(p.y ?? 0),
    timestamp: String(p.timestamp ?? ''),
  }));
}

function mapMetricsSummary(raw: Record<string, unknown>): MetricsSummary {
  return {
    highFrictionSteps: Array.isArray(raw.high_friction_steps) ? raw.high_friction_steps.map(Number) : [],
    slowestSteps: Array.isArray(raw.slowest_steps) ? raw.slowest_steps.map(Number) : [],
    topFrictionTypes: Array.isArray(raw.top_friction_types) ? raw.top_friction_types.map(String) : [],
    recommendedActions: Array.isArray(raw.recommended_actions) ? raw.recommended_actions.map(String) : [],
  };
}

function mapWorkflowAggregate(raw: Record<string, unknown>): WorkflowMetricsAggregate {
  const freqMap: Record<number, number> = {};
  if (raw.high_friction_step_frequency && typeof raw.high_friction_step_frequency === 'object') {
    for (const [k, v] of Object.entries(raw.high_friction_step_frequency as Record<string, unknown>)) {
      freqMap[Number(k)] = Number(v);
    }
  }
  return {
    workflowId: String(raw.workflow_id ?? ''),
    executionCount: Number(raw.execution_count ?? 0),
    avgFrictionScore: Number(raw.avg_friction_score ?? 0),
    avgDurationMs: Number(raw.avg_duration_ms ?? 0),
    trendDirection: String(raw.trend_direction ?? 'stable') as 'improving' | 'degrading' | 'stable',
    highFrictionStepFrequency: freqMap,
  };
}

// =============================================================================
// Store Implementation
// =============================================================================

export const useUXMetricsStore = create<UXMetricsState>((set) => ({
  executionMetrics: new Map(),
  workflowAggregates: new Map(),
  currentStepMetrics: null,
  isLoading: false,
  isComputing: false,
  error: null,

  fetchExecutionMetrics: async (executionId: string) => {
    set({ isLoading: true, error: null });

    const result = await fetchApi<Record<string, unknown>>(
      `executions/${executionId}/ux-metrics`
    );

    if (result.status === 403) {
      set({ error: result.error, isLoading: false });
      return null;
    }

    if (result.status === 404) {
      // Metrics not yet computed - not an error
      set({ isLoading: false });
      return null;
    }

    if (result.error || !result.data) {
      set({ error: result.error ?? 'Failed to fetch metrics', isLoading: false });
      return null;
    }

    const metrics = mapExecutionMetrics(result.data);

    set((state) => ({
      executionMetrics: new Map(state.executionMetrics).set(executionId, metrics),
      isLoading: false,
    }));

    return metrics;
  },

  computeMetrics: async (executionId: string) => {
    set({ isComputing: true, error: null });

    const result = await fetchApi<Record<string, unknown>>(
      `executions/${executionId}/ux-metrics/compute`,
      { method: 'POST' }
    );

    if (result.status === 403) {
      set({ error: result.error, isComputing: false });
      return null;
    }

    if (result.error || !result.data) {
      set({ error: result.error ?? 'Failed to compute metrics', isComputing: false });
      return null;
    }

    const metrics = mapExecutionMetrics(result.data);

    set((state) => ({
      executionMetrics: new Map(state.executionMetrics).set(executionId, metrics),
      isComputing: false,
    }));

    return metrics;
  },

  fetchWorkflowAggregate: async (workflowId: string, limit = 10) => {
    set({ isLoading: true, error: null });

    const result = await fetchApi<Record<string, unknown>>(
      `workflows/${workflowId}/ux-metrics/aggregate?limit=${limit}`
    );

    if (result.status === 403) {
      set({ error: result.error, isLoading: false });
      return null;
    }

    if (result.error || !result.data) {
      set({ error: result.error ?? 'Failed to fetch workflow aggregate', isLoading: false });
      return null;
    }

    const aggregate = mapWorkflowAggregate(result.data);

    set((state) => ({
      workflowAggregates: new Map(state.workflowAggregates).set(workflowId, aggregate),
      isLoading: false,
    }));

    return aggregate;
  },

  handleMetricsUpdate: (payload) => {
    set({
      currentStepMetrics: {
        stepIndex: payload.stepIndex,
        nodeId: '',
        stepType: '',
        timeToActionMs: 0,
        actionDurationMs: 0,
        totalDurationMs: 0,
        retryCount: 0,
        frictionSignals: payload.signals ?? [],
        frictionScore: payload.frictionScore,
      },
    });
  },

  handleFrictionAlert: (payload) => {
    // Log friction alert for debugging/monitoring
    logger.warn('UX Friction Alert:', {
      stepIndex: payload.stepIndex,
      type: payload.signal.type,
      severity: payload.signal.severity,
      description: payload.signal.description,
    });
  },

  clearMetrics: (executionId: string) => {
    set((state) => {
      const newMap = new Map(state.executionMetrics);
      newMap.delete(executionId);
      return { executionMetrics: newMap };
    });
  },

  clearWorkflowAggregate: (workflowId: string) => {
    set((state) => {
      const newMap = new Map(state.workflowAggregates);
      newMap.delete(workflowId);
      return { workflowAggregates: newMap };
    });
  },

  reset: () => {
    set({
      executionMetrics: new Map(),
      workflowAggregates: new Map(),
      currentStepMetrics: null,
      isLoading: false,
      isComputing: false,
      error: null,
    });
  },
}));

// =============================================================================
// Selectors
// =============================================================================

export const selectExecutionMetrics = (executionId: string) => (state: UXMetricsState) =>
  state.executionMetrics.get(executionId);

export const selectWorkflowAggregate = (workflowId: string) => (state: UXMetricsState) =>
  state.workflowAggregates.get(workflowId);

export const selectIsProTierRequired = (state: UXMetricsState) =>
  state.error === 'UX metrics requires Pro plan or higher';

// =============================================================================
// Utilities
// =============================================================================

/**
 * Get severity color class for styling
 */
export function getSeverityColor(severity: Severity): string {
  switch (severity) {
    case 'high':
      return 'text-red-400';
    case 'medium':
      return 'text-yellow-400';
    case 'low':
      return 'text-blue-400';
    default:
      return 'text-slate-400';
  }
}

/**
 * Get severity background color class for styling
 */
export function getSeverityBgColor(severity: Severity): string {
  switch (severity) {
    case 'high':
      return 'bg-red-500/20';
    case 'medium':
      return 'bg-yellow-500/20';
    case 'low':
      return 'bg-blue-500/20';
    default:
      return 'bg-slate-500/20';
  }
}

/**
 * Get friction score color based on value (0-100)
 */
export function getFrictionScoreColor(score: number): string {
  if (score >= 70) return 'text-red-400';
  if (score >= 40) return 'text-yellow-400';
  return 'text-green-400';
}

/**
 * Get human-readable friction type label
 */
export function getFrictionTypeLabel(type: FrictionType): string {
  const labels: Record<FrictionType, string> = {
    excessive_time: 'Excessive Time',
    zigzag_path: 'Zigzag Path',
    multiple_retries: 'Multiple Retries',
    rapid_clicks: 'Rapid Clicks',
    long_hesitation: 'Long Hesitation',
    back_navigation: 'Back Navigation',
    element_miss: 'Element Miss',
  };
  return labels[type] ?? type;
}

/**
 * Format duration in milliseconds to human-readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}
