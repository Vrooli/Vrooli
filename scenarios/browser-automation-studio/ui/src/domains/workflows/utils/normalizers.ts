import type { Edge, Node } from 'reactflow';
import { scenariosClient } from '@/api/scenarios';
import { fromJson, toJson } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  type ActionDefinition as ProtoActionDefinition,
  type ActionMetadata as ProtoActionMetadata,
  type AssertParams as ProtoAssertParams,
  type BlurParams as ProtoBlurParams,
  type ClickParams as ProtoClickParams,
  type ConditionalParams as ProtoConditionalParams,
  type CookieStorageParams as ProtoCookieStorageParams,
  type DownloadParams as ProtoDownloadParams,
  type DragDropParams as ProtoDragDropParams,
  type EvaluateParams as ProtoEvaluateParams,
  type ExtractParams as ProtoExtractParams,
  type FocusParams as ProtoFocusParams,
  type FrameSwitchParams as ProtoFrameSwitchParams,
  type GestureParams as ProtoGestureParams,
  type HoverParams as ProtoHoverParams,
  type InputParams as ProtoInputParams,
  type KeyboardParams as ProtoKeyboardParams,
  type LoopParams as ProtoLoopParams,
  type NavigateParams as ProtoNavigateParams,
  type NetworkMockParams as ProtoNetworkMockParams,
  type RotateParams as ProtoRotateParams,
  type ScreenshotParams as ProtoScreenshotParams,
  type ScrollParams as ProtoScrollParams,
  type SelectParams as ProtoSelectParams,
  type SetVariableParams as ProtoSetVariableParams,
  type ShortcutParams as ProtoShortcutParams,
  type SubflowParams as ProtoSubflowParams,
  type TabSwitchParams as ProtoTabSwitchParams,
  type UploadFileParams as ProtoUploadFileParams,
  type WaitParams as ProtoWaitParams,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

type ActionTypeName = Exclude<keyof typeof ActionType, `${number}`>;
type ActionTypeValueFor<T extends ActionTypeName> = `ACTION_TYPE_${T}`;

/** Names shown in persisted proto JSON, derived directly from the generated enum. */
export const ACTION_TYPES = Object.fromEntries(
  Object.keys(ActionType)
    .filter((name): name is ActionTypeName => Number.isNaN(Number(name)))
    .map((name) => [name, `ACTION_TYPE_${name}`]),
) as { [T in ActionTypeName]: ActionTypeValueFor<T> };

export type ActionTypeValue = (typeof ACTION_TYPES)[keyof typeof ACTION_TYPES];

type JsonParamValue<Value> = Value extends number
  ? number | string
  : Value extends readonly (infer Item)[]
    ? JsonParamValue<Item>[]
    : Value extends object
      ? { [Key in keyof Value as Key extends '$typeName' | '$unknown' ? never : Key]?: JsonParamValue<Value[Key]> }
      : Value;
type JsonParams<Value> = Partial<{
  [Key in keyof Value as Key extends '$typeName' | '$unknown' ? never : Key]: JsonParamValue<Value[Key]>;
}>;

type EditorParams<Schema, Compatibility extends object = {}> =
  Omit<JsonParams<Schema>, keyof Compatibility> & Compatibility;

