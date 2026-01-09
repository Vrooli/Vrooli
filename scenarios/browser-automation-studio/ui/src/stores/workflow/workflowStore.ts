import { create } from 'zustand';
import type { Node, Edge } from 'reactflow';
import {
  ListWorkflowsResponseSchema,
  WorkflowVersionListSchema,
  RestoreWorkflowVersionResponseSchema,
  type ListWorkflowsResponse,
  type WorkflowVersionList,
  type WorkflowVersion as ProtoWorkflowVersion,
  type RestoreWorkflowVersionResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { getConfig } from '../../config';
import { logger } from '../../utils/logger';
import { parseProtoStrict } from '../../utils/proto';

// Import types
import type {
  Workflow,
  WorkflowVersionSummary,
  WorkflowStore,
  SaveWorkflowOptions,
  AutosaveOptions,
  SaveErrorType,
} from './types';

// Import utilities
import { sanitizeViewportSettings } from './utils/viewport';
import { buildFlowDefinition, sanitizeNodesForPersistence, sanitizeEdgesForPersistence } from './utils/serialization';
import { computeWorkflowFingerprint } from './utils/fingerprint';
import {
  parseWorkflowSummaryMessage,
  parseWorkflowFromCreateResponse,
  parseWorkflowFromUpdateResponse,
  parseWorkflowVersionMessage,
} from './utils/proto';
import {
  normalizeVersionSummary,
  normalizeWorkflowPayloadOrThrow,
  buildWorkflowLoadState,
} from './utils/normalization';

// ============================================================================
// Autosave Timer
// ============================================================================

const AUTOSAVE_DELAY_MS = 2500;
let autosaveTimeout: ReturnType<typeof setTimeout> | null = null;

const clearAutosaveTimer = () => {
  if (autosaveTimeout) {
    clearTimeout(autosaveTimeout);
    autosaveTimeout = null;
  }
};

// ============================================================================
// Store Implementation
// ============================================================================

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

      let workflows: Workflow[] = [];
      try {
        const protoList = parseProtoStrict<ListWorkflowsResponse>(ListWorkflowsResponseSchema, data);
        workflows = (protoList.workflows ?? []).map((item, idx) => {
          const parsed = parseWorkflowSummaryMessage(item);
          if (!parsed) {
            logger.warn('Skipping invalid workflow in proto list', { component: 'WorkflowStore', action: 'loadWorkflows', index: idx });
          }
          return parsed;
        }).filter((item): item is Workflow => Boolean(item));
      } catch (err) {
        logger.error('Failed to parse workflow list proto', { component: 'WorkflowStore', action: 'loadWorkflows' }, err);
        // Fallback for legacy payloads
        const legacyList = Array.isArray((data as any)?.workflows) ? (data as any).workflows as Record<string, unknown>[] : [];
        workflows = legacyList
          .map((entry, idx) => {
            const parsed = parseWorkflowSummaryMessage(entry);
            if (!parsed) {
              logger.warn('Skipping invalid workflow in legacy list', { component: 'WorkflowStore', action: 'loadWorkflows', index: idx });
            }
            return parsed;
          })
          .filter((item): item is Workflow => Boolean(item));
      }

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
        const errorText = await response.text();
        throw new Error(errorText || `Failed to load workflow: ${response.status}`);
      }
      const payload = await response.json();
      const normalized = normalizeWorkflowPayloadOrThrow(payload, 'loadWorkflow');
      set(buildWorkflowLoadState(normalized));
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
      const payload = await response.json();
      const normalized = parseWorkflowFromCreateResponse(payload);
      if (!normalized) {
        throw new Error('Failed to parse created workflow payload');
      }
      set(buildWorkflowLoadState(normalized, { lastSavedAt: new Date() }));
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
        currentWorkflow.flowDefinition,
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

      const responsePayload = await response.json();
      const normalized = parseWorkflowFromUpdateResponse(responsePayload);
      if (!normalized) {
        throw new Error('Failed to parse workflow save payload');
      }
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
    const baseDefinition = updates.flowDefinition ?? currentWorkflow.flowDefinition;
    const persistedNodes = sanitizeNodesForPersistence(nextNodes as Node[]);
    const persistedEdges = sanitizeEdgesForPersistence(nextEdges as Edge[] | undefined);
    const flowDefinition = buildFlowDefinition(baseDefinition, persistedNodes, persistedEdges, sanitizedViewport);

    const updatedWorkflow = {
      ...currentWorkflow,
      ...updates,
      executionViewport: sanitizedViewport,
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
      const payload = await response.json();
      const normalized = parseWorkflowFromCreateResponse(payload);
      if (!normalized) {
        throw new Error('Failed to parse generated workflow payload');
      }
      set(buildWorkflowLoadState(normalized, { lastSavedAt: new Date() }));
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
      const payload = await response.json();
      const normalized = parseWorkflowFromUpdateResponse(payload);
      if (!normalized) {
        throw new Error('Failed to parse modified workflow payload');
      }
      set(buildWorkflowLoadState(normalized, { lastSavedAt: new Date() }));
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
      let summaries: WorkflowVersionSummary[] = [];

      try {
        const protoList = parseProtoStrict<WorkflowVersionList>(WorkflowVersionListSchema, payload);
        summaries = (protoList.versions ?? [])
          .map((item) => parseWorkflowVersionMessage(item))
          .filter((item: WorkflowVersionSummary | null): item is WorkflowVersionSummary => item !== null)
          .sort((a: WorkflowVersionSummary, b: WorkflowVersionSummary) => b.version - a.version);
      } catch (err) {
        logger.error('Failed to parse workflow version list proto, falling back to raw payload', {
          component: 'WorkflowStore',
          action: 'loadWorkflowVersions',
        }, err);
      }

      if (summaries.length === 0 && Array.isArray((payload as any)?.versions)) {
        summaries = (payload as any).versions
          .map((item: unknown) => {
            try {
              return normalizeVersionSummary(item as Record<string, unknown>);
            } catch {
              return null;
            }
          })
          .filter((item: WorkflowVersionSummary | null): item is WorkflowVersionSummary => item !== null)
          .sort((a: WorkflowVersionSummary, b: WorkflowVersionSummary) => b.version - a.version);
      }

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

      let restoredWorkflowPayload: unknown = data?.workflow ?? data;
      let restoredVersionPayload: ProtoWorkflowVersion | null = null;

      try {
        const proto = parseProtoStrict<RestoreWorkflowVersionResponse>(RestoreWorkflowVersionResponseSchema, data);
        if (proto.workflow) {
          restoredWorkflowPayload = proto.workflow;
        }
        if (proto.restoredVersion) {
          restoredVersionPayload = proto.restoredVersion;
        }
      } catch (err) {
        logger.error('Failed to parse restore workflow proto response', { component: 'WorkflowStore', action: 'restoreWorkflowVersion', workflowId }, err);
      }

      const normalized = normalizeWorkflowPayloadOrThrow(restoredWorkflowPayload, 'restoreWorkflowVersion');

      const restoredVersionSummary = restoredVersionPayload
        ? parseWorkflowVersionMessage(restoredVersionPayload)
        : null;
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
        versionHistory: restoredVersionSummary
          ? [restoredVersionSummary, ...prevState.versionHistory.filter((version) => version.version !== restoredVersionSummary.version)]
          : prevState.versionHistory,
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
      const normalized = normalizeWorkflowPayloadOrThrow(payload, 'refreshConflictWorkflow');
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
