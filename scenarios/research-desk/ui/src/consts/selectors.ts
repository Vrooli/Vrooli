import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    title: "app-title",
  },
  layout: {
    // The library AppShell derives every part id from the root id it is given.
    shell: "layout-shell",
    navigation: "layout-shell-navigation",
    tabs: "layout-shell-tabs",
    main: "layout-shell-main",
    brand: "layout-shell-brand",
    skip: "layout-shell-skip",
  },
  desk: {
    themeSelect: "page-desk-theme",
  },
  launch: {
    launchButton: "launch-button",
    blocker: "launch-blocker",
  },
  now: {
    status: "now-status",
  },
  pages: {
    now: "page-now",
    needsYou: "page-needs-you",
    topics: "page-topics",
    topic: "page-topic",
    commissions: "page-commissions",
    commission: "page-commission",
    launch: "page-launch",
    launchStatus: "page-launch-status",
    reviews: "page-reviews",
    review: "page-review",
    file: "page-file",
    desk: "page-desk",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    navLink: defineDynamicSelector({
      description: "Primary navigation link by canonical nav key; the phone tab derives `-tab` from the same id",
      testIdPattern: "layout-nav-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "now",
            "needsYou",
            "topics",
            "commissions",
            "launch",
            "reviews",
            "desk",
          ] as const,
        },
      },
    }),
    navTab: defineDynamicSelector({
      description: "Phone tab-bar item by canonical nav key",
      testIdPattern: "layout-nav-${key}-tab",
      params: {
        key: {
          type: "enum",
          values: [
            "now",
            "needsYou",
            "topics",
            "commissions",
            "launch",
            "reviews",
            "desk",
          ] as const,
        },
      },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
