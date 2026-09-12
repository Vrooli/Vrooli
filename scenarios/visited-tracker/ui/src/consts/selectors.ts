import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry } from "@vrooli/ui-selectors";
export { defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  campaignsList: 'campaigns-list',
  filesList: 'files-list',
  fileRow: 'file-row',
} as const;

const dynamicSelectorDefinitions = {};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
