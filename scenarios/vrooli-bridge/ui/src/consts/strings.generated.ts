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
      runs: "layout.nav.runs",
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
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    runs: {
      title: "pages.runs.title",
      description: "pages.runs.description",
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
  fleet: {
    title: "fleet.title",
    description: "fleet.description",
    loading: "fleet.loading",
    empty: "fleet.empty",
    onlineLabel: "fleet.onlineLabel",
    offlineLabel: "fleet.offlineLabel",
    neverSeen: "fleet.neverSeen",
    revoke: "fleet.revoke",
    revokeConfirm: "fleet.revokeConfirm",
    osLabel: "fleet.osLabel",
    archLabel: "fleet.archLabel",
    versionLabel: "fleet.versionLabel",
    healthLabel: "fleet.healthLabel",
    unknownValue: "fleet.unknownValue",
    jobsHeading: "fleet.jobsHeading",
    jobsIdle: "fleet.jobsIdle",
    jobsBusy: "fleet.jobsBusy",
    status: {
      unspecified: "fleet.status.unspecified",
      offline: "fleet.status.offline",
      online: "fleet.status.online",
      needsUpdate: "fleet.status.needsUpdate",
      revoked: "fleet.status.revoked",
    },
    pairing: {
      heading: "fleet.pairing.heading",
      description: "fleet.pairing.description",
      nameLabel: "fleet.pairing.nameLabel",
      namePlaceholder: "fleet.pairing.namePlaceholder",
      submit: "fleet.pairing.submit",
      submitting: "fleet.pairing.submitting",
      codeHeading: "fleet.pairing.codeHeading",
      codeHelp: "fleet.pairing.codeHelp",
      publicKeyLabel: "fleet.pairing.publicKeyLabel",
      expiresLabel: "fleet.pairing.expiresLabel",
      copy: "fleet.pairing.copy",
    },
  },
  runs: {
    title: "runs.title",
    description: "runs.description",
    loading: "runs.loading",
    empty: "runs.empty",
    nodeLabel: "runs.nodeLabel",
    jobLabel: "runs.jobLabel",
    startedLabel: "runs.startedLabel",
    durationLabel: "runs.durationLabel",
    exitLabel: "runs.exitLabel",
    progressLabel: "runs.progressLabel",
    etaLabel: "runs.etaLabel",
    etaPending: "runs.etaPending",
    view: "runs.view",
    close: "runs.close",
    cancel: "runs.cancel",
    cancelling: "runs.cancelling",
    cancelConfirm: "runs.cancelConfirm",
    outputHeading: "runs.outputHeading",
    outputEmpty: "runs.outputEmpty",
    artifactsHeading: "runs.artifactsHeading",
    artifactsEmpty: "runs.artifactsEmpty",
    download: "runs.download",
    status: {
      unspecified: "runs.status.unspecified",
      queued: "runs.status.queued",
      running: "runs.status.running",
      passed: "runs.status.passed",
      failed: "runs.status.failed",
      aborted: "runs.status.aborted",
    },
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
