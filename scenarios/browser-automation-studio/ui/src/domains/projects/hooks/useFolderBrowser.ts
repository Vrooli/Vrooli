/**
 * useFolderBrowser Hook
 *
 * Handles folder scanning and navigation for project import.
 * Scans directories for BAS projects and regular folders,
 * detecting which are already registered in the database.
 */

import { useState, useCallback, useRef } from 'react';
import { getApiBase } from '../../../config';
import { logger } from '../../../utils/logger';

export interface FolderEntry {
  name: string;
  path: string;
  is_project: boolean;
  is_registered: boolean;
  project_id?: string;
  suggested_name?: string;
}

export interface ScanResult {
  path: string;
  parent: string | null;
  default_projects_root: string;
  entries: FolderEntry[];
}

interface UseFolderBrowserReturn {
  isScanning: boolean;
  scanResult: ScanResult | null;
  error: string | null;
  defaultPath: string | null;
  scanFolder: (path?: string, depth?: number) => Promise<ScanResult | null>;
  navigateUp: () => Promise<void>;
  navigateTo: (path: string) => Promise<void>;
  clearError: () => void;
  reset: () => void;
}

export function useFolderBrowser(): UseFolderBrowserReturn {
  const [isScanning, setIsScanning] = useState(false);
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [defaultPath, setDefaultPath] = useState<string | null>(null);

  // Cache to avoid redundant scans when navigating back
  const cacheRef = useRef<Map<string, ScanResult>>(new Map());

  const scanFolder = useCallback(async (path?: string, depth: number = 1): Promise<ScanResult | null> => {
    // Check cache first
    const cacheKey = `${path || 'default'}:${depth}`;
    const cached = cacheRef.current.get(cacheKey);
    if (cached) {
      setScanResult(cached);
      if (!defaultPath && cached.default_projects_root) {
        setDefaultPath(cached.default_projects_root);
      }
      return cached;
    }

    setIsScanning(true);
    setError(null);

    try {
      const apiBase = getApiBase();
      const response = await fetch(`${apiBase}/fs/scan-for-projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, depth }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        const errorMsg = errorData.details?.error || errorData.message || 'Failed to scan folder';
        setError(errorMsg);
        return null;
      }

      const data: ScanResult = await response.json();

      // Normalize parent to null if empty string
      if (data.parent === '') {
        data.parent = null;
      }

      // Cache the result
      cacheRef.current.set(cacheKey, data);

      // Store default path on first successful scan
      if (!defaultPath && data.default_projects_root) {
        setDefaultPath(data.default_projects_root);
      }

      setScanResult(data);
      return data;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to scan folder';
      logger.error('Failed to scan folder', { error: err, path });
      setError(errorMsg);
      return null;
    } finally {
      setIsScanning(false);
    }
  }, [defaultPath]);

  const navigateUp = useCallback(async () => {
    if (!scanResult?.parent) {
      return;
    }
    await scanFolder(scanResult.parent);
  }, [scanResult, scanFolder]);

  const navigateTo = useCallback(async (path: string) => {
    await scanFolder(path);
  }, [scanFolder]);

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
