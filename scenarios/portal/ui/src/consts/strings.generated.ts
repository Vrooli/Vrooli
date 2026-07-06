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
  integrations: {
    mode: {
      label: "integrations.mode.label",
      loading: "integrations.mode.loading",
      off: "integrations.mode.off",
      passive: "integrations.mode.passive",
      full: "integrations.mode.full",
      unknown: "integrations.mode.unknown",
    },
    settings: {
      title: "integrations.settings.title",
      description: "integrations.settings.description",
      overrideLabel: "integrations.settings.overrideLabel",
      overrideAuto: "integrations.settings.overrideAuto",
      overrideForceOff: "integrations.settings.overrideForceOff",
      overrideForcePassive: "integrations.settings.overrideForcePassive",
      nameHeader: "integrations.settings.nameHeader",
      stateHeader: "integrations.settings.stateHeader",
      latencyHeader: "integrations.settings.latencyHeader",
      samplesHeader: "integrations.settings.samplesHeader",
      reasonHeader: "integrations.settings.reasonHeader",
      empty: "integrations.settings.empty",
      error: "integrations.settings.error",
    },
    state: {
      available: "integrations.state.available",
      degraded: "integrations.state.degraded",
      unavailable: "integrations.state.unavailable",
      unknown: "integrations.state.unknown",
      unspecified: "integrations.state.unspecified",
    },
  },
  search: {
    omnibox: {
      title: "search.omnibox.title",
      description: "search.omnibox.description",
      inputLabel: "search.omnibox.inputLabel",
      placeholder: "search.omnibox.placeholder",
      loading: "search.omnibox.loading",
      empty: "search.omnibox.empty",
      degraded: "search.omnibox.degraded",
    },
  },
  chat: {
    title: "chat.title",
    sidebar: {
      label: "chat.sidebar.label",
      description: "chat.sidebar.description",
      newChat: "chat.sidebar.newChat",
      newAgent: "chat.sidebar.newAgent",
      newGroup: "chat.sidebar.newGroup",
      newChatTitle: "chat.sidebar.newChatTitle",
      newAgentTitle: "chat.sidebar.newAgentTitle",
      newGroupTitle: "chat.sidebar.newGroupTitle",
      ungrouped: "chat.sidebar.ungrouped",
      empty: "chat.sidebar.empty",
    },
    controls: {
      mode: "chat.controls.mode",
      modeLlm: "chat.controls.modeLlm",
      modeAgent: "chat.controls.modeAgent",
      model: "chat.controls.model",
      modelFast: "chat.controls.modelFast",
      modelReasoning: "chat.controls.modelReasoning",
      agentHarness: "chat.controls.agentHarness",
      harnessClaude: "chat.controls.harnessClaude",
      harnessCodex: "chat.controls.harnessCodex",
      harnessOpencode: "chat.controls.harnessOpencode",
      harnessGrok: "chat.controls.harnessGrok",
      webSearch: "chat.controls.webSearch",
    },
    empty: {
      description: "chat.empty.description",
    },
    message: {
      roleUser: "chat.message.roleUser",
      roleAssistant: "chat.message.roleAssistant",
      roleAgent: "chat.message.roleAgent",
      branchCount: "chat.message.branchCount",
      previousBranch: "chat.message.previousBranch",
      nextBranch: "chat.message.nextBranch",
      edit: "chat.message.edit",
      editPrompt: "chat.message.editPrompt",
      regenerate: "chat.message.regenerate",
      searchAttachments: "chat.message.searchAttachments",
      agentActivity: "chat.message.agentActivity",
      streaming: "chat.message.streaming",
    },
    composer: {
      label: "chat.composer.label",
      placeholder: "chat.composer.placeholder",
      skills: "chat.composer.skills",
      skillsPlaceholder: "chat.composer.skillsPlaceholder",
      send: "chat.composer.send",
      stop: "chat.composer.stop",
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
