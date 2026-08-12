/**
 * Static descriptors for the starter-action cards shown on an empty session.
 *
 * Extracted from `SessionConversation` so the renderer (badges + disable) and
 * `useStarterContextCounts` (which fetches only the types these cards need) share
 * one definition — the set of countable types is derived here, never duplicated.
 *
 * **A card carries two strings, not one.** `label` is menu text: terse, scannable,
 * and read while choosing. `prompt` is what lands in the composer: natural prose
 * that states the operator's situation, states the intent, names the shape of
 * answer wanted, and — where the card needs the operator's own material — ends
 * with an open invitation for it.
 *
 * These are two incompatible jobs and one string cannot do both. When the label
 * doubled as the prompt, "Turn this idea into goals and backlog items." was sent
 * as a complete message with no idea attached, and nothing in the composer
 * signalled that the operator was expected to supply one.
 *
 * A card whose prompt needs nothing further from the operator is send-ready; a
 * card that needs their material ends with a trailing invitation and a blank
 * line for it.
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
  /** Server-declared prompt framing rendered in the session job band. */
  jobId?: string;
  icon: LucideIcon;
  /** Menu text. Terse and scannable — read while choosing, never sent. */
  label: string;
  /**
   * Composer seed. Natural prose in the operator's voice. Ends with an open
   * invitation when the card needs the operator's own material; otherwise it is
   * complete and can be sent as-is.
   */
  prompt: string;
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
  /** Variant of `prompt` for that same single-entity case. */
  contextPrompt?: (title: string) => string;
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
 * The trailing invitation a card appends when it needs the operator's own
 * material. The blank line after it is the slot: the composer places the cursor
 * at the end, so the operator types directly into it.
 */
