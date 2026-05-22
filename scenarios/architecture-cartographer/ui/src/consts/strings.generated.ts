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
    mainContentLabel: "layout.mainContentLabel",
    nav: {
      overview: "layout.nav.overview",
      targets: "layout.nav.targets",
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
    overview: {
      title: "pages.overview.title",
      description: "pages.overview.description",
      recentTargetsHeading: "pages.overview.recentTargetsHeading",
      activeSnapshotsHeading: "pages.overview.activeSnapshotsHeading",
      healthHeading: "pages.overview.healthHeading",
      noActiveSnapshots: "pages.overview.noActiveSnapshots",
      loadingSnapshots: "pages.overview.loadingSnapshots",
      snapshotsError: "pages.overview.snapshotsError",
      startExtraction: "pages.overview.startExtraction",
    },
    newTarget: {
      title: "pages.newTarget.title",
      description: "pages.newTarget.description",
      scenarioPathLabel: "pages.newTarget.scenarioPathLabel",
      scenarioPathHint: "pages.newTarget.scenarioPathHint",
      submitButton: "pages.newTarget.submitButton",
      submitting: "pages.newTarget.submitting",
      successMessage: "pages.newTarget.successMessage",
      fromCacheMessage: "pages.newTarget.fromCacheMessage",
      openWorkspace: "pages.newTarget.openWorkspace",
      validationRequired: "pages.newTarget.validationRequired",
      validationInvalid: "pages.newTarget.validationInvalid",
    },
    targetWorkspace: {
      title: "pages.targetWorkspace.title",
      scenarioLabel: "pages.targetWorkspace.scenarioLabel",
      subnavComingSoon: "pages.targetWorkspace.subnavComingSoon",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
  },
  targets: {
    recent: {
      openButton: "targets.recent.openButton",
      removeAriaLabel: "targets.recent.removeAriaLabel",
      emptyTitle: "targets.recent.emptyTitle",
      emptyDescription: "targets.recent.emptyDescription",
      openedAt: "targets.recent.openedAt",
    },
    snapshots: {
      snapshotIdLabel: "targets.snapshots.snapshotIdLabel",
      scenarioLabel: "targets.snapshots.scenarioLabel",
      extractedAtLabel: "targets.snapshots.extractedAtLabel",
      openButton: "targets.snapshots.openButton",
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
  shared: {
    empty: {
      title: "shared.empty.title",
      description: "shared.empty.description",
    },
    loading: {
      label: "shared.loading.label",
    },
    error: {
      title: "shared.error.title",
      retry: "shared.error.retry",
    },
    dataTable: {
      empty: "shared.dataTable.empty",
    },
    severity: {
      info: "shared.severity.info",
      low: "shared.severity.low",
      medium: "shared.severity.medium",
      high: "shared.severity.high",
      critical: "shared.severity.critical",
    },
    diff: {
      added: "shared.diff.added",
      removed: "shared.diff.removed",
      unchanged: "shared.diff.unchanged",
    },
    splitPane: {
      resizeHandle: "shared.splitPane.resizeHandle",
    },
    routeError: {
      title: "shared.routeError.title",
      message: "shared.routeError.message",
      retry: "shared.routeError.retry",
      home: "shared.routeError.home",
    },
  },
} as const;

export type Strings = typeof strings;
