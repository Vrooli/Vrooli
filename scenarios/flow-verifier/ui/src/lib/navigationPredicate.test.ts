import { describe, expect, it } from "vitest";

import { evaluatePredicate } from "./navigationPredicate";

describe("evaluatePredicate", () => {
  const lookup = (m: Record<string, string>) => (n: string) => m[n];

  it("returns true for empty source", () => {
    expect(evaluatePredicate("", lookup({}))).toBe(true);
  });

  it("evaluates simple equality", () => {
    expect(evaluatePredicate("viewport = desktop", lookup({ viewport: "desktop" }))).toBe(true);
    expect(evaluatePredicate("viewport = desktop", lookup({ viewport: "mobile" }))).toBe(false);
  });

  it("supports the dense '=' form without spaces", () => {
    expect(evaluatePredicate("viewport=desktop", lookup({ viewport: "desktop" }))).toBe(true);
  });

  it("evaluates inequality", () => {
    expect(evaluatePredicate("viewport != desktop", lookup({ viewport: "mobile" }))).toBe(true);
  });

  it("evaluates AND / OR / NOT", () => {
    const ctx = lookup({ viewport: "mobile", auth: "logged_in" });
    expect(evaluatePredicate("viewport = mobile AND auth = logged_in", ctx)).toBe(true);
    expect(evaluatePredicate("viewport = mobile AND auth = anonymous", ctx)).toBe(false);
    expect(evaluatePredicate("viewport = desktop OR auth = logged_in", ctx)).toBe(true);
    expect(evaluatePredicate("NOT (viewport = desktop)", ctx)).toBe(true);
  });

  it("evaluates IN over a list", () => {
    const ctx = lookup({ viewport: "tablet" });
    expect(evaluatePredicate("viewport IN [mobile, tablet]", ctx)).toBe(true);
    expect(evaluatePredicate("viewport IN [desktop]", ctx)).toBe(false);
  });

  it("evaluates CONTAINS", () => {
    expect(
      evaluatePredicate("requires CONTAINS auth=logged_in", lookup({ requires: "auth=logged_in" })),
    ).toBe(true);
    expect(evaluatePredicate("requires CONTAINS missing", lookup({ requires: "" }))).toBe(false);
  });
});
