import { create } from 'zustand';
import { Node, Edge } from 'reactflow';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { normalizeNodes, normalizeEdges } from '../utils/workflowNormalizers';

const AUTOSAVE_DELAY_MS = 2500;
let autosaveTimeout: ReturnType<typeof setTimeout> | null = null;

type SaveErrorType = 'conflict' | 'network' | 'server';

export interface SaveErrorState {
  type: SaveErrorType;
  message: string;
  status?: number;
}

type AutosaveOptions = SaveWorkflowOptions & { debounceMs?: number };

export type ViewportPreset = 'desktop' | 'mobile' | 'custom';

export interface ExecutionViewportSettings {
  width: number;
  height: number;
  preset?: ViewportPreset;
}

const MIN_VIEWPORT_DIMENSION = 200;
const MAX_VIEWPORT_DIMENSION = 10000;

const isPlainObject = (value: unknown): value is Record<string, unknown> => {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
};

const clampDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION), MAX_VIEWPORT_DIMENSION);
};

const parseViewportDimension = (value: unknown): number | null => {
  if (typeof value === 'number') {
    return value > 0 ? clampDimension(value) : null;
  }
  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) {
      return null;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (Number.isNaN(parsed)) {
      return null;
    }
    return parsed > 0 ? clampDimension(parsed) : null;
  }
  return null;
};

const parseViewportPreset = (value: unknown): ViewportPreset | undefined => {
  if (typeof value !== 'string') {
    return undefined;
  }
  const normalized = value.trim().toLowerCase();
  if (normalized === 'desktop' || normalized === 'mobile' || normalized === 'custom') {
    return normalized;
  }
  return undefined;
};

const sanitizeViewportSettings = (viewport: ExecutionViewportSettings | undefined | null): ExecutionViewportSettings | undefined => {
  if (!viewport) {
    return undefined;
  }
  const width = parseViewportDimension((viewport as ExecutionViewportSettings).width);
  const height = parseViewportDimension((viewport as ExecutionViewportSettings).height);
  if (!width || !height) {
    return undefined;
  }
  const preset = parseViewportPreset((viewport as ExecutionViewportSettings).preset);
  return preset ? { width, height, preset } : { width, height };
};

const extractExecutionViewport = (definition: unknown): ExecutionViewportSettings | undefined => {
  if (!isPlainObject(definition)) {
    return undefined;
  }
  const settingsValue = definition.settings;
  if (!isPlainObject(settingsValue)) {
    return undefined;
  }
  const viewportValue = settingsValue.executionViewport ?? settingsValue.viewport;
  if (!isPlainObject(viewportValue)) {
    return undefined;
  }
  const width = parseViewportDimension(viewportValue.width);
  const height = parseViewportDimension(viewportValue.height);
  if (!width || !height) {
    return undefined;
  }
  const preset = parseViewportPreset(viewportValue.preset ?? viewportValue.mode);
  return preset ? { width, height, preset } : { width, height };
};

const buildFlowDefinition = (
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

  let existingSettings: Record<string, unknown> = {};
  if (isPlainObject(baseRecord.settings)) {
    existingSettings = { ...(baseRecord.settings as Record<string, unknown>) };
  }

  if (viewport) {
    existingSettings.executionViewport = viewport;
  } else {
    delete existingSettings.executionViewport;
  }

  if (Object.keys(existingSettings).length > 0) {
    next.settings = existingSettings;
  }

  return next;
};

