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
      graph: "layout.nav.graph",
      planning: "layout.nav.planning",
      roadmap: "layout.nav.roadmap",
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
  },
  graph: {
    eyebrow: "graph.eyebrow",
    title: "graph.title",
    description: "graph.description",
    group: {
      none: "graph.group.none",
      sector: "graph.group.sector",
      tier: "graph.group.tier",
    },
    actions: {
      refresh: "graph.actions.refresh",
      dot: "graph.actions.dot",
    },
    metrics: {
      live: "graph.metrics.live",
      planned: "graph.metrics.planned",
      dependencies: "graph.metrics.dependencies",
      warnings: "graph.metrics.warnings",
    },
    states: {
      loading: "graph.states.loading",
      error: "graph.states.error",
      empty: "graph.states.empty",
    },
    warnings: {
      title: "graph.warnings.title",
      item: "graph.warnings.item",
    },
    node: {
      planned: "graph.node.planned",
      live: "graph.node.live",
      none: "graph.node.none",
      unknown: "graph.node.unknown",
      unassigned: "graph.node.unassigned",
      untiered: "graph.node.untiered",
    },
  },
  planning: {
    eyebrow: "planning.eyebrow",
    title: "planning.title",
    slugPlaceholder: "planning.slugPlaceholder",
    actions: {
      create: "planning.actions.create",
      save: "planning.actions.save",
      validate: "planning.actions.validate",
      materialize: "planning.actions.materialize",
      delete: "planning.actions.delete",
    },
    states: {
      loading: "planning.states.loading",
      emptyList: "planning.states.emptyList",
      noSelection: "planning.states.noSelection",
      noFiles: "planning.states.noFiles",
    },
    fallbacks: {
      unassigned: "planning.fallbacks.unassigned",
      untiered: "planning.fallbacks.untiered",
    },
    validation: {
      passed: "planning.validation.passed",
      findings: "planning.validation.findings",
      none: "planning.validation.none",
    },
    materialized: "planning.materialized",
    materialized_one: "planning.materialized_one",
  },
  roadmap: {
    eyebrow: "roadmap.eyebrow",
    title: "roadmap.title",
    description: "roadmap.description",
    metrics: {
      sectors: "roadmap.metrics.sectors",
      milestones: "roadmap.metrics.milestones",
      buckets: "roadmap.metrics.buckets",
    },
    tiers: {
      title: "roadmap.tiers.title",
      sectorColumn: "roadmap.tiers.sectorColumn",
      progressCounts: "roadmap.tiers.progressCounts",
    },
    sectors: {
      title: "roadmap.sectors.title",
    },
    milestones: {
      title: "roadmap.milestones.title",
      requiredScenarios: "roadmap.milestones.requiredScenarios",
      requiredScenarios_one: "roadmap.milestones.requiredScenarios_one",
    },
    states: {
      loadingProgress: "roadmap.states.loadingProgress",
      noSectors: "roadmap.states.noSectors",
      noMilestones: "roadmap.states.noMilestones",
    },
    fallbacks: {
      unassigned: "roadmap.fallbacks.unassigned",
      noDescription: "roadmap.fallbacks.noDescription",
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
