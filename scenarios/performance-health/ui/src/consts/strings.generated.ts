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
  common: {
    loading: "common.loading",
    errorTitle: "common.errorTitle",
    retry: "common.retry",
  },
  tier: {
    none: "tier.none",
    tier0: "tier.tier0",
    tier1: "tier.tier1",
    unknown: "tier.unknown",
  },
  layout: {
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    mainLabel: "layout.mainLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      audit: "layout.nav.audit",
      trends: "layout.nav.trends",
      fleet: "layout.nav.fleet",
      trace: "layout.nav.trace",
      readiness: "layout.nav.readiness",
      budgets: "layout.nav.budgets",
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
  perf: {
    picker: {
      label: "perf.picker.label",
    },
  },
  pages: {
    dashboard: {
      title: "pages.dashboard.title",
      description: "pages.dashboard.description",
      snapshotTitle: "pages.dashboard.snapshotTitle",
      snapshotError: "pages.dashboard.snapshotError",
      workflowsTitle: "pages.dashboard.workflowsTitle",
      snapshotEmpty: "pages.dashboard.snapshotEmpty",
      snapshotCta: "pages.dashboard.snapshotCta",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
  },
  audit: {
    title: "audit.title",
    description: "audit.description",
    run: "audit.run",
    running: "audit.running",
    tier: {
      title: "audit.tier.title",
      framework: "audit.tier.framework",
      surfaces: "audit.tier.surfaces",
      errorTitle: "audit.tier.errorTitle",
      loadingTitle: "audit.tier.loadingTitle",
    },
    findings: {
      title: "audit.findings.title",
      empty: "audit.findings.empty",
      autofixable: "audit.findings.autofixable",
      readinessLink: "audit.findings.readinessLink",
    },
    result: {
      title: "audit.result.title",
      trace: "audit.result.trace",
      webVitals: "audit.result.webVitals",
    },
    outcome: {
      captured: "audit.outcome.captured",
      skipped: "audit.outcome.skipped",
      failed: "audit.outcome.failed",
      unknown: "audit.outcome.unknown",
    },
  },
  trends: {
    title: "trends.title",
    description: "trends.description",
    empty: "trends.empty",
    emptyTitle: "trends.emptyTitle",
    emptyCta: "trends.emptyCta",
    errorTitle: "trends.errorTitle",
    loadingTitle: "trends.loadingTitle",
    sparklineLabel: "trends.sparklineLabel",
    samplesTitle: "trends.samplesTitle",
    metric: {
      goBuild: "trends.metric.goBuild",
      uiBuild: "trends.metric.uiBuild",
      bundle: "trends.metric.bundle",
      lcp: "trends.metric.lcp",
      component: "trends.metric.component",
      startup: "trends.metric.startup",
    },
    col: {
      captured: "trends.col.captured",
      note: "trends.col.note",
    },
  },
  fleet: {
    title: "fleet.title",
    description: "fleet.description",
    refresh: "fleet.refresh",
    empty: "fleet.empty",
    emptyTitle: "fleet.emptyTitle",
    errorTitle: "fleet.errorTitle",
    loadingTitle: "fleet.loadingTitle",
    summary: {
      scenarios: "fleet.summary.scenarios",
      noBudget: "fleet.summary.noBudget",
      regressed: "fleet.summary.regressed",
    },
    tiers: {
      title: "fleet.tiers.title",
    },
    slowest: {
      title: "fleet.slowest.title",
      empty: "fleet.slowest.empty",
    },
    regressedSection: {
      title: "fleet.regressedSection.title",
      empty: "fleet.regressedSection.empty",
      flagged: "fleet.regressedSection.flagged",
    },
    noBudgetSection: {
      title: "fleet.noBudgetSection.title",
      empty: "fleet.noBudgetSection.empty",
    },
    errors: {
      title: "fleet.errors.title",
    },
    col: {
      scenario: "fleet.col.scenario",
      tier: "fleet.col.tier",
      goBuild: "fleet.col.goBuild",
      uiBuild: "fleet.col.uiBuild",
      reason: "fleet.col.reason",
    },
  },
  trace: {
    title: "trace.title",
    description: "trace.description",
    artifactLabel: "trace.artifactLabel",
    artifactPlaceholder: "trace.artifactPlaceholder",
    analyze: "trace.analyze",
    empty: "trace.empty",
    emptyTitle: "trace.emptyTitle",
    errorTitle: "trace.errorTitle",
    loadingTitle: "trace.loadingTitle",
    devtoolsHint: "trace.devtoolsHint",
    componentsTitle: "trace.componentsTitle",
    componentsEmpty: "trace.componentsEmpty",
    findingsTitle: "trace.findingsTitle",
    findingsEmpty: "trace.findingsEmpty",
    vital: {
      lcp: "trace.vital.lcp",
      fcp: "trace.vital.fcp",
      longTask: "trace.vital.longTask",
      components: "trace.vital.components",
    },
    col: {
      component: "trace.col.component",
      commits: "trace.col.commits",
      avg: "trace.col.avg",
      max: "trace.col.max",
      definition: "trace.col.definition",
    },
  },
  readiness: {
    title: "readiness.title",
    description: "readiness.description",
    framework: "readiness.framework",
    errorTitle: "readiness.errorTitle",
    loadingTitle: "readiness.loadingTitle",
    autofixableCount: "readiness.autofixableCount",
    autofixableCount_one: "readiness.autofixableCount_one",
    autofixableCount_zero: "readiness.autofixableCount_zero",
    preview: "readiness.preview",
    apply: "readiness.apply",
    applying: "readiness.applying",
    gaps: {
      title: "readiness.gaps.title",
      empty: "readiness.gaps.empty",
      autofixable: "readiness.gaps.autofixable",
    },
    fixResult: {
      title: "readiness.fixResult.title",
      applied: "readiness.fixResult.applied",
      previewed: "readiness.fixResult.previewed",
    },
  },
  budgets: {
    title: "budgets.title",
    description: "budgets.description",
    errorTitle: "budgets.errorTitle",
    loadingTitle: "budgets.loadingTitle",
    notDeclared: "budgets.notDeclared",
    unsetPlaceholder: "budgets.unsetPlaceholder",
    ratchet: "budgets.ratchet",
    save: "budgets.save",
    saved: "budgets.saved",
    check: "budgets.check",
    field: {
      goBuild: "budgets.field.goBuild",
      uiBuild: "budgets.field.uiBuild",
      bundle: "budgets.field.bundle",
      lcp: "budgets.field.lcp",
      startup: "budgets.field.startup",
    },
    checkResult: {
      title: "budgets.checkResult.title",
      passed: "budgets.checkResult.passed",
      failed: "budgets.checkResult.failed",
      col: {
        axis: "budgets.checkResult.col.axis",
        measured: "budgets.checkResult.col.measured",
        budget: "budgets.checkResult.col.budget",
      },
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
