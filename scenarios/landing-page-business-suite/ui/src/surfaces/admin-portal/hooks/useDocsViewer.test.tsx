import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useDocsViewer } from './useDocsViewer';
import type { DocEntry, DocContent } from '../../../shared/api';
import type {
  fetchDocsTree,
  fetchDocContent,
  findFirstMarkdownFile,
  getExpandedPathsForDocPath,
} from '../services/docs.service';

// Mock the service module
type FetchDocsTreeFn = typeof fetchDocsTree;
type FetchDocContentFn = typeof fetchDocContent;
type FindFirstMarkdownFileFn = typeof findFirstMarkdownFile;
type GetExpandedPathsForDocPathFn = typeof getExpandedPathsForDocPath;

const fetchDocsTreeMock = vi.fn<Parameters<FetchDocsTreeFn>, ReturnType<FetchDocsTreeFn>>();
const fetchDocContentMock = vi.fn<Parameters<FetchDocContentFn>, ReturnType<FetchDocContentFn>>();
const findFirstMarkdownFileMock = vi.fn<Parameters<FindFirstMarkdownFileFn>, ReturnType<FindFirstMarkdownFileFn>>();
const getExpandedPathsForDocPathMock = vi.fn<
  Parameters<GetExpandedPathsForDocPathFn>,
  ReturnType<GetExpandedPathsForDocPathFn>
>();

vi.mock('../services/docs.service', () => ({
  fetchDocsTree: (...args: Parameters<FetchDocsTreeFn>) => fetchDocsTreeMock(...args),
  fetchDocContent: (...args: Parameters<FetchDocContentFn>) => fetchDocContentMock(...args),
  findFirstMarkdownFile: (...args: Parameters<FindFirstMarkdownFileFn>) => findFirstMarkdownFileMock(...args),
  getExpandedPathsForDocPath: (...args: Parameters<GetExpandedPathsForDocPathFn>) =>
    getExpandedPathsForDocPathMock(...args),
}));

const mockTree: DocEntry[] = [
  {
    name: 'getting-started',
    path: 'getting-started',
    isDir: true,
    children: [
      { name: 'installation.md', path: 'getting-started/installation.md', isDir: false },
    ],
  },
  { name: 'README.md', path: 'README.md', isDir: false },
];

const mockDoc: DocContent = {
  path: 'getting-started/installation.md',
  title: 'Installation',
  content: '# Installation\n\nThis is the installation guide.',
};

