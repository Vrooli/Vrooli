/**
 * operationLabelKey tests — the label resolver now reads the generated
 * capability catalog, so a deterministic op and a model-backed op both resolve
 * through the same path and an unknown op returns null.
 */
import { describe, expect, it } from "vitest";

import { capabilityPresentation } from "./capabilityCatalog";
import { operationLabelKey } from "./operationLabel";

describe("operationLabelKey", () => {
  it("resolves a deterministic op", () => {
    expect(operationLabelKey("resize")).toBe(capabilityPresentation("resize")?.labelKey);
    expect(operationLabelKey("resize")).toBeTruthy();
  });

  it("resolves a model-backed op", () => {
    expect(operationLabelKey("background_removal")).toBe(
      capabilityPresentation("background_removal")?.labelKey,
    );
    expect(operationLabelKey("background_removal")).toBeTruthy();
  });

  it("resolves a generation op", () => {
    expect(operationLabelKey("text_to_image")).toBe(
      capabilityPresentation("text_to_image")?.labelKey,
    );
  });

  it("returns null for an unknown operation", () => {
    expect(operationLabelKey("totally_unknown_op")).toBeNull();
    expect(operationLabelKey("")).toBeNull();
  });
});
