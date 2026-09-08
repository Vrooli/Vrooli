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
    fleet: {
    panel: "fleet-panel",
    loading: "fleet-loading",
    error: "fleet-error",
    empty: "fleet-empty",
    list: "fleet-list",
    failedOnboardings: "fleet-failed-onboardings",
    bridgeReadiness: "fleet-bridge-readiness",
    machineLifecycle: "machine-lifecycle-panel",
    management: "fleet-node-management",
    managementNameInput: "fleet-node-management-name",
    pairing: {
      disclosure: "fleet-pairing-disclosure",
      form: "fleet-pairing-form",
      nameInput: "fleet-pairing-name",
      submit: "fleet-pairing-submit",
      error: "fleet-pairing-error",
      result: "fleet-pairing-result",
      code: "fleet-pairing-code",
      copy: "fleet-pairing-copy",
    },
    pairingRequests: {
      panel: "fleet-pairing-requests",
    },
    onboard: {
      form: "fleet-onboard-form",
      addNode: "fleet-onboard-add-node",
      stepIndicator: "fleet-onboard-step-indicator",
      next: "fleet-onboard-next",
      back: "fleet-onboard-back",
      moreToggle: "fleet-onboard-more-toggle",
      advancedToggle: "fleet-onboard-advanced-toggle",
      host: "fleet-onboard-host",
      user: "fleet-onboard-user",
      port: "fleet-onboard-port",
      name: "fleet-onboard-name",
      password: "fleet-onboard-password",
      passwordToggle: "fleet-onboard-password-toggle",
      capabilities: "fleet-onboard-capabilities",
      revision: "fleet-onboard-revision",
      provisionSudo: "fleet-onboard-provision-sudo",
      controlPlaneUrl: "fleet-onboard-control-plane-url",
      setupPreset: "fleet-onboard-setup-preset",
      setupResources: "fleet-onboard-setup-resources",
      setupScenarios: "fleet-onboard-setup-scenarios",
      includeOptional: "fleet-onboard-include-optional",
      sourceWorkingTree: "fleet-onboard-source-working-tree",
      sourceWarning: "fleet-onboard-source-warning",
      sourceSummary: "fleet-onboard-source-summary",
      submit: "fleet-onboard-submit",
      error: "fleet-onboard-error",
      progress: "fleet-onboard-progress",
      steps: "fleet-onboard-steps",
      success: "fleet-onboard-success",
      failure: "fleet-onboard-failure",
      failureOutput: "fleet-onboard-failure-output",
      failureOutputToggle: "fleet-onboard-failure-output-toggle",
    },
  },
  runs: {
    panel: "runs-panel",
    loading: "runs-loading",
    error: "runs-error",
    empty: "runs-empty",
    list: "runs-list",
    detail: "runs-detail",
    output: "runs-output",
    artifacts: "runs-artifacts",
  },
  notifications: {
    summary: "notifications-summary",
  },
  locale: {
    switcher: "locale-switcher",
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
    runs: "page-runs",
    settings: "page-settings",
    sessions: "page-sessions",
    rollouts: "page-rollouts",
    trust: "page-trust",
    setup: "page-setup",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  session: {
    signInScreen: "session-sign-in-screen",
    firstTimeNote: "session-first-time-note",
    owner: {
      panel: "session-owner-panel",
      status: "session-owner-status",
      signOutButton: "session-owner-sign-out",
    },
    topbar: {
      status: "session-topbar-status",
      signOut: "session-topbar-sign-out",
    },
    login: {
      form: "session-login-form",
      tabSignIn: "session-login-tab-signin",
      tabCreate: "session-login-tab-create",
      emailInput: "session-login-email",
      passwordInput: "session-login-password",
      usernameInput: "session-login-username",
      submit: "session-login-submit",
      error: "session-login-error",
    },
  },
} satisfies LiteralSelectorTree;

