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
    description: "app.description",
    eyebrow: "app.eyebrow",
    title: "app.title",
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
    statusLabel: "health.statusLabel",
    timestampLabel: "health.timestampLabel",
    title: "health.title",
  },
  board: {
    title: "board.title",
    subtitle: "board.subtitle",
    refresh: "board.refresh",
    loading: "board.loading",
    error: "board.error",
    emptyTitle: "board.emptyTitle",
    emptyDetail: "board.emptyDetail",
    currentWorkTitle: "board.currentWorkTitle",
    currentWorkCaption: "board.currentWorkCaption",
    capabilityTitle: "board.capabilityTitle",
    capabilityCaption: "board.capabilityCaption",
    offerTitle: "board.offerTitle",
    offerCaption: "board.offerCaption",
    offerUnavailable: "board.offerUnavailable",
    offerEmpty: "board.offerEmpty",
    filterLabel: "board.filterLabel",
    summary: {
      currentWork: "board.summary.currentWork",
      activeCampaigns: "board.summary.activeCampaigns",
      capabilities: "board.summary.capabilities",
      gaps: "board.summary.gaps",
    },
    columns: {
      draft: "board.columns.draft",
      campaign: "board.columns.campaign",
      channel: "board.columns.channel",
      status: "board.columns.status",
      nextAction: "board.columns.nextAction",
      capability: "board.columns.capability",
      readiness: "board.columns.readiness",
      owner: "board.columns.owner",
      nextCapabilityAction: "board.columns.nextCapabilityAction",
      offerScenario: "board.columns.offerScenario",
      offerStatus: "board.columns.offerStatus",
      offerRank: "board.columns.offerRank",
      offerBlockedBy: "board.columns.offerBlockedBy",
    },
    dimensions: {
      definition: "board.dimensions.definition",
      implementation: "board.dimensions.implementation",
      operational: "board.dimensions.operational",
      quality: "board.dimensions.quality",
      distribution: "board.dimensions.distribution",
    },
    nextAction: {
      requested: "board.nextAction.requested",
      drafting: "board.nextAction.drafting",
      drafted: "board.nextAction.drafted",
      checking: "board.nextAction.checking",
      blocked: "board.nextAction.blocked",
      reviewed: "board.nextAction.reviewed",
      approved: "board.nextAction.approved",
      drift: "board.nextAction.drift",
    },
  },
  layout: {
    bottomNavLabel: "layout.bottomNavLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
      board: "layout.nav.board",
      settings: "layout.nav.settings",
    },
    sidebarLabel: "layout.sidebarLabel",
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
      statPlaceholderLabel: "pages.dashboard.statPlaceholderLabel",
      title: "pages.dashboard.title",
    },
    settings: {
      localeHeading: "pages.settings.localeHeading",
      themeHeading: "pages.settings.themeHeading",
      title: "pages.settings.title",
    },
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
