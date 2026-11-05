// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-key-combos-p0-007
/**
 * Key definitions for the mobile toolbar.
 *
 * ── VOLATILE: This file is an extension point. ──
 * When adding new toolbar keys or modifying escape sequences,
 * changes should land HERE and not in the MobileToolbar component.
 *
 * This file owns what a key *sends*. Which controls appear on the toolbar,
 * how large they are, and how they are arranged is a separate concern owned by
 * `lib/toolbarLayout.ts` — add a control there, not here.
 *
 * [REQ:P0-007b] Terminal Key/Chord Mapping
 */

/** Key definition for the mobile toolbar. */
export interface ToolbarKey {
  label: string;
  /** The data to send to the terminal (escape sequence or character). */
  input: string;
  /** Visual width hint: "narrow" | "normal" | "wide" */
  width?: "narrow" | "normal" | "wide";
}

/** Active modifier key state. */
export interface ModifierState {
  ctrl: boolean;
  alt: boolean;
  shift: boolean;
}

/**
 * Apply active modifier keys to a toolbar key's input sequence.
 * Returns the modified escape sequence and whether modifiers were consumed.
 */
export { applyModifiers, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ENTER_KEY, ESC_KEY, TAB_KEY, TOOLBAR_KEYS } from "../lib/terminalKeys";
