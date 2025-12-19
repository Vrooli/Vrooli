import type { Node, Edge } from 'reactflow';
import type { ExecutionViewportSettings } from '../types';
import type { ActionDefinition } from '../../../utils/actionBuilder';
import { isPlainObject } from './viewport';

// ============================================================================
// Constants
// ============================================================================

export const PREVIEW_DATA_KEYS = ['previewScreenshot', 'previewScreenshotCapturedAt', 'previewScreenshotSourceUrl'];

export const TRANSIENT_NODE_KEYS = new Set([
  'selected',
  'dragging',
  'draggingPosition',
  'draggingAngle',
  'positionAbsolute',
  'widthInitialized',
  'heightInitialized',
  'width',
  'height',
  'resizing',
  'measured',
  'handleBounds',
  'internalsSymbol',
  'dataInternals',
  'z',
  '__rf',
  '_rf',
  'selectedHandles',
]);

export const TRANSIENT_EDGE_KEYS = new Set([
  'selected',
  'dragging',
  'draggingPosition',
  'sourceX',
  'sourceY',
  'targetX',
  'targetY',
  'sourceHandleBounds',
  'targetHandleBounds',
  '__rf',
  '_rf',
]);

// ============================================================================
// Flow Definition Building
// ============================================================================

export const buildFlowDefinition = (
  rawDefinition: unknown,
  nodes: unknown[],
  edges: unknown[],
  viewport?: ExecutionViewportSettings,
): Record<string, unknown> => {
  const base = isPlainObject(rawDefinition) ? rawDefinition : {};
  const baseRecord = base as Record<string, unknown>;
  const next: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(baseRecord)) {
    if (key === 'nodes' || key === 'edges' || key === 'settings') {
      continue;
    }
    next[key] = value;
  }

  next.nodes = nodes;
  next.edges = edges;

  // Preserve/derive typed metadata if available
  if (isPlainObject(baseRecord.metadata_typed)) {
    next.metadata_typed = baseRecord.metadata_typed as Record<string, unknown>;
  } else if (isPlainObject(baseRecord.metadata)) {
    const meta = baseRecord.metadata as Record<string, unknown>;
    const derived: Record<string, unknown> = {};
    if (typeof meta.name === 'string') derived.name = meta.name;
    if (typeof meta.description === 'string') derived.description = meta.description;
    if (isPlainObject(meta.labels)) derived.labels = meta.labels as Record<string, string>;
    if (typeof meta.version === 'string') derived.version = meta.version;
    if (Object.keys(derived).length > 0) {
      next.metadata_typed = derived;
    }
  }

  let existingSettings: Record<string, unknown> = {};
  if (isPlainObject(baseRecord.settings)) {
    existingSettings = { ...(baseRecord.settings as Record<string, unknown>) };
  }

  if (viewport) {
    existingSettings.executionViewport = viewport;
  } else {
    delete existingSettings.executionViewport;
  }

  // Populate proto-typed settings alongside legacy settings
  if (viewport) {
    next.settings_typed = {
      viewport_width: viewport.width,
      viewport_height: viewport.height,
    };
  } else if (isPlainObject(baseRecord.settings_typed)) {
    next.settings_typed = baseRecord.settings_typed as Record<string, unknown>;
  }

  if (Object.keys(existingSettings).length > 0) {
    next.settings = existingSettings;
  }

  return next;
};

// ============================================================================
// Node Sanitization
// ============================================================================

export const stripPreviewDataFromNodes = (nodes: Node[] | undefined | null): Node[] => {
  if (!nodes || nodes.length === 0) {
    return [];
  }
  return nodes.map((node) => {
    if (!node || typeof node !== 'object') {
      return node;
    }
    const data = node.data as Record<string, unknown> | undefined;
    if (!data || typeof data !== 'object') {
      return node;
    }

    let modified = false;
    const cleanedData: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(data)) {
      if (PREVIEW_DATA_KEYS.includes(key)) {
        modified = true;
        continue;
      }
      cleanedData[key] = value;
    }

    if (!modified) {
      return node;
    }

    return {
      ...node,
      data: cleanedData,
    } as Node;
  });
};