export type NavigateParams = EditorParams<ProtoNavigateParams, {
  timeoutMs?: number; waitUntil?: string; destinationType?: string; scenario?: string; scenarioPath?: string;
}>;
export type ClickParams = EditorParams<ProtoClickParams, {
  selector?: string; button?: string; modifiers?: string[]; timeoutMs?: number;
}>;
export type InputParams = EditorParams<ProtoInputParams, { selector?: string; value?: string; delayMs?: number }>;
export type WaitParams = EditorParams<ProtoWaitParams, {
  selector?: string; durationMs?: number; state?: string; timeoutMs?: number;
}>;
export type AssertParams = EditorParams<ProtoAssertParams, {
  selector?: string; mode?: string; expected?: unknown; timeoutMs?: number;
}>;
export type ScrollParams = EditorParams<ProtoScrollParams, { selector?: string; behavior?: string }>;
export type SelectParams = EditorParams<ProtoSelectParams, {
  selector?: string; value?: string; label?: string; index?: number; timeoutMs?: number;
}>;
export type EvaluateParams = EditorParams<ProtoEvaluateParams>;
export type KeyboardParams = EditorParams<ProtoKeyboardParams, { modifiers?: string[]; action?: string }>;
export type HoverParams = EditorParams<ProtoHoverParams, { selector?: string; timeoutMs?: number }>;
export type ScreenshotParams = EditorParams<ProtoScreenshotParams>;
export type FocusParams = EditorParams<ProtoFocusParams, { selector?: string; timeoutMs?: number }>;
export type BlurParams = EditorParams<ProtoBlurParams, { selector?: string; timeoutMs?: number }>;
export type SubflowParams = EditorParams<ProtoSubflowParams, {
  workflowId?: string; workflowPath?: string; workflowVersion?: number; parameters?: Record<string, unknown>;
}>;
export type ExtractParams = EditorParams<ProtoExtractParams>;
export type UploadFileParams = EditorParams<ProtoUploadFileParams>;
export type DownloadParams = EditorParams<ProtoDownloadParams>;
export type FrameSwitchParams = EditorParams<ProtoFrameSwitchParams>;
export type TabSwitchParams = EditorParams<ProtoTabSwitchParams>;
export type CookieStorageParams = EditorParams<ProtoCookieStorageParams>;
export type ShortcutParams = EditorParams<ProtoShortcutParams>;
export type DragDropParams = EditorParams<ProtoDragDropParams>;
export type GestureParams = EditorParams<ProtoGestureParams>;
export type NetworkMockParams = EditorParams<ProtoNetworkMockParams>;
export type RotateParams = EditorParams<ProtoRotateParams>;
export type SetVariableParams = EditorParams<ProtoSetVariableParams, {
  name?: string; value?: string; sourceType?: string; valueType?: string; expression?: string;
  selector?: string; extractType?: string; attribute?: string; storeAs?: string; timeoutMs?: number;
  allMatches?: boolean; transform?: string; required?: boolean;
}>;
export type LoopParams = EditorParams<ProtoLoopParams, {
  loopType?: string; arraySource?: string; count?: number; maxIterations?: number; itemVariable?: string;
  indexVariable?: string; conditionType?: string; conditionVariable?: string; conditionOperator?: string;
  conditionValue?: string; conditionExpression?: string; iterationTimeoutMs?: number; totalTimeoutMs?: number;
}>;
export type ConditionalParams = EditorParams<ProtoConditionalParams, {
  conditionType?: string; expression?: string; selector?: string; variable?: string; operator?: string;
  value?: string; negate?: boolean; timeoutMs?: number; pollIntervalMs?: number;
}>;
export type ActionMetadata = JsonParams<ProtoActionMetadata>;

type ProtoParams = NonNullable<ProtoActionDefinition['params']>;
type ProtoParamCase = Exclude<ProtoParams['case'], undefined>;
type ProtoParamValue<Case extends ProtoParamCase> = Extract<ProtoParams, { case: Case }>['value'];
type GeneratedParamsByCase = { [Case in ProtoParamCase]?: JsonParams<ProtoParamValue<Case>> };
type EditorCompatibilityParams = {
  navigate?: NavigateParams; click?: ClickParams; input?: InputParams; wait?: WaitParams;
  assert?: AssertParams; scroll?: ScrollParams; selectOption?: SelectParams; keyboard?: KeyboardParams;
  hover?: HoverParams; focus?: FocusParams; blur?: BlurParams; subflow?: SubflowParams;
  setVariable?: SetVariableParams; loop?: LoopParams; conditional?: ConditionalParams;
};
type ParamsByCase = Omit<GeneratedParamsByCase, keyof EditorCompatibilityParams> & EditorCompatibilityParams;
export type ActionParamsField = keyof ParamsByCase;
export type ActionDefinition = { type: ActionTypeValue; metadata?: ActionMetadata } & ParamsByCase;

