import type { Node, Edge } from 'reactflow';
import type { Workflow } from '../types';
import { canvasToWorkflowDefinition } from './codec';
import { sanitizeViewportSettings } from './viewport';

// ============================================================================
// Stable Serialization
// ============================================================================

export const stableSerialize = (value: unknown): string => {
  const seen = new WeakSet<object>();

  const normalize = (input: unknown): unknown => {
    if (input === null || typeof input !== 'object') {
      return input;
    }

    if (seen.has(input as object)) {
      return null;
    }

    seen.add(input as object);

    if (Array.isArray(input)) {
      return input.map(normalize);
    }

    const entries = Object.entries(input as Record<string, unknown>)
      .filter(([, val]) => typeof val !== 'function' && val !== undefined)
      .sort(([a], [b]) => a.localeCompare(b));

    const result: Record<string, unknown> = {};
    for (const [key, val] of entries) {
      result[key] = normalize(val);
    }
    return result;
  };

  return JSON.stringify(normalize(value));
};

// ============================================================================
// Workflow Fingerprint
// ============================================================================

export const computeWorkflowFingerprint = (workflow: Workflow | null, nodes: Node[], edges: Edge[]): string => {
  if (!workflow) {
    return '';
  }

  const definition = canvasToWorkflowDefinition(workflow.flowDefinition, nodes, edges ?? [], workflow.executionViewport);
  const sanitizedViewport = sanitizeViewportSettings(workflow.executionViewport);

  return stableSerialize({
    name: workflow.name ?? '',
    description: workflow.description ?? '',
    folderPath: workflow.folderPath ?? '/',
    tags: Array.isArray(workflow.tags) ? [...workflow.tags].sort() : [],
    flowDefinition: definition,
    executionViewport: sanitizedViewport ?? null,
  });
};
