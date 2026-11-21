import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    title: "app-title",
    eyebrow: "app-eyebrow",
    description: "app-description",
  },
  health: {
    card: "health-card",
    loading: "health-loading",
    error: "health-error",
    statusValue: "health-status-value",
    serviceValue: "health-service-value",
    timestampValue: "health-timestamp-value",
    refreshButton: "health-refresh-button",
    refreshCount: "health-refresh-count",
  },
  notifications: {
    summary: "notifications-summary",
  },
  layout: {
    shell: "layout-shell",
    topBar: "layout-top-bar",
    sidebar: "layout-sidebar",
    bottomNav: "layout-bottom-nav",
    main: "layout-main",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  delivery: {
    targetMatrix: "dashboard-target-matrix",
    targetDisposition: "dashboard-target-disposition",
    gateVerdict: "dashboard-gate-verdict",
    readinessSummary: "dashboard-readiness-summary",
    executingNode: "dashboard-executing-node",
    generateProject: "dashboard-generate-project",
    rowPromotability: "dashboard-row-promotability",
  },
  distribution: {
    channelList: "distribution-channel-list",
    channelAvailability: "distribution-channel-availability",
    channelRequirement: "distribution-channel-requirement",
    channelBlockingRung: "distribution-channel-blocking-rung",
    channelNextAction: "distribution-channel-next-action",
    artifactIdentity: "distribution-artifact-identity",
    promotabilityBlock: "distribution-promotability-block",
  },
  readiness: {
    rungList: "readiness-rung-list",
    rungState: "readiness-rung-state",
    rungOwner: "readiness-rung-owner",
    rungCost: "readiness-rung-cost",
    rungNextAction: "readiness-rung-next-action",
    rungDeadline: "readiness-rung-deadline",
    mirrorPathNote: "readiness-mirror-path-note",
  },
  runReview: {
    runVerdict: "run-review-verdict",
    chapterList: "run-review-chapter-list",
    chapterExpected: "run-review-chapter-expected",
    chapterObserved: "run-review-chapter-observed",
    chapterStrategy: "run-review-chapter-strategy",
    promotabilityNotice: "run-review-promotability-notice",
    exceededBound: "run-review-exceeded-bound",
    recording: "run-review-recording",
    leaseInterruption: "run-review-lease-interruption",
  },
  targetDetail: {
    targetDisposition: "target-detail-disposition",
    capabilityList: "target-detail-capability-list",
    probeTimestamp: "target-detail-probe-timestamp",
    missingCapability: "target-detail-missing-capability",
    nextAction: "target-detail-next-action",
    transport: "target-detail-transport",
    strategyTier: "target-detail-strategy-tier",
    leaseHolder: "target-detail-lease-holder",
    deviceIdentity: "target-detail-device-identity",
  },
  settingsPage: {
    form: "settings-form",
    signingIdentity: "settings-signing-identity-ref",
    defaultTransport: "settings-default-transport",
    localeControl: "settings-locale-control",
  },
  pages: {
    dashboard: "page-dashboard",
    targetDetail: "page-target-detail",
    runReview: "page-run-review",
    readiness: "page-readiness",
    distribution: "page-distribution",
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    navLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "settings",
          ] as const,
        },
      },
    }),

  },
  settingsPage: {
    themeOption: defineDynamicSelector({
      description: "Theme choice radio button on the settings page",
      testIdPattern: "page-settings-theme-${choice}",
      params: { choice: { type: "enum", values: ["light", "dark", "system"] as const } },
    }),
    localeOption: defineDynamicSelector({
      description: "Locale choice radio button on the settings page",
      testIdPattern: "page-settings-locale-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
