import { create } from "zustand";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";
import type { Workflow } from "@stores/workflowStore";

// Extended Workflow interface with API response fields
export interface WorkflowWithStats extends Workflow {
  folder_path?: string;
  created_at?: string;
  updated_at?: string;
  project_id?: string;
  stats?: {
    execution_count: number;
    last_execution?: string;
    success_rate?: number;
  };
}

export type ProjectEntryKind = "folder" | "workflow_file" | "asset_file";

export interface ProjectEntry {
  id: string;
  project_id: string;
  path: string;
  kind: ProjectEntryKind;
  workflow_id?: string;
  metadata?: Record<string, unknown>;
}

export type ViewMode = "card" | "tree";
export type ActiveTab = "workflows" | "executions";

interface ProjectDetailState {
  // Current project context
  projectId: string | null;

  // Data
  workflows: WorkflowWithStats[];
  projectEntries: ProjectEntry[];

  // Loading/Error
  isLoading: boolean;
  error: string | null;
  projectEntriesLoading: boolean;
  projectEntriesError: string | null;

  // UI State
  searchTerm: string;
  viewMode: ViewMode;
  activeTab: ActiveTab;

  // Selection
  selectionMode: boolean;
  selectedWorkflows: Set<string>;

  // Tree state
  expandedFolders: Set<string>;
  selectedTreeFolder: string | null;
  dragSourcePath: string | null;
  dropTargetFolder: string | null;

  // Menu states
  showWorkflowActionsFor: string | null;
  showStatsPopover: boolean;
  showViewModeDropdown: boolean;
  showMoreMenu: boolean;
  showNewFileMenu: boolean;
  showEditProjectModal: boolean;

  // Operations in progress
  executionInProgress: Record<string, boolean>;
  isBulkDeleting: boolean;
  isDeletingProject: boolean;
  deletingWorkflowId: string | null;
  isImportingRecording: boolean;
}

interface ProjectDetailActions {
  // Initialize/reset for new project
  initializeForProject: (projectId: string) => void;
  reset: () => void;

  // Data fetching
  fetchWorkflows: (projectId: string) => Promise<void>;
  fetchProjectEntries: (projectId: string) => Promise<void>;
  setWorkflows: (workflows: WorkflowWithStats[]) => void;
  setProjectEntries: (entries: ProjectEntry[]) => void;

  // Error handling
  setError: (error: string | null) => void;
  clearError: () => void;

  // Search/View
  setSearchTerm: (term: string) => void;
  setViewMode: (mode: ViewMode) => void;
  setActiveTab: (tab: ActiveTab) => void;

  // Selection
  toggleSelectionMode: () => void;
  setSelectionMode: (mode: boolean) => void;
  toggleWorkflowSelection: (id: string) => void;
  selectAll: (workflowIds: string[]) => void;
  clearSelection: () => void;

  // Tree operations
  toggleFolder: (path: string) => void;
  setSelectedTreeFolder: (path: string | null) => void;
  setDragSourcePath: (path: string | null) => void;
  setDropTargetFolder: (folder: string | null) => void;

  // Menu states
  setShowWorkflowActionsFor: (id: string | null) => void;
  setShowStatsPopover: (show: boolean) => void;
  setShowViewModeDropdown: (show: boolean) => void;
  setShowMoreMenu: (show: boolean) => void;
  setShowNewFileMenu: (show: boolean) => void;
  setShowEditProjectModal: (show: boolean) => void;

  // Operation states
  setExecutionInProgress: (workflowId: string, inProgress: boolean) => void;
  setDeletingWorkflowId: (id: string | null) => void;
  setBulkDeleting: (deleting: boolean) => void;
  setIsDeletingProject: (deleting: boolean) => void;
  setIsImportingRecording: (importing: boolean) => void;

  // Workflow mutations
  removeWorkflow: (workflowId: string) => void;
  removeWorkflows: (workflowIds: string[]) => void;
}

const initialState: ProjectDetailState = {
  projectId: null,
  workflows: [],
  projectEntries: [],
  isLoading: false,
  error: null,
  projectEntriesLoading: false,
  projectEntriesError: null,
  searchTerm: "",
  viewMode: "card",
  activeTab: "workflows",
  selectionMode: false,
  selectedWorkflows: new Set(),
  expandedFolders: new Set(),
  selectedTreeFolder: null,
  dragSourcePath: null,
  dropTargetFolder: null,
  showWorkflowActionsFor: null,
  showStatsPopover: false,
  showViewModeDropdown: false,
  showMoreMenu: false,
  showNewFileMenu: false,
  showEditProjectModal: false,
  executionInProgress: {},
  isBulkDeleting: false,
  isDeletingProject: false,
  deletingWorkflowId: null,
  isImportingRecording: false,
};

