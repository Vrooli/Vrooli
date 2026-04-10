/**
 * useRoutineImport Hook
 *
 * Handles routine/workflow import operations including file inspection
 * and actual import.
 */

import { useState, useCallback } from 'react';
import { getApiBase } from '../../../config';
import { logger } from '../../../utils/logger';
import type { ValidationSummary } from '../types';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean';
const isNumber = (value: unknown): value is number => typeof value === 'number';
const isStringArray = (value: unknown): value is string[] =>
  Array.isArray(value) && value.every(isString);

const parseJson = async (response: Response): Promise<unknown> => {
  try {
    return await response.json();
  } catch {
    return null;
  }
};

const extractErrorMessage = (value: unknown): string | null => {
  if (!isRecord(value)) {
    return null;
  }
  const message = value.message;
  return typeof message === 'string' ? message : null;
};

/** Response from inspecting a routine file */
export interface InspectRoutineResponse {
  file_path: string;
  exists: boolean;
  is_valid: boolean;
  validation_error?: string;
  already_indexed: boolean;
  indexed_id?: string;
  preview?: WorkflowPreview;
  /** Structured validation checks with status, labels, and descriptions */
  validation?: ValidationSummary;
}

/** Workflow preview data */
export interface WorkflowPreview {
  id?: string;
  name: string;
  description?: string;
  node_count: number;
  edge_count: number;
  tags?: string[];
  version: number;
  has_start_node: boolean;
  has_end_node: boolean;
}

/** Request to import a routine */
export interface ImportRoutineRequest {
  file_path: string;
  dest_path?: string;
  name?: string;
  overwrite_if_exists?: boolean;
}

/** Response from importing a routine */
export interface ImportRoutineResponse {
  workflow_id: string;
  name: string;
  path: string;
  warnings?: string[];
}

/** Routine entry from scanning */
export interface RoutineEntry {
  name: string;
  path: string;
  is_dir: boolean;
  is_target: boolean;
  is_registered: boolean;
  registered_id?: string;
  suggested_name?: string;
  mime_type?: string;
  size_bytes?: number;
}

/** Response from scanning for routines */
export interface ScanRoutinesResponse {
  path: string;
  parent: string | null;
  default_root?: string;
  entries: RoutineEntry[];
}

const isWorkflowPreview = (value: unknown): value is WorkflowPreview => {
  if (!isRecord(value)) return false;
  if (!isString(value.name)) return false;
  if (!isNumber(value.node_count)) return false;
  if (!isNumber(value.edge_count)) return false;
  if (!isNumber(value.version)) return false;
  if (!isBoolean(value.has_start_node)) return false;
  if (!isBoolean(value.has_end_node)) return false;
  if (value.tags !== undefined && !isStringArray(value.tags)) return false;
  if (value.id !== undefined && !isString(value.id)) return false;
  if (value.description !== undefined && !isString(value.description)) return false;
  return true;
};

const isInspectRoutineResponse = (value: unknown): value is InspectRoutineResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.file_path)) return false;
  if (!isBoolean(value.exists)) return false;
  if (!isBoolean(value.is_valid)) return false;
  if (!isBoolean(value.already_indexed)) return false;
  if (value.preview !== undefined && !isWorkflowPreview(value.preview)) return false;
  return true;
};

const isImportRoutineResponse = (value: unknown): value is ImportRoutineResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.workflow_id)) return false;
  if (!isString(value.name)) return false;
  if (!isString(value.path)) return false;
  if (value.warnings !== undefined && !isStringArray(value.warnings)) return false;
  return true;
};

const isRoutineEntry = (value: unknown): value is RoutineEntry => {
  if (!isRecord(value)) return false;
  if (!isString(value.name)) return false;
  if (!isString(value.path)) return false;
  if (!isBoolean(value.is_dir)) return false;
  if (!isBoolean(value.is_target)) return false;
  if (!isBoolean(value.is_registered)) return false;
  if (value.registered_id !== undefined && !isString(value.registered_id)) return false;
  if (value.suggested_name !== undefined && !isString(value.suggested_name)) return false;
  if (value.mime_type !== undefined && !isString(value.mime_type)) return false;
  if (value.size_bytes !== undefined && !isNumber(value.size_bytes)) return false;
  return true;
};

const isScanRoutinesResponse = (value: unknown): value is ScanRoutinesResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.path)) return false;
  if (value.parent !== null && value.parent !== undefined && !isString(value.parent)) return false;
  if (value.default_root !== undefined && !isString(value.default_root)) return false;
  if (!Array.isArray(value.entries)) return false;
  if (!value.entries.every(isRoutineEntry)) return false;
  return true;
};

export interface UseRoutineImportOptions {
  /** Project ID for import operations */
  projectId: string;
}

