import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { parseProjectList } from '../utils/projectProto';

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

// Helper to normalize workflow response
const normalizeRecentWorkflow = (raw: Record<string, unknown>, projects: Map<string, string>): RecentWorkflow => {
  const projectId = String(raw.project_id ?? raw.projectId ?? '');
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? 'Untitled'),
    projectId,
    projectName: projects.get(projectId) ?? 'Unknown Project',
    updatedAt: new Date(String(raw.updated_at ?? raw.updatedAt ?? new Date().toISOString())),
    folderPath: String(raw.folder_path ?? raw.folderPath ?? '/'),
  };
};

// Helper to normalize execution response
const normalizeRecentExecution = (raw: Record<string, unknown>, workflowNames: Map<string, { name: string; projectId?: string; projectName?: string }>): RecentExecution => {
  const workflowId = String(raw.workflow_id ?? raw.workflowId ?? '');
  const workflowInfo = workflowNames.get(workflowId);
  const statusValue = String(raw.status ?? 'pending');
  const validStatuses = ['pending', 'running', 'completed', 'failed', 'cancelled'];
  const status = validStatuses.includes(statusValue) ? statusValue as RecentExecution['status'] : 'pending';

  return {
    id: String(raw.id ?? raw.execution_id ?? ''),
    workflowId,
    workflowName: workflowInfo?.name ?? 'Unknown Workflow',
    projectId: workflowInfo?.projectId,
    projectName: workflowInfo?.projectName,
    status,
    startedAt: new Date(String(raw.started_at ?? raw.startedAt ?? new Date().toISOString())),
    completedAt: raw.completed_at || raw.completedAt
      ? new Date(String(raw.completed_at ?? raw.completedAt))
      : undefined,
    error: raw.error ? String(raw.error) : undefined,
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

          // Fetch projects first to get names
          const projectsResponse = await fetch(`${config.API_URL}/projects`);
          const projectsData = await projectsResponse.json();
          const projectEntries = parseProjectList(projectsData);
          const projectsMap = new Map<string, string>();
          projectEntries.forEach((p) => projectsMap.set(p.id, p.name));

          // Fetch recent workflows (sorted by updated_at desc on server)
          const response = await fetch(`${config.API_URL}/workflows?limit=10`);
          if (!response.ok) {
            throw new Error(`Failed to fetch workflows: ${response.status}`);
          }
          const data = await response.json();
          const workflows = Array.isArray(data.workflows)
            ? data.workflows.map((w: Record<string, unknown>) => normalizeRecentWorkflow(w, projectsMap))
            : [];

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

          // Fetch executions first
          const response = await fetch(`${config.API_URL}/executions?limit=10`);
          if (!response.ok) {
            throw new Error(`Failed to fetch executions: ${response.status}`);
          }
          const data = await response.json();
          const rawExecutions = Array.isArray(data.executions) ? data.executions : [];

          // Collect unique workflow IDs
          const workflowIds = new Set<string>();
          rawExecutions.forEach((e: Record<string, unknown>) => {
            const wfId = String(e.workflow_id ?? e.workflowId ?? '');
            if (wfId) workflowIds.add(wfId);
          });

          // Fetch workflow details to get names and project info
          const workflowNames = new Map<string, { name: string; projectId?: string; projectName?: string }>();

          // Fetch projects for names
          const projectsResponse = await fetch(`${config.API_URL}/projects`);
          const projectsData = await projectsResponse.json();
          const projects = parseProjectList(projectsData);
          const projectsMap = new Map<string, string>();
          projects.forEach((p) => projectsMap.set(p.id, p.name));

          // Fetch workflows for names
          const workflowsResponse = await fetch(`${config.API_URL}/workflows?limit=100`);
          const workflowsData = await workflowsResponse.json();
          if (Array.isArray(workflowsData.workflows)) {
            workflowsData.workflows.forEach((w: Record<string, unknown>) => {
              const projectId = String(w.project_id ?? w.projectId ?? '');
              workflowNames.set(String(w.id), {
                name: String(w.name ?? 'Untitled'),
                projectId,
                projectName: projectsMap.get(projectId),
              });
            });
          }

          const executions = rawExecutions.map((e: Record<string, unknown>) =>
            normalizeRecentExecution(e, workflowNames)
          );

          // Sort by startedAt descending
          executions.sort((a: RecentExecution, b: RecentExecution) => b.startedAt.getTime() - a.startedAt.getTime());

          // Separate running from completed
          const running = executions.filter((e: RecentExecution) => e.status === 'running' || e.status === 'pending');
          const completed = executions.filter((e: RecentExecution) => e.status !== 'running' && e.status !== 'pending');

          set({
            recentExecutions: completed.slice(0, 5),
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

          const data = await response.json();
          const rawExecutions = Array.isArray(data.executions) ? data.executions : [];

          // Fetch workflow names
          const workflowNames = new Map<string, { name: string; projectId?: string; projectName?: string }>();
          const workflowsResponse = await fetch(`${config.API_URL}/workflows?limit=100`);
          const workflowsData = await workflowsResponse.json();

          // Fetch projects for names
          const projectsResponse = await fetch(`${config.API_URL}/projects`);
          const projectsData = await projectsResponse.json();
          const projects = parseProjectList(projectsData);
          const projectsMap = new Map<string, string>();
          projects.forEach((p) => projectsMap.set(p.id, p.name));

          if (Array.isArray(workflowsData.workflows)) {
            workflowsData.workflows.forEach((w: Record<string, unknown>) => {
              const projectId = String(w.project_id ?? w.projectId ?? '');
              workflowNames.set(String(w.id), {
                name: String(w.name ?? 'Untitled'),
                projectId,
                projectName: projectsMap.get(projectId),
              });
            });
          }

          const running = rawExecutions
            .filter((e: Record<string, unknown>) => {
              const status = String(e.status ?? '');
              return status === 'running' || status === 'pending';
            })
            .map((e: Record<string, unknown>) => normalizeRecentExecution(e, workflowNames));

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
