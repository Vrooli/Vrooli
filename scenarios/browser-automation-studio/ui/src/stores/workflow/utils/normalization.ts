import type { Workflow, WorkflowVersionSummary, WorkflowLoadState } from '../types';
import { computeWorkflowFingerprint } from './fingerprint';
import { setNormalizeVersionSummary } from './proto';

export const parseDate = (value: unknown): Date => {
  if (value instanceof Date) return value;
  if (typeof value === 'string' || typeof value === 'number') {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? new Date() : parsed;
  }
  return new Date();
};

export const ensureArray = <T>(value: unknown, fallback: T[] = []): T[] =>
  Array.isArray(value) ? (value as T[]).slice() : fallback.slice();

export const normalizeVersionSummary = (summary: unknown): WorkflowVersionSummary => {
  const data = summary as Record<string, unknown>;
  const version = typeof data.version === 'number' ? data.version : parseInt(String(data.version ?? '0'), 10) || 0;
  return {
    version,
    workflowId: data.workflow_id ?? data.workflowId ?? '',
    createdAt: parseDate(data.created_at ?? data.createdAt),
    createdBy: data.created_by ?? data.createdBy ?? '',
    changeDescription: data.change_description ?? data.changeDescription ?? '',
    definitionHash: data.definition_hash ?? data.definitionHash ?? '',
    nodeCount: typeof data.node_count === 'number' ? data.node_count : 0,
    edgeCount: typeof data.edge_count === 'number' ? data.edge_count : 0,
    flowDefinition: data.flow_definition ?? data.flowDefinition ?? {},
  } as WorkflowVersionSummary;
};

export const buildWorkflowLoadState = (workflow: Workflow, options: { lastSavedAt?: Date } = {}): WorkflowLoadState => {
  const fingerprint = computeWorkflowFingerprint(workflow, workflow.nodes, workflow.edges);
  return {
    currentWorkflow: workflow, nodes: workflow.nodes, edges: workflow.edges,
    isDirty: false, isSaving: false, lastSavedFingerprint: fingerprint, draftFingerprint: fingerprint,
    lastSavedAt: options.lastSavedAt ?? (workflow.updatedAt instanceof Date ? workflow.updatedAt : new Date()),
    hasVersionConflict: false, lastSaveError: null, versionHistory: [], versionHistoryLoadedFor: null,
    versionHistoryError: null, conflictWorkflow: null, conflictMetadata: null,
  };
};

setNormalizeVersionSummary(normalizeVersionSummary);
