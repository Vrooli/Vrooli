import { isMessage } from '@bufbuild/protobuf';
import {
  CreateWorkflowResponseSchema,
  UpdateWorkflowResponseSchema,
  WorkflowSummarySchema,
  WorkflowVersionSchema,
  type WorkflowSummary,
  type WorkflowVersion as ProtoWorkflowVersion,
  type CreateWorkflowResponse,
  type UpdateWorkflowResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { WorkflowDefinitionV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { logger } from '../../../utils/logger';
import { parseProtoStrict, protoMessageToJson } from '../../../utils/proto';
import type { Workflow, WorkflowVersionSummary } from '../types';
import { isPlainObject } from './viewport';

// ============================================================================
// Proto Helpers
// ============================================================================

export const toJsonRecord = (schema: any, message: any): Record<string, unknown> =>
  protoMessageToJson(schema, message);

// ============================================================================
// Workflow Summary Parsing
// ============================================================================

export const workflowSummaryToPayload = (summary: WorkflowSummary | null | undefined): Record<string, unknown> | null => {
  if (!summary) {
    return null;
  }

  if (isMessage(summary)) {
    const summaryJson = toJsonRecord(WorkflowSummarySchema, summary as any);
    const flowDef = (summary as any).flowDefinition;
    if (flowDef) {
      const flowJson = toJsonRecord(WorkflowDefinitionV2Schema, flowDef);
      if (Object.keys(flowJson).length > 0) {
        summaryJson.flow_definition = flowJson;
      }
    }
    return summaryJson;
  }

  if (isPlainObject(summary)) {
    return { ...(summary as Record<string, unknown>) };
  }

  return null;
};

/**
 * Parse a WorkflowSummary proto message or plain object into a Workflow.
 * Note: This function requires normalizeWorkflowResponse from normalization.ts
 * It's defined here but the implementation is provided via injection.
 */
export type NormalizeWorkflowFn = (payload: Record<string, unknown>) => Workflow;

let _normalizeWorkflowResponse: NormalizeWorkflowFn | null = null;

export const setNormalizeWorkflowResponse = (fn: NormalizeWorkflowFn): void => {
  _normalizeWorkflowResponse = fn;
};

export const parseWorkflowSummaryMessage = (summary: WorkflowSummary | Record<string, unknown> | null | undefined): Workflow | null => {
  const payload = workflowSummaryToPayload(summary as WorkflowSummary);
  if (!payload) {
    return null;
  }
  try {
    if (!_normalizeWorkflowResponse) {
      throw new Error('normalizeWorkflowResponse not initialized');
    }
    return _normalizeWorkflowResponse(payload);
  } catch (error) {
    logger.error('Failed to normalize workflow from payload', { component: 'WorkflowStore', action: 'parseWorkflowSummary' }, error);
    return null;
  }
};

// ============================================================================
// Create/Update Response Parsing
// ============================================================================

export const parseWorkflowFromCreateResponse = (raw: unknown): Workflow | null => {
  try {
    const proto = parseProtoStrict<CreateWorkflowResponse>(CreateWorkflowResponseSchema, raw);
    const summaryPayload = workflowSummaryToPayload(proto.workflow);
    if (!summaryPayload) return null;
    if (proto.flowDefinition) {
      summaryPayload.flowDefinition = toJsonRecord(WorkflowDefinitionV2Schema, proto.flowDefinition);
    }
    if (!_normalizeWorkflowResponse) {
      throw new Error('normalizeWorkflowResponse not initialized');
    }
    return _normalizeWorkflowResponse(summaryPayload);
  } catch (error) {
    logger.error('Failed to parse create workflow proto response', { component: 'WorkflowStore', action: 'parseCreateWorkflow' }, error);
    return null;
  }
};

export const parseWorkflowFromUpdateResponse = (raw: unknown): Workflow | null => {
  try {
    const proto = parseProtoStrict<UpdateWorkflowResponse>(UpdateWorkflowResponseSchema, raw);
    const summaryPayload = workflowSummaryToPayload(proto.workflow);
    if (!summaryPayload) return null;
    if (proto.flowDefinition) {
      summaryPayload.flowDefinition = toJsonRecord(WorkflowDefinitionV2Schema, proto.flowDefinition);
    }
    if (!_normalizeWorkflowResponse) {
      throw new Error('normalizeWorkflowResponse not initialized');
    }
    return _normalizeWorkflowResponse(summaryPayload);
  } catch (error) {
    logger.error('Failed to parse update workflow proto response', { component: 'WorkflowStore', action: 'parseUpdateWorkflow' }, error);
    return null;
  }
};

// ============================================================================
// Version Parsing
// ============================================================================

export type NormalizeVersionFn = (payload: Record<string, unknown>) => WorkflowVersionSummary;

let _normalizeVersionSummary: NormalizeVersionFn | null = null;

export const setNormalizeVersionSummary = (fn: NormalizeVersionFn): void => {
  _normalizeVersionSummary = fn;
};

export const parseWorkflowVersionMessage = (version: ProtoWorkflowVersion | null | undefined): WorkflowVersionSummary | null => {
  if (!version) return null;
  try {
    const payload = toJsonRecord(WorkflowVersionSchema, version);
    if (!_normalizeVersionSummary) {
      throw new Error('normalizeVersionSummary not initialized');
    }
    return _normalizeVersionSummary(payload);
  } catch (error) {
    logger.error('Failed to parse workflow version proto', { component: 'WorkflowStore', action: 'parseWorkflowVersion' }, error);
    return null;
  }
};