export const sanitizeNodesForPersistence = (nodes: Node[] | undefined | null): unknown[] => {
  if (!nodes || nodes.length === 0) {
    return [];
  }

  return stripPreviewDataFromNodes(nodes).map((node) => {
    if (!node || typeof node !== 'object') {
      return node;
    }

    const nodeRecord = node as Node & Record<string, unknown>;
    const cleaned: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(nodeRecord)) {
      if (TRANSIENT_NODE_KEYS.has(key)) {
        continue;
      }
      if (typeof value === 'function') {
        continue;
      }
      if (key === 'data' && value && typeof value === 'object') {
        cleaned[key] = JSON.parse(JSON.stringify(value));
        continue;
      }
      // Preserve V2 action field for type-safe execution
      if (key === 'action' && value && typeof value === 'object') {
        cleaned[key] = JSON.parse(JSON.stringify(value));
        continue;
      }
      if (key === 'position' && value && typeof value === 'object') {
        const pos = value as Record<string, unknown>;
        cleaned[key] = {
          x: typeof pos.x === 'number' ? pos.x : Number(pos.x ?? 0) || 0,
          y: typeof pos.y === 'number' ? pos.y : Number(pos.y ?? 0) || 0,
        };
        continue;
      }
      cleaned[key] = value;
    }

    if (!('id' in cleaned)) {
      cleaned.id = nodeRecord.id;
    }
    if (!('type' in cleaned) && 'type' in nodeRecord) {
      cleaned.type = nodeRecord.type;
    }
    if (!('data' in cleaned)) {
      cleaned.data = {};
    }
    if (!('position' in cleaned)) {
      const pos = nodeRecord.position as unknown as Record<string, unknown> | undefined;
      cleaned.position = {
        x: typeof pos?.x === 'number' ? pos.x : Number(pos?.x ?? 0) || 0,
        y: typeof pos?.y === 'number' ? pos.y : Number(pos?.y ?? 0) || 0,
      };
    }

    // V2 Native: Preserve action field as-is (it's the source of truth)
    // The action is already present from normalization or component updates
    // No rebuild needed - this eliminates the type/data -> action conversion on every save
    const existingAction = (nodeRecord as Node & { action?: ActionDefinition }).action;
    if (existingAction && typeof existingAction === 'object') {
      cleaned.action = JSON.parse(JSON.stringify(existingAction));
    }

    return JSON.parse(JSON.stringify(cleaned));
  });
};

// ============================================================================
// Edge Sanitization
// ============================================================================

export const sanitizeEdgesForPersistence = (edges: Edge[] | undefined | null): unknown[] => {
  if (!edges || edges.length === 0) {
    return [];
  }

  return edges.map((edge) => {
    if (!edge || typeof edge !== 'object') {
      return edge;
    }

    const edgeRecord = edge as Edge & Record<string, unknown>;
    const cleaned: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(edgeRecord)) {
      if (TRANSIENT_EDGE_KEYS.has(key)) {
        continue;
      }
      if (typeof value === 'function') {
        continue;
      }
      if ((key === 'data' || key === 'markerEnd' || key === 'markerStart' || key === 'style') && value && typeof value === 'object') {
        cleaned[key] = JSON.parse(JSON.stringify(value));
        continue;
      }
      cleaned[key] = value;
    }

    if (!('id' in cleaned)) {
      cleaned.id = edgeRecord.id;
    }
    if (!('source' in cleaned) && 'source' in edgeRecord) {
      cleaned.source = edgeRecord.source;
    }
    if (!('target' in cleaned) && 'target' in edgeRecord) {
      cleaned.target = edgeRecord.target;
    }

    return JSON.parse(JSON.stringify(cleaned));
  });
};
