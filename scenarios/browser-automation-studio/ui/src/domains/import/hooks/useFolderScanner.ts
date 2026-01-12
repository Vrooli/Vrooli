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

interface ScanResponse {
  path: string;
  parent: string | null;
  default_root?: string;
  entries: Array<{
    name: string;
    path: string;
    is_dir: boolean;
    is_target: boolean;
    is_registered: boolean;
    registered_id?: string;
    suggested_name?: string;
    mime_type?: string;
    size_bytes?: number;
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

function transformScanEntry(entry: ScanResponse['entries'][0]): FolderEntry {
  return {
    name: entry.name,
    path: entry.path,
    isDir: entry.is_dir,
    isTarget: entry.is_target,
    isRegistered: entry.is_registered,
    registeredId: entry.registered_id,
    suggestedName: entry.suggested_name,
    mimeType: entry.mime_type,
    sizeBytes: entry.size_bytes,
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
        const endpoint = `${apiBase}/fs/scan`;
        const body: Record<string, unknown> = {
          mode,
          path,
          depth,
        };
        if (projectId) {
          body.project_id = projectId;
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

        const responseData = data as ScanResponse;
        const result: ScanResult = {
          path: responseData.path,
          parent: responseData.parent || null,
          defaultRoot: responseData.default_root || responseData.path,
          entries: (responseData.entries || []).map(transformScanEntry),
        };

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
