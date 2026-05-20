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
    brand: "app.brand",
    brandInitials: "app.brandInitials",
    eyebrow: "app.eyebrow",
  },
  layout: {
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    mobileHeaderLabel: "layout.mobileHeaderLabel",
    drawerLabel: "layout.drawerLabel",
    openDrawer: "layout.openDrawer",
    closeDrawer: "layout.closeDrawer",
    skipToContent: "layout.skipToContent",
    main: "layout.main",
    nav: {
      dashboard: "layout.nav.dashboard",
      validation: "layout.nav.validation",
      search: "layout.nav.search",
      inventory: "layout.nav.inventory",
      reindex: "layout.nav.reindex",
      settings: "layout.nav.settings",
    },
  },
  route: {
    loading: "route.loading",
  },
  theme: {
    switcherLabel: "theme.switcherLabel",
    light: "theme.light",
    dark: "theme.dark",
    system: "theme.system",
    choice: {
      light: "theme.choice.light",
      dark: "theme.choice.dark",
      system: "theme.choice.system",
    },
  },
  inspector: {
    close: "inspector.close",
  },
  pages: {
    dashboard: {
      title: "pages.dashboard.title",
      description: "pages.dashboard.description",
      stats: {
        scenariosValidated: "pages.dashboard.stats.scenariosValidated",
        surfacesIndexed: "pages.dashboard.stats.surfacesIndexed",
        openIssues: "pages.dashboard.stats.openIssues",
      },
      activity: {
        heading: "pages.dashboard.activity.heading",
        empty: "pages.dashboard.activity.empty",
      },
      quickActions: {
        heading: "pages.dashboard.quickActions.heading",
        search: "pages.dashboard.quickActions.search",
        validate: "pages.dashboard.quickActions.validate",
        reindex: "pages.dashboard.quickActions.reindex",
      },
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    validation: {
      title: "pages.validation.title",
      description: "pages.validation.description",
      form: {
        heading: "pages.validation.form.heading",
        scenarioLabel: "pages.validation.form.scenarioLabel",
        scenarioPlaceholder: "pages.validation.form.scenarioPlaceholder",
        scenarioHelp: "pages.validation.form.scenarioHelp",
        submit: "pages.validation.form.submit",
        submitting: "pages.validation.form.submitting",
      },
      recent: {
        heading: "pages.validation.recent.heading",
        empty: "pages.validation.recent.empty",
        columns: {
          scenario: "pages.validation.recent.columns.scenario",
          status: "pages.validation.recent.columns.status",
          errors: "pages.validation.recent.columns.errors",
          warnings: "pages.validation.recent.columns.warnings",
          ranAt: "pages.validation.recent.columns.ranAt",
        },
        clear: "pages.validation.recent.clear",
        open: "pages.validation.recent.open",
      },
      status: {
        passed: "pages.validation.status.passed",
        failed: "pages.validation.status.failed",
      },
      summary: {
        heading: "pages.validation.summary.heading",
        errors: "pages.validation.summary.errors",
        warnings: "pages.validation.summary.warnings",
        infos: "pages.validation.summary.infos",
        ranAt: "pages.validation.summary.ranAt",
      },
      filters: {
        heading: "pages.validation.filters.heading",
        all: "pages.validation.filters.all",
        error: "pages.validation.filters.error",
        warning: "pages.validation.filters.warning",
        info: "pages.validation.filters.info",
      },
      findings: {
        heading: "pages.validation.findings.heading",
        empty: "pages.validation.findings.empty",
        noneForFilter: "pages.validation.findings.noneForFilter",
        code: "pages.validation.findings.code",
        location: "pages.validation.findings.location",
        suggestion: "pages.validation.findings.suggestion",
      },
      loading: "pages.validation.loading",
      error: "pages.validation.error",
      revalidate: "pages.validation.revalidate",
      back: "pages.validation.back",
    },
    search: {
      title: "pages.search.title",
      description: "pages.search.description",
      placeholder: "pages.search.placeholder",
      empty: "pages.search.empty",
    },
    inventory: {
      title: "pages.inventory.title",
      description: "pages.inventory.description",
      empty: "pages.inventory.empty",
      detail: {
        title: "pages.inventory.detail.title",
      },
    },
    reindex: {
      title: "pages.reindex.title",
      description: "pages.reindex.description",
      trigger: "pages.reindex.trigger",
      empty: "pages.reindex.empty",
      job: {
        title: "pages.reindex.job.title",
      },
    },
    notFound: {
      title: "pages.notFound.title",
      description: "pages.notFound.description",
      back: "pages.notFound.back",
    },
  },
  health: {
    pill: {
      ok: "health.pill.ok",
      error: "health.pill.error",
      loading: "health.pill.loading",
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
