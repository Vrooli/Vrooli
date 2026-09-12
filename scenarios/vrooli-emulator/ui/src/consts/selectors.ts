import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  /*
  Example literal selectors:
  dashboard: {
    newProjectButton: 'dashboard-new-project-button',
  },
  */
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  /*
  Example dynamic selectors:
  projects: {
    cardByName: defineDynamicSelector({
      description: 'Project card filtered by name',
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: 'string' } },
    }),
  },
  */
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
