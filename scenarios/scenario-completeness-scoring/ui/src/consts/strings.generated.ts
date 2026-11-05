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
      settings: "layout.nav.settings",
    },
    mainLabel: "layout.mainLabel",
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
    statusLabel: "health.statusLabel",
    serviceLabel: "health.serviceLabel",
    timestampLabel: "health.timestampLabel",
  },
  notifications: {
    summary: "notifications.summary",
    summary_zero: "notifications.summary_zero",
    summary_one: "notifications.summary_one",
  },
  scoring: {
    form: {
      label: "scoring.form.label",
      placeholder: "scoring.form.placeholder",
      submit: "scoring.form.submit",
    },
    loading: "scoring.loading",
    empty: "scoring.empty",
    calculatedAtLabel: "scoring.calculatedAtLabel",
    trend: {
      title: "scoring.trend.title",
      since: "scoring.trend.since",
      empty: "scoring.trend.empty",
      unknownDate: "scoring.trend.unknownDate",
    },
    fleet: {
      title: "scoring.fleet.title",
      scenario: "scoring.fleet.scenario",
      score: "scoring.fleet.score",
      rung: "scoring.fleet.rung",
      priority: "scoring.fleet.priority",
      calculated: "scoring.fleet.calculated",
      next: "scoring.fleet.next",
      empty: "scoring.fleet.empty",
      unknownDate: "scoring.fleet.unknownDate",
    },
    maturity: {
      title: "scoring.maturity.title",
      ladderClean: "scoring.maturity.ladderClean",
      workingRungLabel: "scoring.maturity.workingRungLabel",
      satisfiedThroughLabel: "scoring.maturity.satisfiedThroughLabel",
      none: "scoring.maturity.none",
      buildLabel: "scoring.maturity.buildLabel",
      buildPassing: "scoring.maturity.buildPassing",
      buildFailing: "scoring.maturity.buildFailing",
      digestLabel: "scoring.maturity.digestLabel",
      digestUnavailable: "scoring.maturity.digestUnavailable",
      dimensionHeader: "scoring.maturity.dimensionHeader",
      errorPlusHeader: "scoring.maturity.errorPlusHeader",
      openHeader: "scoring.maturity.openHeader",
      approximate: "scoring.maturity.approximate",
    },
    composite: {
      title: "scoring.composite.title",
      outOf: "scoring.composite.outOf",
      groupPoints: "scoring.composite.groupPoints",
      metricPoints: "scoring.composite.metricPoints",
    },
    freshness: {
      title: "scoring.freshness.title",
      verdict: {
        fresh: "scoring.freshness.verdict.fresh",
        stale: "scoring.freshness.verdict.stale",
        unknown: "scoring.freshness.verdict.unknown",
      },
      lastRun: "scoring.freshness.lastRun",
      lastPassed: "scoring.freshness.lastPassed",
      unstampedDigest: "scoring.freshness.unstampedDigest",
      neverPassed: "scoring.freshness.neverPassed",
      noEvidence: "scoring.freshness.noEvidence",
      refreshLabel: "scoring.freshness.refreshLabel",
    },
    importance: {
      title: "scoring.importance.title",
      scoreLabel: "scoring.importance.scoreLabel",
      systemRequired: "scoring.importance.systemRequired",
      dependents: "scoring.importance.dependents",
      required: "scoring.importance.required",
      core: "scoring.importance.core",
      coreDistance: "scoring.importance.coreDistance",
      coreUnknown: "scoring.importance.coreUnknown",
      recentActivity: "scoring.importance.recentActivity",
      partial: "scoring.importance.partial",
    },
    recommendations: {
      title: "scoring.recommendations.title",
      impact: "scoring.recommendations.impact",
      priority: {
        high: "scoring.recommendations.priority.high",
        medium: "scoring.recommendations.priority.medium",
        low: "scoring.recommendations.priority.low",
      },
    },
    actionPlan: {
      title: "scoring.actionPlan.title",
      phaseTitle: "scoring.actionPlan.phaseTitle",
      estimated: "scoring.actionPlan.estimated",
      projected: "scoring.actionPlan.projected",
    },
    degradations: {
      title: "scoring.degradations.title",
      line: "scoring.degradations.line",
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
