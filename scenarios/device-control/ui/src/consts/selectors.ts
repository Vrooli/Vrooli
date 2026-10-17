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
  pages: {
    dashboard: "page-dashboard",
    onboardingReport: "page-dashboard-onboarding-report",
    dashboardDevices: "dashboard-devices",
    dashboardReprobe: "dashboard-reprobe",
    dashboardEmptyReprobe: "dashboard-empty-reprobe",
    dashboardAcquire: "dashboard-acquire",
    dashboardDiscovery: "dashboard-discovery",
    dashboardDiscoveryList: "dashboard-discovery-list",
    dashboardPairDialog: "dashboard-pair-dialog",
    dashboardPairPin: "dashboard-pair-pin",
    dashboardSessionKill: "dashboard-session-kill",
    flows: "page-flows",
    flowStrategy: "flow-strategy",
    flowDevice: "flow-device",
    flowDefinition: "flow-definition",
    flowValidate: "flow-validate",
    flowRun: "flow-run",
    flowGapReport: "flow-gap-report",
    flowActiveSession: "flow-active-session",
    flowKillSession: "flow-kill-session",
    flowRunReview: "flow-run-review",
    flowError: "flow-error",
    flowEvidenceImage: "flow-evidence-image",
    evidence: "page-evidence",
    settings: "page-settings",
    deviceDetail: "page-device-detail",
    deviceCapabilityMatrix: "device-capability-matrix",
    capabilityDisposition: "capability-disposition",
    unsupportedReason: "unsupported-reason",
    capabilityReason: "capability-reason",
    deviceStrategySelection: "device-strategy-selection",
    deviceStrategyRationale: "device-strategy-rationale",
    deviceIdentityClaims: "device-identity-claims",
    deviceIdentityReason: "device-identity-reason",
    deviceTransport: "device-transport",
    deviceUnreachableReason: "device-unreachable-reason",
    deviceProbeNow: "device-probe-now",
    deviceSessionHistory: "device-session-history",
    deviceRemotePanel: "device-remote-panel",
    deviceMediaPanel: "device-media-panel",
    devicePropertyPanel: "device-property-panel",
    deviceSensorPanel: "device-sensor-panel",
    deviceLogPanel: "device-log-panel",
    deviceLiveView: "device-live-view",
    deviceNowPlaying: "device-now-playing",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "flows",
            "evidence",
            "settings",
          ] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "flows",
            "evidence",
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
