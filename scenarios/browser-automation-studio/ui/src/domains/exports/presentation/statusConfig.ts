/**
 * Export Status Configuration
 *
 * UI configuration for export and execution status display.
 * Consolidates status-related styling and icons.
 * Uses canonical types from executions/export/config/types.ts.
 */

import {
  Clock,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from 'lucide-react';
import type {
  ExportStatus,
  ExportStatusConfig,
} from '@/domains/executions/export/config/types';
import { formatCapturedLabel } from './formatters';

/**
 * Export status display configuration.
 */
export const EXPORT_STATUS_CONFIG: Record<ExportStatus, ExportStatusConfig> = {
  pending: {
    icon: Clock,
    color: 'text-gray-400',
    bgColor: 'bg-gray-500/10',
    label: 'Pending',
  },
  processing: {
    icon: Loader2,
    color: 'text-yellow-400',
    bgColor: 'bg-yellow-500/10',
    label: 'Processing',
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-green-400',
    bgColor: 'bg-green-500/10',
    label: 'Ready',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-400',
    bgColor: 'bg-red-500/10',
    label: 'Failed',
  },
};

/**
 * Execution status display configuration.
 */
export const EXECUTION_STATUS_CONFIG: Record<string, ExportStatusConfig> = {
  pending: {
    icon: Clock,
    color: 'text-gray-400',
    bgColor: 'bg-gray-500/10',
    label: 'Pending',
  },
  running: {
    icon: Loader2,
    color: 'text-green-400',
    bgColor: 'bg-green-500/10',
    label: 'Running',
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    label: 'Completed',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-400',
    bgColor: 'bg-red-500/10',
    label: 'Failed',
  },
  cancelled: {
    icon: AlertCircle,
    color: 'text-amber-400',
    bgColor: 'bg-amber-500/10',
    label: 'Cancelled',
  },
};

/**
 * Gets status config with a fallback to pending if status is unknown.
 */
export const getExportStatusConfig = (status: string): ExportStatusConfig => {
  return EXPORT_STATUS_CONFIG[status as ExportStatus] ?? EXPORT_STATUS_CONFIG.pending;
};

/**
 * Gets execution status config with a fallback to pending if status is unknown.
 */
export const getExecutionStatusConfig = (status: string): ExportStatusConfig => {
  return EXECUTION_STATUS_CONFIG[status] ?? EXECUTION_STATUS_CONFIG.pending;
};

/**
 * Export preview status labels (for API responses).
 */
export type ExportPreviewStatusLabel = 'ready' | 'pending' | 'error' | 'unavailable' | 'unknown';

/**
 * Maps proto export status enum to display label.
 */
export const mapExportPreviewStatus = (status?: number | null): ExportPreviewStatusLabel => {
  switch (status) {
    case 1: // READY
      return 'ready';
    case 2: // PENDING
      return 'pending';
    case 3: // ERROR
      return 'error';
    case 4: // UNAVAILABLE
      return 'unavailable';
    default:
      return 'unknown';
  }
};

/**
 * Normalizes a preview status string to lowercase.
 */
export const normalizePreviewStatus = (value?: string | null): string => {
  if (!value) {
    return '';
  }
  return value.trim().toLowerCase();
};

/**
 * Generates a human-readable message describing the export preview status.
 */
export const describePreviewStatusMessage = (
  status: string,
  fallback?: string | null,
  metrics?: { capturedFrames?: number; assetCount?: number },
): string => {
  if (fallback && fallback.trim().length > 0) {
    return fallback.trim();
  }
  const capturedFrames = metrics?.capturedFrames ?? 0;
  switch (status) {
    case 'pending':
      if (capturedFrames > 0) {
        return `Replay export pending – ${formatCapturedLabel(capturedFrames, 'frame')} captured so far`;
      }
      return 'Replay export pending – timeline frames not captured yet';
    case 'unavailable':
      if (capturedFrames > 0) {
        return `Replay export unavailable – ${formatCapturedLabel(capturedFrames, 'frame')} captured but not yet playable`;
      }
      return 'Replay export unavailable – execution did not capture any timeline frames';
    default:
      return 'Replay export unavailable';
  }
};
