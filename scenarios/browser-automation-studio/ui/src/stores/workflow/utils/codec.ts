import { fromJson, toJson } from '@bufbuild/protobuf';
import {
  GetWorkflowResponseSchema,
  WorkflowSummarySchema,
  type GetWorkflowResponse,
  type WorkflowSummary,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import {
  WorkflowDefinitionV2Schema,
  type WorkflowDefinitionV2,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import type { Edge, Node } from 'reactflow';
import { logger } from '../../../utils/logger';
import { parseProtoStrict } from '../../../utils/proto';
import type { Workflow, ExecutionViewportSettings } from '../types';
import { actionTypeToNodeType, type ActionDefinition, type NodeWithAction } from '../../../domains/workflows/utils/normalizers';
import { extractExecutionViewport, isPlainObject, sanitizeViewportSettings } from './viewport';
import { parseDate, ensureArray } from './normalization';

export const PREVIEW_DATA_KEYS = ['previewScreenshot', 'previewScreenshotCapturedAt', 'previewScreenshotSourceUrl'];

export const TRANSIENT_NODE_KEYS = new Set([
  'selected', 'dragging', 'draggingPosition', 'draggingAngle', 'positionAbsolute',
  'widthInitialized', 'heightInitialized', 'width', 'height', 'resizing', 'measured',
  'handleBounds', 'internalsSymbol', 'dataInternals', 'z', '__rf', '_rf', 'selectedHandles',
]);

export const TRANSIENT_EDGE_KEYS = new Set([
  'selected', 'dragging', 'draggingPosition', 'sourceX', 'sourceY', 'targetX', 'targetY',
  'sourceHandleBounds', 'targetHandleBounds', '__rf', '_rf',
]);

const deepClone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

export const stripWorkflowPreviewData = (nodes: Node[] | undefined | null): Node[] => {
  if (!nodes?.length) return [];
  return nodes.map((node) => {
    const data = node?.data as Record<string, unknown> | undefined;
    if (!data || typeof data !== 'object') return node;
    const cleaned = Object.fromEntries(Object.entries(data).filter(([key]) => !PREVIEW_DATA_KEYS.includes(key)));
    return Object.keys(cleaned).length === Object.keys(data).length ? node : { ...node, data: cleaned };
  });
};

export const workflowDefinitionToCanvas = (raw: unknown): { nodes: NodeWithAction[]; edges: Edge[] } => {
  const definition = isPlainObject(raw) ? raw : {};
  const rawNodes = Array.isArray(definition.nodes) ? definition.nodes : [];
  const nodes = rawNodes.map((value, index) => {
    const node = isPlainObject(value) ? value : {};
    const id = node.id ? String(node.id) : `node-${index + 1}`;
    const position = isPlainObject(node.position) ? node.position : {};
    const action = isPlainObject(node.action) ? node.action as ActionDefinition : undefined;
    const derivedType = action?.type ? actionTypeToNodeType(action.type) : 'unknown';
    return {
      ...node,
      id,
      type: derivedType !== 'unknown' ? derivedType : node.type ? String(node.type) : 'unknown',
      position: {
        x: Number(position.x ?? 100 + index * 200) || 0,
        y: Number(position.y ?? 100 + index * 120) || 0,
      },
      data: isPlainObject(node.data) ? node.data : {},
      action,
    } as NodeWithAction;
  });
  const edges = (Array.isArray(definition.edges) ? definition.edges : []).flatMap((value, index) => {
    if (!isPlainObject(value)) return [];
    const source = value.source ? String(value.source) : '';
    const target = value.target ? String(value.target) : '';
    if (!source || !target) return [];
    const edge = {
      ...value,
      id: value.id ? String(value.id) : `edge-${index + 1}`,
      source,
      target,
      sourceHandle: value.sourceHandle ?? value.source_handle,
      targetHandle: value.targetHandle ?? value.target_handle,
    } as Edge;
    delete (edge as Edge & Record<string, unknown>).source_handle;
    delete (edge as Edge & Record<string, unknown>).target_handle;
    const data = isPlainObject(value.data) ? value.data : undefined;
    if (data) edge.data = data;
    const condition = typeof data?.condition === 'string' ? data.condition : '';
    const labels: Record<string, [string, string]> = {
      if_true: ['IF TRUE', '#4ade80'], if_false: ['IF FALSE', '#f87171'],
      loop_body: ['LOOP BODY', '#38bdf8'], loop_next: ['AFTER LOOP', '#7c3aed'],
      loop_continue: ['CONTINUE', '#22c55e'], loop_break: ['BREAK', '#f43f5e'],
    };
    if (labels[condition]) {
      edge.label = labels[condition][0];
      edge.style = { ...(edge.style ?? {}), stroke: labels[condition][1] };
    }
    return [edge];
  });
  return { nodes: stripWorkflowPreviewData(nodes), edges };
};

const cleanNode = (node: Node): Record<string, unknown> => {
  const item = node as Node & Record<string, unknown>;
  const action = item.action as ActionDefinition | undefined;
  if (!action || typeof action !== 'object' || typeof action.type !== 'string') {
    throw new Error(`workflow node ${item.id ?? '<unknown>'} is missing a V2 action`);
  }
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(item)) {
    if (key === 'type' || key === 'data' || TRANSIENT_NODE_KEYS.has(key) || typeof value === 'function') continue;
    if (key === 'position' && isPlainObject(value)) {
      result.position = { x: Number(value.x ?? 0) || 0, y: Number(value.y ?? 0) || 0 };
    } else result[key] = key === 'action' && isPlainObject(value) ? deepClone(value) : value;
  }
  result.id ??= item.id;
  result.position ??= { x: Number(node.position?.x ?? 0) || 0, y: Number(node.position?.y ?? 0) || 0 };
  result.action = deepClone(action);
  return deepClone(result);
};

const cleanEdge = (edge: Edge): Record<string, unknown> => {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(edge as Edge & Record<string, unknown>)) {
    if (TRANSIENT_EDGE_KEYS.has(key) || typeof value === 'function') continue;
    if (key === 'sourceHandle') { result.source_handle = value; continue; }
    if (key === 'targetHandle') { result.target_handle = value; continue; }
    result[key] = ['data', 'markerEnd', 'markerStart', 'style'].includes(key) && value && typeof value === 'object'
      ? deepClone(value)
      : value;
  }
  result.id ??= edge.id;
  result.source ??= edge.source;
  result.target ??= edge.target;
  return deepClone(result);
};

