/**
 * Exports Domain Store
 *
 * Manages export state including:
 * - Exports list and CRUD operations
 * - Selected export for viewing
 * - Export status tracking (pending, processing, completed, failed)
 *
 * All API I/O flows through the generated Connect-Web client; never through
 * raw fetch (post-Phase-9 of the proto+Connect migration).
 */

import { create } from 'zustand';
import { ConnectError } from '@connectrpc/connect';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { JsonObject } from '@bufbuild/protobuf';
import type { Export as ProtoExport } from '@vrooli/proto-types/browser-automation-studio/v1/exports/exports_pb';

import { exportsClient } from '@/api/exports';
import { logger } from '../../utils/logger';

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
  /** Replace an export: create new export, then delete the old one */
  replaceExport: (oldId: string, input: CreateExportInput) => Promise<Export | null>;
  setSelectedExport: (export_: Export | null) => void;
  clearError: () => void;

  // Sync helpers for checking existing exports
  /** Get exports for an execution from the current store state (sync) */
  getExportsByExecutionId: (executionId: string) => Export[];
  /** Check if an execution has any exports in the current store state (sync) */
  hasExportsForExecution: (executionId: string) => boolean;
}

const validFormats = new Set<Export['format']>(['mp4', 'gif', 'json', 'html']);
const validStatuses = new Set<Export['status']>(['pending', 'processing', 'completed', 'failed']);

const normalizeFormat = (raw: string): Export['format'] => {
  const candidate = raw.toLowerCase();
  return validFormats.has(candidate as Export['format']) ? (candidate as Export['format']) : 'mp4';
};

const normalizeStatus = (raw: string): Export['status'] => {
  const candidate = raw.toLowerCase();
  return validStatuses.has(candidate as Export['status']) ? (candidate as Export['status']) : 'completed';
};

const protoToExport = (proto: ProtoExport): Export => ({
  id: proto.id,
  executionId: proto.executionId,
  workflowId: proto.workflowId || undefined,
  name: proto.name || 'Untitled Export',
  format: normalizeFormat(proto.format ?? ''),
  settings: (proto.settings as Record<string, unknown> | undefined) ?? undefined,
  storageUrl: proto.storageUrl || undefined,
  thumbnailUrl: proto.thumbnailUrl || undefined,
  fileSizeBytes: typeof proto.fileSizeBytes === 'bigint' ? Number(proto.fileSizeBytes) : undefined,
  durationMs: typeof proto.durationMs === 'number' ? proto.durationMs : undefined,
  frameCount: typeof proto.frameCount === 'number' ? proto.frameCount : undefined,
  aiCaption: proto.aiCaption || undefined,
  aiCaptionGeneratedAt: proto.aiCaptionGeneratedAt ? timestampDate(proto.aiCaptionGeneratedAt) : undefined,
  status: normalizeStatus(proto.status ?? ''),
  error: proto.error || undefined,
  createdAt: proto.createdAt ? timestampDate(proto.createdAt) : new Date(),
  updatedAt: proto.updatedAt ? timestampDate(proto.updatedAt) : new Date(),
  workflowName: proto.workflowName || undefined,
  executionDate: proto.executionDate ? timestampDate(proto.executionDate).toISOString() : undefined,
});

const errorMessage = (err: unknown, fallback: string): string => {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return fallback;
};

const settingsToStructInput = (settings: Record<string, unknown> | undefined): JsonObject | undefined =>
  (settings as JsonObject | undefined);

