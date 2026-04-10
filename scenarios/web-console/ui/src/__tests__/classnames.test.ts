import { describe, it, expect } from "vitest";
import { cn } from "../lib/classnames";

// [REQ:P0-001a] Responsive Pane Grid Layout - cn utility for dynamic styling
describe("cn utility", () => {
  it("merges class strings", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("handles conditional classes", () => {
    const isHidden = false;
    expect(cn("base", isHidden && "hidden", "visible")).toBe("base visible");
  });

  it("merges tailwind conflicts", () => {
    // twMerge should resolve conflicting tailwind classes
    expect(cn("p-4", "p-2")).toBe("p-2");
  });

  it("handles undefined and null", () => {
    expect(cn("base", undefined, null, "end")).toBe("base end");
  });

  it("handles empty input", () => {
    expect(cn()).toBe("");
  });

  it("handles array inputs", () => {
    expect(cn(["foo", "bar"])).toBe("foo bar");
  });
});
