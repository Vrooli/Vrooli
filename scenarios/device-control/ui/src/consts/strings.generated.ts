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
      dashboard: "layout.nav.dashboard",
      flows: "layout.nav.flows",
      evidence: "layout.nav.evidence",
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
      onboardingReport: "pages.dashboard.onboardingReport",
      onboardingProbeComplete: "pages.dashboard.onboardingProbeComplete",
      devicesAvailable: "pages.dashboard.devicesAvailable",
      unavailablePrerequisites: "pages.dashboard.unavailablePrerequisites",
      liveLeases: "pages.dashboard.liveLeases",
      fleetSnapshot: "pages.dashboard.fleetSnapshot",
      noDevices: "pages.dashboard.noDevices",
      reprobe: "pages.dashboard.reprobe",
      acquireLease: "pages.dashboard.acquireLease",
      probePassed: "pages.dashboard.probePassed",
      osUnavailable: "pages.dashboard.osUnavailable",
      localTransport: "pages.dashboard.localTransport",
      strategyMatrix: "pages.dashboard.strategyMatrix",
      status: "pages.dashboard.status",
      tiers: "pages.dashboard.tiers",
      unknown: "pages.dashboard.unknown",
      promotable: "pages.dashboard.promotable",
      yes: "pages.dashboard.yes",
      no: "pages.dashboard.no",
      liveSessionControls: "pages.dashboard.liveSessionControls",
      noLiveSessions: "pages.dashboard.noLiveSessions",
      leaseExpires: "pages.dashboard.leaseExpires",
      killImmediately: "pages.dashboard.killImmediately",
      killSession: "pages.dashboard.killSession",
      leaseFailed: "pages.dashboard.leaseFailed",
      onboardingFailed: "pages.dashboard.onboardingFailed",
    },
    flows: {
      title: "pages.flows.title",
      acquireAndRun: "pages.flows.acquireAndRun",
      running: "pages.flows.running",
      liveSession: "pages.flows.liveSession",
      killAvailable: "pages.flows.killAvailable",
      killActiveSession: "pages.flows.killActiveSession",
      description: "pages.flows.description",
      flowDefinition: "pages.flows.flowDefinition",
      strategy: "pages.flows.strategy",
      device: "pages.flows.device",
      json: "pages.flows.json",
      validate: "pages.flows.validate",
      capabilityGapReport: "pages.flows.capabilityGapReport",
      runnable: "pages.flows.runnable",
      blockedBeforeExecution: "pages.flows.blockedBeforeExecution",
      noValidation: "pages.flows.noValidation",
      runReview: "pages.flows.runReview",
      noRun: "pages.flows.noRun",
      retainedEvidence: "pages.flows.retainedEvidence",
      checksumUnavailable: "pages.flows.checksumUnavailable",
      bytes: "pages.flows.bytes",
      redactionVerified: "pages.flows.redactionVerified",
      retainedEvidenceAlt: "pages.flows.retainedEvidenceAlt",
      invalidFlow: "pages.flows.invalidFlow",
      flowFailed: "pages.flows.flowFailed",
      sessionKillFailed: "pages.flows.sessionKillFailed",
    },
    evidence: {
      title: "pages.evidence.title",
      description: "pages.evidence.description",
      recentVerbs: "pages.evidence.recentVerbs",
      noVerbs: "pages.evidence.noVerbs",
      verb: "pages.evidence.verb",
      device: "pages.evidence.device",
      actor: "pages.evidence.actor",
      outcome: "pages.evidence.outcome",
      created: "pages.evidence.created",
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
} as const;

export type Strings = typeof strings;
