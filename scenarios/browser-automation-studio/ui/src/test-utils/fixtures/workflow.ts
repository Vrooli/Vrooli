import type { Edge, Node, Viewport } from 'reactflow';
import { vi } from 'vitest';

type WorkflowNodeData = Record<string, unknown> & {
  label?: string;
};

export function createWorkflowNode(
  overrides: Partial<Node<WorkflowNodeData>> = {}
): Node<WorkflowNodeData> {
  return {
    id: 'node-1',
    type: 'navigate',
    position: { x: 100, y: 100 },
    data: { label: 'Navigate' },
    ...overrides,
  };
}

export function createWorkflowEdge(overrides: Partial<Edge> = {}): Edge {
  return {
    id: 'edge-1',
    source: 'node-1',
    target: 'node-2',
    ...overrides,
  };
}

export function createReactFlowViewport(overrides: Partial<Viewport> = {}): Viewport {
  return {
    x: 100,
    y: 50,
    zoom: 0.8,
    ...overrides,
  };
}

export function createWorkflowValidationResponse() {
  return {
    valid: true,
    errors: [],
    warnings: [],
    stats: {
      node_count: 0,
      edge_count: 0,
      selector_count: 0,
      unique_selector_count: 0,
      element_wait_count: 0,
      has_metadata: false,
      has_execution_viewport: false,
    },
    schema_version: 'test',
    checked_at: new Date().toISOString(),
    duration_ms: 1,
  };
}

export function createWorkflowBuilderStoreState() {
  return {
    nodes: [],
    edges: [],
    workflows: [],
    currentWorkflow: { id: 'workflow-1', name: 'Test Workflow' },
    isDirty: false,
    hasVersionConflict: false,
    updateWorkflow: vi.fn(),
    scheduleAutosave: vi.fn(),
    cancelAutosave: vi.fn(),
    loadWorkflows: vi.fn().mockResolvedValue([]),
  };
}
