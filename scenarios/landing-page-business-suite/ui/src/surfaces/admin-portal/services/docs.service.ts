import {
  getDocsTree,
  getDocContent,
  type DocEntry,
  type DocContent,
} from '../../../shared/api';

// Re-export types for convenience
export type { DocEntry, DocContent };

/**
 * Fetch the documentation tree from the API
 */
export async function fetchDocsTree(): Promise<DocEntry[]> {
  return getDocsTree();
}

/**
 * Fetch document content by path from the API
 */
export async function fetchDocContent(path: string): Promise<DocContent> {
  return getDocContent(path);
}

/**
 * Recursively find the first markdown file in the tree
 * Returns the path of the first non-directory entry, or null if none found
 */
export function findFirstMarkdownFile(entries: DocEntry[]): string | null {
  for (const entry of entries) {
    if (!entry.isDir) {
      return entry.path;
    }
    if (entry.children) {
      const found = findFirstMarkdownFile(entry.children);
      if (found) {
        return found;
      }
    }
  }
  return null;
}

/**
 * Get the initial set of expanded paths (root-level directories)
 * This auto-expands the first level of the tree for better UX
 */
export function getInitialExpandedPaths(entries: DocEntry[]): Set<string> {
  return new Set(entries.filter((e) => e.isDir).map((e) => e.path));
}

/**
 * Get expanded paths for a specific document path, including parent folders.
 */
export function getExpandedPathsForDocPath(entries: DocEntry[], docPath: string | null): Set<string> {
  const expanded = getInitialExpandedPaths(entries);
  if (!docPath) {
    return expanded;
  }

  const normalized = docPath.replace(/^\/+/, '').replace(/\/+$/, '');
  const segments = normalized.split('/').filter(Boolean);

  if (segments.length <= 1) {
    return expanded;
  }

  let current = '';
  for (const segment of segments.slice(0, -1)) {
    current = current ? `${current}/${segment}` : segment;
    expanded.add(current);
  }

  return expanded;
}
