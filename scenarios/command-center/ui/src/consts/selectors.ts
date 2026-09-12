import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import type { LiteralSelectorTree, DynamicSelectorTree } from "@vrooli/ui-selectors";
import { createSelectorRegistry } from "@vrooli/ui-selectors";

const literalSelectors = {
  dashboard: {
    metricList: "metric-list",
    sceneCanvas: "scene-canvas",
    sceneStill: "scene-still",
    roomHero: "room-hero",
    roomLegend: "room-legend",
    roomSources: "room-sources",
    freshnessHairline: "freshness-hairline",
    samplesModeStamp: "samples-mode-stamp",
    cycleRail: "cycle-rail",
    controlBar: "control-bar-controls",
    shortcutHelp: "shortcut-help",
    errorBanner: "error-banner",
    loading: "loading",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