const PREVIEW_DATA_KEYS = ['previewScreenshot', 'previewScreenshotCapturedAt', 'previewScreenshotSourceUrl'];
const TRANSIENT_NODE_KEYS = new Set([
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
const TRANSIENT_EDGE_KEYS = new Set([
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

const stripPreviewDataFromNodes = (nodes: Node[] | undefined | null): Node[] => {
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

const sanitizeNodesForPersistence = (nodes: Node[] | undefined | null): unknown[] => {
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

    return JSON.parse(JSON.stringify(cleaned));
  });
};

const sanitizeEdgesForPersistence = (edges: Edge[] | undefined | null): unknown[] => {
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

const clearAutosaveTimer = () => {
  if (autosaveTimeout) {
    clearTimeout(autosaveTimeout);
    autosaveTimeout = null;
  }
};

const stableSerialize = (value: unknown): string => {
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

const computeWorkflowFingerprint = (workflow: Workflow | null, nodes: Node[], edges: Edge[]): string => {
  if (!workflow) {
    return '';
  }

  const serializableNodes = sanitizeNodesForPersistence(nodes);
  const serializableEdges = sanitizeEdgesForPersistence(edges ?? []);
  const sanitizedViewport = sanitizeViewportSettings(workflow.executionViewport);

  return stableSerialize({
    name: workflow.name ?? '',
    description: workflow.description ?? '',
    folderPath: workflow.folderPath ?? '/',
    tags: Array.isArray(workflow.tags) ? [...workflow.tags].sort() : [],
    nodes: serializableNodes,
    edges: serializableEdges,
    executionViewport: sanitizedViewport ?? null,
  });
};

export interface Workflow {
  id: string;
  projectId?: string;
  name: string;
  description?: string;
  folderPath: string;
  nodes: Node[];
  edges: Edge[];
  tags?: string[];
  version: number;
  lastChangeSource?: string;
  lastChangeDescription?: string;
  createdAt: Date;
  updatedAt: Date;
  flow_definition?: Record<string, unknown>;
  flowDefinition?: Record<string, unknown>;
  executionViewport?: ExecutionViewportSettings;
  [key: string]: unknown;
}

export interface WorkflowVersionSummary {
  version: number;
  workflowId: string;
  createdAt: Date;
  createdBy: string;
  changeDescription: string;
  definitionHash: string;
  nodeCount: number;
  edgeCount: number;
  flowDefinition: Record<string, unknown>;
}

interface WorkflowConflictMetadata {
  detectedAt: Date;
  remoteVersion: number;
  remoteUpdatedAt: Date;
  changeDescription: string;
  changeSource: string;
  nodeCount: number;
  edgeCount: number;
}

type SaveWorkflowOptions = {
  source?: string;
  changeDescription?: string;
  force?: boolean;
  skipConflictRetry?: boolean;
};

const parseDate = (value: unknown): Date => {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === 'string' || typeof value === 'number') {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? new Date() : parsed;
  }
  return new Date();
};

const ensureArray = <T>(value: unknown, fallback: T[] = []): T[] => {
  if (Array.isArray(value)) {
    return [...value] as T[];
  }
  return [...fallback];
};

const normalizeWorkflowResponse = (workflow: unknown): Workflow => {
  const workflowData = workflow as Record<string, unknown>;
  const flowDef = (workflowData.flow_definition ?? workflowData.flowDefinition) as Record<string, unknown> | undefined;
  const rawNodes = workflowData.nodes ?? flowDef?.nodes ?? [];
  const rawEdges = workflowData.edges ?? flowDef?.edges ?? [];
  const normalizedNodesRaw = normalizeNodes(Array.isArray(rawNodes) ? rawNodes : []);
  const normalizedNodes = stripPreviewDataFromNodes(normalizedNodesRaw);
  const normalizedEdges = normalizeEdges(Array.isArray(rawEdges) ? rawEdges : []);

  const rawDefinition = flowDef ?? {};
  const executionViewport = sanitizeViewportSettings(extractExecutionViewport(rawDefinition));
  const flowDefinition = buildFlowDefinition(
    rawDefinition,
    sanitizeNodesForPersistence(normalizedNodes),
    JSON.parse(JSON.stringify(normalizedEdges ?? [])),
    executionViewport,
  );

  const versionValue = workflowData.version;
  const version = typeof versionValue === 'number'
    ? versionValue
    : parseInt(String(versionValue ?? '1'), 10) || 1;

  return {
    id: workflowData.id ?? '',
    projectId: workflowData.project_id ?? workflowData.projectId,
    name: workflowData.name ?? '',
    description: workflowData.description ?? '',
    folderPath: workflowData.folder_path ?? workflowData.folderPath ?? '/',
    nodes: normalizedNodes,
    edges: normalizedEdges,
    tags: ensureArray<string>(workflowData.tags),
    version,
    lastChangeSource: workflowData.last_change_source ?? workflowData.lastChangeSource ?? 'manual',
    lastChangeDescription: workflowData.last_change_description ?? workflowData.lastChangeDescription ?? '',
    createdAt: parseDate(workflowData.created_at ?? workflowData.createdAt),
    updatedAt: parseDate(workflowData.updated_at ?? workflowData.updatedAt),
    flow_definition: flowDefinition,
    flowDefinition,
    executionViewport,
  } as Workflow;
};

const normalizeVersionSummary = (summary: unknown): WorkflowVersionSummary => {
  const summaryData = summary as Record<string, unknown>;
  const versionNumber = typeof summaryData.version === 'number'
    ? summaryData.version
    : parseInt(String(summaryData.version ?? '0'), 10) || 0;

  return {
    version: versionNumber,
    workflowId: summaryData.workflow_id ?? summaryData.workflowId ?? '',
    createdAt: parseDate(summaryData.created_at ?? summaryData.createdAt),
    createdBy: summaryData.created_by ?? summaryData.createdBy ?? '',
    changeDescription: summaryData.change_description ?? summaryData.changeDescription ?? '',
    definitionHash: summaryData.definition_hash ?? summaryData.definitionHash ?? '',
    nodeCount: typeof summaryData.node_count === 'number' ? summaryData.node_count : 0,
    edgeCount: typeof summaryData.edge_count === 'number' ? summaryData.edge_count : 0,
    flowDefinition: summaryData.flow_definition ?? summaryData.flowDefinition ?? {},
  } as WorkflowVersionSummary;
};

interface WorkflowStore {
  workflows: Workflow[];
  currentWorkflow: Workflow | null;
  nodes: Node[];
  edges: Edge[];
  isDirty: boolean;
  isSaving: boolean;
  lastSavedAt: Date | null;
  lastSavedFingerprint: string | null;
  draftFingerprint: string | null;
  lastSaveError: SaveErrorState | null;
  hasVersionConflict: boolean;
  conflictWorkflow: Workflow | null;
  conflictMetadata: WorkflowConflictMetadata | null;
  versionHistory: WorkflowVersionSummary[];
  isVersionHistoryLoading: boolean;
  versionHistoryError: string | null;
  versionHistoryLoadedFor: string | null;
  restoringVersion: number | null;

  loadWorkflows: (projectId?: string) => Promise<void>;
  loadWorkflow: (id: string) => Promise<void>;
  createWorkflow: (name: string, folderPath: string, projectId?: string) => Promise<Workflow>;
  saveWorkflow: (options?: SaveWorkflowOptions) => Promise<void>;
  updateWorkflow: (updates: Partial<Workflow>) => void;
  generateWorkflow: (prompt: string, name: string, folderPath: string, projectId?: string) => Promise<Workflow>;
  modifyWorkflow: (prompt: string) => Promise<Workflow>;
  deleteWorkflow: (id: string) => Promise<void>;
  bulkDeleteWorkflows: (projectId: string, workflowIds: string[]) => Promise<string[]>;
  loadWorkflowVersions: (workflowId: string, options?: { force?: boolean }) => Promise<void>;
  restoreWorkflowVersion: (workflowId: string, version: number, changeDescription?: string) => Promise<void>;
  refreshConflictWorkflow: () => Promise<Workflow | null>;
  resolveConflictWithReload: () => Promise<void>;
  forceSaveWorkflow: (options?: SaveWorkflowOptions) => Promise<void>;
  scheduleAutosave: (options?: AutosaveOptions) => void;
  cancelAutosave: () => void;
  acknowledgeSaveError: () => void;
}

export const useWorkflowStore = create<WorkflowStore>((set, get) => ({
  workflows: [],
  currentWorkflow: null,
  nodes: [],
  edges: [],
  isDirty: false,
  isSaving: false,
  lastSavedAt: null,
  lastSavedFingerprint: null,
  draftFingerprint: null,
  lastSaveError: null,
  hasVersionConflict: false,
  conflictWorkflow: null,
  conflictMetadata: null,
  versionHistory: [],
  isVersionHistoryLoading: false,
  versionHistoryError: null,
  versionHistoryLoadedFor: null,
  restoringVersion: null,

  loadWorkflows: async (projectId?: string) => {
    try {
      const config = await getConfig();
      let url = `${config.API_URL}/workflows`;
      if (projectId) {
        url = `${config.API_URL}/projects/${projectId}/workflows`;
      }
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to load workflows: ${response.status}`);
      }
      const data = await response.json();
      const workflows = Array.isArray(data.workflows)
        ? (data.workflows as any[]).map(normalizeWorkflowResponse)
        : [];
      set({ workflows });
    } catch (error) {
      logger.error('Failed to load workflows', { component: 'WorkflowStore', action: 'loadWorkflows', projectId }, error);
    }
  },
  
  loadWorkflow: async (id: string) => {
    try {
      clearAutosaveTimer();
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${id}`);
      if (!response.ok) {
        throw new Error(`Failed to load workflow: ${response.status}`);
      }
      const workflow = await response.json();
      const normalized = normalizeWorkflowResponse(workflow);
      const fingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);
      set({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        isDirty: false,
        isSaving: false,
      lastSavedFingerprint: fingerprint,
      draftFingerprint: fingerprint,
      lastSavedAt: normalized.updatedAt instanceof Date ? normalized.updatedAt : new Date(),
      hasVersionConflict: false,
      lastSaveError: null,
      versionHistory: [],
      versionHistoryLoadedFor: null,
      versionHistoryError: null,
      conflictWorkflow: null,
      conflictMetadata: null,
    });
    } catch (error) {
      logger.error('Failed to load workflow', { component: 'WorkflowStore', action: 'loadWorkflow', workflowId: id }, error);

      // Set error state so UI knows loading failed
      const errorMessage = error instanceof Error ? error.message : 'Failed to load workflow';
      set({
        currentWorkflow: null,
        nodes: [],
        edges: [],
        lastSaveError: {
          type: 'network',
          message: errorMessage,
          status: error instanceof Error && error.message.includes('404') ? 404 : undefined,
        },
      });

      // Re-throw so caller can handle the error
      throw error;
    }
  },
  
  createWorkflow: async (name: string, folderPath: string, projectId?: string) => {
    try {
      clearAutosaveTimer();
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          project_id: projectId,
          name,
          folder_path: folderPath,
          flow_definition: {
            nodes: [],
            edges: []
          }
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to create workflow: ${response.status}`);
      }
      const workflow = await response.json();
      const normalized = normalizeWorkflowResponse(workflow);
      const fingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);
      set({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        isDirty: false,
        isSaving: false,
        lastSavedFingerprint: fingerprint,
        draftFingerprint: fingerprint,
        lastSavedAt: new Date(),
        hasVersionConflict: false,
        lastSaveError: null,
        versionHistory: [],
        versionHistoryLoadedFor: null,
        versionHistoryError: null,
        conflictWorkflow: null,
        conflictMetadata: null,
      });
      return normalized;
    } catch (error) {
      logger.error('Failed to create workflow', { component: 'WorkflowStore', action: 'createWorkflow', name, projectId }, error);
      throw error;
    }
  },
  
  saveWorkflow: async (options: SaveWorkflowOptions = {}) => {
    const { skipConflictRetry = false, ...effectiveOptions } = options;
    const state = get();
    const {
      currentWorkflow,
      nodes,
      edges,
      isDirty,
      isSaving,
      draftFingerprint,
      lastSavedFingerprint,
      conflictWorkflow,
    } = state;
    if (!currentWorkflow) return;

    const fingerprint = draftFingerprint ?? computeWorkflowFingerprint(currentWorkflow, nodes, edges);
    if (isSaving) {
      return;
    }

    if (!effectiveOptions.force && (!isDirty || fingerprint === lastSavedFingerprint)) {
      return;
    }

    clearAutosaveTimer();

    const source = effectiveOptions.source?.trim() || (effectiveOptions.force ? 'manual-force-save' : 'manual');
    set({ isSaving: true, lastSaveError: null, hasVersionConflict: effectiveOptions.force ? false : state.hasVersionConflict });

    try {
      const config = await getConfig();
      const serializableNodes = sanitizeNodesForPersistence(nodes ?? []);
      const serializableEdges = sanitizeEdgesForPersistence(edges ?? []);
      const sanitizedViewport = sanitizeViewportSettings(currentWorkflow.executionViewport);
      const flowDefinition = buildFlowDefinition(
        currentWorkflow.flow_definition ?? currentWorkflow.flowDefinition,
        serializableNodes,
        serializableEdges,
        sanitizedViewport,
      );
      const expectedVersion = effectiveOptions.force && conflictWorkflow
        ? conflictWorkflow.version
        : currentWorkflow.version;
      const payload: Record<string, unknown> = {
        name: currentWorkflow.name,
        description: currentWorkflow.description ?? '',
        folder_path: currentWorkflow.folderPath,
        nodes: serializableNodes,
        edges: serializableEdges,
        flow_definition: flowDefinition,
        expected_version: expectedVersion,
        source,
      };

      if (Array.isArray(currentWorkflow.tags)) {
        payload.tags = currentWorkflow.tags;
      }

      const changeDescription = effectiveOptions.changeDescription?.trim() || (effectiveOptions.force ? 'Force save after conflict' : '');
      if (changeDescription) {
        payload.change_description = changeDescription;
      }

      const response = await fetch(`${config.API_URL}/workflows/${currentWorkflow.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const message = await response.text();
        throw { message: message || `Failed to save workflow: ${response.status}` , status: response.status };
      }

      const updated = await response.json();
      const normalized = normalizeWorkflowResponse(updated);
      const nextFingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);

      set((prevState) => ({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        workflows: prevState.workflows.map((workflow) =>
          workflow.id === normalized.id ? { ...workflow, ...normalized } : workflow
        ),
        isSaving: false,
        isDirty: false,
        lastSavedAt: new Date(),
        lastSavedFingerprint: nextFingerprint,
        draftFingerprint: nextFingerprint,
        lastSaveError: null,
        hasVersionConflict: false,
        conflictWorkflow: null,
        conflictMetadata: null,
      }));
    } catch (error) {
      const errorObj = error as { status?: unknown; message?: unknown };
      const status = typeof errorObj.status === 'number' ? errorObj.status : undefined;
      const message =
        (error instanceof Error && error.message) ||
        (typeof errorObj.message === 'string' ? errorObj.message : 'Failed to save workflow');
      const errorType: SaveErrorType = status === 409
        ? 'conflict'
        : typeof status === 'number'
          ? 'server'
          : 'network';

      let conflictSnapshot: Workflow | null = null;
      let autoResolved = false;

      if (errorType === 'conflict') {
        try {
          conflictSnapshot = await get().refreshConflictWorkflow();
        } catch (refreshError) {
          logger.warn('Failed to refresh conflict workflow snapshot', {
            component: 'WorkflowStore',
            action: 'saveWorkflow',
            workflowId: currentWorkflow.id,
          }, refreshError);
        }

        if (conflictSnapshot) {
          const remoteFingerprint = computeWorkflowFingerprint(conflictSnapshot, conflictSnapshot.nodes ?? [], conflictSnapshot.edges ?? []);
          if (remoteFingerprint === fingerprint) {
            logger.info('Resolved workflow save conflict by adopting server revision', {
              component: 'WorkflowStore',
              action: 'saveWorkflow',
              workflowId: currentWorkflow.id,
              source,
              status,
            });

            const resolvedSnapshot = conflictSnapshot;

            set((prevState) => ({
              currentWorkflow: resolvedSnapshot,
              nodes: resolvedSnapshot.nodes,
              edges: resolvedSnapshot.edges,
              workflows: prevState.workflows.map((workflow) =>
                workflow.id === resolvedSnapshot.id ? { ...workflow, ...resolvedSnapshot } : workflow
              ),
              isSaving: false,
              isDirty: false,
              lastSavedAt: resolvedSnapshot.updatedAt instanceof Date ? resolvedSnapshot.updatedAt : new Date(),
              lastSavedFingerprint: remoteFingerprint,
              draftFingerprint: remoteFingerprint,
              lastSaveError: null,
              hasVersionConflict: false,
              conflictWorkflow: null,
              conflictMetadata: null,
            }));

            autoResolved = true;
          }
        }
      }

      if (autoResolved) {
        return;
      }

      let attemptedAutoRetry = false;

      if (
        errorType === 'conflict' &&
        conflictSnapshot &&
        !skipConflictRetry &&
        !effectiveOptions.force
      ) {
        const conflictSource = (conflictSnapshot.lastChangeSource ?? '').toString().toLowerCase();
        const conflictDescription = (conflictSnapshot.lastChangeDescription ?? '').toString().toLowerCase();
        const isFileSyncConflict = conflictSource === 'file-sync' || conflictDescription.includes('workflow file');
        const isAutosaveLoop = conflictSource === 'autosave' && source.toLowerCase() === 'autosave';

        if (isFileSyncConflict || isAutosaveLoop) {
          attemptedAutoRetry = true;
          // Allow a follow-up save attempt with the updated server version.
          set((prevState) => ({ ...prevState, isSaving: false }));

          try {
            await get().saveWorkflow({
              ...effectiveOptions,
              force: true,
              skipConflictRetry: true,
              source: effectiveOptions.source ?? source,
              changeDescription:
                effectiveOptions.changeDescription ??
                (source.toLowerCase() === 'autosave'
                  ? 'Autosave after conflict retry'
                  : 'Retry after conflict'),
            });
            return;
          } catch (retryError) {
            logger.warn('Workflow conflict auto-retry failed', {
              component: 'WorkflowStore',
              action: 'saveWorkflow',
              workflowId: currentWorkflow.id,
              source,
              status,
            }, retryError);
          }
        }
      }

      logger.error('Failed to save workflow', {
        component: 'WorkflowStore',
        action: 'saveWorkflow',
        workflowId: currentWorkflow.id,
        source,
        status,
        attemptedAutoRetry,
      }, error);

      set((prevState) => ({
        isSaving: false,
        isDirty: true,
        lastSaveError: { type: errorType, message, status },
        hasVersionConflict: errorType === 'conflict' ? true : prevState.hasVersionConflict,
      }));

      if (error instanceof Error) {
        throw error;
      }
      throw new Error(message);
    }
  },
  
  updateWorkflow: (updates: Partial<Workflow>) => {
    const state = get();
    const { currentWorkflow, nodes, edges, lastSavedFingerprint, lastSaveError } = state;
    if (!currentWorkflow) return;
    const nextNodes = updates.nodes ?? nodes;
    const nextEdges = updates.edges ?? edges;
    const sanitizedViewport = sanitizeViewportSettings(updates.executionViewport ?? currentWorkflow.executionViewport);
    const baseDefinition = updates.flow_definition
      ?? updates.flowDefinition
      ?? currentWorkflow.flow_definition
      ?? currentWorkflow.flowDefinition;
    const persistedNodes = sanitizeNodesForPersistence(nextNodes as Node[]);
    const persistedEdges = sanitizeEdgesForPersistence(nextEdges as Edge[] | undefined);
    const flowDefinition = buildFlowDefinition(baseDefinition, persistedNodes, persistedEdges, sanitizedViewport);

    const updatedWorkflow = {
      ...currentWorkflow,
      ...updates,
      executionViewport: sanitizedViewport,
      flow_definition: flowDefinition,
      flowDefinition,
    };
    const nextFingerprint = computeWorkflowFingerprint(updatedWorkflow, nextNodes, nextEdges);
    const isDirty = nextFingerprint !== lastSavedFingerprint;

    set({
      currentWorkflow: updatedWorkflow,
      nodes: nextNodes,
      edges: nextEdges,
      isDirty,
      draftFingerprint: nextFingerprint,
      lastSaveError: isDirty ? lastSaveError : null,
    });
  },
  
  generateWorkflow: async (prompt: string, name: string, folderPath: string, projectId?: string) => {
    try {
      clearAutosaveTimer();
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          project_id: projectId,
          name,
          folder_path: folderPath,
          ai_prompt: prompt
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to generate workflow: ${response.status}`);
      }
      const workflow = await response.json();
      const normalized = normalizeWorkflowResponse(workflow);
      const fingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);
      set({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        isDirty: false,
        isSaving: false,
        lastSavedFingerprint: fingerprint,
        draftFingerprint: fingerprint,
        lastSavedAt: new Date(),
        hasVersionConflict: false,
        lastSaveError: null,
        versionHistory: [],
        versionHistoryLoadedFor: null,
        versionHistoryError: null,
        conflictWorkflow: null,
        conflictMetadata: null,
      });
      return normalized;
    } catch (error) {
      logger.error('Failed to generate workflow', { component: 'WorkflowStore', action: 'generateWorkflow', name, projectId }, error);
      throw error;
    }
  },
  
  modifyWorkflow: async (prompt: string) => {
    const { currentWorkflow, nodes, edges } = get();
    if (!currentWorkflow) {
      throw new Error('No workflow loaded to modify');
    }
    
    try {
      clearAutosaveTimer();
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${currentWorkflow.id}/modify`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          modification_prompt: prompt,
          current_flow: { nodes, edges }
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to modify workflow: ${response.status}`);
      }
      const modifiedWorkflow = await response.json();
      const normalized = normalizeWorkflowResponse(modifiedWorkflow);
      const fingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);
      set({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        isDirty: false,
        isSaving: false,
        lastSavedFingerprint: fingerprint,
        draftFingerprint: fingerprint,
        lastSavedAt: new Date(),
        hasVersionConflict: false,
        lastSaveError: null,
        versionHistory: [],
        versionHistoryLoadedFor: null,
        versionHistoryError: null,
        conflictWorkflow: null,
        conflictMetadata: null,
      });
      return normalized;
    } catch (error) {
      logger.error('Failed to modify workflow', { component: 'WorkflowStore', action: 'modifyWorkflow', workflowId: currentWorkflow.id }, error);
      throw error;
    }
  },
  
  deleteWorkflow: async (id: string) => {
    try {
      clearAutosaveTimer();
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Failed to delete workflow: ${response.status}`);
      }
      const workflows = get().workflows.filter(w => w.id !== id);
      set({ workflows });
      if (get().currentWorkflow?.id === id) {
        set({
          currentWorkflow: null,
          nodes: [],
          edges: [],
          isDirty: false,
          isSaving: false,
          lastSavedAt: null,
          lastSavedFingerprint: null,
          draftFingerprint: null,
          hasVersionConflict: false,
          lastSaveError: null,
          versionHistory: [],
          versionHistoryLoadedFor: null,
          versionHistoryError: null,
          restoringVersion: null,
          conflictWorkflow: null,
          conflictMetadata: null,
        });
      }
    } catch (error) {
      logger.error('Failed to delete workflow', { component: 'WorkflowStore', action: 'deleteWorkflow', workflowId: id }, error);
      throw error;
    }
  },

  bulkDeleteWorkflows: async (projectId: string, workflowIds: string[]) => {
    if (workflowIds.length === 0) {
      return [];
    }

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${projectId}/workflows/bulk-delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ workflow_ids: workflowIds }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to delete workflows: ${response.status}`);
      }

      const data = await response.json();
      const deletedIds = Array.isArray(data.deleted_ids) ? (data.deleted_ids as string[]) : workflowIds;
      const deletedSet = new Set(deletedIds);

      const current = get().currentWorkflow;
      if (current && deletedSet.has(current.id)) {
        clearAutosaveTimer();
      }

      set((state) => {
        const currentDeleted = state.currentWorkflow && deletedSet.has(state.currentWorkflow.id);
        return {
        workflows: state.workflows.filter((workflow) => !deletedSet.has(workflow.id)),
        currentWorkflow: currentDeleted ? null : state.currentWorkflow,
        nodes: currentDeleted ? [] : state.nodes,
        edges: currentDeleted ? [] : state.edges,
        isDirty: currentDeleted ? false : state.isDirty,
        isSaving: currentDeleted ? false : state.isSaving,
        lastSavedAt: currentDeleted ? null : state.lastSavedAt,
        lastSavedFingerprint: currentDeleted ? null : state.lastSavedFingerprint,
        draftFingerprint: currentDeleted ? null : state.draftFingerprint,
        hasVersionConflict: currentDeleted ? false : state.hasVersionConflict,
        lastSaveError: currentDeleted ? null : state.lastSaveError,
        versionHistory: currentDeleted ? [] : state.versionHistory,
        versionHistoryLoadedFor: currentDeleted ? null : state.versionHistoryLoadedFor,
        versionHistoryError: currentDeleted ? null : state.versionHistoryError,
        restoringVersion: currentDeleted ? null : state.restoringVersion,
        conflictWorkflow: currentDeleted ? null : state.conflictWorkflow,
        conflictMetadata: currentDeleted ? null : state.conflictMetadata,
      };
    });

      return deletedIds;
    } catch (error) {
      logger.error('Failed to bulk delete workflows', { component: 'WorkflowStore', action: 'bulkDeleteWorkflows', projectId, count: workflowIds.length }, error);
      throw error;
    }
  },

  loadWorkflowVersions: async (workflowId: string, options: { force?: boolean } = {}) => {
    if (!workflowId) {
      return;
    }

    const { isVersionHistoryLoading, versionHistoryLoadedFor } = get();
    if (isVersionHistoryLoading && !options.force) {
      return;
    }

    if (!options.force && versionHistoryLoadedFor === workflowId) {
      return;
    }

    set({ isVersionHistoryLoading: true, versionHistoryError: null });

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/versions?limit=50`);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load workflow versions (${response.status})`);
      }

      const payload = await response.json();
      const items = Array.isArray(payload?.versions) ? payload.versions : [];
      const summaries = items
        .map(normalizeVersionSummary)
        .sort((a: WorkflowVersionSummary, b: WorkflowVersionSummary) => b.version - a.version);

      set({
        versionHistory: summaries,
        isVersionHistoryLoading: false,
        versionHistoryError: null,
        versionHistoryLoadedFor: workflowId,
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load workflow versions';
      logger.error('Failed to load workflow versions', {
        component: 'WorkflowStore',
        action: 'loadWorkflowVersions',
        workflowId,
      }, error);
      set({
        isVersionHistoryLoading: false,
        versionHistoryError: message,
      });
      throw error;
    }
  },

  restoreWorkflowVersion: async (workflowId: string, version: number, changeDescription?: string) => {
    const { currentWorkflow } = get();
    if (!currentWorkflow || currentWorkflow.id !== workflowId) {
      throw new Error('Workflow is not loaded');
    }

    clearAutosaveTimer();
    set({ restoringVersion: version, lastSaveError: null });

    try {
      const config = await getConfig();
      const payload = changeDescription && changeDescription.trim().length > 0
        ? JSON.stringify({ change_description: changeDescription.trim() })
        : undefined;

      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/versions/${version}/restore`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: payload,
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to restore version (${response.status})`);
      }

      const data = await response.json();
      const normalized = normalizeWorkflowResponse(data?.workflow);
      const nextFingerprint = computeWorkflowFingerprint(normalized, normalized.nodes, normalized.edges);

      set((prevState) => ({
        currentWorkflow: normalized,
        nodes: normalized.nodes,
        edges: normalized.edges,
        workflows: prevState.workflows.map((workflow) =>
          workflow.id === normalized.id ? { ...workflow, ...normalized } : workflow
        ),
        isDirty: false,
        isSaving: false,
        lastSavedAt: new Date(),
        lastSavedFingerprint: nextFingerprint,
        draftFingerprint: nextFingerprint,
        hasVersionConflict: false,
        lastSaveError: null,
        versionHistoryLoadedFor: null,
        restoringVersion: null,
        conflictWorkflow: null,
        conflictMetadata: null,
      }));

      try {
        await get().loadWorkflowVersions(workflowId, { force: true });
      } catch (historyError) {
        logger.warn('Failed to refresh version history after restore', {
          component: 'WorkflowStore',
          action: 'restoreWorkflowVersion',
          workflowId,
          version,
        }, historyError);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to restore workflow version';
      logger.error('Failed to restore workflow version', {
        component: 'WorkflowStore',
        action: 'restoreWorkflowVersion',
        workflowId,
        version,
      }, error);

      set({ restoringVersion: null, versionHistoryError: message });

      if (error instanceof Error) {
        throw error;
      }
      throw new Error(message);
    }
  },

  refreshConflictWorkflow: async () => {
    const { currentWorkflow } = get();
    if (!currentWorkflow) {
      return null;
    }

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${currentWorkflow.id}`);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to refresh workflow snapshot: ${response.status}`);
      }
      const payload = await response.json();
      const normalized = normalizeWorkflowResponse(payload);
      set({
        conflictWorkflow: normalized,
        conflictMetadata: {
          detectedAt: new Date(),
          remoteVersion: normalized.version,
          remoteUpdatedAt: normalized.updatedAt,
          changeDescription: normalized.lastChangeDescription ?? '',
          changeSource: normalized.lastChangeSource ?? 'manual',
          nodeCount: normalized.nodes.length,
          edgeCount: normalized.edges.length,
        },
      });
      return normalized;
    } catch (error) {
      logger.error('Failed to refresh conflict workflow', {
        component: 'WorkflowStore',
        action: 'refreshConflictWorkflow',
        workflowId: currentWorkflow.id,
      }, error);
      throw error;
    }
  },

  resolveConflictWithReload: async () => {
    const { currentWorkflow } = get();
    if (!currentWorkflow) {
      return;
    }
    await get().loadWorkflow(currentWorkflow.id);
    set({
      hasVersionConflict: false,
      conflictWorkflow: null,
      conflictMetadata: null,
      lastSaveError: null,
    });
  },

  forceSaveWorkflow: async (options: SaveWorkflowOptions = {}) => {
    await get().saveWorkflow({ ...options, force: true });
    set({
      hasVersionConflict: false,
      conflictWorkflow: null,
      conflictMetadata: null,
    });
  },

  scheduleAutosave: (options: AutosaveOptions = {}) => {
    const state = get();
    const { currentWorkflow, isDirty, isSaving, hasVersionConflict } = state;
    if (!currentWorkflow || !isDirty || isSaving || hasVersionConflict) {
      return;
    }

    clearAutosaveTimer();
    const delay = typeof options.debounceMs === 'number' ? Math.max(options.debounceMs, 250) : AUTOSAVE_DELAY_MS;
    const workflowId = currentWorkflow.id;

    autosaveTimeout = setTimeout(async () => {
      autosaveTimeout = null;
      try {
        await get().saveWorkflow({ ...options, source: options.source ?? 'autosave' });
      } catch (error) {
        logger.warn('Autosave failed', {
          component: 'WorkflowStore',
          action: 'scheduleAutosave',
          workflowId,
        }, error);
      }
    }, delay);
  },

  cancelAutosave: () => {
    clearAutosaveTimer();
  },

  acknowledgeSaveError: () => {
    set({ lastSaveError: null });
  }
}));
