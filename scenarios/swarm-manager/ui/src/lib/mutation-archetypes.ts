/**
 * Maps every proposal op onto one of eight rendering archetypes.
 *
 * Per-op special casing is what produced the state this replaces: exactly one
 * op (`reset_artifacts`) had a renderer and the other twenty-one displayed
 * nothing but their name. Grouping by "what does this do to an object" keeps
 * the contract small enough to be exhaustive, and `PROPOSAL_OPS` plus the
 * exhaustive Record below make an unmapped op a type error rather than a blank
 * card in production.
 */
import {
  PROPOSAL_OPS,
  type Proposal,
  type ProposalItemPatch,
  type ProposalItemSpec,
  type ProposalMutation,
  type ProposalOp,
} from "../types/proposal";

export type MutationArchetype =
  /** Creation — show the object as it will exist. */
  | "object_preview"
  /** Partial edit of an existing object — show a per-field before/after. */
  | "field_diff"
  /** One value replaced — show `old → new`. */
  | "scalar_transition"
  /** Dependency edge added or removed — show both endpoints. */
  | "edge_delta"
  /** Membership change — show added and removed members. */
  | "list_delta"
  /** One object becomes many, or many become one. */
  | "fan"
  /** Something is lost — show what, what depends on it, and how to reverse. */
  | "destructive"
  /** Selected derived artifacts are removed. */
  | "scope_checklist";

export const OP_ARCHETYPE: Record<ProposalOp, MutationArchetype> = {
  add_item: "object_preview",
  create_goal: "object_preview",
  create_milestone: "object_preview",

  update_item: "field_diff",
  update_milestone: "field_diff",

  change_status: "scalar_transition",
  change_priority: "scalar_transition",
  move_milestone: "scalar_transition",

  add_edge: "edge_delta",
  remove_edge: "edge_delta",

  assign_milestone_items: "list_delta",
  unassign_milestone_items: "list_delta",
  add_goal_target: "list_delta",
  remove_goal_target: "list_delta",

  split_item: "fan",
  merge_items: "fan",

  archive_item: "destructive",
  archive_milestone: "destructive",
  interrupt_in_progress: "destructive",
  recreate_item: "destructive",
  recreate_milestone: "destructive",

  reset_artifacts: "scope_checklist",
};

export const OP_HEADLINE: Record<ProposalOp, string> = {
  add_item: "Create item",
  update_item: "Update item",
  change_status: "Change status",
  change_priority: "Change priority",
  add_edge: "Add dependency",
  remove_edge: "Remove dependency",
  move_milestone: "Move milestone",
  archive_item: "Archive item",
  interrupt_in_progress: "Interrupt running work",
  split_item: "Split item",
  merge_items: "Merge items",
  recreate_item: "Recreate item",
  reset_artifacts: "Reset artifacts",
  recreate_milestone: "Recreate milestone",
  create_goal: "Create goal",
  create_milestone: "Create milestone",
  update_milestone: "Update milestone",
  archive_milestone: "Archive milestone",
  assign_milestone_items: "Assign items to milestone",
  unassign_milestone_items: "Remove items from milestone",
  add_goal_target: "Add goal target",
  remove_goal_target: "Remove goal target",
};

/**
 * Ops that destroy or interrupt something. Wider than the `destructive`
 * archetype on purpose: split and merge archive their sources, so they warrant
 * the same warning treatment even though their primary shape is a fan.
 */
const DESTRUCTIVE_OPS = new Set<ProposalOp>([
  "archive_item",
  "archive_milestone",
  "interrupt_in_progress",
  "recreate_item",
  "recreate_milestone",
  "reset_artifacts",
  "split_item",
  "merge_items",
]);

/*
 * The lookups below accept `string | undefined` rather than `ProposalOp`.
 * Payloads are agent-authored JSON parsed at runtime, so the declared type is
 * a statement of intent, not a guarantee: a mutation that reaches the UI with
 * no `op` is malformed data, and it must degrade to a warning rather than
 * throw — one bad mutation used to be able to blank the whole decision queue.
 */

