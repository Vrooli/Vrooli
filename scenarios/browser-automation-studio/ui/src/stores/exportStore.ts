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
  deleteAllExports: () => Promise<{ deleted: number; errors: string[] }>;
  setSelectedExport: (export_: Export | null) => void;
  clearError: () => void;
}

// Helper to normalize API response to Export interface. Expects snake_case protojson payloads.
const normalizeExport = (raw: unknown): Export | null => {
  if (!raw || typeof raw !== 'object') return null;
  const obj = raw as Record<string, unknown>;

  const id = typeof obj.id === 'string' ? obj.id : '';
  const executionId = typeof obj.execution_id === 'string' ? obj.execution_id : '';
  if (!id || !executionId) return null;

  const formatValue = typeof obj.format === 'string' ? obj.format.toLowerCase() : 'mp4';
  const validFormats = new Set<Export['format']>(['mp4', 'gif', 'json', 'html']);
  const format: Export['format'] = validFormats.has(formatValue as Export['format'])
    ? (formatValue as Export['format'])
    : 'mp4';

  const statusValue = typeof obj.status === 'string' ? obj.status.toLowerCase() : 'completed';
  const validStatuses = new Set<Export['status']>(['pending', 'processing', 'completed', 'failed']);
  const status: Export['status'] = validStatuses.has(statusValue as Export['status'])
    ? (statusValue as Export['status'])
    : 'completed';

  const asNumber = (value: unknown): number | undefined => (typeof value === 'number' ? value : undefined);

  return {
    id,
    executionId,
    workflowId: typeof obj.workflow_id === 'string' ? obj.workflow_id : undefined,
    name: typeof obj.name === 'string' ? obj.name : 'Untitled Export',
    format,
    settings: (obj.settings as Record<string, unknown> | undefined) ?? undefined,
    storageUrl: typeof obj.storage_url === 'string' ? obj.storage_url : undefined,
    thumbnailUrl: typeof obj.thumbnail_url === 'string' ? obj.thumbnail_url : undefined,
    fileSizeBytes: asNumber(obj.file_size_bytes),
    durationMs: asNumber(obj.duration_ms),
    frameCount: asNumber(obj.frame_count),
    aiCaption: typeof obj.ai_caption === 'string' ? obj.ai_caption : undefined,
    aiCaptionGeneratedAt:
      typeof obj.ai_caption_generated_at === 'string' ? new Date(obj.ai_caption_generated_at) : undefined,
    status,
    error: typeof obj.error === 'string' ? obj.error : undefined,
    createdAt: new Date(
      typeof obj.created_at === 'string' ? obj.created_at : new Date().toISOString(),
    ),
    updatedAt: new Date(
      typeof obj.updated_at === 'string' ? obj.updated_at : new Date().toISOString(),
    ),
    workflowName: typeof obj.workflow_name === 'string' ? obj.workflow_name : undefined,
    executionDate: typeof obj.execution_date === 'string' ? obj.execution_date : undefined,
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
        ? data.exports
            .map((e: unknown) => normalizeExport(e))
            .filter((e: Export | null | undefined): e is Export => Boolean(e))
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
        ? data.exports
            .map((e: unknown) => normalizeExport(e))
            .filter((e: Export | null | undefined): e is Export => Boolean(e))
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
        ? data.exports
            .map((e: unknown) => normalizeExport(e))
            .filter((e: Export | null | undefined): e is Export => Boolean(e))
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
      const normalized = normalizeExport(data.export);
      return normalized;
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
      const newExport = normalizeExport(data.export);

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
      const updatedExport = normalizeExport(data.export);

      if (updatedExport) {
        set((state) => ({
          exports: state.exports.map((e: Export) => (e.id === id ? updatedExport : e)),
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

      set((state) => ({
        exports: state.exports.filter((e: Export) => e.id !== id),
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

  deleteAllExports: async () => {
    const { exports } = useExportStore.getState();
    const errors: string[] = [];
    let deleted = 0;

    set({ isDeleting: true, error: null });

    for (const exportItem of exports) {
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/exports/${exportItem.id}`, {
          method: 'DELETE',
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          errors.push(`Failed to delete "${exportItem.name}": ${errorData.message ?? response.status}`);
        } else {
          deleted++;
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        errors.push(`Failed to delete "${exportItem.name}": ${message}`);
        logger.error('Failed to delete export during bulk delete', { component: 'ExportStore', action: 'deleteAllExports', id: exportItem.id }, error);
      }
    }

    // Clear local state
    set({
      exports: [],
      selectedExport: null,
      isDeleting: false,
    });

    return { deleted, errors };
  },
}));