function withSlot(body: string, invitation: string): string {
  return `${body}\n\n${invitation}\n\n`;
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
          prompt:
            "Give me the current state of active work: what is moving, what is stalled, and what needs attention. Then tell me the single item or goal whose next action unblocks the most, why it beats the runner-up, and the exact command to start it. Note any evidence that is missing, stale, or contradicts the projection.",
          softCountType: "goal",
        },
        {
          id: "operations-decisions",
          icon: ListTodo,
          label: "Review a Plan Workshop and recommend the next operator action.",
          prompt:
            "Review the Plan Workshop for the attached item. Tell me what has already been decided, what is still open, and what the workshop is waiting on. Then recommend the specific next operator action and why it is the right one now.",
          contextLabel: (title) => `Review the Plan Workshop for "${title}" and recommend the next operator action.`,
          contextPrompt: (title) =>
            `Review the Plan Workshop for "${title}". Tell me what has already been decided, what is still open, and what the workshop is waiting on. Then recommend the specific next operator action and why it is the right one now.`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "operations-run",
          icon: Activity,
          label: "Review a failed or stale run and recommend recovery.",
          prompt:
            "The attached run failed or went stale. Read its typed terminal reason and the evidence around it, then tell me what actually went wrong as distinct from what it reported. Recommend the registered correction, review, or follow-up path, and say plainly what you could not determine.",
          contextLabel: (title) => `Review run "${title}" and recommend recovery.`,
          contextPrompt: (title) =>
            `Run "${title}" failed or went stale. Read its typed terminal reason and the evidence around it, then tell me what actually went wrong as distinct from what it reported. Recommend the registered correction, review, or follow-up path, and say plainly what you could not determine.`,
          requirements: [
            { kind: "context", type: "execution", filterKey: "execution_failed_or_stale" },
            { kind: "context", type: "agent_activity", optional: true },
          ],
        },
        {
          id: "operations-goal",
          icon: Layers,
          label: "Assess a goal and recommend its next registered transition.",
          prompt:
            "Assess the attached goal. Tell me its true state against its acceptance criteria rather than its status field, what is actually blocking it, and its next registered transition with the command to start it. If your reading disagrees with the server's next-action projection, trust the projection and tell me why I might still deviate.",
          contextLabel: (title) => `Assess "${title}" and recommend its next registered transition.`,
          contextPrompt: (title) =>
            `Assess "${title}". Tell me its true state against its acceptance criteria rather than its status field, what is actually blocking it, and its next registered transition with the command to start it. If your reading disagrees with the server's next-action projection, trust the projection and tell me why I might still deviate.`,
          requirements: [{ kind: "context", type: "goal" }],
        },
        // Two staleness cards, because they are two different jobs.
        //
        // The scoped card promises to work on "the attached items", so its
        // backlog requirement is hard: with every requirement optional the
        // card dropped its text into the composer and never opened the
        // picker, producing a prompt that referred to an attachment set that
        // did not exist. A hard requirement makes the click open the picker,
        // and earns the card a count badge.
        {
          id: "operations-triage-staleness",
          icon: GitPullRequestArrow,
          label: "Triage specific items for staleness.",
          prompt:
            "Triage the attached items for staleness. For each one, return keep, refresh, or supersede, with the evidence that produced the verdict — compare its recorded intent against the repository and its owning goal, not against its age. Propose the mutations; do not apply them.",
          contextLabel: (title) => `Triage "${title}" for staleness.`,
          contextPrompt: (title) =>
            `Triage "${title}" for staleness. Return keep, refresh, or supersede, with the evidence that produced the verdict — compare its recorded intent against the repository and its owning goal, not against its age. Propose the mutations; do not apply them.`,
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
          // Send-ready, and deliberately explicit that the agent picks the set.
          prompt:
            "Find the stalest work in the backlog yourself, using Swarm Manager's staleness signal — do not ask me to pick it. Walk me through it one item at a time, starting with the one where the gap between its recorded intent and reality is widest. For each, give keep, refresh, or supersede with the evidence behind the verdict, and wait for my decision before moving to the next.",
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
          prompt: withSlot(
            "I have a way of working with coding agents that works well for me, and I want it supported properly by this system instead of living in my head.\n\nRead what I describe, then reframe it back to me so I can check we are on the same page. Tell me whether it belongs as a session, a declared workflow, a skill change, or a deterministic action, and recommend the specific disposition. Search for precedent first — we have usually solved something close to this already somewhere in the repo.",
            "Here is how I work:",
          ),
        },
        {
          id: "workflow-author-friction",
          icon: Gauge,
          label: "Something about working with agents here is worse than it should be.",
          prompt: withSlot(
            "Something about the way I work with agents in this project is more painful than it should be, and I want to work out what is actually causing it before we decide what to change.\n\nRead the friction I describe, find where it actually originates in the system rather than where I noticed it, and recommend the smallest change that removes it. Tell me what you had to assume, and if you think I have misdiagnosed it, say so.",
            "Here is what is bothering me:",
          ),
        },
        {
          id: "workflow-author-transition",
          icon: GitPullRequestArrow,
          label: "Review an existing transition and propose a safer or clearer workflow.",
          prompt: withSlot(
            "I want to look at an existing transition and work out whether its workflow is still the right shape.\n\nRead its declaration, its skill, and its terminal outcomes, then tell me where the contract and the actual behaviour have drifted apart. Recommend whether to improve it, replace it, or leave it alone, and say what the change would cost.",
            "The transition, or the problem I have noticed:",
          ),
        },
        {
          id: "workflow-author-scenario",
          icon: Layers,
          label: "Design how agents should handle a kind of work end to end.",
          // Both requirements are optional, so this prompt can land with
          // nothing attached. It must not speak about "the attached context".
          prompt: withSlot(
            "I want to work out how agents should handle a particular kind of work in this project, from start to finish.\n\nRead how that work is handled today — the transitions, skills, and scenarios involved — tell me where the friction actually is, and recommend the smallest change that removes it: a skill change, an improvement to an existing transition, a new one, or backlog work because the capability does not exist yet.",
            "The work I have in mind:",
          ),
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
          prompt: withSlot(
            "I have an idea for this project and I want it worked into the backlog, but I am not sure yet how it maps onto what we already have — whether it is new work, an update to something existing, or not worth doing at all.\n\nReframe the idea back to me in your own words so I can check we are on the same page, tell me where it fits against the goals and items that already exist, and recommend specifically what to create or change. Flag anything you had to assume.",
            "Here is the idea:",
          ),
        },
        {
          id: "meta-existing",
          icon: Search,
          label: "Inspect existing Swarm context first, then propose a plan.",
          // Both requirements are optional, so this prompt can land with
          // nothing attached. It must not speak about "the attached context".
          prompt: withSlot(
            "Before we plan anything new, I want to understand what is already here.\n\nRead the goals, items, and scenarios around what I describe, then tell me what already exists, where the real gaps are, and what you would propose building next and why. Say which of the gaps are worth closing now and which can wait.",
            "What I am trying to work out:",
          ),
          contextLabel: (title) => `Inspect "${title}" and related Swarm context first, then propose a plan.`,
          contextPrompt: (title) =>
            withSlot(
              `Before we plan anything new, I want to understand what is already here. Start from "${title}" and the Swarm context around it.\n\nTell me what already exists there, where the real gaps are, and what you would propose building next and why. Say which of the gaps are worth closing now and which can wait.`,
              "What I am trying to work out:",
            ),
          requirements: [
            { kind: "context", type: "goal", optional: true },
            { kind: "context", type: "scenario", optional: true },
          ],
        },
        {
          id: "meta-backlog",
          icon: ListTodo,
          label: "Plan follow-up work for a backlog item.",
          prompt:
            "Look at the attached item and work out what follow-up work it needs — what it leaves unfinished, what it puts at risk, and what it makes newly possible. Recommend the specific items to create, with their scope and dependencies, and say which are genuinely independent of each other and which must be sequenced.",
          contextLabel: (title) => `Plan follow-up work for "${title}".`,
          contextPrompt: (title) =>
            `Look at "${title}" and work out what follow-up work it needs — what it leaves unfinished, what it puts at risk, and what it makes newly possible. Recommend the specific items to create, with their scope and dependencies, and say which are genuinely independent of each other and which must be sequenced.`,
          requirements: [{ kind: "context", type: "backlog_item" }],
        },
        {
          id: "meta-image",
          icon: Image,
          label: "Use an image or whiteboard as source material.",
          prompt: withSlot(
            "The attached image is source material for work I want to add to the backlog — it may be a whiteboard, a screenshot of something wrong, or a sketch of an interface.\n\nRead it carefully and tell me what you see in it. Reframe what you think I am asking for, place it against the goals and items we already have, and recommend what to create. Flag anything in the image you could not interpret rather than guessing at it.",
            "Any context worth adding:",
          ),
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
  /** Composer seed written into the target session's draft. */
  prompt: string;
  /** True when the text speaks about the attached entity specifically. */
  specific: boolean;
  /** Starts a proposal-targeted session instead of a generic conversation. */
  proposalFlavor?: "mutation_list";
  detail?: string;
  group?: "Shape" | "Discover" | "Reconcile" | "Lifecycle";
}

