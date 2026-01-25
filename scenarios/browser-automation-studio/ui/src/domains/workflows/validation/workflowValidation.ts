import { toJson } from '@bufbuild/protobuf';
import {
  WorkflowDefinitionV2Schema,
  type WorkflowDefinitionV2 as ProtoWorkflowDefinitionV2,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import {
  WorkflowValidationResultSchema,
  type WorkflowValidationResult as ProtoWorkflowValidationResult,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { getConfig } from '@/config';
import { parseProtoStrict } from '@/utils/proto';
import type { WorkflowDefinition, WorkflowValidationResult } from '@/types/workflow';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const safeJson = async (response: Response): Promise<unknown> => {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
};

interface ValidationOptions {
  strict?: boolean;
}

export const validateWorkflowDefinition = async (
  workflow: WorkflowDefinition,
  options: ValidationOptions = {},
): Promise<WorkflowValidationResult> => {
  const config = await getConfig();
  const normalizedWorkflow = normalizeWorkflowDefinition(workflow);
  const response = await fetch(`${config.API_URL}/workflows/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ workflow: normalizedWorkflow, strict: Boolean(options.strict) }),
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response));
  }

  const payload = await safeJson(response);
  if (!payload) {
    throw new Error(`Workflow validation failed (${response.status})`);
  }
  const proto = parseProtoStrict<ProtoWorkflowValidationResult>(WorkflowValidationResultSchema, payload);
  return toJson(WorkflowValidationResultSchema, proto, { useProtoFieldName: true }) as unknown as WorkflowValidationResult;
};

const normalizeWorkflowDefinition = (workflow: WorkflowDefinition): Record<string, unknown> => {
  try {
    const parsed = parseProtoStrict<ProtoWorkflowDefinitionV2>(WorkflowDefinitionV2Schema, workflow as unknown);
    return toJson(WorkflowDefinitionV2Schema, parsed, { useProtoFieldName: true }) as unknown as Record<string, unknown>;
  } catch {
    return workflow as unknown as Record<string, unknown>;
  }
};

const readErrorMessage = async (response: Response): Promise<string> => {
  try {
    const payload = await safeJson(response);
    if (isRecord(payload) && typeof payload.message === 'string') {
      return payload.message;
    }
  } catch (_error) {
    // Ignore JSON parsing errors and fall back to status message
  }
  return `Workflow validation failed (${response.status})`;
};