export const canvasToWorkflowDefinition = (
  baseValue: unknown,
  nodes: Node[] = [],
  edges: Edge[] = [],
  viewport?: ExecutionViewportSettings,
): Record<string, unknown> & { nodes: unknown[]; edges: unknown[] } => {
  const base = isPlainObject(baseValue) ? baseValue : {};
  const next = Object.fromEntries(Object.entries(base).filter(([key]) => !['nodes', 'edges', 'settings'].includes(key)));
  next.nodes = stripWorkflowPreviewData(nodes).map(cleanNode);
  next.edges = edges.map(cleanEdge);
  if (isPlainObject(base.metadata)) next.metadata = base.metadata;
  const settings = isPlainObject(base.settings) ? { ...base.settings } : {};
  const cleanViewport = sanitizeViewportSettings(viewport);
  if (cleanViewport) {
    settings.executionViewport = cleanViewport;
    settings.viewport_width = cleanViewport.width;
    settings.viewport_height = cleanViewport.height;
  } else delete settings.executionViewport;
  if (Object.keys(settings).length) next.settings = settings;
  return next as Record<string, unknown> & { nodes: unknown[]; edges: unknown[] };
};

/** Encode the canonical JSON representation as the generated V2 protobuf message. */
export const workflowDefinitionToProto = (raw: unknown): WorkflowDefinitionV2 =>
  fromJson(WorkflowDefinitionV2Schema, raw as Parameters<typeof fromJson>[1], { ignoreUnknownFields: true });

/** Decode the generated V2 message while retaining JSON forward fields in the editor projection. */
export const workflowDefinitionFromProto = (message: WorkflowDefinitionV2): Record<string, unknown> =>
  toJson(WorkflowDefinitionV2Schema, message, { useProtoFieldName: true }) as Record<string, unknown>;

export const workflowDataToCanvas = (workflow: unknown): Workflow => {
  const data = isPlainObject(workflow) ? workflow : {};
  const definition = isPlainObject(data.flow_definition) ? data.flow_definition
    : isPlainObject(data.flowDefinition) ? data.flowDefinition : {};
  const source = { ...definition };
  const nodes = Array.isArray(data.nodes) ? data.nodes : definition.nodes;
  const edges = Array.isArray(data.edges) ? data.edges : definition.edges;
  const canvas = workflowDefinitionToCanvas({ ...source, nodes, edges });
  const rawSettings = isPlainObject(source.settings) ? source.settings : {};
  const executionViewport = sanitizeViewportSettings(
    extractExecutionViewport(source) ?? {
      width: Number(rawSettings.viewport_width ?? rawSettings.viewportWidth),
      height: Number(rawSettings.viewport_height ?? rawSettings.viewportHeight),
    },
  );
  const flowDefinition = canvasToWorkflowDefinition(source, canvas.nodes, canvas.edges, executionViewport);
  const versionValue = data.version;
  const version = typeof versionValue === 'number' ? versionValue : parseInt(String(versionValue ?? '1'), 10) || 1;
  return {
    ...data,
    id: data.id ?? '', projectId: data.project_id ?? data.projectId,
    name: data.name ?? '', description: data.description ?? '',
    folderPath: data.folder_path ?? data.folderPath ?? '/',
    nodes: canvas.nodes, edges: canvas.edges, tags: ensureArray<string>(data.tags), version,
    lastChangeSource: data.last_change_source ?? data.lastChangeSource ?? 'manual',
    lastChangeDescription: data.last_change_description ?? data.lastChangeDescription ?? '',
    createdAt: parseDate(data.created_at ?? data.createdAt),
    updatedAt: parseDate(data.updated_at ?? data.updatedAt), flowDefinition, executionViewport,
  } as unknown as Workflow;
};

export const workflowPayloadToCanvas = (payload: unknown, action: string): Workflow => {
  let parsed: Workflow | null = null;
  try {
    const response = parseProtoStrict<GetWorkflowResponse>(GetWorkflowResponseSchema, payload);
    if (response.workflow) parsed = workflowDataToCanvas(response.workflow);
  } catch (error) {
    logger.error('Failed to parse get workflow proto', { component: 'WorkflowStore', action }, error);
  }
  if (!parsed) {
    try {
      const summary = parseProtoStrict<WorkflowSummary>(WorkflowSummarySchema, payload);
      parsed = workflowDataToCanvas(summary);
    } catch (error) {
      logger.error('Failed to parse workflow summary proto', { component: 'WorkflowStore', action }, error);
    }
  }
  if (!parsed && isPlainObject(payload)) parsed = workflowDataToCanvas(isPlainObject(payload.workflow) ? payload.workflow : payload);
  if (!parsed) throw new Error('Failed to parse workflow payload');
  return parsed;
};
