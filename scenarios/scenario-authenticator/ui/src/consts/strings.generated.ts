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
  auth: {
    accountAction: "auth.accountAction",
    createAccount: "auth.createAccount",
    email: "auth.email",
    invalidCredentials: "auth.invalidCredentials",
    password: "auth.password",
    returnTarget: "auth.returnTarget",
    returnToApplication: "auth.returnToApplication",
    signedInAs: "auth.signedInAs",
    signedInHeading: "auth.signedInHeading",
    signIn: "auth.signIn",
    signInDescription: "auth.signInDescription",
    requiredFields: "auth.requiredFields",
    unableToComplete: "auth.unableToComplete",
    username: "auth.username",
    viewReadOnly: "auth.viewReadOnly",
    working: "auth.working",
  },
  "data-display": {
    "audit-trail": {
      "audit-trail": "data-display.audit-trail.audit-trail",
    },
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
  feedback: {
    "permission-state": {
      "permission-required": "feedback.permission-state.permission-required",
      "request-access": "feedback.permission-state.request-access",
      "request-access-to-continue": "feedback.permission-state.request-access-to-continue",
    },
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
  layout: {
    bottomNavLabel: "layout.bottomNavLabel",
    mainLabel: "layout.mainLabel",
    nav: {
      dashboard: "layout.nav.dashboard",
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
  security: {
    auditEmpty: "security.auditEmpty",
    auditHeading: "security.auditHeading",
    description: "security.description",
    heading: "security.heading",
    permissionDescription: "security.permissionDescription",
    permissionHeading: "security.permissionHeading",
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
