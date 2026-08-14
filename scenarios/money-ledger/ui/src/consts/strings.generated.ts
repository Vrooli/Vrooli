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
    description: "app.description",
  },
  layout: {
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    mainLabel: "layout.mainLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      journal: "layout.nav.journal",
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
      runwayLabel: "pages.dashboard.runwayLabel",
      runwayMonths: "pages.dashboard.runwayMonths",
      runwayUndefined: "pages.dashboard.runwayUndefined",
      runwayBasis: "pages.dashboard.runwayBasis",
      completenessLabel: "pages.dashboard.completenessLabel",
      partial: "pages.dashboard.partial",
      complete: "pages.dashboard.complete",
      notConfigured: "pages.dashboard.notConfigured",
      missingAdapter: "pages.dashboard.missingAdapter",
      positionUnavailable: "pages.dashboard.positionUnavailable",
      sourceUnavailable: "pages.dashboard.sourceUnavailable",
      emptyGuidance: "pages.dashboard.emptyGuidance",
      goalsTitle: "pages.dashboard.goalsTitle",
      goalsDescription: "pages.dashboard.goalsDescription",
      changeTitle: "pages.dashboard.changeTitle",
      changeDescription: "pages.dashboard.changeDescription",
    },
    journal: {
      title: "pages.journal.title",
      cardTitle: "pages.journal.cardTitle",
      description: "pages.journal.description",
      eventBasis: "pages.journal.eventBasis",
      eventSource: "pages.journal.eventSource",
      reversalLink: "pages.journal.reversalLink",
      reversalReason: "pages.journal.reversalReason",
      auditTrail: "pages.journal.auditTrail",
      manualEntry: "pages.journal.manualEntry",
      emptyGuidance: "pages.journal.emptyGuidance",
    },
    accounts: {
      title: "pages.accounts.title",
      cardTitle: "pages.accounts.cardTitle",
      description: "pages.accounts.description",
      balanceBasis: "pages.accounts.balanceBasis",
      balanceGap: "pages.accounts.balanceGap",
      transferPair: "pages.accounts.transferPair",
      emptyGuidance: "pages.accounts.emptyGuidance",
    },
    adapters: {
      title: "pages.adapters.title",
      cardTitle: "pages.adapters.cardTitle",
      description: "pages.adapters.description",
      manualAdapter: "pages.adapters.manualAdapter",
      available: "pages.adapters.available",
      unavailable: "pages.adapters.unavailable",
      failureReason: "pages.adapters.failureReason",
      lastSuccessAge: "pages.adapters.lastSuccessAge",
      missingImpact: "pages.adapters.missingImpact",
      credentialGap: "pages.adapters.credentialGap",
      emptyGuidance: "pages.adapters.emptyGuidance",
    },
    statements: {
      title: "pages.statements.title",
      cardTitle: "pages.statements.cardTitle",
      description: "pages.statements.description",
      periodSelector: "pages.statements.periodSelector",
      coverageNote: "pages.statements.coverageNote",
      categoryBreakdown: "pages.statements.categoryBreakdown",
      uncategorisedCount: "pages.statements.uncategorisedCount",
      notTaxAdvice: "pages.statements.notTaxAdvice",
      exportAction: "pages.statements.exportAction",
      emptyGuidance: "pages.statements.emptyGuidance",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
      goalsHeading: "pages.settings.goalsHeading",
      goalDescription: "pages.settings.goalDescription",
      goalThreshold: "pages.settings.goalThreshold",
      goalSustainWindow: "pages.settings.goalSustainWindow",
      goalBuffer: "pages.settings.goalBuffer",
      goalsEmpty: "pages.settings.goalsEmpty",
      defaultBook: "pages.settings.defaultBook",
      currencyDisplay: "pages.settings.currencyDisplay",
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
