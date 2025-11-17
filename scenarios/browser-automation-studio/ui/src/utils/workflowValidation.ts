import { getConfig } from '../config';
import type { WorkflowDefinition, WorkflowValidationResult } from '../types/workflow';

interface ValidationOptions {
  strict?: boolean;
}

export const validateWorkflowDefinition = async (
  workflow: WorkflowDefinition,
  options: ValidationOptions = {},
): Promise<WorkflowValidationResult> => {
  const config = await getConfig();
  const response = await fetch(`${config.API_URL}/workflows/validate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ workflow, strict: Boolean(options.strict) }),
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response));
  }

  return response.json() as Promise<WorkflowValidationResult>;
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