export function isDestructiveOp(op: string | undefined): boolean {
  return op !== undefined && DESTRUCTIVE_OPS.has(op as ProposalOp);
}

export function archetypeFor(op: string | undefined): MutationArchetype | undefined {
  return op === undefined ? undefined : OP_ARCHETYPE[op as ProposalOp];
}

export function headlineFor(op: string | undefined): string {
  if (!op) return "Unspecified change";
  return OP_HEADLINE[op as ProposalOp] ?? op.replace(/_/g, " ");
}

export function itemSpecRef(spec: ProposalItemSpec | undefined): string {
  if (!spec?.kind || !spec.name) return "";
  return `${spec.kind}/${spec.name}`;
}

/**
 * The reference this mutation acts on, derived per op.
 *
 * `Mutation.target` is empty for creations by design — the object does not
 * exist yet — which is why `add_item` rows used to render a blank code span.
 * Creations resolve to the ref the mutation will bring into existence.
 */
export function mutationSubject(mutation: ProposalMutation): string {
  switch (mutation.op) {
    case "add_item":
    case "merge_items":
      return itemSpecRef(mutation.item) || mutation.target || "";
    case "create_goal":
      return mutation.goal?.name ?? "";
    case "create_milestone":
    case "update_milestone":
    case "archive_milestone":
      return mutation.goal_milestone?.name || mutation.milestone_name || mutation.target || "";
    case "assign_milestone_items":
    case "unassign_milestone_items":
      return mutation.milestone_name || mutation.target || "";
    case "add_edge":
    case "remove_edge":
      return mutation.from || mutation.target || "";
    default:
      return mutation.target ?? "";
  }
}

// ---------------------------------------------------------------------------
// update_item patch analysis
// ---------------------------------------------------------------------------

export type ItemPatchField = keyof ProposalItemPatch;

/** Display order for patch fields — identity first, then body, then metadata. */
export const ITEM_PATCH_FIELDS: readonly ItemPatchField[] = [
  "title",
  "description",
  "note",
  "priority",
  "effort",
  "tags",
  "depends_on",
  "acceptance_allow",
  "acceptance_deny",
] as const;

export const ITEM_PATCH_LABELS: Record<ItemPatchField, string> = {
  title: "title",
  description: "description",
  note: "note",
  priority: "priority",
  effort: "effort",
  tags: "tags",
  depends_on: "depends on",
  acceptance_allow: "acceptance allow",
  acceptance_deny: "acceptance deny",
};

/** Fields whose values are long enough to warrant a line diff. */
const PROSE_FIELDS = new Set<ItemPatchField>(["description", "note", "title"]);

export interface PatchFieldChange {
  field: ItemPatchField;
  label: string;
  /** Incoming value, rendered. Empty string when the field is being cleared. */
  after: string;
  /** Current value when base state is available; undefined when it is not. */
  before?: string;
  /**
   * The field is present in the patch but holds an empty value. `ItemPatch` is
   * pointer-per-field, so this is an explicit clear — a different intent from
   * an absent field, and indistinguishable from it if not stated.
   */
  cleared: boolean;
  presentation: "prose" | "inline";
}

export interface PatchSummary {
  changed: PatchFieldChange[];
  /** Patch fields left untouched — stated so silence is never ambiguous. */
  unchanged: ItemPatchField[];
}

/**
 * The target's current state, when the caller could resolve it.
 *
 * Every field is optional: renderers must stay useful without it, because the
 * queue shows proposals against items that may since have been archived.
 */
export interface MutationBaseState {
  /** Current values for the fields an `update_item` patch may touch. */
  patch?: Partial<ProposalItemPatch>;
  status?: string;
  milestone?: string;
  title?: string;
}

