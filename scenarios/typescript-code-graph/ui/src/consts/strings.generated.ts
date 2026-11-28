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
    mainLabel: "layout.mainLabel",
    nav: {
      workbench: "layout.nav.workbench",
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
    workbench: {
      title: "pages.workbench.title",
      description: "pages.workbench.description",
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
  shared: {
    loading: {
      label: "shared.loading.label",
    },
    error: {
      title: "shared.error.title",
      retry: "shared.error.retry",
    },
  },
  workbench: {
    extract: {
      targetLabel: "workbench.extract.targetLabel",
      targetPlaceholder: "workbench.extract.targetPlaceholder",
      projectDirLabel: "workbench.extract.projectDirLabel",
      projectDirPlaceholder: "workbench.extract.projectDirPlaceholder",
      submit: "workbench.extract.submit",
      submitting: "workbench.extract.submitting",
    },
    stats: {
      files: "workbench.stats.files",
      modules: "workbench.stats.modules",
      symbols: "workbench.stats.symbols",
      imports: "workbench.stats.imports",
      warnings: "workbench.stats.warnings",
      hash: "workbench.stats.hash",
      durationLabel: "workbench.stats.durationLabel",
    },
    status: {
      idleTitle: "workbench.status.idleTitle",
      idleDescription: "workbench.status.idleDescription",
      errorTitle: "workbench.status.errorTitle",
    },
    workspaceUnsupported: {
      title: "workbench.workspaceUnsupported.title",
      description: "workbench.workspaceUnsupported.description",
    },
    tabs: {
      label: "workbench.tabs.label",
      graph: "workbench.tabs.graph",
      warnings: "workbench.tabs.warnings",
      rewrite: "workbench.tabs.rewrite",
      fixtures: "workbench.tabs.fixtures",
    },
  },
  sidecar: {
    title: "sidecar.title",
    loading: "sidecar.loading",
    unreachable: "sidecar.unreachable",
    lastMessage: "sidecar.lastMessage",
    status: {
      unspecified: "sidecar.status.unspecified",
      ready: "sidecar.status.ready",
      unhealthy: "sidecar.status.unhealthy",
      restarting: "sidecar.status.restarting",
      permanentlyUnhealthy: "sidecar.status.permanentlyUnhealthy",
    },
  },
  explorer: {
    canvas: {
      ariaLabel: "explorer.canvas.ariaLabel",
      summaryPackages: "explorer.canvas.summaryPackages",
      summaryPackages_one: "explorer.canvas.summaryPackages_one",
      summaryImports: "explorer.canvas.summaryImports",
      summaryImports_one: "explorer.canvas.summaryImports_one",
      nodeAriaLabel: "explorer.canvas.nodeAriaLabel",
      nodeCyclePresent: "explorer.canvas.nodeCyclePresent",
      nodeCycleAbsent: "explorer.canvas.nodeCycleAbsent",
    },
    accessibleList: {
      title: "explorer.accessibleList.title",
      description: "explorer.accessibleList.description",
      importsLabel: "explorer.accessibleList.importsLabel",
    },
    emptyTitle: "explorer.emptyTitle",
    emptyDescription: "explorer.emptyDescription",
    legend: {
      title: "explorer.legend.title",
      package: "explorer.legend.package",
      cycle: "explorer.legend.cycle",
    },
    filterBar: {
      label: "explorer.filterBar.label",
      all: "explorer.filterBar.all",
      none: "explorer.filterBar.none",
    },
    cycleBanner: {
      title: "explorer.cycleBanner.title",
      message: "explorer.cycleBanner.message",
      message_one: "explorer.cycleBanner.message_one",
    },
    files: {
      title: "explorer.files.title",
      empty: "explorer.files.empty",
      symbolCount: "explorer.files.symbolCount",
      symbolCount_one: "explorer.files.symbolCount_one",
      symbolCount_zero: "explorer.files.symbolCount_zero",
    },
    symbols: {
      title: "explorer.symbols.title",
      selectFile: "explorer.symbols.selectFile",
      empty: "explorer.symbols.empty",
      exported: "explorer.symbols.exported",
      unexported: "explorer.symbols.unexported",
      jsdocLabel: "explorer.symbols.jsdocLabel",
    },
    kind: {
      component: "explorer.kind.component",
      hook: "explorer.kind.hook",
      class: "explorer.kind.class",
      interface: "explorer.kind.interface",
      type: "explorer.kind.type",
      function: "explorer.kind.function",
      var: "explorer.kind.var",
      const: "explorer.kind.const",
      reExport: "explorer.kind.reExport",
      unknown: "explorer.kind.unknown",
    },
  },
  warnings: {
    title: "warnings.title",
    description: "warnings.description",
    empty: "warnings.empty",
    fileLabel: "warnings.fileLabel",
    projectLevel: "warnings.projectLevel",
    count: "warnings.count",
    count_zero: "warnings.count_zero",
    count_one: "warnings.count_one",
    kind: {
      parse_error: "warnings.kind.parse_error",
      unresolved_import: "warnings.kind.unresolved_import",
      type_check_failure: "warnings.kind.type_check_failure",
      ambiguous_declaration: "warnings.kind.ambiguous_declaration",
      unspecified: "warnings.kind.unspecified",
    },
  },
  rewrite: {
    title: "rewrite.title",
    description: "rewrite.description",
    ops: {
      title: "rewrite.ops.title",
      empty: "rewrite.ops.empty",
      addFileMove: "rewrite.ops.addFileMove",
      addImportRewrite: "rewrite.ops.addImportRewrite",
      fileMove: "rewrite.ops.fileMove",
      importRewrite: "rewrite.ops.importRewrite",
      fromPath: "rewrite.ops.fromPath",
      toPath: "rewrite.ops.toPath",
      oldImport: "rewrite.ops.oldImport",
      newImport: "rewrite.ops.newImport",
      remove: "rewrite.ops.remove",
    },
    plan: {
      button: "rewrite.plan.button",
      planning: "rewrite.plan.planning",
      title: "rewrite.plan.title",
      planIdLabel: "rewrite.plan.planIdLabel",
      empty: "rewrite.plan.empty",
      count: "rewrite.plan.count",
      count_one: "rewrite.plan.count_one",
    },
    apply: {
      button: "rewrite.apply.button",
      applying: "rewrite.apply.applying",
      confirmTitle: "rewrite.apply.confirmTitle",
      confirmMessage: "rewrite.apply.confirmMessage",
      confirm: "rewrite.apply.confirm",
      cancel: "rewrite.apply.cancel",
      resultTitle: "rewrite.apply.resultTitle",
      dryRunNote: "rewrite.apply.dryRunNote",
      status: {
        ok: "rewrite.apply.status.ok",
        failed: "rewrite.apply.status.failed",
      },
    },
  },
  fixtures: {
    title: "fixtures.title",
    description: "fixtures.description",
    empty: "fixtures.empty",
    validate: "fixtures.validate",
    validating: "fixtures.validating",
    passed: "fixtures.passed",
    failed: "fixtures.failed",
    passMessage: "fixtures.passMessage",
    failMessage: "fixtures.failMessage",
    diffTitle: "fixtures.diffTitle",
    expectedBytes: "fixtures.expectedBytes",
    actualBytes: "fixtures.actualBytes",
    hashLabel: "fixtures.hashLabel",
    hasExpected: "fixtures.hasExpected",
    noExpected: "fixtures.noExpected",
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
