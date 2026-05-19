import { workflowsClient } from '@/api/workflows';
import { fromJson } from '@bufbuild/protobuf';
import { WorkflowDefinitionV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { ChangeSource } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import {
  CreateWorkflowResponseSchema,
  DeleteWorkflowResponseSchema,
  GetWorkflowResponseSchema,
  ListWorkflowsResponseSchema,
  RestoreWorkflowVersionResponseSchema,
  UpdateWorkflowResponseSchema,
  ValidateWorkflowResponseSchema,
  WorkflowVersionListSchema,
  WorkflowVersionSchema,
  type CreateWorkflowResponse,
  type DeleteWorkflowResponse,
  type GetWorkflowResponse,
  type ListWorkflowsResponse,
  type RestoreWorkflowVersionResponse,
  type UpdateWorkflowResponse,
  type ValidateWorkflowResponse,
  type WorkflowSummary,
  type WorkflowVersion as ProtoWorkflowVersion,
  type WorkflowVersionList,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import type {
  ExecuteAdhocResponse,
  ExecuteWorkflowOptions,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import type { WorkflowDefinitionV2 } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import type { WorkflowItem } from '@/shared/api/schemas';

// ---------------------------------------------------------------------------
// Connect-RPC adapters
//
// Each function takes a plain object that mirrors the proto request fields and
// returns the Connect-Web typed response. Callers are encouraged to use the
// proto types directly rather than re-deriving JSON shapes.
// ---------------------------------------------------------------------------

export interface ListWorkflowsInput {
  projectId?: string;
  folderPath?: string;
  limit?: number;
  offset?: number;
}

export const listWorkflowsViaApi = async (
  input: ListWorkflowsInput = {},
): Promise<ListWorkflowsResponse> => workflowsClient.listWorkflows(input);

export const getWorkflowViaApi = async (
  workflowId: string,
  version?: number,
): Promise<GetWorkflowResponse> =>
  workflowsClient.getWorkflow({ workflowId, version });

export interface CreateWorkflowInput {
  projectId: string;
  name: string;
  folderPath?: string;
  flowDefinition?: WorkflowDefinitionV2;
  aiPrompt?: string;
}

export const createWorkflowViaApi = async (
  input: CreateWorkflowInput,
): Promise<CreateWorkflowResponse> => workflowsClient.createWorkflow(input);

export interface UpdateWorkflowInput {
  workflowId: string;
  name?: string;
  description?: string;
  folderPath?: string;
  tags?: string[];
  flowDefinition?: WorkflowDefinitionV2;
  changeDescription?: string;
  source?: number;
  expectedVersion?: number;
}

export const updateWorkflowViaApi = async (
  input: UpdateWorkflowInput,
): Promise<UpdateWorkflowResponse> => workflowsClient.updateWorkflow(input);

export const deleteWorkflowViaApi = async (
  workflowId: string,
): Promise<DeleteWorkflowResponse> =>
  workflowsClient.deleteWorkflow({ workflowId });

export interface ModifyWorkflowInput {
  workflowId: string;
  modificationPrompt: string;
  currentFlow: WorkflowDefinitionV2;
}

export const modifyWorkflowViaApi = async (
  input: ModifyWorkflowInput,
): Promise<UpdateWorkflowResponse> => workflowsClient.modifyWorkflow(input);

export interface ExecuteWorkflowInput {
  workflowId: string;
  workflowVersion?: number;
  waitForCompletion?: boolean;
  parameters?: unknown;
  options?: ExecuteWorkflowOptions;
}

export const executeWorkflowViaApi = async (input: ExecuteWorkflowInput) =>
  workflowsClient.executeWorkflow(input as Parameters<typeof workflowsClient.executeWorkflow>[0]);

export interface ExecuteAdhocWorkflowInput {
  flowDefinition: WorkflowDefinitionV2;
  waitForCompletion?: boolean;
  metadata?: { name?: string; description?: string };
  parameters?: unknown;
  options?: ExecuteWorkflowOptions;
}

export const executeAdhocWorkflowViaApi = async (
  input: ExecuteAdhocWorkflowInput,
): Promise<ExecuteAdhocResponse> =>
  workflowsClient.executeAdhocWorkflow(
    input as Parameters<typeof workflowsClient.executeAdhocWorkflow>[0],
  );

export const validateWorkflowViaApi = async (
  workflow: WorkflowDefinitionV2,
): Promise<ValidateWorkflowResponse> =>
  workflowsClient.validateWorkflow({ workflow });

export const validateResolvedWorkflowViaApi = async (
  workflow: WorkflowDefinitionV2,
): Promise<ValidateWorkflowResponse> =>
  workflowsClient.validateResolvedWorkflow({ workflow });

export const listWorkflowVersionsViaApi = async (
  workflowId: string,
): Promise<WorkflowVersionList> =>
  workflowsClient.listWorkflowVersions({ workflowId });

export const getWorkflowVersionViaApi = async (
  workflowId: string,
  version: number,
): Promise<ProtoWorkflowVersion> =>
  workflowsClient.getWorkflowVersion({ workflowId, version });

export const restoreWorkflowVersionViaApi = async (
  workflowId: string,
  version: number,
  changeDescription = '',
): Promise<RestoreWorkflowVersionResponse> =>
  workflowsClient.restoreWorkflowVersion({ workflowId, version, changeDescription });

// ---------------------------------------------------------------------------
// Legacy convenience wrappers retained while higher layers migrate to proto
// types directly. Both call the Connect-RPC client; no REST fetches remain.
// ---------------------------------------------------------------------------

const timestampToIso = (ts: { seconds?: bigint; nanos?: number } | undefined): string | undefined => {
  if (!ts) return undefined;
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : 0;
  const nanos = typeof ts.nanos === 'number' ? ts.nanos : 0;
  if (seconds === 0 && nanos === 0) return undefined;
  return new Date(seconds * 1000 + Math.floor(nanos / 1e6)).toISOString();
};

const summaryToItem = (summary: WorkflowSummary): WorkflowItem => ({
  id: summary.id,
  name: summary.name || undefined,
  project_id: summary.projectId || undefined,
  folder_path: summary.folderPath || undefined,
  updated_at: timestampToIso(summary.updatedAt),
});

/**
 * fetchWorkflowList preserves the legacy WorkflowItem[] surface used by
 * dashboard/global-search views, now backed by Connect-RPC.
 */
export const fetchWorkflowList = async (limit = 100): Promise<WorkflowItem[]> => {
  const resp = await listWorkflowsViaApi({ limit });
  return (resp.workflows ?? []).map(summaryToItem).filter((w) => w.id !== '');
};

/** Backwards-compatible helper used by execution and dashboard view code. */
export const fetchWorkflowProjectId = async (workflowId: string): Promise<string> => {
  const resp = await getWorkflowViaApi(workflowId);
  const projectId = resp.workflow?.projectId ?? '';
  if (!projectId) {
    throw new Error('Workflow has no associated project');
  }
  return projectId;
};

// Re-export schemas/types frequently consumed by the store so callers can
// import everything from one location.
export {
  CreateWorkflowResponseSchema,
  DeleteWorkflowResponseSchema,
  GetWorkflowResponseSchema,
  ListWorkflowsResponseSchema,
  RestoreWorkflowVersionResponseSchema,
  UpdateWorkflowResponseSchema,
  ValidateWorkflowResponseSchema,
  WorkflowVersionListSchema,
  WorkflowVersionSchema,
};

export type {
  ExecuteWorkflowOptions,
  WorkflowDefinitionV2,
};

export { ChangeSource };

const CHANGE_SOURCE_MAP: Record<string, ChangeSource> = {
  manual: ChangeSource.MANUAL,
  autosave: ChangeSource.AUTOSAVE,
  import: ChangeSource.IMPORT,
  ai_generated: ChangeSource.AI_GENERATED,
  'ai-generated': ChangeSource.AI_GENERATED,
  recording: ChangeSource.RECORDING,
};

/**
 * parseChangeSource maps a free-form string ("manual", "autosave", ...) to
 * the ChangeSource proto enum. Unknown values map to UNSPECIFIED.
 */
export const parseChangeSource = (source: string | undefined | null): ChangeSource => {
  if (!source) return ChangeSource.UNSPECIFIED;
  return CHANGE_SOURCE_MAP[source.toLowerCase()] ?? ChangeSource.UNSPECIFIED;
};

/**
 * flowDefinitionFromJson decodes a snake_case proto JSON object (as produced
 * by buildFlowDefinition) into a typed WorkflowDefinitionV2 message accepted
 * by the Connect-RPC client.
 */
export const flowDefinitionFromJson = (raw: unknown): WorkflowDefinitionV2 =>
  fromJson(WorkflowDefinitionV2Schema, raw as Parameters<typeof fromJson>[1], {
    ignoreUnknownFields: true,
  });

