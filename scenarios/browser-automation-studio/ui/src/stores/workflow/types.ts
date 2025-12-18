import type { Node, Edge } from 'reactflow';

// ============================================================================
// Error Types
// ============================================================================

export type SaveErrorType = 'conflict' | 'network' | 'server';

export interface SaveErrorState {
  type: SaveErrorType;
  message: string;
  status?: number;
}

// ============================================================================
// Viewport Types
// ============================================================================

export type ViewportPreset = 'desktop' | 'mobile' | 'custom';

export interface ExecutionViewportSettings {
  width: number;
  height: number;
  preset?: ViewportPreset;
}

// ============================================================================
// Workflow Types
// ============================================================================

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

export interface WorkflowConflictMetadata {
  detectedAt: Date;
  remoteVersion: number;
  remoteUpdatedAt: Date;
  changeDescription: string;
  changeSource: string;
  nodeCount: number;
  edgeCount: number;
}

// ============================================================================
// Save Options
// ============================================================================

export type SaveWorkflowOptions = {
  source?: string;
  changeDescription?: string;
  force?: boolean;
  skipConflictRetry?: boolean;
};

export type AutosaveOptions = SaveWorkflowOptions & { debounceMs?: number };

// ============================================================================
// Store Interface
// ============================================================================

export interface WorkflowStore {
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

// ============================================================================
// Helper Types
// ============================================================================

export type WorkflowLoadState = Pick<
  WorkflowStore,
  | 'currentWorkflow'
  | 'nodes'
  | 'edges'
  | 'isDirty'
  | 'isSaving'
  | 'lastSavedAt'
  | 'lastSavedFingerprint'
  | 'draftFingerprint'
  | 'hasVersionConflict'
  | 'lastSaveError'
  | 'versionHistory'
  | 'versionHistoryLoadedFor'
  | 'versionHistoryError'
  | 'conflictWorkflow'
  | 'conflictMetadata'
>;
