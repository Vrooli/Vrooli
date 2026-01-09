/**
 * useRoutineImport Hook
 *
 * Handles routine/workflow import operations including file inspection
 * and actual import.
 */

import { useState, useCallback } from 'react';
import { getApiBase } from '../../../config';
import { logger } from '../../../utils/logger';

/** Response from inspecting a routine file */
export interface InspectRoutineResponse {
  file_path: string;
  exists: boolean;
  is_valid: boolean;
  validation_error?: string;
  already_indexed: boolean;
  indexed_id?: string;
  preview?: WorkflowPreview;
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
  is_valid: boolean;
  is_registered: boolean;
  workflow_id?: string;
  preview_name?: string;
}

/** Response from scanning for routines */
export interface ScanRoutinesResponse {
  path: string;
  parent: string | null;
  entries: RoutineEntry[];
}

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
          const errorData = await response.json().catch(() => ({}));
          const errorMsg = errorData.message || 'Failed to inspect file';
          setError(errorMsg);
          return null;
        }

        const data = await response.json();
        setInspectResult(data);
        return data;
      } catch (err) {
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
          const errorData = await response.json().catch(() => ({}));
          const errorMsg = errorData.message || 'Failed to import routine';
          setError(errorMsg);
          return null;
        }

        return await response.json();
      } catch (err) {
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
        const response = await fetch(`${apiBase}/projects/${projectId}/routines/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path, depth }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          const errorMsg = errorData.message || 'Failed to scan routines';
          setError(errorMsg);
          return null;
        }

        const data = await response.json();
        setScanResult(data);
        return data;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to scan routines';
        logger.error('Failed to scan routines', { error: err, path });
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
