/**
 * Static descriptors for the starter-action cards shown on an empty session.
 *
 * Extracted from `SessionConversation` so the renderer (badges + disable) and
 * `useStarterContextCounts` (which fetches only the types these cards need) share
 * one definition — the set of countable types is derived here, never duplicated.
 */
import {
  Activity,
  Gauge,
  GitPullRequestArrow,
  Image,
  Layers,
  ListTodo,
  Search,
  Workflow,
  type LucideIcon,
} from "lucide-react";
import type { AgentSessionContextType, AgentSessionKind } from "../../types";
import type { StarterContextFilterKey } from "./context/starter-context-filters";
import type { SessionContextOption } from "./context/session-context-refs";

export type SuggestionRequirement =
  | { kind: "context"; type: AgentSessionContextType; optional?: boolean; filterKey?: StarterContextFilterKey }
  | { kind: "image"; optional?: boolean };

export interface StarterSuggestion {
  id: string;
  icon: LucideIcon;
  text: string;
  detail?: string;
  requirements?: SuggestionRequirement[];
  /**
   * Non-gating soft count to surface even though the card has no required
   * context (e.g. "12 active initiatives"). Never disables the card.
   */
  softCountType?: AgentSessionContextType;
  /**
   * Variant of `text` that speaks about a specific attached entity by title —
   * used by the attach-to-session sheet where exactly one entity is in hand.
   */
  contextText?: (title: string) => string;
}

/** What count badge (if any) a card should show, and whether a zero gates the card. */
export interface StarterCardBadgeSpec {
  type: AgentSessionContextType;
  filterKey?: StarterContextFilterKey;
  /** True → a zero count disables the card (required context); false → soft signal only. */
  gating: boolean;
}

/**
 * The first non-optional context requirement gates the card and drives its badge.
 * Optional-only or image-only cards fall back to an explicit `softCountType`, or
 * to no badge at all.
 */
export function starterCardBadgeSpec(suggestion: StarterSuggestion): StarterCardBadgeSpec | undefined {
  const required = (suggestion.requirements ?? []).find(
    (requirement): requirement is Extract<SuggestionRequirement, { kind: "context" }> =>
      requirement.kind === "context" && !requirement.optional,
  );
  if (required) {
    return { type: required.type, filterKey: required.filterKey, gating: true };
  }
  if (suggestion.softCountType) {
    return { type: suggestion.softCountType, gating: false };
  }
  return undefined;
}

/** Context types whose stores the count hook must fetch for a given kind. */
export function countableTypesForKind(kind: AgentSessionKind): AgentSessionContextType[] {
  const types = new Set<AgentSessionContextType>();
  for (const suggestion of starterSuggestionsForKind(kind)) {
    const spec = starterCardBadgeSpec(suggestion);
    if (spec) types.add(spec.type);
  }
  return [...types];
}

export function starterSuggestionsForKind(kind: AgentSessionKind): StarterSuggestion[] {
  switch (kind) {
    case "swarm_operations":
      return [
        {
          id: "operations-review",
          icon: Gauge,
          text: "Review active initiatives and recommend the top next action.",
          softCountType: "initiative",
        },
        {
          id: "operations-decisions",
          icon: ListTodo,
          text: "Help me drain workshop decisions for a backlog item.",
          contextText: (title) => `Help me drain workshop decisions for "${title}".`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "operations-run",
          icon: Activity,
          text: "Review a failed or stale run and recommend recovery.",
          contextText: (title) => `Review run "${title}" and recommend recovery.`,
          requirements: [
            { kind: "context", type: "execution", filterKey: "execution_failed_or_stale" },
            { kind: "context", type: "agent_activity", optional: true },
          ],
        },
        {
          id: "operations-initiative",
          icon: Layers,
          text: "Assess an initiative and recommend its next registered transition.",
          contextText: (title) => `Assess "${title}" and recommend its next registered transition.`,
          requirements: [{ kind: "context", type: "initiative" }],
        },
        {
          id: "operations-triage-staleness",
          icon: GitPullRequestArrow,
          text: "Triage the attached items for staleness.",
          detail: "Attach a few items or initiatives; you control token spend. The agent proposes changes and never applies them.",
          requirements: [
            { kind: "context", type: "backlog_item", optional: true },
            { kind: "context", type: "initiative", optional: true },
          ],
        },
      ];
    case "operating_mode_authoring":
      return [];
	case "workflow_authoring":
		return [
			{
				id: "workflow-author-method",
				icon: Workflow,
				text: "Describe a coding-agent method you want Swarm Manager to support.",
			},
			{
				id: "workflow-author-transition",
				icon: GitPullRequestArrow,
				text: "Review an existing transition and propose a safer or clearer workflow.",
			},
			{
				id: "workflow-author-scenario",
				icon: Layers,
				text: "Design a workflow for a scenario or initiative.",
				requirements: [{ kind: "context", type: "scenario", optional: true }, { kind: "context", type: "initiative", optional: true }],
			},
		];
    case "meta_orchestration":
    default:
      return [
        {
          id: "meta-plan",
          icon: Workflow,
          text: "Turn this idea into initiatives and backlog items.",
        },
        {
          id: "meta-existing",
          icon: Search,
          text: "Inspect existing Swarm context first, then propose a plan.",
          contextText: (title) => `Inspect "${title}" and related Swarm context first, then propose a plan.`,
          requirements: [
            { kind: "context", type: "initiative", optional: true },
            { kind: "context", type: "scenario", optional: true },
          ],
        },
        {
          id: "meta-backlog",
          icon: ListTodo,
          text: "Plan follow-up work for a backlog item.",
          contextText: (title) => `Plan follow-up work for "${title}".`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "meta-image",
          icon: Image,
          text: "Use an image or whiteboard as source material for backlog candidates.",
          requirements: [{ kind: "image" }],
        },
      ];
  }
}

