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
  errorBoundary: {
    somethingWentWrong: "errorBoundary.somethingWentWrong",
    tryAgain: "errorBoundary.tryAgain",
  },
  errorBanner: {
    retry: "errorBanner.retry",
    dismiss: "errorBanner.dismiss",
  },
  enableAudioBanner: {
    title: "enableAudioBanner.title",
    description: "enableAudioBanner.description",
    enable: "enableAudioBanner.enable",
    enabling: "enableAudioBanner.enabling",
    enableTitle: "enableAudioBanner.enableTitle",
    dismiss: "enableAudioBanner.dismiss",
  },
  voiceRejection: {
    title: "voiceRejection.title",
    explanatoryDetail: "voiceRejection.explanatoryDetail",
    retryableDetail: "voiceRejection.retryableDetail",
    transcribeAnyway: "voiceRejection.transcribeAnyway",
    retry: "voiceRejection.retry",
    transcribing: "voiceRejection.transcribing",
    retryTitle: "voiceRejection.retryTitle",
    dismiss: "voiceRejection.dismiss",
    dismissAriaLabel: "voiceRejection.dismissAriaLabel",
  },
  summarizeError: {
    autoFailed: "summarizeError.autoFailed",
    failed: "summarizeError.failed",
    retry: "summarizeError.retry",
    retrying: "summarizeError.retrying",
    retryTitle: "summarizeError.retryTitle",
    dismiss: "summarizeError.dismiss",
    dismissAriaLabel: "summarizeError.dismissAriaLabel",
  },
  confirmClose: {
    title: "confirmClose.title",
    body: "confirmClose.body",
    cancel: "confirmClose.cancel",
    confirm: "confirmClose.confirm",
  },
  recoverableSessions: {
    heading: "recoverableSessions.heading",
    heading_one: "recoverableSessions.heading_one",
    agentLabel: "recoverableSessions.agentLabel",
    cwdLabel: "recoverableSessions.cwdLabel",
    agentNone: "recoverableSessions.agentNone",
    reattach: "recoverableSessions.reattach",
    reattachTitle: "recoverableSessions.reattachTitle",
    dismiss: "recoverableSessions.dismiss",
    dismissTitle: "recoverableSessions.dismissTitle",
  },
} as const;

export type Strings = typeof strings;
