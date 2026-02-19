import { describe, it, expect } from "vitest";
import { TOOLBAR_KEYS } from "../consts/toolbar-keys";

// [REQ:P0-007b] Terminal Key/Chord Mapping
describe("Toolbar key/chord escape sequences", () => {
  it("Ctrl+C sends correct escape sequence (0x03)", () => {
    const ctrlC = TOOLBAR_KEYS.find((k) => k.label === "Ctrl+C");
    expect(ctrlC).toBeDefined();
    expect(ctrlC?.input).toBe("\x03");
  });

  it("Ctrl+D sends correct escape sequence (0x04)", () => {
    const ctrlD = TOOLBAR_KEYS.find((k) => k.label === "Ctrl+D");
    expect(ctrlD).toBeDefined();
    expect(ctrlD?.input).toBe("\x04");
  });

  it("Ctrl+Z sends correct escape sequence (0x1a)", () => {
    const ctrlZ = TOOLBAR_KEYS.find((k) => k.label === "Ctrl+Z");
    expect(ctrlZ).toBeDefined();
    expect(ctrlZ?.input).toBe("\x1a");
  });

  it("arrow keys send correct ANSI escape sequences", () => {
    const up = TOOLBAR_KEYS.find((k) => k.label === "\u2191");
    const down = TOOLBAR_KEYS.find((k) => k.label === "\u2193");
    const left = TOOLBAR_KEYS.find((k) => k.label === "\u2190");
    const right = TOOLBAR_KEYS.find((k) => k.label === "\u2192");

    expect(up?.input).toBe("\x1b[A");
    expect(down?.input).toBe("\x1b[B");
    expect(left?.input).toBe("\x1b[D");
    expect(right?.input).toBe("\x1b[C");
  });

  it("Esc sends correct escape character (0x1b)", () => {
    const esc = TOOLBAR_KEYS.find((k) => k.label === "Esc");
    expect(esc).toBeDefined();
    expect(esc?.input).toBe("\x1b");
  });

  it("Tab sends correct tab character", () => {
    const tab = TOOLBAR_KEYS.find((k) => k.label === "Tab");
    expect(tab).toBeDefined();
    expect(tab?.input).toBe("\t");
  });
});
