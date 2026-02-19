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
  /** Whether this is a modifier that combines with the next key press. */
  isModifier?: boolean;
  /** Visual width hint: "narrow" | "normal" | "wide" */
  width?: "narrow" | "normal" | "wide";
}

/** Standard terminal keys for mobile usage. */
export const TOOLBAR_KEYS: ToolbarKey[] = [
  { label: "Esc", input: "\x1b", width: "normal" },
  { label: "Tab", input: "\t", width: "normal" },
  { label: "Ctrl+C", input: "\x03", width: "wide" },
  { label: "Ctrl+D", input: "\x04", width: "wide" },
  { label: "Ctrl+Z", input: "\x1a", width: "wide" },
  { label: "\u2191", input: "\x1b[A", width: "narrow" },
  { label: "\u2193", input: "\x1b[B", width: "narrow" },
  { label: "\u2190", input: "\x1b[D", width: "narrow" },
  { label: "\u2192", input: "\x1b[C", width: "narrow" },
];