export interface UseRoutineImportReturn {
  /** Whether inspecting a file */
  isInspecting: boolean;
  /** Whether importing */
  isImporting: boolean;
  /** Whether scanning */
  isScanning: boolean;
  /** Inspect result */
  inspectResult: InspectRoutineResponse | null;
  /** Scan result */
  scanResult: ScanRoutinesResponse | null;
  /** Error message */
  error: string | null;
  /** Inspect a workflow file */
  inspectFile: (filePath: string) => Promise<InspectRoutineResponse | null>;
  /** Import a workflow file */
  importRoutine: (params: ImportRoutineRequest) => Promise<ImportRoutineResponse | null>;
  /** Scan a directory for workflows */
  scanRoutines: (path?: string, depth?: number) => Promise<ScanRoutinesResponse | null>;
  /** Clear inspect result */
  clearInspectResult: () => void;
  /** Clear error */
  clearError: () => void;
  /** Reset all state */
  reset: () => void;
}

export function useRoutineImport(options: UseRoutineImportOptions): UseRoutineImportReturn {
  const { projectId } = options;

  const [isInspecting, setIsInspecting] = useState(false);
  const [isImporting, setIsImporting] = useState(false);
  const [isScanning, setIsScanning] = useState(false);
  const [inspectResult, setInspectResult] = useState<InspectRoutineResponse | null>(null);
  const [scanResult, setScanResult] = useState<ScanRoutinesResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const inspectFile = useCallback(
    async (filePath: string): Promise<InspectRoutineResponse | null> => {
      setIsInspecting(true);
      setError(null);

      try {
        const apiBase = getApiBase();
        const response = await fetch(`${apiBase}/projects/${projectId}/routines/inspect`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ file_path: filePath }),
        });

        if (!response.ok) {
          const errorData = await parseJson(response);
          const errorMsg = extractErrorMessage(errorData) ?? 'Failed to inspect file';
          setError(errorMsg);
          return null;
        }

        const data: unknown = await response.json();
        if (!isInspectRoutineResponse(data)) {
          setError('Invalid inspect response');
          return null;
        }
        setInspectResult(data);
        return data;
      } catch (err: unknown) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to inspect file';
        logger.error('Failed to inspect routine', { error: err, filePath });
        setError(errorMsg);
        return null;
      } finally {
        setIsInspecting(false);
      }
    },
    [projectId]
  );

  const importRoutine = useCallback(
    async (params: ImportRoutineRequest): Promise<ImportRoutineResponse | null> => {
      setIsImporting(true);
      setError(null);

      try {
        const apiBase = getApiBase();
        const response = await fetch(`${apiBase}/projects/${projectId}/routines/import`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(params),
        });

        if (!response.ok) {
          const errorData = await parseJson(response);
          const errorMsg = extractErrorMessage(errorData) ?? 'Failed to import routine';
          setError(errorMsg);
          return null;
        }

        const data: unknown = await response.json();
        if (!isImportRoutineResponse(data)) {
          setError('Invalid import response');
          return null;
        }
        return data;
      } catch (err: unknown) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to import routine';
        logger.error('Failed to import routine', { error: err, params });
        setError(errorMsg);
        return null;
      } finally {
        setIsImporting(false);
      }
    },
    [projectId]
  );

  const scanRoutines = useCallback(
    async (path?: string, depth?: number): Promise<ScanRoutinesResponse | null> => {
      setIsScanning(true);
      setError(null);

      try {
        const apiBase = getApiBase();
        const response = await fetch(`${apiBase}/fs/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: 'workflows', project_id: projectId, path, depth }),
        });

        if (!response.ok) {
          const errorData = await parseJson(response);
          const errorMsg = extractErrorMessage(errorData) ?? 'Failed to scan workflows';
          setError(errorMsg);
          return null;
        }

        const data: unknown = await response.json();
        if (!isScanRoutinesResponse(data)) {
          setError('Invalid scan response');
          return null;
        }
        setScanResult(data);
        return data;
      } catch (err: unknown) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to scan workflows';
        logger.error('Failed to scan workflows', { error: err, path });
        setError(errorMsg);
        return null;
      } finally {
        setIsScanning(false);
      }
    },
    [projectId]
  );

  const clearInspectResult = useCallback(() => {
    setInspectResult(null);
  }, []);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const reset = useCallback(() => {
    setInspectResult(null);
    setScanResult(null);
    setError(null);
    setIsInspecting(false);
    setIsImporting(false);
    setIsScanning(false);
  }, []);

  return {
    isInspecting,
    isImporting,
    isScanning,
    inspectResult,
    scanResult,
    error,
    inspectFile,
    importRoutine,
    scanRoutines,
    clearInspectResult,
    clearError,
    reset,
  };
}
