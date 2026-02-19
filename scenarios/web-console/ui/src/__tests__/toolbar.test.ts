import { describe, it, expect } from "vitest";
import { TOOLBAR_KEYS } from "../consts/toolbar-keys";

// [REQ:P0-007a] Floating Toolbar Component
describe("MobileToolbar component", () => {
  it("exports TOOLBAR_KEYS with essential terminal keys", () => {
    const labels = TOOLBAR_KEYS.map((k) => k.label);
    expect(labels).toContain("Esc");
    expect(labels).toContain("Tab");
    expect(labels).toContain("Ctrl+C");
    expect(labels).toContain("Ctrl+D");
    expect(labels).toContain("Ctrl+Z");
  });

  it("component module exports default function", async () => {
    const mod = await import("../components/MobileToolbar");
    expect(typeof mod.default).toBe("function");
  });

  it("all keys have valid input sequences", () => {
    for (const key of TOOLBAR_KEYS) {
      expect(key.input.length).toBeGreaterThan(0);
      expect(key.label.length).toBeGreaterThan(0);
    }
  });
});
