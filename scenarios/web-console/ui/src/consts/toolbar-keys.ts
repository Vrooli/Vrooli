// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
/**
 * Key definitions for the mobile toolbar.
 *
 * ── VOLATILE: This file is an extension point. ──
 * When adding new toolbar keys or modifying escape sequences,
 * changes should land HERE and not in the MobileToolbar component.
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

/** Named key constants — used directly by the expanded layout's positional rendering. */
export const ESC_KEY: ToolbarKey = { label: "Esc", input: "\x1b", width: "normal" };
export const TAB_KEY: ToolbarKey = { label: "Tab", input: "\t", width: "normal" };
export const ENTER_KEY: ToolbarKey = { label: "Enter", input: "\r", width: "normal" };
export const ARROW_UP: ToolbarKey = { label: "\u2191", input: "\x1b[A", width: "narrow" };
export const ARROW_DOWN: ToolbarKey = { label: "\u2193", input: "\x1b[B", width: "narrow" };
export const ARROW_LEFT: ToolbarKey = { label: "\u2190", input: "\x1b[D", width: "narrow" };
export const ARROW_RIGHT: ToolbarKey = { label: "\u2192", input: "\x1b[C", width: "narrow" };

/** Standard terminal keys for mobile usage (compact layout). */
export const TOOLBAR_KEYS: ToolbarKey[] = [ESC_KEY, TAB_KEY, ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT];

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
export function applyModifiers(input: string, mods: ModifierState): { data: string; consumed: boolean } {
  const hasModifier = mods.ctrl || mods.alt || mods.shift;
  if (!hasModifier) {
    return { data: input, consumed: false };
  }

  // Tab (\t) — handle before the generic single-char branch
  if (input === "\t") {
    if (mods.shift) {
      let result = "\x1b[Z"; // reverse tab (CSI Z)
      if (mods.alt) result = "\x1b" + result;
      return { data: result, consumed: true };
    }
    return { data: input, consumed: true };
  }

  // For arrow keys and other CSI sequences (e.g. \x1b[A), add modifier parameter.
  // CSI sequences with modifiers use the form: \x1b[1;{mod}{final}
  // Modifier encoding: 1=none, 2=Shift, 3=Alt, 4=Shift+Alt, 5=Ctrl, 6=Ctrl+Shift, 7=Ctrl+Alt, 8=Ctrl+Shift+Alt
  if (input.startsWith("\x1b[") && input.length === 3) {
    const finalChar = input[2];
    let modNum = 1;
    if (mods.shift) modNum += 1;
    if (mods.alt) modNum += 2;
    if (mods.ctrl) modNum += 4;
    if (modNum > 1) {
      return { data: `\x1b[1;${modNum}${finalChar}`, consumed: true };
    }
  }

  // Single characters (letters, digits, symbols)
  if (input.length === 1) {
    let result = input;

    if (mods.shift) {
      result = result.toUpperCase();
    }

    if (mods.ctrl) {
      // Ctrl+letter: send the control character (A=1, B=2, ..., Z=26)
      const upper = result.toUpperCase();
      const code = upper.charCodeAt(0);
      if (code >= 0x41 && code <= 0x5a) {
        result = String.fromCharCode(code - 0x40);
      }
    }

    if (mods.alt) {
      // Alt+key: prepend ESC
      result = "\x1b" + result;
    }

    return { data: result, consumed: true };
  }

  // For other special keys, just pass through
  return { data: input, consumed: hasModifier };
}
