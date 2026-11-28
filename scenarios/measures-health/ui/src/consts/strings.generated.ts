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
    mainLabel: "layout.mainLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      fleet: "layout.nav.fleet",
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
      statPlaceholderLabel: "pages.dashboard.statPlaceholderLabel",
    },
    fleet: {
      title: "pages.fleet.title",
      description: "pages.fleet.description",
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
  fleet: {
    title: "fleet.title",
    loading: "fleet.loading",
    empty: "fleet.empty",
    refresh: "fleet.refresh",
    passed: "fleet.passed",
    failed: "fleet.failed",
    col: {
      scenario: "fleet.col.scenario",
      verdict: "fleet.col.verdict",
      expected: "fleet.col.expected",
      covered: "fleet.col.covered",
      waived: "fleet.col.waived",
      uncovered: "fleet.col.uncovered",
      tier: "fleet.col.tier",
      measures: "fleet.col.measures",
    },
    status: {
      covered: "fleet.status.covered",
      uncovered: "fleet.status.uncovered",
      waived: "fleet.status.waived",
      notExpected: "fleet.status.notExpected",
    },
    tier: {
      full: "fleet.tier.full",
      partial: "fleet.tier.partial",
      fallback: "fleet.tier.fallback",
      none: "fleet.tier.none",
    },
    detail: {
      title: "fleet.detail.title",
      hint: "fleet.detail.hint",
      loading: "fleet.detail.loading",
      empty: "fleet.detail.empty",
      measureCount: "fleet.detail.measureCount",
      measureCount_one: "fleet.detail.measureCount_one",
      waiverLabel: "fleet.detail.waiverLabel",
      noteLabel: "fleet.detail.noteLabel",
      probePassed: "fleet.detail.probePassed",
      probeFailed: "fleet.detail.probeFailed",
    },
  },
  measures: {
    title: "measures.title",
    description: "measures.description",
    loading: "measures.loading",
    windowLabel: "measures.windowLabel",
    failedLabel: "measures.failedLabel",
    failedLabel_one: "measures.failedLabel_one",
    passedLabel: "measures.passedLabel",
    passedLabel_one: "measures.passedLabel_one",
    window: {
      thisWeek: "measures.window.thisWeek",
      last7d: "measures.window.last7d",
      last30d: "measures.window.last30d",
      thisMonth: "measures.window.thisMonth",
      lastMonth: "measures.window.lastMonth",
      thisQuarter: "measures.window.thisQuarter",
    },
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
