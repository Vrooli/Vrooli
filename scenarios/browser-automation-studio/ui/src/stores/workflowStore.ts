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

  const serializableNodes = JSON.parse(JSON.stringify(nodes ?? []));
  const serializableEdges = JSON.parse(JSON.stringify(edges ?? []));

  return stableSerialize({
    name: workflow.name ?? '',
    description: workflow.description ?? '',
    folderPath: workflow.folderPath ?? '/',
    tags: Array.isArray(workflow.tags) ? [...workflow.tags].sort() : [],
    nodes: serializableNodes,
    edges: serializableEdges,
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

const normalizeWorkflowResponse = (workflow: any): Workflow => {
  const rawNodes = workflow?.nodes ?? workflow?.flow_definition?.nodes ?? workflow?.flowDefinition?.nodes ?? [];
  const rawEdges = workflow?.edges ?? workflow?.flow_definition?.edges ?? workflow?.flowDefinition?.edges ?? [];
  const normalizedNodes = normalizeNodes(rawNodes ?? []);
  const normalizedEdges = normalizeEdges(rawEdges ?? []);

  const versionValue = workflow?.version;
  const version = typeof versionValue === 'number'
    ? versionValue
    : parseInt(String(versionValue ?? '1'), 10) || 1;

  return {
    id: workflow?.id ?? '',
    projectId: workflow?.project_id ?? workflow?.projectId,
    name: workflow?.name ?? '',
    description: workflow?.description ?? '',
    folderPath: workflow?.folder_path ?? workflow?.folderPath ?? '/',
    nodes: normalizedNodes,
    edges: normalizedEdges,
    tags: ensureArray<string>(workflow?.tags),
    version,
    lastChangeSource: workflow?.last_change_source ?? workflow?.lastChangeSource ?? 'manual',
    lastChangeDescription: workflow?.last_change_description ?? workflow?.lastChangeDescription ?? '',
    createdAt: parseDate(workflow?.created_at ?? workflow?.createdAt),
    updatedAt: parseDate(workflow?.updated_at ?? workflow?.updatedAt),
    flow_definition: workflow?.flow_definition ?? workflow?.flowDefinition ?? { nodes: normalizedNodes, edges: normalizedEdges },
  } as Workflow;
};

const normalizeVersionSummary = (summary: any): WorkflowVersionSummary => {
  const versionNumber = typeof summary?.version === 'number'
    ? summary.version
    : parseInt(String(summary?.version ?? '0'), 10) || 0;

  return {
    version: versionNumber,
    workflowId: summary?.workflow_id ?? summary?.workflowId ?? '',
    createdAt: parseDate(summary?.created_at ?? summary?.createdAt),
    createdBy: summary?.created_by ?? summary?.createdBy ?? '',
    changeDescription: summary?.change_description ?? summary?.changeDescription ?? '',
    definitionHash: summary?.definition_hash ?? summary?.definitionHash ?? '',
    nodeCount: typeof summary?.node_count === 'number' ? summary.node_count : 0,
    edgeCount: typeof summary?.edge_count === 'number' ? summary.edge_count : 0,
    flowDefinition: summary?.flow_definition ?? summary?.flowDefinition ?? {},
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

    if (!options.force && (!isDirty || fingerprint === lastSavedFingerprint)) {
      return;
    }

    clearAutosaveTimer();

    const source = options.source?.trim() || (options.force ? 'manual-force-save' : 'manual');
    set({ isSaving: true, lastSaveError: null, hasVersionConflict: options.force ? false : state.hasVersionConflict });

    try {
      const config = await getConfig();
      const serializableNodes = JSON.parse(JSON.stringify(nodes ?? []));
      const serializableEdges = JSON.parse(JSON.stringify(edges ?? []));
      const expectedVersion = options.force && conflictWorkflow
        ? conflictWorkflow.version
        : currentWorkflow.version;
      const payload: Record<string, unknown> = {
        name: currentWorkflow.name,
        description: currentWorkflow.description ?? '',
        folder_path: currentWorkflow.folderPath,
        nodes: serializableNodes,
        edges: serializableEdges,
        flow_definition: { nodes: serializableNodes, edges: serializableEdges },
        expected_version: expectedVersion,
        source,
      };

      if (Array.isArray(currentWorkflow.tags)) {
        payload.tags = currentWorkflow.tags;
      }

      const changeDescription = options.changeDescription?.trim() || (options.force ? 'Force save after conflict' : '');
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
      const status = typeof (error as any)?.status === 'number' ? (error as any).status as number : undefined;
      const message = (error as Error).message || (typeof (error as any)?.message === 'string' ? (error as any).message : 'Failed to save workflow');
      const errorType: SaveErrorType = status === 409
        ? 'conflict'
        : typeof status === 'number'
          ? 'server'
          : 'network';

      logger.error('Failed to save workflow', {
        component: 'WorkflowStore',
        action: 'saveWorkflow',
        workflowId: currentWorkflow.id,
        source,
        status,
      }, error);

      set((prevState) => ({
        isSaving: false,
        isDirty: true,
        lastSaveError: { type: errorType, message, status },
        hasVersionConflict: errorType === 'conflict' ? true : prevState.hasVersionConflict,
      }));

      if (errorType === 'conflict') {
        try {
          await get().refreshConflictWorkflow();
        } catch (refreshError) {
          logger.warn('Failed to refresh conflict workflow snapshot', {
            component: 'WorkflowStore',
            action: 'saveWorkflow',
            workflowId: currentWorkflow.id,
          }, refreshError);
        }
      }

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
    const updatedWorkflow = { ...currentWorkflow, ...updates };
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
