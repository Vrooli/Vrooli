import { toJson } from '@bufbuild/protobuf';
import {
  WorkflowDefinitionSchema,
  type WorkflowDefinition as ProtoWorkflowDefinition,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflow_pb';
import {
  WorkflowValidationResultSchema,
  type WorkflowValidationResult as ProtoWorkflowValidationResult,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflow_service_pb';
import { getConfig } from '../config';
import { parseProtoStrict } from './proto';
import type { WorkflowDefinition, WorkflowValidationResult } from '../types/workflow';

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

  const payload = await response.json();
  const proto = parseProtoStrict<ProtoWorkflowValidationResult>(WorkflowValidationResultSchema, payload);
  return toJson(WorkflowValidationResultSchema, proto, { useProtoFieldName: true }) as unknown as WorkflowValidationResult;
};

const normalizeWorkflowDefinition = (workflow: WorkflowDefinition): Record<string, unknown> => {
  try {
    const parsed = parseProtoStrict<ProtoWorkflowDefinition>(WorkflowDefinitionSchema, workflow as unknown);
    return toJson(WorkflowDefinitionSchema, parsed, { useProtoFieldName: true }) as unknown as Record<string, unknown>;
  } catch {
    return workflow as unknown as Record<string, unknown>;
  }
};

const readErrorMessage = async (response: Response): Promise<string> => {
  try {
    const payload = await response.json();
    if (payload && typeof payload.message === 'string') {
      return payload.message;
    }
  } catch (error) {
    // Ignore JSON parsing errors and fall back to status message
  }
  return `Workflow validation failed (${response.status})`;
};
