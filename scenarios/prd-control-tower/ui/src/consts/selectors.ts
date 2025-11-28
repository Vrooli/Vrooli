/**
 * PRD Control Tower selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Browser Automation Studio workflows. The manifest is auto-generated from
 * this file during testing.
 *
 * ## Auto-Generated Manifest
 *
 * The `selectors.manifest.json` file is automatically generated from this file
 * during the testing process. If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. The manifest will be regenerated automatically when tests run
 *
 * DO NOT manually edit `selectors.manifest.json` - your changes will be overwritten!
 */

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };

const literalSelectors = {
  catalog: {
    card: "catalog-card",
    searchInput: "catalog-search-input",
    newDraftButton: "new-draft-button",
    typeFilter: "catalog-type-filter",
    sortDropdown: "catalog-sort-dropdown",
    statusBadge: "catalog-status-badge",
    progressBar: "catalog-progress-bar",
  },
  drafts: {
    draftCard: "draft-card",
    newDraftButton: "new-draft-button",
    editor: {
      textarea: "draft-editor-textarea",
      saveButton: "draft-save-button",
      publishButton: "draft-publish-button",
      deleteButton: "draft-delete-button",
      aiAssistButton: "ai-assist-button",
    },
    searchInput: "drafts-search-input",
  },
  backlog: {
    intakeCard: "backlog-intake-card",
    textArea: "backlog-textarea",
    submitButton: "backlog-submit-button",
    saveButton: "backlog-save-button",
    previewPanel: "backlog-preview-panel",
    selectAllButton: "backlog-select-all-button",
    convertButton: "backlog-convert-button",
    entryCard: "backlog-entry-card",
    entriesTable: "backlog-entries-table",
    entryRow: "backlog-entry-row",
  },
  requirements: {
    registryView: "requirements-registry-view",
    targetsList: "operational-targets-list",
    requirementCard: "requirement-card",
    linkageIndicator: "linkage-indicator",
  },
  dialogs: {
    aiAssistant: {
      root: "ai-assistant-dialog",
      sectionSelect: "ai-section-select",
      promptInput: "ai-prompt-input",
      generateButton: "ai-generate-button",
      closeButton: "ai-dialog-close",
    },
    confirmation: {
      root: "confirmation-dialog",
      confirmButton: "confirm-button",
      cancelButton: "cancel-button",
    },
    newDraft: {
      root: "new-draft-dialog",
      nameInput: "new-draft-name-input",
      typeSelect: "new-draft-type-select",
      submitButton: "new-draft-submit-button",
    },
  },
  scenario: {
    tabs: {
      prd: "tab-prd",
      requirements: "tab-requirements",
      targets: "tab-targets",
    },
  },
  navigation: {
    orientationLink: "nav-orientation",
    catalogLink: "nav-catalog",
    draftsLink: "nav-drafts",
    backlogLink: "nav-backlog",
    qualityScannerLink: "nav-quality-scanner",
    requirementsLink: "nav-requirements",
  },
  qualityScanner: {
    scanButton: "scan-button",
    reportButton: "report-button",
    issueCard: "issue-card",
    severityBadge: "severity-badge",
  },
} as const satisfies LiteralSelectorTree;

// Future: Add dynamic selectors here if needed
// const dynamicSelectorDefinitions = {} as const;

const flattenLiteralSelectors = (
  tree: LiteralSelectorTree,
  prefix: string[] = [],
  target: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (typeof value === "string") {
      const manifestKey = nextPath.join(".");
      target[manifestKey] = {
        testId: value,
        selector: `[data-testid="${value}"]`,
      };
      continue;
    }
    flattenLiteralSelectors(value, nextPath, target);
  }
  return target;
};

const mergeLiteralNodes = (
  literalNode: LiteralSelectorTree | undefined,
  path: string[] = [],
): Record<string, unknown> => {
  const merged: Record<string, unknown> = {};
  const keys = Object.keys(literalNode ?? {});

  keys.forEach((key) => {
    const literalValue = literalNode?.[key];
    const nextPath = [...path, key];

    if (typeof literalValue === "string") {
      merged[key] = literalValue;
      return;
    }

    if (literalValue && typeof literalValue === "object") {
      merged[key] = mergeLiteralNodes(
        literalValue as LiteralSelectorTree,
        nextPath,
      );
    }
  });

  return merged;
};

const createSelectorRegistry = <L extends LiteralSelectorTree>(literalTree: L) => {
  const selectors = mergeLiteralNodes(literalTree) as any;
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: {},
  };
  return { selectors, manifest };
};

const { selectors, manifest } = createSelectorRegistry(literalSelectors);

// Export for UI components
export { selectors };

// Export for manifest generation script
export const selectorsManifest = manifest;
