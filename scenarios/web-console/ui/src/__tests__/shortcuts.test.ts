import { describe, it, expect } from "vitest";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";

/**
 * Tests for the shortcut definitions extension point.
 *
 * These tests encode the invariants that must hold regardless of how
 * many shortcuts exist or how they change. When adding new shortcuts
 * to DEFAULT_SHORTCUTS, these tests validate the structural contract.
 *
 * [REQ:P0-006b] Configurable Shortcut Entries
 */
describe("Shortcut definitions (change axis: shortcut profiles)", () => {
  it("exports a non-empty array of shortcuts", () => {
    expect(Array.isArray(DEFAULT_SHORTCUTS)).toBe(true);
    expect(DEFAULT_SHORTCUTS.length).toBeGreaterThan(0);
  });

  it("every shortcut has a non-empty label and command", () => {
    for (const shortcut of DEFAULT_SHORTCUTS) {
      expect(shortcut.label.trim().length).toBeGreaterThan(0);
      expect(shortcut.command.trim().length).toBeGreaterThan(0);
    }
  });

  it("shortcut labels are unique", () => {
    const labels = DEFAULT_SHORTCUTS.map((s) => s.label);
    expect(new Set(labels).size).toBe(labels.length);
  });

  it("shortcut commands are unique", () => {
    const commands = DEFAULT_SHORTCUTS.map((s) => s.command);
    expect(new Set(commands).size).toBe(commands.length);
  });

  it("includes a launcher for every captured agent runtime", () => {
    const labels = DEFAULT_SHORTCUTS.map((s) => s.label.toLowerCase());
    for (const agent of ["claude", "codex", "opencode", "grok"]) {
      expect(labels.some((l) => l.includes(agent))).toBe(true);
    }
  });

  it("custom ShortcutEntry can be created with minimal fields", () => {
    const custom: ShortcutEntry = { label: "Test", command: "echo test" };
    expect(custom.label).toBe("Test");
    expect(custom.command).toBe("echo test");
    expect(custom.description).toBeUndefined();
  });

  it("custom ShortcutEntry can include description", () => {
    const custom: ShortcutEntry = {
      label: "Test",
      command: "echo test",
      description: "A test shortcut",
    };
    expect(custom.description).toBe("A test shortcut");
  });
});