describe('useDocsViewer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchDocsTreeMock.mockResolvedValue(mockTree);
    fetchDocContentMock.mockResolvedValue(mockDoc);
    findFirstMarkdownFileMock.mockReturnValue('getting-started/installation.md');
    getExpandedPathsForDocPathMock.mockReturnValue(new Set(['getting-started']));
  });

  describe('initial state', () => {
    it('starts with loading true', () => {
      const { result } = renderHook(() => useDocsViewer());

      expect(result.current.loading).toBe(true);
    });

    it('starts with empty tree', () => {
      const { result } = renderHook(() => useDocsViewer());

      expect(result.current.tree).toEqual([]);
    });

    it('starts with no selected path or doc', () => {
      const { result } = renderHook(() => useDocsViewer());

      expect(result.current.selectedPath).toBeNull();
      expect(result.current.selectedDoc).toBeNull();
    });

    it('starts with empty expanded paths', () => {
      const { result } = renderHook(() => useDocsViewer());

      expect(result.current.expandedPaths.size).toBe(0);
    });

    it('starts with no error', () => {
      const { result } = renderHook(() => useDocsViewer());

      expect(result.current.error).toBeNull();
    });
  });

  describe('tree loading', () => {
    it('loads tree on mount', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(fetchDocsTreeMock).toHaveBeenCalledTimes(1);
      expect(result.current.tree).toEqual(mockTree);
    });

    it('expands root directories after loading', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(getExpandedPathsForDocPathMock).toHaveBeenCalledWith(mockTree, null);
      expect(result.current.expandedPaths.has('getting-started')).toBe(true);
    });

    it('auto-selects first markdown file', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(findFirstMarkdownFileMock).toHaveBeenCalledWith(mockTree);
      expect(fetchDocContentMock).toHaveBeenCalledWith('getting-started/installation.md');
    });

    it('does not load doc when no first file found', async () => {
      findFirstMarkdownFileMock.mockReturnValue(null);

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(fetchDocContentMock).not.toHaveBeenCalled();
      expect(result.current.selectedDoc).toBeNull();
    });

    it('handles tree loading error', async () => {
      fetchDocsTreeMock.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Network error');
      expect(result.current.tree).toEqual([]);
    });

    it('handles non-Error rejection', async () => {
      fetchDocsTreeMock.mockRejectedValue('String error');

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Failed to load documentation');
    });
  });

  describe('document loading', () => {
    it('loads document content on selection', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Wait for auto-selected doc to load
      await waitFor(() => {
        expect(result.current.selectedDoc).not.toBeNull();
      });

      expect(result.current.selectedPath).toBe('getting-started/installation.md');
      expect(result.current.selectedDoc).toEqual(mockDoc);
    });

    it('sets loadingDoc during document fetch', async () => {
      let resolveDoc: (doc: DocContent) => void;
      fetchDocContentMock.mockReturnValue(
        new Promise<DocContent>((resolve) => {
          resolveDoc = resolve;
        })
      );

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // loadingDoc should be true while waiting
      expect(result.current.loadingDoc).toBe(true);

      // Resolve the document
      await act(async () => {
        resolveDoc!(mockDoc);
      });

      await waitFor(() => {
        expect(result.current.loadingDoc).toBe(false);
      });
    });

    it('handles document loading error', async () => {
      fetchDocContentMock.mockRejectedValue(new Error('Document not found'));

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await waitFor(() => {
        expect(result.current.loadingDoc).toBe(false);
      });

      expect(result.current.error).toBe('Document not found');
      expect(result.current.selectedDoc).toBeNull();
    });

    it('handles non-Error rejection in document loading', async () => {
      fetchDocContentMock.mockRejectedValue('String error');

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(fetchDocContentMock).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to load document');
      });
    });

    it('can load a different document', async () => {
      const anotherDoc: DocContent = {
        path: 'README.md',
        title: 'README',
        content: '# README',
      };
      fetchDocContentMock.mockResolvedValueOnce(mockDoc).mockResolvedValueOnce(anotherDoc);

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.selectedDoc).toEqual(mockDoc);
      });

      await act(async () => {
        await result.current.loadDoc('README.md');
      });

      expect(result.current.selectedPath).toBe('README.md');
      expect(result.current.selectedDoc).toEqual(anotherDoc);
    });
  });

  describe('requested doc', () => {
    it('loads requested doc instead of auto-selecting the first file', async () => {
      const { result } = renderHook(() => useDocsViewer({ requestedPath: 'README.md' }));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(findFirstMarkdownFileMock).not.toHaveBeenCalled();
      expect(fetchDocContentMock).toHaveBeenCalledWith('README.md');
    });

    it('expands to the requested doc path', async () => {
      const { result } = renderHook(() => useDocsViewer({ requestedPath: 'getting-started/installation.md' }));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(getExpandedPathsForDocPathMock).toHaveBeenCalledWith(mockTree, 'getting-started/installation.md');
      expect(result.current.expandedPaths.has('getting-started')).toBe(true);
    });
  });

  describe('folder toggle', () => {
    it('adds path to expanded paths when not present', async () => {
      getExpandedPathsForDocPathMock.mockReturnValue(new Set());

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleToggle('getting-started');
      });

      expect(result.current.expandedPaths.has('getting-started')).toBe(true);
    });

    it('removes path from expanded paths when present', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Initially expanded via getExpandedPathsForDocPath
      expect(result.current.expandedPaths.has('getting-started')).toBe(true);

      act(() => {
        result.current.handleToggle('getting-started');
      });

      expect(result.current.expandedPaths.has('getting-started')).toBe(false);
    });

    it('can toggle multiple folders', async () => {
      getExpandedPathsForDocPathMock.mockReturnValue(new Set());

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleToggle('folder1');
        result.current.handleToggle('folder2');
      });

      expect(result.current.expandedPaths.has('folder1')).toBe(true);
      expect(result.current.expandedPaths.has('folder2')).toBe(true);
    });
  });

  describe('refresh tree', () => {
    it('can reload the tree', async () => {
      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      fetchDocsTreeMock.mockClear();
      fetchDocContentMock.mockClear();

      await act(async () => {
        await result.current.loadTree();
      });

      expect(fetchDocsTreeMock).toHaveBeenCalledTimes(1);
    });

    it('clears previous error on refresh', async () => {
      fetchDocsTreeMock.mockRejectedValueOnce(new Error('First error'));

      const { result } = renderHook(() => useDocsViewer());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('First error');

      fetchDocsTreeMock.mockResolvedValueOnce(mockTree);

      await act(async () => {
        await result.current.loadTree();
      });

      expect(result.current.error).toBeNull();
      expect(result.current.tree).toEqual(mockTree);
    });
  });
});