// Per-locale toggle test IDs are emitted by `locale.toggle({ code })` below.
// We deliberately do NOT also declare static `toggleEn` / `toggleJa` literals —
// the dynamic form is the single source of truth, and duplicating it here would
// drift the moment a new locale is added to LOCALE_CODES.
//
// `code` is constrained to `LOCALE_CODES` so `selectors.locale.toggle({ code: "fr" })`
// is a TypeScript error when "fr" isn't a supported locale. The runtime enum
// validation in `normalizeParams` provides the same guarantee at call time.
const dynamicSelectorDefinitions = {
  locale: {
    toggle: defineDynamicSelector({
      description: "Locale toggle button by language code",
      testIdPattern: "locale-toggle-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "runs",
            "settings",
            "sessions",
            "rollouts",
            "trust",
            "setup",
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
            "runs",
            "settings",
            "sessions",
            "rollouts",
            "trust",
            "setup",
          ] as const,
        },
      },
    }),
  },
  fleet: {
    pairingRequests: {
      row: defineDynamicSelector({
        description: "Pending pairing request by request id",
        testIdPattern: "fleet-pairing-request-${id}",
        params: { id: { type: "string" } },
      }),
      words: defineDynamicSelector({
        description: "Confirmation words for a pending pairing request",
        testIdPattern: "fleet-pairing-request-words-${id}",
        params: { id: { type: "string" } },
      }),
      preset: defineDynamicSelector({
        description: "Permission preset for a pending pairing request",
        testIdPattern: "fleet-pairing-request-preset-${id}",
        params: { id: { type: "string" } },
      }),
      wordsMatch: defineDynamicSelector({
        description: "Confirmation checkbox for a pending pairing request",
        testIdPattern: "fleet-pairing-request-words-match-${id}",
        params: { id: { type: "string" } },
      }),
      approve: defineDynamicSelector({
        description: "Approve a pending pairing request",
        testIdPattern: "fleet-pairing-request-approve-${id}",
        params: { id: { type: "string" } },
      }),
      reject: defineDynamicSelector({
        description: "Reject a pending pairing request",
        testIdPattern: "fleet-pairing-request-reject-${id}",
        params: { id: { type: "string" } },
      }),
      error: defineDynamicSelector({
        description: "Pairing approval error",
        testIdPattern: "fleet-pairing-request-error",
      }),
    },
    machineLifecycleRow: defineDynamicSelector({
      description: "Durable Machine lifecycle row by Machine id",
      testIdPattern: "machine-lifecycle-${id}",
      params: { id: { type: "string" } },
    }),
    row: defineDynamicSelector({
      description: "Fleet node row by node id",
      testIdPattern: "fleet-row-${id}",
      params: { id: { type: "string" } },
    }),
    revoke: defineDynamicSelector({
      description: "Revoke action for a fleet node by id",
      testIdPattern: "fleet-revoke-${id}",
      params: { id: { type: "string" } },
    }),
    jobs: defineDynamicSelector({
      description: "Live job-status summary for a fleet node by id",
      testIdPattern: "fleet-jobs-${id}",
      params: { id: { type: "string" } },
    }),
    onboardStep: defineDynamicSelector({
      description: "One onboarding step row by step id",
      testIdPattern: "fleet-onboard-step-${step}",
      params: { step: { type: "string" } },
    }),
    onboardRetry: defineDynamicSelector({
      description: "Retry action for a saved failed onboarding target by operation id",
      testIdPattern: "fleet-onboard-retry-${id}",
      params: { id: { type: "string" } },
    }),
    onboardViewLogs: defineDynamicSelector({
      description: "View-or-hide durable onboarding diagnostics by operation id",
      testIdPattern: "fleet-onboard-view-logs-${id}",
      params: { id: { type: "string" } },
    }),
    onboardRemove: defineDynamicSelector({
      description: "Remove action for a saved failed onboarding target by operation id",
      testIdPattern: "fleet-onboard-remove-${id}",
      params: { id: { type: "string" } },
    }),
  },
  runs: {
    row: defineDynamicSelector({
      description: "Run-history row by run id",
      testIdPattern: "runs-row-${id}",
      params: { id: { type: "string" } },
    }),
    view: defineDynamicSelector({
      description: "View-output action for a run by id",
      testIdPattern: "runs-view-${id}",
      params: { id: { type: "string" } },
    }),
    cancel: defineDynamicSelector({
      description: "Cancel (abort) action for an in-flight run by id",
      testIdPattern: "runs-cancel-${id}",
      params: { id: { type: "string" } },
    }),
    artifact: defineDynamicSelector({
      description: "Downloadable artifact link by run id and ordinal index",
      testIdPattern: "runs-artifact-${id}-${index}",
      params: { id: { type: "string" }, index: { type: "number" } },
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
