/**
 * Static descriptors for the starter-action cards shown on an empty session.
 *
 * Extracted from `SessionConversation` so the renderer (badges + disable) and
 * `useStarterContextCounts` (which fetches only the types these cards need) share
 * one definition — the set of countable types is derived here, never duplicated.
 *
 * **A card's text is split by audience, not by length.** Two different readers
 * consume a starter card, and each gets its own string in its own place:
 *
 * | String | Reader | Lives in | Reaches the agent as |
 * |---|---|---|---|
 * | `label` | the operator, while choosing | this file | nothing — it is menu text |
 * | `opener` | the operator, while typing | this file | their own message, if they send it |
 * | job text | the agent | `api/internal/agentsessions/starter_jobs.go` | the stable `starter-job` prompt band |
 *
 * The job instruction is agent-directed and identical every time that card is
 * used, so it belongs server-side in a band the provider can cache. The opener
 * is operator-voiced and operator-editable, so it belongs in the composer,
 * where it can be rewritten or deleted.
 *
 * This replaced a single `prompt` field that tried to be all three. When the
 * label doubled as the prompt, "Turn this idea into goals and backlog items."
 * was sent as a complete message with no idea attached. When that was fixed by
 * moving the framing server-side, the composer was left empty instead, and
 * clicking a card produced no visible change at all — so the card read as a
 * dead control. An opener is the smallest thing that fixes both: it shows the
 * click landed, and it says what the operator is expected to supply.
 *
 * **A card without an opener is send-ready by design.** "Review active goals and
 * recommend the top next action" needs nothing from the operator, and seeding it
 * with an invitation would invent a requirement that does not exist and turn a
 * one-click send into a chore. The presence of `opener` *is* the taxonomy: it
 * marks the cards that want the operator's own material.
 */