const ACTION_PARAMS_FIELDS = new Map<string, ActionParamsField>(
  ActionDefinitionSchema.fields.flatMap((field) => {
    if (field.oneof?.name !== 'params' || field.fieldKind !== 'message') return [];

    const paramsMessageName = field.message.typeName.split('.').pop() ?? '';
    const actionName = paramsMessageName
      .replace(/Params$/, '')
      .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
      .toUpperCase();
    return [[`ACTION_TYPE_${actionName}`, field.localName as ActionParamsField]];
  }),
);

export function getActionParamsFieldName(actionType: ActionTypeValue): ActionParamsField | null {
  return ACTION_PARAMS_FIELDS.get(actionType) ?? null;
}

/** Maps legacy React Flow node names to the generated V2 ActionType vocabulary. */
export function nodeTypeToActionType(nodeType: string): ActionTypeValue {
  const normalized = nodeType.toLowerCase().trim().replace(/[^a-z0-9]/g, '');
  const aliases: Partial<Record<string, ActionTypeName>> = {
    goto: 'NAVIGATE', type: 'INPUT', fill: 'INPUT', eval: 'EVALUATE', script: 'EVALUATE',
    keypress: 'KEYBOARD', setvar: 'SET_VARIABLE', usevariable: 'SET_VARIABLE', usevar: 'SET_VARIABLE',
    if: 'CONDITIONAL', branch: 'CONDITIONAL', upload: 'UPLOAD_FILE', frame: 'FRAME_SWITCH', tab: 'TAB_SWITCH',
    setcookie: 'COOKIE_STORAGE', getcookie: 'COOKIE_STORAGE', clearcookie: 'COOKIE_STORAGE',
    setstorage: 'COOKIE_STORAGE', getstorage: 'COOKIE_STORAGE', clearstorage: 'COOKIE_STORAGE',
    cookie: 'COOKIE_STORAGE', storage: 'COOKIE_STORAGE', drag: 'DRAG_DROP', mock: 'NETWORK_MOCK',
  };
  const actionName = aliases[normalized] ?? Object.keys(ACTION_TYPES).find(
    (name) => name.toLowerCase().replace(/_/g, '') === normalized,
  ) as ActionTypeName | undefined;
  return actionName ? ACTION_TYPES[actionName] : ACTION_TYPES.UNSPECIFIED;
}

