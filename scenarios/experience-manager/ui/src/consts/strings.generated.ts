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
    nav: {
      fleet: "layout.nav.fleet",
      explorer: "layout.nav.explorer",
      evidence: "layout.nav.evidence",
      studio: "layout.nav.studio",
      findings: "layout.nav.findings",
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
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
  },
  experience: {
    common: {
      scenario: "experience.common.scenario",
      page: "experience.common.page",
      depth: "experience.common.depth",
      debt: "experience.common.debt",
      status: "experience.common.status",
      claims: "experience.common.claims",
      viewEvidence: "experience.common.viewEvidence",
    },
    fleet: {
      title: "experience.fleet.title",
      description: "experience.fleet.description",
      depthLabel: "experience.fleet.depthLabel",
      specCoverage: "experience.fleet.specCoverage",
      depthDistribution: "experience.fleet.depthDistribution",
      refresh: "experience.fleet.refresh",
      tableLabel: "experience.fleet.tableLabel",
      pagesTracked: "experience.fleet.pagesTracked",
      loadingData: "experience.fleet.loadingData",
    },
    explorer: {
      title: "experience.explorer.title",
      description: "experience.explorer.description",
      gridLabel: "experience.explorer.gridLabel",
      gapsLabel: "experience.explorer.gapsLabel",
      openStudio: "experience.explorer.openStudio",
      claimsLabel: "experience.explorer.claimsLabel",
      defaultState: "experience.explorer.defaultState",
      emptyState: "experience.explorer.emptyState",
      staleState: "experience.explorer.staleState",
      loadingSpec: "experience.explorer.loadingSpec",
      emptySpec: "experience.explorer.emptySpec",
      loadError: "experience.explorer.loadError",
      emptyGap: "experience.explorer.emptyGap",
      staleData: "experience.explorer.staleData",
      summary: "experience.explorer.summary",
      refreshing: "experience.explorer.refreshing",
      loadingClaims: "experience.explorer.loadingClaims",
      emptyClaims: "experience.explorer.emptyClaims",
    },
    evidence: {
      title: "experience.evidence.title",
      description: "experience.evidence.description",
      captureLabel: "experience.evidence.captureLabel",
      treeLabel: "experience.evidence.treeLabel",
      verdictsLabel: "experience.evidence.verdictsLabel",
      recapture: "experience.evidence.recapture",
      refreshing: "experience.evidence.refreshing",
      loadingEvidence: "experience.evidence.loadingEvidence",
      emptyEvidence: "experience.evidence.emptyEvidence",
      emptyVerdicts: "experience.evidence.emptyVerdicts",
      emptyTree: "experience.evidence.emptyTree",
      loadError: "experience.evidence.loadError",
      staleEvidence: "experience.evidence.staleEvidence",
      captureReference: "experience.evidence.captureReference",
    },
    studio: {
      title: "experience.studio.title",
      description: "experience.studio.description",
      formLabel: "experience.studio.formLabel",
      validationLabel: "experience.studio.validationLabel",
      validationCopy: "experience.studio.validationCopy",
      wireframeLabel: "experience.studio.wireframeLabel",
      variantsLabel: "experience.studio.variantsLabel",
      save: "experience.studio.save",
      promote: "experience.studio.promote",
      defaultPage: "experience.studio.defaultPage",
      defaultClaim: "experience.studio.defaultClaim",
      previewDepthSummary: "experience.studio.previewDepthSummary",
      previewCoverageMeter: "experience.studio.previewCoverageMeter",
      previewDebtTable: "experience.studio.previewDebtTable",
      variantCompactTable: "experience.studio.variantCompactTable",
      variantEvidenceForward: "experience.studio.variantEvidenceForward",
    },
    findings: {
      title: "experience.findings.title",
      description: "experience.findings.description",
      listLabel: "experience.findings.listLabel",
      preview: "experience.findings.preview",
      apply: "experience.findings.apply",
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