export const useProjectDetailStore = create<ProjectDetailState & ProjectDetailActions>((set, get) => ({
  ...initialState,

  // Initialize/reset for new project
  initializeForProject: (projectId: string) => {
    const currentProjectId = get().projectId;
    if (currentProjectId !== projectId) {
      set({
        ...initialState,
        projectId,
      });
    }
  },

  reset: () => {
    set(initialState);
  },

  // Data fetching
  fetchWorkflows: async (projectId: string) => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/workflows`,
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch workflows: ${response.status}`);
      }
      const data = await response.json();
      const workflowList = data.workflows || [];
      set({
        workflows: workflowList,
        selectedWorkflows: new Set(),
        selectionMode: false,
        isLoading: false,
      });
    } catch (error) {
      logger.error(
        "Failed to fetch workflows",
        {
          component: "useProjectDetailStore",
          action: "fetchWorkflows",
          projectId,
        },
        error,
      );
      set({
        error: error instanceof Error ? error.message : "Failed to fetch workflows",
        isLoading: false,
      });
    }
  },

  fetchProjectEntries: async (projectId: string) => {
    set({ projectEntriesLoading: true, projectEntriesError: null });
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/tree`,
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch project files: ${response.status}`);
      }
      const payload = (await response.json()) as { entries?: ProjectEntry[] };
      set({
        projectEntries: Array.isArray(payload.entries) ? payload.entries : [],
        projectEntriesLoading: false,
      });
    } catch (error) {
      logger.error(
        "Failed to fetch project file tree",
        {
          component: "useProjectDetailStore",
          action: "fetchProjectEntries",
          projectId,
        },
        error,
      );
      set({
        projectEntriesError: error instanceof Error ? error.message : "Failed to fetch project files",
        projectEntriesLoading: false,
      });
    }
  },

  setWorkflows: (workflows) => set({ workflows }),

  setProjectEntries: (entries) => set({ projectEntries: entries }),

  // Error handling
  setError: (error) => set({ error }),
  clearError: () => set({ error: null }),

  // Search/View
  setSearchTerm: (term) => set({ searchTerm: term }),
  setViewMode: (mode) => set({ viewMode: mode }),
  setActiveTab: (tab) => set({ activeTab: tab }),

  // Selection
  toggleSelectionMode: () => {
    const { selectionMode } = get();
    if (selectionMode) {
      set({ selectionMode: false, selectedWorkflows: new Set() });
    } else {
      set({ selectionMode: true });
    }
  },

  setSelectionMode: (mode) => {
    if (!mode) {
      set({ selectionMode: false, selectedWorkflows: new Set() });
    } else {
      set({ selectionMode: true });
    }
  },

  toggleWorkflowSelection: (id) => {
    const { selectedWorkflows } = get();
    const next = new Set(selectedWorkflows);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    set({ selectedWorkflows: next });
  },

  selectAll: (workflowIds) => {
    set({ selectedWorkflows: new Set(workflowIds) });
  },

  clearSelection: () => {
    set({ selectedWorkflows: new Set() });
  },

  // Tree operations
  toggleFolder: (path) => {
    const { expandedFolders } = get();
    const next = new Set(expandedFolders);
    if (next.has(path)) {
      next.delete(path);
    } else {
      next.add(path);
    }
    set({ expandedFolders: next });
  },

  setSelectedTreeFolder: (path) => set({ selectedTreeFolder: path }),
  setDragSourcePath: (path) => set({ dragSourcePath: path }),
  setDropTargetFolder: (folder) => set({ dropTargetFolder: folder }),

  // Menu states
  setShowWorkflowActionsFor: (id) => set({ showWorkflowActionsFor: id }),
  setShowStatsPopover: (show) => set({ showStatsPopover: show }),
  setShowViewModeDropdown: (show) => set({ showViewModeDropdown: show }),
  setShowMoreMenu: (show) => set({ showMoreMenu: show }),
  setShowNewFileMenu: (show) => set({ showNewFileMenu: show }),
  setShowEditProjectModal: (show) => set({ showEditProjectModal: show }),

  // Operation states
  setExecutionInProgress: (workflowId, inProgress) => {
    const { executionInProgress } = get();
    set({
      executionInProgress: { ...executionInProgress, [workflowId]: inProgress },
    });
  },

  setDeletingWorkflowId: (id) => set({ deletingWorkflowId: id }),
  setBulkDeleting: (deleting) => set({ isBulkDeleting: deleting }),
  setIsDeletingProject: (deleting) => set({ isDeletingProject: deleting }),
  setIsImportingRecording: (importing) => set({ isImportingRecording: importing }),

  // Workflow mutations
  removeWorkflow: (workflowId) => {
    const { workflows } = get();
    set({ workflows: workflows.filter((w) => w.id !== workflowId) });
  },

  removeWorkflows: (workflowIds) => {
    const { workflows } = get();
    const deletedSet = new Set(workflowIds);
    set({ workflows: workflows.filter((w) => !deletedSet.has(w.id)) });
  },
}));

// Selector hooks for common derived state
export const useFilteredWorkflows = () => {
  return useProjectDetailStore((state) => {
    const { workflows, searchTerm } = state;
    if (!searchTerm) return workflows;
    const term = searchTerm.toLowerCase();
    return workflows.filter(
      (workflow) =>
        workflow.name.toLowerCase().includes(term) ||
        (workflow.description as string | undefined)?.toLowerCase().includes(term),
    );
  });
};

export const useIsAllSelected = () => {
  return useProjectDetailStore((state) => {
    const { selectedWorkflows, workflows } = state;
    return workflows.length > 0 && selectedWorkflows.size === workflows.length;
  });
};
