/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. Types, helpers, and the registry builder are
 * imported from ./selectorTypes.ts. This file defines the literal and dynamic
 * selector maps and exports the final `selectors` and `selectorsManifest`.
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

import type { LiteralSelectorTree, DynamicSelectorTree } from "./selectorTypes";
import { createSelectorRegistry } from "./selectorTypes";

const literalSelectors: LiteralSelectorTree = {
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
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