function renderPatchValue(value: unknown): string {
  if (value === undefined || value === null) return "";
  if (Array.isArray(value)) return value.map(renderPatchValue).join(", ");
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  // Agent-authored JSON can hold a shape the schema does not describe; show it
  // rather than "[object Object]".
  return JSON.stringify(value) ?? "";
}

function isEmptyValue(value: unknown): boolean {
  if (value === undefined || value === null) return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "string") return value.trim() === "";
  return false;
}

/**
 * Splits a patch into the fields it changes and the fields it leaves alone.
 *
 * `base` is the item's current state when the server supplied it; without it
 * the change list still renders, just without a "before" side.
 */
export function describePatch(
  patch: ProposalItemPatch | undefined,
  base?: Partial<ProposalItemPatch>,
): PatchSummary {
  const changed: PatchFieldChange[] = [];
  const unchanged: ItemPatchField[] = [];

  for (const field of ITEM_PATCH_FIELDS) {
    const present = patch ? Object.prototype.hasOwnProperty.call(patch, field) && patch[field] !== undefined : false;
    if (!present) {
      unchanged.push(field);
      continue;
    }
    const after = renderPatchValue(patch?.[field]);
    const before = base && Object.prototype.hasOwnProperty.call(base, field) ? renderPatchValue(base[field]) : undefined;
    if (before !== undefined && before === after) {
      unchanged.push(field);
      continue;
    }
    changed.push({
      field,
      label: ITEM_PATCH_LABELS[field],
      after,
      before,
      cleared: isEmptyValue(patch?.[field]),
      presentation: PROSE_FIELDS.has(field) && (after.length > 80 || (before?.length ?? 0) > 80) ? "prose" : "inline",
    });
  }

  return { changed, unchanged };
}

// ---------------------------------------------------------------------------
// Payload parsing
// ---------------------------------------------------------------------------

/**
 * Parses a proposal's `payload_json`. Returns an empty envelope on malformed
 * JSON so a single bad proposal cannot blank the whole queue.
 */
export function parseProposalPayload(payloadJson: string | undefined): Proposal {
  if (!payloadJson) return { form: "mutation_list" };
  try {
    const parsed = JSON.parse(payloadJson) as Proposal;
    return parsed && typeof parsed === "object" ? parsed : { form: "mutation_list" };
  } catch {
    return { form: "mutation_list" };
  }
}

/** Mutations that carry an id, which is what the accept API keys on. */
export function proposalMutations(payloadJson: string | undefined): ProposalMutation[] {
  return (parseProposalPayload(payloadJson).mutations ?? []).filter((mutation) => Boolean(mutation?.id));
}

/**
 * True when the proposal concludes "keep this as it is".
 *
 * Sessions created before the dedicated outcome kind recorded the same
 * conclusion as a mutation list with zero mutations, so both shapes must be
 * recognised. Shared between the panel and the decision stream: the stream
 * used to filter on `kind === "mutation_list"` alone, which made keep
 * recommendations undecidable from the queue even though they need an
 * explicit accept.
 */
export function isNoChangeProposal(proposal: { kind?: string; payload_json?: string }): boolean {
  if (proposal.kind === "no_change_recommendation") return true;
  if (proposal.kind !== "mutation_list") return false;
  const payload = parseProposalPayload(proposal.payload_json);
  return payload.form === "mutation_list" && Array.isArray(payload.mutations) && payload.mutations.length === 0;
}

/** True when the op is one this UI knows how to render in full. */
export function isKnownOp(op: string): op is ProposalOp {
  return (PROPOSAL_OPS as readonly string[]).includes(op);
}

/** Whole-number days between an ISO timestamp and now; negative clamps to 0. */
export function daysSince(iso: string | undefined, now: Date = new Date()): number | undefined {
  if (!iso) return undefined;
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return undefined;
  return Math.max(0, Math.floor((now.getTime() - then) / 86_400_000));
}
