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
    mainLabel: "layout.mainLabel",
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
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
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    fleet: {
      title: "pages.fleet.title",
      description: "pages.fleet.description",
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
  fleet: {
    title: "fleet.title",
    refresh: "fleet.refresh",
    loading: "fleet.loading",
    empty: "fleet.empty",
    summary: {
      scenarios: "fleet.summary.scenarios",
      passing: "fleet.summary.passing",
      missingFreshness: "fleet.summary.missingFreshness",
      autofixable: "fleet.summary.autofixable",
    },
    profiles: {
      title: "fleet.profiles.title",
      unrecognized: "fleet.profiles.unrecognized",
      countLabel: "fleet.profiles.countLabel",
      countLabel_one: "fleet.profiles.countLabel_one",
    },
    rules: {
      title: "fleet.rules.title",
      empty: "fleet.rules.empty",
      col: {
        code: "fleet.rules.col.code",
        severity: "fleet.rules.col.severity",
        offenders: "fleet.rules.col.offenders",
        findings: "fleet.rules.col.findings",
        autofixable: "fleet.rules.col.autofixable",
      },
    },
    scenarios: {
      title: "fleet.scenarios.title",
      passed: "fleet.scenarios.passed",
      failed: "fleet.scenarios.failed",
      recognized: "fleet.scenarios.recognized",
      unrecognized: "fleet.scenarios.unrecognized",
      missingFreshnessBadge: "fleet.scenarios.missingFreshnessBadge",
      col: {
        scenario: "fleet.scenarios.col.scenario",
        verdict: "fleet.scenarios.col.verdict",
        profile: "fleet.scenarios.col.profile",
        errors: "fleet.scenarios.col.errors",
        warnings: "fleet.scenarios.col.warnings",
        autofixable: "fleet.scenarios.col.autofixable",
        freshness: "fleet.scenarios.col.freshness",
      },
    },
    errors: {
      title: "fleet.errors.title",
    },
  },
} as const;

export type Strings = typeof strings;