/** A starter card as offered by the attach-to-session sheet. */
export interface AttachStarterSuggestion {
  id: string;
  icon: LucideIcon;
  text: string;
  /** True when the text speaks about the attached entity specifically. */
  specific: boolean;
  /** Starts a proposal-targeted session instead of a generic conversation. */
  proposalFlavor?: "mutation_list";
  detail?: string;
  group?: "Shape" | "Discover" | "Reconcile" | "Lifecycle";
}

const ATTACH_TITLE_MAX = 70;

/**
 * The starter cards that make sense when drafting a session around one
 * attached entity: cards whose every hard requirement is met by that entity
 * (image-gated cards drop out — the sheet has no attachment tray). Cards that
 * mention the entity's context type render their `contextText` interpolation
 * and sort first.
 *
 * Requirement filter keys (e.g. failed-or-stale runs) are ignored here: the
 * user explicitly chose this entity, so it outranks the picker's narrowing.
 */
export function attachStarterSuggestions(
  kind: AgentSessionKind,
  option: Pick<SessionContextOption, "type" | "title" | "ref">,
): AttachStarterSuggestion[] {
  const rawTitle = option.title.trim() || option.ref;
  const title = rawTitle.length > ATTACH_TITLE_MAX ? `${rawTitle.slice(0, ATTACH_TITLE_MAX - 3)}...` : rawTitle;
  const generic = starterSuggestionsForKind(kind)
    .filter((suggestion) =>
      (suggestion.requirements ?? []).every(
        (requirement) => requirement.optional || (requirement.kind === "context" && requirement.type === option.type),
      ),
    )
    .map((suggestion) => {
      const mentionsOption = (suggestion.requirements ?? []).some(
        (requirement) => requirement.kind === "context" && requirement.type === option.type,
      );
      const specific = Boolean(suggestion.contextText && mentionsOption);
      return {
        id: suggestion.id,
        icon: suggestion.icon,
        text: specific && suggestion.contextText ? suggestion.contextText(title) : suggestion.text,
        specific,
      };
    })
    .sort((a, b) => Number(b.specific) - Number(a.specific));

  if (option.type !== "initiative" && option.type !== "backlog_item") return generic;
  const entityLabel = title;
  const proposalActions: AttachStarterSuggestion[] = option.type === "initiative"
    ? [
      { id: "proposal-split", prefix: "Split oversized items in", group: "Shape", detail: "Propose smaller, independently reviewable work." },
      { id: "proposal-merge", prefix: "Merge tightly coupled items in", group: "Shape", detail: "Propose a safer combined work item where boundaries are artificial." },
      { id: "proposal-identify-missing", prefix: "Identify missing work for", group: "Discover", detail: "Find necessary work that the current initiative does not cover." },
      { id: "proposal-reconcile", prefix: "Reconcile this initiative with code drift:", group: "Reconcile", detail: "Compare recorded intent with the repository and propose corrections." },
      { id: "proposal-reframe", prefix: "Reframe the scope and outcomes for", group: "Shape", detail: "Propose a clearer goal, boundaries, and success criteria." },
    ].map(({ id, prefix, group, detail }) => ({ id, icon: GitPullRequestArrow, text: `${prefix} "${entityLabel}".`, group: group as AttachStarterSuggestion["group"], detail, specific: true, proposalFlavor: "mutation_list" }))
    : [
      { id: "proposal-split", prefix: "Split", group: "Shape", detail: "Propose smaller, independently reviewable follow-up items." },
      { id: "proposal-merge", prefix: "Find merge candidates for", group: "Shape", detail: "Find overlapping work that should be represented once." },
      { id: "proposal-identify-followups", prefix: "Identify follow-up work for", group: "Discover", detail: "Discover missing work needed to complete this item safely." },
      { id: "proposal-reframe-item", prefix: "Reframe the scope for", group: "Shape", detail: "Propose a clearer outcome and boundary for this item." },
      { id: "proposal-reconcile-item", prefix: "Reconcile this item with related work:", group: "Reconcile", detail: "Compare this record with related work and repository evidence." },
    ].map(({ id, prefix, group, detail }) => ({ id, icon: GitPullRequestArrow, text: `${prefix} "${entityLabel}".`, group: group as AttachStarterSuggestion["group"], detail, specific: true, proposalFlavor: "mutation_list" }));
  proposalActions.push({
    id: "proposal-triage-staleness",
    icon: GitPullRequestArrow,
    text: `Triage "${entityLabel}" for staleness. Return keep (explain only), refresh (update_item with reset_artifacts or recreate_item), or supersede (archive_item with a note). Propose mutations only; never apply them.`,
    detail: "Lifecycle · attach only a few entities per session to control token spend.",
    group: "Lifecycle",
    specific: true,
    proposalFlavor: "mutation_list",
  });
  return [...proposalActions, ...generic];
}
