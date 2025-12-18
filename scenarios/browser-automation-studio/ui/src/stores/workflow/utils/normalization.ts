// Note: Node and Edge types are used indirectly through workflow.nodes and workflow.edges
import {
  GetWorkflowResponseSchema,
  WorkflowSummarySchema,
  type GetWorkflowResponse,
  type WorkflowSummary,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { logger } from '../../../utils/logger';
import { normalizeNodes, normalizeEdges } from '@/domains/workflows/utils/normalizers';
import { parseProtoStrict } from '../../../utils/proto';
import type { Workflow, WorkflowVersionSummary, WorkflowLoadState } from '../types';
import { sanitizeViewportSettings, extractExecutionViewport } from './viewport';
import { buildFlowDefinition, sanitizeNodesForPersistence, stripPreviewDataFromNodes } from './serialization';
import { computeWorkflowFingerprint } from './fingerprint';
import { parseWorkflowSummaryMessage, setNormalizeWorkflowResponse, setNormalizeVersionSummary } from './proto';

// ============================================================================
// Date/Array Helpers
// ============================================================================

export const parseDate = (value: unknown): Date => {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === 'string' || typeof value === 'number') {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? new Date() : parsed;
  }
  return new Date();
};

export const ensureArray = <T>(value: unknown, fallback: T[] = []): T[] => {
  if (Array.isArray(value)) {
    return [...value] as T[];
  }
  return [...fallback];
};

// ============================================================================
// Workflow Normalization
// ============================================================================

export const normalizeWorkflowResponse = (workflow: unknown): Workflow => {
  const workflowData = workflow as Record<string, unknown>;
  const flowDef = (workflowData.flow_definition ?? workflowData.flowDefinition) as Record<string, unknown> | undefined;
  const rawNodes = workflowData.nodes ?? flowDef?.nodes ?? [];
  const rawEdges = workflowData.edges ?? flowDef?.edges ?? [];
  const normalizedNodesRaw = normalizeNodes(Array.isArray(rawNodes) ? rawNodes : []);
  const normalizedNodes = stripPreviewDataFromNodes(normalizedNodesRaw);
  const normalizedEdges = normalizeEdges(Array.isArray(rawEdges) ? rawEdges : []);

  const rawDefinition = flowDef ?? {};
  const executionViewport = sanitizeViewportSettings(extractExecutionViewport(rawDefinition));
  const flowDefinition = buildFlowDefinition(
    rawDefinition,
    sanitizeNodesForPersistence(normalizedNodes),
    JSON.parse(JSON.stringify(normalizedEdges ?? [])),
    executionViewport,
  );

  const versionValue = workflowData.version;
  const version = typeof versionValue === 'number'
    ? versionValue
    : parseInt(String(versionValue ?? '1'), 10) || 1;

  return {
    id: workflowData.id ?? '',
    projectId: workflowData.project_id ?? workflowData.projectId,
    name: workflowData.name ?? '',
    description: workflowData.description ?? '',
    folderPath: workflowData.folder_path ?? workflowData.folderPath ?? '/',
    nodes: normalizedNodes,
    edges: normalizedEdges,
    tags: ensureArray<string>(workflowData.tags),
    version,
    lastChangeSource: workflowData.last_change_source ?? workflowData.lastChangeSource ?? 'manual',
    lastChangeDescription: workflowData.last_change_description ?? workflowData.lastChangeDescription ?? '',
    createdAt: parseDate(workflowData.created_at ?? workflowData.createdAt),
    updatedAt: parseDate(workflowData.updated_at ?? workflowData.updatedAt),
    flowDefinition,
    executionViewport,
  } as Workflow;
};

// ============================================================================
// Version Summary Normalization
// ============================================================================

export const normalizeVersionSummary = (summary: unknown): WorkflowVersionSummary => {
  const summaryData = summary as Record<string, unknown>;
  const versionNumber = typeof summaryData.version === 'number'
    ? summaryData.version
    : parseInt(String(summaryData.version ?? '0'), 10) || 0;

  return {
    version: versionNumber,
    workflowId: summaryData.workflow_id ?? summaryData.workflowId ?? '',
    createdAt: parseDate(summaryData.created_at ?? summaryData.createdAt),
    createdBy: summaryData.created_by ?? summaryData.createdBy ?? '',
    changeDescription: summaryData.change_description ?? summaryData.changeDescription ?? '',
    definitionHash: summaryData.definition_hash ?? summaryData.definitionHash ?? '',
    nodeCount: typeof summaryData.node_count === 'number' ? summaryData.node_count : 0,
    edgeCount: typeof summaryData.edge_count === 'number' ? summaryData.edge_count : 0,
    flowDefinition: summaryData.flow_definition ?? summaryData.flowDefinition ?? {},
  } as WorkflowVersionSummary;
};

// ============================================================================
// Payload Normalization
// ============================================================================

export const normalizeWorkflowPayloadOrThrow = (payload: unknown, action: string): Workflow => {
  let normalized: Workflow | null = null;

  try {
    const proto = parseProtoStrict<GetWorkflowResponse>(GetWorkflowResponseSchema, payload);
    if (proto.workflow) {
      normalized = parseWorkflowSummaryMessage(proto.workflow);
    }
  } catch (err) {
    logger.error('Failed to parse get workflow proto', { component: 'WorkflowStore', action }, err);
  }

  if (!normalized) {
    try {
      const proto = parseProtoStrict<WorkflowSummary>(WorkflowSummarySchema, payload);
      normalized = parseWorkflowSummaryMessage(proto);
    } catch (err) {
      logger.error('Failed to parse workflow summary proto', { component: 'WorkflowStore', action }, err);
    }
  }

  if (!normalized && payload && typeof payload === 'object') {
    const maybeWrapper = payload as Record<string, unknown>;
    normalized = parseWorkflowSummaryMessage((maybeWrapper.workflow ?? maybeWrapper) as Record<string, unknown>);
  }

  if (!normalized) {
    throw new Error('Failed to parse workflow payload');
  }

  return normalized;
};

// ============================================================================
// Load State Builder
// ============================================================================

export const buildWorkflowLoadState = (workflow: Workflow, options: { lastSavedAt?: Date } = {}): WorkflowLoadState => {
  const fingerprint = computeWorkflowFingerprint(workflow, workflow.nodes, workflow.edges);
  return {
    currentWorkflow: workflow,
    nodes: workflow.nodes,
    edges: workflow.edges,
    isDirty: false,
    isSaving: false,
    lastSavedFingerprint: fingerprint,
    draftFingerprint: fingerprint,
    lastSavedAt: options.lastSavedAt ?? (workflow.updatedAt instanceof Date ? workflow.updatedAt : new Date()),
    hasVersionConflict: false,
    lastSaveError: null,
    versionHistory: [],
    versionHistoryLoadedFor: null,
    versionHistoryError: null,
    conflictWorkflow: null,
    conflictMetadata: null,
  };
};

// ============================================================================
// Initialize Proto Setters
// ============================================================================

// Register the normalization functions with the proto module to avoid circular deps
setNormalizeWorkflowResponse(normalizeWorkflowResponse);
setNormalizeVersionSummary(normalizeVersionSummary);
