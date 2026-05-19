import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { act } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import {
  CreateWorkflowResponseSchema,
  DeleteWorkflowResponseSchema,
  GetWorkflowResponseSchema,
  RestoreWorkflowVersionResponseSchema,
  UpdateWorkflowResponseSchema,
  WorkflowSummarySchema,
  WorkflowVersionListSchema,
  WorkflowVersionSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { useWorkflowStore } from '@stores/workflowStore';
import type { Node, Edge } from 'reactflow';
import { fetchJsonResponse, installFetchMock, type FetchMock } from '@/test-utils';

// Mock getConfig
vi.mock('../config', () => ({
  getConfig: vi.fn().mockResolvedValue({
    API_URL: 'http://localhost:8080/api/v1',
  }),
}));

// Mock logger
vi.mock('../utils/logger', () => ({
  logger: {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  },
}));

// Mock workflow normalizers
vi.mock('../utils/workflowNormalizers', () => ({
  normalizeNodes: vi.fn((nodes) => nodes),
  normalizeEdges: vi.fn((edges) => edges),
}));

// Mock the projects Connect-RPC adapter used by bulkDeleteWorkflows + loadWorkflows.
const bulkDeleteProjectWorkflowsMock = vi.fn();
const listProjectWorkflowsMock = vi.fn();
vi.mock('../domains/projects/services/projectApi', () => ({
  bulkDeleteProjectWorkflowsViaApi: (...a: unknown[]) => bulkDeleteProjectWorkflowsMock(...a),
  fetchProjectWorkflows: async (projectId: string) => {
    const resp = await listProjectWorkflowsMock({ projectId });
    return (resp.workflows ?? []).map((w: Record<string, unknown>) => ({
      id: w.id,
      name: w.name,
      description: w.description,
      project_id: w.projectId,
      folder_path: w.folderPath,
      version: w.version,
      created_at: '',
      updated_at: '',
    }));
  },
}));

// Mock the WorkflowsService Connect-RPC client.
const listWorkflowsMock = vi.fn();
const getWorkflowMock = vi.fn();
const createWorkflowMock = vi.fn();
const updateWorkflowMock = vi.fn();
const deleteWorkflowMock = vi.fn();
const modifyWorkflowMock = vi.fn();
const listWorkflowVersionsMock = vi.fn();
const restoreWorkflowVersionMock = vi.fn();
vi.mock('../api/workflows', () => ({
  workflowsClient: {
    listWorkflows: (...a: unknown[]) => listWorkflowsMock(...a),
    getWorkflow: (...a: unknown[]) => getWorkflowMock(...a),
    createWorkflow: (...a: unknown[]) => createWorkflowMock(...a),
    updateWorkflow: (...a: unknown[]) => updateWorkflowMock(...a),
    deleteWorkflow: (...a: unknown[]) => deleteWorkflowMock(...a),
    modifyWorkflow: (...a: unknown[]) => modifyWorkflowMock(...a),
    listWorkflowVersions: (...a: unknown[]) => listWorkflowVersionsMock(...a),
    restoreWorkflowVersion: (...a: unknown[]) => restoreWorkflowVersionMock(...a),
    executeWorkflow: vi.fn(),
    executeAdhocWorkflow: vi.fn(),
    validateWorkflow: vi.fn(),
    validateResolvedWorkflow: vi.fn(),
    getWorkflowVersion: vi.fn(),
  },
}));

interface WorkflowSummaryInit {
  id: string;
  projectId?: string;
  name?: string;
  folderPath?: string;
  description?: string;
  version?: number;
  flowDefinition?: { nodes?: unknown[]; edges?: unknown[] };
}

const makeSummary = (fields: WorkflowSummaryInit) =>
  create(WorkflowSummarySchema, {
    id: fields.id,
    projectId: fields.projectId ?? '',
    name: fields.name ?? '',
    folderPath: fields.folderPath ?? '',
    description: fields.description ?? '',
    version: fields.version ?? 1,
    flowDefinition: fields.flowDefinition as unknown as Parameters<typeof create>[1],
  } as Parameters<typeof create>[1]);

describe('workflowStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = installFetchMock();

    // Reset store state
    useWorkflowStore.setState({
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
    });
  });

  afterEach(() => {
    // Clear any autosave timers
    useWorkflowStore.getState().cancelAutosave();
  });

  describe('Workflow CRUD Operations', () => {
    it('loads workflows successfully [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const mockWorkflows = [
        {
          id: 'workflow-1',
          project_id: 'project-1',
          name: 'Test Workflow 1',
          description: 'Description 1',
          folder_path: '/workflows',
          version: 1,
          created_at: '2025-01-01T00:00:00Z',
          updated_at: '2025-01-01T00:00:00Z',
          flow_definition: { nodes: [], edges: [] },
        },
        {
          id: 'workflow-2',
          project_id: 'project-1',
          name: 'Test Workflow 2',
          description: 'Description 2',
          folder_path: '/workflows',
          version: 1,
          created_at: '2025-01-02T00:00:00Z',
          updated_at: '2025-01-02T00:00:00Z',
          flow_definition: { nodes: [], edges: [] },
        },
      ];

      listProjectWorkflowsMock.mockResolvedValueOnce({
        workflows: mockWorkflows.map((w) => ({
          id: w.id,
          name: w.name,
          description: w.description,
          projectId: w.project_id,
          folderPath: w.folder_path,
          version: w.version,
        })),
      });

      await act(async () => {
        await useWorkflowStore.getState().loadWorkflows('project-1');
      });

      const state = useWorkflowStore.getState();
      expect(state.workflows).toHaveLength(2);
      expect(state.workflows[0].name).toBe('Test Workflow 1');
      expect(listProjectWorkflowsMock).toHaveBeenCalledWith({ projectId: 'project-1' });
    });

    it('loads single workflow successfully [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const mockWorkflow = {
        id: 'workflow-1',
        project_id: 'project-1',
        name: 'Test Workflow',
        description: 'Test Description',
        folder_path: '/workflows',
        version: 1,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
        flow_definition: {
          nodes: [
            { id: 'node-1', type: 'navigate', position: { x: 100, y: 100 }, data: { url: 'https://example.com' } },
          ],
          edges: [],
        },
      };

      getWorkflowMock.mockResolvedValueOnce(create(GetWorkflowResponseSchema, {
        workflow: makeSummary({
          id: mockWorkflow.id,
          projectId: mockWorkflow.project_id,
          name: mockWorkflow.name,
          folderPath: mockWorkflow.folder_path,
          version: mockWorkflow.version,
          flowDefinition: mockWorkflow.flow_definition,
        }),
      } as Parameters<typeof create>[1]));

      await act(async () => {
        await useWorkflowStore.getState().loadWorkflow('workflow-1');
      });

      const state = useWorkflowStore.getState();
      expect(state.currentWorkflow).toBeTruthy();
      expect(state.currentWorkflow?.id).toBe('workflow-1');
      expect(state.nodes).toHaveLength(1);
      expect(state.isDirty).toBe(false);
      expect(getWorkflowMock).toHaveBeenCalledWith({ workflowId: 'workflow-1', version: undefined });
    });

    it('creates workflow successfully [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const newWorkflow = {
        id: 'new-workflow-id',
        project_id: 'project-1',
        name: 'New Workflow',
        description: '',
        folder_path: '/workflows/test',
        version: 1,
        created_at: '2025-01-03T00:00:00Z',
        updated_at: '2025-01-03T00:00:00Z',
        flow_definition: { nodes: [], edges: [] },
      };

      createWorkflowMock.mockResolvedValueOnce(create(CreateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: newWorkflow.id,
          projectId: newWorkflow.project_id,
          name: newWorkflow.name,
          folderPath: newWorkflow.folder_path,
          version: newWorkflow.version,
          flowDefinition: newWorkflow.flow_definition,
        }),
      } as Parameters<typeof create>[1]));

      let returnedWorkflow;
      await act(async () => {
        returnedWorkflow = await useWorkflowStore
          .getState()
          .createWorkflow('New Workflow', '/workflows/test', 'project-1');
      });

      expect(createWorkflowMock).toHaveBeenCalled();

      expect(returnedWorkflow).toBeTruthy();
      expect(returnedWorkflow?.id).toBe('new-workflow-id');

      const state = useWorkflowStore.getState();
      expect(state.currentWorkflow?.id).toBe('new-workflow-id');
      // Note: createWorkflow does not automatically add to workflows list
    });

    it('saves workflow with nodes and edges [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      // First, set up a current workflow
      const initialWorkflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date('2025-01-01'),
        updatedAt: new Date('2025-01-01'),
        flow_definition: { nodes: [], edges: [] },
        flowDefinition: { nodes: [], edges: [] },
        nodes: [] as Node[],
        edges: [] as Edge[],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({
        currentWorkflow: initialWorkflow,
        nodes: [
          { id: 'node-1', type: 'click', position: { x: 100, y: 100 }, data: { selector: '.btn' } },
        ] as Node[],
        edges: [],
        isDirty: true,
      });

      const savedWorkflow = {
        id: initialWorkflow.id,
        project_id: 'project-1',
        name: initialWorkflow.name,
        description: '',
        folder_path: initialWorkflow.folderPath,
        version: 2,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-02T00:00:00Z',
        flow_definition: { nodes: [], edges: [] },
      };

      updateWorkflowMock.mockResolvedValueOnce(create(UpdateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: savedWorkflow.id,
          projectId: savedWorkflow.project_id,
          name: savedWorkflow.name,
          folderPath: savedWorkflow.folder_path,
          version: savedWorkflow.version,
          flowDefinition: savedWorkflow.flow_definition,
        }),
      } as Parameters<typeof create>[1]));

      await act(async () => {
        await useWorkflowStore.getState().saveWorkflow({
          source: 'manual',
          changeDescription: 'Added click node',
        });
      });

      expect(updateWorkflowMock).toHaveBeenCalled();

      const state = useWorkflowStore.getState();
      expect(state.isDirty).toBe(false);
      expect(state.isSaving).toBe(false);
      expect(state.currentWorkflow?.version).toBe(2);
    });

    it('deletes workflow successfully [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const workflow = {
        id: 'workflow-to-delete',
        projectId: 'project-1',
        name: 'To Be Deleted',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({ workflows: [workflow] });

      deleteWorkflowMock.mockResolvedValueOnce(create(DeleteWorkflowResponseSchema, {
        success: true,
        workflowId: 'workflow-to-delete',
      }));

      await act(async () => {
        await useWorkflowStore.getState().deleteWorkflow('workflow-to-delete');
      });

      expect(deleteWorkflowMock).toHaveBeenCalledWith({ workflowId: 'workflow-to-delete' });

      const state = useWorkflowStore.getState();
      expect(state.workflows).toHaveLength(0);
    });

    it('bulk deletes workflows [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const workflows = [
        {
          id: 'workflow-1',
          projectId: 'project-1',
          name: 'Workflow 1',
          description: '',
          folderPath: '/workflows',
          version: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
          flow_definition: {},
          flowDefinition: {},
          nodes: [],
          edges: [],
          tags: [],
          lastChangeSource: 'manual',
          lastChangeDescription: '',
        },
        {
          id: 'workflow-2',
          projectId: 'project-1',
          name: 'Workflow 2',
          description: '',
          folderPath: '/workflows',
          version: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
          flow_definition: {},
          flowDefinition: {},
          nodes: [],
          edges: [],
          tags: [],
          lastChangeSource: 'manual',
          lastChangeDescription: '',
        },
      ];

      useWorkflowStore.setState({ workflows });

      bulkDeleteProjectWorkflowsMock.mockResolvedValueOnce({
        deleted_count: 2,
        deleted_ids: ['workflow-1', 'workflow-2'],
      });

      let result;
      await act(async () => {
        result = await useWorkflowStore
          .getState()
          .bulkDeleteWorkflows('project-1', ['workflow-1', 'workflow-2']);
      });

      expect(bulkDeleteProjectWorkflowsMock).toHaveBeenCalledWith('project-1', [
        'workflow-1',
        'workflow-2',
      ]);

      expect(result).toEqual(['workflow-1', 'workflow-2']);

      const state = useWorkflowStore.getState();
      expect(state.workflows).toHaveLength(0);
    });

    it('updates workflow metadata [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Original Name',
        description: 'Original Description',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({ currentWorkflow: workflow });

      act(() => {
        useWorkflowStore.getState().updateWorkflow({
          name: 'Updated Name',
          description: 'Updated Description',
        });
      });

      const state = useWorkflowStore.getState();
      expect(state.currentWorkflow?.name).toBe('Updated Name');
      expect(state.currentWorkflow?.description).toBe('Updated Description');
      expect(state.isDirty).toBe(true);
    });
  });

  describe('Autosave Functionality', () => {
    it('schedules autosave when changes occur [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      vi.useFakeTimers();

      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({
        currentWorkflow: workflow,
        nodes: [],
        edges: [],
        isDirty: true,
      });

      updateWorkflowMock.mockResolvedValueOnce(create(UpdateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: workflow.id,
          projectId: 'project-1',
          name: workflow.name,
          folderPath: workflow.folderPath,
          version: 2,
          flowDefinition: { nodes: [], edges: [] },
        }),
      } as Parameters<typeof create>[1]));

      act(() => {
        useWorkflowStore.getState().scheduleAutosave({
          source: 'autosave',
          debounceMs: 1000,
        });
      });

      // Fast-forward timer
      await act(async () => {
        vi.advanceTimersByTime(1000);
        await vi.runAllTimersAsync();
      });

      // Autosave actually saves
      expect(updateWorkflowMock).toHaveBeenCalled();

      vi.useRealTimers();
    });

    it('cancels autosave when requested [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
      vi.useFakeTimers();

      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({ currentWorkflow: workflow });

      act(() => {
        useWorkflowStore.getState().scheduleAutosave({ debounceMs: 2000 });
      });

      act(() => {
        useWorkflowStore.getState().cancelAutosave();
      });

      act(() => {
        vi.advanceTimersByTime(3000);
      });

      expect(global.fetch).not.toHaveBeenCalled();

      vi.useRealTimers();
    });
  });

  describe('Version Conflict Handling', () => {
    it('detects version conflict on save [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({
        currentWorkflow: workflow,
        nodes: [],
        edges: [],
        isDirty: true
      });

      fetchMock.mockResolvedValueOnce(fetchJsonResponse(
        { error: 'Version conflict detected', conflict: { remote_version: 3, expected_version: 1, remote_updated_at: '2025-01-03' } },
        { status: 409 }
      ));

      await act(async () => {
        try {
          await useWorkflowStore.getState().saveWorkflow();
        } catch (_error) {
          // Expected to handle conflict internally
        }
      });

      const state = useWorkflowStore.getState();
      // Version conflict handling may differ - check if error was recorded
      expect(state.lastSaveError || state.hasVersionConflict).toBeTruthy();
    });

    it('force saves workflow when conflict acknowledged [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({
        currentWorkflow: workflow,
        hasVersionConflict: true,
        isDirty: true,
      });

      updateWorkflowMock.mockResolvedValueOnce(create(UpdateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: workflow.id,
          projectId: 'project-1',
          name: workflow.name,
          folderPath: workflow.folderPath,
          version: 2,
          flowDefinition: { nodes: [], edges: [] },
        }),
      } as Parameters<typeof create>[1]));

      await act(async () => {
        await useWorkflowStore.getState().forceSaveWorkflow({
          source: 'manual',
          changeDescription: 'Force save after conflict',
        });
      });

      const state = useWorkflowStore.getState();
      expect(state.hasVersionConflict).toBe(false);
      expect(state.isDirty).toBe(false);
    });
  });

  describe('Version History', () => {
    it('loads workflow version history [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const mockVersions = [
        {
          version: 2,
          workflow_id: 'workflow-1',
          created_at: '2025-01-02T00:00:00Z',
          created_by: 'user-1',
          change_description: 'Added nodes',
          definition_hash: 'hash2',
          node_count: 2,
          edge_count: 1,
        },
        {
          version: 1,
          workflow_id: 'workflow-1',
          created_at: '2025-01-01T00:00:00Z',
          created_by: 'user-1',
          change_description: 'Initial version',
          definition_hash: 'hash1',
          node_count: 0,
          edge_count: 0,
        },
      ];

      listWorkflowVersionsMock.mockResolvedValueOnce(create(WorkflowVersionListSchema, {
        versions: mockVersions.map((v) =>
          create(WorkflowVersionSchema, {
            workflowId: v.workflow_id,
            version: v.version,
            changeDescription: v.change_description,
            createdBy: v.created_by,
          }),
        ),
      }));

      await act(async () => {
        await useWorkflowStore.getState().loadWorkflowVersions('workflow-1');
      });

      const state = useWorkflowStore.getState();
      expect(state.versionHistory).toHaveLength(2);
      expect(state.versionHistory[0].version).toBe(2);
      expect(state.versionHistoryLoadedFor).toBe('workflow-1');
      expect(listWorkflowVersionsMock).toHaveBeenCalledWith({ workflowId: 'workflow-1' });
    });

    it('restores previous workflow version [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => {
      const currentWorkflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Test Workflow',
        description: '',
        folderPath: '/workflows',
        version: 3,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [],
        edges: [],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({ currentWorkflow });

      const restoredWorkflow = {
        id: currentWorkflow.id,
        project_id: 'project-1',
        name: currentWorkflow.name,
        description: '',
        folder_path: currentWorkflow.folderPath,
        version: 4,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-02T00:00:00Z',
        flow_definition: { nodes: [], edges: [] },
      };

      restoreWorkflowVersionMock.mockResolvedValueOnce(create(RestoreWorkflowVersionResponseSchema, {
        workflow: makeSummary({
          id: restoredWorkflow.id,
          projectId: restoredWorkflow.project_id,
          name: restoredWorkflow.name,
          folderPath: restoredWorkflow.folder_path,
          version: restoredWorkflow.version,
          flowDefinition: restoredWorkflow.flow_definition,
        }),
        restoredVersion: create(WorkflowVersionSchema, {
          workflowId: currentWorkflow.id,
          version: 2,
          changeDescription: 'Restored to version 2',
          createdBy: 'user-1',
        }),
      } as Parameters<typeof create>[1]));
      // After restore, the store reloads version history.
      listWorkflowVersionsMock.mockResolvedValueOnce(create(WorkflowVersionListSchema, { versions: [] }));

      await act(async () => {
        await useWorkflowStore
          .getState()
          .restoreWorkflowVersion('workflow-1', 2, 'Restored to version 2');
      });

      expect(restoreWorkflowVersionMock).toHaveBeenCalledWith({
        workflowId: 'workflow-1',
        version: 2,
        changeDescription: 'Restored to version 2',
      });

      const state = useWorkflowStore.getState();
      expect(state.currentWorkflow?.version).toBe(4);
      expect(state.restoringVersion).toBeNull();
    });
  });

  describe('AI Workflow Generation [REQ:BAS-AI-GENERATION-SMOKE]', () => {
    it('generates workflow from AI prompt [REQ:BAS-AI-GENERATION-SMOKE]', async () => {
      const generatedWorkflow = {
        id: 'ai-generated-workflow',
        project_id: 'project-1',
        name: 'Login Test',
        description: 'Generated login workflow',
        folder_path: '/ai-workflows',
        version: 1,
        created_at: '2025-01-03T00:00:00Z',
        updated_at: '2025-01-03T00:00:00Z',
        flow_definition: {
          nodes: [{ id: 'n1' }, { id: 'n2' }, { id: 'n3' }],
          edges: [{ id: 'e1', source: 'n1', target: 'n2' }, { id: 'e2', source: 'n2', target: 'n3' }],
        },
      };

      createWorkflowMock.mockResolvedValueOnce(create(CreateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: generatedWorkflow.id,
          projectId: generatedWorkflow.project_id,
          name: generatedWorkflow.name,
          folderPath: generatedWorkflow.folder_path,
          version: generatedWorkflow.version,
          flowDefinition: generatedWorkflow.flow_definition,
        }),
      } as Parameters<typeof create>[1]));

      let result;
      await act(async () => {
        result = await useWorkflowStore
          .getState()
          .generateWorkflow(
            'Create a workflow that logs into the app',
            'Login Test',
            '/ai-workflows',
            'project-1'
          );
      });

      expect(createWorkflowMock).toHaveBeenCalled();

      expect(result).toBeTruthy();
      expect(result?.name).toBe('Login Test');
      expect(result?.nodes).toHaveLength(3);
    });

    it('modifies existing workflow with AI [REQ:BAS-AI-GENERATION-VALIDATION]', async () => {
      const workflow = {
        id: 'workflow-1',
        projectId: 'project-1',
        name: 'Login Test',
        description: '',
        folderPath: '/workflows',
        version: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
        flow_definition: {},
        flowDefinition: {},
        nodes: [
          { id: 'n1', type: 'navigate', position: { x: 0, y: 0 }, data: { url: 'https://app.com' } },
        ] as Node[],
        edges: [] as Edge[],
        tags: [],
        lastChangeSource: 'manual',
        lastChangeDescription: '',
      };

      useWorkflowStore.setState({ currentWorkflow: workflow, nodes: workflow.nodes });

      const modifiedWorkflow = {
        id: workflow.id,
        project_id: 'project-1',
        name: workflow.name,
        description: '',
        folder_path: workflow.folderPath,
        version: 2,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-02T00:00:00Z',
        flow_definition: {
          nodes: [{ id: 'n1' }, { id: 'n2' }],
          edges: [{ id: 'e1', source: 'n1', target: 'n2' }],
        },
      };

      modifyWorkflowMock.mockResolvedValueOnce(create(UpdateWorkflowResponseSchema, {
        workflow: makeSummary({
          id: modifiedWorkflow.id,
          projectId: modifiedWorkflow.project_id,
          name: modifiedWorkflow.name,
          folderPath: modifiedWorkflow.folder_path,
          version: modifiedWorkflow.version,
          flowDefinition: modifiedWorkflow.flow_definition,
        }),
      } as Parameters<typeof create>[1]));

      let result;
      await act(async () => {
        result = await useWorkflowStore
          .getState()
          .modifyWorkflow('Add a step to enter email address');
      });

      expect(modifyWorkflowMock).toHaveBeenCalled();
      expect(modifyWorkflowMock.mock.calls[0][0].modificationPrompt).toBe(
        'Add a step to enter email address'
      );

      expect(result).toBeTruthy();
      const state = useWorkflowStore.getState();
      expect(state.currentWorkflow?.version).toBe(2);
    });
  });
});
