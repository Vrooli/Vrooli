/**
 * Workflow-related type definitions
 */

import { Node, Edge } from 'reactflow';

export interface FlowDefinition {
  nodes: Node[];
  edges: Edge[];
}

export interface WorkflowDefinition {
  metadata?: Record<string, unknown> | null;
  settings?: Record<string, unknown> | null;
  nodes: unknown[];
  edges: unknown[];
}

export interface WorkflowValidationIssue {
  severity: 'error' | 'warning';
  code: string;
  message: string;
  node_id?: string;
  node_type?: string;
  field?: string;
  pointer?: string;
  hint?: string;
}

export interface WorkflowValidationStats {
  node_count: number;
  edge_count: number;
  selector_count: number;
  unique_selector_count: number;
  element_wait_count: number;
  has_metadata: boolean;
  has_execution_viewport: boolean;
}

export interface WorkflowValidationResult {
  valid: boolean;
  errors: WorkflowValidationIssue[];
  warnings: WorkflowValidationIssue[];
  stats: WorkflowValidationStats;
  schema_version: string;
  checked_at: string;
  duration_ms: number;
}

export interface ResilienceSettings {
  maxAttempts?: number;
  delayMs?: number;
  backoffFactor?: number;
  preconditionSelector?: string;
  preconditionTimeoutMs?: number;
  preconditionWaitMs?: number;
  successSelector?: string;
  successTimeoutMs?: number;
  successWaitMs?: number;
}
