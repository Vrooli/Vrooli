import type { ModifierState, ToolbarKey } from "../consts/toolbar-keys";
import sharedKeyMap from "../../../shared/terminal-keymap.json";

export const NAMED_KEY_BYTES: ReadonlyMap<string, string> = new Map(
  sharedKeyMap.map((entry) => [entry.name, String.fromCharCode(...entry.bytes)]),
);

export function namedKeySequence(name: string, modifiers: { alt?: boolean } = {}): string | undefined {
  const bytes = NAMED_KEY_BYTES.get(name.trim().toLowerCase());
  if (!bytes) return undefined;
  return modifiers.alt ? ESC + bytes : bytes;
}

/** The single owner of terminal escape sequences and control bytes emitted by the UI. */
export const ESC = "\x1b";
export const CSI = `${ESC}[`;
export const CTRL_C = "\x03";
export const CTRL_D = "\x04";
export const CTRL_Z = "\x1a";
export const CTRL_BACKSLASH = "\x1c";
export const CTRL_L = "\x0c";
export const CTRL_A = "\x01";
export const CTRL_E = "\x05";
export const CTRL_U = "\x15";
export const CTRL_K = "\x0b";
export const CTRL_R = "\x12";
export const CTRL_P = "\x10";
export const CTRL_N = "\x0e";
export const CTRL_W = "\x17";
export const CTRL_Y = "\x19";
export const CTRL_T = "\x14";
export const CTRL_S = "\x13";
export const CTRL_Q = "\x11";
export const CSI_HOME = `${CSI}H`;
export const CSI_END = `${CSI}F`;
export const CSI_REVERSE_TAB = `${CSI}Z`;
export const ARROW_UP_BYTES = `${CSI}A`;
export const ARROW_DOWN_BYTES = `${CSI}B`;
export const ARROW_LEFT_BYTES = `${CSI}D`;
export const ARROW_RIGHT_BYTES = `${CSI}C`;

export const ESC_KEY: ToolbarKey = { label: "Esc", input: ESC, width: "normal" };
export const TAB_KEY: ToolbarKey = { label: "Tab", input: "\t", width: "normal" };
export const ENTER_KEY: ToolbarKey = { label: "Enter", input: "\r", width: "normal" };
export const ARROW_UP: ToolbarKey = { label: "\u2191", input: ARROW_UP_BYTES, width: "narrow" };
export const ARROW_DOWN: ToolbarKey = { label: "\u2193", input: ARROW_DOWN_BYTES, width: "narrow" };
export const ARROW_LEFT: ToolbarKey = { label: "\u2190", input: ARROW_LEFT_BYTES, width: "narrow" };
export const ARROW_RIGHT: ToolbarKey = { label: "\u2192", input: ARROW_RIGHT_BYTES, width: "narrow" };
export const TOOLBAR_KEYS: ToolbarKey[] = [ESC_KEY, TAB_KEY, ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT];

export function applyModifiers(input: string, mods: ModifierState): { data: string; consumed: boolean } {
  const hasModifier = mods.ctrl || mods.alt || mods.shift;
  if (!hasModifier) return { data: input, consumed: false };
  if (input === "\t") {
    if (mods.shift) {
      let result = CSI_REVERSE_TAB;
      if (mods.alt) result = ESC + result;
      return { data: result, consumed: true };
    }
    return { data: input, consumed: true };
  }
  if (input.startsWith(CSI) && input.length === 3) {
    const finalChar = input[2];
    let modNum = 1;
    if (mods.shift) modNum += 1;
    if (mods.alt) modNum += 2;
    if (mods.ctrl) modNum += 4;
    if (modNum > 1) return { data: `${CSI}1;${modNum}${finalChar}`, consumed: true };
  }
  if (input.length === 1) {
    let result = input;
    if (mods.shift) result = result.toUpperCase();
    if (mods.ctrl) {
      const code = result.toUpperCase().charCodeAt(0);
      if (code >= 0x41 && code <= 0x5a) result = String.fromCharCode(code - 0x40);
    }
    if (mods.alt) result = ESC + result;
    return { data: result, consumed: true };
  }
  return { data: input, consumed: hasModifier };
}

export function mouseWheelSequence(up: boolean, col: number, row: number): string {
  const button = up ? 64 : 65;
  return `${CSI}<${button};${col + 1};${row + 1}M`;
}

/**
 * Returns true when xterm emitted a mouse-tracking report. These bytes are
 * terminal-client controls, not shell input, and must use the best-effort
 * control lane regardless of whether they came from a wheel, touch gesture,
 * or another pointing device.
 */
export function isMouseTrackingSequence(data: string): boolean {
  if (data.length < 6 || data.charCodeAt(0) !== 0x1b || data[1] !== "[") {
    return false;
  }
  if (data[2] === "M") return data.length >= 6;
  if (data[2] !== "<") return false;
  return /^\x1b\[<\d+;\d+;\d+[Mm]$/.test(data);
}
