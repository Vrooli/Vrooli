/** Stable identifiers for quick-action lenses. These are emitted into the
 * `<requested_actions>` envelope and the swarm-manager-initiative-feedback
 * skill is contracted against them. */
export type QuickActionKey =
  | "split_oversized"
  | "merge_coupled"
  | "identify_missing_work"
  | "reconcile_with_code_drift"
  | "reframe_scope";

/** Build the XML envelope sent to the feedback agent when at least one
 * Quick action is selected, or items are selected without an action.
 * The skill's "Requested-actions interpretation" section consumes this
 * shape; action names are stable identifiers, not human labels. */
export function buildEnvelope(input: { items: string[]; actions: QuickActionKey[]; note: string }): string {
  const itemsBlock = input.items.length
    ? `<selection>\n${input.items.map((ref) => `  <item ref="${escapeXml(ref)}" />`).join("\n")}\n</selection>`
    : "<selection></selection>";
  const actionsBlock = input.actions.length
    ? `<requested_actions>\n${input.actions.map((name) => `  <action name="${escapeXml(name)}" />`).join("\n")}\n</requested_actions>`
    : "<requested_actions></requested_actions>";
  const note = input.note.trim();
  const noteBlock = `<user_note>\n${note}\n</user_note>`;
  return `${itemsBlock}\n\n${actionsBlock}\n\n${noteBlock}`;
}

// pruneActionsForSelection removes split/merge if the new selection size
// drops below their gating thresholds. Called when the user toggles items
// after picking an action.
export function pruneActionsForSelection(prev: Set<QuickActionKey>, nextSelectionSize: number): Set<QuickActionKey> {
  if (prev.size === 0) return prev;
  const next = new Set(prev);
  if (nextSelectionSize < 1 && next.has("split_oversized")) next.delete("split_oversized");
  if (nextSelectionSize < 2 && next.has("merge_coupled")) next.delete("merge_coupled");
  return next;
}

function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
