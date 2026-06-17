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
    nav: {
      dashboard: "layout.nav.dashboard",
      editor: "layout.nav.editor",
      jobs: "layout.nav.jobs",
      models: "layout.nav.models",
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
    jobs: {
      title: "pages.jobs.title",
    },
    models: {
      title: "pages.models.title",
    },
    settings: {
      title: "pages.settings.title",
      themeHeading: "pages.settings.themeHeading",
      localeHeading: "pages.settings.localeHeading",
    },
    editor: {
      title: "pages.editor.title",
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
  jobs: {
    title: "jobs.title",
    loading: "jobs.loading",
    empty: "jobs.empty",
    operationLabel: "jobs.operationLabel",
    laneLabel: "jobs.laneLabel",
    stateLabel: "jobs.stateLabel",
    progressLabel: "jobs.progressLabel",
    cancel: "jobs.cancel",
    lane: {
      unspecified: "jobs.lane.unspecified",
      gpu: "jobs.lane.gpu",
      cpu: "jobs.lane.cpu",
    },
    state: {
      unspecified: "jobs.state.unspecified",
      queued: "jobs.state.queued",
      running: "jobs.state.running",
      succeeded: "jobs.state.succeeded",
      failed: "jobs.state.failed",
      canceled: "jobs.state.canceled",
    },
  },
  models: {
    title: "models.title",
    loading: "models.loading",
    empty: "models.empty",
    tierLabel: "models.tierLabel",
    backendLabel: "models.backendLabel",
    enabledLabel: "models.enabledLabel",
    disabledLabel: "models.disabledLabel",
    enable: "models.enable",
    disable: "models.disable",
  },
  editor: {
    title: "editor.title",
    loading: "editor.loading",
    error: "editor.error",
    uploadLabel: "editor.uploadLabel",
    uploadHint: "editor.uploadHint",
    operationLabel: "editor.operationLabel",
    overlayLabel: "editor.overlayLabel",
    run: "editor.run",
    running: "editor.running",
    empty: "editor.empty",
    originalLabel: "editor.originalLabel",
    resultLabel: "editor.resultLabel",
    resultMeta: "editor.resultMeta",
    metadataLabel: "editor.metadataLabel",
    download: "editor.download",
    field: {
      width: "editor.field.width",
      height: "editor.field.height",
      fit: "editor.field.fit",
      gravity: "editor.field.gravity",
      x: "editor.field.x",
      y: "editor.field.y",
      angle: "editor.field.angle",
      background: "editor.field.background",
      axis: "editor.field.axis",
      brightness: "editor.field.brightness",
      contrast: "editor.field.contrast",
      gamma: "editor.field.gamma",
      saturation: "editor.field.saturation",
      hue: "editor.field.hue",
      filter: "editor.field.filter",
      amount: "editor.field.amount",
      format: "editor.field.format",
      quality: "editor.field.quality",
      lossless: "editor.field.lossless",
      targetBytes: "editor.field.targetBytes",
      text: "editor.field.text",
      position: "editor.field.position",
      opacity: "editor.field.opacity",
      color: "editor.field.color",
      fontSize: "editor.field.fontSize",
      stripAll: "editor.field.stripAll",
      stripGps: "editor.field.stripGps",
      autoOrient: "editor.field.autoOrient",
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