const ATTACH_TITLE_MAX = 70;

/** The lens cards offered against a goal, with the prompt each one sends. */
function goalLensCards(label: string): Array<Omit<AttachStarterSuggestion, "icon" | "specific" | "proposalFlavor">> {
  return [
    {
      id: "proposal-split",
      label: `Split oversized items in "${label}".`,
      group: "Shape",
      detail: "Propose smaller, independently reviewable work.",
      prompt: `Look at the items under "${label}" and find any that are too large to converge on their own. For each one, propose a split into smaller items that can each be reviewed and finished independently, and name the seam you are splitting on. Leave alone anything that only looks large.`,
    },
    {
      id: "proposal-merge",
      label: `Merge tightly coupled items in "${label}".`,
      group: "Shape",
      detail: "Propose a safer combined work item where boundaries are artificial.",
      prompt: `Look at the items under "${label}" and find any whose boundaries are artificial — work that shares a substrate, or that passes through an unsafe intermediate state if the pieces ship separately. Propose merges only where separation is actually harmful, and say what harm it causes.`,
    },
    {
      id: "proposal-identify-missing",
      label: `Identify missing work for "${label}".`,
      group: "Discover",
      detail: "Find necessary work that the current goal does not cover.",
      prompt: `Work out what "${label}" needs that is not currently represented — missing tests, cleanup, prerequisites, or follow-through. Compare the goal's stated outcome and acceptance criteria against its current item set and against the repository, then propose the items that close the gap. Say which gaps you verified and which you inferred.`,
    },
    {
      id: "proposal-reconcile",
      label: `Reconcile "${label}" with code drift.`,
      group: "Reconcile",
      detail: "Compare recorded intent with the repository and propose corrections.",
      prompt: `Compare "${label}" against what the repository actually contains now. Tell me where the recorded intent has drifted from the code — work described as pending that is already done, work described as done that is not, and scope that no longer matches reality. Propose the updates or archives that bring the record back in line.`,
    },
    {
      id: "proposal-reframe",
      label: `Reframe the scope and outcomes for "${label}".`,
      group: "Shape",
      detail: "Propose a clearer goal, boundaries, and success criteria.",
      prompt: `Reconsider the scope and outcomes of "${label}". Tell me whether its boundaries and success criteria still describe what we are actually trying to achieve. Propose a clearer statement of the outcome, what is explicitly in and out of scope, and how we would know it is genuinely done.`,
    },
  ];
}

