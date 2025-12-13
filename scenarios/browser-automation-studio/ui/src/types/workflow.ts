/**
 * Workflow-related type definitions
 */

import { Node, Edge } from 'reactflow';

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
  timeout_seconds?: number;
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
// V2 Workflow Types (Proto-based unified format)
// ========================================================================

/**
 * ActionType enum values matching the proto ActionType.
 */
export type ActionType =
  | 'navigate'
  | 'click'
  | 'input'
  | 'wait'
  | 'assert'
  | 'scroll'
  | 'select'
  | 'evaluate'
  | 'keyboard'
  | 'hover'
  | 'screenshot'
  | 'focus'
  | 'blur';

/**
 * Navigate action parameters.
 */
export interface NavigateParams {
  url: string;
  waitForSelector?: string;
  timeoutMs?: number;
  waitUntil?: string;
}

/**
 * Click action parameters.
 */
export interface ClickParams {
  selector: string;
  button?: string;
  clickCount?: number;
  delayMs?: number;
  modifiers?: string[];
  force?: boolean;
}

/**
 * Input/type action parameters.
 */
export interface InputParams {
  selector: string;
  value: string;
  isSensitive?: boolean;
  submit?: boolean;
  clearFirst?: boolean;
  delayMs?: number;
}

/**
 * Wait action parameters.
 */
export interface WaitParams {
  durationMs?: number;
  selector?: string;
  state?: string;
  timeoutMs?: number;
}

/**
 * Assert action parameters.
 */
export interface AssertParams {
  selector: string;
  mode: string;
  expected?: unknown;
  negated?: boolean;
  caseSensitive?: boolean;
  attributeName?: string;
  failureMessage?: string;
}

/**
 * Scroll action parameters.
 */
export interface ScrollParams {
  selector?: string;
  x?: number;
  y?: number;
  deltaX?: number;
  deltaY?: number;
  behavior?: string;
}

/**
 * Select option action parameters.
 */
export interface SelectParams {
  selector: string;
  value?: string;
  label?: string;
  index?: number;
  timeoutMs?: number;
}

/**
 * Evaluate/execute JavaScript action parameters.
 */
export interface EvaluateParams {
  expression: string;
  storeResult?: string;
}

/**
 * Keyboard action parameters.
 */
export interface KeyboardParams {
  key?: string;
  keys?: string[];
  modifiers?: string[];
  action?: string;
}

/**
 * Hover action parameters.
 */
export interface HoverParams {
  selector: string;
  timeoutMs?: number;
}

/**
 * Screenshot action parameters.
 */
export interface ScreenshotParams {
  fullPage?: boolean;
  selector?: string;
  quality?: number;
}

/**
 * Focus action parameters.
 */
export interface FocusParams {
  selector: string;
  scroll?: boolean;
  timeoutMs?: number;
}

/**
 * Blur action parameters.
 */
export interface BlurParams {
  selector?: string;
  timeoutMs?: number;
}

/**
 * Union type for all action params.
 */
export type ActionParams =
  | NavigateParams
  | ClickParams
  | InputParams
  | WaitParams
  | AssertParams
  | ScrollParams
  | SelectParams
  | EvaluateParams
  | KeyboardParams
  | HoverParams
  | ScreenshotParams
  | FocusParams
  | BlurParams;

/**
 * Selector candidate from recording.
 */
export interface SelectorCandidateV2 {
  type: string;
  value: string;
  confidence: number;
  specificity: number;
}

/**
 * Element snapshot captured during recording.
 */
export interface ElementSnapshotV2 {
  tagName: string;
  id?: string;
  className?: string;
  innerText?: string;
  attributes?: Record<string, string>;
  isVisible: boolean;
  isEnabled: boolean;
  role?: string;
  ariaLabel?: string;
}

/**
 * Bounding box for element positions.
 */
export interface BoundingBoxV2 {
  x: number;
  y: number;
  width: number;
  height: number;
}

/**
 * Action metadata from recording.
 */
export interface ActionMetadataV2 {
  label?: string;
  selectorCandidates?: SelectorCandidateV2[];
  elementSnapshot?: ElementSnapshotV2;
  confidence?: number;
  recordedAt?: string;
  recordedBoundingBox?: BoundingBoxV2;
}

/**
 * Unified action definition.
 */
export interface ActionDefinitionV2 {
  type: ActionType;
  params: ActionParams;
  metadata?: ActionMetadataV2;
}

/**
 * Resilience configuration for node execution.
 */
export interface ResilienceConfigV2 {
  maxAttempts?: number;
  delayMs?: number;
  backoffFactor?: number;
  preconditionSelector?: string;
  preconditionTimeoutMs?: number;
  successSelector?: string;
  successTimeoutMs?: number;
}

/**
 * Node execution settings.
 */
export interface NodeExecutionSettingsV2 {
  timeoutMs?: number;
  waitAfterMs?: number;
  continueOnError?: boolean;
  resilience?: ResilienceConfigV2;
}

/**
 * Node position in the workflow canvas.
 */
export interface NodePositionV2 {
  x: number;
  y: number;
}

