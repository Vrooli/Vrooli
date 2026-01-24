import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { parseProjectList } from '../utils/projectProto';
import { safeParse } from '../shared/api/safeParse';
import {
  WorkflowsListResponseSchema,
  ExecutionsListResponseSchema,
  type WorkflowItem,
  type ExecutionItem,
} from '../shared/api/schemas';

export interface RecentWorkflow {
  id: string;
  name: string;
  projectId: string;
  projectName: string;
  updatedAt: Date;
  folderPath: string;
}

export interface RecentExecution {
  id: string;
  workflowId: string;
  workflowName: string;
  projectId?: string;
  projectName?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  startedAt: Date;
  completedAt?: Date;
  error?: string;
}

export interface FavoriteWorkflow {
  id: string;
  name: string;
  projectId: string;
  projectName: string;
  addedAt: Date;
}

interface DashboardState {
  // Recent items
  recentWorkflows: RecentWorkflow[];
  recentExecutions: RecentExecution[];
  runningExecutions: RecentExecution[];

  // Last edited workflow (for "Continue Editing")
  lastEditedWorkflow: RecentWorkflow | null;

  // Favorites (persisted)
  favoriteWorkflows: FavoriteWorkflow[];

  // Loading states
  isLoadingRecent: boolean;
  isLoadingExecutions: boolean;

  // Actions
  fetchRecentWorkflows: () => Promise<void>;
  fetchRecentExecutions: () => Promise<void>;
  fetchRunningExecutions: () => Promise<void>;
  setLastEditedWorkflow: (workflow: RecentWorkflow | null) => void;
  addFavorite: (workflow: FavoriteWorkflow) => void;
  removeFavorite: (workflowId: string) => void;
  isFavorite: (workflowId: string) => boolean;
  clearLastEdited: () => void;
}

const PROJECTS_CACHE_TTL_MS = 30_000;
const WORKFLOW_NAMES_CACHE_TTL_MS = 30_000;

let projectsCache: {
  fetchedAt: number;
  value: Map<string, string>;
  inFlight: Promise<Map<string, string>> | null;
} = { fetchedAt: 0, value: new Map(), inFlight: null };

let workflowNamesCache: {
  fetchedAt: number;
  value: Map<string, { name: string; projectId?: string; projectName?: string }>;
  inFlight: Promise<Map<string, { name: string; projectId?: string; projectName?: string }>> | null;
} = { fetchedAt: 0, value: new Map(), inFlight: null };

const getProjectsMapCached = async (apiBase: string): Promise<Map<string, string>> => {
  const now = Date.now();
  if (projectsCache.value.size > 0 && now - projectsCache.fetchedAt < PROJECTS_CACHE_TTL_MS) {
    return projectsCache.value;
  }
  if (projectsCache.inFlight) {
    return projectsCache.inFlight;
  }

  projectsCache.inFlight = (async () => {
    const projectsResponse = await fetch(`${apiBase}/projects`);
    const projectsData = await projectsResponse.json();
    const projectEntries = parseProjectList(projectsData);
    const next = new Map<string, string>();
    projectEntries.forEach((p) => next.set(p.id, p.name));
    projectsCache = { fetchedAt: Date.now(), value: next, inFlight: null };
    return next;
  })();

  return projectsCache.inFlight;
};

const getWorkflowNamesCached = async (
  apiBase: string,
  projectsMap: Map<string, string>,
): Promise<Map<string, { name: string; projectId?: string; projectName?: string }>> => {
  const now = Date.now();
  if (workflowNamesCache.value.size > 0 && now - workflowNamesCache.fetchedAt < WORKFLOW_NAMES_CACHE_TTL_MS) {
    return workflowNamesCache.value;
  }
  if (workflowNamesCache.inFlight) {
    return workflowNamesCache.inFlight;
  }

  workflowNamesCache.inFlight = (async () => {
    const workflowsResponse = await fetch(`${apiBase}/workflows?limit=100`);
    const rawData = await workflowsResponse.json() as Record<string, unknown>;
    // Ensure workflows array exists for schema validation
    const normalizedData = { workflows: Array.isArray(rawData?.workflows) ? rawData.workflows : [] };
    const result = safeParse(WorkflowsListResponseSchema, normalizedData, 'WorkflowNamesCache');
    const next = new Map<string, { name: string; projectId?: string; projectName?: string }>();
    if (result.success) {
      result.data.workflows.forEach((w) => {
        const projectId = w.project_id ?? w.projectId ?? '';
        next.set(w.id, {
          name: w.name ?? 'Untitled',
          projectId,
          projectName: projectsMap.get(projectId),
        });
      });
    }
    workflowNamesCache = { fetchedAt: Date.now(), value: next, inFlight: null };
    return next;
  })();

  return workflowNamesCache.inFlight;
};

