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
    statusLabel: "health.statusLabel",
    serviceLabel: "health.serviceLabel",
    timestampLabel: "health.timestampLabel",
  },
  notifications: {
    summary: "notifications.summary",
    summary_zero: "notifications.summary_zero",
    summary_one: "notifications.summary_one",
  },
  validation: {
    title: "validation.title",
    scenarioLabel: "validation.scenarioLabel",
    scenarioPlaceholder: "validation.scenarioPlaceholder",
    run: "validation.run",
    running: "validation.running",
    loading: "validation.loading",
    idle: "validation.idle",
    status: "validation.status",
    maturity: "validation.maturity",
    findings: "validation.findings",
    surfaces: "validation.surfaces",
    runId: "validation.runId",
    targetPath: "validation.targetPath",
    countSummary: "validation.countSummary",
    unknown: "validation.unknown",
    findingsTitle: "validation.findingsTitle",
    noFindings: "validation.noFindings",
    maturitySummaryTitle: "validation.maturitySummaryTitle",
    currentLevelLabel: "validation.currentLevelLabel",
    nextLevelLabel: "validation.nextLevelLabel",
    nextLevelBlockers: "validation.nextLevelBlockers",
    exitCriteriaTitle: "validation.exitCriteriaTitle",
    noBlockers: "validation.noBlockers",
    noNextLevel: "validation.noNextLevel",
    testPlanTitle: "validation.testPlanTitle",
    testPlanEmpty: "validation.testPlanEmpty",
    colWorkspace: "validation.colWorkspace",
    colLanguage: "validation.colLanguage",
    colFramework: "validation.colFramework",
    colTestCommand: "validation.colTestCommand",
    colCoverageCommand: "validation.colCoverageCommand",
    colTimeout: "validation.colTimeout",
    noncanonicalFramework: "validation.noncanonicalFramework",
    timeoutSeconds: "validation.timeoutSeconds",
    executionTitle: "validation.executionTitle",
    executionEmpty: "validation.executionEmpty",
    colStatus: "validation.colStatus",
    colExitCode: "validation.colExitCode",
    durationMs: "validation.durationMs",
    toggleOutput: "validation.toggleOutput",
    stdoutLabel: "validation.stdoutLabel",
    stderrLabel: "validation.stderrLabel",
    failureReasonLabel: "validation.failureReasonLabel",
    noOutput: "validation.noOutput",
    coverageTitle: "validation.coverageTitle",
    coverageEmpty: "validation.coverageEmpty",
    coverageSurfaceRollup: "validation.coverageSurfaceRollup",
    colFile: "validation.colFile",
    colCovered: "validation.colCovered",
    colPercent: "validation.colPercent",
    colThreshold: "validation.colThreshold",
    findingExpected: "validation.findingExpected",
    findingObserved: "validation.findingObserved",
    findingWhy: "validation.findingWhy",
    findingRemediation: "validation.findingRemediation",
    diagnosticsTitle: "validation.diagnosticsTitle",
    diagnosticsEmpty: "validation.diagnosticsEmpty",
    globalImpactTitle: "validation.globalImpactTitle",
    globalImpactEmpty: "validation.globalImpactEmpty",
    recommendedSkillsTitle: "validation.recommendedSkillsTitle",
    recommendedSkillsEmpty: "validation.recommendedSkillsEmpty",
    nextStepsTitle: "validation.nextStepsTitle",
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
