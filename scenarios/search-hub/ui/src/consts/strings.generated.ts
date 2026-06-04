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
      search: "layout.nav.search",
      evals: "layout.nav.evals",
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
      statPlaceholderLabel: "pages.dashboard.statPlaceholderLabel",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    search: {
      title: "pages.search.title",
      description: "pages.search.description",
    },
    evals: {
      title: "pages.evals.title",
      description: "pages.evals.description",
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
  search: {
    queryLabel: "search.queryLabel",
    queryPlaceholder: "search.queryPlaceholder",
    submit: "search.submit",
    searching: "search.searching",
    expandLabel: "search.expandLabel",
    typesLabel: "search.typesLabel",
    emptyState: "search.emptyState",
    noResults: "search.noResults",
    error: "search.error",
    summary: "search.summary",
    rankedHeading: "search.rankedHeading",
    groupedHeading: "search.groupedHeading",
    routingHeading: "search.routingHeading",
    reranked: "search.reranked",
    grouped: "search.grouped",
    degraded: "search.degraded",
    groupEmpty: "search.groupEmpty",
    score: "search.score",
    rerank: "search.rerank",
    provenance: "search.provenance",
    statusHeading: "search.statusHeading",
    classifier: "search.classifier",
    reranker: "search.reranker",
    available: "search.available",
    unavailable: "search.unavailable",
    statusError: "search.statusError",
    providersReachable: "search.providersReachable",
  },
  evals: {
    suitesHeading: "evals.suitesHeading",
    loading: "evals.loading",
    error: "evals.error",
    noSuites: "evals.noSuites",
    suiteMeta: "evals.suiteMeta",
    selectSuite: "evals.selectSuite",
    runsHeading: "evals.runsHeading",
    noRuns: "evals.noRuns",
    trendHeading: "evals.trendHeading",
    trendStrong: "evals.trendStrong",
    trendGibberish: "evals.trendGibberish",
    runMet: "evals.runMet",
    runConfig: "evals.runConfig",
    casesHeading: "evals.casesHeading",
    colCase: "evals.colCase",
    colOutcome: "evals.colOutcome",
    colTop: "evals.colTop",
    colRank: "evals.colRank",
    compareHint: "evals.compareHint",
    compareButton: "evals.compareButton",
    compareHeading: "evals.compareHeading",
    compareLegend: "evals.compareLegend",
    compareColA: "evals.compareColA",
    compareColB: "evals.compareColB",
    colDelta: "evals.colDelta",
    viewRun: "evals.viewRun",
    hideRun: "evals.hideRun",
    tagLabel: "evals.tagLabel",
    untagged: "evals.untagged",
  },
} as const;

export type Strings = typeof strings;
