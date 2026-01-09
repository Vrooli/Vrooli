/**
 * useFolderScanner Hook
 *
 * Generalized folder scanning for import operations.
 * Supports different scan modes: projects, workflows, assets, files.
 * Extracted and enhanced from useFolderBrowser.
 */

import { useState, useCallback, useRef } from 'react';
import { getApiBase } from '../../../config';
import { logger } from '../../../utils/logger';
import type { FolderEntry, ScanResult, ScanMode } from '../types';

/** API response format for project scanning */
interface ProjectScanResponse {
  path: string;
  parent: string | null;
  default_projects_root: string;
  entries: Array<{
    name: string;
    path: string;
    is_project: boolean;
    is_registered: boolean;
    project_id?: string;
    suggested_name?: string;
  }>;
}

/** API response format for routine/workflow scanning */
interface RoutineScanResponse {
  path: string;
  parent: string | null;
  entries: Array<{
    name: string;
    path: string;
    is_valid: boolean;
    is_registered: boolean;
    workflow_id?: string;
    preview_name?: string;
  }>;
}

export interface UseFolderScannerOptions {
  /** Scan mode determines what we're looking for */
  mode: ScanMode;
  /** Project ID for project-scoped scans (workflows, assets) */
  projectId?: string;
  /** Scan depth (1 or 2) */
  depth?: 1 | 2;
  /** Initial path to scan */
  initialPath?: string;
}

export interface UseFolderScannerReturn {
  /** Whether scanning is in progress */
  isScanning: boolean;
  /** Current scan result */
  scanResult: ScanResult | null;
  /** Error message if any */
  error: string | null;
  /** Default starting path for this mode */
  defaultPath: string | null;
  /** Scan a folder */
  scanFolder: (path?: string) => Promise<ScanResult | null>;
  /** Navigate to parent directory */
  navigateUp: () => Promise<void>;
  /** Navigate to a specific path */
  navigateTo: (path: string) => Promise<void>;
  /** Clear error */
  clearError: () => void;
  /** Reset state */
  reset: () => void;
}

/** Transform project scan response to generic FolderEntry */
function transformProjectEntry(entry: ProjectScanResponse['entries'][0]): FolderEntry {
  return {
    name: entry.name,
    path: entry.path,
    isTarget: entry.is_project,
    isRegistered: entry.is_registered,
    registeredId: entry.project_id,
    suggestedName: entry.suggested_name,
  };
}

/** Transform routine scan response to generic FolderEntry */
function transformRoutineEntry(entry: RoutineScanResponse['entries'][0]): FolderEntry {
  return {
    name: entry.name,
    path: entry.path,
    isTarget: entry.is_valid,
    isRegistered: entry.is_registered,
    registeredId: entry.workflow_id,
    suggestedName: entry.preview_name,
  };
}

export function useFolderScanner(options: UseFolderScannerOptions): UseFolderScannerReturn {
  const { mode, projectId, depth = 1, initialPath } = options;

  const [isScanning, setIsScanning] = useState(false);
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [defaultPath, setDefaultPath] = useState<string | null>(initialPath || null);

  // Cache to avoid redundant scans
  const cacheRef = useRef<Map<string, ScanResult>>(new Map());

  const scanFolder = useCallback(
    async (path?: string): Promise<ScanResult | null> => {
      // Build cache key
      const cacheKey = `${mode}:${projectId || ''}:${path || 'default'}:${depth}`;
      const cached = cacheRef.current.get(cacheKey);
      if (cached) {
        setScanResult(cached);
        if (!defaultPath && cached.defaultRoot) {
          setDefaultPath(cached.defaultRoot);
        }
        return cached;
      }

      setIsScanning(true);
      setError(null);

      try {
        const apiBase = getApiBase();
        let endpoint: string;
        let body: Record<string, unknown>;

        // Choose endpoint and transform based on mode
        switch (mode) {
          case 'projects':
            endpoint = `${apiBase}/fs/scan-for-projects`;
            body = { path, depth };
            break;
          case 'workflows':
            if (!projectId) {
              throw new Error('Project ID required for workflow scanning');
            }
            endpoint = `${apiBase}/projects/${projectId}/routines/scan`;
            body = { path, depth };
            break;
          case 'assets':
            if (!projectId) {
              throw new Error('Project ID required for asset scanning');
            }
            // Assets use filesystem listing for now
            endpoint = `${apiBase}/projects/${projectId}/files/list`;
            body = { path };
            break;
          case 'files':
          default:
            endpoint = `${apiBase}/fs/list-directories`;
            body = { path };
            break;
        }

        const response = await fetch(endpoint, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          const errorMsg =
            errorData.details?.error || errorData.message || 'Failed to scan folder';
          setError(errorMsg);
          return null;
        }

        const data = await response.json();

        // Transform response based on mode
        let result: ScanResult;
        switch (mode) {
          case 'projects': {
            const projectData = data as ProjectScanResponse;
            result = {
              path: projectData.path,
              parent: projectData.parent || null,
              defaultRoot: projectData.default_projects_root,
              entries: projectData.entries.map(transformProjectEntry),
            };
            break;
          }
          case 'workflows': {
            const routineData = data as RoutineScanResponse;
            result = {
              path: routineData.path,
              parent: routineData.parent || null,
              defaultRoot: routineData.path, // Use current path as default
              entries: routineData.entries.map(transformRoutineEntry),
            };
            break;
          }
          case 'assets':
          case 'files':
          default: {
            // Generic directory listing
            result = {
              path: data.path,
              parent: data.parent || null,
              defaultRoot: data.path,
              entries: (data.entries || []).map((e: { name: string; path: string }) => ({
                name: e.name,
                path: e.path,
                isTarget: false,
                isRegistered: false,
              })),
            };
            break;
          }
        }

        // Cache result
        cacheRef.current.set(cacheKey, result);

        // Store default path on first successful scan
        if (!defaultPath && result.defaultRoot) {
          setDefaultPath(result.defaultRoot);
        }

        setScanResult(result);
        return result;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to scan folder';
        logger.error('Failed to scan folder', { error: err, path, mode });
        setError(errorMsg);
        return null;
      } finally {
        setIsScanning(false);
      }
    },
    [mode, projectId, depth, defaultPath]
  );

  const navigateUp = useCallback(async () => {
    if (!scanResult?.parent) {
      return;
    }
    await scanFolder(scanResult.parent);
  }, [scanResult, scanFolder]);

  const navigateTo = useCallback(
    async (path: string) => {
      await scanFolder(path);
    },
    [scanFolder]
  );

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const reset = useCallback(() => {
    setScanResult(null);
    setError(null);
    setIsScanning(false);
    // Keep defaultPath and cache for subsequent uses
  }, []);

  return {
    isScanning,
    scanResult,
    error,
    defaultPath,
    scanFolder,
    navigateUp,
    navigateTo,
    clearError,
    reset,
  };
}
