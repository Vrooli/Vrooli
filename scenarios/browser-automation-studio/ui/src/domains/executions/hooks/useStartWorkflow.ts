import { useCallback } from 'react';
import {
  useExecutionStore,
  type StartExecutionOptions,
  type ExecutionSettingsOverrides,
  type ArtifactProfile,
} from '../store';
import { usePromptDialog } from '@hooks/usePromptDialog';
import { workflowStartsWithNavigate } from '@utils/nodeUtils';
import { getConfig } from '../../../config';
import type { Node, Edge } from 'reactflow';

interface WorkflowDefinition {
  nodes: Node[];
  edges: Edge[];
}

/**
 * Fetches the workflow definition (nodes/edges) for a given workflow ID.
 * Returns null if the fetch fails or the workflow has no nodes.
 */
async function fetchWorkflowDefinition(workflowId: string): Promise<WorkflowDefinition | null> {
  try {
    const config = await getConfig();
    const response = await fetch(`${config.API_URL}/workflows/${workflowId}`);
    if (!response.ok) {
      return null;
    }
    const data = await response.json();

    // Handle both direct response and wrapped response formats
    const workflow = data.workflow ?? data;

    // Parse nodes from flow_definition if present
    let nodes: Node[] = [];
    let edges: Edge[] = [];

    if (workflow.flow_definition) {
      const flowDef = typeof workflow.flow_definition === 'string'
        ? JSON.parse(workflow.flow_definition)
        : workflow.flow_definition;
      nodes = Array.isArray(flowDef.nodes) ? flowDef.nodes : [];
      edges = Array.isArray(flowDef.edges) ? flowDef.edges : [];
    } else if (Array.isArray(workflow.nodes)) {
      nodes = workflow.nodes;
      edges = Array.isArray(workflow.edges) ? workflow.edges : [];
    }

    return { nodes, edges };
  } catch {
    return null;
  }
}

export interface UseStartWorkflowOptions {
  /** Called before execution starts, e.g., to save workflow */
  beforeExecute?: () => Promise<void>;
  /** Called after execution starts successfully */
  onSuccess?: (executionId: string) => void;
  /** Called if execution fails */
  onError?: (error: Error) => void;
}

export interface StartWorkflowParams {
  /** The workflow ID to execute */
  workflowId: string;
  /** Optional pre-fetched nodes (avoids API call if available) */
  nodes?: Node[];
  /** Optional pre-fetched edges (avoids API call if available) */
  edges?: Edge[];
  /** Session profile ID to use for this execution */
  sessionProfileId?: string | null;
  /** Runtime overrides for workflow execution settings */
  overrides?: ExecutionSettingsOverrides;
  /** Artifact collection profile */
  artifactProfile?: ArtifactProfile;
  /** Additional execution options (legacy, prefer individual params above) */
  options?: StartExecutionOptions;
}

/**
 * Hook that provides a function to start a workflow execution with automatic
 * handling of the "start URL required" case for workflows without a navigate step.
 *
 * This hook will:
 * 1. Check if the workflow starts with a navigate step
 * 2. If not, prompt the user for a start URL
 * 3. Start the execution with the provided URL
 *
 * @example
 * const { startWorkflow, promptDialogProps } = useStartWorkflow({
 *   onSuccess: (executionId) => console.log('Started:', executionId),
 * });
 *
 * // In JSX:
 * <PromptDialog {...promptDialogProps} />
 *
 * // To start:
 * await startWorkflow({ workflowId: 'abc-123' });
 */
export function useStartWorkflow(hookOptions: UseStartWorkflowOptions = {}) {
  const startExecution = useExecutionStore((state) => state.startExecution);

  const {
    dialogState,
    prompt,
    setValue,
    close,
    submit,
  } = usePromptDialog();

  const startWorkflow = useCallback(async ({
    workflowId,
    nodes: providedNodes,
    edges: providedEdges,
    sessionProfileId,
    overrides,
    artifactProfile,
    options = {},
  }: StartWorkflowParams): Promise<string | null> => {
    try {
      // Get nodes/edges either from params or by fetching
      let nodes = providedNodes;
      let edges = providedEdges;

      if (!nodes || !edges) {
        const definition = await fetchWorkflowDefinition(workflowId);
        if (definition) {
          nodes = definition.nodes;
          edges = definition.edges;
        } else {
          nodes = [];
          edges = [];
        }
      }

      // Check if workflow starts with navigate step
      const hasNavigateStep = workflowStartsWithNavigate(nodes, edges);

      let startUrl: string | undefined = options.startUrl;

      // If no navigate step and no startUrl provided, prompt for one
      if (!hasNavigateStep && nodes.length > 0 && !startUrl) {
        const url = await prompt(
          {
            title: 'Start URL Required',
            message: "This workflow doesn't begin with a Navigate step. Please enter the URL where the workflow should start.",
            label: 'Start URL',
            placeholder: 'example.com',
            submitLabel: 'Run Workflow',
            cancelLabel: 'Cancel',
          },
          {
            validate: (value) => {
              if (!value.trim()) {
                return 'Please enter a URL';
              }
              // Normalize the URL for validation
              let urlToValidate = value.trim();
              if (!/^https?:\/\//i.test(urlToValidate)) {
                urlToValidate = `https://${urlToValidate}`;
              }
              try {
                new URL(urlToValidate);
                return null;
              } catch {
                return 'Please enter a valid URL (e.g., example.com)';
              }
            },
            normalize: (value) => {
              const trimmed = value.trim();
              // Auto-prepend https:// if no protocol
              if (!/^https?:\/\//i.test(trimmed)) {
                return `https://${trimmed}`;
              }
              return trimmed;
            },
          }
        );

        if (!url) {
          // User cancelled
          return null;
        }
        startUrl = url;
      }

      // Run beforeExecute hook if provided
      if (hookOptions.beforeExecute) {
        await hookOptions.beforeExecute();
      }

      // Start the execution with all options merged
      const executionId = await startExecution(workflowId, {
        ...options,
        startUrl,
        sessionProfileId,
        executionOverrides: overrides,
        artifactConfig: artifactProfile ? { profile: artifactProfile } : options.artifactConfig,
      });

      if (hookOptions.onSuccess) {
        hookOptions.onSuccess(executionId);
      }

      return executionId;
    } catch (error) {
      const err = error instanceof Error ? error : new Error('Failed to start workflow');
      if (hookOptions.onError) {
        hookOptions.onError(err);
      }
      throw err;
    }
  }, [startExecution, prompt, hookOptions]);

  return {
    startWorkflow,
    /** Props to pass to PromptDialog component */
    promptDialogProps: {
      state: dialogState,
      onValueChange: setValue,
      onClose: close,
      onSubmit: submit,
    },
  };
}
