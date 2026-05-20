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
  common: {
    code: {
      copy: "common.code.copy",
      copied: "common.code.copied",
      copyShort: "common.code.copyShort",
    },
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
      input: {
        label: "pages.search.input.label",
        clear: "pages.search.input.clear",
      },
      filters: {
        heading: "pages.search.filters.heading",
      },
      kind: {
        all: "pages.search.kind.all",
        component: "pages.search.kind.component",
        page: "pages.search.kind.page",
        feature: "pages.search.kind.feature",
        hook: "pages.search.kind.hook",
        layout: "pages.search.kind.layout",
        other: "pages.search.kind.other",
        unspecified: "pages.search.kind.unspecified",
      },
      provenance: {
        custom: "pages.search.provenance.custom",
        adoptedUnmodified: "pages.search.provenance.adoptedUnmodified",
        adoptedModified: "pages.search.provenance.adoptedModified",
        unknown: "pages.search.provenance.unknown",
        unspecified: "pages.search.provenance.unspecified",
      },
      loading: "pages.search.loading",
      error: "pages.search.error",
      shortQuery: "pages.search.shortQuery",
      empty: {
        title: "pages.search.empty.title",
        description: "pages.search.empty.description",
      },
      noResults: {
        title: "pages.search.noResults.title",
        description: "pages.search.noResults.description",
      },
      noResultsForFilter: "pages.search.noResultsForFilter",
      results: {
        summary: "pages.search.results.summary",
        listLabel: "pages.search.results.listLabel",
        score: "pages.search.results.score",
        path: "pages.search.results.path",
        openInInventory: "pages.search.results.openInInventory",
      },
    },
    inventory: {
      title: "pages.inventory.title",
      description: "pages.inventory.description",
      form: {
        heading: "pages.inventory.form.heading",
        scenarioLabel: "pages.inventory.form.scenarioLabel",
        scenarioPlaceholder: "pages.inventory.form.scenarioPlaceholder",
        scenarioHelp: "pages.inventory.form.scenarioHelp",
        submit: "pages.inventory.form.submit",
        submitting: "pages.inventory.form.submitting",
      },
      filters: {
        heading: "pages.inventory.filters.heading",
      },
      loading: "pages.inventory.loading",
      error: "pages.inventory.error",
      empty: {
        title: "pages.inventory.empty.title",
        description: "pages.inventory.empty.description",
      },
      noSurfaces: {
        title: "pages.inventory.noSurfaces.title",
        description: "pages.inventory.noSurfaces.description",
      },
      noResultsForFilter: "pages.inventory.noResultsForFilter",
      summary: "pages.inventory.summary",
      columns: {
        displayName: "pages.inventory.columns.displayName",
        kind: "pages.inventory.columns.kind",
        slot: "pages.inventory.columns.slot",
        filePath: "pages.inventory.columns.filePath",
      },
      detail: {
        title: "pages.inventory.detail.title",
        back: "pages.inventory.detail.back",
        scenario: "pages.inventory.detail.scenario",
        slot: "pages.inventory.detail.slot",
        kind: "pages.inventory.detail.kind",
        filePath: "pages.inventory.detail.filePath",
        description: "pages.inventory.detail.description",
        provenance: {
          heading: "pages.inventory.detail.provenance.heading",
          empty: "pages.inventory.detail.provenance.empty",
          library: "pages.inventory.detail.provenance.library",
          libraryVersion: "pages.inventory.detail.provenance.libraryVersion",
          adoptionId: "pages.inventory.detail.provenance.adoptionId",
        },
        widgets: {
          heading: "pages.inventory.detail.widgets.heading",
          empty: "pages.inventory.detail.widgets.empty",
          propsSchema: "pages.inventory.detail.widgets.propsSchema",
        },
        loading: "pages.inventory.detail.loading",
        error: "pages.inventory.detail.error",
        notFound: {
          title: "pages.inventory.detail.notFound.title",
          description: "pages.inventory.detail.notFound.description",
        },
      },
    },
    reindex: {
      title: "pages.reindex.title",
      description: "pages.reindex.description",
      form: {
        heading: "pages.reindex.form.heading",
        scenarioLabel: "pages.reindex.form.scenarioLabel",
        scenarioPlaceholder: "pages.reindex.form.scenarioPlaceholder",
        scenarioHelp: "pages.reindex.form.scenarioHelp",
        dryRunLabel: "pages.reindex.form.dryRunLabel",
        dryRunHelp: "pages.reindex.form.dryRunHelp",
        submit: "pages.reindex.form.submit",
        submitting: "pages.reindex.form.submitting",
      },
      error: "pages.reindex.error",
      confirm: {
        title: "pages.reindex.confirm.title",
        description: "pages.reindex.confirm.description",
        accept: "pages.reindex.confirm.accept",
        cancel: "pages.reindex.confirm.cancel",
        close: "pages.reindex.confirm.close",
        backdrop: "pages.reindex.confirm.backdrop",
      },
      jobs: {
        heading: "pages.reindex.jobs.heading",
        emptyTitle: "pages.reindex.jobs.emptyTitle",
        emptyDescription: "pages.reindex.jobs.emptyDescription",
        clearTerminal: "pages.reindex.jobs.clearTerminal",
        columns: {
          progress: "pages.reindex.jobs.columns.progress",
          dryRun: "pages.reindex.jobs.columns.dryRun",
        },
        open: "pages.reindex.jobs.open",
        allScenarios: "pages.reindex.jobs.allScenarios",
      },
      state: {
        queued: "pages.reindex.state.queued",
        running: "pages.reindex.state.running",
        succeeded: "pages.reindex.state.succeeded",
        failed: "pages.reindex.state.failed",
        cancelled: "pages.reindex.state.cancelled",
        unknown: "pages.reindex.state.unknown",
      },
      job: {
        title: "pages.reindex.job.title",
        back: "pages.reindex.job.back",
        meta: {
          scenario: "pages.reindex.job.meta.scenario",
          state: "pages.reindex.job.meta.state",
          triggeredAt: "pages.reindex.job.meta.triggeredAt",
          dryRun: "pages.reindex.job.meta.dryRun",
          processed: "pages.reindex.job.meta.processed",
          total: "pages.reindex.job.meta.total",
        },
        progress: "pages.reindex.job.progress",
        error: "pages.reindex.job.error",
        cancel: "pages.reindex.job.cancel",
        cancelling: "pages.reindex.job.cancelling",
        loading: "pages.reindex.job.loading",
        loadError: "pages.reindex.job.loadError",
        notFound: {
          title: "pages.reindex.job.notFound.title",
          description: "pages.reindex.job.notFound.description",
        },
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
