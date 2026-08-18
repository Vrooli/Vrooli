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
      dashboard: "layout.nav.dashboard",
      corpus: "layout.nav.corpus",
      reader: "layout.nav.reader",
      receipt: "layout.nav.receipt",
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
    corpus: {
      title: "pages.corpus.title",
      description: "pages.corpus.description",
      collections: "pages.corpus.collections",
      collectionDescription: "pages.corpus.collectionDescription",
      documents: "pages.corpus.documents",
      documentDescription: "pages.corpus.documentDescription",
      empty: "pages.corpus.empty",
      noDocuments: "pages.corpus.noDocuments",
      unnamed: "pages.corpus.unnamed",
      federated: "pages.corpus.federated",
      localOnly: "pages.corpus.localOnly",
      privacy: "pages.corpus.privacy",
    },
    reader: {
      title: "pages.reader.title",
      description: "pages.reader.description",
      searchLabel: "pages.reader.searchLabel",
      placeholder: "pages.reader.placeholder",
      partial: "pages.reader.partial",
      sourceRegion: "pages.reader.sourceRegion",
      derivedUnits: "pages.reader.derivedUnits",
      derivedUnitsLabel: "pages.reader.derivedUnitsLabel",
      noMatches: "pages.reader.noMatches",
      sourceHint: "pages.reader.sourceHint",
      confidence: "pages.reader.confidence",
      local: "pages.reader.local",
      sourceFor: "pages.reader.sourceFor",
    },
    receipt: {
      title: "pages.receipt.title",
      description: "pages.receipt.description",
      timeline: "pages.receipt.timeline",
      empty: "pages.receipt.empty",
      unnamed: "pages.receipt.unnamed",
      privacy: "pages.receipt.privacy",
      local: "pages.receipt.local",
      parsed: "pages.receipt.parsed",
    },
  },
  health: {
    title: "health.title",
    loading: "health.loading",
    error: "health.error",
    refresh: "health.refresh",
    refreshCount: "health.refreshCount",
    refreshCount_one: "health.refreshCount_one",
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
