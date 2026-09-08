import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Root container
  app: "git-control-tower",

  // Header
  header: {
    root: "status-header",
    branchInfo: "branch-info",
    commitOid: "commit-oid",
    fileStats: "file-stats",
    healthStatus: "health-status",
    refreshButton: "refresh-button"
  },

  // File list panel
  fileList: {
    root: "file-list-panel",
    stageAllButton: "stage-all-button",
    unstageAllButton: "unstage-all-button",
    emptyState: "empty-state"
  },

  // Diff viewer panel
  diffViewer: {
    root: "diff-viewer-panel",
    stats: "diff-stats",
    loading: "diff-loading",
    error: "diff-error",
    empty: "diff-empty",
    noChanges: "diff-no-changes",
    content: "diff-content",
    raw: "diff-raw",
    minimap: "diff-minimap",
    minimapRail: "diff-minimap-rail",
    minimapViewport: "diff-minimap-viewport",
    minimapMarker: "diff-minimap-marker",
    minimapTexture: "diff-minimap-texture",
    minimapTextureLine: "diff-minimap-texture-line"
  },

  // Toast/notifications
  errorToast: "error-toast"
} as const;

const dynamicSelectorDefinitions = {
  fileList: {
    section: defineDynamicSelector({
      description: "File section by category (staged, unstaged, untracked, conflicts)",
      testIdPattern: "file-section-${category}",
      params: {
        category: {
          type: "enum",
          values: ["staged", "unstaged", "untracked", "conflicts"]
        }
      }
    }),
    sectionToggle: defineDynamicSelector({
      description: "File section toggle button",
      testIdPattern: "file-section-toggle-${category}",
      params: {
        category: {
          type: "enum",
          values: ["staged", "unstaged", "untracked", "conflicts"]
        }
      }
    }),
    fileItem: defineDynamicSelector({
      description: "File item by category",
      testIdPattern: "file-item-${category}",
      params: {
        category: {
          type: "enum",
          values: ["staged", "unstaged", "untracked", "conflicts"]
        }
      }
    }),
    fileAction: defineDynamicSelector({
      description: "File action button (stage/unstage)",
      testIdPattern: "file-action-${category}",
      params: {
        category: {
          type: "enum",
          values: ["staged", "unstaged", "untracked", "conflicts"]
        }
      }
    }),
    fileByPath: defineDynamicSelector({
      description: "File item by path",
      selectorPattern: '[data-file-path="${path}"]',
      params: {
        path: { type: "string" }
      }
    })
  },
  diffViewer: {
    hunk: defineDynamicSelector({
      description: "Diff hunk by index",
      testIdPattern: "diff-hunk-${index}",
      params: {
        index: { type: "number" }
      }
    }),
    line: defineDynamicSelector({
      description: "Diff line",
      testIdPattern: "diff-line",
      params: {}
    })
  }
} as const;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
