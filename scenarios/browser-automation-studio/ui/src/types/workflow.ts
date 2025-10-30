/**
 * Workflow-related type definitions
 */

import { Node, Edge } from 'reactflow';

export interface FlowDefinition {
  nodes: Node[];
  edges: Edge[];
}

export interface WorkflowDefinition {
  nodes: unknown[];
  edges: unknown[];
}