import {
  Activity,
  Gauge,
  GitPullRequestArrow,
  History,
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
  /** Selects the server-owned job text rendered in the session job band. */
  jobId?: string;
  icon: LucideIcon;
  /** Menu text. Terse and scannable — read while choosing, never sent. */
  label: string;
  /**
   * Composer seed for a card that needs the operator's own material: a short
   * invitation in their voice, ending in a colon. Omitted on send-ready cards.
   *
   * Never phrase an opener as an instruction to the agent. That text belongs in
   * the server's job catalog; an opener that says "Recommend…" or "Return…" has
   * re-merged the two audiences this split exists to keep apart.
   */
  opener?: string;
  detail?: string;
  requirements?: SuggestionRequirement[];
  /**
   * Non-gating soft count to surface even though the card has no required
   * context (e.g. "12 active goals"). Never disables the card.
   */
  softCountType?: AgentSessionContextType;
  /**
   * Narrows the soft count to the subset the card's wording promises, so a
   * card that offers to sweep stale work previews the stale count rather than
   * the whole backlog.
   */
  softCountFilterKey?: StarterContextFilterKey;
  /**
   * Variant of `label` that speaks about a specific attached entity by title —
   * used by the attach-to-session sheet where exactly one entity is in hand.
   */
  contextLabel?: (title: string) => string;
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
    return { type: suggestion.softCountType, filterKey: suggestion.softCountFilterKey, gating: false };
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

/**
 * The text a chosen opener puts in the composer: the invitation, then a blank
 * line. The composer places the caret at the end, so the operator types
 * directly into that line.
 */
export function composerSeedForOpener(opener: string): string {
  return `${opener}\n\n`;
}

/**
 * True when the composer still holds nothing but an unfilled invitation.
 *
 * Two callers need this and must agree. Choosing a different card may replace
 * the composer only while its content is still ours, and sending must drop an
 * invitation the operator never answered — otherwise the agent receives a bare
 * "Here is the idea:" with nothing after it, which reads as a truncated message
 * rather than an empty one.
 *
 * Deliberately derived from the card catalog rather than remembered in React
 * state: composer drafts are persisted per session and survive a reload, so a
 * remembered flag would be lost exactly when the check still matters.
 */
export function isUnfilledOpener(kind: AgentSessionKind, draft: string): boolean {
  const text = draft.trim();
  if (!text) return false;
  return starterSuggestionsForKind(kind).some((suggestion) => suggestion.opener === text);
}

export function starterSuggestionsForKind(kind: AgentSessionKind): StarterSuggestion[] {
  switch (kind) {
    case "swarm_operations":
      return withStarterJobs([
        {
          id: "operations-review",
          icon: Gauge,
          label: "Review active goals and recommend the top next action.",
          // Send-ready: everything this needs is already in the startup brief.
          softCountType: "goal",
        },
        {
          id: "operations-decisions",
          icon: ListTodo,
          label: "Review a Plan Workshop and recommend the next operator action.",
          contextLabel: (title) => `Review the Plan Workshop for "${title}" and recommend the next operator action.`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "operations-run",
          icon: Activity,
          label: "Review a failed or stale run and recommend recovery.",
          contextLabel: (title) => `Review run "${title}" and recommend recovery.`,
          requirements: [
            { kind: "context", type: "execution", filterKey: "execution_failed_or_stale" },
            { kind: "context", type: "agent_activity", optional: true },
          ],
        },
        {
          id: "operations-goal",
          icon: Layers,
          label: "Assess a goal and recommend its next registered transition.",
          contextLabel: (title) => `Assess "${title}" and recommend its next registered transition.`,
          requirements: [{ kind: "context", type: "goal" }],
        },
        // Two staleness cards, because they are two different jobs.
        //
        // The scoped card promises to work on the attached items, so its
        // backlog requirement is hard: with every requirement optional the
        // card seeded the composer and never opened the picker, producing a
        // message that referred to an attachment set that did not exist. A
        // hard requirement makes the click open the picker, and earns the card
        // a count badge.
        {
          id: "operations-triage-staleness",
          icon: GitPullRequestArrow,
          label: "Triage specific items for staleness.",
          contextLabel: (title) => `Triage "${title}" for staleness.`,
          detail:
            "Pick the items or goals yourself when you already know what to look at. The agent proposes changes and never applies them.",
          requirements: [
            { kind: "context", type: "backlog_item" },
            { kind: "context", type: "goal", optional: true },
          ],
        },
        // The unscoped card is the one that answers "the backlog is stale and
        // I don't want to select every item by hand". Nothing is attached: the
        // agent resolves the set itself from the server's staleness verdict,
        // and the badge previews how much work that is.
        {
          id: "operations-sweep-staleness",
          icon: History,
          label: "Find the stalest work and walk me through it.",
          // Send-ready, and deliberately so: the agent picks the set itself.
          detail:
            "The agent picks the set itself from Swarm Manager's staleness signal, then triages it with you one item at a time.",
          softCountType: "backlog_item",
          softCountFilterKey: "backlog_item_stale",
        },
      ]);
    case "operating_mode_authoring":
      return [];
    case "workflow_authoring":
      return withStarterJobs([
        {
          id: "workflow-author-method",
          icon: Workflow,
          label: "Describe a way of working you want the system to support.",
          opener: "Here is how I work:",
        },
        {
          id: "workflow-author-friction",
          icon: Gauge,
          label: "Something about working with agents here is worse than it should be.",
          opener: "Here is what is bothering me:",
        },
        {
          id: "workflow-author-transition",
          icon: GitPullRequestArrow,
          label: "Review an existing transition and propose a safer or clearer workflow.",
          opener: "The transition, or the problem I have noticed:",
        },
        {
          id: "workflow-author-scenario",
          icon: Layers,
          label: "Design how agents should handle a kind of work end to end.",
          opener: "The work I have in mind:",
          requirements: [
            { kind: "context", type: "scenario", optional: true },
            { kind: "context", type: "goal", optional: true },
          ],
        },
      ]);
    case "meta_orchestration":
    default:
      return withStarterJobs([
        {
          id: "meta-plan",
          icon: Workflow,
          label: "Turn an idea into goals and backlog items.",
          opener: "Here is the idea:",
        },
        {
          id: "meta-existing",
          icon: Search,
          label: "Inspect existing Swarm context first, then propose a plan.",
          opener: "What I am trying to work out:",
          contextLabel: (title) => `Inspect "${title}" and related Swarm context first, then propose a plan.`,
          requirements: [
            { kind: "context", type: "goal", optional: true },
            { kind: "context", type: "scenario", optional: true },
          ],
        },
        {
          id: "meta-backlog",
          icon: ListTodo,
          label: "Plan follow-up work for a backlog item.",
          contextLabel: (title) => `Plan follow-up work for "${title}".`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "meta-image",
          icon: Image,
          label: "Use an image or whiteboard as source material.",
          opener: "Any context worth adding:",
          requirements: [{ kind: "image" }],
        },
      ]);
  }
}

function withStarterJobs(cards: StarterSuggestion[]): StarterSuggestion[] {
  return cards.map((card) => ({ ...card, jobId: card.jobId ?? card.id }));
}

/** A starter card as offered by the attach-to-session sheet. */
export interface AttachStarterSuggestion {
  id: string;
  jobId?: string;
  icon: LucideIcon;
  /** Menu text shown in the sheet. */
  label: string;
  /** Composer seed written into the target session's draft, when the card wants one. */
  opener?: string;
  /** True when the label speaks about the attached entity specifically. */
  specific: boolean;
  /** Starts a proposal-targeted session instead of a generic conversation. */
  proposalFlavor?: "mutation_list";
  detail?: string;
  group?: "Shape" | "Discover" | "Reconcile" | "Lifecycle";
}

const ATTACH_TITLE_MAX = 70;

/**
 * The lens cards offered against a goal.
 *
 * Every lens is send-ready: the sheet stages the entity as context, and the
 * mutation-list contract lives in the server's job catalog. Nothing is left for
 * the operator to type, so no lens carries an opener.
 */
function goalLensCards(label: string): Array<Omit<AttachStarterSuggestion, "icon" | "specific" | "proposalFlavor">> {
  return [
    {
      id: "proposal-split",
      label: `Split oversized items in "${label}".`,
      group: "Shape",
      detail: "Propose smaller, independently reviewable work.",
    },
    {
      id: "proposal-merge",
      label: `Merge tightly coupled items in "${label}".`,
      group: "Shape",
      detail: "Propose a safer combined work item where boundaries are artificial.",
    },
    {
      id: "proposal-identify-missing",
      label: `Identify missing work for "${label}".`,
      group: "Discover",
      detail: "Find necessary work that the current goal does not cover.",
    },
    {
      id: "proposal-reconcile",
      label: `Reconcile "${label}" with code drift.`,
      group: "Reconcile",
      detail: "Compare recorded intent with the repository and propose corrections.",
    },
    {
      id: "proposal-reframe",
      label: `Reframe the scope and outcomes for "${label}".`,
      group: "Shape",
      detail: "Propose a clearer goal, boundaries, and success criteria.",
    },
  ];
}

/** The lens cards offered against a backlog item. Send-ready, as above. */
function itemLensCards(label: string): Array<Omit<AttachStarterSuggestion, "icon" | "specific" | "proposalFlavor">> {
  return [
    {
      id: "proposal-split",
      label: `Split "${label}".`,
      group: "Shape",
      detail: "Propose smaller, independently reviewable follow-up items.",
    },
    {
      id: "proposal-merge",
      label: `Find merge candidates for "${label}".`,
      group: "Shape",
      detail: "Find overlapping work that should be represented once.",
    },
    {
      id: "proposal-identify-followups",
      label: `Identify follow-up work for "${label}".`,
      group: "Discover",
      detail: "Discover missing work needed to complete this item safely.",
    },
    {
      id: "proposal-reframe-item",
      label: `Reframe the scope for "${label}".`,
      group: "Shape",
      detail: "Propose a clearer outcome and boundary for this item.",
    },
    {
      id: "proposal-reconcile-item",
      label: `Reconcile "${label}" with related work.`,
      group: "Reconcile",
      detail: "Compare this record with related work and repository evidence.",
    },
  ];
}

/**
 * The starter cards that make sense when drafting a session around one
 * attached entity: cards whose every hard requirement is met by that entity
 * (image-gated cards drop out — the sheet has no attachment tray). Cards that
 * mention the entity's context type render their entity-specific labels and
 * sort first.
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
      const specific = Boolean(suggestion.contextLabel && mentionsOption);
      return {
        id: suggestion.id,
        jobId: suggestion.jobId,
        icon: suggestion.icon,
        label: specific && suggestion.contextLabel ? suggestion.contextLabel(title) : suggestion.label,
        opener: suggestion.opener,
        specific,
      };
    })
    .sort((a, b) => Number(b.specific) - Number(a.specific));

  if (option.type !== "goal" && option.type !== "backlog_item") return generic;
  const lenses = option.type === "goal" ? goalLensCards(title) : itemLensCards(title);
  const proposalActions: AttachStarterSuggestion[] = lenses.map((lens) => ({
    ...lens,
    jobId: lens.jobId ?? lens.id,
    icon: GitPullRequestArrow,
    specific: true,
    proposalFlavor: "mutation_list" as const,
  }));
  proposalActions.push({
    id: "proposal-triage-staleness",
    jobId: "operations-triage-staleness",
    icon: GitPullRequestArrow,
    label: `Triage "${title}" for staleness.`,
    detail: "Lifecycle · attach only a few entities per session to control token spend.",
    group: "Lifecycle",
    specific: true,
    proposalFlavor: "mutation_list",
  });
  return [...proposalActions, ...generic];
}