/**
 * V2 workflow node - lean proto-based format.
 */
export interface WorkflowNodeV2 {
  id: string;
  action: ActionDefinitionV2;
  position?: NodePositionV2;
  executionSettings?: NodeExecutionSettingsV2;
}

/**
 * V2 workflow edge - connection between nodes.
 */
export interface WorkflowEdgeV2 {
  id: string;
  source: string;
  target: string;
  type?: string;
  label?: string;
  sourceHandle?: string;
  targetHandle?: string;
}

/**
 * V2 workflow metadata.
 */
export interface WorkflowMetadataV2 {
  name?: string;
  description?: string;
  labels?: Record<string, string>;
  version?: string;
  requirement?: string;
  owner?: string;
}

/**
 * V2 workflow settings.
 */
export interface WorkflowSettingsV2 {
  viewportWidth?: number;
  viewportHeight?: number;
  userAgent?: string;
  locale?: string;
  timeoutSeconds?: number;
  headless?: boolean;
  entrySelector?: string;
  entrySelectorTimeoutMs?: number;
}

/**
 * V2 workflow definition - complete workflow storage format.
 */
export interface WorkflowDefinitionV2 {
  metadata?: WorkflowMetadataV2;
  settings?: WorkflowSettingsV2;
  nodes: WorkflowNodeV2[];
  edges: WorkflowEdgeV2[];
}

// ========================================================================
// V1 to V2 Conversion Utilities
// ========================================================================

/**
 * Converts a V1 React Flow node to a V2 WorkflowNode.
 */
export function v1NodeToV2(node: Node): WorkflowNodeV2 {
  const data = node.data || {};
  const type = (node.type || 'click') as ActionType;

  // Build action definition
  const action: ActionDefinitionV2 = {
    type,
    params: extractV1Params(type, data),
  };

  // Add metadata if present
  if (data.label || data.confidence || data.selectorCandidates) {
    action.metadata = {
      label: data.label,
      confidence: data.confidence,
      selectorCandidates: data.selectorCandidates,
    };
  }

  const v2Node: WorkflowNodeV2 = {
    id: node.id,
    action,
  };

  if (node.position) {
    v2Node.position = {
      x: node.position.x,
      y: node.position.y,
    };
  }

  // Extract execution settings
  if (data.timeoutMs || data.waitAfterMs || data.continueOnError || data.resilience) {
    v2Node.executionSettings = {
      timeoutMs: data.timeoutMs,
      waitAfterMs: data.waitAfterMs,
      continueOnError: data.continueOnError,
      resilience: data.resilience,
    };
  }

  return v2Node;
}

/**
 * Converts a V2 WorkflowNode to a V1 React Flow node.
 */
export function v2NodeToV1(v2Node: WorkflowNodeV2): Node {
  const data: Record<string, unknown> = extractV2ParamsToData(v2Node.action);

  // Add metadata
  if (v2Node.action.metadata) {
    if (v2Node.action.metadata.label) data.label = v2Node.action.metadata.label;
    if (v2Node.action.metadata.confidence) data.confidence = v2Node.action.metadata.confidence;
    if (v2Node.action.metadata.selectorCandidates) {
      data.selectorCandidates = v2Node.action.metadata.selectorCandidates;
    }
  }

  // Add execution settings
  if (v2Node.executionSettings) {
    if (v2Node.executionSettings.timeoutMs) data.timeoutMs = v2Node.executionSettings.timeoutMs;
    if (v2Node.executionSettings.waitAfterMs) data.waitAfterMs = v2Node.executionSettings.waitAfterMs;
    if (v2Node.executionSettings.continueOnError) data.continueOnError = v2Node.executionSettings.continueOnError;
    if (v2Node.executionSettings.resilience) data.resilience = v2Node.executionSettings.resilience;
  }

  return {
    id: v2Node.id,
    type: v2Node.action.type,
    position: v2Node.position || { x: 0, y: 0 },
    data,
  };
}

/**
 * Converts a V1 React Flow edge to a V2 WorkflowEdge.
 */
export function v1EdgeToV2(edge: Edge): WorkflowEdgeV2 {
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    type: edge.type,
    label: typeof edge.label === 'string' ? edge.label : undefined,
    sourceHandle: edge.sourceHandle || undefined,
    targetHandle: edge.targetHandle || undefined,
  };
}

/**
 * Converts a V2 WorkflowEdge to a V1 React Flow edge.
 */
export function v2EdgeToV1(v2Edge: WorkflowEdgeV2): Edge {
  return {
    id: v2Edge.id,
    source: v2Edge.source,
    target: v2Edge.target,
    type: v2Edge.type,
    label: v2Edge.label,
    sourceHandle: v2Edge.sourceHandle,
    targetHandle: v2Edge.targetHandle,
  };
}

/**
 * Converts a complete V1 workflow to V2 format.
 */