export function actionTypeToNodeType(actionType: ActionTypeValue): string {
  if (actionType === ACTION_TYPES.UNSPECIFIED || !Object.values(ACTION_TYPES).includes(actionType)) {
    return 'unknown';
  }
  return actionType.replace(/^ACTION_TYPE_/, '').toLowerCase().replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

/** Create flat proto-JSON action data by validating fields against generated message metadata. */
export function buildActionDefinition(nodeType: string, data: Record<string, unknown> = {}): ActionDefinition {
  const type = nodeTypeToActionType(nodeType);
  const paramsField = getActionParamsFieldName(type);
  const descriptor = paramsField
    ? ActionDefinitionSchema.fields.find((field) => field.oneof?.name === 'params' && field.localName === paramsField)
    : undefined;
  const payload = descriptor?.fieldKind === 'message'
    ? Object.fromEntries(Object.entries(data).filter(([key, value], index, entries) => {
        const field = descriptor.message.fields.find((candidate) => candidate.localName === key);
        if (!field || value === undefined) return false;
        if (!field.oneof) return true;
        return entries.findIndex(([candidateKey, candidateValue]) =>
          candidateValue !== undefined && descriptor.message.fields.some((candidate) =>
            candidate.localName === candidateKey && candidate.oneof?.name === field.oneof?.name)) === index;
      }))
    : undefined;
  const label = typeof data.label === 'string' ? data.label : undefined;
  const json = {
    type,
    ...(paramsField && payload ? { [paramsField]: payload } : {}),
    ...(label ? { metadata: { label } } : {}),
  };
  return toJson(ActionDefinitionSchema, fromJson(ActionDefinitionSchema, json, { ignoreUnknownFields: true })) as unknown as ActionDefinition;
}

export function getNodeTypeFromAction(action: ActionDefinition | undefined): string {
  return action?.type ? actionTypeToNodeType(action.type) : 'navigate';
}

export function hasValidAction(node: { action?: ActionDefinition }): boolean {
  return !!node.action?.type && node.action.type !== ACTION_TYPES.UNSPECIFIED;
}

/**
 * Extended Node type that includes the V2 action field.
 */
export type NodeWithAction = Node & { action?: ActionDefinition };

/**
 * Normalizes raw node data from API/storage into valid ReactFlow Node objects.
 * Ensures all required fields exist with proper defaults.
 *
 * V2 Native Format:
 * - If node.action exists, it is the source of truth
 * - node.type is derived from action.type for ReactFlow routing
 * - A missing action remains invalid input; workflow compatibility conversion
 *   belongs at the API ingress boundary, never in the editor.
 */
export const normalizeNodes = (nodes: unknown[] | undefined | null): NodeWithAction[] => {
  if (!Array.isArray(nodes)) return [];
  return nodes.map((node, index) => {
    const nodeData = node as Record<string, unknown>;
    const id = nodeData?.id ? String(nodeData.id) : `node-${index + 1}`;
    const positionData = nodeData?.position as Record<string, unknown> | undefined;
    const position = {
      x: Number(positionData?.x ?? 100 + index * 200) || 0,
      y: Number(positionData?.y ?? 100 + index * 120) || 0,
    };
    const data = nodeData?.data && typeof nodeData.data === 'object' ? nodeData.data : {};

    // Get existing action if present
    const action = nodeData?.action && typeof nodeData.action === 'object'
      ? (nodeData.action as ActionDefinition)
      : undefined;

    // React Flow needs a renderer key, but the action remains authoritative.
    let type = nodeData?.type ? String(nodeData.type) : 'unknown';

    // V2 Native: If action exists, derive type from action.type
    if (action?.type) {
      const derivedType = actionTypeToNodeType(action.type);
      if (derivedType !== 'unknown') {
        type = derivedType;
      }
    }

    const result: NodeWithAction = {
      ...nodeData,
      id,
      type,
      position,
      data,
      action,
    } as NodeWithAction;

    return result;
  });
};

/**
 * Normalizes raw edge data from API/storage into valid ReactFlow Edge objects.
 * Filters out invalid edges that are missing source or target.
 */
export const normalizeEdges = (edges: unknown[] | undefined | null): Edge[] => {
  if (!Array.isArray(edges)) return [];
  return edges
    .map((edge, index) => {
      const edgeData = edge as Record<string, unknown>;
      const id = edgeData?.id ? String(edgeData.id) : `edge-${index + 1}`;
      const source = edgeData?.source ? String(edgeData.source) : '';
      const target = edgeData?.target ? String(edgeData.target) : '';
      if (!source || !target) return null;
      const normalized: Edge = {
        ...edgeData,
        id,
        source,
        target,
      } as Edge;
      const data = (edgeData?.data && typeof edgeData.data === 'object') ? edgeData.data as Record<string, unknown> : undefined;
      if (data) {
        normalized.data = data;
      }
      const condition = typeof data?.condition === 'string' ? data.condition : undefined;
      if (condition === 'if_true' || condition === 'if_false') {
        const stroke = condition === 'if_true' ? '#4ade80' : '#f87171';
        normalized.label = condition === 'if_true' ? 'IF TRUE' : 'IF FALSE';
        normalized.style = { ...(normalized.style ?? {}), stroke };
      }
      if (condition === 'loop_body') {
        normalized.label = 'LOOP BODY';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#38bdf8' };
      }
      if (condition === 'loop_next') {
        normalized.label = 'AFTER LOOP';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#7c3aed' };
      }
      if (condition === 'loop_continue') {
        normalized.label = 'CONTINUE';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#22c55e' };
      }
      if (condition === 'loop_break') {
        normalized.label = 'BREAK';
        normalized.style = { ...(normalized.style ?? {}), stroke: '#f43f5e' };
      }
      return normalized;
    })
    .filter(Boolean) as Edge[];
};

/**
 * Automatically layouts nodes if they don't have valid positions.
 * Uses a simple BFS-based layering algorithm.
 */
export const autoLayoutNodes = (nodes: Node[], edges: Edge[]): Node[] => {
  if (nodes.length === 0) return [];

  // Check if we need to layout: if all nodes are at (0,0) or very close, we assume they need layout.
  // Or if the user specifically requested "optional positions", we can check if positions are missing in the raw data.
  // Since normalizeNodes supplies default (0,0) or index-based positions, we might need a better heuristic.
  // For now, let's assume if the first few nodes are all at x=0, we should layout.
  // Actually, normalizeNodes gives `100 + index * 200` for X.
  // Let's check if the "raw" positions were missing. But we don't have raw data here.
  // We can rely on a heuristic: if all nodes have y=0 (which normalizeNodes does NOT do by default, it does `100 + index * 120`),
  // checking for the default pattern from normalizeNodes might be tricky if we want to support "some positions present".
  //
  // However, the requirement is: "remove them from all workflows... and add logic which properly spaces them automatically when position data isn't provided."
  // If we strip positions, normalizeNodes will assign the diagonal layout:
  // x: 100 + index * 200
  // y: 100 + index * 120
  // We can detect this specific pattern or just always run auto-layout if we detect "default-like" positions.
  //
  // Better approach: Let's just run the layout algorithm. It's deterministic.
  // If the nodes already have good positions, we might not want to overwrite them.
  // But how do we know?
  //
  // Let's look at `normalizeNodes` again.
  // It assigns defaults if `positionData` is missing.
  //
  // To be safe and explicit, maybe we should export a function that takes the RAW data?
  // But `WorkflowBuilder` calls `normalizeNodes` then `normalizeEdges`.
  //
  // Let's implement a layout that respects existing non-default positions?
  // Or, simpler: If we detect that the nodes are in the "default diagonal" (which is what normalizeNodes does when pos is missing), we re-layout.
  //
  // Default diagonal: x = 100 + i*200, y = 100 + i*120.
  // Let's check if the nodes follow this pattern.

  const isDefaultLayout = nodes.every((node, i) => {
    return node.position.x === 100 + i * 200 && node.position.y === 100 + i * 120;
  });

  // Also check if all are at 0,0 (legacy default)
  const isZeroLayout = nodes.every(n => n.position.x === 0 && n.position.y === 0);

  if (!isDefaultLayout && !isZeroLayout) {
    return nodes;
  }

  // Build adjacency list
  const adj = new Map<string, string[]>();
  const inDegree = new Map<string, number>();

  nodes.forEach(node => {
    adj.set(node.id, []);
    inDegree.set(node.id, 0);
  });

  edges.forEach(edge => {
    if (adj.has(edge.source) && adj.has(edge.target)) {
      adj.get(edge.source)?.push(edge.target);
      inDegree.set(edge.target, (inDegree.get(edge.target) || 0) + 1);
    }
  });

  // BFS for levels
  const levels = new Map<string, number>();
  const queue: string[] = [];

  // Find roots (in-degree 0)
  nodes.forEach(node => {
    if ((inDegree.get(node.id) || 0) === 0) {
      levels.set(node.id, 0);
      queue.push(node.id);
    }
  });

  // If no roots (cycle?), pick the first one
  if (queue.length === 0 && nodes.length > 0) {
    const firstNode = nodes[0];
    if (firstNode) {
      levels.set(firstNode.id, 0);
      queue.push(firstNode.id);
    }
  }

  const visited = new Set<string>(queue);

  while (queue.length > 0) {
    const currId = queue.shift();
    if (currId === undefined) continue;
    const currLevel = levels.get(currId) ?? 0;

    const neighbors = adj.get(currId) || [];
    for (const nextId of neighbors) {
      if (!visited.has(nextId)) {
        visited.add(nextId);
        levels.set(nextId, currLevel + 1);
        queue.push(nextId);
      } else {
        // If already visited, we might want to push it deeper if this path is longer?
        // For simple tree-like, max level is better.
        const existingLevel = levels.get(nextId) ?? 0;
        if (existingLevel < currLevel + 1) {
          levels.set(nextId, currLevel + 1);
          // If we update level, we might need to re-process children? 
          // For a simple DAG, topological sort is better, but this BFS is "okay" for simple flows.
          // Let's stick to simple BFS for now to avoid infinite loops in cycles.
        }
      }
    }
  }

  // Group by level
  const levelGroups = new Map<number, Node[]>();
  nodes.forEach(node => {
    const lvl = levels.get(node.id) ?? 0;
    if (!levelGroups.has(lvl)) {
      levelGroups.set(lvl, []);
    }
    levelGroups.get(lvl)?.push(node);
  });

  // Assign positions
  const HORIZONTAL_SPACING = 300;
  const VERTICAL_SPACING = 150;

  return nodes.map(node => {
    const lvl = levels.get(node.id) ?? 0;
    const nodesInLevel = levelGroups.get(lvl) ?? [];
    const indexInLevel = nodesInLevel.indexOf(node);

    return {
      ...node,
      position: {
        x: lvl * HORIZONTAL_SPACING,
        y: indexInLevel * VERTICAL_SPACING + 100, // +100 padding
      },
    };
  });
};

// Workflow graph lookups used by execution and editor hooks.

// Cache for resolved scenario URLs to avoid redundant API calls
const scenarioUrlCache = new Map<string, { url: string; timestamp: number }>();
const CACHE_DURATION_MS = 30000; // 30 seconds

export interface NodeScreenshot {
  dataUrl: string;
  nodeId: string;
  nodeType: string;
  capturedAt?: string;
  sourceUrl?: string;
}

const pickString = (obj: Record<string, unknown> | null | undefined, key: string): string | null => {
  if (!obj || typeof obj !== 'object') {
    return null;
  }
  const value = obj[key];
  return typeof value === 'string' && value.trim().length > 0 ? value : null;
};

const getNodeData = (node: Node): Record<string, unknown> => {
  const raw = node.data as unknown;
  if (raw && typeof raw === 'object') {
    return raw as Record<string, unknown>;
  }
  return {};
};

const extractScreenshotFromNode = (node: Node): NodeScreenshot | null => {
  const data = getNodeData(node);
  if (!data) {
    return null;
  }

  const candidateKeys = [
    'previewScreenshot',
    'screenshot',
    'screenshotDataUrl',
    'screenshotUrl',
    'previewImage',
  ];

  let screenshot: string | null = null;
  for (const key of candidateKeys) {
    screenshot = pickString(data, key);
    if (screenshot) break;
  }

  if (!screenshot) {
    const preview = data.preview as Record<string, unknown> | undefined;
    screenshot = pickString(preview, 'screenshot') ?? pickString(preview, 'image') ?? pickString(preview, 'dataUrl');
  }

  if (!screenshot) {
    const elementInfo = data.elementInfo as Record<string, unknown> | undefined;
    const elementScreenshot = elementInfo?.screenshot as Record<string, unknown> | string | undefined;
    if (typeof elementScreenshot === 'string') {
      screenshot = elementScreenshot;
    } else if (elementScreenshot && typeof elementScreenshot === 'object') {
      screenshot = pickString(elementScreenshot, 'dataUrl') ?? pickString(elementScreenshot, 'url');
    }
  }

  if (!screenshot || screenshot.trim().length === 0) {
    return null;
  }

  const capturedAt = pickString(data, 'previewScreenshotCapturedAt') ?? pickString(data, 'screenshotCapturedAt');
  const sourceUrl = pickString(data, 'previewScreenshotSourceUrl') ?? pickString(data, 'screenshotSourceUrl');

  return {
    dataUrl: screenshot,
    nodeId: node.id,
    nodeType: typeof node.type === 'string' ? node.type : 'unknown',
    capturedAt: capturedAt ?? undefined,
    sourceUrl: sourceUrl ?? undefined,
  };
};

/**
 * Checks if a node is a navigate node by examining its canonical action type.
 */
function isNavigateNode(node: Node): boolean {
  const action = (node as Node & { action?: ActionDefinition }).action;
  return action?.type === ACTION_TYPES.NAVIGATE;
}

/**
 * Resolves a scenario name (and optional path) to its actual URL
 */
async function resolveScenarioUrl(scenarioName: string, scenarioPath?: string): Promise<string | null> {
  const cacheKey = `${scenarioName}:${scenarioPath || ''}`;
  const cached = scenarioUrlCache.get(cacheKey);

  if (cached && Date.now() - cached.timestamp < CACHE_DURATION_MS) {
    return cached.url;
  }

  try {
    const info = await scenariosClient.getPort({ name: scenarioName });
    const baseUrl: string | undefined = info.url
      ? info.url
      : info.port > 0
        ? `http://localhost:${info.port}`
        : undefined;

    if (!baseUrl) {
      return null;
    }

    const trimmedPath = scenarioPath?.trim() || '';
    const resolvedUrl = trimmedPath
      ? `${baseUrl.replace(/\/+$/, '')}/${trimmedPath.replace(/^\/+/, '')}`
      : baseUrl;

    // Cache the result
    scenarioUrlCache.set(cacheKey, { url: resolvedUrl, timestamp: Date.now() });

    return resolvedUrl;
  } catch (_error) {
    return null;
  }
}

/**
 * Finds the most recent Navigate node upstream from a given node
 * by tracing back through the workflow connections
 */
export function findUpstreamNavigateNode(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): Node | null {
  const visited = new Set<string>();
  const queue: string[] = [nodeId];

  while (queue.length > 0) {
    const currentNodeId = queue.shift();
    if (currentNodeId === undefined) continue;

    if (visited.has(currentNodeId)) {
      continue;
    }
    visited.add(currentNodeId);

    // Find the current node
    const currentNode = nodes.find(n => n.id === currentNodeId);
    if (!currentNode) {
      continue;
    }

    // If this is a Navigate node and it's not the starting node, return it
    if (isNavigateNode(currentNode) && currentNodeId !== nodeId) {
      return currentNode;
    }

    // Find all edges that target this node (incoming edges)
    const incomingEdges = edges.filter(edge => edge.target === currentNodeId);
    
    // Add all source nodes to the queue for traversal
    for (const edge of incomingEdges) {
      if (!visited.has(edge.source)) {
        queue.push(edge.source);
      }
    }
  }

  return null;
}

export function getUpstreamScreenshot(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): NodeScreenshot | null {
  if (!nodeId) {
    return null;
  }

  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  const visited = new Set<string>();
  const queue: string[] = [nodeId];

  while (queue.length > 0) {
    const currentId = queue.shift();
    if (currentId === undefined) continue;
    if (visited.has(currentId)) {
      continue;
    }
    visited.add(currentId);

    const currentNode = nodeMap.get(currentId);
    if (!currentNode) {
      continue;
    }

    if (currentId !== nodeId) {
      const screenshot = extractScreenshotFromNode(currentNode);
      if (screenshot) {
        return screenshot;
      }
    }

    const incomingEdges = edges.filter((edge) => edge.target === currentId);
    for (const edge of incomingEdges) {
      if (!visited.has(edge.source)) {
        queue.push(edge.source);
      }
    }
  }

  return null;
}

/**
 * Gets the URL from a Navigate node's data (synchronous version)
 * For scenario navigation, returns a placeholder URL
 * Use getNavigateNodeUrlAsync for actual resolution
 */
export function getNavigateNodeUrl(node: Node): string | null {
  if (!isNavigateNode(node)) {
    return null;
  }

  // Check V2 action format first
  const action = (node as Node & { action?: ActionDefinition }).action;
  if (action?.navigate?.url) {
    return action.navigate.url;
  }

  const nodeData = getNodeData(node);
  const destinationType = pickString(nodeData, 'destinationType') ?? '';
  const scenarioName = pickString(nodeData, 'scenario') ?? pickString(nodeData, 'scenarioName') ?? '';

  // If it's a scenario navigation, return a placeholder
  if (destinationType === 'scenario' || (scenarioName && !pickString(nodeData, 'url'))) {
    if (!scenarioName) {
      return null;
    }

    const scenarioPath = pickString(nodeData, 'scenarioPath') ?? '';
    return scenarioPath
      ? `scenario://${scenarioName}${scenarioPath}`
      : `scenario://${scenarioName}`;
  }

  // Regular URL navigation (legacy format)
  const rawUrl = pickString(nodeData, 'url');
  if (!rawUrl) {
    return null;
  }
  return rawUrl;
}

/**
 * Gets the URL from a Navigate node's data (async version)
 * Resolves scenario URLs to actual HTTP URLs
 */
export async function getNavigateNodeUrlAsync(node: Node): Promise<string | null> {
  if (!isNavigateNode(node)) {
    return null;
  }

  // Check V2 action format first
  const action = (node as Node & { action?: ActionDefinition }).action;
  if (action?.navigate?.url) {
    return action.navigate.url;
  }

  const nodeData = getNodeData(node);
  const destinationType = pickString(nodeData, 'destinationType') ?? '';
  const scenarioName = pickString(nodeData, 'scenario') ?? pickString(nodeData, 'scenarioName') ?? '';

  // If it's a scenario navigation, resolve the URL
  if (destinationType === 'scenario' || (scenarioName && !pickString(nodeData, 'url'))) {
    if (!scenarioName) {
      return null;
    }

    const scenarioPath = pickString(nodeData, 'scenarioPath') ?? '';
    return await resolveScenarioUrl(scenarioName, scenarioPath);
  }

  // Regular URL navigation (legacy format)
  const rawUrl = pickString(nodeData, 'url');
  if (!rawUrl) {
    return null;
  }
  return rawUrl;
}

/**
 * Gets the upstream URL for a node by finding its Navigate node (synchronous)
 * Returns placeholder URLs for scenario navigation
 */
export function getUpstreamUrl(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): string | null {
  const navigateNode = findUpstreamNavigateNode(nodeId, nodes, edges);
  if (!navigateNode) {
    return null;
  }
  return getNavigateNodeUrl(navigateNode);
}

/**
 * Gets the upstream URL for a node by finding its Navigate node (async)
 * Resolves scenario URLs to actual HTTP URLs
 */
export async function getUpstreamUrlAsync(
  nodeId: string,
  nodes: Node[],
  edges: Edge[]
): Promise<string | null> {
  const navigateNode = findUpstreamNavigateNode(nodeId, nodes, edges);
  if (!navigateNode) {
    return null;
  }
  return await getNavigateNodeUrlAsync(navigateNode);
}

/** Finds all entry nodes in a workflow (nodes with no incoming edges). */
export function findEntryNodes(nodes: Node[], edges: Edge[]): Node[] {
  const nodesWithIncomingEdges = new Set(edges.map((edge) => edge.target));
  return nodes.filter((node) => !nodesWithIncomingEdges.has(node.id));
}

/** Checks whether every workflow entry node is a navigate step. */
export function workflowStartsWithNavigate(nodes: Node[], edges: Edge[]): boolean {
  if (nodes.length === 0) return false;
  const entryNodes = findEntryNodes(nodes, edges);
  if (entryNodes.length === 0) return false;
  return entryNodes.every((node) => {
    const action = (node as Node & { action?: ActionDefinition }).action;
    return action?.type === ACTION_TYPES.NAVIGATE;
  });
}