export const useExportStore = create<ExportState>((set, get) => ({
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
      const res = await exportsClient.listExports({ limit: 100 });
      const exports = res.exports.map(protoToExport);
      exports.sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
      set({ exports, isLoading: false });
    } catch (err: unknown) {
      const message = errorMessage(err, 'Failed to fetch exports');
      logger.error('Failed to fetch exports', { component: 'ExportStore', action: 'fetchExports' }, err);
      set({ error: message, isLoading: false });
    }
  },

  fetchExportsByExecution: async (executionId: string) => {
    try {
      const res = await exportsClient.listExports({ executionId });
      return res.exports.map(protoToExport);
    } catch (err: unknown) {
      logger.error(
        'Failed to fetch exports by execution',
        { component: 'ExportStore', action: 'fetchExportsByExecution', executionId },
        err,
      );
      return [];
    }
  },

  fetchExportsByWorkflow: async (workflowId: string) => {
    try {
      const res = await exportsClient.listExports({ workflowId });
      return res.exports.map(protoToExport);
    } catch (err: unknown) {
      logger.error(
        'Failed to fetch exports by workflow',
        { component: 'ExportStore', action: 'fetchExportsByWorkflow', workflowId },
        err,
      );
      return [];
    }
  },

  getExport: async (id: string) => {
    try {
      const res = await exportsClient.getExport({ id });
      return res.export ? protoToExport(res.export) : null;
    } catch (err: unknown) {
      if (err instanceof ConnectError && err.code === 5 /* NotFound */) {
        return null;
      }
      logger.error('Failed to get export', { component: 'ExportStore', action: 'getExport', id }, err);
      return null;
    }
  },

  createExport: async (input: CreateExportInput) => {
    set({ isCreating: true, error: null });
    try {
      const res = await exportsClient.createExport({
        executionId: input.executionId,
        workflowId: input.workflowId ?? '',
        name: input.name,
        format: input.format,
        settings: settingsToStructInput(input.settings),
        storageUrl: input.storageUrl ?? '',
        thumbnailUrl: input.thumbnailUrl ?? '',
        fileSizeBytes: typeof input.fileSizeBytes === 'number' ? BigInt(input.fileSizeBytes) : undefined,
        durationMs: input.durationMs,
        frameCount: input.frameCount,
        status: input.status ?? 'completed',
      });
      const newExport = res.export ? protoToExport(res.export) : null;
      if (newExport) {
        set((state) => ({
          exports: [newExport, ...state.exports],
          isCreating: false,
        }));
      } else {
        set({ isCreating: false });
      }
      return newExport;
    } catch (err: unknown) {
      const message = errorMessage(err, 'Failed to create export');
      logger.error('Failed to create export', { component: 'ExportStore', action: 'createExport' }, err);
      set({ error: message, isCreating: false });
      return null;
    }
  },

  updateExport: async (id: string, input: UpdateExportInput) => {
    set({ isUpdating: true, error: null });
    try {
      const res = await exportsClient.updateExport({
        id,
        name: input.name ?? '',
        settings: settingsToStructInput(input.settings),
        storageUrl: input.storageUrl ?? '',
        thumbnailUrl: input.thumbnailUrl ?? '',
        fileSizeBytes: typeof input.fileSizeBytes === 'number' ? BigInt(input.fileSizeBytes) : undefined,
        durationMs: input.durationMs,
        frameCount: input.frameCount,
        aiCaption: input.aiCaption ?? '',
        status: input.status ?? '',
        error: input.error ?? '',
      });
      const updatedExport = res.export ? protoToExport(res.export) : null;
      if (updatedExport) {
        set((state) => ({
          exports: state.exports.map((e) => (e.id === id ? updatedExport : e)),
          selectedExport: state.selectedExport?.id === id ? updatedExport : state.selectedExport,
          isUpdating: false,
        }));
      } else {
        set({ isUpdating: false });
      }
      return updatedExport;
    } catch (err: unknown) {
      const message = errorMessage(err, 'Failed to update export');
      logger.error('Failed to update export', { component: 'ExportStore', action: 'updateExport', id }, err);
      set({ error: message, isUpdating: false });
      return null;
    }
  },

  deleteExport: async (id: string) => {
    set({ isDeleting: true, error: null });
    try {
      await exportsClient.deleteExport({ id });
      set((state) => ({
        exports: state.exports.filter((e) => e.id !== id),
        selectedExport: state.selectedExport?.id === id ? null : state.selectedExport,
        isDeleting: false,
      }));
      return true;
    } catch (err: unknown) {
      const message = errorMessage(err, 'Failed to delete export');
      logger.error('Failed to delete export', { component: 'ExportStore', action: 'deleteExport', id }, err);
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

  replaceExport: async (oldId: string, input: CreateExportInput) => {
    const newExport = await get().createExport(input);
    if (!newExport) {
      return null;
    }
    try {
      await exportsClient.deleteExport({ id: oldId });
      set((state) => ({
        exports: state.exports.filter((e) => e.id !== oldId),
        selectedExport: state.selectedExport?.id === oldId ? null : state.selectedExport,
      }));
    } catch (err: unknown) {
      logger.error(
        'Failed to delete old export during replace',
        { component: 'ExportStore', action: 'replaceExport', oldId },
        err,
      );
      // New export was created — don't fail the whole operation.
    }
    return newExport;
  },

  getExportsByExecutionId: (executionId: string): Export[] => {
    return get().exports.filter((e) => e.executionId === executionId);
  },

  hasExportsForExecution: (executionId: string): boolean => {
    return get().exports.some((e) => e.executionId === executionId);
  },

  deleteAllExports: async () => {
    const { exports } = get();
    const errors: string[] = [];
    let deleted = 0;

    set({ isDeleting: true, error: null });

    for (const exportItem of exports) {
      try {
        await exportsClient.deleteExport({ id: exportItem.id });
        deleted++;
      } catch (err: unknown) {
        const message = errorMessage(err, 'Unknown error');
        errors.push(`Failed to delete "${exportItem.name}": ${message}`);
        logger.error(
          'Failed to delete export during bulk delete',
          { component: 'ExportStore', action: 'deleteAllExports', id: exportItem.id },
          err,
        );
      }
    }

    set({
      exports: [],
      selectedExport: null,
      isDeleting: false,
    });

    return { deleted, errors };
  },
}));

// Export types for consumers
export type { ExportState };
