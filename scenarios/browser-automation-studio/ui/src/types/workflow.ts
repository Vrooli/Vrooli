/**
 * Workflow-related type definitions
 */

import type { Node, Edge } from 'reactflow';

export interface FlowDefinition {
  nodes: Node[];
  edges: Edge[];
}

export interface WorkflowMetadataTyped {
  name?: string;
  description?: string;
  labels?: Record<string, string>;
  version?: string;
}

export interface WorkflowSettingsTyped {
  viewport_width?: number;
  viewport_height?: number;
  user_agent?: string;
  locale?: string;
  timeout_ms?: number;
  entry_selector_timeout_ms?: number;
  headless?: boolean;
  extras?: Record<string, unknown>;
}

export interface WorkflowDefinition {
  metadata?: Record<string, unknown> | null;
  metadata_typed?: WorkflowMetadataTyped | null;
  settings?: Record<string, unknown> | null;
  settings_typed?: WorkflowSettingsTyped | null;
  nodes: Node[];
  edges: Edge[];
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

// ========================================================================
// NOTE: V2 Workflow Types are defined in Proto schemas
// ========================================================================
// V2 types (WorkflowNodeV2, ActionDefinition, etc.) should be imported from:
//   @vrooli/proto-types/browser-automation-studio/v1/workflow_v2_pb
//   @vrooli/proto-types/browser-automation-studio/v1/action_pb
//
// This file only contains UI-specific types that are not in proto.
