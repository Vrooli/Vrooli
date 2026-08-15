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
      offers: "layout.nav.offers",
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
      priorityBoard: "pages.dashboard.priorityBoard",
      offerRecords: "pages.dashboard.offerRecords",
      firedTriggers: "pages.dashboard.firedTriggers",
      ledgerPosture: "pages.dashboard.ledgerPosture",
      runwayMonths: "pages.dashboard.runwayMonths",
      postureUnavailable: "pages.dashboard.postureUnavailable",
      postureBasis: "pages.dashboard.postureBasis",
      earningNothing: "pages.dashboard.earningNothing",
      emptyGuidance: "pages.dashboard.emptyGuidance",
      sourceUnavailable: "pages.dashboard.sourceUnavailable",
      sourceUnavailableReason: "pages.dashboard.sourceUnavailableReason",
      boardUnavailable: "pages.dashboard.boardUnavailable",
    },
    offers: {
      title: "pages.offers.title",
      cardTitle: "pages.offers.cardTitle",
      description: "pages.offers.description",
      statusLabel: "pages.offers.statusLabel",
      waitingOn: "pages.offers.waitingOn",
      legalTransitions: "pages.offers.legalTransitions",
      refusalReason: "pages.offers.refusalReason",
      refusalRemedy: "pages.offers.refusalRemedy",
      auditTrail: "pages.offers.auditTrail",
      promoteAction: "pages.offers.promoteAction",
      roleRequirement: "pages.offers.roleRequirement",
      membershipFinding: "pages.offers.membershipFinding",
      emptyGuidance: "pages.offers.emptyGuidance",
    },
    triggers: {
      title: "pages.triggers.title",
      cardTitle: "pages.triggers.cardTitle",
      description: "pages.triggers.description",
      parseReady: "pages.triggers.parseReady",
      parseError: "pages.triggers.parseError",
      parseErrorDetail: "pages.triggers.parseErrorDetail",
      dryRunAction: "pages.triggers.dryRunAction",
      dryRunVerdict: "pages.triggers.dryRunVerdict",
      dryRunUnknownVerdict: "pages.triggers.dryRunUnknownVerdict",
      dryRunUnsatisfiedVerdict: "pages.triggers.dryRunUnsatisfiedVerdict",
      factTrace: "pages.triggers.factTrace",
      missingFact: "pages.triggers.missingFact",
      factRegistry: "pages.triggers.factRegistry",
      evaluationFreshness: "pages.triggers.evaluationFreshness",
      stalledAlert: "pages.triggers.stalledAlert",
      emptyGuidance: "pages.triggers.emptyGuidance",
    },
    proposals: {
      title: "pages.proposals.title",
      cardTitle: "pages.proposals.cardTitle",
      description: "pages.proposals.description",
      proposer: "pages.proposals.proposer",
      evidence: "pages.proposals.evidence",
      effect: "pages.proposals.effect",
      declineHistory: "pages.proposals.declineHistory",
      acceptAction: "pages.proposals.acceptAction",
      operatorOnly: "pages.proposals.operatorOnly",
      emptyGuidance: "pages.proposals.emptyGuidance",
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
