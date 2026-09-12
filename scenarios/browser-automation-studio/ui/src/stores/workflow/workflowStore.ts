import { create } from 'zustand';
import type { Node, Edge } from 'reactflow';
import type {
  WorkflowVersionList,
  WorkflowVersion as ProtoWorkflowVersion,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import {
  bulkDeleteProjectWorkflowsViaApi,
  fetchProjectWorkflows,
} from '../../domains/projects/services/projectApi';
import {
  createWorkflowViaApi,
  deleteWorkflowViaApi,
  flowDefinitionFromJson,
  getWorkflowViaApi,
  listWorkflowsViaApi,
  listWorkflowVersionsViaApi,
  modifyWorkflowViaApi,
  parseChangeSource,
  restoreWorkflowVersionViaApi,
  updateWorkflowViaApi,
} from '../../domains/workflows/services/workflowApi';
import { logger } from '../../utils/logger';
import { protoMessageToJson } from '../../utils/proto';
import { WorkflowSummarySchema } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';

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
  parseWorkflowFromCreateProto,
  parseWorkflowFromUpdateProto,
  parseWorkflowVersionMessage,
} from './utils/proto';
import {
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
      // Project-scoped listing uses ProjectsService.ListProjectWorkflows
      // (Connect-RPC). Cross-project /workflows is still REST until that
      // domain migrates — see plan Phase 7.
      if (projectId) {
        const items = await fetchProjectWorkflows(projectId);
        const workflows = items
          .map((it) => parseWorkflowSummaryMessage({
            id: it.id,
            name: it.name,
            description: it.description,
            project_id: it.project_id,
            folder_path: it.folder_path,
            version: it.version,
            created_at: it.created_at,
            updated_at: it.updated_at,
          } as Record<string, unknown>))
          .filter((w): w is Workflow => Boolean(w));
        set({ workflows });
        return;
      }
      const protoList = await listWorkflowsViaApi();
      const workflows = (protoList.workflows ?? [])
        .map((item, idx) => {
          const parsed = parseWorkflowSummaryMessage(item);
          if (!parsed) {
            logger.warn('Skipping invalid workflow in proto list', { component: 'WorkflowStore', action: 'loadWorkflows', index: idx });
          }
          return parsed;
        })
        .filter((item): item is Workflow => Boolean(item));

      set({ workflows });
    } catch (error) {
      logger.error('Failed to load workflows', { component: 'WorkflowStore', action: 'loadWorkflows', projectId }, error);
    }
  },

  loadWorkflow: async (id: string) => {
    try {
      clearAutosaveTimer();
      const resp = await getWorkflowViaApi(id);
      const summaryRecord = resp.workflow ? protoMessageToJson(WorkflowSummarySchema, resp.workflow) : null;
      if (!summaryRecord) {
        throw new Error('Workflow not found');
      }
      const normalized = normalizeWorkflowPayloadOrThrow(summaryRecord, 'loadWorkflow');
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
      const resp = await createWorkflowViaApi({
        projectId: projectId ?? '',
        name,
        folderPath,
        flowDefinition: flowDefinitionFromJson({ nodes: [], edges: [] }),
      });
      const normalized = parseWorkflowFromCreateProto(resp);
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
      const serializableNodes = sanitizeNodesForPersistence(nodes ?? []);
      const serializableEdges = sanitizeEdgesForPersistence(edges ?? []);
      const sanitizedViewport = sanitizeViewportSettings(currentWorkflow.executionViewport);
      const flowDefinitionJson = buildFlowDefinition(
        currentWorkflow.flowDefinition,
        serializableNodes,
        serializableEdges,
        sanitizedViewport,
      );
      const expectedVersion = effectiveOptions.force && conflictWorkflow
        ? conflictWorkflow.version
        : currentWorkflow.version;

      const changeDescription = effectiveOptions.changeDescription?.trim() || (effectiveOptions.force ? 'Force save after conflict' : '');

      let resp;
      try {
        resp = await updateWorkflowViaApi({
          workflowId: currentWorkflow.id,
          name: currentWorkflow.name,
          description: currentWorkflow.description ?? '',
          folderPath: currentWorkflow.folderPath ?? '',
          tags: Array.isArray(currentWorkflow.tags) ? currentWorkflow.tags : [],
          flowDefinition: flowDefinitionFromJson(flowDefinitionJson),
          expectedVersion: expectedVersion,
          source: parseChangeSource(source),
          changeDescription,
        });
      } catch (err) {
        const errAny = err as { code?: string; message?: string };
        // Map Connect error codes to legacy status numbers the rest of the
        // store branches on (conflict → 409, etc.).
        const status = errAny.code === 'aborted' ? 409 : 500;
        throw { message: errAny.message ?? 'Failed to save workflow', status };
      }

      const normalized = parseWorkflowFromUpdateProto(resp);
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
      const resp = await createWorkflowViaApi({
        projectId: projectId ?? '',
        name,
        folderPath,
        aiPrompt: prompt,
      });
      const normalized = parseWorkflowFromCreateProto(resp);
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
      const resp = await modifyWorkflowViaApi({
        workflowId: currentWorkflow.id,
        modificationPrompt: prompt,
        currentFlow: flowDefinitionFromJson({ nodes, edges }),
      });
      const normalized = parseWorkflowFromUpdateProto(resp);
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
      await deleteWorkflowViaApi(id);
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
      const result = await bulkDeleteProjectWorkflowsViaApi(projectId, workflowIds);
      const deletedSet = new Set(result.deleted_ids.length > 0 ? result.deleted_ids : workflowIds);

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

      return Array.from(deletedSet);
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
      const protoList: WorkflowVersionList = await listWorkflowVersionsViaApi(workflowId);
      const summaries: WorkflowVersionSummary[] = (protoList.versions ?? [])
        .map((item) => parseWorkflowVersionMessage(item))
        .filter((item: WorkflowVersionSummary | null): item is WorkflowVersionSummary => item !== null)
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
      const proto = await restoreWorkflowVersionViaApi(workflowId, version, changeDescription?.trim() ?? '');
      const restoredWorkflowPayload = proto.workflow
        ? protoMessageToJson(WorkflowSummarySchema, proto.workflow)
        : null;
      const restoredVersionPayload: ProtoWorkflowVersion | null = proto.restoredVersion ?? null;

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
      const resp = await getWorkflowViaApi(currentWorkflow.id);
      const summaryRecord = resp.workflow ? protoMessageToJson(WorkflowSummarySchema, resp.workflow) : null;
      if (!summaryRecord) {
        throw new Error('Workflow not found');
      }
      const normalized = normalizeWorkflowPayloadOrThrow(summaryRecord, 'refreshConflictWorkflow');
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
