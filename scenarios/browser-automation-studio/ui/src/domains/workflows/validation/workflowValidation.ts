import { toJson } from '@bufbuild/protobuf';
import {
  WorkflowDefinitionV2Schema,
  type WorkflowDefinitionV2 as ProtoWorkflowDefinitionV2,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { WorkflowValidationResultSchema } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import {
  flowDefinitionFromJson,
  validateWorkflowViaApi,
} from '@/domains/workflows/services/workflowApi';
import { parseProtoStrict } from '@/utils/proto';
import type { WorkflowDefinition, WorkflowValidationResult } from '@/types/workflow';

interface ValidationOptions {
  strict?: boolean;
}

export const validateWorkflowDefinition = async (
  workflow: WorkflowDefinition,
  _options: ValidationOptions = {},
): Promise<WorkflowValidationResult> => {
  // Strict validation is enforced server-side; the proto contract no longer
  // exposes a boolean toggle on the wire. The WorkflowValidator's V2 lint pass
  // already runs the resolved-token checks.
  const defProto = flowDefinitionFromJson(normalizeWorkflowDefinition(workflow));
  const resp = await validateWorkflowViaApi(defProto);
  if (!resp.result) {
    throw new Error('Workflow validation returned empty result');
  }
  return toJson(WorkflowValidationResultSchema, resp.result, { useProtoFieldName: true }) as unknown as WorkflowValidationResult;
};

const normalizeWorkflowDefinition = (workflow: WorkflowDefinition): Record<string, unknown> => {
  try {
    const parsed = parseProtoStrict<ProtoWorkflowDefinitionV2>(WorkflowDefinitionV2Schema, workflow as unknown);
    return toJson(WorkflowDefinitionV2Schema, parsed, { useProtoFieldName: true }) as unknown as Record<string, unknown>;
  } catch {
    return workflow as unknown as Record<string, unknown>;
  }
};
