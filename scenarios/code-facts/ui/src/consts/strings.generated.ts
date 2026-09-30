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
    mainContentLabel: "layout.mainContentLabel",
    sidebarLabel: "layout.sidebarLabel",
    bottomNavLabel: "layout.bottomNavLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      facts: "layout.nav.facts",
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
      statPlaceholderLabel: "pages.dashboard.statPlaceholderLabel",
    },
    facts: {
      title: "pages.facts.title",
      description: "pages.facts.description",
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
  facts: {
    title: "facts.title",
    description: "facts.description",
    loading: "facts.loading",
    loadingDetail: "facts.loadingDetail",
    empty: "facts.empty",
    analyze: "facts.analyze",
    cacheHit: "facts.cacheHit",
    cacheMiss: "facts.cacheMiss",
    targetKind: "facts.targetKind",
    targetValue: "facts.targetValue",
    targetPlaceholder: "facts.targetPlaceholder",
    useCache: "facts.useCache",
    familyControls: "facts.familyControls",
    target: {
      scenario: "facts.target.scenario",
      path: "facts.target.path",
      module: "facts.target.module",
      project: "facts.target.project",
    },
    family: {
      surfaces: "facts.family.surfaces",
      parseUnits: "facts.family.parseUnits",
      imports: "facts.family.imports",
      symbols: "facts.family.symbols",
      references: "facts.family.references",
      calls: "facts.family.calls",
      protoAdoption: "facts.family.protoAdoption",
      endpointProofs: "facts.family.endpointProofs",
    },
    surfaces: "facts.surfaces",
    parseUnits: "facts.parseUnits",
    facts: "facts.facts",
    evidence: "facts.evidence",
    warnings: "facts.warnings",
    targetContext: "facts.targetContext",
    resolvedKind: "facts.resolvedKind",
    scenario: "facts.scenario",
    scenarioAware: "facts.scenarioAware",
    rootPath: "facts.rootPath",
    cachePanel: "facts.cachePanel",
    cacheState: "facts.cacheState",
    cacheKey: "facts.cacheKey",
    cacheReason: "facts.cacheReason",
    cacheScope: "facts.cacheScope",
    sourceHash: "facts.sourceHash",
    configHash: "facts.configHash",
    providerVersion: "facts.providerVersion",
    hitCount: "facts.hitCount",
    surfaceInventory: "facts.surfaceInventory",
    parseUnitInventory: "facts.parseUnitInventory",
    factsTable: "facts.factsTable",
    evidenceTable: "facts.evidenceTable",
    warningsPanel: "facts.warningsPanel",
    rawJson: "facts.rawJson",
    noSurfaces: "facts.noSurfaces",
    noParseUnits: "facts.noParseUnits",
    noFacts: "facts.noFacts",
    noEvidence: "facts.noEvidence",
    noWarnings: "facts.noWarnings",
    table: {
      id: "facts.table.id",
      kind: "facts.table.kind",
      status: "facts.table.status",
      path: "facts.table.path",
      language: "facts.table.language",
      family: "facts.table.family",
      subject: "facts.table.subject",
      attributes: "facts.table.attributes",
      file: "facts.table.file",
      range: "facts.table.range",
      analyzer: "facts.table.analyzer",
      message: "facts.table.message",
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
