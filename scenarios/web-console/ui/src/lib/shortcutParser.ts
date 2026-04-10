/** Parsed representation of a keyboard shortcut like "Alt+Space" or "Ctrl+Shift+V". */
export interface ParsedShortcut {
  key: string;
  ctrlKey: boolean;
  altKey: boolean;
  shiftKey: boolean;
  metaKey: boolean;
}

type ModifierKey = "ctrlKey" | "altKey" | "shiftKey" | "metaKey";

const MODIFIER_MAP: Record<string, ModifierKey> = {
  ctrl: "ctrlKey",
  control: "ctrlKey",
  alt: "altKey",
  shift: "shiftKey",
  meta: "metaKey",
  cmd: "metaKey",
  command: "metaKey",
};

const KEY_ALIASES: Record<string, string> = {
  space: " ",
  enter: "Enter",
  tab: "Tab",
  escape: "Escape",
  esc: "Escape",
  backspace: "Backspace",
  delete: "Delete",
  arrowup: "ArrowUp",
  arrowdown: "ArrowDown",
  arrowleft: "ArrowLeft",
  arrowright: "ArrowRight",
};

/** Parse a shortcut string like "Alt+Space" into a structured object. Returns null if invalid. */
export function parseShortcut(shortcut: string): ParsedShortcut | null {
  const parts = shortcut.split("+").map((p) => p.trim());
  if (parts.length === 0) return null;

  const result: ParsedShortcut = {
    key: "",
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    metaKey: false,
  };

  for (const part of parts) {
    const lower = part.toLowerCase();
    const mod = MODIFIER_MAP[lower];
    if (mod) {
      result[mod] = true;
    } else {
      // Last non-modifier part is the key
      result.key = KEY_ALIASES[lower] ?? part;
    }
  }

  return result.key ? result : null;
}

/** Check if a keyboard event matches a parsed shortcut. */
export function matchesShortcut(event: KeyboardEvent, shortcut: ParsedShortcut): boolean {
  return (
    event.key === shortcut.key &&
    event.ctrlKey === shortcut.ctrlKey &&
    event.altKey === shortcut.altKey &&
    event.shiftKey === shortcut.shiftKey &&
    event.metaKey === shortcut.metaKey
  );
}

/** Format a KeyboardEvent into a shortcut string like "Alt+Space". */
export function formatShortcutFromEvent(event: KeyboardEvent): string {
  const parts: string[] = [];
  if (event.ctrlKey) parts.push("Ctrl");
  if (event.altKey) parts.push("Alt");
  if (event.shiftKey) parts.push("Shift");
  if (event.metaKey) parts.push("Meta");

  let key = event.key;
  if (key === " ") key = "Space";
  else if (key.length === 1) key = key.toUpperCase();

  parts.push(key);
  return parts.join("+");
}
