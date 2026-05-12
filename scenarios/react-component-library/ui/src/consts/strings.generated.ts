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
  notes: {
    title: "notes.title",
    loading: "notes.loading",
    empty: "notes.empty",
    create: "notes.create",
    createdAtLabel: "notes.createdAtLabel",
    attachmentsLabel: "notes.attachmentsLabel",
    attachmentsLabel_one: "notes.attachmentsLabel_one",
    uploadAttachment: "notes.uploadAttachment",
    attachmentFileLabel: "notes.attachmentFileLabel",
    uploadSuccess: "notes.uploadSuccess",
    noFileSelected: "notes.noFileSelected",
  },
  components: {
    title: "components.title",
    loading: "components.loading",
    empty: "components.empty",
    indexAction: "components.indexAction",
    searchLabel: "components.searchLabel",
    searchPlaceholder: "components.searchPlaceholder",
    tagLabel: "components.tagLabel",
    tagPlaceholder: "components.tagPlaceholder",
    summary: "components.summary",
    summary_one: "components.summary_one",
    versionLabel: "components.versionLabel",
    tagsLabel: "components.tagsLabel",
    noTags: "components.noTags",
    editAction: "components.editAction",
    editor: {
      title: "components.editor.title",
      loading: "components.editor.loading",
      save: "components.editor.save",
      saving: "components.editor.saving",
      close: "components.editor.close",
      dirty: "components.editor.dirty",
      sha: "components.editor.sha",
      saved: "components.editor.saved",
      noSelection: "components.editor.noSelection",
      previewHeading: "components.editor.previewHeading",
      previewWaiting: "components.editor.previewWaiting",
      previewReady: "components.editor.previewReady",
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
