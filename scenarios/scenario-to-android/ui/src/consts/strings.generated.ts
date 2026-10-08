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
    description: "app.description",
    eyebrow: "app.eyebrow",
    title: "app.title",
  },
  errorBoundary: {
    message: "errorBoundary.message",
    retry: "errorBoundary.retry",
    title: "errorBoundary.title",
  },
  errors: {
    aborted: "errors.aborted",
    already_exists: "errors.already_exists",
    canceled: "errors.canceled",
    data_loss: "errors.data_loss",
    deadline_exceeded: "errors.deadline_exceeded",
    failed_precondition: "errors.failed_precondition",
    internal: "errors.internal",
    invalid_argument: "errors.invalid_argument",
    not_found: "errors.not_found",
    out_of_range: "errors.out_of_range",
    permission_denied: "errors.permission_denied",
    resource_exhausted: "errors.resource_exhausted",
    unauthenticated: "errors.unauthenticated",
    unavailable: "errors.unavailable",
    unimplemented: "errors.unimplemented",
    unknown: "errors.unknown",
  },
  health: {
    error: "health.error",
    loading: "health.loading",
    refresh: "health.refresh",
    refreshCount: "health.refreshCount",
    refreshCount_one: "health.refreshCount_one",
    serviceLabel: "health.serviceLabel",
    statusLabel: "health.statusLabel",
    timestampLabel: "health.timestampLabel",
    title: "health.title",
  },
  layout: {
    bottomNavLabel: "layout.bottomNavLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      readiness: "layout.nav.readiness",
      runs: "layout.nav.runs",
      settings: "layout.nav.settings",
      targets: "layout.nav.targets",
    },
    sidebarLabel: "layout.sidebarLabel",
  },
  locale: {
    switcherLabel: "locale.switcherLabel",
  },
  navigation: {
    "app-shell": {
      "close-navigation": "navigation.app-shell.close-navigation",
      "mobile-navigation": "navigation.app-shell.mobile-navigation",
      "open-navigation": "navigation.app-shell.open-navigation",
      "primary-navigation": "navigation.app-shell.primary-navigation",
      "skip-to-content": "navigation.app-shell.skip-to-content",
    },
  },
  notifications: {
    summary: "notifications.summary",
    summary_one: "notifications.summary_one",
    summary_zero: "notifications.summary_zero",
  },
  pages: {
    dashboard: {
      description: "pages.dashboard.description",
      statPlaceholderLabel: "pages.dashboard.statPlaceholderLabel",
      title: "pages.dashboard.title",
    },
    readiness: {
      description: "pages.readiness.description",
      next: "pages.readiness.next",
      obligation: "pages.readiness.obligation",
      owner: "pages.readiness.owner",
      title: "pages.readiness.title",
      unknown: "pages.readiness.unknown",
    },
    runs: {
      description: "pages.runs.description",
      empty: "pages.runs.empty",
      evidence: "pages.runs.evidence",
      gate: "pages.runs.gate",
      none: "pages.runs.none",
      notPassed: "pages.runs.notPassed",
      passed: "pages.runs.passed",
      title: "pages.runs.title",
    },
    settings: {
      localeHeading: "pages.settings.localeHeading",
      themeHeading: "pages.settings.themeHeading",
      title: "pages.settings.title",
    },
    targets: {
      description: "pages.targets.description",
      empty: "pages.targets.empty",
      kind: "pages.targets.kind",
      next: "pages.targets.next",
      ready: "pages.targets.ready",
      readyForSelection: "pages.targets.readyForSelection",
      title: "pages.targets.title",
      unavailable: "pages.targets.unavailable",
    },
  },
  theme: {
    choice: {
      dark: "theme.choice.dark",
      light: "theme.choice.light",
      system: "theme.choice.system",
    },
    switcherLabel: "theme.switcherLabel",
  },
} as const;

export type Strings = typeof strings;
