import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';

export interface Export {
  id: string;
  executionId: string;
  workflowId?: string;
  name: string;
  format: 'mp4' | 'gif' | 'json' | 'html';
  settings?: Record<string, unknown>;
  storageUrl?: string;
  thumbnailUrl?: string;
  fileSizeBytes?: number;
  durationMs?: number;
  frameCount?: number;
  aiCaption?: string;
  aiCaptionGeneratedAt?: Date;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  error?: string;
  createdAt: Date;
  updatedAt: Date;
  // Joined fields from API
  workflowName?: string;
  executionDate?: string;
}

export interface CreateExportInput {
  executionId: string;
  workflowId?: string;
  name: string;
  format: 'mp4' | 'gif' | 'json' | 'html';
  settings?: Record<string, unknown>;
  storageUrl?: string;
  thumbnailUrl?: string;
  fileSizeBytes?: number;
  durationMs?: number;
  frameCount?: number;
  status?: 'pending' | 'processing' | 'completed' | 'failed';
}

export interface UpdateExportInput {
  name?: string;
  settings?: Record<string, unknown>;
  storageUrl?: string;
  thumbnailUrl?: string;
  fileSizeBytes?: number;
  durationMs?: number;
  frameCount?: number;
  aiCaption?: string;
  status?: 'pending' | 'processing' | 'completed' | 'failed';
  error?: string;
}

interface ExportState {
  // Data
  exports: Export[];
  selectedExport: Export | null;

  // Loading states
  isLoading: boolean;
  isCreating: boolean;
  isUpdating: boolean;
  isDeleting: boolean;

  // Error state
  error: string | null;

  // Actions
  fetchExports: () => Promise<void>;
  fetchExportsByExecution: (executionId: string) => Promise<Export[]>;
  fetchExportsByWorkflow: (workflowId: string) => Promise<Export[]>;
  getExport: (id: string) => Promise<Export | null>;
  createExport: (input: CreateExportInput) => Promise<Export | null>;
  updateExport: (id: string, input: UpdateExportInput) => Promise<Export | null>;
  deleteExport: (id: string) => Promise<boolean>;
  setSelectedExport: (export_: Export | null) => void;
  clearError: () => void;
}

// Helper to normalize API response to Export interface
const normalizeExport = (raw: Record<string, unknown>): Export => {
  const statusValue = String(raw.status ?? 'completed');
  const validStatuses = ['pending', 'processing', 'completed', 'failed'];
  const status = validStatuses.includes(statusValue) ? statusValue as Export['status'] : 'completed';

  const formatValue = String(raw.format ?? 'mp4').toLowerCase();
  const validFormats = ['mp4', 'gif', 'json', 'html'];
  const format = validFormats.includes(formatValue) ? formatValue as Export['format'] : 'mp4';

  return {
    id: String(raw.id ?? ''),
    executionId: String(raw.execution_id ?? raw.executionId ?? ''),
    workflowId: raw.workflow_id || raw.workflowId ? String(raw.workflow_id ?? raw.workflowId) : undefined,
    name: String(raw.name ?? 'Untitled Export'),
    format,
    settings: raw.settings as Record<string, unknown> | undefined,
    storageUrl: raw.storage_url || raw.storageUrl ? String(raw.storage_url ?? raw.storageUrl) : undefined,
    thumbnailUrl: raw.thumbnail_url || raw.thumbnailUrl ? String(raw.thumbnail_url ?? raw.thumbnailUrl) : undefined,
    fileSizeBytes: typeof raw.file_size_bytes === 'number' ? raw.file_size_bytes :
                   typeof raw.fileSizeBytes === 'number' ? raw.fileSizeBytes : undefined,
    durationMs: typeof raw.duration_ms === 'number' ? raw.duration_ms :
                typeof raw.durationMs === 'number' ? raw.durationMs : undefined,
    frameCount: typeof raw.frame_count === 'number' ? raw.frame_count :
                typeof raw.frameCount === 'number' ? raw.frameCount : undefined,
    aiCaption: raw.ai_caption || raw.aiCaption ? String(raw.ai_caption ?? raw.aiCaption) : undefined,
    aiCaptionGeneratedAt: raw.ai_caption_generated_at || raw.aiCaptionGeneratedAt
      ? new Date(String(raw.ai_caption_generated_at ?? raw.aiCaptionGeneratedAt))
      : undefined,
    status,
    error: raw.error ? String(raw.error) : undefined,
    createdAt: new Date(String(raw.created_at ?? raw.createdAt ?? new Date().toISOString())),
    updatedAt: new Date(String(raw.updated_at ?? raw.updatedAt ?? new Date().toISOString())),
    workflowName: raw.workflow_name || raw.workflowName ? String(raw.workflow_name ?? raw.workflowName) : undefined,
    executionDate: raw.execution_date || raw.executionDate ? String(raw.execution_date ?? raw.executionDate) : undefined,
  };
};