// Helper to normalize workflow response
const normalizeRecentWorkflow = (raw: WorkflowItem, projects: Map<string, string>): RecentWorkflow => {
  const projectId = raw.project_id ?? raw.projectId ?? '';
  return {
    id: raw.id,
    name: raw.name ?? 'Untitled',
    projectId,
    projectName: projects.get(projectId) ?? 'Unknown Project',
    updatedAt: new Date(raw.updated_at ?? raw.updatedAt ?? new Date().toISOString()),
    folderPath: raw.folder_path ?? raw.folderPath ?? '/',
  };
};

// Helper to normalize proto enum status to lowercase
const normalizeExecutionStatus = (rawStatus: unknown): RecentExecution['status'] => {
  const statusStr = String(rawStatus ?? 'pending').toUpperCase();
  // Handle proto enum format (e.g., "EXECUTION_STATUS_RUNNING") and lowercase format
  if (statusStr.includes('RUNNING')) return 'running';
  if (statusStr.includes('COMPLETED')) return 'completed';
  if (statusStr.includes('FAILED')) return 'failed';
  if (statusStr.includes('CANCELLED')) return 'cancelled';
  if (statusStr.includes('PENDING')) return 'pending';
  return 'pending';
};

// Helper to normalize execution response
const normalizeRecentExecution = (raw: ExecutionItem, workflowNames: Map<string, { name: string; projectId?: string; projectName?: string }>): RecentExecution => {
  const workflowId = raw.workflow_id ?? raw.workflowId ?? '';
  const workflowInfo = workflowNames.get(workflowId);
  const status = normalizeExecutionStatus(raw.status);

  return {
    id: raw.id ?? raw.executionId ?? raw.execution_id ?? '',
    workflowId,
    workflowName: workflowInfo?.name ?? 'Unknown Workflow',
    projectId: workflowInfo?.projectId,
    projectName: workflowInfo?.projectName,
    status,
    startedAt: new Date(raw.started_at ?? raw.startedAt ?? new Date().toISOString()),
    completedAt: raw.completed_at ?? raw.completedAt
      ? new Date(raw.completed_at ?? raw.completedAt ?? '')
      : undefined,
    error: raw.error ?? undefined,
  };
};

