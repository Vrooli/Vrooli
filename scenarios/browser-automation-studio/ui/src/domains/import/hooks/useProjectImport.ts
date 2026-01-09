/**
 * useProjectImport Hook
 *
 * Handles project import functionality including folder inspection
 * and actual import operations.
 */

import { useState, useCallback } from 'react';
import { getApiBase } from '../../../config';
import { logger } from '../../../utils/logger';
import { parseProject } from '../../../utils/projectProto';
import type { Project } from '../../projects/store';

export interface InspectFolderResponse {
  folder_path: string;
  exists: boolean;
  is_dir: boolean;
  has_bas_metadata: boolean;
  metadata_error?: string;
  has_workflows: boolean;
  already_indexed: boolean;
  indexed_project_id?: string;
  suggested_name?: string;
  suggested_description?: string;
}

export interface ImportProjectRequest {
  folder_path: string;
  name?: string;
  description?: string;
}

export interface UseProjectImportOptions {
  /** Optional initial folder path */
  initialPath?: string;
}

export interface UseProjectImportReturn {
  isInspecting: boolean;
  isImporting: boolean;
  inspectResult: InspectFolderResponse | null;
  error: string | null;
  inspectFolder: (folderPath: string) => Promise<InspectFolderResponse | null>;
  importProject: (params: ImportProjectRequest) => Promise<Project | null>;
  clearInspectResult: () => void;
  clearError: () => void;
  reset: () => void;
}

export function useProjectImport(_options?: UseProjectImportOptions): UseProjectImportReturn {
  const [isInspecting, setIsInspecting] = useState(false);
  const [isImporting, setIsImporting] = useState(false);
  const [inspectResult, setInspectResult] = useState<InspectFolderResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const inspectFolder = useCallback(async (folderPath: string): Promise<InspectFolderResponse | null> => {
    setIsInspecting(true);
    setError(null);

    try {
      const apiBase = getApiBase();
      const response = await fetch(`${apiBase}/projects/inspect-folder`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ folder_path: folderPath }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const errorMsg = errorData.details?.error || errorData.message || 'Failed to inspect folder';
        setError(errorMsg);
        return null;
      }

      const data = await response.json();
      setInspectResult(data);
      return data;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to inspect folder';
      logger.error('Failed to inspect folder', { error: err, folderPath });
      setError(errorMsg);
      return null;
    } finally {
      setIsInspecting(false);
    }
  }, []);

  const importProject = useCallback(async (params: ImportProjectRequest): Promise<Project | null> => {
    setIsImporting(true);
    setError(null);

    try {
      const apiBase = getApiBase();
      const response = await fetch(`${apiBase}/projects/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const errorMsg = errorData.details?.error || errorData.message || 'Failed to import project';
        setError(errorMsg);
        return null;
      }

      const data = await response.json();
      const project = parseProject(data);
      return project;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to import project';
      logger.error('Failed to import project', { error: err, params });
      setError(errorMsg);
      return null;
    } finally {
      setIsImporting(false);
    }
  }, []);

  const clearInspectResult = useCallback(() => {
    setInspectResult(null);
  }, []);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const reset = useCallback(() => {
    setInspectResult(null);
    setError(null);
    setIsInspecting(false);
    setIsImporting(false);
  }, []);

  return {
    isInspecting,
    isImporting,
    inspectResult,
    error,
    inspectFolder,
    importProject,
    clearInspectResult,
    clearError,
    reset,
  };
}
