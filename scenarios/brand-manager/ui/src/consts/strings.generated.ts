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
    mainContent: "layout.mainContent",
    nav: {
      dashboard: "layout.nav.dashboard",
      settings: "layout.nav.settings",
      brands: "layout.nav.brands",
      assignments: "layout.nav.assignments",
      assets: "layout.nav.assets",
      generation: "layout.nav.generation",
      apply: "layout.nav.apply",
      discovery: "layout.nav.discovery",
      design: "layout.nav.design",
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
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    brands: {
      title: "pages.brands.title",
    },
    assignments: {
      title: "pages.assignments.title",
    },
    assets: {
      title: "pages.assets.title",
    },
    generation: {
      title: "pages.generation.title",
    },
    apply: {
      title: "pages.apply.title",
    },
    discovery: {
      title: "pages.discovery.title",
    },
    design: {
      title: "pages.design.title",
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
  brands: {
    title: "brands.title",
    loading: "brands.loading",
    empty: "brands.empty",
    create: "brands.create",
    primaryLabel: "brands.primaryLabel",
  },
  assignments: {
    title: "assignments.title",
    loading: "assignments.loading",
    empty: "assignments.empty",
    brandLabel: "assignments.brandLabel",
    elementsLabel: "assignments.elementsLabel",
  },
  assets: {
    title: "assets.title",
    loading: "assets.loading",
    empty: "assets.empty",
    brandLabel: "assets.brandLabel",
    typeLabel: "assets.typeLabel",
  },
  generation: {
    title: "generation.title",
    loading: "generation.loading",
    empty: "generation.empty",
    availableLabel: "generation.availableLabel",
    unavailableLabel: "generation.unavailableLabel",
    summaryAvailable: "generation.summaryAvailable",
    summaryUnavailable: "generation.summaryUnavailable",
    imageTitle: "generation.imageTitle",
    imageLoading: "generation.imageLoading",
    imageSummaryAvailable: "generation.imageSummaryAvailable",
    imageSummaryUnavailable: "generation.imageSummaryUnavailable",
    imageReadyLabel: "generation.imageReadyLabel",
    imageNotReadyLabel: "generation.imageNotReadyLabel",
    imageCloudLabel: "generation.imageCloudLabel",
  },
  apply: {
    title: "apply.title",
    description: "apply.description",
    brandPlaceholder: "apply.brandPlaceholder",
    scenarioPlaceholder: "apply.scenarioPlaceholder",
    previewButton: "apply.previewButton",
    previewing: "apply.previewing",
    previewFor: "apply.previewFor",
    appliedHeading: "apply.appliedHeading",
    skippedHeading: "apply.skippedHeading",
    empty: "apply.empty",
  },
  discovery: {
    title: "discovery.title",
    description: "discovery.description",
    scenarioPlaceholder: "discovery.scenarioPlaceholder",
    scanButton: "discovery.scanButton",
    scanning: "discovery.scanning",
    scanFor: "discovery.scanFor",
    sourcesHeading: "discovery.sourcesHeading",
    draftHeading: "discovery.draftHeading",
    suggestionsHeading: "discovery.suggestionsHeading",
    confidenceLabel: "discovery.confidenceLabel",
    empty: "discovery.empty",
    displayNameLabel: "discovery.displayNameLabel",
    primaryLabel: "discovery.primaryLabel",
  },
  design: {
    title: "design.title",
    description: "design.description",
    brandPlaceholder: "design.brandPlaceholder",
    generateButton: "design.generateButton",
    generating: "design.generating",
    resultHeading: "design.resultHeading",
  },
} as const;

export type Strings = typeof strings;
