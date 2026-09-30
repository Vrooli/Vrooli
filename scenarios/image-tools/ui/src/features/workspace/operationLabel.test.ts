/**
 * operationLabelKey tests — the cross-catalog label resolver. Each branch of
 * the chained `??` fallback is exercised: a deterministic op (OP_CATALOG), an
 * enhancement op (AI_CATALOG), a generation op (CREATE_CATALOG), and an unknown
 * op (the `null` tail). The expected key is read back from the same catalog so
 * the assertion can't drift from a renamed string.
 */
import { describe, expect, it } from "vitest";

import { AI_CATALOG } from "./aiCatalog";
import { CREATE_CATALOG } from "./createCatalog";
import { OP_CATALOG } from "./opCatalog";
import { operationLabelKey } from "./operationLabel";

describe("operationLabelKey", () => {
  it("resolves a deterministic op from OP_CATALOG", () => {
    expect(operationLabelKey("resize")).toBe(OP_CATALOG.resize?.labelKey);
  });

  it("resolves an enhancement op from AI_CATALOG", () => {
    expect(operationLabelKey("background_removal")).toBe(
      AI_CATALOG.background_removal?.labelKey,
    );
  });

  it("resolves a generation op from CREATE_CATALOG", () => {
    expect(operationLabelKey("text_to_image")).toBe(CREATE_CATALOG.text_to_image?.labelKey);
  });

  it("returns null for an operation absent from every catalog", () => {
    expect(operationLabelKey("totally_unknown_op")).toBeNull();
    expect(operationLabelKey("")).toBeNull();
  });
});
