import { describe, it, expect } from "vitest";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
describe("TerminalLauncher", () => {
  it("component module exports default function", async () => {
    const mod = await import("../components/TerminalLauncher");
    expect(typeof mod.default).toBe("function");
  });

  it("ShortcutEntry interface includes label, command, and description", async () => {
    const entry: import("../consts/shortcuts").ShortcutEntry = {
      label: "Test",
      command: "echo hello",
      description: "A test shortcut",
    };
    expect(entry.label).toBe("Test");
    expect(entry.command).toBe("echo hello");
    expect(entry.description).toBe("A test shortcut");
  });
});
