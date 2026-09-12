import { describe, expect, it } from "vitest";
import { formatShortcutFromEvent, matchesShortcut, parseShortcut } from "./shortcutParser";

describe("shortcut parser", () => {
  it("parses modifier aliases and named keys", () => {
    expect(parseShortcut("Ctrl+Shift+V")).toEqual({ key: "V", ctrlKey: true, altKey: false, shiftKey: true, metaKey: false });
    expect(parseShortcut("command+space")).toEqual({ key: " ", ctrlKey: false, altKey: false, shiftKey: false, metaKey: true });
    expect(parseShortcut("Alt+Escape")?.key).toBe("Escape");
    expect(parseShortcut("Ctrl")).toBeNull();
  });

  it("matches and formats keyboard events", () => {
    const shortcut = parseShortcut("Ctrl+Alt+K");
    expect(shortcut).not.toBeNull();
    const event = new KeyboardEvent("keydown", { key: "K", ctrlKey: true, altKey: true });
    if (!shortcut) throw new Error("shortcut should parse");
    expect(matchesShortcut(event, shortcut)).toBe(true);
    expect(formatShortcutFromEvent(new KeyboardEvent("keydown", { key: " ", shiftKey: true }))).toBe("Shift+Space");
    expect(formatShortcutFromEvent(new KeyboardEvent("keydown", { key: "x", metaKey: true }))).toBe("Meta+X");
  });
});
