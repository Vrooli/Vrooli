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
    eyebrow: "app.eyebrow",
    description: "app.description",
  },
  layout: {
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      focus: "layout.nav.focus",
      convergence: "layout.nav.convergence",
      trials: "layout.nav.trials",
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
  common: {
    loading: "common.loading",
    error: "common.error",
  },
  pages: {
    dashboard: {
      title: "pages.dashboard.title",
      description: "pages.dashboard.description",
      projectionHeading: "pages.dashboard.projectionHeading",
      coverageLabel: "pages.dashboard.coverageLabel",
      confidenceLabel: "pages.dashboard.confidenceLabel",
      cellsLabel: "pages.dashboard.cellsLabel",
      trendHeading: "pages.dashboard.trendHeading",
      trendValue: "pages.dashboard.trendValue",
      noTrend: "pages.dashboard.noTrend",
      unavailableReason: "pages.dashboard.unavailableReason",
      empty: "pages.dashboard.empty",
    },
    focus: {
      title: "pages.focus.title",
      description: "pages.focus.description",
      focusHeading: "pages.focus.focusHeading",
      gapsHeading: "pages.focus.gapsHeading",
      priorityLabel: "pages.focus.priorityLabel",
      globalBadge: "pages.focus.globalBadge",
      empty: "pages.focus.empty",
    },
    convergence: {
      title: "pages.convergence.title",
      description: "pages.convergence.description",
      methodologyNote: "pages.convergence.methodologyNote",
      templatesHeading: "pages.convergence.templatesHeading",
      referencesHeading: "pages.convergence.referencesHeading",
      lensLabel: "pages.convergence.lensLabel",
      referenceLabel: "pages.convergence.referenceLabel",
      cleanBadge: "pages.convergence.cleanBadge",
      empty: "pages.convergence.empty",
    },
    trials: {
      title: "pages.trials.title",
      description: "pages.trials.description",
      coverageHeading: "pages.trials.coverageHeading",
      coverageValue: "pages.trials.coverageValue",
      historyHeading: "pages.trials.historyHeading",
      recentHeading: "pages.trials.recentHeading",
      historyPoint: "pages.trials.historyPoint",
      runMetrics: "pages.trials.runMetrics",
      empty: "pages.trials.empty",
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
