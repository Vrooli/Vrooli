import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    title: "app-title",
    eyebrow: "app-eyebrow",
    description: "app-description",
  },
  health: {
    card: "health-card",
    loading: "health-loading",
    error: "health-error",
    statusValue: "health-status-value",
    serviceValue: "health-service-value",
    timestampValue: "health-timestamp-value",
    refreshButton: "health-refresh-button",
    refreshCount: "health-refresh-count",
  },
  notifications: {
    summary: "notifications-summary",
  },
  locale: {
    switcher: "locale-switcher",
  },
  layout: {
    shell: "layout-shell",
    topBar: "layout-top-bar",
    sidebar: "layout-sidebar",
    bottomNav: "layout-bottom-nav",
    main: "layout-main",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    dashboard: "page-dashboard",
    settings: "page-settings",
    plans: "page-plans",
    planDetail: "page-plan-detail",
    authoring: "page-authoring",
    execution: "page-execution",
    families: "page-families",
    validation: "page-validation",
    triage: "page-triage",
    velocity: "page-velocity",
  },
  asyncSuffix: {
    loading: "loading",
    error: "error",
    empty: "empty",
  },
  plans: {
    list: "plans-list",
    createForm: "plans-create-form",
    templateSelect: "plans-template-select",
    titleInput: "plans-title-input",
    createButton: "plans-create-button",
    detailMarkdownToggle: "plan-detail-markdown-toggle",
    detailMarkdown: "plan-detail-markdown",
    detailGraph: "plan-detail-graph",
    detailPhases: "plan-detail-phases",
    relevantContext: "plan-relevant-context",
    archiveButton: "plan-detail-archive-button",
  },
  authoring: {
    startForm: "authoring-start-form",
    titleInput: "authoring-title-input",
    templateInput: "authoring-template-input",
    startButton: "authoring-start-button",
    sections: "authoring-sections",
    contentInput: "authoring-content-input",
    submitButton: "authoring-submit-button",
    nextButton: "authoring-next-button",
    validateButton: "authoring-validate-button",
    violations: "authoring-violations",
    autofillButton: "authoring-autofill-button",
    autofillResults: "authoring-autofill-results",
    guidance: "authoring-guidance",
    phases: "authoring-phases",
    phaseTitleInput: "authoring-phase-title-input",
    phaseIntentInput: "authoring-phase-intent-input",
    phaseAddButton: "authoring-phase-add-button",
    phaseNextButton: "authoring-phase-next-button",
    phaseFieldSelect: "authoring-phase-field-select",
    phaseFieldInput: "authoring-phase-field-input",
    phaseSubmitButton: "authoring-phase-submit-button",
    contextItems: "authoring-context-items",
    contextKindSelect: "authoring-context-kind-select",
    contextPhaseToggle: "authoring-context-phase-toggle",
    contextLabelInput: "authoring-context-label-input",
    contextReasonInput: "authoring-context-reason-input",
    contextInstructionInput: "authoring-context-instruction-input",
    contextCommandInput: "authoring-context-command-input",
    contextTargetInput: "authoring-context-target-input",
    contextSubmitButton: "authoring-context-submit-button",
    contextRemoveButton: "authoring-context-remove-button",
    contextConceptsInput: "authoring-context-concepts-input",
    contextComplexityInput: "authoring-context-complexity-input",
    skillPackButton: "authoring-skill-pack-button",
    finalizeButton: "authoring-finalize-button",
    finalizedBanner: "authoring-finalized-banner",
  },
  execution: {
    startForm: "execution-start-form",
    planSelect: "execution-plan-select",
    runIdInput: "execution-run-id-input",
    startButton: "execution-start-button",
    resumeForm: "execution-resume-form",
    resumeExecutionIdInput: "execution-resume-execution-id-input",
    resumeButton: "execution-resume-button",
    guidedStep: "execution-guided-step",
    context: "execution-context",
    contextButton: "execution-context-button",
    setupContext: "execution-setup-context",
    freshenStatus: "execution-freshen-status",
    transitionSelect: "execution-transition-select",
    transitionButton: "execution-transition-button",
    decisionSummary: "execution-decision-summary",
    decisionDetail: "execution-decision-detail",
    recordDecisionButton: "execution-record-decision-button",
    findingTitle: "execution-finding-title",
    findingDetail: "execution-finding-detail",
    recordFindingButton: "execution-record-finding-button",
    feedbackCheckpoint: "execution-feedback-checkpoint",
    noFeedbackButton: "execution-no-feedback-button",
    bugTitle: "execution-bug-title",
    bugDetail: "execution-bug-detail",
    recordBugButton: "execution-record-bug-button",
    recordTitle: "execution-record-title",
    recordDetail: "execution-record-detail",
    recordRecordButton: "execution-record-record-button",
    noteTitle: "execution-note-title",
    noteDetail: "execution-note-detail",
    recordNoteButton: "execution-record-note-button",
    logSummary: "execution-log-summary",
    completeButton: "execution-complete-button",
    handoff: "execution-handoff",
  },
  families: {
    createForm: "families-create-form",
    select: "families-select",
    detail: "families-detail",
    launchState: "families-launch-state",
    membersTable: "families-members-table",
    claimsTable: "families-claims-table",
    review: "families-review",
    frontier: "families-frontier",
  },
  validation: {
    planSelect: "validation-plan-select",
    phaseSelect: "validation-phase-select",
    resolveButton: "validation-resolve-button",
    references: "validation-references",
    stalenessButton: "validation-staleness-button",
    staleness: "validation-staleness",
    baselineButton: "validation-baseline-button",
    baseline: "validation-baseline",
    runButton: "validation-run-button",
    result: "validation-result",
    commandFindings: "validation-command-findings",
    dodButton: "validation-dod-button",
    dod: "validation-dod",
  },
  triage: {
    list: "triage-list",
  },
  velocity: {
    planSelect: "velocity-plan-select",
    chart: "velocity-chart",
    table: "velocity-table",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

// Per-locale toggle test IDs are emitted by `locale.toggle({ code })` below.
// We deliberately do NOT also declare static `toggleEn` / `toggleJa` literals —
// the dynamic form is the single source of truth, and duplicating it here would
// drift the moment a new locale is added to LOCALE_CODES.
//
// `code` is constrained to `LOCALE_CODES` so `selectors.locale.toggle({ code: "fr" })`
// is a TypeScript error when "fr" isn't a supported locale. The runtime enum
// validation in `normalizeParams` provides the same guarantee at call time.
const dynamicSelectorDefinitions = {
  locale: {
    toggle: defineDynamicSelector({
      description: "Locale toggle button by language code",
      testIdPattern: "locale-toggle-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "plans",
            "authoring",
            "execution",
            "families",
            "validation",
            "triage",
            "velocity",
            "settings",
          ] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "plans",
            "authoring",
            "execution",
            "families",
            "validation",
            "triage",
            "velocity",
            "settings",
          ] as const,
        },
      },
    }),
  },
  plans: {
    row: defineDynamicSelector({
      description: "Plan list row by plan id",
      testIdPattern: "plans-row-${id}",
      params: { id: { type: "string" } },
    }),
    phase: defineDynamicSelector({
      description: "Plan-detail phase card by phase id",
      testIdPattern: "plan-phase-${id}",
      params: { id: { type: "string" } },
    }),
  },
  authoring: {
    section: defineDynamicSelector({
      description: "Authoring section row by section key",
      testIdPattern: "authoring-section-${key}",
      params: { key: { type: "string" } },
    }),
  },
  execution: {
    logEntry: defineDynamicSelector({
      description: "Execution log entry by durable ledger id",
      testIdPattern: "execution-log-entry-${id}",
      params: { id: { type: "string" } },
    }),
    retrySync: defineDynamicSelector({
      description: "Retry downstream sync for a durable ledger entry",
      testIdPattern: "execution-retry-sync-${id}",
      params: { id: { type: "string" } },
    }),
  },
  families: {
    edge: defineDynamicSelector({
      description: "Family graph edge by stable display ordinal",
      testIdPattern: "families-edge-${index}",
      params: { index: { type: "number" } },
    }),
  },
  triage: {
    row: defineDynamicSelector({
      description: "Triage finding row by finding id",
      testIdPattern: "triage-row-${id}",
      params: { id: { type: "string" } },
    }),
    promote: defineDynamicSelector({
      description: "Triage promote button by finding id",
      testIdPattern: "triage-promote-${id}",
      params: { id: { type: "string" } },
    }),
    dismiss: defineDynamicSelector({
      description: "Triage dismiss button by finding id",
      testIdPattern: "triage-dismiss-${id}",
      params: { id: { type: "string" } },
    }),
  },
  settingsPage: {
    themeOption: defineDynamicSelector({
      description: "Theme choice radio button on the settings page",
      testIdPattern: "page-settings-theme-${choice}",
      params: { choice: { type: "enum", values: ["light", "dark", "system"] as const } },
    }),
    localeOption: defineDynamicSelector({
      description: "Locale choice radio button on the settings page",
      testIdPattern: "page-settings-locale-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
