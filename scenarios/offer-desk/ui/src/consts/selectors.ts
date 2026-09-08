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
    firedTriggers: "board-fired-triggers",
    blockedOffers: "board-blocked-offers",
    earningNothing: "board-earning-nothing",
    evaluationCondition: "board-evaluation-condition",
    sourceAvailability: "board-source-availability",
    ledgerGap: "board-ledger-gap",
    boardRanking: "board-ranking",
    emptyGuidance: "board-empty-guidance",
    postureSummary: "board-posture-summary",
    postureGoalVerdicts: "board-posture-goal-verdicts",
    postureBasis: "board-posture-basis",
    postureGap: "board-posture-gap",
    defaultAliveGap: "board-default-alive-gap",
    offers: "page-offers",
    triggers: "page-triggers",
    proposals: "page-proposals",
    settings: "page-settings",
    releaseLadder: "page-release-ladder",
    releaseLadderSchedule: "release-ladder-schedule",
    releaseLadderStream: "release-ladder-stream",
    releaseLadderPrerequisites: "release-ladder-prerequisites",
    releaseLadderGoals: "release-ladder-goals",
    releaseLadderSourceDegraded: "release-ladder-source-degraded",
    offerGraph: "offer-graph",
    offerTable: "offer-table",
    offerStatus: "offer-status",
    offerRankReason: "offer-rank-reason",
    offerWaitingOn: "offer-waiting-on",
    offerLegalTransitions: "offer-legal-transitions",
    offerRefusalReason: "offer-refusal-reason",
    offerRefusalRemedy: "offer-refusal-remedy",
    offerAuditTrail: "offer-audit-trail",
    offerPromote: "offer-promote",
    offerRoleRequirement: "offer-role-requirement",
    offersEmptyGuidance: "offers-empty-guidance",
    offerMembershipFinding: "offer-membership-finding",
    offerCreateNode: "offer-create-node",
    offerTransition: "offer-transition",
    offerCreateEdge: "offer-create-edge",
    offerMergeForm: "offer-merge-form",
    offerMergeSummary: "offer-merge-summary",
    catalogGroupedView: "catalog-grouped-view",
    catalogKindGroup: "catalog-kind-group",
    catalogNodeEdgeCounts: "catalog-node-edge-counts",
    catalogViewToggle: "catalog-view-toggle",
    catalogDuplicateBanner: "catalog-duplicate-banner",
    offerChangeNotice: "offer-change-notice",
    triggerEditor: "trigger-editor",
    triggerParseStatus: "trigger-parse-status",
    triggerParseError: "trigger-parse-error",
    triggerDryRun: "trigger-dry-run",
    triggerDryRunVerdict: "trigger-dry-run-verdict",
    triggerFactTrace: "trigger-fact-trace",
    triggerMissingFact: "trigger-missing-fact",
    factRegistry: "fact-registry",
    evaluationFreshness: "evaluation-freshness",
    evaluationStalledAlert: "evaluation-stalled-alert",
    triggersEmptyGuidance: "triggers-empty-guidance",
    triggerDeclare: "trigger-declare",
    triggerAddFact: "trigger-add-fact",
    triggerDeclareAction: "trigger-declare-action",
    triggerAddFactAction: "trigger-add-fact-action",
    triggerChangeNotice: "trigger-change-notice",
    proposalList: "proposal-list",
    proposalTable: "proposal-table",
    proposalProposer: "proposal-proposer",
    proposalRequestedStatus: "proposal-requested-status",
    proposalReason: "proposal-reason",
    proposalEvidence: "proposal-evidence",
    proposalAge: "proposal-age",
    proposalEffect: "proposal-effect",
    proposalDeclineHistory: "proposal-decline-history",
    proposalAccept: "proposal-accept",
    proposalDecline: "proposal-decline",
    proposalDeclineReason: "proposal-decline-reason",
    proposalChangeNotice: "proposal-change-notice",
    proposalOperatorOnly: "proposal-operator-only",
    proposalsEmptyGuidance: "proposals-empty-guidance",
    evaluationSchedule: "settings-evaluation-schedule",
    schedulePausedNotice: "settings-schedule-paused",
    ledgerConnection: "settings-ledger-connection",
    ledgerConnectionReason: "settings-ledger-connection-reason",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "offers",
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
            "offers",
            "settings",
          ] as const,
        },
      },
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
