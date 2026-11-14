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
  nav: {
    search: "nav.search",
    validate: "nav.validate",
    status: "nav.status",
  },
  search: {
    title: "search.title",
    description: "search.description",
    placeholder: "search.placeholder",
    submit: "search.submit",
    modeAi: "search.modeAi",
    modeText: "search.modeText",
    modeLabel: "search.modeLabel",
    noResults: "search.noResults",
    loading: "search.loading",
    error: "search.error",
    modeUsed: "search.modeUsed",
    reranker: "search.reranker",
    resultScore: "search.resultScore",
    resultSource: "search.resultSource",
    weakMatch: "search.weakMatch",
  },
  validate: {
    title: "validate.title",
    description: "validate.description",
    placeholder: "validate.placeholder",
    submit: "validate.submit",
    loading: "validate.loading",
    error: "validate.error",
    passed: "validate.passed",
    failed: "validate.failed",
    summary: "validate.summary",
    noFindings: "validate.noFindings",
    severityError: "validate.severityError",
    severityWarning: "validate.severityWarning",
    severityInfo: "validate.severityInfo",
  },
  status: {
    title: "status.title",
    description: "status.description",
    loading: "status.loading",
    error: "status.error",
    availableYes: "status.availableYes",
    availableNo: "status.availableNo",
    ollamaLabel: "status.ollamaLabel",
    qdrantLabel: "status.qdrantLabel",
    indexedLabel: "status.indexedLabel",
    lastReconcileLabel: "status.lastReconcileLabel",
    yes: "status.yes",
    no: "status.no",
    never: "status.never",
  },
} as const;

export type Strings = typeof strings;
