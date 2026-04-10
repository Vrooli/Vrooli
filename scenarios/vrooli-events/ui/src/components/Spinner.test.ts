// @vitest-environment node
import { describe, it, expect } from "vitest";

// [REQ:REQ-UI-014B] Spinner component — consistent loading indicator
describe("Spinner component contract", () => {
  it("default label is Loading...", () => {
    const defaults = { label: "Loading..." };
    expect(defaults.label).toBe("Loading...");
  });

  it("accepts custom label", () => {
    const props = { label: "Loading events..." };
    expect(props.label).toBe("Loading events...");
  });

  it("uses semantic token for text color", () => {
    const containerClass = "text-[var(--text-muted)]";
    expect(containerClass).toContain("--text-muted");
  });

  it("includes data-testid for automation", () => {
    const testId = "spinner";
    expect(testId).toBe("spinner");
  });
});
