/**
 * Executions Connect-RPC adapter layer.
 *
 * Every executions JSON RPC goes through `executionsClient` (generated from
 * proto). UI call sites should import these adapters rather than calling
 * fetch directly — that's how we keep API/UI/CLI contracts aligned.
 *
 * Two endpoints intentionally stay on REST and are NOT exposed here:
 *   - POST /executions/{id}/export — multipart-shaped replay export.
 *   - POST /executions/{executionId}/frames — playwright-driver ops probe.
 */
import { create, toJson } from '@bufbuild/protobuf';
import { executionsClient } from '@/api/executions';
import {
  ListExecutionsRequestSchema,
  type ListExecutionsResponse,
  type ExecutionExportability as ProtoExecutionExportability,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import {
  ExecutionSchema,
  type Execution as ProtoExecution,
  type GetScreenshotsResponse as ProtoGetScreenshotsResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import type { ExecutionTimeline as ProtoExecutionTimeline } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/container_pb';
import type { ExecutionItem } from '@/shared/api/schemas';

/**
 * Shape returned by ListExecutions+include_exportability=true. Mirrors the
 * legacy REST shape so dashboardStore/store can keep their normalization
 * helpers unchanged.
 */
export interface ExecutionExportabilityItem {
  has_timeline: boolean;
  has_screenshots: boolean;
  has_recorded_video: boolean;
  is_exportable: boolean;
}

export const exportabilityToLegacy = (e: ProtoExecutionExportability): ExecutionExportabilityItem => ({
  has_timeline: Boolean(e.hasTimeline),
  has_screenshots: Boolean(e.hasScreenshots),
  has_recorded_video: Boolean(e.hasRecordedVideo),
  is_exportable: Boolean(e.isExportable),
});

/** Convert a typed Execution proto message into the JSON shape consumed by
 * the existing `parseExecutionProto` parser in the store. */
export const executionMsgToJson = (msg: ProtoExecution): Record<string, unknown> =>
  toJson(ExecutionSchema, msg) as Record<string, unknown>;

/** Convert a typed Execution proto message into the legacy `ExecutionItem`
 * shape consumed by dashboardStore.normalizeRecentExecution. */
export const executionMsgToItem = (msg: ProtoExecution): ExecutionItem =>
  executionMsgToJson(msg) as unknown as ExecutionItem;

interface ListOptions {
  limit?: number;
  workflowId?: string;
  projectId?: string;
  includeExportability?: boolean;
}

export const listExecutionsViaApi = async (
  opts: ListOptions = {}
): Promise<ListExecutionsResponse> => {
  const req = create(ListExecutionsRequestSchema, {});
  if (opts.workflowId) req.workflowId = opts.workflowId;
  if (opts.projectId) req.projectId = opts.projectId;
  if (typeof opts.limit === 'number') req.limit = opts.limit;
  if (opts.includeExportability) req.includeExportability = true;
  return executionsClient.listExecutions(req);
};

/** Legacy compatibility wrapper for callers that expect `ExecutionItem[]`. */
export const fetchExecutionsList = async (limit = 100): Promise<ExecutionItem[]> => {
  const resp = await listExecutionsViaApi({ limit });
  return resp.executions.map(executionMsgToItem);
};

export const getExecutionViaApi = async (executionId: string): Promise<ProtoExecution> => {
  const resp = await executionsClient.getExecution({ executionId });
  if (!resp.execution) throw new Error('execution missing from response');
  return resp.execution;
};

export const getExecutionTimelineViaApi = async (executionId: string): Promise<ProtoExecutionTimeline> =>
  executionsClient.getExecutionTimeline({ executionId });

export const stopExecutionViaApi = async (executionId: string): Promise<void> => {
  await executionsClient.stopExecution({ executionId });
};

export const resumeExecutionViaApi = async (
  executionId: string,
  resumeUrl?: string,
): Promise<ProtoExecution> => {
  const resp = await executionsClient.resumeExecution({
    executionId,
    resumeUrl: resumeUrl ?? '',
  });
  if (!resp.execution) throw new Error('execution missing from resume response');
  return resp.execution;
};

export const getExecutionScreenshotsViaApi = async (
  executionId: string,
): Promise<ProtoGetScreenshotsResponse> =>
  executionsClient.getExecutionScreenshots({ executionId });

export const getRecordedVideosViaApi = async (executionId: string) =>
  executionsClient.getExecutionRecordedVideos({ executionId });

export const getRecordedTracesViaApi = async (executionId: string) =>
  executionsClient.getExecutionRecordedTraces({ executionId });

export const getRecordedHarViaApi = async (executionId: string) =>
  executionsClient.getExecutionRecordedHar({ executionId });

export const scheduleSeedCleanupViaApi = async (
  executionId: string,
  cleanupToken: string,
  seedScenario?: string,
) =>
  executionsClient.scheduleExecutionSeedCleanup({
    executionId,
    cleanupToken,
    seedScenario: seedScenario ?? '',
  });
