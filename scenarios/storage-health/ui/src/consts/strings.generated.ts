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
      fleet: "layout.nav.fleet",
      validate: "layout.nav.validate",
      advisor: "layout.nav.advisor",
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
  common: {
    loading: "common.loading",
    errorTitle: "common.errorTitle",
    retry: "common.retry",
    scanNow: "common.scanNow",
    scanning: "common.scanning",
    useLastSnapshot: "common.useLastSnapshot",
  },
  isolation: {
    ready: "isolation.ready",
    unready: "isolation.unready",
    readyLabel: "isolation.readyLabel",
    unreadyLabel: "isolation.unreadyLabel",
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
  dashboard: {
    stat: {
      scenarios: "dashboard.stat.scenarios",
      isolationUnready: "dashboard.stat.isolationUnready",
      noBackup: "dashboard.stat.noBackup",
      findings: "dashboard.stat.findings",
    },
    scorecard: {
      title: "dashboard.scorecard.title",
      viewAll: "dashboard.scorecard.viewAll",
      empty: "dashboard.scorecard.empty",
    },
    engines: {
      title: "dashboard.engines.title",
      empty: "dashboard.engines.empty",
    },
    freshness: {
      title: "dashboard.freshness.title",
      scannedAt: "dashboard.freshness.scannedAt",
      never: "dashboard.freshness.never",
    },
    empty: {
      title: "dashboard.empty.title",
      message: "dashboard.empty.message",
      action: "dashboard.empty.action",
    },
    loadingTitle: "dashboard.loadingTitle",
  },
  fleet: {
    title: "fleet.title",
    description: "fleet.description",
    loadingTitle: "fleet.loadingTitle",
    errorTitle: "fleet.errorTitle",
    view: {
      label: "fleet.view.label",
      all: "fleet.view.all",
      isolation: "fleet.view.isolation",
      noBackup: "fleet.view.noBackup",
      engines: "fleet.view.engines",
      stages: "fleet.view.stages",
    },
    source: {
      scan: "fleet.source.scan",
      snapshot: "fleet.source.snapshot",
    },
    col: {
      scenario: "fleet.col.scenario",
      engines: "fleet.col.engines",
      stage: "fleet.col.stage",
      isolation: "fleet.col.isolation",
      findings: "fleet.col.findings",
      backup: "fleet.col.backup",
      count: "fleet.col.count",
    },
    backup: {
      present: "fleet.backup.present",
      missing: "fleet.backup.missing",
    },
    empty: {
      all: "fleet.empty.all",
      isolation: "fleet.empty.isolation",
      noBackup: "fleet.empty.noBackup",
      engines: "fleet.empty.engines",
      stages: "fleet.empty.stages",
    },
    errors: {
      title: "fleet.errors.title",
    },
    howToFix: "fleet.howToFix",
  },
  validate: {
    title: "validate.title",
    description: "validate.description",
    input: {
      label: "validate.input.label",
      placeholder: "validate.input.placeholder",
      run: "validate.input.run",
    },
    loadingTitle: "validate.loadingTitle",
    errorTitle: "validate.errorTitle",
    prompt: {
      title: "validate.prompt.title",
      message: "validate.prompt.message",
    },
    status: {
      passed: "validate.status.passed",
      failed: "validate.status.failed",
      degraded: "validate.status.degraded",
      error: "validate.status.error",
      skipped: "validate.status.skipped",
      unspecified: "validate.status.unspecified",
    },
    counts: {
      errors: "validate.counts.errors",
      warnings: "validate.counts.warnings",
      infos: "validate.counts.infos",
    },
    findings: {
      title: "validate.findings.title",
      remediationLabel: "validate.findings.remediationLabel",
      autofix: "validate.findings.autofix",
    },
    severity: {
      error: "validate.severity.error",
      warning: "validate.severity.warning",
      info: "validate.severity.info",
    },
    clean: {
      title: "validate.clean.title",
      message: "validate.clean.message",
    },
    fix: {
      preview: "validate.fix.preview",
      apply: "validate.fix.apply",
      applying: "validate.fix.applying",
      previewing: "validate.fix.previewing",
      candidatesTitle: "validate.fix.candidatesTitle",
      confirm: "validate.fix.confirm",
      applied: "validate.fix.applied",
      noop: "validate.fix.noop",
    },
  },
  advisor: {
    title: "advisor.title",
    description: "advisor.description",
    tab: {
      engines: "advisor.tab.engines",
      migrations: "advisor.tab.migrations",
    },
    loadingTitle: "advisor.loadingTitle",
    errorTitle: "advisor.errorTitle",
    engines: {
      fitnessLabel: "advisor.engines.fitnessLabel",
      currentLabel: "advisor.engines.currentLabel",
      recommendedLabel: "advisor.engines.recommendedLabel",
      rationaleLabel: "advisor.engines.rationaleLabel",
      blockersLabel: "advisor.engines.blockersLabel",
      empty: {
        title: "advisor.engines.empty.title",
        message: "advisor.engines.empty.message",
      },
    },
    migrations: {
      summary: "advisor.migrations.summary",
      debtLabel: "advisor.migrations.debtLabel",
      notesLabel: "advisor.migrations.notesLabel",
      empty: {
        title: "advisor.migrations.empty.title",
        message: "advisor.migrations.empty.message",
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
