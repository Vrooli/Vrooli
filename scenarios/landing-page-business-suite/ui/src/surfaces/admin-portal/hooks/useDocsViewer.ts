import { useCallback, useEffect, useState } from 'react';
import type { DocEntry, DocContent } from '../../../shared/api';
import {
  fetchDocsTree,
  fetchDocContent,
  findFirstMarkdownFile,
  getInitialExpandedPaths,
} from '../services/docs.service';

/**
 * Return type for the useDocsViewer hook
 */
export interface UseDocsViewerReturn {
  // State
  tree: DocEntry[];
  selectedPath: string | null;
  selectedDoc: DocContent | null;
  expandedPaths: Set<string>;
  loading: boolean;
  loadingDoc: boolean;
  error: string | null;

  // Actions
  loadTree: () => Promise<void>;
  loadDoc: (path: string) => Promise<void>;
  handleToggle: (path: string) => void;
}

/**
 * Hook for managing documentation viewer state and actions
 *
 * Handles:
 * - Loading the documentation tree structure
 * - Auto-expanding root directories and selecting first file on load
 * - Loading individual document content
 * - Managing folder expand/collapse state
 */
export function useDocsViewer(): UseDocsViewerReturn {
  // Tree and document state
  const [tree, setTree] = useState<DocEntry[]>([]);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [selectedDoc, setSelectedDoc] = useState<DocContent | null>(null);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());

  // Loading states
  const [loading, setLoading] = useState(true);
  const [loadingDoc, setLoadingDoc] = useState(false);

  // Error state
  const [error, setError] = useState<string | null>(null);

  /**
   * Load a specific document by path
   */
  const loadDoc = useCallback(async (path: string) => {
    setSelectedPath(path);
    setLoadingDoc(true);
    try {
      const doc = await fetchDocContent(path);
      setSelectedDoc(doc);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load document');
      setSelectedDoc(null);
    } finally {
      setLoadingDoc(false);
    }
  }, []);

  /**
   * Load the documentation tree and auto-select first file
   */
  const loadTree = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchDocsTree();
      setTree(data);

      // Auto-expand root level directories
      setExpandedPaths(getInitialExpandedPaths(data));

      // Auto-select first markdown file if available
      const firstFile = findFirstMarkdownFile(data);
      if (firstFile) {
        loadDoc(firstFile);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load documentation');
    } finally {
      setLoading(false);
    }
  }, [loadDoc]);

  /**
   * Toggle folder expansion state
   */
  const handleToggle = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  // Load tree on mount
  useEffect(() => {
    loadTree();
  }, [loadTree]);

  return {
    // State
    tree,
    selectedPath,
    selectedDoc,
    expandedPaths,
    loading,
    loadingDoc,
    error,

    // Actions
    loadTree,
    loadDoc,
    handleToggle,
  };
}