export const useDashboardStore = create<DashboardState>()(
  persist(
    (set, get) => ({
      recentWorkflows: [],
      recentExecutions: [],
      runningExecutions: [],
      lastEditedWorkflow: null,
      favoriteWorkflows: [],
      isLoadingRecent: false,
      isLoadingExecutions: false,

      fetchRecentWorkflows: async () => {
        set({ isLoadingRecent: true });
        try {
          const config = await getConfig();

          const projectsMap = await getProjectsMapCached(config.API_URL);

          // Fetch recent workflows (sorted by updated_at desc on server)
          const response = await fetch(`${config.API_URL}/workflows?limit=10`);
          if (!response.ok) {
            throw new Error(`Failed to fetch workflows: ${response.status}`);
          }
          const rawData = await response.json() as Record<string, unknown>;
          // Ensure workflows array exists for schema validation
          const normalizedData = { workflows: Array.isArray(rawData?.workflows) ? rawData.workflows : [] };
          const result = safeParse(WorkflowsListResponseSchema, normalizedData, 'WorkflowsList');
          if (!result.success) {
            throw new Error(result.error);
          }

          const workflows = result.data.workflows.map((w) => normalizeRecentWorkflow(w, projectsMap));

          // Sort by updatedAt descending
          workflows.sort((a: RecentWorkflow, b: RecentWorkflow) => b.updatedAt.getTime() - a.updatedAt.getTime());

          set({ recentWorkflows: workflows.slice(0, 5), isLoadingRecent: false });
        } catch (error) {
          logger.error('Failed to fetch recent workflows', { component: 'DashboardStore', action: 'fetchRecentWorkflows' }, error);
          set({ isLoadingRecent: false });
        }
      },

      fetchRecentExecutions: async () => {
        set({ isLoadingExecutions: true });
        try {
          const config = await getConfig();

          // Fetch executions - use higher limit to show more in dashboard
          const response = await fetch(`${config.API_URL}/executions?limit=50`);
          if (!response.ok) {
            throw new Error(`Failed to fetch executions: ${response.status}`);
          }
          const rawData = await response.json() as Record<string, unknown>;
          // Ensure executions array exists for schema validation
          const normalizedData = { executions: Array.isArray(rawData?.executions) ? rawData.executions : [] };
          const result = safeParse(ExecutionsListResponseSchema, normalizedData, 'ExecutionsList');
          if (!result.success) {
            throw new Error(result.error);
          }

          const projectsMap = await getProjectsMapCached(config.API_URL);
          const workflowNames = await getWorkflowNamesCached(config.API_URL, projectsMap);

          const executions = result.data.executions.map((e) =>
            normalizeRecentExecution(e, workflowNames)
          );

          // Sort by startedAt descending
          executions.sort((a: RecentExecution, b: RecentExecution) => b.startedAt.getTime() - a.startedAt.getTime());

          // Separate running from completed
          const running = executions.filter((e: RecentExecution) => e.status === 'running' || e.status === 'pending');
          const completed = executions.filter((e: RecentExecution) => e.status !== 'running' && e.status !== 'pending');

          set({
            recentExecutions: completed,
            runningExecutions: running,
            isLoadingExecutions: false
          });
        } catch (error) {
          logger.error('Failed to fetch recent executions', { component: 'DashboardStore', action: 'fetchRecentExecutions' }, error);
          set({ isLoadingExecutions: false });
        }
      },

      fetchRunningExecutions: async () => {
        try {
          const config = await getConfig();
          const response = await fetch(`${config.API_URL}/executions?limit=50`);
          if (!response.ok) return;

          const rawData = await response.json() as Record<string, unknown>;
          // Ensure executions array exists for schema validation
          const normalizedData = { executions: Array.isArray(rawData?.executions) ? rawData.executions : [] };
          const result = safeParse(ExecutionsListResponseSchema, normalizedData, 'RunningExecutionsList');
          if (!result.success) {
            logger.warn('Failed to parse running executions', { component: 'DashboardStore', action: 'fetchRunningExecutions', error: result.error });
            return;
          }

          const projectsMap = await getProjectsMapCached(config.API_URL);
          const workflowNames = await getWorkflowNamesCached(config.API_URL, projectsMap);

          const running = result.data.executions
            .filter((e) => {
              const status = normalizeExecutionStatus(e.status);
              return status === 'running' || status === 'pending';
            })
            .map((e) => normalizeRecentExecution(e, workflowNames));

          set({ runningExecutions: running });
        } catch (error) {
          logger.error('Failed to fetch running executions', { component: 'DashboardStore', action: 'fetchRunningExecutions' }, error);
        }
      },

      setLastEditedWorkflow: (workflow) => {
        set({ lastEditedWorkflow: workflow });
      },

      addFavorite: (workflow) => {
        const current = get().favoriteWorkflows;
        if (!current.find(f => f.id === workflow.id)) {
          set({ favoriteWorkflows: [workflow, ...current] });
        }
      },

      removeFavorite: (workflowId) => {
        set({
          favoriteWorkflows: get().favoriteWorkflows.filter(f => f.id !== workflowId)
        });
      },

      isFavorite: (workflowId) => {
        return get().favoriteWorkflows.some(f => f.id === workflowId);
      },

      clearLastEdited: () => {
        set({ lastEditedWorkflow: null });
      },
    }),
    {
      name: 'browser-automation-studio-dashboard',
      partialize: (state) => ({
        lastEditedWorkflow: state.lastEditedWorkflow,
        favoriteWorkflows: state.favoriteWorkflows,
      }),
    }
  )
);
