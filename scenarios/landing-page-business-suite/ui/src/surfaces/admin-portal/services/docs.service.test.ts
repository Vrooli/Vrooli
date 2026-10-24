import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { DocEntry, DocContent, getDocsTree, getDocContent } from '../../../shared/api';
import {
  fetchDocsTree,
  fetchDocContent,
  findFirstMarkdownFile,
  getInitialExpandedPaths,
} from './docs.service';

// Mock the API module
type GetDocsTreeFn = typeof getDocsTree;
type GetDocContentFn = typeof getDocContent;

const getDocsTreeMock = vi.fn<GetDocsTreeFn>();
const getDocContentMock = vi.fn<GetDocContentFn>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getDocsTree: (...args: Parameters<GetDocsTreeFn>) => getDocsTreeMock(...args),
    getDocContent: (...args: Parameters<GetDocContentFn>) => getDocContentMock(...args),
  };
});

const mockTree: DocEntry[] = [
  {
    name: 'getting-started',
    path: 'getting-started',
    isDir: true,
    children: [
      { name: 'installation.md', path: 'getting-started/installation.md', isDir: false },
      { name: 'configuration.md', path: 'getting-started/configuration.md', isDir: false },
    ],
  },
  {
    name: 'api',
    path: 'api',
    isDir: true,
    children: [
      { name: 'endpoints.md', path: 'api/endpoints.md', isDir: false },
    ],
  },
  { name: 'README.md', path: 'README.md', isDir: false },
];

const mockDoc: DocContent = {
  path: 'getting-started/installation.md',
  title: 'Installation',
  content: '# Installation\n\nThis is the installation guide.',
};

describe('docs.service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('fetchDocsTree', () => {
    it('returns the docs tree on success', async () => {
      getDocsTreeMock.mockResolvedValue(mockTree);

      const result = await fetchDocsTree();

      expect(getDocsTreeMock).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockTree);
    });

    it('propagates errors from the API', async () => {
      getDocsTreeMock.mockRejectedValue(new Error('Network error'));

      await expect(fetchDocsTree()).rejects.toThrow('Network error');
    });
  });

  describe('fetchDocContent', () => {
    it('returns document content on success', async () => {
      getDocContentMock.mockResolvedValue(mockDoc);

      const result = await fetchDocContent('getting-started/installation.md');

      expect(getDocContentMock).toHaveBeenCalledWith('getting-started/installation.md');
      expect(result).toEqual(mockDoc);
    });

    it('propagates errors from the API', async () => {
      getDocContentMock.mockRejectedValue(new Error('Document not found'));

      await expect(fetchDocContent('nonexistent.md')).rejects.toThrow('Document not found');
    });
  });

  describe('findFirstMarkdownFile', () => {
    it('returns the first file in a flat tree', () => {
      const tree: DocEntry[] = [
        { name: 'README.md', path: 'README.md', isDir: false },
        { name: 'CHANGELOG.md', path: 'CHANGELOG.md', isDir: false },
      ];

      const result = findFirstMarkdownFile(tree);

      expect(result).toBe('README.md');
    });

    it('finds nested file inside directories', () => {
      const result = findFirstMarkdownFile(mockTree);

      expect(result).toBe('getting-started/installation.md');
    });

    it('returns null for empty tree', () => {
      const result = findFirstMarkdownFile([]);

      expect(result).toBeNull();
    });

    it('returns null when tree has only empty directories', () => {
      const tree: DocEntry[] = [
        { name: 'empty-dir', path: 'empty-dir', isDir: true, children: [] },
        { name: 'another-empty', path: 'another-empty', isDir: true, children: [] },
      ];

      const result = findFirstMarkdownFile(tree);

      expect(result).toBeNull();
    });

    it('finds file in deeply nested structure', () => {
      const tree: DocEntry[] = [
        {
          name: 'level1',
          path: 'level1',
          isDir: true,
          children: [
            {
              name: 'level2',
              path: 'level1/level2',
              isDir: true,
              children: [
                { name: 'deep.md', path: 'level1/level2/deep.md', isDir: false },
              ],
            },
          ],
        },
      ];

      const result = findFirstMarkdownFile(tree);

      expect(result).toBe('level1/level2/deep.md');
    });

    it('skips directories without children property', () => {
      const tree: DocEntry[] = [
        { name: 'dir-no-children', path: 'dir-no-children', isDir: true },
        { name: 'file.md', path: 'file.md', isDir: false },
      ];

      const result = findFirstMarkdownFile(tree);

      expect(result).toBe('file.md');
    });
  });

  describe('getInitialExpandedPaths', () => {
    it('returns paths of root-level directories', () => {
      const result = getInitialExpandedPaths(mockTree);

      expect(result.has('getting-started')).toBe(true);
      expect(result.has('api')).toBe(true);
      expect(result.size).toBe(2);
    });

    it('excludes files from expanded paths', () => {
      const result = getInitialExpandedPaths(mockTree);

      expect(result.has('README.md')).toBe(false);
    });

    it('returns empty set for empty tree', () => {
      const result = getInitialExpandedPaths([]);

      expect(result.size).toBe(0);
    });

    it('returns empty set for tree with only files', () => {
      const tree: DocEntry[] = [
        { name: 'file1.md', path: 'file1.md', isDir: false },
        { name: 'file2.md', path: 'file2.md', isDir: false },
      ];

      const result = getInitialExpandedPaths(tree);

      expect(result.size).toBe(0);
    });

    it('returns set for tree with only directories', () => {
      const tree: DocEntry[] = [
        { name: 'dir1', path: 'dir1', isDir: true, children: [] },
        { name: 'dir2', path: 'dir2', isDir: true, children: [] },
      ];

      const result = getInitialExpandedPaths(tree);

      expect(result.size).toBe(2);
      expect(result.has('dir1')).toBe(true);
      expect(result.has('dir2')).toBe(true);
    });
  });
});
