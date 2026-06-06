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
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "operations-run",
          icon: Activity,
          text: "Review a failed or stale run and recommend recovery.",
          requirements: [
            { kind: "context", type: "execution", filterKey: "execution_failed_or_stale" },
            { kind: "context", type: "agent_activity", optional: true },
          ],
        },
        {
          id: "operations-initiative",
          icon: Layers,
          text: "Assess an initiative and recommend the best operating mode.",
          requirements: [{ kind: "context", type: "initiative" }],
        },
      ];
    case "operating_mode_authoring":
      return [
        {
          id: "mode-classify",
          icon: Search,
          text: "Classify whether this workflow deserves a new operating mode.",
        },
        {
          id: "mode-draft",
          icon: GitPullRequestArrow,
          text: "Draft a mode proposal with phases, artifacts, metrics, and tests.",
        },
        {
          id: "mode-compare",
          icon: Gauge,
          text: "Compare this workflow against an existing operating mode.",
          requirements: [{ kind: "context", type: "operating_mode" }],
        },
        {
          id: "mode-initiative",
          icon: Layers,
          text: "Design an operating mode for this initiative's workflow.",
          requirements: [{ kind: "context", type: "initiative" }],
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
          requirements: [
            { kind: "context", type: "initiative", optional: true },
            { kind: "context", type: "scenario", optional: true },
          ],
        },
        {
          id: "meta-backlog",
          icon: ListTodo,
          text: "Plan follow-up work for a backlog item.",
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
