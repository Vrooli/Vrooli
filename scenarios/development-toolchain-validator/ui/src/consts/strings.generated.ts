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
  },
  goldens: {
    title: "goldens.title",
    subtitle: "goldens.subtitle",
    empty: "goldens.empty",
    emptyDescription: "goldens.emptyDescription",
    registerHeading: "goldens.registerHeading",
    registerOpen: "goldens.registerOpen",
    slugLabel: "goldens.slugLabel",
    templateLabel: "goldens.templateLabel",
    versionLabel: "goldens.versionLabel",
    pathLabel: "goldens.pathLabel",
    registerSubmit: "goldens.registerSubmit",
    registering: "goldens.registering",
    rowLabel: "goldens.rowLabel",
    regenerate: "goldens.regenerate",
    regenerating: "goldens.regenerating",
    delete: "goldens.delete",
    deleting: "goldens.deleting",
    close: "goldens.close",
    confirmDelete: "goldens.confirmDelete",
    confirmRegenerate: "goldens.confirmRegenerate",
    regenerateSuccess: "goldens.regenerateSuccess",
    lastRegeneratedLabel: "goldens.lastRegeneratedLabel",
    verdictSummaryPending: "goldens.verdictSummaryPending",
    verdictSummaryPlaceholder: "goldens.verdictSummaryPlaceholder",
    skillsGridCaption: "goldens.skillsGridCaption",
    toolsGridCaption: "goldens.toolsGridCaption",
    backToIndex: "goldens.backToIndex",
  },
  nav: {
    skillsTodo: "nav.skillsTodo",
    manifestsTodo: "nav.manifestsTodo",
    menuToggle: "nav.menuToggle",
    convergenceLabel: "nav.convergenceLabel",
    convergencePlaceholder: "nav.convergencePlaceholder",
    staleLabel: "nav.staleLabel",
    stalePlaceholder: "nav.stalePlaceholder",
    healthLabel: "nav.healthLabel",
    logo: "nav.logo",
    goldensLabel: "nav.goldensLabel",
    skillsLabel: "nav.skillsLabel",
    manifestsLabel: "nav.manifestsLabel",
    settingsLabel: "nav.settingsLabel",
  },
  settings: {
    title: "settings.title",
    subtitle: "settings.subtitle",
    themeHeading: "settings.themeHeading",
    themeDarkLabel: "settings.themeDarkLabel",
    themeLightLabel: "settings.themeLightLabel",
    localeHeading: "settings.localeHeading",
    densityHeading: "settings.densityHeading",
    densityComfortableLabel: "settings.densityComfortableLabel",
    densityCompactLabel: "settings.densityCompactLabel",
    sidebarHeading: "settings.sidebarHeading",
    sidebarCollapsed: "settings.sidebarCollapsed",
    catalogSyncHeading: "settings.catalogSyncHeading",
    catalogSyncPending: "settings.catalogSyncPending",
    watcherHeading: "settings.watcherHeading",
    watcherPending: "settings.watcherPending",
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
  errorBoundary: {
    title: "errorBoundary.title",
    message: "errorBoundary.message",
    retry: "errorBoundary.retry",
  },
} as const;

export type Strings = typeof strings;