/** The lens cards offered against a backlog item. */
function itemLensCards(label: string): Array<Omit<AttachStarterSuggestion, "icon" | "specific" | "proposalFlavor">> {
  return [
    {
      id: "proposal-split",
      label: `Split "${label}".`,
      group: "Shape",
      detail: "Propose smaller, independently reviewable follow-up items.",
      prompt: `Look at "${label}" and tell me whether it can realistically converge as one item. If it cannot, propose a split into smaller items that can each be reviewed and finished independently, and name the seam you are splitting on. If it can, say so and leave it alone.`,
    },
    {
      id: "proposal-merge",
      label: `Find merge candidates for "${label}".`,
      group: "Shape",
      detail: "Find overlapping work that should be represented once.",
      prompt: `Find work that overlaps "${label}" — items covering the same substrate, or that would pass through an unsafe intermediate state if done separately. Propose merges only where separation is actually harmful, and say what harm it causes.`,
    },
    {
      id: "proposal-identify-followups",
      label: `Identify follow-up work for "${label}".`,
      group: "Discover",
      detail: "Discover missing work needed to complete this item safely.",
      prompt: `Work out what "${label}" needs that is not currently represented — missing tests, cleanup, prerequisites, or follow-through needed to finish it safely. Compare its stated scope against the repository, then propose the items that close the gap. Say which gaps you verified and which you inferred.`,
    },
    {
      id: "proposal-reframe-item",
      label: `Reframe the scope for "${label}".`,
      group: "Shape",
      detail: "Propose a clearer outcome and boundary for this item.",
      prompt: `Reconsider the scope of "${label}". Tell me whether its boundary and done-condition still describe what we are actually trying to achieve, and propose a clearer statement of the outcome, what is in and out, and how we would know it is finished.`,
    },
    {
      id: "proposal-reconcile-item",
      label: `Reconcile "${label}" with related work.`,
      group: "Reconcile",
      detail: "Compare this record with related work and repository evidence.",
      prompt: `Compare "${label}" against related items and what the repository actually contains now. Tell me where the recorded intent has drifted from reality or duplicates work represented elsewhere, and propose the updates, links, or archives that bring the record back in line.`,
    },
  ];
}

/**
 * The starter cards that make sense when drafting a session around one
 * attached entity: cards whose every hard requirement is met by that entity
 * (image-gated cards drop out — the sheet has no attachment tray). Cards that
 * mention the entity's context type render their entity-specific variants and
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
        prompt: specific && suggestion.contextPrompt ? suggestion.contextPrompt(title) : suggestion.prompt,
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
    prompt: `Triage "${title}" for staleness. Compare its recorded intent against the repository and its owning goal — not against its age — and return one of three verdicts with the evidence behind it: keep (explain only), refresh (update_item, with reset_artifacts or recreate_item when the plan is invalid), or supersede (archive_item with a note naming what replaced it). Propose the mutations; do not apply them.`,
    detail: "Lifecycle · attach only a few entities per session to control token spend.",
    group: "Lifecycle",
    specific: true,
    proposalFlavor: "mutation_list",
  });
  return [...proposalActions, ...generic];
}
