// AUTO-GENERATED — do not edit by hand.
//
// Source : src/i18n/locales/en.json
// Codegen: scripts/gen-strings.mjs (invoked automatically by
//          vite-plugin-strings-codegen on dev start, HMR of en.json, and
//          build start; also available as `pnpm strings:gen` and
//          `pnpm strings:check`).
//
// See src/consts/strings.ts for the registry's purpose, why it exists, and
// how to use it. This file mirrors the shape of en.json with each leaf
// replaced by its dotted key path — that's the value the i18next `t()`
// function takes as its first argument.

export const strings = {
  app: {
    title: "app.title",
  },
  layout: {
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    nav: {
      posture: "layout.nav.posture",
      dependencies: "layout.nav.dependencies",
      secrets: "layout.nav.secrets",
      settings: "layout.nav.settings",
    },
  },
  theme: {
    switcherLabel: "theme.switcherLabel",
    choice: {
      light: "theme.choice.light",
      dark: "theme.choice.dark",
      system: "theme.choice.system",
    },
  },
  pages: {
    posture: {
      title: "pages.posture.title",
      description: "pages.posture.description",
    },
    dependencies: {
      title: "pages.dependencies.title",
      description: "pages.dependencies.description",
    },
    secrets: {
      title: "pages.secrets.title",
      description: "pages.secrets.description",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
  },
  health: {
    title: "health.title",
    loading: "health.loading",
    error: "health.error",
    refresh: "health.refresh",
    refreshCount: "health.refreshCount",
    refreshCount_one: "health.refreshCount_one",
    statusLabel: "health.statusLabel",
    serviceLabel: "health.serviceLabel",
    timestampLabel: "health.timestampLabel",
  },
  notifications: {
    summary: "notifications.summary",
    summary_zero: "notifications.summary_zero",
    summary_one: "notifications.summary_one",
  },
  posture: {
    title: "posture.title",
    scenarioLabel: "posture.scenarioLabel",
    scenarioPlaceholder: "posture.scenarioPlaceholder",
    scan: "posture.scan",
    scanning: "posture.scanning",
    loading: "posture.loading",
    empty: "posture.empty",
    passed: "posture.passed",
    failed: "posture.failed",
    lastScan: "posture.lastScan",
    summary: "posture.summary",
    skippedScanners: "posture.skippedScanners",
    remediationLabel: "posture.remediationLabel",
    fileLabel: "posture.fileLabel",
    severity: {
      error: "posture.severity.error",
      warning: "posture.severity.warning",
      info: "posture.severity.info",
    },
  },
  dependencies: {
    title: "dependencies.title",
    queryLabel: "dependencies.queryLabel",
    queryPlaceholder: "dependencies.queryPlaceholder",
    search: "dependencies.search",
    searching: "dependencies.searching",
    vulnerableOnly: "dependencies.vulnerableOnly",
    ecosystem: "dependencies.ecosystem",
    ecosystemAll: "dependencies.ecosystemAll",
    ecosystemGo: "dependencies.ecosystemGo",
    ecosystemNpm: "dependencies.ecosystemNpm",
    empty: "dependencies.empty",
    textModeHint: "dependencies.textModeHint",
    indexedCount: "dependencies.indexedCount",
    vulnerableCount: "dependencies.vulnerableCount",
    lastReconcile: "dependencies.lastReconcile",
    columns: {
      scenario: "dependencies.columns.scenario",
      name: "dependencies.columns.name",
      version: "dependencies.columns.version",
      ecosystem: "dependencies.columns.ecosystem",
      vulns: "dependencies.columns.vulns",
    },
  },
  secrets: {
    redactedNote: "secrets.redactedNote",
  },
  widget: {
    title: "widget.title",
    loading: "widget.loading",
    error: "widget.error",
    clean: "widget.clean",
  },
  errors: {
    canceled: "errors.canceled",
    unknown: "errors.unknown",
    invalid_argument: "errors.invalid_argument",
    deadline_exceeded: "errors.deadline_exceeded",
    not_found: "errors.not_found",
    already_exists: "errors.already_exists",
    permission_denied: "errors.permission_denied",
    resource_exhausted: "errors.resource_exhausted",
    failed_precondition: "errors.failed_precondition",
    aborted: "errors.aborted",
    out_of_range: "errors.out_of_range",
    unimplemented: "errors.unimplemented",
    internal: "errors.internal",
    unavailable: "errors.unavailable",
    data_loss: "errors.data_loss",
    unauthenticated: "errors.unauthenticated",
  },
  locale: {
    switcherLabel: "locale.switcherLabel",
  },
  errorBoundary: {
    title: "errorBoundary.title",
    message: "errorBoundary.message",
    retry: "errorBoundary.retry",
  },
} as const;

export type Strings = typeof strings;
