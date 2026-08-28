// Folding the attributed shortcut variants into their parent agent.
//
// The shortcut list ships eight entries, four of which are "(attributed)"
// duplicates of the other four. Rendered as eight rows they read as eight
// choices; they are really four agents and one setting. This module derives
// the folded view WITHOUT changing DEFAULT_SHORTCUTS, so the stored list keeps
// its shape and REQ-P0-006b stays satisfied — only the display changes.
//
// [REQ:P0-014a] Launcher Destination And Appearance Disclosure

import type { ShortcutEntry } from "../../consts/shortcuts";

/** The suffix a shortcut uses to mark itself as the attributed variant. */
const ATTRIBUTED_SUFFIX = " (attributed)";

export interface AgentCard {
  /** The parent label, with the attributed suffix stripped. */
  label: string;
  /** The plain command. Undefined when only an attributed variant exists. */
  command?: string;
  description?: string;
  /** The attributed command, when the list carries one. */
  attributedCommand?: string;
  attributedDescription?: string;
}

/**
 * Group shortcuts into agent cards.
 *
 * A shortcut with no attributed sibling still becomes a card with no toggle,
 * so an operator-authored shortcut is never hidden by this fold. Order follows
 * first appearance in the source list, because that list is the operator's.
 */
export function foldAttributedShortcuts(shortcuts: readonly ShortcutEntry[]): AgentCard[] {
  const byLabel = new Map<string, AgentCard>();
  const order: string[] = [];

  const cardFor = (label: string): AgentCard => {
    const existing = byLabel.get(label);
    if (existing) return existing;
    const created: AgentCard = { label };
    byLabel.set(label, created);
    order.push(label);
    return created;
  };

  for (const shortcut of shortcuts) {
    if (shortcut.label.endsWith(ATTRIBUTED_SUFFIX)) {
      const card = cardFor(shortcut.label.slice(0, -ATTRIBUTED_SUFFIX.length));
      card.attributedCommand = shortcut.command;
      card.attributedDescription = shortcut.description;
      continue;
    }
    const card = cardFor(shortcut.label);
    card.command = shortcut.command;
    card.description = shortcut.description;
  }

  return order
    .map((label) => byLabel.get(label))
    .filter((card): card is AgentCard => Boolean(card?.command ?? card?.attributedCommand));
}

/**
 * The command a card launches, honouring the attribution setting.
 *
 * Falls back rather than failing: an agent with only one of the two variants
 * still launches whichever it has, so the toggle can never leave a card dead.
 */
export function commandForCard(card: AgentCard, attributed: boolean): string | undefined {
  if (attributed) return card.attributedCommand ?? card.command;
  return card.command ?? card.attributedCommand;
}

/** True when the card actually offers both variants, so a toggle is meaningful. */
export function cardSupportsAttribution(card: AgentCard): boolean {
  return Boolean(card.command && card.attributedCommand);
}