export const useExportStore = create<ExportState>((set) => ({
  exports: [],
  selectedExport: null,
  isLoading: false,
  isCreating: false,
  isUpdating: false,
  isDeleting: false,
  error: null,

  fetchExports: async () => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports?limit=100`);

      if (!response.ok) {
        throw new Error(`Failed to fetch exports: ${response.status}`);
      }

      const data = await response.json();
      const exports = Array.isArray(data.exports)
        ? data.exports.map((e: Record<string, unknown>) => normalizeExport(e))
        : [];

      // Sort by createdAt descending
      exports.sort((a: Export, b: Export) => b.createdAt.getTime() - a.createdAt.getTime());

      set({ exports, isLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to fetch exports';
      logger.error('Failed to fetch exports', { component: 'ExportStore', action: 'fetchExports' }, error);
      set({ error: message, isLoading: false });
    }
  },

  fetchExportsByExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports?execution_id=${executionId}`);

      if (!response.ok) {
        throw new Error(`Failed to fetch exports: ${response.status}`);
      }

      const data = await response.json();
      const exports = Array.isArray(data.exports)
        ? data.exports.map((e: Record<string, unknown>) => normalizeExport(e))
        : [];

      return exports;
    } catch (error) {
      logger.error('Failed to fetch exports by execution', { component: 'ExportStore', action: 'fetchExportsByExecution', executionId }, error);
      return [];
    }
  },

  fetchExportsByWorkflow: async (workflowId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports?workflow_id=${workflowId}`);

      if (!response.ok) {
        throw new Error(`Failed to fetch exports: ${response.status}`);
      }

      const data = await response.json();
      const exports = Array.isArray(data.exports)
        ? data.exports.map((e: Record<string, unknown>) => normalizeExport(e))
        : [];

      return exports;
    } catch (error) {
      logger.error('Failed to fetch exports by workflow', { component: 'ExportStore', action: 'fetchExportsByWorkflow', workflowId }, error);
      return [];
    }
  },

  getExport: async (id: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports/${id}`);

      if (!response.ok) {
        if (response.status === 404) {
          return null;
        }
        throw new Error(`Failed to get export: ${response.status}`);
      }

      const data = await response.json();
      return data.export ? normalizeExport(data.export) : null;
    } catch (error) {
      logger.error('Failed to get export', { component: 'ExportStore', action: 'getExport', id }, error);
      return null;
    }
  },

  createExport: async (input: CreateExportInput) => {
    set({ isCreating: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          execution_id: input.executionId,
          workflow_id: input.workflowId,
          name: input.name,
          format: input.format,
          settings: input.settings,
          storage_url: input.storageUrl,
          thumbnail_url: input.thumbnailUrl,
          file_size_bytes: input.fileSizeBytes,
          duration_ms: input.durationMs,
          frame_count: input.frameCount,
          status: input.status ?? 'completed',
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message ?? `Failed to create export: ${response.status}`);
      }

      const data = await response.json();
      const newExport = data.export ? normalizeExport(data.export) : null;

      if (newExport) {
        // Add to the beginning of the exports list
        set(state => ({
          exports: [newExport, ...state.exports],
          isCreating: false,
        }));
      } else {
        set({ isCreating: false });
      }

      return newExport;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to create export';
      logger.error('Failed to create export', { component: 'ExportStore', action: 'createExport' }, error);
      set({ error: message, isCreating: false });
      return null;
    }
  },

  updateExport: async (id: string, input: UpdateExportInput) => {
    set({ isUpdating: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: input.name,
          settings: input.settings,
          storage_url: input.storageUrl,
          thumbnail_url: input.thumbnailUrl,
          file_size_bytes: input.fileSizeBytes,
          duration_ms: input.durationMs,
          frame_count: input.frameCount,
          ai_caption: input.aiCaption,
          status: input.status,
          error: input.error,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message ?? `Failed to update export: ${response.status}`);
      }

      const data = await response.json();
      const updatedExport = data.export ? normalizeExport(data.export) : null;

      if (updatedExport) {
        set(state => ({
          exports: state.exports.map(e => e.id === id ? updatedExport : e),
          selectedExport: state.selectedExport?.id === id ? updatedExport : state.selectedExport,
          isUpdating: false,
        }));
      } else {
        set({ isUpdating: false });
      }

      return updatedExport;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update export';
      logger.error('Failed to update export', { component: 'ExportStore', action: 'updateExport', id }, error);
      set({ error: message, isUpdating: false });
      return null;
    }
  },

  deleteExport: async (id: string) => {
    set({ isDeleting: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message ?? `Failed to delete export: ${response.status}`);
      }

      set(state => ({
        exports: state.exports.filter(e => e.id !== id),
        selectedExport: state.selectedExport?.id === id ? null : state.selectedExport,
        isDeleting: false,
      }));

      return true;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to delete export';
      logger.error('Failed to delete export', { component: 'ExportStore', action: 'deleteExport', id }, error);
      set({ error: message, isDeleting: false });
      return false;
    }
  },

  setSelectedExport: (export_: Export | null) => {
    set({ selectedExport: export_ });
  },

  clearError: () => {
    set({ error: null });
  },
}));
