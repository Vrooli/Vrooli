/** Stable identifiers for phase quick-action chips. Emitted into the
 *  `<requested_actions>` envelope; the operating-mode skill is contracted
 *  against these names. Do not rename without updating the skill. */
export type PhaseQuickActionKey =
  | "continue_from_prior"
  | "reset_and_reinvestigate"
  | "focus_on_items"
  | "skip_unblock"
  | "tighten_scope"
  | "expand_scope";

const VALID_ACTIONS: ReadonlySet<PhaseQuickActionKey> = new Set([
  "continue_from_prior",
  "reset_and_reinvestigate",
  "focus_on_items",
  "skip_unblock",
  "tighten_scope",
  "expand_scope",
]);

export function isPhaseQuickActionKey(value: string): value is PhaseQuickActionKey {
  return VALID_ACTIONS.has(value as PhaseQuickActionKey);
}

/** Build the XML envelope sent as the phase note when actions or items are
 *  selected. Empty action and selection sets emit empty blocks; the skill
 *  detects "raw note" by checking whether both blocks are empty. */
export function buildPhaseEnvelope(input: {
  phase: string;
  items: string[];
  actions: PhaseQuickActionKey[];
  note: string;
}): string {
  const phaseLine = `  <phase name="${escapeXml(input.phase)}" />`;
  const itemsBlock = input.items.length
    ? `  <selection>\n${input.items.map((ref) => `    <item ref="${escapeXml(ref)}" />`).join("\n")}\n  </selection>`
    : "  <selection></selection>";
  const actionsBlock = input.actions.length
    ? `  <requested_actions>\n${input.actions.map((name) => `    <action name="${escapeXml(name)}" />`).join("\n")}\n  </requested_actions>`
    : "  <requested_actions></requested_actions>";
  const note = input.note.trim();
  const noteBlock = `  <user_note>\n${note}\n  </user_note>`;
  return `<phase_request>\n${phaseLine}\n${itemsBlock}\n${actionsBlock}\n${noteBlock}\n</phase_request>`;
}

/** Mutual-exclusion + selection-threshold pruning. Removes actions that no
 *  longer have a valid context after a state change. */
export function prunePhaseActions(
  prev: Set<PhaseQuickActionKey>,
  ctx: { itemSelectionSize: number; phaseStartable: boolean },
): Set<PhaseQuickActionKey> {
  const next = new Set(prev);
  // tighten_scope ⊥ expand_scope; no enforcement here because applyPhaseAction
  // handles toggle logic. This prune step only removes contextually-invalid
  // actions after external state changes.
  if (ctx.itemSelectionSize === 0) {
    next.delete("focus_on_items");
  }
  if (ctx.phaseStartable) {
    next.delete("skip_unblock");
  }
  return next;
}

/** Toggle a phase action with the mutual-exclusion rules:
 *   - tighten_scope ⊥ expand_scope (selecting one removes the other)
 *   - continue_from_prior ⊥ reset_and_reinvestigate (same)
 *  All other actions stack freely. */
export function applyPhaseAction(
  prev: Set<PhaseQuickActionKey>,
  key: PhaseQuickActionKey,
): Set<PhaseQuickActionKey> {
  const next = new Set(prev);
  if (next.has(key)) {
    next.delete(key);
    return next;
  }
  if (key === "tighten_scope") next.delete("expand_scope");
  if (key === "expand_scope") next.delete("tighten_scope");
  if (key === "continue_from_prior") next.delete("reset_and_reinvestigate");
  if (key === "reset_and_reinvestigate") next.delete("continue_from_prior");
  next.add(key);
  return next;
}

function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
