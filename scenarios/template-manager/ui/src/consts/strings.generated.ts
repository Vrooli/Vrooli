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
  dashboard: {
    debt: {
      description: "dashboard.debt.description",
      empty: "dashboard.debt.empty",
      title: "dashboard.debt.title",
    },
    drift: {
      description: "dashboard.drift.description",
      empty: "dashboard.drift.empty",
      title: "dashboard.drift.title",
    },
    empty: {
      description: "dashboard.empty.description",
      title: "dashboard.empty.title",
    },
    errorDescription: "dashboard.errorDescription",
    errorTitle: "dashboard.errorTitle",
    loadingDescription: "dashboard.loadingDescription",
    loadingTitle: "dashboard.loadingTitle",
    metrics: {
      deepStreak: "dashboard.metrics.deepStreak",
      openDebt: "dashboard.metrics.openDebt",
      templates: "dashboard.metrics.templates",
      versionLag: "dashboard.metrics.versionLag",
    },
    monitor: {
      description: "dashboard.monitor.description",
      interval: "dashboard.monitor.interval",
      lastRun: "dashboard.monitor.lastRun",
      nextRun: "dashboard.monitor.nextRun",
      status: "dashboard.monitor.status",
      streak: "dashboard.monitor.streak",
      title: "dashboard.monitor.title",
      unscheduled: "dashboard.monitor.unscheduled",
    },
    registry: {
      description: "dashboard.registry.description",
      title: "dashboard.registry.title",
    },
    runs: {
      description: "dashboard.runs.description",
      empty: "dashboard.runs.empty",
      title: "dashboard.runs.title",
    },
    standing: {
      description: "dashboard.standing.description",
      title: "dashboard.standing.title",
    },
  },
  debtDetail: {
    fields: {
      firstSeen: "debtDetail.fields.firstSeen",
      key: "debtDetail.fields.key",
      lastSeen: "debtDetail.fields.lastSeen",
      severity: "debtDetail.fields.severity",
      source: "debtDetail.fields.source",
      status: "debtDetail.fields.status",
      template: "debtDetail.fields.template",
    },
    message: "debtDetail.message",
    messageEmpty: "debtDetail.messageEmpty",
    overview: "debtDetail.overview",
  },
  debtList: {
    empty: "debtList.empty",
    filters: {
      allStatuses: "debtList.filters.allStatuses",
      allTemplates: "debtList.filters.allTemplates",
      status: "debtList.filters.status",
      template: "debtList.filters.template",
    },
    search: "debtList.search",
    searchPlaceholder: "debtList.searchPlaceholder",
    viewEntry: "debtList.viewEntry",
  },
  detail: {
    back: {
      dashboard: "detail.back.dashboard",
      debtList: "detail.back.debtList",
    },
    errorDescription: "detail.errorDescription",
    errorTitle: "detail.errorTitle",
    loadingDescription: "detail.loadingDescription",
    loadingTitle: "detail.loadingTitle",
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
    bottomNavLabel: "layout.bottomNavLabel",
    closeMenuLabel: "layout.closeMenuLabel",
    mainLabel: "layout.mainLabel",
    mobileMenuLabel: "layout.mobileMenuLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      debt: "layout.nav.debt",
      runs: "layout.nav.runs",
      settings: "layout.nav.settings",
      templates: "layout.nav.templates",
    },
    sidebarLabel: "layout.sidebarLabel",
  },
  list: {
    pagination: {
      label: "list.pagination.label",
      next: "list.pagination.next",
      prev: "list.pagination.prev",
      status: "list.pagination.status",
    },
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
    },
    debtList: {
      description: "pages.debtList.description",
      title: "pages.debtList.title",
    },
    runList: {
      description: "pages.runList.description",
      title: "pages.runList.title",
    },
    settings: {
      localeHeading: "pages.settings.localeHeading",
      themeHeading: "pages.settings.themeHeading",
      title: "pages.settings.title",
    },
    templateList: {
      description: "pages.templateList.description",
      title: "pages.templateList.title",
    },
  },
  runDetail: {
    columns: {
      findingCount: "runDetail.columns.findingCount",
      source: "runDetail.columns.source",
    },
    fields: {
      duration: "runDetail.fields.duration",
      finishedAt: "runDetail.fields.finishedAt",
      mode: "runDetail.fields.mode",
      startedAt: "runDetail.fields.startedAt",
      status: "runDetail.fields.status",
      target: "runDetail.fields.target",
      template: "runDetail.fields.template",
      trigger: "runDetail.fields.trigger",
    },
    findings: "runDetail.findings",
    findingsEmpty: "runDetail.findingsEmpty",
    overview: "runDetail.overview",
    phases: "runDetail.phases",
    phasesEmpty: "runDetail.phasesEmpty",
  },
  runList: {
    empty: "runList.empty",
    filters: {
      allStatuses: "runList.filters.allStatuses",
      status: "runList.filters.status",
    },
    search: "runList.search",
    searchPlaceholder: "runList.searchPlaceholder",
    viewEntry: "runList.viewEntry",
  },
  templateDetail: {
    debt: "templateDetail.debt",
    debtEmpty: "templateDetail.debtEmpty",
    drift: "templateDetail.drift",
    driftEmpty: "templateDetail.driftEmpty",
    fields: {
      kind: "templateDetail.fields.kind",
      lag: "templateDetail.fields.lag",
      latestVersion: "templateDetail.fields.latestVersion",
      manifestPath: "templateDetail.fields.manifestPath",
      sourcePath: "templateDetail.fields.sourcePath",
      status: "templateDetail.fields.status",
      tags: "templateDetail.fields.tags",
      updatedAt: "templateDetail.fields.updatedAt",
      version: "templateDetail.fields.version",
    },
    overview: "templateDetail.overview",
    runs: "templateDetail.runs",
    runsEmpty: "templateDetail.runsEmpty",
  },
  templateList: {
    empty: "templateList.empty",
    filters: {
      allKinds: "templateList.filters.allKinds",
      kind: "templateList.filters.kind",
    },
    search: "templateList.search",
    searchPlaceholder: "templateList.searchPlaceholder",
    viewEntry: "templateList.viewEntry",
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
