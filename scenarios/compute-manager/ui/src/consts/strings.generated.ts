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
  "data-display": {
    "data-table": {
      "access-is-limited": "data-display.data-table.access-is-limited",
      all: "data-display.data-table.all",
      "data-table-content": "data-display.data-table.data-table-content",
      dense: "data-display.data-table.dense",
      next: "data-display.data-table.next",
      previous: "data-display.data-table.previous",
      roomy: "data-display.data-table.roomy",
      "row-density": "data-display.data-table.row-density",
    },
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
    description: "health.description",
    error: "health.error",
    loading: "health.loading",
    refresh: "health.refresh",
    refreshCount: "health.refreshCount",
    refreshCount_one: "health.refreshCount_one",
    serviceLabel: "health.serviceLabel",
    timestampLabel: "health.timestampLabel",
    title: "health.title",
  },
  layout: {
    closeNavigation: "layout.closeNavigation",
    mobileNavigationLabel: "layout.mobileNavigationLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      findings: "layout.nav.findings",
      request: "layout.nav.request",
      settings: "layout.nav.settings",
    },
    navigationLabel: "layout.navigationLabel",
    openNavigation: "layout.openNavigation",
    skipToContent: "layout.skipToContent",
  },
  locale: {
    switcherLabel: "locale.switcherLabel",
  },
  notifications: {
    summary: "notifications.summary",
    summary_one: "notifications.summary_one",
    summary_zero: "notifications.summary_zero",
  },
  pages: {
    dashboard: {
      description: "pages.dashboard.description",
      title: "pages.dashboard.title",
      inventoryTitle: "pages.dashboard.inventoryTitle",
      inventoryDescription: "pages.dashboard.inventoryDescription",
      loading: "pages.dashboard.loading",
      error: "pages.dashboard.error",
      empty: "pages.dashboard.empty",
      openFindings: "pages.dashboard.openFindings",
      noFindings: "pages.dashboard.noFindings",
      state: "pages.dashboard.state",
      region: "pages.dashboard.region",
      size: "pages.dashboard.size",
      remaining: "pages.dashboard.remaining",
    },
    findings: {
      title: "pages.findings.title",
      description: "pages.findings.description",
      kind: "pages.findings.kind",
      providerInstance: "pages.findings.providerInstance",
      detail: "pages.findings.detail",
      error: "pages.findings.error",
      empty: "pages.findings.empty",
    },
    instance: {
      title: "pages.instance.title",
      description: "pages.instance.description",
      error: "pages.instance.error",
      loading: "pages.instance.loading",
      state: "pages.instance.state",
      address: "pages.instance.address",
      provider: "pages.instance.provider",
    },
    request: {
      title: "pages.request.title",
      description: "pages.request.description",
      provider: "pages.request.provider",
      region: "pages.request.region",
      size: "pages.request.size",
      lifetime: "pages.request.lifetime",
      estimate: "pages.request.estimate",
      submit: "pages.request.submit",
      submitting: "pages.request.submitting",
      success: "pages.request.success",
      error: "pages.request.error",
      validation: "pages.request.validation",
    },
    settings: {
      description: "pages.settings.description",
      localeHeading: "pages.settings.localeHeading",
      localeHint: "pages.settings.localeHint",
      preferences: "pages.settings.preferences",
      themeHeading: "pages.settings.themeHeading",
      themeHint: "pages.settings.themeHint",
      title: "pages.settings.title",
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
