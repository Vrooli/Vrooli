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
    mobileMenuLabel: "layout.mobileMenuLabel",
    closeMenuLabel: "layout.closeMenuLabel",
    mainLabel: "layout.mainLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
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
    dashboard: {
      title: "pages.dashboard.title",
      description: "pages.dashboard.description",
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
  dashboard: {
    loadingTitle: "dashboard.loadingTitle",
    loadingDescription: "dashboard.loadingDescription",
    errorTitle: "dashboard.errorTitle",
    errorDescription: "dashboard.errorDescription",
    metrics: {
      openDebt: "dashboard.metrics.openDebt",
      deepStreak: "dashboard.metrics.deepStreak",
      versionLag: "dashboard.metrics.versionLag",
      templates: "dashboard.metrics.templates",
    },
    standing: {
      title: "dashboard.standing.title",
      description: "dashboard.standing.description",
    },
    registry: {
      title: "dashboard.registry.title",
      description: "dashboard.registry.description",
    },
    monitor: {
      title: "dashboard.monitor.title",
      description: "dashboard.monitor.description",
      status: "dashboard.monitor.status",
      nextRun: "dashboard.monitor.nextRun",
      interval: "dashboard.monitor.interval",
      streak: "dashboard.monitor.streak",
      lastRun: "dashboard.monitor.lastRun",
      unscheduled: "dashboard.monitor.unscheduled",
    },
    runs: {
      title: "dashboard.runs.title",
      description: "dashboard.runs.description",
      empty: "dashboard.runs.empty",
    },
    debt: {
      title: "dashboard.debt.title",
      description: "dashboard.debt.description",
      empty: "dashboard.debt.empty",
    },
    drift: {
      title: "dashboard.drift.title",
      description: "dashboard.drift.description",
      empty: "dashboard.drift.empty",
    },
    empty: {
      title: "dashboard.empty.title",
      description: "dashboard.empty.description",
    },
  },
} as const;

export type Strings = typeof strings;