export function workflowV1ToV2(v1: WorkflowDefinition): WorkflowDefinitionV2 {
  return {
    metadata: v1.metadata_typed ? {
      name: v1.metadata_typed.name,
      description: v1.metadata_typed.description,
      labels: v1.metadata_typed.labels,
      version: v1.metadata_typed.version,
    } : undefined,
    settings: v1.settings_typed ? {
      viewportWidth: v1.settings_typed.viewport_width,
      viewportHeight: v1.settings_typed.viewport_height,
      userAgent: v1.settings_typed.user_agent,
      locale: v1.settings_typed.locale,
      timeoutSeconds: v1.settings_typed.timeout_seconds,
      headless: v1.settings_typed.headless,
    } : undefined,
    nodes: v1.nodes.map(v1NodeToV2),
    edges: v1.edges.map(v1EdgeToV2),
  };
}

/**
 * Converts a complete V2 workflow to V1 format.
 */
export function workflowV2ToV1(v2: WorkflowDefinitionV2): WorkflowDefinition {
  return {
    metadata_typed: v2.metadata ? {
      name: v2.metadata.name,
      description: v2.metadata.description,
      labels: v2.metadata.labels,
      version: v2.metadata.version,
    } : undefined,
    settings_typed: v2.settings ? {
      viewport_width: v2.settings.viewportWidth,
      viewport_height: v2.settings.viewportHeight,
      user_agent: v2.settings.userAgent,
      locale: v2.settings.locale,
      timeout_seconds: v2.settings.timeoutSeconds,
      headless: v2.settings.headless,
    } : undefined,
    nodes: v2.nodes.map(v2NodeToV1),
    edges: v2.edges.map(v2EdgeToV1),
  };
}

// ========================================================================
// Helper functions for param extraction
// ========================================================================

function extractV1Params(type: ActionType, data: Record<string, unknown>): ActionParams {
  switch (type) {
    case 'navigate':
      return {
        url: (data.url as string) || '',
        waitForSelector: data.waitForSelector as string | undefined,
        timeoutMs: data.timeoutMs as number | undefined,
        waitUntil: data.waitUntil as string | undefined,
      };
    case 'click':
      return {
        selector: (data.selector as string) || '',
        button: data.button as string | undefined,
        clickCount: data.clickCount as number | undefined,
        delayMs: data.delayMs as number | undefined,
        modifiers: data.modifiers as string[] | undefined,
        force: data.force as boolean | undefined,
      };
    case 'input':
      return {
        selector: (data.selector as string) || '',
        value: (data.value as string) || (data.text as string) || '',
        isSensitive: data.isSensitive as boolean | undefined,
        submit: data.submit as boolean | undefined,
        clearFirst: data.clearFirst as boolean | undefined,
        delayMs: data.delayMs as number | undefined,
      };
    case 'wait':
      return {
        durationMs: data.durationMs as number | undefined,
        selector: data.selector as string | undefined,
        state: data.state as string | undefined,
        timeoutMs: data.timeoutMs as number | undefined,
      };
    case 'assert':
      return {
        selector: (data.selector as string) || '',
        mode: (data.mode as string) || (data.assertMode as string) || 'exists',
        expected: data.expected,
        negated: data.negated as boolean | undefined,
        caseSensitive: data.caseSensitive as boolean | undefined,
        attributeName: data.attributeName as string | undefined,
        failureMessage: data.failureMessage as string | undefined,
      };
    case 'scroll':
      return {
        selector: data.selector as string | undefined,
        x: data.x as number | undefined,
        y: data.y as number | undefined,
        deltaX: data.deltaX as number | undefined,
        deltaY: data.deltaY as number | undefined,
        behavior: data.behavior as string | undefined,
      };
    case 'select':
      return {
        selector: (data.selector as string) || '',
        value: data.value as string | undefined,
        label: data.label as string | undefined,
        index: data.index as number | undefined,
        timeoutMs: data.timeoutMs as number | undefined,
      };
    case 'evaluate':
      return {
        expression: (data.expression as string) || '',
        storeResult: data.storeResult as string | undefined,
      };
    case 'keyboard':
      return {
        key: data.key as string | undefined,
        keys: data.keys as string[] | undefined,
        modifiers: data.modifiers as string[] | undefined,
        action: data.action as string | undefined,
      };
    case 'hover':
      return {
        selector: (data.selector as string) || '',
        timeoutMs: data.timeoutMs as number | undefined,
      };
    case 'screenshot':
      return {
        fullPage: data.fullPage as boolean | undefined,
        selector: data.selector as string | undefined,
        quality: data.quality as number | undefined,
      };
    case 'focus':
      return {
        selector: (data.selector as string) || '',
        scroll: data.scroll as boolean | undefined,
        timeoutMs: data.timeoutMs as number | undefined,
      };
    case 'blur':
      return {
        selector: data.selector as string | undefined,
        timeoutMs: data.timeoutMs as number | undefined,
      };
    default:
      return { selector: (data.selector as string) || '' } as ClickParams;
  }
}

function extractV2ParamsToData(action: ActionDefinitionV2): Record<string, unknown> {
  const params = action.params as Record<string, unknown>;
  const data: Record<string, unknown> = {};

  // Copy all params to data
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null) {
      data[key] = value;
    }
  }

  return data;
}
