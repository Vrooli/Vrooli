import { describe, it, expect } from "vitest";
import { slugify } from "../lib/slugify";

// [REQ:P0-007b] Terminal Key/Chord Mapping - testId generation
// [REQ:P0-006b] Configurable Shortcut Entries - testId generation
describe("slugify", () => {
  it("lowercases and replaces spaces with hyphens", () => {
    expect(slugify("Claude Code")).toBe("claude-code");
  });

  it("replaces plus signs with hyphens", () => {
    expect(slugify("Ctrl+C")).toBe("ctrl-c");
  });

  it("strips non-alphanumeric, non-hyphen characters", () => {
    expect(slugify("\u2191")).toBe("");
  });

  it("handles combined special characters", () => {
    expect(slugify("Ctrl+Z")).toBe("ctrl-z");
  });

  it("handles already-lowercased input", () => {
    expect(slugify("bash")).toBe("bash");
  });

  it("handles empty string", () => {
    expect(slugify("")).toBe("");
  });

  it("collapses multiple spaces into single hyphen", () => {
    expect(slugify("hello   world")).toBe("hello-world");
  });
});
